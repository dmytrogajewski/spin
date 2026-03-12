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

// TestNewSpinACPAgent tests the constructor with valid inputs.
func TestNewSpinACPAgentWithStorage(t *testing.T) {
	t.Parallel(
	// Create minimal mocks for required components.
	)

	agentInstance := &agent.Agent{} // Will need proper setup in real tests.
	mcpService := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpService, emitter, storage)

	require.NoError(t, err)
	require.NotNil(t, acpAgent)
	assert.Equal(t, agentInstance, acpAgent.agent)
	assert.Equal(t, mcpService, acpAgent.mcpService)
	assert.Equal(t, emitter, acpAgent.emitter)
}

// TestNewSpinACPAgent_Validation tests constructor validation.
func TestNewSpinACPAgent_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		agent       *agent.Agent
		mcpService  *mcp.Service
		emitter     *events.EventEmitter
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil agent",
			agent:       nil,
			mcpService:  mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "agent cannot be nil",
		},
		{
			name:        "nil mcp service",
			agent:       &agent.Agent{},
			mcpService:  nil,
			emitter:     events.NewEventEmitter(100),
			wantErr:     true,
			errContains: "mcp service cannot be nil",
		},
		{
			name:        "nil emitter",
			agent:       &agent.Agent{},
			mcpService:  mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
			emitter:     nil,
			wantErr:     true,
			errContains: "emitter cannot be nil",
		},
		{
			name:       "all valid",
			agent:      &agent.Agent{},
			mcpService: mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
			emitter:    events.NewEventEmitter(100),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			storage, err := session.NewFileStorage(t.TempDir())
			require.NoError(t, err)
			acpAgent, err := NewSpinACPAgentWithStorage(tt.agent, tt.mcpService, tt.emitter, storage)

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
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpService := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpService, emitter, storage)
	require.NoError(t, err)

	// Verify it implements the interface by assignment.
	var _ acp.Agent = acpAgent
}

// TestSpinACPAgent_MethodStubs tests that all methods exist and return errors (stubs).
func TestSpinACPAgent_MethodStubs(t *testing.T) {
	t.Parallel()

	agentInstance := &agent.Agent{}
	mcpService := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpService, emitter, storage)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Initialize", func(t *testing.T) {
		t.Parallel()

		req := acp.InitializeRequest{
			ProtocolVersion: acp.ProtocolVersionNumber,
		}

		var resp acp.InitializeResponse

		resp, err = acpAgent.Initialize(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, acp.ProtocolVersion(acp.ProtocolVersionNumber), resp.ProtocolVersion)
		assert.NotNil(t, resp.AgentCapabilities)
	})

	t.Run("NewSession", func(t *testing.T) {
		t.Parallel()

		req := acp.NewSessionRequest{
			Cwd: "/tmp/test",
		}

		var resp acp.NewSessionResponse

		resp, err = acpAgent.NewSession(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.SessionId)
	})

	t.Run("Prompt", func(t *testing.T) {
		t.Parallel(
		// Skip this test - Prompt is now implemented and requires proper agent setup
		// See prompt_test.go for proper Prompt tests.
		)

		t.Skip("Prompt is implemented - see prompt_test.go for tests")
	})

	t.Run("LoadSession", func(t *testing.T) {
		t.Parallel(
		// Skip this test - LoadSession is now implemented and requires storage
		// See load_session_test.go for proper LoadSession tests.
		)

		t.Skip("LoadSession is implemented - see load_session_test.go for tests")
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Parallel()
		t.Skip("Cancel is implemented - see cancel_test.go for tests")
	})

	t.Run("SetSessionMode", func(t *testing.T) {
		t.Parallel()
		t.Skip("SetSessionMode is implemented - see session_mode_test.go for tests")
	})

	t.Run("RequestPermission", func(t *testing.T) {
		t.Parallel()
		t.Skip("RequestPermission is implemented - see request_permission_test.go for tests")
	})

	t.Run("Authenticate", func(t *testing.T) {
		t.Parallel()

		req := acp.AuthenticateRequest{
			MethodId: acp.AuthMethodId("test"),
		}
		_, authErr := acpAgent.Authenticate(ctx, req)
		require.Error(t, authErr)
		assert.Contains(t, authErr.Error(), "not implemented")
	})
}
