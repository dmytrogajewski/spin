package agent

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// AgentRequest represents a request to the agent.
type AgentRequest struct {
	// Input is the user's request
	Input string

	// Context is the environment context
	Context *Environment

	// Task is the task mode (required)
	// Use task.NewTask(name) to create task instances
	Task task.Task

	// Timeout for this request
	Timeout time.Duration

	// History contains previous conversation messages for context
	History []message.Message
}

// AgentResponse represents the agent's response.
type AgentResponse struct {
	// Output is the agent's response
	Output string

	// Success indicates if the request was successful
	Success bool

	// Error if any occurred
	Error error

	// ToolCalls are the tool calls made
	ToolCalls []ToolCall

	// FinishReason indicates why the conversation finished
	FinishReason string

	// Duration of the request
	Duration time.Duration

	// Messages contains all messages from this turn (excluding input history)
	// This includes: user input, assistant messages with tool calls, tool results, final assistant message
	// The conversation layer should persist these to maintain proper OpenAI message format
	Messages []message.Message

	// RetrievedBullets contains ACE bullets retrieved during execution
	// Populated from TrajectoryContext or simple retrieval mode
	RetrievedBullets []*bullet.Bullet

	// TrajectoryContext contains progressive execution context (for Reflector)
	TrajectoryContext *trajectory.TrajectoryContext
}

// EventType represents the category of event.
type EventType int

const (
	EventWarning EventType = iota
	EventTurnPaused
	EventToolCallStart
	EventToolCallComplete
)

// Event represents an event.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// ToolCallCompleteData represents tool call completion data.
type ToolCallCompleteData struct {
	ToolID   string
	ToolName string
	Success  bool
	Error    string
	Output   string
}

// ToolCallStartData represents tool call start data.
type ToolCallStartData struct {
	ToolID     string
	ToolName   string
	Parameters tools.ToolParameters
}

// Role constants
