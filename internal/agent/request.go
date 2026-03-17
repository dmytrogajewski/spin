package agent

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ToolCall is an alias for message.ToolCall to avoid duplication.
type ToolCall = message.ToolCall

// ToolCallFunction is an alias for message.ToolCallFunction to avoid duplication.
type ToolCallFunction = message.ToolCallFunction

// ToolResult is an alias for tools.ToolResult to provide a unified type.
type ToolResult = tools.ToolResult

// Request represents a request to the agent.
type Request struct {
	// Input is the user's request.
	Input string

	// Context is the environment context.
	Context *Environment

	// CallParams holds pre-resolved system prompt and token budget.
	CallParams CallParams

	// Timeout for this request.
	Timeout time.Duration

	// History contains previous conversation messages for context.
	History []message.Message
}

// Response represents the agent's response.
type Response struct {
	// Output is the agent's response.
	Output string

	// Success indicates if the request was successful.
	Success bool

	// Error if any occurred.
	Error error

	// ToolCalls are the tool calls made.
	ToolCalls []ToolCall

	// FinishReason indicates why the conversation finished.
	FinishReason string

	// Duration of the request.
	Duration time.Duration

	// Messages contains all messages from this turn (excluding input history)
	// This includes: user input, assistant messages with tool calls, tool results, final assistant message
	// The conversation layer should persist these to maintain proper OpenAI message format.
	Messages []message.Message

	// RetrievedBullets contains ACE bullets retrieved during execution
	// Populated from TrajectoryContext or simple retrieval mode.
	RetrievedBullets []*bullet.Bullet

	// TrajectoryContext contains progressive execution context (for Reflector).
	TrajectoryContext *trajectory.Context
}

// AppendToolCall adds a completed tool call to the response.
// Implements tool.ResponseAccumulator.
func (r *Response) AppendToolCall(tc message.ToolCall) {
	r.ToolCalls = append(r.ToolCalls, tc)
}
