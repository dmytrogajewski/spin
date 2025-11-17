package tools

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateToolCall_Valid(t *testing.T) {
	call := &message.ToolCall{
		ID:   "test-id",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "test_function",
			Arguments: `{"key": "value"}`,
		},
	}

	err := ValidateToolCall(call)
	assert.NoError(t, err)
}

func TestValidateToolCall_Nil(t *testing.T) {
	err := ValidateToolCall(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestValidateToolCall_EmptyID(t *testing.T) {
	call := &message.ToolCall{
		ID:   "",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "test_function",
			Arguments: `{"key": "value"}`,
		},
	}

	err := ValidateToolCall(call)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID cannot be empty")
}

func TestValidateToolCall_EmptyFunctionName(t *testing.T) {
	call := &message.ToolCall{
		ID:   "test-id",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "",
			Arguments: `{"key": "value"}`,
		},
	}

	err := ValidateToolCall(call)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "function name cannot be empty")
}

func TestValidateToolCall_EmptyType(t *testing.T) {
	// Empty Type should not cause validation error (not required)
	call := &message.ToolCall{
		ID:   "test-id",
		Type: "",
		Function: message.ToolCallFunction{
			Name:      "test_function",
			Arguments: `{"key": "value"}`,
		},
	}

	err := ValidateToolCall(call)
	assert.NoError(t, err)
}

func TestValidateToolCall_EmptyArguments(t *testing.T) {
	// Empty Arguments should not cause validation error (not required)
	call := &message.ToolCall{
		ID:   "test-id",
		Type: "function",
		Function: message.ToolCallFunction{
			Name:      "test_function",
			Arguments: "",
		},
	}

	err := ValidateToolCall(call)
	assert.NoError(t, err)
}


