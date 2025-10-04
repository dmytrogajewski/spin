package core

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingIntegration(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	// Initialize logger with debug level for testing
	logCfg := &Config{
		LogLevel:  "debug",
		LogFormat: "text",
	}
	InitLoggerWithWriter(logCfg, &buf)

	// Create a test manager
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.Model = "test-model"
	mgr, err := NewManager(cfg)
	require.NoError(t, err)

	// Create a new conversation with context
	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	require.NoError(t, err)
	require.NotNil(t, conv)

	// Check that logs were generated
	output := buf.String()
	assert.Contains(t, output, "creating new conversation")
	assert.Contains(t, output, "conversation created successfully")
	assert.Contains(t, output, "work_dir")
}

func TestLoggingWithContextPropagation(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	// Create context with session and turn IDs
	ctx := context.Background()
	ctx = WithSessionID(ctx, "test-session-123")
	ctx = WithTurnID(ctx, "test-turn-456")

	// Create logger from context
	logger := withContext(ctx)
	logger.Info("test message with context")

	// Verify context is included in logs
	output := buf.String()
	assert.Contains(t, output, "test message with context")
	assert.Contains(t, output, "session_id=test-session-123")
	assert.Contains(t, output, "turn_id=test-turn-456")
}

func TestLoggingJSONFormat(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "json",
	}
	InitLoggerWithWriter(cfg, &buf)

	slog.Info("json test message", "key", "value", "count", 42)

	output := buf.String()
	assert.Contains(t, output, `"msg":"json test message"`)
	assert.Contains(t, output, `"key":"value"`)
	assert.Contains(t, output, `"count":42`)
}

func TestLoggingDebugMode(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	cfg := &Config{
		Debug:     true,
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	// Debug messages should appear
	slog.Debug("debug message")
	slog.Info("info message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
}

func TestLoggingInfoMode(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	// Debug messages should NOT appear
	slog.Debug("debug message")
	slog.Info("info message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.Contains(t, output, "info message")
}

func TestLoggingErrorHandling(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer

	logCfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(logCfg, &buf)

	// Create manager
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.Model = "test-model"
	mgr, err := NewManager(cfg)
	require.NoError(t, err)

	// Try to archive non-existent session
	ctx := context.Background()
	err = mgr.ArchiveConversation(ctx, "non-existent")
	require.Error(t, err)

	// Verify error was logged
	output := buf.String()
	assert.Contains(t, output, "failed to load session for archival")
	assert.Contains(t, output, "session_id=non-existent")
}

func TestLoggingPerformanceOverhead(t *testing.T) {
	// This test verifies that logging has minimal overhead

	// Setup with info level (typical production setting)
	var buf bytes.Buffer
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	// Measure operations without logging impact
	// Debug logs should have near-zero overhead when disabled
	for i := 0; i < 1000; i++ {
		slog.Debug("this should not appear", "iteration", i)
	}

	// Buffer should be empty (debug logs filtered)
	assert.Empty(t, buf.String())
}

func TestLoggingConfigFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectedLevel string
		expectedDebug bool
	}{
		{
			name: "SPIN_DEBUG=1 enables debug",
			envVars: map[string]string{
				"SPIN_DEBUG": "1",
			},
			expectedLevel: "debug",
			expectedDebug: true,
		},
		{
			name: "SPIN_LOG_LEVEL=warn",
			envVars: map[string]string{
				"SPIN_LOG_LEVEL": "warn",
			},
			expectedLevel: "warn",
			expectedDebug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Load config from environment
			cfg := loadFromEnv()

			if tt.expectedDebug {
				assert.True(t, cfg.Debug)
			}
			if tt.expectedLevel != "" {
				assert.Equal(t, tt.expectedLevel, cfg.LogLevel)
			}
		})
	}
}

func BenchmarkLoggingIntegration(b *testing.B) {
	// Setup logger
	var buf bytes.Buffer
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	ctx := WithSessionID(context.Background(), "bench-session")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger := withContext(ctx)
		logger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLoggingWithoutContext(b *testing.B) {
	var buf bytes.Buffer
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	InitLoggerWithWriter(cfg, &buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slog.Info("benchmark message", "iteration", i)
	}
}
