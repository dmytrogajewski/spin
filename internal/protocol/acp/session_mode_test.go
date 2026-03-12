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
)

// TestGetAvailableModes tests that all Spin task modes are mapped to ACP session modes.
func TestGetAvailableModes(t *testing.T) {
	modes := getAvailableModes()

	require.Len(t, modes, 4, "should have 4 modes")

	// Check each mode.
	modeMap := make(map[acp.SessionModeId]acp.SessionMode)
	for _, mode := range modes {
		modeMap[mode.Id] = mode
	}

	// Regular mode.
	regular, ok := modeMap["regular"]
	require.True(t, ok, "regular mode should exist")
	assert.Equal(t, "Regular", regular.Name)
	assert.NotEmpty(t, regular.Description)

	// Review mode.
	review, ok := modeMap["review"]
	require.True(t, ok, "review mode should exist")
	assert.Equal(t, "Review", review.Name)
	assert.NotEmpty(t, review.Description)

	// Compact mode.
	compact, ok := modeMap["compact"]
	require.True(t, ok, "compact mode should exist")
	assert.Equal(t, "Compact", compact.Name)
	assert.NotEmpty(t, compact.Description)

	// Planning mode.
	planning, ok := modeMap["planning"]
	require.True(t, ok, "planning mode should exist")
	assert.Equal(t, "Planning", planning.Name)
	assert.NotEmpty(t, planning.Description)
}

// TestGetDefaultMode tests that default mode is "regular".
func TestGetDefaultMode(t *testing.T) {
	defaultMode := getDefaultMode()
	assert.Equal(t, acp.SessionModeId("regular"), defaultMode)
}

// TestNewSession_IncludesModeState tests that NewSessionResponse includes SessionModeState.
func TestNewSession_IncludesModeState(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	req := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}

	resp, err := acpAgent.NewSession(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp.Modes, "SessionModeState should be included")

	// Check available modes.
	assert.Len(t, resp.Modes.AvailableModes, 4, "should have 4 available modes")

	// Check default mode.
	assert.Equal(t, acp.SessionModeId("regular"), resp.Modes.CurrentModeId, "default mode should be regular")
}

// TestSetSessionMode_SessionNotFound tests error when session doesn't exist.
func TestSetSessionMode_SessionNotFound(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	req := acp.SetSessionModeRequest{
		SessionId: acp.SessionId("nonexistent-session"),
		ModeId:    acp.SessionModeId("regular"),
	}

	_, err = acpAgent.SetSessionMode(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestSetSessionMode_InvalidMode tests error when mode ID is invalid.
func TestSetSessionMode_InvalidMode(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	// Create a session first.
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	// Try to set invalid mode.
	req := acp.SetSessionModeRequest{
		SessionId: sessionResp.SessionId,
		ModeId:    acp.SessionModeId("invalid-mode"),
	}

	_, err = acpAgent.SetSessionMode(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

// TestSetSessionMode_Success tests successful mode change.
func TestSetSessionMode_Success(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	// Create a session first.
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	// Set mode to "review".
	req := acp.SetSessionModeRequest{
		SessionId: sessionResp.SessionId,
		ModeId:    acp.SessionModeId("review"),
	}

	resp, err := acpAgent.SetSessionMode(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify mode is stored.
	acpAgent.mu.RLock()
	storedMode, exists := acpAgent.sessionModes[sessionResp.SessionId]
	acpAgent.mu.RUnlock()

	require.True(t, exists, "mode should be stored")
	assert.Equal(t, acp.SessionModeId("review"), storedMode)
}

// TestSetSessionMode_AllModes tests that all valid modes can be set.
func TestSetSessionMode_AllModes(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	// Create a session.
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	// Test all modes.
	modes := []acp.SessionModeId{"regular", "review", "compact", "planning"}

	for _, mode := range modes {
		req := acp.SetSessionModeRequest{
			SessionId: sessionResp.SessionId,
			ModeId:    mode,
		}

		var resp acp.SetSessionModeResponse
		resp, err = acpAgent.SetSessionMode(context.Background(), req)
		require.NoError(t, err, "should set mode %s", mode)
		require.NotNil(t, resp)

		// Verify mode is stored.
		acpAgent.mu.RLock()
		storedMode, exists := acpAgent.sessionModes[sessionResp.SessionId]
		acpAgent.mu.RUnlock()

		require.True(t, exists, "mode should be stored")
		assert.Equal(t, mode, storedMode, "stored mode should match")
	}
}

// TestSetSessionMode_SendsNotification tests that mode change sends notification.
func TestSetSessionMode_SendsNotification(t *testing.T) {
	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	// Create mock connection.
	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	// Create a session.
	sessionReq := acp.NewSessionRequest{
		Cwd: "/tmp/test",
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	// Set mode.
	req := acp.SetSessionModeRequest{
		SessionId: sessionResp.SessionId,
		ModeId:    acp.SessionModeId("review"),
	}

	_, err = acpAgent.SetSessionMode(context.Background(), req)
	require.NoError(t, err)

	// Verify notification was sent.
	notifications := mockConn.GetNotifications()
	require.Greater(t, len(notifications), 0, "should have at least one notification")

	// Find the mode update notification.
	found := false

	for _, notif := range notifications {
		if notif.Update.CurrentModeUpdate != nil {
			found = true

			assert.Equal(t, acp.SessionModeId("review"), notif.Update.CurrentModeUpdate.CurrentModeId)

			break
		}
	}

	assert.True(t, found, "should send CurrentModeUpdate notification")
}
