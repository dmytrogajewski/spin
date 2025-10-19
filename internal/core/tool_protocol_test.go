package core

import (
	"testing"
)

func TestToolCall_Structure(t *testing.T) {
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "execute_command",
			Arguments: `{"command": "ls -la"}`,
		},
	}

	if toolCall.ID != "call_123" {
		t.Errorf("ToolCall.ID = %v, want %v", toolCall.ID, "call_123")
	}

	if toolCall.Type != "function" {
		t.Errorf("ToolCall.Type = %v, want %v", toolCall.Type, "function")
	}

	if toolCall.Function.Name != "execute_command" {
		t.Errorf("ToolCall.Function.Name = %v, want %v", toolCall.Function.Name, "execute_command")
	}

	if toolCall.Function.Arguments != `{"command": "ls -la"}` {
		t.Errorf("ToolCall.Function.Arguments = %v, want %v", toolCall.Function.Arguments, `{"command": "ls -la"}`)
	}
}

func TestToolCallFunction_Structure(t *testing.T) {
	function := ToolCallFunction{
		Name:      "read_file",
		Arguments: `{"path": "/tmp/test.txt"}`,
	}

	if function.Name != "read_file" {
		t.Errorf("ToolCallFunction.Name = %v, want %v", function.Name, "read_file")
	}

	if function.Arguments != `{"path": "/tmp/test.txt"}` {
		t.Errorf("ToolCallFunction.Arguments = %v, want %v", function.Arguments, `{"path": "/tmp/test.txt"}`)
	}
}

func TestToolResult_Structure(t *testing.T) {
	toolResult := ToolResult{
		ID:       "call_123",
		Success:  true,
		Output:   "file content here",
		Error:    nil,
		ExitCode: 0,
	}

	if toolResult.ID != "call_123" {
		t.Errorf("ToolResult.ID = %v, want %v", toolResult.ID, "call_123")
	}

	if !toolResult.Success {
		t.Errorf("ToolResult.Success = %v, want %v", toolResult.Success, true)
	}

	if toolResult.Output != "file content here" {
		t.Errorf("ToolResult.Output = %v, want %v", toolResult.Output, "file content here")
	}

	if toolResult.Error != nil {
		t.Errorf("ToolResult.Error = %v, want %v", toolResult.Error, nil)
	}

	if toolResult.ExitCode != 0 {
		t.Errorf("ToolResult.ExitCode = %v, want %v", toolResult.ExitCode, 0)
	}
}

func TestToolResult_ErrorCase(t *testing.T) {
	err := &TestError{Message: "command failed"}
	toolResult := ToolResult{
		ID:       "call_456",
		Success:  false,
		Output:   "",
		Error:    err,
		ExitCode: 1,
	}

	if toolResult.ID != "call_456" {
		t.Errorf("ToolResult.ID = %v, want %v", toolResult.ID, "call_456")
	}

	if toolResult.Success {
		t.Errorf("ToolResult.Success = %v, want %v", toolResult.Success, false)
	}

	if toolResult.Output != "" {
		t.Errorf("ToolResult.Output = %v, want %v", toolResult.Output, "")
	}

	if toolResult.Error == nil {
		t.Errorf("ToolResult.Error = %v, want error", toolResult.Error)
	}

	if toolResult.ExitCode != 1 {
		t.Errorf("ToolResult.ExitCode = %v, want %v", toolResult.ExitCode, 1)
	}
}

func TestToolResult_EmptyValues(t *testing.T) {
	toolResult := ToolResult{}

	if toolResult.ID != "" {
		t.Errorf("ToolResult.ID = %v, want %v", toolResult.ID, "")
	}

	if toolResult.Success {
		t.Errorf("ToolResult.Success = %v, want %v", toolResult.Success, false)
	}

	if toolResult.Output != "" {
		t.Errorf("ToolResult.Output = %v, want %v", toolResult.Output, "")
	}

	if toolResult.Error != nil {
		t.Errorf("ToolResult.Error = %v, want %v", toolResult.Error, nil)
	}

	if toolResult.ExitCode != 0 {
		t.Errorf("ToolResult.ExitCode = %v, want %v", toolResult.ExitCode, 0)
	}
}

func TestToolCall_EmptyValues(t *testing.T) {
	toolCall := ToolCall{}

	if toolCall.ID != "" {
		t.Errorf("ToolCall.ID = %v, want %v", toolCall.ID, "")
	}

	if toolCall.Type != "" {
		t.Errorf("ToolCall.Type = %v, want %v", toolCall.Type, "")
	}

	if toolCall.Function.Name != "" {
		t.Errorf("ToolCall.Function.Name = %v, want %v", toolCall.Function.Name, "")
	}

	if toolCall.Function.Arguments != "" {
		t.Errorf("ToolCall.Function.Arguments = %v, want %v", toolCall.Function.Arguments, "")
	}
}

func TestToolCallFunction_EmptyValues(t *testing.T) {
	function := ToolCallFunction{}

	if function.Name != "" {
		t.Errorf("ToolCallFunction.Name = %v, want %v", function.Name, "")
	}

	if function.Arguments != "" {
		t.Errorf("ToolCallFunction.Arguments = %v, want %v", function.Arguments, "")
	}
}

func TestToolCall_MultipleToolCalls(t *testing.T) {
	toolCalls := []ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "read_file",
				Arguments: `{"path": "/tmp/file1.txt"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: ToolCallFunction{
				Name:      "write_file",
				Arguments: `{"path": "/tmp/file2.txt", "content": "hello"}`,
			},
		},
	}

	if len(toolCalls) != 2 {
		t.Errorf("len(toolCalls) = %v, want %v", len(toolCalls), 2)
	}

	if toolCalls[0].ID != "call_1" {
		t.Errorf("toolCalls[0].ID = %v, want %v", toolCalls[0].ID, "call_1")
	}

	if toolCalls[1].ID != "call_2" {
		t.Errorf("toolCalls[1].ID = %v, want %v", toolCalls[1].ID, "call_2")
	}
}

// TestError is a simple error type for testing
type TestError struct {
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}
