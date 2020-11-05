package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
)

const (
	resourceKindNetwork = iota
	resourceKindContainer

	resourceActionCreate
	resourceActionDestroy
	resourceActionStart
	resourceActionStop
)

var (
	dockerHost = strings.TrimSpace(os.ExpandEnv("${DOCKER_HOST}"))

	resourceKindByString = map[string]int{
		"network":   resourceKindNetwork,
		"container": resourceKindContainer,
	}

	resourceActionByString = map[string]int{
		"create":  resourceActionCreate,
		"destroy": resourceActionDestroy,
		"start":   resourceActionStart,
		"stop":    resourceActionStop,
		"die":     resourceActionStop,
	}

	hostProxySubnetFirstOctet  = 127
	usedHostSecondOctet        = [256]bool{}
	dockerSubnetOctetReplRegex = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}`)
)

type DockerClient struct {
	logger *log.Logger
	cache  *Cache
	docker *docker.Client
}

func NewDockerClient(logger *log.Logger, cache *Cache) (*DockerClient, error) {
	if dockerHost == "" {
		return nil, errors.New("empty DOCKER_HOST")
	}

	logger.Println("creating docker client")
	docker, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return &DockerClient{
		logger: logger,
		cache:  cache,
		docker: docker,
	}, nil
}

func (c *DockerClient) WaitForDockerd() error {
	if !strings.HasPrefix(dockerHost, "tcp://") {
		return errors.New("DOCKER_HOST not tcp://")
	}

	connected := false
	ipAndPort := strings.TrimPrefix(dockerHost, "tcp://")
	for i := 0; i < 15; i++ {
		c.logger.Printf("trying to connect to dockerd on %s\n", ipAndPort)
		conn, err := net.DialTimeout("tcp", ipAndPort, time.Second)
		if err == nil && conn != nil {
			defer conn.Close()
			c.logger.Printf("connected to dockerd successfully")
			connected = true
			break
		}
		time.Sleep(time.Second)
	}

	if !connected {
		return fmt.Errorf("connection to dockerd on %s failed", ipAndPort)
	}

	return nil
}

func (c *DockerClient) GetResourcesInfoFromEvents() (chan NetworkInfo, chan ContainerInfo) {
	networks := make(chan NetworkInfo)
	containers := make(chan ContainerInfo)

	go func() {
		for {
			msgs, errs := c.docker.Events(context.Background(), dockerTypes.EventsOptions{})

			for {
				select {
				case err := <-errs:
					c.logger.Printf("error: %+v\n", err)
					if err == io.EOF || err == nil {
						c.logger.Println("EOF received from events channel, shutdown")
						close(networks)
						close(containers)
						return
					}
					break

				case msg := <-msgs:
					kind, knownKind := resourceKindByString[msg.Type]
					action, knownAction := resourceActionByString[msg.Action]
					if !knownKind || !knownAction {
						continue
					}

					id := msg.Actor.ID
					c.logger.Printf("new docker event: %+v\n", msg)

					switch kind {
					case resourceKindNetwork:
						switch action {
						case resourceActionCreate:
							dockerSubnets, err := c.getNetworkSubnets(id)
							if err != nil {
								c.logger.Printf("error inspecting network '%s': %+v\n", id, err)
							}

							subnets := []Subnet{}
							for _, dockerSubnet := range dockerSubnets {
								nextSecondOctet := c.markAndGetFirstAvailableHostSecondOctet()
								hostProxyTwoOctets := fmt.Sprintf("%d.%d", hostProxySubnetFirstOctet, nextSecondOctet)
								proxySubnet := dockerSubnetOctetReplRegex.ReplaceAllLiteralString(dockerSubnet, hostProxyTwoOctets)
								subnet := Subnet{DockerSubnet: dockerSubnet, HostProxySubnet: proxySubnet, SecondOctet: nextSecondOctet}
								subnets = append(subnets, subnet)
							}

							networkInfo := NetworkInfo{ID: id, Action: action, Subnets: subnets}
							c.cache.AddNetworkInfo(networkInfo)
							networks <- networkInfo
							break

						case resourceActionDestroy:
							info, ok := c.cache.GetNetworkInfo(id)
							if !ok {
								c.logger.Printf("destroy event received for unknown network: %s\n", id)
							} else {
								for _, subnet := range info.Subnets {
									usedHostSecondOctet[subnet.SecondOctet] = false
								}

								c.cache.DeleteNetworkInfo(id)
								info.Action = action
								networks <- info
							}
							break
						}
						break

					case resourceKindContainer:
						switch action {
						case resourceActionStart:
							inspectInfo, err := c.getContainerInspectInfo(id)
							if err != nil {
								c.logger.Printf("error inspecting container '%s': %+v\n", id, err)
								continue
							}

							ips := []string{}
							for _, containerNetwork := range inspectInfo.Networks {
								network, ok := c.cache.GetNetworkInfo(containerNetwork.ID)
								if !ok {
									c.logger.Printf("container network '%s' not registered", containerNetwork.ID)
									continue
								}

								for _, subnet := range network.Subnets {
									ipSecondOctet := subnet.SecondOctet
									hostProxyTwoOctets := fmt.Sprintf("%d.%d", hostProxySubnetFirstOctet, ipSecondOctet)
									proxyIP := dockerSubnetOctetReplRegex.ReplaceAllLiteralString(containerNetwork.IP, hostProxyTwoOctets)
									ips = append(ips, proxyIP)
								}
							}

							containerInfo := ContainerInfo{ID: id, Action: action, Pid: inspectInfo.Pid, Ports: inspectInfo.Ports, IPs: ips}
							c.cache.AddContainerInfo(containerInfo)
							containers <- containerInfo
							break

						case resourceActionStop:
							if info, ok := c.cache.DeleteContainerInfo(id); ok {
								info.Action = action
								containers <- info
							}
							break
						}
						break
					}
				}
			}
		}
	}()

	return networks, containers
}

func (c *DockerClient) getNetworkSubnets(id string) ([]string, error) {
	info, err := c.docker.NetworkInspect(context.Background(), id)
	if err != nil {
		return nil, err
	}

	subnets := []string{}
	for _, config := range info.IPAM.Config {
		if config.Subnet != "" {
			subnets = append(subnets, config.Subnet)
		}
	}

	return subnets, nil
}

func (c *DockerClient) markAndGetFirstAvailableHostSecondOctet() int {
	available := false
	i := 1

	for ; i < len(usedHostSecondOctet); i++ {
		if !usedHostSecondOctet[i] {
			available = true
			usedHostSecondOctet[i] = true
			break
		}
	}
	if !available {
		c.logger.Panic("no available host proxy subnet")
	}

	return i
}

func (c *DockerClient) getContainerInspectInfo(id string) (ContainerInspectInfo, error) {
	inspect, err := c.docker.ContainerInspect(context.Background(), id)
	if err != nil {
		return ContainerInspectInfo{}, err
	}

	ports := []Port{}
	for port := range inspect.NetworkSettings.Ports {
		ports = append(ports, Port{
			Proto: port.Proto(),
			Port:  port.Int(),
		})
	}

	info := ContainerInspectInfo{
		Pid:   inspect.State.Pid,
		Ports: ports,
		Networks: []struct {
			ID string
			IP string
		}{},
	}
	for _, endpoint := range inspect.NetworkSettings.Networks {
		info.Networks = append(info.Networks, struct {
			ID string
			IP string
		}{
			ID: endpoint.NetworkID,
			IP: endpoint.IPAddress,
		})
	}

	return info, nil
}
