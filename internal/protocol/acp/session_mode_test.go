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
	t.Parallel()

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
	t.Parallel()

	defaultMode := getDefaultMode()
	assert.Equal(t, acp.SessionModeId("regular"), defaultMode)
}

// TestNewSession_IncludesModeState tests that NewSessionResponse includes SessionModeState.
func TestNewSession_IncludesModeState(t *testing.T) {
	t.Parallel()
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

// Journey: specs/journeys/JOURNEY-R1.3-acp-config-options-in-session.md.

// TestNewSession_IncludesConfigOptions tests that NewSessionResponse includes ConfigOptions.
func TestNewSession_IncludesConfigOptions(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	resp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	// Verify config options present.
	require.NotEmpty(t, resp.ConfigOptions, "NewSessionResponse should include config options")
	require.NotNil(t, resp.ConfigOptions[0].Select, "first config option must be select type")

	modeOpt := resp.ConfigOptions[0].Select
	assert.Equal(t, acp.SessionConfigId(configIDMode), modeOpt.Id)
	assert.Equal(t, acp.SessionConfigValueId("regular"), modeOpt.CurrentValue, "default should be regular")

	require.NotNil(t, modeOpt.Options.Ungrouped)
	assert.Len(t, *modeOpt.Options.Ungrouped, 4, "should have 4 mode options")

	// Verify backward compat: modes field still populated.
	require.NotNil(t, resp.Modes, "legacy Modes field should still be populated")
}

// TestSetSessionMode_SessionNotFound tests error when session doesn't exist.
func TestSetSessionMode_SessionNotFound(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()

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
	require.NotEmpty(t, notifications, "should have at least one notification")

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

// Journey: specs/journeys/JOURNEY-R1.1-acp-config-option-mode.md.

// TestSetSessionConfigOption_Success tests setting mode via config option.
func TestSetSessionConfigOption_Success(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	resp, err := acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: sessionResp.SessionId,
		ConfigId:  acp.SessionConfigId(configIDMode),
		Value:     acp.SessionConfigValueId("review"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ConfigOptions, "response must include config options")

	// Verify mode was updated in internal state.
	acpAgent.mu.RLock()
	storedMode := acpAgent.sessionModes[sessionResp.SessionId]
	acpAgent.mu.RUnlock()
	assert.Equal(t, acp.SessionModeId("review"), storedMode)

	// Verify response config options reflect new mode.
	require.NotNil(t, resp.ConfigOptions[0].Select, "first config option must be select type")
	assert.Equal(t, acp.SessionConfigValueId("review"), resp.ConfigOptions[0].Select.CurrentValue)
}

// TestSetSessionConfigOption_AllModes tests all valid modes via config option.
func TestSetSessionConfigOption_AllModes(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		resp, setErr := acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
			SessionId: sessionResp.SessionId,
			ConfigId:  acp.SessionConfigId(configIDMode),
			Value:     acp.SessionConfigValueId(mode),
		})
		require.NoError(t, setErr, "should set mode %s", mode)
		assert.Equal(t, acp.SessionConfigValueId(mode), resp.ConfigOptions[0].Select.CurrentValue)
	}
}

// TestSetSessionConfigOption_SessionNotFound tests error for missing session.
func TestSetSessionConfigOption_SessionNotFound(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	_, err = acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: acp.SessionId("nonexistent"),
		ConfigId:  acp.SessionConfigId(configIDMode),
		Value:     acp.SessionConfigValueId("review"),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

// TestSetSessionConfigOption_InvalidMode tests error for invalid mode value.
func TestSetSessionConfigOption_InvalidMode(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	_, err = acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: sessionResp.SessionId,
		ConfigId:  acp.SessionConfigId(configIDMode),
		Value:     acp.SessionConfigValueId("nonexistent-mode"),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMode)
}

// TestSetSessionConfigOption_UnknownConfigID tests error for unknown config option.
func TestSetSessionConfigOption_UnknownConfigID(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		createTestAgent(t),
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	_, err = acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: sessionResp.SessionId,
		ConfigId:  acp.SessionConfigId("model"),
		Value:     acp.SessionConfigValueId("gpt-4"),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownConfigOption)
}

