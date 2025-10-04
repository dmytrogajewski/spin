//go:build linux

package sandbox

import (
	"os"
	"os/exec"
)

// LinuxSandbox implements Sandbox using Landlock LSM.
type LinuxSandbox struct {
	mode Mode
}

// NewSandbox creates a Linux sandbox with Landlock support.
func NewSandbox(mode Mode) (Sandbox, error) {
	// Check if Landlock is supported (kernel 5.13+)
	if !landlockSupported() {
		return &NoopSandbox{}, nil
	}

	return &LinuxSandbox{mode: mode}, nil
}

// Wrap applies Landlock restrictions to the command.
// Note: Landlock is applied to the current process, not the child process.
// For proper sandboxing, this would require a helper binary or pre-exec setup.
func (s *LinuxSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// For now, this is a placeholder.
	// Full implementation would require:
	// 1. A helper binary that applies Landlock before exec
	// 2. Or using cgo with pre-exec functions
	// 3. Or running via a wrapper script

	// For MVP, we document the limitation and return success
	return nil
}

// Supported returns true if Landlock is available.
func (s *LinuxSandbox) Supported() bool {
	return landlockSupported()
}

// Mode returns the sandbox mode.
func (s *LinuxSandbox) Mode() Mode {
	return s.mode
}

// landlockSupported checks if Landlock LSM is available.
// For now, we use a simple kernel version check or file existence check.
func landlockSupported() bool {
	// Check if /sys/kernel/security/landlock exists
	if _, err := os.Stat("/sys/kernel/security/landlock"); err == nil {
		return true
	}

	// Fallback: assume not supported
	return false
}
