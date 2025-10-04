//go:build darwin

package hardening

/*
#include <sys/types.h>
#include <sys/ptrace.h>

// Wrapper to call ptrace with PT_DENY_ATTACH
int deny_ptrace_attach(void) {
    return ptrace(PT_DENY_ATTACH, 0, 0, 0);
}
*/
import "C"
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
// It calls ptrace(PT_DENY_ATTACH) to prevent debuggers from attaching to this process.
// This is a security measure to prevent tampering with the running process.
func disablePtrace() error {
	ret := C.deny_ptrace_attach()
	if ret != 0 {
		return fmt.Errorf("ptrace(PT_DENY_ATTACH) failed with code %d", ret)
	}
	return nil
}
