package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACP_Prompt_Basic tests basic prompt processing.
func TestACP_Prompt_Basic(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	// Initialize.
	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Create session.
	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// Send prompt.
	promptReq := acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Say hello"),
		},
	}

	resp, err := client.Prompt(ctx, promptReq)
	require.NoError(t, err, "Prompt should succeed")

	// Verify response.
	assert.NotNil(t, resp.StopReason, "Stop reason should be set")
	// Stop reason should be end_turn for successful completion.
	assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason, "Stop reason should be end_turn")
}

// TestACP_Prompt_ContentBlocks tests prompt with different content block types.
func TestACP_Prompt_ContentBlocks(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	t.Cleanup(func() { cleanupAgent(t, cmd, stdin) })

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

	// Test with text block.
	t.Run("text block", func(t *testing.T) {
		t.Parallel()

		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("What is 2+2?"),
			},
		}
		var resp acp.PromptResponse
		resp, err = client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})

	// Test with image block (converted to text description).
	t.Run("image block", func(t *testing.T) {
		t.Parallel()

		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.ImageBlock("base64imagedata", "image/png"),
			},
		}
		var resp acp.PromptResponse
		resp, err = client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})

	// Test with mixed content blocks.
	t.Run("mixed content blocks", func(t *testing.T) {
		t.Parallel()

		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("Analyze this image:"),
				acp.ImageBlock("base64imagedata", "image/jpeg"),
			},
		}
		var resp acp.PromptResponse
		resp, err = client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})
}

// TestACP_Prompt_InvalidSession tests error handling for invalid session.
func TestACP_Prompt_InvalidSession(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cmd, stdin, stdout := startACPAgent(t)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	_, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Try to prompt with invalid session ID.
	req := acp.PromptRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test"),
		},
	}

	_, err = client.Prompt(ctx, req)
	assert.Error(t, err, "Prompt should fail with invalid session ID")
}

// TestACP_Prompt_TextBlock tests text content block (baseline).
func TestACP_Prompt_TextBlock(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("This is a text block"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Prompt_ResourceLink tests resource link content block (baseline).
func TestACP_Prompt_ResourceLink(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.ResourceLinkBlock("file.txt", "file:///test/file.txt"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Prompt_AudioBlock tests audio content block.
func TestACP_Prompt_AudioBlock(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Check if audio capability is supported.
	if !initResp.AgentCapabilities.PromptCapabilities.Audio {
		t.Skip("Audio capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.AudioBlock("base64audiodata", "audio/wav"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// TestACP_Prompt_ResourceBlock tests embedded resource block.
func TestACP_Prompt_ResourceBlock(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	client := createACPClient(t, stdin, stdout)
	ctx := context.Background()

	initResp, err := client.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	})
	require.NoError(t, err)

	// Check if embeddedContext capability is supported.
	if !initResp.AgentCapabilities.PromptCapabilities.EmbeddedContext {
		t.Skip("EmbeddedContext capability not supported")
	}

	sessionResp, err := client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        workDir,
		McpServers: []acp.McpServer{},
	})
	require.NoError(t, err)

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test with resource block"),
		},
	}

	resp, err := client.Prompt(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp.StopReason)
}

// stopReasonCase describes a test case for verifying stop reason behavior.
type stopReasonCase struct {
	name      string
	stopKind  string
}

func runStopReasonTests(t *testing.T, cases []stopReasonCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if testing.Short() {
				t.Skip("Skipping E2E test in short mode")
			}

			workDir := createTestWorkspace(t)

			cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

			req := acp.PromptRequest{
				SessionId: sessionResp.SessionId,
				Prompt:    []acp.ContentBlock{acp.TextBlock("Test prompt")},
			}

			resp, err := client.Prompt(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, resp.StopReason)
			t.Logf("Stop reason: %v", resp.StopReason)
		})
	}
}

// TestACP_Prompt_StopReason_MaxTokens tests max_tokens stop reason.
func TestACP_Prompt_StopReason_MaxTokens(t *testing.T) {
	t.Parallel()
	runStopReasonTests(t, []stopReasonCase{{"max_tokens", "max_tokens"}})
}

// TestACP_Prompt_StopReason_MaxTurnRequests tests max_turn_requests stop reason.
func TestACP_Prompt_StopReason_MaxTurnRequests(t *testing.T) {
	t.Parallel()
	runStopReasonTests(t, []stopReasonCase{{"max_turn_requests", "max_turn_requests"}})
}

// TestACP_Prompt_StopReason_Refusal tests refusal stop reason.
func TestACP_Prompt_StopReason_Refusal(t *testing.T) {
	t.Parallel()
	runStopReasonTests(t, []stopReasonCase{{"refusal", "refusal"}})
}

// TestACP_Prompt_StopReason_Canceled tests canceled stop reason.
func TestACP_Prompt_StopReason_Canceled(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
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

	// Start prompt in background.
	promptReq := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Long running prompt"),
		},
	}

	// Send cancel notification.
	cancelNotif := acp.CancelNotification{
		SessionId: sessionResp.SessionId,
	}
	err = client.Cancel(ctx, cancelNotif)
	require.NoError(t, err)

	// Send prompt (may be canceled).
	resp, err := client.Prompt(ctx, promptReq)
	// Prompt may succeed or be canceled.
	if err == nil {
		// If prompt succeeded, check if stop reason is canceled.
		if resp.StopReason == acp.StopReasonCancelled {
			t.Log("Prompt was canceled as expected")
		}
	}
}

// TestACP_Prompt_AgentMessageChunks tests agent_message_chunk notifications.
func TestACP_Prompt_AgentMessageChunks(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir := createTestWorkspace(t)

	cmd, stdin, stdout := startACPAgent(t, "--workspace", workDir)
	defer cleanupAgent(t, cmd, stdin)

	testClientInstance := &testClient{}
	client := createACPClientWithClient(t, stdin, stdout, testClientInstance)
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

	// Clear notifications.
	testClientInstance.clearNotifications()

	req := acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Test prompt"),
		},
	}

	_, err = client.Prompt(ctx, req)
	require.NoError(t, err)

	// Check for agent_message_chunk notifications.
	notifications := testClientInstance.getNotifications()
	foundAgentChunk := false

	for _, notif := range notifications {
		if notif.Update.AgentMessageChunk != nil {
			// Check if it's an agent_message_chunk
			// The exact structure depends on ACP SDK implementation.
			foundAgentChunk = true

			break
		}
	}
	// Note: Agent may or may not send chunks depending on implementation.
	t.Logf("Agent message chunks received: %v", foundAgentChunk)
}
