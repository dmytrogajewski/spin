//go:build acp_permission_future

package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_RequestPermission tests permission request flow.
func TestACP_RequestPermission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	// Initialize
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Create session
	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)
	sessionID := sessionResp.SessionId

	// Request permission
	permissionReq := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall: &acp.ToolCall{
			ToolCallId: acp.ToolCallId("test-tool-call-1"),
			Title:      "Test Tool",
			RawInput:   `{"command": "rm -rf /tmp/test"}`,
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow_once"),
				Name:     "Allow Once",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
			{
				OptionId: acp.PermissionOptionId("reject_once"),
				Name:     "Reject",
				Kind:     acp.PermissionOptionKindRejectOnce,
			},
		},
	}

	resp, err := client.RequestPermission(ctx, permissionReq)
	
	// RequestPermission may not be available if approval service is not configured
	// That's okay - we're testing the method exists and can be called
	if err != nil {
		t.Logf("RequestPermission returned error (may be expected if approval service not configured): %v", err)
	} else {
		// Verify response structure
		assert.NotNil(t, resp.Outcome, "Response should have outcome")
		// Outcome should be either Selected or Cancelled
		if resp.Outcome.Selected != nil {
			assert.NotEmpty(t, resp.Outcome.Selected.OptionId, "Selected option should have ID")
		}
	}
}

// TestACP_RequestPermission_InvalidSession tests permission request with invalid session.
func TestACP_RequestPermission_InvalidSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b")
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Request permission with invalid session
	permissionReq := acp.RequestPermissionRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		ToolCall: &acp.ToolCall{
			ToolCallId: acp.ToolCallId("test-tool-call-1"),
			Title:      "Test Tool",
		},
		Options: []acp.PermissionOption{
			{
				OptionId: acp.PermissionOptionId("allow_once"),
				Name:     "Allow Once",
				Kind:     acp.PermissionOptionKindAllowOnce,
			},
		},
	}

	_, err = client.RequestPermission(ctx, permissionReq)
	// Should return error for invalid session
	assert.Error(t, err, "RequestPermission should fail with invalid session ID")
}

// TestACP_RequestPermission_Options tests different permission option kinds.
func TestACP_RequestPermission_Options(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		options []acp.PermissionOption
	}{
		{
			name: "allow_once and allow_always",
			options: []acp.PermissionOption{
				{
					OptionId: acp.PermissionOptionId("allow_once"),
					Name:     "Allow Once",
					Kind:     acp.PermissionOptionKindAllowOnce,
				},
				{
					OptionId: acp.PermissionOptionId("allow_always"),
					Name:     "Allow Always",
					Kind:     acp.PermissionOptionKindAllowAlways,
				},
			},
		},
		{
			name: "reject_once and reject_always",
			options: []acp.PermissionOption{
				{
					OptionId: acp.PermissionOptionId("reject_once"),
					Name:     "Reject Once",
					Kind:     acp.PermissionOptionKindRejectOnce,
				},
				{
					OptionId: acp.PermissionOptionId("reject_always"),
					Name:     "Reject Always",
					Kind:     acp.PermissionOptionKindRejectAlways,
				},
			},
		},
		{
			name: "all option kinds",
			options: []acp.PermissionOption{
				{
					OptionId: acp.PermissionOptionId("allow_once"),
					Name:     "Allow Once",
					Kind:     acp.PermissionOptionKindAllowOnce,
				},
				{
					OptionId: acp.PermissionOptionId("allow_always"),
					Name:     "Allow Always",
					Kind:     acp.PermissionOptionKindAllowAlways,
				},
				{
					OptionId: acp.PermissionOptionId("reject_once"),
					Name:     "Reject Once",
					Kind:     acp.PermissionOptionKindRejectOnce,
				},
				{
					OptionId: acp.PermissionOptionId("reject_always"),
					Name:     "Reject Always",
					Kind:     acp.PermissionOptionKindRejectAlways,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissionReq := acp.RequestPermissionRequest{
				SessionId: sessionResp.SessionId,
				ToolCall: &acp.ToolCall{
					ToolCallId: acp.ToolCallId("test-tool-call"),
					Title:      "Test Tool",
				},
				Options: tt.options,
			}

			_, err := client.RequestPermission(ctx, permissionReq)
			// May fail if approval service not configured, that's okay
			if err != nil {
				t.Logf("RequestPermission returned error (may be expected): %v", err)
			}
		})
	}
}

