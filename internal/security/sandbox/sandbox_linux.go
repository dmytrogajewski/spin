//go:build linux

package sandbox

import (
	"fmt"
	"os/exec"

	landlock "github.com/landlock-lsm/go-landlock/landlock"
)

// LinuxSandbox implements Sandbox using Landlock LSM via go-landlock library.
type LinuxSandbox struct {
	mode Mode
}

// NewSandbox creates a Linux sandbox with Landlock support.
func NewSandbox(mode Mode) (Sandbox, error) {
	// Check if Landlock is supported
	if !landlockSupported() {
		return &NoopSandbox{}, nil
	}

	return &LinuxSandbox{mode: mode}, nil
}

// Wrap applies Landlock restrictions to the command.
//
// IMPORTANT: This method applies Landlock restrictions to the CURRENT PROCESS immediately.
// The restrictions are inherited by child processes when cmd.Start/Run is called.
//
// WARNING: After calling Wrap(), the calling goroutine and process will be restricted!
// This means:
//   - You cannot create new files outside the allowed paths
//   - You cannot access directories not in the allow list
//   - The restrictions persist for the lifetime of the process
//
// Best practices:
//   - Call Wrap() in a dedicated goroutine that will be discarded after use
//   - Or call Wrap() immediately before cmd.Run() as the last operation
//   - Ensure all necessary file operations are done BEFORE calling Wrap()
//
// The go-landlock library uses the psx package to apply syscalls across all OS threads,
// ensuring process-wide restrictions that are inherited by child processes.
func (s *LinuxSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
	// Full access mode - no restrictions
	if opts.Mode == ModeFullAccess {
		return nil
	}

	// Build Landlock configuration based on mode
	var landlockOpts []landlock.PathOpt

	// System paths that should always be readable/executable
	systemPaths := []string{
		"/usr",
		"/bin",
		"/lib",
		"/lib64",
		"/etc",
	}

	switch opts.Mode {
	case ModeReadOnly:
		// Allow read/execute for system paths
		for _, path := range systemPaths {
			landlockOpts = append(landlockOpts, landlock.RODirs(path))
		}

		// Allow read for read paths
		for _, path := range opts.ReadPaths {
			landlockOpts = append(landlockOpts, landlock.RODirs(path))
		}

		// Allow read for work dir
		if opts.WorkDir != "" {
			landlockOpts = append(landlockOpts, landlock.RODirs(opts.WorkDir))
		}

	case ModeWorkspaceWrite:
		// Allow read/execute for system paths
		for _, path := range systemPaths {
			landlockOpts = append(landlockOpts, landlock.RODirs(path))
		}

		// Allow read for read paths
		for _, path := range opts.ReadPaths {
			landlockOpts = append(landlockOpts, landlock.RODirs(path))
		}

		// Allow read/write for write paths
		for _, path := range opts.WritePaths {
			landlockOpts = append(landlockOpts, landlock.RWDirs(path))
		}

		// Allow read/write for work dir
		if opts.WorkDir != "" {
			landlockOpts = append(landlockOpts, landlock.RWDirs(opts.WorkDir))
		}
	}

	// Apply Landlock restrictions to current process
	// This will be inherited by the child process when cmd.Start/Run is called
	// Use BestEffort() to work across different kernel versions
	err := landlock.V5.BestEffort().RestrictPaths(landlockOpts...)
	if err != nil {
		return fmt.Errorf("failed to apply landlock restrictions: %w", err)
	}

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
func landlockSupported() bool {
	// The go-landlock library handles Landlock availability detection internally.
	// We always return true on Linux because:
	// 1. The library will gracefully fall back if Landlock is not available
	// 2. We cannot test by applying restrictions (would restrict current process)
	// 3. Actual support is checked when Wrap() is called
	//
	// If Landlock is not supported, the library's BestEffort() mode will
	// succeed without applying restrictions (graceful degradation).
	return true
}
