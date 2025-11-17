package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSpinACPAgent tests the constructor with valid inputs.
func TestNewSpinACPAgentWithStorage(t *testing.T) {
	// Create minimal mocks for required components
	agentInstance := &agent.Agent{} // Will need proper setup in real tests
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)

	require.NoError(t, err)
	require.NotNil(t, acpAgent)
	assert.Equal(t, agentInstance, acpAgent.agent)
	assert.Equal(t, mcpManager, acpAgent.mcpManager)
	assert.Equal(t, emitter, acpAgent.emitter)
}

// TestNewSpinACPAgent_Validation tests constructor validation.
func TestNewSpinACPAgent_Validation(t *testing.T) {
	tests := []struct {
		name        string
		agent       *agent.Agent
		mcpManager  *mcp.MCPManager
		emitter     *events.EventEmitter
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil agent",
			agent:       nil,
			mcpManager:  mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default()),
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "agent cannot be nil",
		},
		{
			name:        "nil mcp manager",
			agent:       &agent.Agent{},
			mcpManager:  nil,
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "mcp manager cannot be nil",
		},
		{
			name:        "nil emitter",
			agent:       &agent.Agent{},
			mcpManager:  mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default()),
			emitter:     nil,
			wantErr:     true,
			errContains: "emitter cannot be nil",
		},
		{
			name:       "all valid",
			agent:      &agent.Agent{},
			mcpManager: mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default()),
			emitter:    events.NewEventEmitter(100),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := session.NewFileStorage(t.TempDir())
			require.NoError(t, err)
			acpAgent, err := NewSpinACPAgentWithStorage(tt.agent, tt.mcpManager, tt.emitter, storage)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, acpAgent)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, acpAgent)
			}
		})
	}
}

// TestSpinACPAgent_ImplementsInterface verifies that SpinACPAgent implements acp.Agent.
func TestSpinACPAgent_ImplementsInterface(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Verify it implements the interface by assignment
	var _ acp.Agent = acpAgent
}

// TestSpinACPAgent_MethodStubs tests that all methods exist and return errors (stubs).
func TestSpinACPAgent_MethodStubs(t *testing.T) {
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewMCPManager(&mcp.Config{EnableMCP: false}, slog.Default())
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Initialize", func(t *testing.T) {
		req := acp.InitializeRequest{
			ProtocolVersion: acp.ProtocolVersionNumber,
		}
		resp, err := acpAgent.Initialize(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), resp.ProtocolVersion)
		assert.NotNil(t, resp.AgentCapabilities)
	})

	t.Run("NewSession", func(t *testing.T) {
		req := acp.NewSessionRequest{
			Cwd: "/tmp/test",
		}
		resp, err := acpAgent.NewSession(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.SessionId)
	})

	t.Run("Prompt", func(t *testing.T) {
		// Skip this test - Prompt is now implemented and requires proper agent setup
		// See prompt_test.go for proper Prompt tests
		t.Skip("Prompt is implemented - see prompt_test.go for tests")
	})

	t.Run("LoadSession", func(t *testing.T) {
		// Skip this test - LoadSession is now implemented and requires storage
		// See load_session_test.go for proper LoadSession tests
		t.Skip("LoadSession is implemented - see load_session_test.go for tests")
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Skip("Cancel is implemented - see cancel_test.go for tests")
	})

	t.Run("SetSessionMode", func(t *testing.T) {
		t.Skip("SetSessionMode is implemented - see session_mode_test.go for tests")
	})

	t.Run("RequestPermission", func(t *testing.T) {
		t.Skip("RequestPermission is implemented - see request_permission_test.go for tests")
	})

	t.Run("Authenticate", func(t *testing.T) {
		req := acp.AuthenticateRequest{
			MethodId: acp.AuthMethodId("test"),
		}
		_, err := acpAgent.Authenticate(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
