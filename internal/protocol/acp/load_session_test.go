package acp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
)

// TestSpinACPAgent_LoadSession_NoStorage tests LoadSession when storage is not available.
func TestSpinACPAgent_LoadSession_NoStorage(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	// This test specifically tests the case where storage is nil.
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, nil)
	require.NoError(t, err)

	req := acp.LoadSessionRequest{
		SessionId: acp.SessionId("test-session"),
		Cwd:       "/tmp/test",
	}

	_, err = acpAgent.LoadSession(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "session persistence not available")
}

// TestSpinACPAgent_LoadSession_NotFound tests LoadSession when session doesn't exist in storage.
func TestSpinACPAgent_LoadSession_NotFound(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.LoadSessionRequest{
		SessionId: acp.SessionId("non-existent-session"),
		Cwd:       "/tmp/test",
	}

	_, err = acpAgent.LoadSession(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestSpinACPAgent_LoadSession_Success tests successful session loading.
func TestSpinACPAgent_LoadSession_Success(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create and save a session.
	sess := session.NewSession("/tmp/test")
	_ = sess.SetTitle("Test Session")
	err = storage.Save(context.Background(), sess.ID, *sess)
	require.NoError(t, err)

	req := acp.LoadSessionRequest{
		SessionId: acp.SessionId(sess.ID),
		Cwd:       "/tmp/test",
	}

	resp, err := acpAgent.LoadSession(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify session was stored.
	acpAgent.mu.RLock()
	loadedSession, exists := acpAgent.sessions[req.SessionId]
	acpAgent.mu.RUnlock()

	assert.True(t, exists, "session should be stored")
	assert.NotNil(t, loadedSession)
	assert.Equal(t, sess.ID, loadedSession.ID)
	assert.Equal(t, sess.WorkDir, loadedSession.WorkDir)
}

// TestSpinACPAgent_LoadSession_WithMcpServers tests session loading with MCP servers.
func TestSpinACPAgent_LoadSession_WithMcpServers(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create and save a session.
	sess := session.NewSession("/tmp/test")
	err = storage.Save(context.Background(), sess.ID, *sess)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req := acp.LoadSessionRequest{
		SessionId: acp.SessionId(sess.ID),
		Cwd:       "/tmp/test",
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

	resp, err := acpAgent.LoadSession(ctx, req)

	// Session loading should succeed even if MCP connection fails.
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify session was stored.
	acpAgent.mu.RLock()
	loadedSession, exists := acpAgent.sessions[req.SessionId]
	acpAgent.mu.RUnlock()

	assert.True(t, exists, "session should be stored")
	assert.NotNil(t, loadedSession)
}

// TestSpinACPAgent_LoadSession_InvalidMcpServer tests LoadSession with invalid MCP server config.
func TestSpinACPAgent_LoadSession_InvalidMcpServer(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create and save a session.
	sess := session.NewSession("/tmp/test")
	err = storage.Save(context.Background(), sess.ID, *sess)
	require.NoError(t, err)

	req := acp.LoadSessionRequest{
		SessionId: acp.SessionId(sess.ID),
		Cwd:       "/tmp/test",
		McpServers: []acp.McpServer{
			{
				Http: &acp.McpServerHttpInline{
					Url: "http://example.com",
				},
			},
		},
	}

	_, err = acpAgent.LoadSession(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP transport is not supported")
}
