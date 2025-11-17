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

	// Send prompt
	promptReq := acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Say hello"),
		},
	}

	resp, err := client.Prompt(ctx, promptReq)
	require.NoError(t, err, "Prompt should succeed")

	// Verify response
	assert.NotNil(t, resp.StopReason, "Stop reason should be set")
	// Stop reason should be end_turn for successful completion
	assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason, "Stop reason should be end_turn")
}

// TestACP_Prompt_ContentBlocks tests prompt with different content block types.
func TestACP_Prompt_ContentBlocks(t *testing.T) {
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

	// Test with text block
	t.Run("text block", func(t *testing.T) {
		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("What is 2+2?"),
			},
		}
		resp, err := client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})

	// Test with image block (converted to text description)
	t.Run("image block", func(t *testing.T) {
		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.ImageBlock("base64imagedata", "image/png"),
			},
		}
		resp, err := client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})

	// Test with mixed content blocks
	t.Run("mixed content blocks", func(t *testing.T) {
		req := acp.PromptRequest{
			SessionId: sessionResp.SessionId,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("Analyze this image:"),
				acp.ImageBlock("base64imagedata", "image/jpeg"),
			},
		}
		resp, err := client.Prompt(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp.StopReason)
	})
}

// TestACP_Prompt_InvalidSession tests error handling for invalid session.
func TestACP_Prompt_InvalidSession(t *testing.T) {
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

	// Try to prompt with invalid session ID
	req := acp.PromptRequest{
		SessionId: acp.SessionId("invalid-session-id"),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("test"),
		},
	}

	_, err = client.Prompt(ctx, req)
	assert.Error(t, err, "Prompt should fail with invalid session ID")
}

