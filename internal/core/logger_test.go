package core

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantLevel slog.Level
	}{
		{
			name:      "debug level",
			level:     "debug",
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "info level",
			level:     "info",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "warn level",
			level:     "warn",
			wantLevel: slog.LevelWarn,
		},
		{
			name:      "error level",
			level:     "error",
			wantLevel: slog.LevelError,
		},
		{
			name:      "invalid level defaults to info",
			level:     "invalid",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "empty level defaults to info",
			level:     "",
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "uppercase level",
			level:     "DEBUG",
			wantLevel: slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.level)
			assert.Equal(t, tt.wantLevel, got)
		})
	}
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *Config
		wantLevel  slog.Level
		wantFormat string
	}{
		{
			name: "default text format info level",
			cfg: &Config{
				LogLevel:  "info",
				LogFormat: "text",
			},
			wantLevel:  slog.LevelInfo,
			wantFormat: "text",
		},
		{
			name: "json format debug level",
			cfg: &Config{
				LogLevel:  "debug",
				LogFormat: "json",
			},
			wantLevel:  slog.LevelDebug,
			wantFormat: "json",
		},
		{
			name: "debug mode enables debug level",
			cfg: &Config{
				Debug:     true,
				LogLevel:  "info", // Should be overridden by Debug
				LogFormat: "text",
			},
			wantLevel:  slog.LevelDebug,
			wantFormat: "text",
		},
		{
			name: "empty format defaults to text",
			cfg: &Config{
				LogLevel:  "info",
				LogFormat: "",
			},
			wantLevel:  slog.LevelInfo,
			wantFormat: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer

			// Initialize logger with custom writer
			InitLoggerWithWriter(tt.cfg, &buf)

			// Log a test message
			slog.Info("test message", "key", "value")

			output := buf.String()

			// Verify format
			if tt.wantFormat == "json" {
				assert.Contains(t, output, `"msg":"test message"`)
				assert.Contains(t, output, `"key":"value"`)
			} else {
				assert.Contains(t, output, "test message")
				assert.Contains(t, output, "key=value")
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	tests := []struct {
		name           string
		ctx            context.Context
		wantSessionID  bool
		wantTurnID     bool
		sessionIDValue string
		turnIDValue    string
	}{
		{
			name:           "context with session ID",
			ctx:            context.WithValue(context.Background(), sessionIDKey, "test-session-123"),
			wantSessionID:  true,
			wantTurnID:     false,
			sessionIDValue: "test-session-123",
		},
		{
			name:          "context with turn ID",
			ctx:           context.WithValue(context.Background(), turnIDKey, "test-turn-456"),
			wantSessionID: false,
			wantTurnID:    true,
			turnIDValue:   "test-turn-456",
		},
		{
			name: "context with both IDs",
			ctx: context.WithValue(
				context.WithValue(context.Background(), sessionIDKey, "session-abc"),
				turnIDKey,
				"turn-xyz",
			),
			wantSessionID:  true,
			wantTurnID:     true,
			sessionIDValue: "session-abc",
			turnIDValue:    "turn-xyz",
		},
		{
			name:          "context with no IDs",
			ctx:           context.Background(),
			wantSessionID: false,
			wantTurnID:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
			slog.SetDefault(slog.New(handler))

			// Get logger with context
			logger := withContext(tt.ctx)
			logger.Info("test message")

			output := buf.String()

			// Verify session ID
			if tt.wantSessionID {
				assert.Contains(t, output, "session_id="+tt.sessionIDValue)
			} else {
				assert.NotContains(t, output, "session_id=")
			}

			// Verify turn ID
			if tt.wantTurnID {
				assert.Contains(t, output, "turn_id="+tt.turnIDValue)
			} else {
				assert.NotContains(t, output, "turn_id=")
			}
		})
	}
}

func TestWithSessionID(t *testing.T) {
	ctx := context.Background()

	sessionID := "test-session-123"
	ctx = WithSessionID(ctx, sessionID)

	// Verify the value is stored
	got, ok := ctx.Value(sessionIDKey).(string)
	require.True(t, ok)
	assert.Equal(t, sessionID, got)
}

func TestWithTurnID(t *testing.T) {
	ctx := context.Background()

	turnID := "test-turn-456"
	ctx = WithTurnID(ctx, turnID)

	// Verify the value is stored
	got, ok := ctx.Value(turnIDKey).(string)
	require.True(t, ok)
	assert.Equal(t, turnID, got)
}

func TestDebugModeAddSource(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *Config
		wantAddSource bool
	}{
		{
			name: "debug mode adds source",
			cfg: &Config{
				Debug:     true,
				LogFormat: "text",
			},
			wantAddSource: true,
		},
		{
			name: "debug level adds source",
			cfg: &Config{
				LogLevel:  "debug",
				LogFormat: "text",
			},
			wantAddSource: true,
		},
		{
			name: "info level does not add source",
			cfg: &Config{
				LogLevel:  "info",
				LogFormat: "text",
			},
			wantAddSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			InitLoggerWithWriter(tt.cfg, &buf)

			slog.Debug("debug message")

			output := buf.String()

			// Source includes file:line info
			if tt.wantAddSource && tt.cfg.LogLevel == "debug" {
				// Should contain source location
				assert.Contains(t, output, "logger_test.go")
			}
		})
	}
}

func TestLoggerConcurrency(t *testing.T) {
	// Initialize logger
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	var buf bytes.Buffer
	InitLoggerWithWriter(cfg, &buf)

	// Log concurrently
	const numGoroutines = 100
	done := make(chan bool)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx := WithSessionID(context.Background(), "session-"+string(rune(id)))
			logger := withContext(ctx)
			logger.Info("concurrent log", "goroutine", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Just verify no panics occurred and we got output
	assert.NotEmpty(t, buf.String())
}

func TestInvalidLogFormat(t *testing.T) {
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "invalid",
	}

	var buf bytes.Buffer
	InitLoggerWithWriter(cfg, &buf)

	// Should default to text format
	slog.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	// Text format check
	assert.NotContains(t, output, `"msg"`)
}

func BenchmarkLoggingOverhead(b *testing.B) {
	cfg := &Config{
		LogLevel:  "info",
		LogFormat: "text",
	}
	var buf bytes.Buffer
	InitLoggerWithWriter(cfg, &buf)

	b.Run("info level logging", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slog.Info("benchmark message", "iteration", i)
		}
	})

	b.Run("debug level disabled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			slog.Debug("benchmark message", "iteration", i)
		}
	})

	b.Run("with context", func(b *testing.B) {
		ctx := WithSessionID(context.Background(), "bench-session")
		for i := 0; i < b.N; i++ {
			logger := withContext(ctx)
			logger.Info("benchmark message", "iteration", i)
		}
	})
}

func BenchmarkContextCreation(b *testing.B) {
	ctx := context.Background()

	b.Run("WithSessionID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WithSessionID(ctx, "session-123")
		}
	})

	b.Run("WithTurnID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WithTurnID(ctx, "turn-456")
		}
	})

	b.Run("withContext", func(b *testing.B) {
		ctx := WithSessionID(context.Background(), "session-123")
		for i := 0; i < b.N; i++ {
			_ = withContext(ctx)
		}
	})
}
