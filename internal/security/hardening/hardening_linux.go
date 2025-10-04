//go:build linux

package hardening

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// disableCoreDumps prevents core dump generation on Linux.
func disableCoreDumps() error {
	err := syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{
		Cur: 0,
		Max: 0,
	})
	if err != nil {
		return fmt.Errorf("setrlimit: %w", err)
	}
	return nil
}

// disablePtrace prevents debugger attachment on Linux.
func disablePtrace() error {
	// PR_SET_DUMPABLE with 0 prevents ptrace
	err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("prctl: %w", err)
	}
	return nil
}
