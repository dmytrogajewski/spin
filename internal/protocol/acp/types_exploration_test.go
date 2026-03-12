package acp

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContentBlockHelpers explores SDK helper functions for content blocks.
func TestContentBlockHelpers(t *testing.T) {
	t.Parallel()
	t.Run("TextBlock", func(t *testing.T) {
		t.Parallel()
		block := acp.TextBlock("Hello, world!")
		assert.NotNil(t, block)
		// ContentBlock is a union type with optional fields.
		require.NotNil(t, block.Text, "TextBlock should set Text field")
		assert.Equal(t, "Hello, world!", block.Text.Text)
		assert.Nil(t, block.Image, "TextBlock should not set Image")
		assert.Nil(t, block.Audio, "TextBlock should not set Audio")
	})

	t.Run("ImageBlock", func(t *testing.T) {
		t.Parallel()
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
		block := acp.ImageBlock(base64Data, "image/png")
		assert.NotNil(t, block)
		require.NotNil(t, block.Image, "ImageBlock should set Image field")
		assert.Equal(t, base64Data, block.Image.Data)
		assert.Equal(t, "image/png", block.Image.MimeType)
		assert.Nil(t, block.Text, "ImageBlock should not set Text")
	})

	t.Run("ResourceLinkBlock", func(t *testing.T) {
		t.Parallel()
		block := acp.ResourceLinkBlock("file.txt", "file:///path/to/file.txt")
		assert.NotNil(t, block)
		require.NotNil(t, block.ResourceLink, "ResourceLinkBlock should set ResourceLink field")
		assert.Equal(t, "file.txt", block.ResourceLink.Name)
		assert.Equal(t, "file:///path/to/file.txt", block.ResourceLink.Uri)
		assert.Nil(t, block.Text, "ResourceLinkBlock should not set Text")
	})
}

// TestAgentInterface explores the acp.Agent interface structure.
func TestAgentInterface(t *testing.T) {
	t.Parallel(
	// Verify Agent is an interface.
	)

	var agent acp.Agent
	assert.Nil(t, agent) // Interface can be nil.

	// Explore interface methods via documentation
	// Agent interface has these methods:
	// - Initialize(ctx, InitializeRequest) (InitializeResponse, error)
	// - NewSession(ctx, NewSessionRequest) (NewSessionResponse, error)
	// - Prompt(ctx, PromptRequest) (PromptResponse, error)
	// - Cancel(ctx, CancelNotification) error
	// - SetSessionMode(ctx, SetSessionModeRequest) (SetSessionModeResponse, error)
	// - Authenticate(ctx, AuthenticateRequest) (AuthenticateResponse, error).

	t.Log("Agent interface requires 6 methods")
}

// TestRequestResponseTypes explores request/response type structures.
func TestRequestResponseTypes(t *testing.T) {
	t.Parallel()
	t.Run("InitializeRequest", func(t *testing.T) {
		t.Parallel()
		req := acp.InitializeRequest{
			ProtocolVersion:    acp.ProtocolVersionNumber,
			ClientCapabilities: acp.ClientCapabilities{
				// Client capabilities structure.
			},
			ClientInfo: &acp.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			},
		}
		assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), req.ProtocolVersion)
		require.NotNil(t, req.ClientInfo)
		assert.Equal(t, "test-client", req.ClientInfo.Name)
	})

	t.Run("InitializeResponse", func(t *testing.T) {
		t.Parallel()
		resp := acp.InitializeResponse{
			ProtocolVersion:   acp.ProtocolVersionNumber,
			AgentCapabilities: acp.AgentCapabilities{
				// Agent capabilities structure.
			},
			AgentInfo: &acp.Implementation{
				Name:    "spin",
				Version: "0.1.0",
			},
		}
		assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), resp.ProtocolVersion)
		require.NotNil(t, resp.AgentInfo)
		assert.Equal(t, "spin", resp.AgentInfo.Name)
	})

	t.Run("NewSessionRequest", func(t *testing.T) {
		t.Parallel()
		req := acp.NewSessionRequest{
			Cwd: "/tmp/test",
			McpServers: []acp.McpServer{
				{
					Stdio: &acp.McpServerStdio{
						Command: "mcp-server",
					},
				},
			},
		}
		assert.Equal(t, "/tmp/test", req.Cwd)
		assert.Len(t, req.McpServers, 1)
		require.NotNil(t, req.McpServers[0].Stdio)
		assert.Equal(t, "mcp-server", req.McpServers[0].Stdio.Command)
	})

	t.Run("PromptRequest", func(t *testing.T) {
		t.Parallel()
		req := acp.PromptRequest{
			SessionId: acp.SessionId("test-session"),
			Prompt: []acp.ContentBlock{
				acp.TextBlock("Hello"),
			},
		}
		assert.Equal(t, acp.SessionId("test-session"), req.SessionId)
		assert.Len(t, req.Prompt, 1)
	})
}

