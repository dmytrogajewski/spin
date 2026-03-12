package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
)

// TestSpinACPAgent_Initialize_Success tests successful initialization.
func TestSpinACPAgent_Initialize_Success(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
		ClientInfo: &acp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	resp, err := acpAgent.Initialize(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), resp.ProtocolVersion)
	assert.NotNil(t, resp.AgentCapabilities)
	assert.NotNil(t, resp.AgentInfo)
	assert.Equal(t, "spin", resp.AgentInfo.Name)
}

// TestSpinACPAgent_Initialize_ProtocolVersion tests protocol version negotiation.
func TestSpinACPAgent_Initialize_ProtocolVersion(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	tests := []struct {
		name            string
		clientVersion   acp.ProtocolVersion
		expectedVersion acp.ProtocolVersion
	}{
		{
			name:            "version 1 supported",
			clientVersion:   acp.ProtocolVersion(1),
			expectedVersion: acp.ProtocolVersion(1),
		},
		{
			name:            "unsupported version returns version 1",
			clientVersion:   acp.ProtocolVersion(2),
			expectedVersion: acp.ProtocolVersion(1), // Return latest supported.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := acp.InitializeRequest{
				ProtocolVersion: tt.clientVersion,
			}

			var resp acp.InitializeResponse
			resp, err = acpAgent.Initialize(context.Background(), req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedVersion, resp.ProtocolVersion)
		})
	}
}

// TestSpinACPAgent_Initialize_AgentCapabilities tests agent capability advertisement.
func TestSpinACPAgent_Initialize_AgentCapabilities(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(context.Background(), req)
	require.NoError(t, err)

	// Verify capabilities are set.
	caps := resp.AgentCapabilities
	assert.True(t, caps.PromptCapabilities.Image, "should support images")
	assert.True(t, caps.PromptCapabilities.Audio, "should support audio (converted to text description)")
	assert.True(t, caps.PromptCapabilities.EmbeddedContext, "should support embedded context")
	// MCP capabilities depend on manager support.
	assert.NotNil(t, caps.McpCapabilities)
}

// TestSpinACPAgent_Initialize_ClientCapabilitiesStorage tests that client capabilities are stored.
func TestSpinACPAgent_Initialize_ClientCapabilitiesStorage(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	clientCaps := acp.ClientCapabilities{
		// Client capabilities.
	}

	req := acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: clientCaps,
	}

	_, err = acpAgent.Initialize(context.Background(), req)
	require.NoError(t, err)

	// Verify client capabilities are stored.
	assert.NotNil(t, acpAgent.clientCaps)
	assert.Equal(t, clientCaps, *acpAgent.clientCaps)
}

// TestSpinACPAgent_Initialize_AgentInfo tests agent info exchange.
func TestSpinACPAgent_Initialize_AgentInfo(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(context.Background(), req)
	require.NoError(t, err)

	// Verify agent info.
	require.NotNil(t, resp.AgentInfo)
	assert.Equal(t, "spin", resp.AgentInfo.Name)
	assert.NotEmpty(t, resp.AgentInfo.Version)
}

// TestSpinACPAgent_Initialize_AuthMethods tests authentication methods advertisement.
func TestSpinACPAgent_Initialize_AuthMethods(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
	}

	resp, err := acpAgent.Initialize(context.Background(), req)
	require.NoError(t, err)

	// Initially, no auth methods (empty list).
	assert.NotNil(t, resp.AuthMethods)
	assert.Empty(t, resp.AuthMethods, "should have no auth methods initially")
}
