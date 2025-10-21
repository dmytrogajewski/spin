package agent

import (
	"errors"
	"testing"

	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/stretchr/testify/assert"
)

// TestGetToolResultContent tests that error messages are properly sent to LLM on tool failure
func TestGetToolResultContent(t *testing.T) {
	tests := []struct {
		name     string
		toolCall *orchestration.ToolCall
		result   *orchestration.ToolResult
		want     string
	}{
		{
			name: "successful tool call returns output",
			toolCall: &orchestration.ToolCall{
				ID: "call_1",
				Function: orchestration.ToolCallFunction{
					Name:      "list_directory",
					Arguments: `{"path":"."}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_1",
				Success: true,
				Output:  "file1.go\nfile2.go\nREADME.md",
			},
			want: "file1.go\nfile2.go\nREADME.md",
		},
		{
			name: "failed tool call with error returns error message",
			toolCall: &orchestration.ToolCall{
				ID: "call_2",
				Function: orchestration.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path":"nonexistent.txt"}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_2",
				Success: false,
				Error:   errors.New("file not found: nonexistent.txt"),
			},
			want: "Tool read_file failed: file not found: nonexistent.txt",
		},
		{
			name: "failed tool call without error message",
			toolCall: &orchestration.ToolCall{
				ID: "call_3",
				Function: orchestration.ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"cmd":"unknown"}`,
				},
			},
			result: &orchestration.ToolResult{
				ID:      "call_3",
				Success: false,
			},
			want: "Tool execute_command failed with no error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getToolResultContent(tt.toolCall, tt.result)
			assert.Equal(t, tt.want, got)
		})
	}
}
