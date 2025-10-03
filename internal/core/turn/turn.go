// Package turn provides turn state management for conversations.
package turn

import "time"

// Turn represents a single user-AI interaction.
// This is a minimal implementation for Feature 1.1 (Session Management).
// Full implementation will be done in Feature 1.2.
type Turn struct {
	ID          string    // Unique turn identifier
	SessionID   string    // Parent session ID
	UserInput   string    // User's input
	AIResponse  string    // AI's response
	State       TurnState // Turn execution state
	StartedAt   time.Time // Turn start timestamp
	CompletedAt time.Time // Turn completion timestamp
	TokensUsed  int       // Total tokens consumed
}
