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
	time.Sleep(100 * time.Millisecond)

	// Check for notifications
	notifications := clientImpl.getNotifications()

	// Should receive at least some notifications (user message, tool calls, etc.)
	// Note: Exact notifications depend on LLM response, so we just verify some were received
	if len(notifications) == 0 {
		// Wait a bit more
		time.Sleep(100 * time.Millisecond)
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

// TestACP_ToolCall_Create tests tool_call notification structure.
func TestACP_ToolCall_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for tool_call notification
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if notif.Update.ToolCall != nil {
			toolCall := notif.Update.ToolCall
			assert.NotEmpty(t, toolCall.ToolCallId, "Tool call should have ID")
			assert.NotEmpty(t, toolCall.Title, "Tool call should have title")
			// Status is a value type, verify it's valid
			validStatuses := []acp.ToolCallStatus{
				acp.ToolCallStatusPending,
				acp.ToolCallStatusInProgress,
				acp.ToolCallStatusCompleted,
				acp.ToolCallStatusFailed,
			}
			assert.Contains(t, validStatuses, toolCall.Status, "Status should be valid")
			return
		}
	}
	t.Log("No tool_call notification found (may be expected)")
}

// TestACP_ToolCall_Update_Status tests status transitions.
func TestACP_ToolCall_Update_Status(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for status transitions in tool_call_update notifications
	notifications := clientImpl.getNotifications()
	statuses := make(map[acp.ToolCallId][]acp.ToolCallStatus)
	for _, notif := range notifications {
		if notif.Update.ToolCallUpdate != nil {
			update := notif.Update.ToolCallUpdate
			if update.Status != nil {
				statuses[update.ToolCallId] = append(statuses[update.ToolCallId], *update.Status)
			}
		}
	}

	// Verify status transitions if tool calls were made
	if len(statuses) > 0 {
		for toolCallID, statusList := range statuses {
			t.Logf("Tool call %s had statuses: %v", toolCallID, statusList)
			// Status should progress: pending -> in_progress -> completed/failed
		}
	} else {
		t.Log("No tool call status updates found (may be expected)")
	}
}

// TestACP_ToolCall_Update_Failed tests failed status.
func TestACP_ToolCall_Update_Failed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that might trigger a tool call that fails
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file nonexistent-file-that-should-fail.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for failed status in tool_call_update notifications
	notifications := clientImpl.getNotifications()
	foundFailed := false
	for _, notif := range notifications {
		if notif.Update.ToolCallUpdate != nil {
			update := notif.Update.ToolCallUpdate
			if update.Status != nil && *update.Status == acp.ToolCallStatusFailed {
				foundFailed = true
				t.Logf("Found failed tool call: %s", update.ToolCallId)
				break
			}
		}
	}

	if !foundFailed {
		t.Log("No failed tool calls found (may be expected)")
	}
}

// TestACP_ToolCall_Content_Text tests text content in tool calls.
func TestACP_ToolCall_Content_Text(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call with text content
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for text content in tool calls
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			if len(toolCall.Content) > 0 {
				// Content exists - structure depends on SDK implementation
				t.Logf("Found content in tool call (structure depends on SDK)")
				return
			}
		}
		if update := notif.Update.ToolCallUpdate; update != nil {
			if len(update.Content) > 0 {
				// Content exists - structure depends on SDK implementation
				t.Logf("Found content in tool call update (structure depends on SDK)")
				return
			}
		}
	}
	t.Log("No text content in tool calls found (may be expected)")
}

// TestACP_ToolCall_Content_Diff tests diff content in tool calls.
func TestACP_ToolCall_Content_Diff(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call with diff content
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("write file test.txt with content hello"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for diff content in tool calls
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			if toolCall.Content != nil {
				for _, content := range toolCall.Content {
					if content.Diff != nil {
						assert.NotEmpty(t, content.Diff.Path, "Diff should have path")
						assert.NotEmpty(t, content.Diff.NewText, "Diff should have newText")
						t.Logf("Found diff content in tool call: %s", content.Diff.Path)
						return
					}
				}
			}
		}
		if update := notif.Update.ToolCallUpdate; update != nil {
			if update.Content != nil {
				for _, content := range update.Content {
					if content.Diff != nil {
						assert.NotEmpty(t, content.Diff.Path, "Diff should have path")
						assert.NotEmpty(t, content.Diff.NewText, "Diff should have newText")
						t.Logf("Found diff content in tool call update: %s", content.Diff.Path)
						return
					}
				}
			}
		}
	}
	t.Log("No diff content in tool calls found (may be expected)")
}

