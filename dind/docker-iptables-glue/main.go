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

	"github.com/inloco/docker-iptables-glue/portproxy"

	"github.com/coreos/go-iptables/iptables"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

	networkInfoByID            map[string]networkInfo     = map[string]networkInfo{}
	containerInfoByID          map[string]containerInfo   = map[string]containerInfo{}
	proxyPortPIDsByContainerID map[string][]portproxy.PID = map[string][]portproxy.PID{}

	hostProxySubnetFirstOctet  = 127
	usedHostSecondOctet        = [256]bool{}
	dockerSubnetOctetReplRegex = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}`)

	dockerHost = strings.TrimSpace(os.ExpandEnv("${DOCKER_HOST}"))

	logger = log.New(os.Stdout, "inloco-docker-agent: ", 0)
)

type networkInfo struct {
	id      string
	action  int
	subnets []subnet
}

type subnet struct {
	dockerSubnet    string
	hostProxySubnet string
	secondOctet     int
}

type containerInfo struct {
	id     string
	action int
	pid    int
	ports  []portSpec
	ips    []string
}

type portSpec struct {
	proto string
	port  int
}

type containerInspectInfo struct {
	pid      int
	ports    []portSpec
	networks []struct {
		id string
		ip string
	}
}

func main() {
	if dockerHost == "" {
		logger.Panic("Empty DOCKER_HOST")
	}

	logger.Println("Waiting for dockerd")
	err := waitForDockerd()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("Creating iptables handle")
	iptables, err := iptables.New()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("Creating socat port proxy")
	portProxy := portproxy.New()

	logger.Println("Creating docker client")
	docker, err := client.NewEnvClient()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("Waiting docker events")
	networks, containers := getResourcesInfoFromEvents(docker)
	for networks != nil && containers != nil {
		select {
		case networkInfo, ok := <-networks:
			if !ok {
				networks = nil
				continue
			}

			switch networkInfo.action {
			case resourceActionCreate:
				logger.Println("Network created")
				if err := setupNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Printf("Error in network port-forward setup: %+v\n", err)
				}
				break

			case resourceActionDestroy:
				logger.Println("Network destroyed")
				if err := setdownNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Printf("Error in network port-forward setdown: %+v\n", err)
				}
				break
			}
			break

		case containerInfo, ok := <-containers:
			if !ok {
				containers = nil
				continue
			}

			switch containerInfo.action {
			case resourceActionStart:
				logger.Println("Container started")
				if err := setupContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Printf("Error in port-forward setup: %+v\n", err)
				}
				break

			case resourceActionStop:
				logger.Println("Container stopped")
				if err := setdownContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Printf("Error in port-forward setdown: %+v\n", err)
				}
			}
			break
		}
	}
}

func waitForDockerd() error {
	if !strings.HasPrefix(dockerHost, "tcp://") {
		return errors.New("DOCKER_HOST not tcp://")
	}

	connected := false
	ipAndPort := strings.TrimPrefix(dockerHost, "tcp://")
	for i := 0; i < 15; i++ {
		logger.Printf("Trying to connect to dockerd on %s\n", ipAndPort)
		conn, err := net.DialTimeout("tcp", ipAndPort, time.Second)
		if err == nil && conn != nil {
			defer conn.Close()
			logger.Printf("Connected to dockerd successfully")
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

func getResourcesInfoFromEvents(docker *client.Client) (chan networkInfo, chan containerInfo) {
	networks := make(chan networkInfo)
	containers := make(chan containerInfo)

	go func() {
		for {
			msgs, errs := docker.Events(context.Background(), types.EventsOptions{})

			for {
				select {
				case err := <-errs:
					logger.Printf("error: %+v\n", err)
					if err == io.EOF || err == nil {
						logger.Println("EOF received from events channel, shutdown")
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
					logger.Println("resource:", msg.Type, "id: ", id, "event: ", msg.Action)

					switch kind {
					case resourceKindNetwork:
						switch action {
						case resourceActionCreate:
							dockerSubnets, err := getNetworkSubnets(docker, id)
							if err != nil {
								logger.Printf("error inspecting network '%s': %+v\n", id, err)
							}

							subnets := []subnet{}
							for _, dockerSubnet := range dockerSubnets {
								nextSecondOctet := markAndGetFirstAvailableHostSecondOctet()
								hostProxyTwoOctets := fmt.Sprintf("%d.%d", hostProxySubnetFirstOctet, nextSecondOctet)
								proxySubnet := dockerSubnetOctetReplRegex.ReplaceAllLiteralString(dockerSubnet, hostProxyTwoOctets)
								subnet := subnet{dockerSubnet: dockerSubnet, hostProxySubnet: proxySubnet, secondOctet: nextSecondOctet}
								subnets = append(subnets, subnet)
							}

							networkInfo := networkInfo{id: id, action: action, subnets: subnets}
							networkInfoByID[id] = networkInfo
							networks <- networkInfo
							break

						case resourceActionDestroy:
							info, ok := networkInfoByID[id]
							if !ok {
								logger.Printf("Stop event received for unknown network: %s\n", id)
							} else {
								for _, subnet := range info.subnets {
									usedHostSecondOctet[subnet.secondOctet] = false
								}

								delete(networkInfoByID, id)
								info.action = action
								networks <- info
							}
							break
						}
						break

					case resourceKindContainer:
						switch action {
						case resourceActionStart:
							inspectInfo, err := getContainerInspectInfo(docker, id)
							if err != nil {
								logger.Printf("error inspecting container '%s': %+v\n", id, err)
								continue
							}

							ips := []string{}
							for _, containerNetwork := range inspectInfo.networks {
								network, ok := networkInfoByID[containerNetwork.id]
								if !ok {
									logger.Printf("container network '%s' not registered", containerNetwork.id)
									continue
								}

								for _, subnet := range network.subnets {
									ipSecondOctet := subnet.secondOctet
									hostProxyTwoOctets := fmt.Sprintf("%d.%d", hostProxySubnetFirstOctet, ipSecondOctet)
									proxyIP := dockerSubnetOctetReplRegex.ReplaceAllLiteralString(containerNetwork.ip, hostProxyTwoOctets)
									ips = append(ips, proxyIP)
								}
							}

							containerInfo := containerInfo{id: id, action: action, pid: inspectInfo.pid, ports: inspectInfo.ports, ips: ips}
							containerInfoByID[id] = containerInfo
							proxyPortPIDsByContainerID[id] = []portproxy.PID{}
							containers <- containerInfo
							break

						case resourceActionStop:
							info, ok := containerInfoByID[id]
							if !ok {
								logger.Printf("Stop received of unknown container: %s\n", id)
							} else {
								delete(containerInfoByID, id)
								info.action = action
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

func getNetworkSubnets(docker *client.Client, id string) ([]string, error) {
	info, err := docker.NetworkInspect(context.Background(), id)
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

func markAndGetFirstAvailableHostSecondOctet() int {
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
		logger.Panic("No available host proxy subnet")
	}

	return i
}

func getContainerInspectInfo(docker *client.Client, id string) (containerInspectInfo, error) {
	inspect, err := docker.ContainerInspect(context.Background(), id)
	if err != nil {
		return containerInspectInfo{}, err
	}

	ports := []portSpec{}
	for port := range inspect.NetworkSettings.Ports {
		ports = append(ports, portSpec{
			proto: port.Proto(),
			port:  port.Int(),
		})
	}

	info := containerInspectInfo{
		pid:   inspect.State.Pid,
		ports: ports,
		networks: []struct {
			id string
			ip string
		}{},
	}
	for _, endpoint := range inspect.NetworkSettings.Networks {
		info.networks = append(info.networks, struct {
			id string
			ip string
		}{
			id: endpoint.NetworkID,
			ip: endpoint.IPAddress,
		})
	}

	return info, nil
}

func setupNetworkPortForward(iptables *iptables.IPTables, info networkInfo) error {
	for _, subnet := range info.subnets {
		// iptables -t nat -A OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.AppendUnique("nat", "OUTPUT", "-d", subnet.dockerSubnet, "-j", "NETMAP", "--to", subnet.hostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setdownNetworkPortForward(iptables *iptables.IPTables, info networkInfo) error {
	for _, subnet := range info.subnets {
		// iptables -t nat -D OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.Delete("nat", "OUTPUT", "-d", subnet.dockerSubnet, "-j", "NETMAP", "--to", subnet.hostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setupContainerPortProxy(portProxy portproxy.Client, info containerInfo) error {
	for _, ip := range info.ips {
		for _, spec := range info.ports {
			request := portproxy.AddPortProxyRequest{
				Proto:     spec.proto,
				HostIP:    ip,
				HostPort:  spec.port,
				ChildPID:  portproxy.PID(info.pid),
				ChildPort: spec.port,
			}

			logger.Printf("portproxy: sending AddPort request: %+v\n", request)
			pid, err := portProxy.AddPortProxy(request)
			if err != nil {
				return err
			}

			proxyPortPIDsByContainerID[info.id] = append(proxyPortPIDsByContainerID[info.id], pid)
		}
	}

	return nil
}

func setdownContainerPortProxy(portProxy portproxy.Client, info containerInfo) error {
	for _, pid := range proxyPortPIDsByContainerID[info.id] {
		logger.Printf("portproxy: sending RemovePortProxy(%d) request\n", pid)
		if err := portProxy.RemovePortProxy(pid); err != nil {
			return err
		}
	}

	proxyPortPIDsByContainerID[info.id] = []portproxy.PID{}

	return nil
}
