//go:build darwin

package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DarwinSandbox implements Sandbox using macOS sandbox-exec.
type DarwinSandbox struct {
	mode Mode
}

// NewSandbox creates a macOS sandbox.
func NewSandbox(mode Mode) (Sandbox, error) {
	// Check if sandbox-exec is available
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return &NoopSandbox{}, nil
	}

	return &DarwinSandbox{mode: mode}, nil
}

// Wrap applies Seatbelt sandbox to the command.
func (s *DarwinSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// Generate Seatbelt profile
	profile, err := s.generateProfile(opts)
	if err != nil {
		return fmt.Errorf("generate profile: %w", err)
	}

	// Write profile to temp file
	profilePath := filepath.Join(os.TempDir(), "spin-sandbox.sb")
	if err := os.WriteFile(profilePath, []byte(profile), 0600); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}

	// Wrap command with sandbox-exec
	originalPath := cmd.Path
	if originalPath == "" {
		originalPath = cmd.Args[0]
	}

	cmd.Path = "/usr/bin/sandbox-exec"
	cmd.Args = append([]string{"sandbox-exec", "-f", profilePath, originalPath}, cmd.Args[1:]...)

	return nil
}

// Supported returns true if sandbox-exec is available.
func (s *DarwinSandbox) Supported() bool {
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

// Mode returns the sandbox mode.
func (s *DarwinSandbox) Mode() Mode {
	return s.mode
}

// generateProfile creates a Seatbelt profile for the given options.
func (s *DarwinSandbox) generateProfile(opts SandboxOptions) (string, error) {
	profile := `(version 1)
(debug deny)
(allow default)

`

	if opts.Mode == ModeReadOnly || opts.Mode == ModeWorkspaceWrite {
		profile += ";; Deny all file writes by default\n"
		profile += "(deny file-write*)\n\n"
	}

	if len(opts.ReadPaths) > 0 {
		profile += ";; Allow read access\n"
		for _, path := range opts.ReadPaths {
			profile += fmt.Sprintf("(allow file-read* (subpath \"%s\"))\n", path)
		}
		profile += "\n"
	}

	if len(opts.WritePaths) > 0 {
		profile += ";; Allow write access\n"
		for _, path := range opts.WritePaths {
			profile += fmt.Sprintf("(allow file-write* (subpath \"%s\"))\n", path)
		}
		profile += "\n"
	}

	if opts.BlockNetwork {
		profile += ";; Block network\n"
		profile += "(deny network*)\n"
	}

	return profile, nil
}