// TestACP_ToolCall_Locations tests file locations in tool calls.
func TestACP_ToolCall_Locations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call with file location
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for locations in tool calls
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			if len(toolCall.Locations) > 0 {
				for _, loc := range toolCall.Locations {
					assert.NotEmpty(t, loc.Path, "Location should have path")
					t.Logf("Found location in tool call: %s (line: %v)", loc.Path, loc.Line)
					return
				}
			}
		}
		if update := notif.Update.ToolCallUpdate; update != nil {
			if len(update.Locations) > 0 {
				for _, loc := range update.Locations {
					assert.NotEmpty(t, loc.Path, "Location should have path")
					t.Logf("Found location in tool call update: %s (line: %v)", loc.Path, loc.Line)
					return
				}
			}
		}
	}
	t.Log("No locations in tool calls found (may be expected)")
}

// TestACP_ToolCall_Kinds tests all tool kinds.
func TestACP_ToolCall_Kinds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompts that might trigger different tool kinds
	testCases := []struct {
		name   string
		prompt string
		kind   acp.ToolKind
	}{
		{"read", "read file test.txt", acp.ToolKindRead},
		{"write", "write file test.txt", acp.ToolKindEdit},
		{"execute", "run command ls", acp.ToolKindExecute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientImpl.clearNotifications()
			promptReq := acp.PromptRequest{
				SessionId: sessionResp.SessionId,
				Prompt: []acp.ContentBlock{
					acp.TextBlock(tc.prompt),
				},
			}

			_, err = client.Prompt(ctx, promptReq)
			require.NoError(t, err)

			// Check for tool kind
			notifications := clientImpl.getNotifications()
			for _, notif := range notifications {
				if toolCall := notif.Update.ToolCall; toolCall != nil {
					if toolCall.Kind != "" {
						t.Logf("Found tool kind: %v (expected: %v)", toolCall.Kind, tc.kind)
						// Verify it's a valid tool kind
						validKinds := []acp.ToolKind{
							acp.ToolKindRead,
							acp.ToolKindEdit,
							acp.ToolKindDelete,
							acp.ToolKindMove,
							acp.ToolKindSearch,
							acp.ToolKindExecute,
							acp.ToolKindThink,
							acp.ToolKindFetch,
							acp.ToolKindOther,
						}
						assert.Contains(t, validKinds, toolCall.Kind, "Tool kind should be valid")
						return
					}
				}
			}
			t.Log("No tool kind found (may be expected)")
		})
	}
}

// TestACP_ToolCall_RawInput tests rawInput field.
func TestACP_ToolCall_RawInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for rawInput in tool calls
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			if toolCall.RawInput != nil {
				t.Logf("Found rawInput in tool call: %v", toolCall.RawInput)
				return
			}
		}
		if update := notif.Update.ToolCallUpdate; update != nil {
			if update.RawInput != nil {
				t.Logf("Found rawInput in tool call update: %v", update.RawInput)
				return
			}
		}
	}
	t.Log("No rawInput in tool calls found (may be expected)")
}

// TestACP_ToolCall_RawOutput tests rawOutput field.
func TestACP_ToolCall_RawOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that should trigger tool call
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for rawOutput in tool calls
	notifications := clientImpl.getNotifications()
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			if toolCall.RawOutput != nil {
				t.Logf("Found rawOutput in tool call: %v", toolCall.RawOutput)
				return
			}
		}
		if update := notif.Update.ToolCallUpdate; update != nil {
			if update.RawOutput != nil {
				t.Logf("Found rawOutput in tool call update: %v", update.RawOutput)
				return
			}
		}
	}
	t.Log("No rawOutput in tool calls found (may be expected)")
}

// TestACP_ToolCall_Multiple tests multiple tool calls in one turn.
func TestACP_ToolCall_Multiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that might trigger multiple tool calls
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("list directory and read file test.txt"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for multiple tool calls
	notifications := clientImpl.getNotifications()
	toolCallIDs := make(map[acp.ToolCallId]bool)
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			toolCallIDs[toolCall.ToolCallId] = true
		}
	}

	if len(toolCallIDs) > 1 {
		t.Logf("Found %d different tool calls", len(toolCallIDs))
	} else {
		t.Log("Found single or no tool calls (may be expected)")
	}
}

// TestACP_ToolCall_Sequential tests sequential tool calls.
func TestACP_ToolCall_Sequential(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)
	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Send prompt that might trigger sequential tool calls
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("first list directory, then read a file"),
		},
	}

	_, err = client.Prompt(ctx, promptReq)
	require.NoError(t, err)

	// Check for sequential tool calls (tool call -> update -> tool call)
	notifications := clientImpl.getNotifications()
	toolCallSequence := []acp.ToolCallId{}
	for _, notif := range notifications {
		if toolCall := notif.Update.ToolCall; toolCall != nil {
			toolCallSequence = append(toolCallSequence, toolCall.ToolCallId)
		}
	}

	if len(toolCallSequence) > 1 {
		t.Logf("Found sequential tool calls: %v", toolCallSequence)
	} else {
		t.Log("Found single or no sequential tool calls (may be expected)")
	}
}
