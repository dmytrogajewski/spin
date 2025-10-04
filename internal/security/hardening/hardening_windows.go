//go:build windows

package hardening

// disableCoreDumps is not applicable on Windows.
func disableCoreDumps() error {
	return nil
}

// disablePtrace is not applicable on Windows.
func disablePtrace() error {
	return nil
}
