//go:build windows

// Package process provides process-group management for command execution.
package process

import (
	"fmt"
	"os/exec"
)

// SetGroup is a no-op on Windows.
// Process-group isolation via Setpgid is not available on Windows.
func SetGroup(_ *exec.Cmd) {}

// KillGroup kills the process on Windows (no group support).
// Falls back to killing just the process itself.
func KillGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return ErrProcessNotStarted
	}

	err := cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrKillGroup, err)
	}

	return nil
}
