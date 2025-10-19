package core

import (
	"context"
	"log/slog"
)

// Context keys for logging
type contextKey string

const (
	sessionIDKey contextKey = "session_id"
	turnIDKey    contextKey = "turn_id"
)

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
