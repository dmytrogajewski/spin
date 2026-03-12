package tools

import (
	"errors"

	"github.com/dmytrogajewski/spin/internal/message"
)

var (
	ErrToolCallCannotBeNil = errors.New("tool call cannot be nil")
	ErrToolCallIdCannotBeEmpty = errors.New("tool call ID cannot be empty")
	ErrToolFunctionNameCannotBeEmpty = errors.New("tool function name cannot be empty")
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
		return ErrToolCallCannotBeNil
	}

	if call.ID == "" {
		return ErrToolCallIdCannotBeEmpty
	}

	if call.Function.Name == "" {
		return ErrToolFunctionNameCannotBeEmpty
	}

	return nil
}
