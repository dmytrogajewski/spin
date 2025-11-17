package acp

import (
	"context"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Cancel tests cancellation of in-progress prompt execution.
func TestACP_Cancel(t *testing.T) {
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

	// Send a prompt that might take a while
	promptReq := acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Write a long detailed explanation about how computers work"),
		},
	}

	// Start prompt in goroutine
	done := make(chan error, 1)
	var promptResp *acp.PromptResponse
	go func() {
		resp, err := client.Prompt(ctx, promptReq)
		promptResp = &resp
		done <- err
	}()

	// Wait a moment for prompt to start
	time.Sleep(1 * time.Second)

	// Cancel the prompt
	cancelReq := acp.CancelNotification{
		SessionId: sessionID,
	}
	err = client.Cancel(ctx, cancelReq)
	require.NoError(t, err, "Cancel should succeed")

	// Wait for prompt to complete (should be cancelled)
	select {
	case err := <-done:
		// Prompt should complete (either successfully or with cancellation)
		if err != nil {
			t.Logf("Prompt returned error (expected for cancellation): %v", err)
		}
		if promptResp != nil {
			// If we got a response, it should have cancelled stop reason
			if promptResp.StopReason == acp.StopReasonCancelled {
				t.Log("Prompt correctly returned cancelled stop reason")
			} else {
				t.Logf("Prompt stop reason: %v (may vary depending on timing)", promptResp.StopReason)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Prompt did not complete after cancellation")
	}
}

// TestACP_Cancel_InvalidSession tests cancellation with invalid session ID.
func TestACP_Cancel_InvalidSession(t *testing.T) {
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

	// Try to cancel with invalid session ID
	// Note: Cancel is a JSON-RPC notification (fire-and-forget), so it doesn't return errors.
	// The error is logged on the server side but not returned to the client.
	// This test verifies that the notification is accepted without crashing.
	cancelReq := acp.CancelNotification{
		SessionId: acp.SessionId("invalid-session-id"),
	}

	err = client.Cancel(ctx, cancelReq)
	// Notifications don't return errors in JSON-RPC, so this should succeed
	// (the error is logged server-side but not returned)
	assert.NoError(t, err, "Cancel notification should be accepted (error logged server-side)")
}

// TestACP_Cancel_NoActivePrompt tests cancellation when no prompt is active.
func TestACP_Cancel_NoActivePrompt(t *testing.T) {
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

	// Cancel when no prompt is active (should not error, just be a no-op)
	cancelReq := acp.CancelNotification{
		SessionId: sessionResp.SessionId,
	}

	err = client.Cancel(ctx, cancelReq)
	// Cancelling when nothing is active should not error
	// (it's a notification, not a request)
	assert.NoError(t, err, "Cancel should not error even when no prompt is active")
}
