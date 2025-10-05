package ui

import (
	"fmt"
	"time"
)

// MessageRole represents who sent the message.
type MessageRole string

const (
	// RoleUser indicates a message from the user
	RoleUser MessageRole = "user"

	// RoleAssistant indicates a message from the AI assistant
	RoleAssistant MessageRole = "assistant"

	// RoleSystem indicates a system message
	RoleSystem MessageRole = "system"

	// RoleTool indicates a tool-related message
	RoleTool MessageRole = "tool"
)

// Message represents a single chat message in the conversation.
type Message struct {
	Role      MessageRole // Who sent the message
	Content   string      // Message content
	Timestamp time.Time   // When the message was created
	Streaming bool        // True if still streaming (for assistant messages)

	// Optional fields
	ToolCall   *ToolCall   // If this message includes a tool call
	ToolResult *ToolResult // If this message includes a tool result
	Reasoning  string      // Reasoning block (for compatible models)

	// Backtrack mode (Phase 3.8)
	Highlighted bool // True if this message is selected in backtrack mode

	// Error handling (Phase 3.12)
	IsError bool // True if this is an error message
}

// ToolCall represents a tool invocation by the AI.
type ToolCall struct {
	Name      string                 // Tool name (e.g., "shell", "read_file")
	Arguments map[string]interface{} // Tool arguments
	ID        string                 // Unique tool call ID
}

// Validate checks if the tool call is valid.
func (tc ToolCall) Validate() error {
	if tc.Name == "" {
		return fmt.Errorf("tool call missing name")
	}
	if tc.ID == "" {
		return fmt.Errorf("tool call missing ID")
	}
	return nil
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string // ID of the tool call this is a result for
	Output     string // Tool output (stdout)
	Error      string // Tool error (if any)
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   content,
		Timestamp: time.Now(),
		Streaming: false,
	}
}

// NewAssistantMessage creates a new assistant message.
func NewAssistantMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Streaming: false,
	}
}

// NewStreamingMessage creates a new streaming assistant message.
func NewStreamingMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
		Streaming: true,
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   content,
		Timestamp: time.Now(),
		Streaming: false,
	}
}

// NewToolMessage creates a new tool message.
func NewToolMessage(content string, toolCall *ToolCall, toolResult *ToolResult) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		Timestamp:  time.Now(),
		Streaming:  false,
		ToolCall:   toolCall,
		ToolResult: toolResult,
	}
}
