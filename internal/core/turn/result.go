package turn

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

// NewResult creates a Result from a Turn.
// It extracts all relevant information and calculates metrics.
//
// Parameters:
//   - turn: The turn to create a result from
//
// Returns a new Result with all fields populated.
func NewResult(turn *Turn) *Result {
	turn.mu.RLock()
	defer turn.mu.RUnlock()

	result := &Result{
		Success:    turn.State == StateCompleted,
		FinalState: turn.State,
		Error:      turn.Error,
		Response:   turn.AIResponse,
		Tokens:     turn.Tokens,
		ToolCount:  len(turn.ToolCalls),
	}

	// Calculate duration
	if !turn.StartedAt.IsZero() && !turn.CompletedAt.IsZero() {
		result.Duration = turn.CompletedAt.Sub(turn.StartedAt)
	}

	// Calculate context size
	result.ContextSize = len(turn.UserInput) + len(turn.AIResponse)

	// Add tool call/result sizes
	for _, call := range turn.ToolCalls {
		result.ContextSize += len(call.Name)
	}
	for _, res := range turn.ToolResults {
		if str, ok := res.Result.(string); ok {
			result.ContextSize += len(str)
		}
	}

	return result
}

// IsSuccess returns true if the turn completed successfully.
func (r *Result) IsSuccess() bool {
	return r.Success
}

// GetError returns the error if the turn failed, nil otherwise.
func (r *Result) GetError() error {
	return r.Error
}
