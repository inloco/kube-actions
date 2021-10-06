package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
)

func run(command string, args ...string) chan error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGINT}

	ret := make(chan error)
	running := make(chan error)

	if err := cmd.Start(); err != nil {
		ret <-errors.Wrapf(err, "Error starting command [%s %s]", command, args)
		return ret
	}

	signals := make(chan os.Signal, 1)
	go func() {
		for sig := range signals {
			select {
			case <-running:
				return
			default:
				break
			}
			cmd.Process.Signal(sig)
		}
	}()
	signal.Notify(signals)

	go func() {
		err := cmd.Wait()
		ret <- err
		running <- err
	}()

	return ret
}

func async(f func() error) chan error {
	c := make(chan error)
	go func() {
		c <- f()
	}()

	return c
}
