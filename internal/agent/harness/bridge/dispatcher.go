package bridge

import (
	"context"

	agenttool "github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// DispatcherBridge adapts tool.Runtime to the harness.ToolDispatcher interface.
// It processes tool calls sequentially and builds the updated message list.
type DispatcherBridge struct {
	runtime *agenttool.Runtime
}

// NewDispatcherBridge creates a DispatcherBridge wrapping the given tool runtime.
func NewDispatcherBridge(runtime *agenttool.Runtime) *DispatcherBridge {
	return &DispatcherBridge{runtime: runtime}
}

// Dispatch executes tool calls and returns the updated message list.
// Appends the assistant message (with tool calls) and each tool result.
func (b *DispatcherBridge) Dispatch(
	ctx context.Context,
	msgs []message.Message,
	content string,
	toolCalls []message.ToolCall,
) []message.Message {
	result := make([]message.Message, 0, len(msgs)+len(toolCalls)+1)
	result = append(result, msgs...)

	// Append assistant message with tool calls.
	result = append(result, message.Message{
		Role:      message.RoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	})

	// Execute each tool call and append the result message.
	for idx := range toolCalls {
		tc := &toolCalls[idx]
		toolResult, err := b.runtime.Execute(ctx, tc)

		resultContent := ""
		if err != nil {
			resultContent = "Error: " + err.Error()
		} else if toolResult != nil {
			resultContent = buildToolResultContent(toolResult)
		}

		result = append(result, message.Message{
			Role:       message.RoleTool,
			Content:    resultContent,
			ToolCallID: tc.ID,
		})
	}

	return result
}

// buildToolResultContent constructs the message content from a tool result.
// For failed tools, it includes both the output (e.g., compiler errors) and
// the error summary so the LLM can see the full picture.
func buildToolResultContent(r *tools.ToolResult) string {
	if r.Success {
		return r.Output
	}

	// Failed tool — combine output and error for the LLM.
	if r.Output != "" && r.Error != "" {
		return r.Output + "\nError: " + r.Error
	}

	if r.Error != "" {
		return "Error: " + r.Error
	}

	return r.Output
}
