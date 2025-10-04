//go:build darwin

package hardening

import (
	"fmt"
	"syscall"
)

// disableCoreDumps prevents core dump generation on macOS.
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

// disablePtrace prevents debugger attachment on macOS.
// Note: This would require calling ptrace(PT_DENY_ATTACH) via cgo.
// For now, we return nil as a stub.
func disablePtrace() error {
	// TODO: Implement PT_DENY_ATTACH via cgo
	// For now, return nil to avoid build errors
	return nil
}
