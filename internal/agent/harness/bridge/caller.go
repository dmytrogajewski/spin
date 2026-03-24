// Package bridge provides adapters that connect the harness execution loop
// to the existing LLM calling and tool execution infrastructure.
package bridge

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// CallerBridge adapts caller.LLMCaller to the harness.Caller interface.
// It holds pre-resolved call parameters and a tool registry for type
// conversions required by the existing LLM caller infrastructure.
type CallerBridge struct {
	llmCaller  *caller.LLMCaller
	registry   *tools.Registry
	callParams agent.CallParams
}

// NewCallerBridge creates a CallerBridge wrapping the given LLM caller.
func NewCallerBridge(
	llmCaller *caller.LLMCaller,
	registry *tools.Registry,
	params agent.CallParams,
) *CallerBridge {
	return &CallerBridge{
		llmCaller:  llmCaller,
		registry:   registry,
		callParams: params,
	}
}

// Call delegates to the LLM caller with retry logic and converts the result.
func (b *CallerBridge) Call(
	ctx context.Context,
	messages []message.Message,
	toolSchemas []tools.ToolSchema,
	turn int,
) (content string, toolCalls []message.ToolCall, finishReason string, err error) {
	toolList := b.resolveTools(toolSchemas)
	resp := &agent.Response{}

	content, toolCalls, finishReason, err = b.llmCaller.CallWithRetries(
		ctx, messages, b.callParams, toolList, nil, turn, resp,
	)
	if err != nil {
		return "", nil, "", err
	}

	return content, toolCalls, finishReason, nil
}

// resolveTools converts tool schemas to tools by looking them up in the registry.
func (b *CallerBridge) resolveTools(schemas []tools.ToolSchema) []tools.Tool {
	if b.registry == nil {
		return nil
	}

	result := make([]tools.Tool, 0, len(schemas))

	for _, schema := range schemas {
		t, err := b.registry.Get(schema.Function.Name)
		if err == nil {
			result = append(result, t)
		}
	}

	return result
}
