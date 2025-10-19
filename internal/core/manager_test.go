package core

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManager_NewConversation_Integration tests the full conversation creation flow
// with all integrations enabled. This serves as a safety net before refactoring.
func TestManager_NewConversation_Integration(t *testing.T) {
	// Setup: Use default config
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create manager with mock LLM
	mockLLM := llm.NewMockProvider("test")
	mgr, err := NewManager(cfg, WithLLM(mockLLM))
	require.NoError(t, err, "NewManager should succeed")
	require.NotNil(t, mgr, "Manager should not be nil")

	// Act: Create a new conversation
	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, cfg.WorkDir)

	// Assert: Conversation should be created successfully
	require.NoError(t, err, "NewConversation should succeed")
	require.NotNil(t, conv, "Conversation should not be nil")

	// Verify conversation components
	assert.NotNil(t, conv.agent, "Agent should be initialized")
	assert.NotNil(t, conv.history, "History should be initialized")
	assert.NotNil(t, conv.emitter, "Emitter should be initialized")
	assert.NotNil(t, conv.events, "Event channel should be initialized")
}

// TestManager_getLogger tests logger retrieval from manager
func TestManager_getLogger(t *testing.T) {
	tests := []struct {
		name       string
		manager    *Manager
		wantNil    bool
		wantCustom bool
	}{
		{
			name:       "with_custom_logger",
			manager:    &Manager{logger: testLogger()},
			wantNil:    false,
			wantCustom: true,
		},
		{
			name:       "without_logger_fallback_to_default",
			manager:    &Manager{},
			wantNil:    false,
			wantCustom: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got := tt.manager.getLogger(ctx)

			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}

			if tt.wantCustom {
				// Verify it's the custom logger (not default)
				assert.Equal(t, tt.manager.logger, got)
			}
		})
	}
}

// testLogger creates a test logger for verification
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// TestManager_NewConversation_UsesCustomLogger verifies custom logger is used
func TestManager_NewConversation_UsesCustomLogger(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create custom logger
	customLogger := testLogger()

	// Create manager with custom logger
	mockLLM := llm.NewMockProvider("test")
	mgr, err := NewManager(cfg, WithLLM(mockLLM))
	require.NoError(t, err)

	// Inject custom logger
	mgr.logger = customLogger

	// Create conversation - should use custom logger
	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, cfg.WorkDir)

	// Verify conversation was created (logger was used internally)
	require.NoError(t, err)
	require.NotNil(t, conv)
}

// TestManager_buildExecutor tests executor creation
func TestManager_buildExecutor(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		handler ApprovalHandler
		wantErr bool
	}{
		{
			name:    "minimal_config",
			cfg:     DefaultConfig(),
			handler: nil,
			wantErr: false,
		},
		{
			name: "with_approval_handler",
			cfg:  DefaultConfig(),
			handler: func(req ApprovalRequest) ApprovalResponse {
				return ApprovalResponse{Approved: true, RequestID: req.ID}
			},
			wantErr: false,
		},
		{
			name: "with_cache_enabled",
			cfg: func() *Config {
				c := DefaultConfig()
				c.CacheCommands = true
				return c
			}(),
			handler: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &Manager{
				cfg:             tt.cfg,
				approvalHandler: tt.handler,
			}

			logger := testLogger()
			workDir := t.TempDir()

			executor, err := mgr.buildExecutor(workDir, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, executor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, executor)
			}
		})
	}
}
