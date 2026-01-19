package acp

import (
	"context"
	"log/slog"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestACPCommandContext tests the ACP command context implementation.
func TestACPCommandContext(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default()),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	sessionID := acp.SessionId("test-session")
	cmdCtx := &acpCommandContext{
		agent:     acpAgent,
		sessionID: sessionID,
	}

	t.Run("GetCurrentMode_default", func(t *testing.T) {
		mode := cmdCtx.GetCurrentMode()
		assert.Equal(t, "regular", mode, "should return default mode when session not found")
	})

	t.Run("GetWorkDir_no_session", func(t *testing.T) {
		workDir := cmdCtx.GetWorkDir()
		assert.Equal(t, "", workDir, "should return empty string when session not found")
	})
}

// TestExecuteCommand tests command execution in ACP context.
func TestExecuteCommand(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewMCPServerManager(&mcp.Config{EnableMCP: false}, slog.Default()),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	// Create a session first
	sessionID := acp.SessionId("test-session")
	req := acp.NewSessionRequest{
		Cwd: "/tmp",
	}
	_, err = acpAgent.NewSession(context.Background(), req)
	require.NoError(t, err)

	t.Run("execute_mode_command_show_current", func(t *testing.T) {
		result, err := acpAgent.executeCommand(context.Background(), "/mode", []string{}, sessionID)
		require.NoError(t, err)
		assert.Contains(t, result, "Current mode")
	})

	t.Run("execute_help_command", func(t *testing.T) {
		result, err := acpAgent.executeCommand(context.Background(), "/help", []string{}, sessionID)
		require.NoError(t, err)
		assert.Contains(t, result, "Available commands")
		assert.Contains(t, result, "/mode")
		assert.Contains(t, result, "/help")
	})

	t.Run("execute_exit_command_error", func(t *testing.T) {
		_, err := acpAgent.executeCommand(context.Background(), "/exit", []string{}, sessionID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not available via ACP")
	})

	t.Run("execute_unknown_command", func(t *testing.T) {
		_, err := acpAgent.executeCommand(context.Background(), "/unknown", []string{}, sessionID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})
}

