//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (app *appContext) HardRestart() error {
	defer func() {
		quit := make(chan os.Signal, 0)
		if r := recover(); r != nil {
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			<-quit
		}
	}()
	args := os.Args
	// After a single restart, args[0] gets messed up and isnt the real executable.
	// JFA_DEEP tells the new process its a child, and JFA_EXEC is the real executable
	if os.Getenv("JFA_DEEP") == "" {
		os.Setenv("JFA_DEEP", "1")
		os.Setenv("JFA_EXEC", args[0])
	}
	env := os.Environ()
	err := syscall.Exec(os.Getenv("JFA_EXEC"), []string{""}, env)
	if err != nil {
		return err
	}
	panic(fmt.Errorf("r"))
}
