package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type AddPortProxyRequest struct {
	Proto     string
	HostIP    string
	HostPort  int
	ChildPID  int
	ChildPort int
}

type PortProxyClient struct {
}

func NewPortProxyClient() *PortProxyClient {
	return &PortProxyClient{}
}

func (c *PortProxyClient) AddPortProxy(request AddPortProxyRequest) (int, error) {
	cmd := createSocatCommand(request)

	pidc := make(chan int)
	errc := make(chan error)

	go func(cmd *exec.Cmd) {
		if err := cmd.Start(); err != nil {
			errc <- err
			return
		}

		pidc <- int(cmd.Process.Pid)
		cmd.Wait()
	}(cmd)

	select {
	case pid := <-pidc:
		return pid, nil
	case err := <-errc:
		return 0, err
	}
}

func (c *PortProxyClient) RemovePortProxy(pid int) error {
	return syscall.Kill(int(pid), syscall.SIGKILL)
}

func createSocatCommand(request AddPortProxyRequest) *exec.Cmd {
	var cmd *exec.Cmd

	switch request.Proto {
	case "tcp":
		cmd = exec.CommandContext(context.Background(),
			"socat",
			fmt.Sprintf("TCP-LISTEN:%d,bind=%s,reuseaddr,fork,rcvbuf=65536,sndbuf=65536", request.HostPort, request.HostIP),
			fmt.Sprintf("EXEC:\"%s\",nofork",
				fmt.Sprintf("nsenter -U -n --preserve-credentials -t %d socat STDIN TCP4:127.0.0.1:%d", request.ChildPID, request.ChildPort)))
	case "udp":
		cmd = exec.CommandContext(context.Background(),
			"socat",
			fmt.Sprintf("UDP-LISTEN:%d,bind=%s,reuseaddr,fork,rcvbuf=65536,sndbuf=65536", request.HostPort, request.HostIP),
			fmt.Sprintf("EXEC:\"%s\",nofork",
				fmt.Sprintf("nsenter -U -n --preserve-credentials -t %d socat STDIN UDP4:127.0.0.1:%d", request.ChildPID, request.ChildPort)))
	}

	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return cmd
}
