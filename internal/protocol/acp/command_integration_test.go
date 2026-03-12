package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// newTestACPAgent creates a fully configured ACP agent for testing.
func newTestACPAgent(t *testing.T) (*SpinACPAgent, *mockConnectionForPlan, acp.SessionId) {
	t.Helper()

	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: validator,
	})
	securityService := security.NewService(validator, approvalService)
	detectionService := detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         "/tmp",
	})
	mockProvider := llm.NewMockProvider("test")
	planningService := planning.NewService(mockProvider)

	agentInstance, err := agent.NewAgent(
		mockProvider, securityService, detectionService, toolRuntime,
		planningService, &agent.Environment{WorkDir: "/tmp"}, emitter,
	)
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())), emitter, nil)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: "/tmp"})
	require.NoError(t, err)

	return acpAgent, mockConn, sessionResp.SessionId
}

// hasAgentMessageChunk checks if any notification has an AgentMessageChunk.
func hasAgentMessageChunk(notifications []acp.SessionNotification) bool {
	for _, notif := range notifications {
		if notif.Update.AgentMessageChunk != nil {
			return true
		}
	}

	return false
}

// TestPrompt_CommandExecution tests command execution via Prompt method.
func TestPrompt_CommandExecution(t *testing.T) {
	t.Parallel()

	acpAgent, mockConn, sessionID := newTestACPAgent(t)

	t.Run("execute_mode_command", func(t *testing.T) {
		t.Parallel()

		resp, err := acpAgent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("/mode review")},
		})
		require.NoError(t, err)
		assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)
		assert.True(t, hasAgentMessageChunk(mockConn.GetNotifications()), "should send agent message chunk notification")
	})

	t.Run("execute_help_command", func(t *testing.T) {
		t.Parallel()

		resp, err := acpAgent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("/help")},
		})
		require.NoError(t, err)
		assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)
		assert.True(t, hasAgentMessageChunk(mockConn.GetNotifications()), "should send agent message chunk notification")
	})

	t.Run("execute_exit_command_error", func(t *testing.T) {
		t.Parallel()

		resp, err := acpAgent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("/exit")},
		})
		require.Error(t, err)
		assert.Equal(t, acp.StopReasonRefusal, resp.StopReason)
		assert.Contains(t, err.Error(), "not available via ACP")
	})
}

// TestNewSession_SendsAvailableCommandsUpdate tests that NewSession sends available commands notification.
func TestNewSession_SendsAvailableCommandsUpdate(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	req := acp.NewSessionRequest{
		Cwd: "/tmp",
	}
	_, err = acpAgent.NewSession(context.Background(), req)
	require.NoError(t, err)

	// Verify available commands notification was sent.
	notifications := mockConn.GetNotifications()
	found := false

	for _, notif := range notifications {
		if notif.Update.AvailableCommandsUpdate == nil {
			continue
		}

		found = true
		update := notif.Update.AvailableCommandsUpdate
		assert.NotEmpty(t, update.AvailableCommands, "should have available commands")
		// Check that /mode and /help are included.
		commandNames := make(map[string]bool)
		for _, cmd := range update.AvailableCommands {
			commandNames[cmd.Name] = true
		}

		assert.True(t, commandNames["/mode"], "should include /mode command")
		assert.True(t, commandNames["/help"], "should include /help command")
		// Check that /exit and /quit are NOT included (TUI-only).
		assert.False(t, commandNames["/exit"], "should not include /exit command (TUI-only)")
		assert.False(t, commandNames["/quit"], "should not include /quit command (TUI-only)")

		break
	}

	assert.True(t, found, "should send available commands update notification")
}
