package portproxy

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

var (
	logger = log.New(os.Stdout, "inloco-docker-agent: portproxy: ", 0)
)

type AddPortProxyRequest struct {
	Proto     string
	HostIP    string
	HostPort  int
	ChildPID  int
	ChildPort int
}

type Client interface {
	AddPortProxy(request AddPortProxyRequest) (int, error)
	RemovePortProxy(pid int) error
}

type client struct {
}

func New() Client {
	return &client{}
}

func (c *client) AddPortProxy(request AddPortProxyRequest) (int, error) {
	cmd := createSocatCommand(request)

	pidc := make(chan int)
	errc := make(chan error)

	go func(cmd *exec.Cmd) {
		if err := cmd.Start(); err != nil {
			errc <- err
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

func (c *client) RemovePortProxy(pid int) error {
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
