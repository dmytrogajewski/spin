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

// TestPrompt_CommandExecution tests command execution via Prompt method.
func TestPrompt_CommandExecution(t *testing.T) {
	t.Parallel(
	// Create agent with all dependencies.
	)

	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
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
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planningService,
		&agent.Environment{WorkDir: "/tmp"},
		emitter,
	)
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(
		agentInstance,
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		emitter,
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnectionForPlan{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	t.Run("execute_mode_command", func(t *testing.T) {
		t.Parallel()
		promptReq := acp.PromptRequest{
			SessionId: sessionID,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("/mode review"),
			},
		}

		var resp acp.PromptResponse
		resp, err = acpAgent.Prompt(context.Background(), promptReq)
		require.NoError(t, err)
		assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)

		// Verify notification was sent.
		notifications := mockConn.GetNotifications()
		found := false

		for _, notif := range notifications {
			if notif.Update.AgentMessageChunk != nil {
				// Check if it contains mode switch message
				// The exact format depends on implementation.
				found = true

				break
			}
		}

		assert.True(t, found, "should send agent message chunk notification")
	})

	t.Run("execute_help_command", func(t *testing.T) {
		t.Parallel()
		promptReq := acp.PromptRequest{
			SessionId: sessionID,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("/help"),
			},
		}

		var resp acp.PromptResponse
		resp, err = acpAgent.Prompt(context.Background(), promptReq)
		require.NoError(t, err)
		assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)

		// Verify notification was sent.
		notifications := mockConn.GetNotifications()
		found := false

		for _, notif := range notifications {
			if notif.Update.AgentMessageChunk != nil {
				found = true

				break
			}
		}

		assert.True(t, found, "should send agent message chunk notification")
	})

	t.Run("execute_exit_command_error", func(t *testing.T) {
		t.Parallel()
		promptReq := acp.PromptRequest{
			SessionId: sessionID,
			Prompt: []acp.ContentBlock{
				acp.TextBlock("/exit"),
			},
		}

		var resp acp.PromptResponse
		resp, err = acpAgent.Prompt(context.Background(), promptReq)
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
		if notif.Update.AvailableCommandsUpdate != nil {
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
	}

	assert.True(t, found, "should send available commands update notification")
}