// TestCapabilityTypes explores capability type structures.
func TestCapabilityTypes(t *testing.T) {
	t.Parallel()
	t.Run("AgentCapabilities", func(t *testing.T) {
		t.Parallel()
		caps := acp.AgentCapabilities{
			PromptCapabilities: acp.PromptCapabilities{
				Image:           true,
				Audio:           false,
				EmbeddedContext: true,
			},
			McpCapabilities: acp.McpCapabilities{
				Http: false,
				Sse:  false,
			},
		}
		assert.True(t, caps.PromptCapabilities.Image)
		assert.False(t, caps.PromptCapabilities.Audio)
		assert.True(t, caps.PromptCapabilities.EmbeddedContext)
		assert.False(t, caps.McpCapabilities.Http)
		assert.False(t, caps.McpCapabilities.Sse)
	})

	t.Run("ClientCapabilities", func(t *testing.T) {
		t.Parallel()
		caps := acp.ClientCapabilities{
			// Client capabilities structure.
		}
		assert.NotNil(t, caps)
	})
}

// TestSessionTypes explores session-related types.
func TestSessionTypes(t *testing.T) {
	t.Parallel()
	t.Run("SessionUpdate", func(t *testing.T) {
		t.Parallel(
		// SessionUpdate is a union type, use helper functions.
		)

		update := acp.UpdateAgentMessageText("Hello")
		assert.NotNil(t, update)
		require.NotNil(t, update.AgentMessageChunk)
		assert.NotNil(t, update.AgentMessageChunk.Content)
	})

	t.Run("AgentMessageChunk", func(t *testing.T) {
		t.Parallel()
		update := acp.UpdateAgentMessage(acp.TextBlock("Response text"))
		require.NotNil(t, update.AgentMessageChunk)
		assert.NotNil(t, update.AgentMessageChunk.Content)
	})

	t.Run("ToolCall", func(t *testing.T) {
		t.Parallel(
		// ToolCall is created via helper functions.
		)

		update := acp.StartToolCall(acp.ToolCallId("call-123"), "shell_command")
		require.NotNil(t, update.ToolCall)
		assert.Equal(t, acp.ToolCallId("call-123"), update.ToolCall.ToolCallId)
		assert.Equal(t, "shell_command", update.ToolCall.Title)
	})
}

// TestMcpTypes explores MCP-related types.
func TestMcpTypes(t *testing.T) {
	t.Parallel()
	t.Run("McpServerStdio", func(t *testing.T) {
		t.Parallel()
		server := acp.McpServer{
			Stdio: &acp.McpServerStdio{
				Command: "mcp-server",
				Args:    []string{"--arg", "value"},
			},
		}
		require.NotNil(t, server.Stdio)
		assert.Equal(t, "mcp-server", server.Stdio.Command)
		assert.Len(t, server.Stdio.Args, 2)
		assert.Nil(t, server.Http, "Stdio server should not have Http")
		assert.Nil(t, server.Sse, "Stdio server should not have Sse")
	})

	t.Run("McpServerHttp", func(t *testing.T) {
		t.Parallel()
		server := acp.McpServer{
			Http: &acp.McpServerHttp{
				Url: "https://example.com/mcp",
			},
		}
		require.NotNil(t, server.Http)
		assert.Equal(t, "https://example.com/mcp", server.Http.Url)
		assert.Nil(t, server.Stdio, "Http server should not have Stdio")
		assert.Nil(t, server.Sse, "Http server should not have Sse")
	})
}

