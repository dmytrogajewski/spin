package tools

import (
	"errors"

	"github.com/dmytrogajewski/spin/internal/message"
)

// ValidateToolCall validates the tool call structure.
// Returns an error if the tool call is invalid.
//
// Validation checks:
//   - Tool call must not be nil
//   - Tool call ID must not be empty
//   - Tool function name must not be empty
func ValidateToolCall(call *message.ToolCall) error {
	if call == nil {
		return errors.New("tool call cannot be nil")
	}

	if call.ID == "" {
		return errors.New("tool call ID cannot be empty")
	}

	if call.Function.Name == "" {
		return errors.New("tool function name cannot be empty")
	}

	return nil
}
