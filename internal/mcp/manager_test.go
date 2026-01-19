package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_CallTool_AcceptsJSONRawMessage verifies signature change from map[string]interface{} to json.RawMessage
func Test_CallTool_AcceptsJSONRawMessage(t *testing.T) {
	// This test verifies the interface{} elimination requirement from Phase 2.4
	t.Run("accepts json.RawMessage arguments", func(t *testing.T) {
		// Create minimal manager for signature test
		mgr := NewMCPServerManager(&Config{
			EnableMCP:  false, // Don't actually start servers
			MCPServers: []MCPServerConfig{},
		}, nil)

		ctx := context.Background()

		// Raw JSON arguments (type-safe)
		args := json.RawMessage(`{"path": "/test"}`)

		// This should compile with new signature
		_, err := mgr.CallTool(ctx, "nonexistent_tool", args)

		// Expected to fail because tool doesn't exist, but signature is correct
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Test_ConnectServer_UsesSDKClient verifies migration to mcp-go SDK
func Test_ConnectServer_UsesSDKClient(t *testing.T) {
	t.Run("creates SDK client from config", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))

		mgr := NewMCPServerManager(&Config{
			EnableMCP:  true,
			MCPServers: []MCPServerConfig{},
		}, logger)

		// Create config
		cfg := MCPServerConfig{
			Name:    "test",
			Command: "/bin/echo",
			Args:    []string{"hello"},
			Env: map[string]string{
				"TEST_VAR": "test_value",
			},
		}

		// Create client using SDK
		sdkClient, err := mgr.createSDKClient(cfg)

		// Verify client was created without error
		require.NoError(t, err, "Expected no error creating SDK client")
		assert.NotNil(t, sdkClient, "Expected SDK client to be created")
	})
}
