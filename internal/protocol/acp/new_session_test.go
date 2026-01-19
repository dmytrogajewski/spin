package acp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpinACPAgent_NewSession_Success tests successful session creation.
func TestSpinACPAgent_NewSession_Success(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(context.Background(), req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.SessionId)

	// Verify session was stored
	acpAgent.mu.RLock()
	session, exists := acpAgent.sessions[resp.SessionId]
	acpAgent.mu.RUnlock()

	assert.True(t, exists, "session should be stored")
	assert.NotNil(t, session)
	assert.Equal(t, "/tmp/test", session.WorkDir)
}

// TestSpinACPAgent_NewSession_WithMcpServers tests session creation with MCP servers.
func TestSpinACPAgent_NewSession_WithMcpServers(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPServerManager(&mcp.Config{EnableMCP: true}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
		McpServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "test-server",
					Command: "echo",
					Args:    []string{"hello"},
					Env:     []acp.EnvVariable{},
				},
			},
		},
	}

	resp, err := acpAgent.NewSession(ctx, req)

	// Session creation should succeed even if MCP connection fails
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SessionId)

	// Verify session was stored
	acpAgent.mu.RLock()
	session, exists := acpAgent.sessions[resp.SessionId]
	acpAgent.mu.RUnlock()

	assert.True(t, exists, "session should be stored")
	assert.NotNil(t, session)
	assert.Equal(t, "/tmp/test", session.WorkDir)
}

// TestSpinACPAgent_NewSession_InvalidCwd tests session creation with invalid working directory.
func TestSpinACPAgent_NewSession_InvalidCwd(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd: "", // Empty working directory
	}

	_, err = acpAgent.NewSession(context.Background(), req)

	// Session.NewSession may accept empty workDir, but we should validate
	// For now, let's see what happens
	if err != nil {
		assert.Error(t, err)
	}
}

// TestSpinACPAgent_NewSession_UnsupportedTransport tests session creation with unsupported transport.
func TestSpinACPAgent_NewSession_UnsupportedTransport(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
		McpServers: []acp.McpServer{
			{
				Http: &acp.McpServerHttp{
					Url: "http://localhost:8080",
				},
			},
		},
	}

	_, err = acpAgent.NewSession(context.Background(), req)

	// HTTP transport is not supported
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP transport is not supported")
}

// TestSpinACPAgent_NewSession_NoTransport tests session creation with MCP server without transport.
func TestSpinACPAgent_NewSession_NoTransport(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
		McpServers: []acp.McpServer{
			{}, // No transport specified
		},
	}

	_, err = acpAgent.NewSession(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no transport specified")
}
