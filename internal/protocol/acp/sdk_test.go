package acp

import (
	"testing"

	"github.com/coder/acp-go-sdk"
)

// TestSDKImport verifies that acp-go-sdk can be imported successfully.
func TestSDKImport(t *testing.T) {
	// This test verifies the SDK dependency is available
	// We're not using any SDK types yet, just verifying import works.
	var _ acp.Agent // Agent is an interface, not a pointer.

	t.Log("acp-go-sdk imported successfully")
}
