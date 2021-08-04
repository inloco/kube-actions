package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
)

var (
	cache  = NewCache()
	logger = log.New(os.Stdout, "kube-actions[dind]: ", log.LstdFlags)
)

func main() {
	logger.Println("listening for unix signals")
	listenForUnixSignals()

	logger.Println("listening for interrupt")
	if err := listenForInterrupt(); err != nil {
		logger.Panic(err)
	}

	logger.Println("creating docker client")
	docker, err := NewDockerClient(logger, cache)
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("patching runtime dirs")
	if err := docker.PatchRuntimeDirs(); err != nil {
		logger.Panic(err)
	}

	logger.Println("starting dockerd")
	cmdWait, err := docker.StartDockerd()
	if err != nil {
		logger.Printf("error starting dockerd: %+v\n", err)
	}

	logger.Println("waiting for dockerd")
	if err := docker.WaitForDockerd(); err != nil {
		logger.Printf("error waiting for dockerd: %+v\n", err)
	}

	logger.Println("creating iptables client")
	iptables, err := iptables.New()
	if err != nil {
		logger.Panic(err)
	}

	logger.Println("creating port proxy")
	portProxy := NewPortProxyClient()

	logger.Println("waiting for docker events")
	networks, containers := docker.GetResourcesInfoFromEvents()

Loop:
	for {
		select {
		case err := <-cmdWait:
			if err == nil {
				logger.Println("dockerd stopped with no error")
				break Loop
			}
			if exitError, ok := err.(*exec.ExitError); ok {
				if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
					logger.Printf("docker stopped with status: %d\n", waitStatus.ExitStatus())
					os.Exit(waitStatus.ExitStatus())
				}
			}
			logger.Panicf("dockerd stopped with unknown status: %+v\n", err)

		case networkInfo, ok := <-networks:
			if !ok {
				break Loop
			}
			switch networkInfo.Action {
			case resourceActionCreate:
				logger.Printf("network created: %+v\n", networkInfo)
				if err := setupNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Panic(fmt.Errorf("error in network port-forward setup: %w", err))
				}
			case resourceActionDestroy:
				logger.Printf("network destroyed: %+v\n", networkInfo)
				if err := setdownNetworkPortForward(iptables, networkInfo); err != nil {
					logger.Print(fmt.Errorf("error in network port-forward setdown: %w", err))
				}
			}

		case containerInfo, ok := <-containers:
			if !ok {
				break Loop
			}
			switch containerInfo.Action {
			case resourceActionStart:
				logger.Printf("container started: %+v\n", containerInfo)
				if err := setupContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Panic(fmt.Errorf("error in port-forward setup: %w", err))
				}
			case resourceActionStop:
				logger.Printf("container stopped: %+v\n", containerInfo)
				if err := setdownContainerPortProxy(portProxy, containerInfo); err != nil {
					logger.Panic(fmt.Errorf("error in port-forward setdown: %w", err))
				}
			}
		}
	}
}

func listenForUnixSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range signals {
			logger.Printf("received unix signal: %s", sig)
		}
	}()
}

func listenForInterrupt() error {
	listener, err := net.Listen("tcp", ":2378")
	if err != nil {
		return err
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			logger.Panic(err)
		}

		for _, closer := range []io.Closer{conn, listener} {
			if err := closer.Close(); err != nil {
				logger.Panic(err)
			}
		}

		logger.Println("exiting dind after tcp interrupt")
		os.Exit(0)
	}()

	return nil
}

func setupNetworkPortForward(iptables *iptables.IPTables, info NetworkInfo) error {
	for _, subnet := range info.Subnets {
		// iptables -t nat -A OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.AppendUnique("nat", "OUTPUT", "-d", subnet.DockerSubnet, "-j", "NETMAP", "--to", subnet.HostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setdownNetworkPortForward(iptables *iptables.IPTables, info NetworkInfo) error {
	for _, subnet := range info.Subnets {
		// iptables -t nat -D OUTPUT -d 172.16.0.0/16 -j NETMAP --to 127.1.0.0/16
		if err := iptables.Delete("nat", "OUTPUT", "-d", subnet.DockerSubnet, "-j", "NETMAP", "--to", subnet.HostProxySubnet); err != nil {
			return err
		}
	}

	return nil
}

func setupContainerPortProxy(portProxy *PortProxyClient, info ContainerInfo) error {
	for _, ip := range info.IPs {
		for _, spec := range info.Ports {
			request := AddPortProxyRequest{
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

func setdownContainerPortProxy(portProxy *PortProxyClient, info ContainerInfo) error {
	for _, pid := range cache.GetProxyPortPIDs(info.ID) {
		logger.Printf("portproxy: sending RemovePortProxy(%d) request\n", pid)
		if err := portProxy.RemovePortProxy(pid); err != nil {
			return err
		}
	}

	cache.DeleteProxyPortPIDs(info.ID)

	return nil
}
