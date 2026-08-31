package compact

import (
	"os"
	"strings"
)

// Env and backend identifiers for R11 rewrite / escape hatch.
const (
	EnvName    = "SPIN_COMPACT"
	EnvOff     = "0"
	BackendRTK = "rtk"
	BinaryRTK  = "rtk"
)

// EnvDisabled reports SPIN_COMPACT=0.
func EnvDisabled() bool {
	return os.Getenv(EnvName) == EnvOff
}

// RTKPrefixed reports an argv already wrapped by rtk.
func RTKPrefixed(cmd string) bool {
	return cmd == BinaryRTK || strings.HasPrefix(cmd, BinaryRTK+" ")
}

// RewriteArgv prefixes PATH rtk when backend is rtk and lookPath succeeds.
// lookPath is injected so this package stays hermetic.
func RewriteArgv(cmd, backend string, lookPath func(string) (string, error)) (string, bool) {
	if EnvDisabled() || cmd == "" || backend != BackendRTK || lookPath == nil {
		return cmd, false
	}

	if _, err := lookPath(BinaryRTK); err != nil {
		return cmd, false
	}

	if RTKPrefixed(cmd) {
		return cmd, true
	}

	return BinaryRTK + " " + cmd, true
}

// ShouldApply is true when the Go pipeline should filter this exec result.
func ShouldApply(enabled bool, cmd string) bool {
	if !enabled || EnvDisabled() || RTKPrefixed(cmd) {
		return false
	}

	return true
}