// TestSetSessionConfigOption_SendsNotification tests that config option change sends notification.
func TestSetSessionConfigOption_SendsNotification(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	_, err = acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: sessionResp.SessionId,
		ConfigId:  acp.SessionConfigId(configIDMode),
		Value:     acp.SessionConfigValueId("review"),
	})
	require.NoError(t, err)

	// Verify CurrentModeUpdate notification was sent (backward compat).
	notifications := mockConn.GetNotifications()
	found := false

	for _, notif := range notifications {
		if notif.Update.CurrentModeUpdate != nil {
			found = true

			assert.Equal(t, acp.SessionModeId("review"), notif.Update.CurrentModeUpdate.CurrentModeId)

			break
		}
	}

	assert.True(t, found, "should send CurrentModeUpdate notification for backward compat")
}

// TestBuildConfigOptions tests config options construction.
func TestBuildConfigOptions(t *testing.T) {
	t.Parallel()

	opts := buildConfigOptions(acp.SessionModeId("review"))
	require.Len(t, opts, 1, "should have one config option (mode)")

	modeOpt := opts[0]
	require.NotNil(t, modeOpt.Select, "config option must be select type")
	assert.Equal(t, acp.SessionConfigId(configIDMode), modeOpt.Select.Id)
	assert.Equal(t, "Mode", modeOpt.Select.Name)
	assert.Equal(t, acp.SessionConfigValueId("review"), modeOpt.Select.CurrentValue)

	// Verify all 4 modes are in options.
	require.NotNil(t, modeOpt.Select.Options.Ungrouped)
	assert.Len(t, *modeOpt.Select.Options.Ungrouped, 4)
}

// Journey: specs/journeys/JOURNEY-R1.2-acp-config-option-notify.md.

// TestSetSessionConfigOption_EmitsConfigOptionUpdate tests config_option_update notification.
func TestSetSessionConfigOption_EmitsConfigOptionUpdate(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	_, err = acpAgent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		SessionId: sessionResp.SessionId,
		ConfigId:  acp.SessionConfigId(configIDMode),
		Value:     acp.SessionConfigValueId("planning"),
	})
	require.NoError(t, err)

	// Verify ConfigOptionUpdate notification.
	notifications := mockConn.GetNotifications()
	foundConfig := false
	foundMode := false

	for _, notif := range notifications {
		if notif.Update.ConfigOptionUpdate != nil {
			foundConfig = true

			require.NotEmpty(t, notif.Update.ConfigOptionUpdate.ConfigOptions)
			assert.Equal(t,
				acp.SessionConfigValueId("planning"),
				notif.Update.ConfigOptionUpdate.ConfigOptions[0].Select.CurrentValue,
			)
		}

		if notif.Update.CurrentModeUpdate != nil {
			foundMode = true
		}
	}

	assert.True(t, foundConfig, "should send ConfigOptionUpdate notification")
	assert.True(t, foundMode, "should also send CurrentModeUpdate for backward compat")
}

// TestSetSessionMode_EmitsConfigOptionUpdate tests that legacy set_mode also emits config_option_update.
func TestSetSessionMode_EmitsConfigOptionUpdate(t *testing.T) {
	t.Parallel()

	acpAgent, err := NewSpinACPAgentWithStorage(
		&agent.Agent{},
		mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default())),
		events.NewEventEmitter(100),
		nil,
	)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)

	_, err = acpAgent.SetSessionMode(context.Background(), acp.SetSessionModeRequest{
		SessionId: sessionResp.SessionId,
		ModeId:    acp.SessionModeId("compact"),
	})
	require.NoError(t, err)

	// Verify both notifications sent.
	notifications := mockConn.GetNotifications()
	foundConfig := false

	for _, notif := range notifications {
		if notif.Update.ConfigOptionUpdate != nil {
			foundConfig = true

			require.NotEmpty(t, notif.Update.ConfigOptionUpdate.ConfigOptions)
			assert.Equal(t,
				acp.SessionConfigValueId("compact"),
				notif.Update.ConfigOptionUpdate.ConfigOptions[0].Select.CurrentValue,
			)
		}
	}

	assert.True(t, foundConfig, "legacy SetSessionMode should also emit ConfigOptionUpdate")
}