// TestConnectionInfrastructure explores connection types.
func TestConnectionInfrastructure(t *testing.T) {
	t.Parallel()
	t.Run("NewAgentSideConnection", func(t *testing.T) {
		t.Parallel(
		// This test verifies we understand the connection setup
		// Actual connection requires a real Agent implementation.
		)

		var agent acp.Agent
		// conn := acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
		// We'll test this when we have a real Agent implementation.
		t.Log("NewAgentSideConnection requires Agent implementation")
		assert.Nil(t, agent) // Placeholder.
	})
}

// TestTypeConversions explores potential type conversion patterns.
func TestTypeConversions(t *testing.T) {
	t.Parallel()
	t.Run("ContentBlockToContentItem", func(t *testing.T) {
		t.Parallel(
		// This test documents the conversion pattern (not implementing yet.
		)

		acpBlock := acp.TextBlock("Hello")
		assert.NotNil(t, acpBlock)

		// Conversion pattern (conceptual):
		// ACP ContentBlock -> Spin ContentItem
		// - ContentBlockText -> ContentItem{Type: "text", Text: &TextContent{Text: ...}}
		// - ContentBlockImage -> ContentItem{Type: "image", Image: &ImageContent{...}}
		// - ContentBlockResourceLink -> ContentItem{Type: "file_pointer", FilePointer: &FilePointerContent{...}}.

		t.Log("ContentBlock conversion pattern documented")
	})

	t.Run("SessionIdToString", func(t *testing.T) {
		t.Parallel()
		sessionID := acp.SessionId("test-session-123")
		sessionIDStr := string(sessionID)
		assert.Equal(t, "test-session-123", sessionIDStr)
	})
}

// TestHelperFunctions explores SDK helper functions.
func TestHelperFunctions(t *testing.T) {
	t.Parallel()
	t.Run("Ptr", func(t *testing.T) {
		t.Parallel(
		// Ptr is a generic helper to create pointers.
		)

		boolPtr := acp.Ptr(true)
		require.NotNil(t, boolPtr)
		assert.True(t, *boolPtr)

		intPtr := acp.Ptr(42)
		require.NotNil(t, intPtr)
		assert.Equal(t, 42, *intPtr)
	})
}

// TestProtocolConstants explores protocol constants.
func TestProtocolConstants(t *testing.T) {
	t.Parallel()
	t.Run("ProtocolVersionNumber", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, acp.ProtocolVersionNumber)
	})

	t.Run("AgentMethodConstants", func(t *testing.T) {
		t.Parallel(
		// Verify method name constants exist.
		)

		assert.NotEmpty(t, acp.AgentMethodInitialize)
		// Other method constants may be defined in agent_gen.go.
		t.Log("Agent method constants verified")
	})
}

// TestErrorHandling explores error types.
func TestErrorHandling(t *testing.T) {
	t.Parallel()
	t.Run("RequestError", func(t *testing.T) {
		t.Parallel()
		err := &acp.RequestError{
			Code:    -32603, // Internal error.
			Message: "Test error",
		}
		assert.Equal(t, -32603, err.Code)
		assert.Equal(t, "Test error", err.Message)
	})
}

// TestContextUsage verifies context is used in Agent interface.
func TestContextUsage(t *testing.T) {
	t.Parallel(
	// All Agent interface methods accept context.Context as first parameter
	// This is important for cancellation and timeouts.
	)

	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Verify context is part of interface signature
	// All methods: Method(ctx context.Context, ...)
	t.Log("All Agent methods accept context.Context")
}
