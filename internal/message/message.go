package message

import "time"

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
	ToolCalls []interface{} `json:"tool_calls,omitempty"`

	// ToolCallID links a tool role message to the corresponding tool call
	ToolCallID string `json:"tool_call_id,omitempty"`

	// Timestamp records when this message was created
	Timestamp time.Time `json:"timestamp"`

	// Tokens is the estimated token count for this message
	Tokens int `json:"tokens"`

	// Name is an optional name field for the message author
	Name string `json:"name,omitempty"`

	// Metadata stores additional extensible data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetRole returns the message role (implements cycle.Message interface)

// GetContent returns the message content (implements cycle.Message interface)

// GetTimestamp returns the message timestamp (implements cycle.Message interface)
