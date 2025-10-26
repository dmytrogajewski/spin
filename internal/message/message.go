package message

import "time"

// ToolCall represents a tool invocation from an assistant message.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains function invocation details.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Metadata stores string key-value metadata for messages.
type Metadata map[string]string

// Role represents a message role in the conversation.
type Role string

// Standard message roles compatible with LLM APIs.
const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a single message in the conversation history.
//
// Messages follow the format expected by LLM APIs like OpenAI's Chat API,
// supporting system messages, user input, assistant responses, and tool
// interaction results.
type Message struct {
	// ID is a unique identifier for this message
	ID string `json:"id"`

	// Role indicates who created this message
	Role Role `json:"role"`

	// Content is the text content of the message
	Content string `json:"content"`

	// ToolCalls contains tool invocations from assistant messages
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID links a tool role message to the corresponding tool call
	ToolCallID string `json:"tool_call_id,omitempty"`

	// Timestamp records when this message was created
	Timestamp time.Time `json:"timestamp"`

	// Tokens is the estimated token count for this message
	Tokens int `json:"tokens"`

	// Name is an optional name field for the message author
	Name string `json:"name,omitempty"`

	// Metadata stores additional string key-value data
	Metadata Metadata `json:"metadata,omitempty"`
}
