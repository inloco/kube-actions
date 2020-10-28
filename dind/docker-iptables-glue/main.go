package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/fsnotify/fsnotify"
)

const (
	containerStart = iota
	containerStop
	containerDestroy
)

var (
	containerInfoByID map[string]containerInfo = map[string]containerInfo{}

	dockerSocketEnv  = os.ExpandEnv("${DOCKER_HOST}")
	dockerSocketPath = strings.TrimPrefix(dockerSocketEnv, "unix://")
	dockerSocketDir  = path.Dir(dockerSocketPath)
)

type containerInfo struct {
	ID    string
	kind  int
	ports []portSpec
}

type portSpec struct {
	proto, containerPort, hostPort string
}

func main() {
	fmt.Printf("UID: %d\n", os.Getuid())

	fmt.Println("Waiting for docker socket")
	err := waitForSocket()
	if err != nil {
		panic(err)
	}

	fmt.Println("Creating iptables handle")
	iptables, err := iptables.New()
	if err != nil {
		panic(err)
	}

	fmt.Println("Creating docker client")
	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	fmt.Println("Waiting docker events")
	infos := getNewContainersInfo(docker)
	for info := range infos {
		switch info.kind {
		case containerStart:
			fmt.Println("Container started")
			if err := setupContainerPortForward(iptables, docker, info); err != nil {
				fmt.Printf("Error in port-forward setup: %s\n", err)
			}
			break
		case containerStop:
		case containerDestroy:
			fmt.Println("Container stopped")
			if err := setdownContainerPortForward(iptables, docker, info); err != nil {
				fmt.Printf("Error in port-forward setdown: %s\n", err)
			}
		}
	}
}

func waitForSocket() error {
	if _, err := os.Stat(dockerSocketPath); err == nil {
		fmt.Println("Docker socket already exists")
		return nil
	}

	fmt.Println("Creating filesystem watcher for docker socket")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer watcher.Close()

	err = watcher.Add(dockerSocketDir)
	if err != nil {
		return err
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("docker socket watcher failed")
			}
			fmt.Printf("New event from docker socket watcher: %+v\n", event)
			if event.Op == fsnotify.Create && event.Name == dockerSocketPath {
				fmt.Println("Docker socket detected")
				return nil
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("docker socket watcher failed")
			}
			return fmt.Errorf("error in docker socket watcher: %+v", err)
		}
	}
}

func getNewContainersInfo(docker *client.Client) chan containerInfo {
	filters := filters.NewArgs()
	filters.Add("type", "container")

	c := make(chan containerInfo)

	go func() {
		for {
			msgs, errs := docker.Events(context.Background(), types.EventsOptions{
				Filters: filters,
			})

			for {
				select {
				case err := <-errs:
					fmt.Printf("error: %+v\n", err)
					if err == io.EOF {
						fmt.Println("EOF received from events channel, shutdown")
						close(c)
						return
					}
					break
				case msg := <-msgs:
					println("container: ", msg.ID, "event: ", msg.Action)

					var kind int
					switch msg.Action {
					case "start":
						kind = containerStart
						break
					case "destroy":
						kind = containerDestroy
						break
					case "stop":
						kind = containerStop
						break
					default:
						continue
					}

					if kind == containerStart {
						info, err := docker.ContainerInspect(context.Background(), msg.ID)
						if err != nil {
							fmt.Printf("error inspecting container: %+v\n", err)
							continue
						}
						ports := []portSpec{}
						for port, bindings := range info.NetworkSettings.Ports {
							for _, binding := range bindings {
								ports = append(ports, portSpec{
									proto:         port.Proto(),
									containerPort: port.Port(),
									hostPort:      binding.HostPort,
								})
							}
						}

						containerInfo := containerInfo{ID: msg.ID, kind: kind, ports: ports}
						containerInfoByID[msg.ID] = containerInfo
						c <- containerInfo
					} else {
						info, ok := containerInfoByID[msg.ID]
						if !ok {
							fmt.Printf("Uknown container event: %s\n", msg.ID)
						}
						if kind == containerDestroy {
							delete(containerInfoByID, msg.ID)
						}
						c <- info
					}
				}
			}
		}
	}()

	return c
}

func setupContainerPortForward(iptables *iptables.IPTables, docker *client.Client, info containerInfo) error {
	for _, spec := range info.ports {
		// sudo iptables -t nat -A OUTPUT -p tcp --dport 6379 -j DNAT --to-destination 127.0.0.1:8080
		destination := "127.0.0.1:" + spec.hostPort
		if err := iptables.AppendUnique("nat", "OUTPUT", "-p", spec.proto, "--dport", spec.containerPort, "-j", "DNAT", "--to-destination", destination); err != nil {
			return err
		}
	}

	return nil
}

func setdownContainerPortForward(iptables *iptables.IPTables, docker *client.Client, info containerInfo) error {
	for _, spec := range info.ports {
		// sudo iptables -t nat -A OUTPUT -p tcp --dport 6379 -j DNAT --to-destination 127.0.0.1:8080
		destination := "127.0.0.1:" + spec.hostPort
		if err := iptables.Delete("nat", "OUTPUT", "-p", spec.proto, "--dport", spec.containerPort, "-j", "DNAT", "--to-destination", destination); err != nil {
			return err
		}
	}

	return nil
}
