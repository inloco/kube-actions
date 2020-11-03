package main

import (
	"log"
	"os"

	"github.com/inloco/docker-iptables-glue/docker"
	"github.com/inloco/docker-iptables-glue/portproxy"
	"github.com/inloco/docker-iptables-glue/common"

	"github.com/coreos/go-iptables/iptables"
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
	cache = common.NewCache()
	logger = log.New(os.Stdout, "inloco-docker-agent: ", 0)
)

func main() {
	logger.Println("creating docker client")
	docker, err := docker.New(logger, cache)
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("waiting for dockerd")
	err = docker.WaitForDockerd()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("creating iptables handle")
	iptables, err := iptables.New()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("creating socat port proxy")
	portProxy := portproxy.New()

	logger.Println("waiting docker events")
	networks, containers := docker.GetResourcesInfoFromEvents()
	for networks != nil && containers != nil {
		select {
		case networkInfo, ok := <-networks:
			if !ok {
				networks = nil
				continue
			}

			switch networkInfo.Action {
			case resourceActionCreate:
				logger.Printf("network created: %+v\n", networkInfo)
				if err := setupNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Printf("error in network port-forward setup: %+v\n", err)
				}
				break

			case resourceActionDestroy:
				logger.Printf("network destroyed: %+v\n", networkInfo)
				if err := setdownNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Printf("error in network port-forward setdown: %+v\n", err)
				}
				break
			}
			break

		case containerInfo, ok := <-containers:
			if !ok {
				containers = nil
				continue
			}

			switch containerInfo.Action {
			case resourceActionStart:
				logger.Printf("container started: %+v\n", containerInfo)
				if err := setupContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Printf("error in port-forward setup: %+v\n", err)
				}
				break

			case resourceActionStop:
				logger.Printf("container stopped: %+v\n", containerInfo)
				if err := setdownContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Printf("error in port-forward setdown: %+v\n", err)
				}
			}
			break
		}
	}
}

func setupNetworkPortForward(iptables *iptables.IPTables, info common.NetworkInfo) error {
	for _, subnet := range info.Subnets {
		// iptables -t nat -A OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.AppendUnique("nat", "OUTPUT", "-d", subnet.DockerSubnet, "-j", "NETMAP", "--to", subnet.HostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setdownNetworkPortForward(iptables *iptables.IPTables, info common.NetworkInfo) error {
	for _, subnet := range info.Subnets {
		// iptables -t nat -D OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.Delete("nat", "OUTPUT", "-d", subnet.DockerSubnet, "-j", "NETMAP", "--to", subnet.HostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setupContainerPortProxy(portProxy portproxy.Client, info common.ContainerInfo) error {
	for _, ip := range info.IPs {
		for _, spec := range info.Ports {
			request := portproxy.AddPortProxyRequest{
				Proto:     spec.Proto,
				HostIP:    ip,
				HostPort:  spec.Port,
				ChildPID:  info.Pid,
				ChildPort: spec.Port,
			}

			logger.Printf("portproxy: sending AddPort request: %+v\n", request)
			pid, err := portProxy.AddPortProxy(request)
			if err != nil {
				return err
			}

			cache.AddProxyPortPID(info.ID, pid)
		}
	}

	return nil
}

func setdownContainerPortProxy(portProxy portproxy.Client, info common.ContainerInfo) error {
	for _, pid := range cache.GetProxyPortPIDs(info.ID) {
		logger.Printf("portproxy: sending RemovePortProxy(%d) request\n", pid)
		if err := portProxy.RemovePortProxy(pid); err != nil {
			return err
		}
	}

	cache.DeleteProxyPortPIDs(info.ID)

	return nil
}
