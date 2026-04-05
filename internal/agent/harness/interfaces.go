// Package harness implements the Extended ReAct execution loop.
// It consumes compiled scaffold.Spec values and runs an iterative
// action-dispatch loop with pluggable middleware and guard hooks.
package harness

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Caller abstracts LLM invocation for the harness loop.
// Unlike the legacy Caller, it accepts pre-compiled ToolSchemas
// instead of task.Task, enabling structural tool isolation.
type Caller interface {
	Call(
		ctx context.Context,
		messages []message.Message,
		toolSchemas []tools.ToolSchema,
		turn int,
	) (content string, toolCalls []message.ToolCall, finishReason string, err error)
}

// ToolDispatcher executes tool calls and returns updated messages.
// It wraps the tool runtime with message formatting.
type ToolDispatcher interface {
	Dispatch(
		ctx context.Context,
		messages []message.Message,
		content string,
		toolCalls []message.ToolCall,
	) []message.Message
}

// Guard inspects the loop state and can inject messages or halt execution.
// The second return value signals whether the loop should stop.
type Guard interface {
	Check(
		ctx context.Context,
		iterCtx *IterationContext,
		content string,
		toolCalls []message.ToolCall,
	) (injected []message.Message, halt bool, err error)
}

// Middleware provides hooks at loop boundaries.
type Middleware interface {
	BeforeTurn(ctx context.Context, iterCtx *IterationContext)
	AfterExecution(ctx context.Context, iterCtx *IterationContext, resp *Response)
}

// ContextCompactor applies staged context compaction to messages.
// Returns the compacted messages, whether compaction occurred, and any error.
type ContextCompactor interface {
	Compact(
		ctx context.Context,
		messages []message.Message,
	) ([]message.Message, bool, error)
}

// ReminderInjector evaluates conversation state and returns reminder messages.
// The turn parameter is the current loop iteration (zero-based).
type ReminderInjector interface {
	InjectReminders(
		ctx context.Context,
		messages []message.Message,
		turn int,
	) []message.Message
}

// ObservationSummarizer applies compact summaries to tool result messages.
type ObservationSummarizer interface {
	SummarizeToolResults(messages []message.Message) []message.Message
}
