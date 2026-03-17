//go:build !windows

// Package process provides process-group management for command execution.
package process

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

// ErrProcessNotStarted is returned when trying to kill a process that hasn't been started.
var ErrProcessNotStarted = errors.New("process not started")

// ErrKillGroup is returned when killing the process group fails.
var ErrKillGroup = errors.New("failed to kill process group")

// SetGroup configures cmd to run in its own process group.
// This ensures that killing the group kills all child processes.
func SetGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.Setpgid = true
}

// KillGroup kills the entire process group of cmd.
// Sends SIGKILL to the negative PID (process group leader).
func KillGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return ErrProcessNotStarted
	}

	// Negative PID targets the entire process group.
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrKillGroup, err)
	}

	return nil
}
