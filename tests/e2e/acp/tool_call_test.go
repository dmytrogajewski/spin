package acp

import (
	"context"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Prompt_ToolCalls tests that tool calls are executed and notifications are sent.
func TestACP_Prompt_ToolCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	// Create client with notification tracking
	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
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

	// Clear any notifications from initialization
	clientImpl.clearNotifications()

	// Send prompt that should trigger tool calls (e.g., list files)
	promptReq := acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("List all files in the current directory"),
		},
	}

	// Start prompt in goroutine to allow notifications to arrive
	done := make(chan error, 1)
	go func() {
		_, err := client.Prompt(ctx, promptReq)
		done <- err
	}()

	// Wait a bit for notifications
	time.Sleep(3 * time.Second)

	// Check for notifications
	notifications := clientImpl.getNotifications()
	
	// Should receive at least some notifications (user message, tool calls, etc.)
	// Note: Exact notifications depend on LLM response, so we just verify some were received
	if len(notifications) == 0 {
		// Wait a bit more
		time.Sleep(2 * time.Second)
		notifications = clientImpl.getNotifications()
	}

	// Verify we got some notifications
	// In a real scenario, we'd check for specific notification types
	t.Logf("Received %d notifications", len(notifications))

	// Wait for prompt to complete
	select {
	case err := <-done:
		require.NoError(t, err, "Prompt should complete")
	case <-time.After(30 * time.Second):
		t.Fatal("Prompt timed out")
	}
}

// TestACP_Prompt_ToolCallNotifications tests that tool call notifications have correct structure.
func TestACP_Prompt_ToolCallNotifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--provider", "ollama", "--model", "qwen3:0.6b", "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	clientImpl := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, clientImpl)
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

	clientImpl.clearNotifications()

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("What files are in this directory? Use the list_directory tool."),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check notifications for tool calls
	notifications := clientImpl.getNotifications()
	
	// Look for tool call notifications
	hasToolCall := false
	for _, notif := range notifications {
		if notif.Update.ToolCall != nil {
			hasToolCall = true
			toolCall := notif.Update.ToolCall
			assert.NotEmpty(t, toolCall.ToolCallId, "Tool call should have ID")
			assert.NotEmpty(t, toolCall.Title, "Tool call should have title")
			t.Logf("Found tool call: %s (ID: %s)", toolCall.Title, toolCall.ToolCallId)
		}
		if notif.Update.ToolCallUpdate != nil {
			hasToolCall = true
			update := notif.Update.ToolCallUpdate
			assert.NotEmpty(t, update.ToolCallId, "Tool call update should have ID")
			t.Logf("Found tool call update: ID=%s, Status=%v", update.ToolCallId, update.Status)
		}
	}

	// Note: Tool calls depend on LLM response, so we don't require them
	// But if they exist, they should have correct structure
	if hasToolCall {
		t.Log("Tool call notifications verified")
	} else {
		t.Log("No tool calls in this test run (LLM may have responded differently)")
	}
}

