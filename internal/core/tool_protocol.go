package core

// tool_protocol.go defines the protocol for tool calls and results.
//
// This file co-locates all tool-related types that were previously scattered
// across message.go and agent.go, providing a clear definition of the tool
// invocation protocol used between the agent and LLM.

// ToolCall represents a tool invocation in an assistant message.
//
// This structure follows the OpenAI Chat API format for tool calls.
type ToolCall struct {
	// ID is a unique identifier for this tool call
	ID string `json:"id"`

	// Type is the tool call type, typically "function"
	Type string `json:"type"`

	// Function contains the function call details
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the details of a function call.
type ToolCallFunction struct {
	// Name is the function name to call
	Name string `json:"name"`

	// Arguments is a JSON string containing the function arguments
	Arguments string `json:"arguments"`
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	// ID matches the ToolCall.ID
	ID string

	// Success indicates if execution was successful
	Success bool

	// Output is the tool output
	Output string

	// Error is any error that occurred
	Error error

	// ExitCode is the exit code (for commands)
	ExitCode int
}
