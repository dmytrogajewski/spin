//go:build linux

package sandbox

import (
	"testing"
)

// TestLandlock_Supported tests Landlock detection on Linux
func TestLandlock_Supported(t *testing.T) {
	supported := landlockSupported()
	t.Logf("Landlock supported: %v", supported)
}

// TestLandlock_Integration documents Landlock usage and limitations
func TestLandlock_Integration(t *testing.T) {
	t.Log("Landlock Integration Notes:")
	t.Log("- The go-landlock library applies restrictions to the CURRENT PROCESS")
	t.Log("- Restrictions are inherited by child processes")
	t.Log("- Once applied, restrictions cannot be removed")
	t.Log("- Testing requires running sandboxed commands in separate processes")
	t.Log("- Real-world usage: call Wrap() immediately before cmd.Run()")
	t.Log("- See sandbox_linux.go Wrap() documentation for details")
}
