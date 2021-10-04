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

	if err := cmd.Start(); err != nil {
		ret <-errors.Wrapf(err, "Error starting command [%s %s]", command, args)
		return ret
	}

	signals := make(chan os.Signal, 1)
	go func() {
		for sig := range signals {
			if err := cmd.Process.Signal(sig); err != nil {
				logger.Printf("Error sending signal for command [%s %s]: %v\n", command, args, err)
			}
		}
	}()
	signal.Notify(signals)

	go func() {
		ret <- cmd.Wait()
	}()

	return ret
}
