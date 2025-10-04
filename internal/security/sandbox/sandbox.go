package sandbox

import (
	"os/exec"
)

// Sandbox provides OS-level command isolation.
type Sandbox interface {
	// Wrap wraps a command with sandbox restrictions
	Wrap(cmd *exec.Cmd, opts SandboxOptions) error

	// Supported returns true if sandbox is available on this platform
	Supported() bool

	// Mode returns the current sandbox mode
	Mode() Mode
}

// Mode defines the sandbox restriction level.
type Mode int

const (
	// ModeReadOnly allows reading workspace, no writes
	ModeReadOnly Mode = iota

	// ModeWorkspaceWrite allows writes within workspace
	ModeWorkspaceWrite

	// ModeFullAccess disables sandbox (for containers)
	ModeFullAccess
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeReadOnly:
		return "read-only"
	case ModeWorkspaceWrite:
		return "workspace-write"
	case ModeFullAccess:
		return "full-access"
	default:
		return "unknown"
	}
}

// SandboxOptions configures sandbox behavior.
type SandboxOptions struct {
	// Mode is the restriction level
	Mode Mode

	// ReadPaths are paths allowed for reading
	ReadPaths []string

	// WritePaths are paths allowed for writing
	WritePaths []string

	// BlockNetwork blocks network access
	BlockNetwork bool

	// WorkDir is the working directory
	WorkDir string
}

// NoopSandbox is a fallback sandbox that does nothing.
// Used when platform sandboxing is not available.
type NoopSandbox struct{}

// Wrap does nothing in the no-op sandbox.
func (s *NoopSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// No-op: command runs without restrictions
	return nil
}

// Supported always returns false for no-op sandbox.
func (s *NoopSandbox) Supported() bool {
	return false
}

// Mode returns FullAccess for no-op sandbox.
func (s *NoopSandbox) Mode() Mode {
	return ModeFullAccess
}
