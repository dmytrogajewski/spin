package compliance

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompliance_Error_InvalidSession verifies error handling for invalid session.
func TestCompliance_Error_InvalidSession(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Test Prompt with invalid session
	promptReq := acp.PromptRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test"),
		},
	}

	_, err := acpAgent.Prompt(ctx, promptReq)
	require.Error(t, err, "Prompt should fail with invalid session")
	assert.Contains(t, err.Error(), "session", "Error should mention session")
}

// TestCompliance_Error_InvalidParams verifies error handling for invalid parameters.
func TestCompliance_Error_InvalidParams(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Test NewSession with empty CWD (invalid parameter)
	req := acp.NewSessionRequest{
		Cwd: "",
	}

	_, err := acpAgent.NewSession(ctx, req)
	require.Error(t, err, "NewSession should fail with empty CWD")
	assert.Contains(t, err.Error(), "directory", "Error should mention directory")
}

// TestCompliance_Error_InvalidMode verifies error handling for invalid mode.
func TestCompliance_Error_InvalidMode(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Create a session first
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(ctx, sessionReq)
	require.NoError(t, err)

	// Test SetSessionMode with invalid mode
	modeReq := acp.SetSessionModeRequest{
		SessionId: sessionResp.SessionId,
		ModeId:    acp.SessionModeId("invalid-mode"),
	}

	_, err = acpAgent.SetSessionMode(ctx, modeReq)
	require.Error(t, err, "SetSessionMode should fail with invalid mode")
	assert.Contains(t, err.Error(), "invalid mode", "Error should mention invalid mode")
}

// TestCompliance_Error_ResponseFormat verifies error response format compliance.
func TestCompliance_Error_ResponseFormat(t *testing.T) {
	acpAgent := createTestACPAgent(t)
	ctx := context.Background()

	// Trigger an error (invalid session)
	promptReq := acp.PromptRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test"),
		},
	}

	_, err := acpAgent.Prompt(ctx, promptReq)
	require.Error(t, err)

	// Verify error is a proper Go error (not JSON-RPC error, as that's handled by SDK)
	// The error should be descriptive
	assert.NotEmpty(t, err.Error(), "Error should have a message")
}

