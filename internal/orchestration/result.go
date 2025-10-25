package orchestration

import "time"

// Result contains comprehensive turn execution results.
// It provides a summary of the turn outcome, metrics, and context information.
type Result struct {
	// Outcome
	Success    bool      `json:"success"`         // Whether turn succeeded
	FinalState TurnState `json:"final_state"`     // Final turn state
	Error      error     `json:"error,omitempty"` // Error if failed

	// Response
	Response string `json:"response"` // Final AI response

	// Metrics
	Duration  time.Duration `json:"duration"`   // Total execution time
	Tokens    TokenUsage    `json:"tokens"`     // Token usage
	ToolCount int           `json:"tool_count"` // Number of tools called

	// Context
	ContextSize int  `json:"context_size"` // Size of context used (bytes)
	Truncated   bool `json:"truncated"`    // Whether context was truncated
}
