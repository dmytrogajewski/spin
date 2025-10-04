package hardening

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestApply tests the hardening application
func TestApply(t *testing.T) {
	err := Apply()
	// Hardening is best-effort, so errors are acceptable
	// We just verify it doesn't panic
	t.Logf("Apply() returned: %v", err)
}

// TestSanitizeEnvironment tests environment variable sanitization
func TestSanitizeEnvironment(t *testing.T) {
	// Set a dangerous environment variable
	dangerousVars := []string{
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
		"DYLD_INSERT_LIBRARIES",
	}

	// Set them
	for _, v := range dangerousVars {
		os.Setenv(v, "/malicious.so")
	}

	// Sanitize
	err := sanitizeEnvironment()
	assert.NoError(t, err)

	// Verify they're removed
	for _, v := range dangerousVars {
		_, exists := os.LookupEnv(v)
		assert.False(t, exists, "Variable %s should be removed", v)
	}
}

// TestDisableCoreDumps tests core dump disabling
func TestDisableCoreDumps(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Core dump control not applicable on Windows")
	}

	err := disableCoreDumps()
	// May fail on some systems due to permissions
	t.Logf("disableCoreDumps() returned: %v", err)
}

// TestDisablePtrace tests ptrace disabling
func TestDisablePtrace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Ptrace not applicable on Windows")
	}

	err := disablePtrace()
	// May fail on some systems
	t.Logf("disablePtrace() returned: %v", err)
}

// TestSanitizeEnvironment_NoVars tests sanitization with no dangerous vars
func TestSanitizeEnvironment_NoVars(t *testing.T) {
	// Clear all dangerous vars first
	dangerous := []string{
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
		"DYLD_INSERT_LIBRARIES",
	}

	for _, v := range dangerous {
		os.Unsetenv(v)
	}

	// Sanitize should succeed with no vars
	err := sanitizeEnvironment()
	assert.NoError(t, err)
}

// TestApply_MultipleCalls tests calling Apply multiple times
func TestApply_MultipleCalls(t *testing.T) {
	// First call
	err1 := Apply()
	t.Logf("First Apply() returned: %v", err1)

	// Second call should also work
	err2 := Apply()
	t.Logf("Second Apply() returned: %v", err2)

	// Both should complete (error or not)
}
