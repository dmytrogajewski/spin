package core

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Context keys for logging
type contextKey string

const (
	sessionIDKey contextKey = "session_id"
	turnIDKey    contextKey = "turn_id"
)

// InitLogger initializes the global logger based on configuration.
// It sets up structured logging with the specified level and format.
func InitLogger(cfg *Config) {
	InitLoggerWithWriter(cfg, os.Stderr)
}

// InitLoggerWithWriter initializes the logger with a custom writer.
// This is primarily used for testing to capture log output.
func InitLoggerWithWriter(cfg *Config, w io.Writer) {
	// Determine log level
	level := parseLogLevel(cfg.LogLevel)

	// Debug mode overrides log level
	if cfg.Debug {
		level = slog.LevelDebug
	}

	// Handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Add source file:line in debug mode
	}

	// Create handler based on format
	var handler slog.Handler
	format := strings.ToLower(cfg.LogFormat)
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		// Default to text format for empty or invalid values
		handler = slog.NewTextHandler(w, opts)
	}

	// Set as default logger
	slog.SetDefault(slog.New(handler))
}

// parseLogLevel converts a string log level to slog.Level.
// Valid levels: debug, info, warn, error (case-insensitive).
// Invalid levels default to Info with a warning.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		if level != "" {
			// Only warn if a level was actually specified
			slog.Warn("invalid log level, using info", "level", level)
		}
		return slog.LevelInfo
	}
}

// withContext creates a logger with context fields extracted from ctx.
// It adds session_id and turn_id if present in the context.
func withContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	// Extract and add session ID if present
	if sessionID, ok := ctx.Value(sessionIDKey).(string); ok {
		logger = logger.With("session_id", sessionID)
	}

	// Extract and add turn ID if present
	if turnID, ok := ctx.Value(turnIDKey).(string); ok {
		logger = logger.With("turn_id", turnID)
	}

	return logger
}

// WithSessionID adds a session ID to the context for logging.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// WithTurnID adds a turn ID to the context for logging.
func WithTurnID(ctx context.Context, turnID string) context.Context {
	return context.WithValue(ctx, turnIDKey, turnID)
}
