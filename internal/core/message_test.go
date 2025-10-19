package core

import (
	"testing"
	"time"
)

func TestMessage_Structure(t *testing.T) {
	now := time.Now()
	message := Message{
		ID:      "msg_123",
		Role:    RoleUser,
		Content: "Hello, world!",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "test_tool",
					Arguments: `{"param": "value"}`,
				},
			},
		},
		ToolCallID: "call_1",
		Timestamp:  now,
		Tokens:     15,
		Name:       "test_user",
		Metadata: map[string]interface{}{
			"source": "test",
			"count":  42,
		},
	}

	if message.ID != "msg_123" {
		t.Errorf("Message.ID = %v, want %v", message.ID, "msg_123")
	}

	if message.Role != RoleUser {
		t.Errorf("Message.Role = %v, want %v", message.Role, RoleUser)
	}

	if message.Content != "Hello, world!" {
		t.Errorf("Message.Content = %v, want %v", message.Content, "Hello, world!")
	}

	if len(message.ToolCalls) != 1 {
		t.Errorf("len(Message.ToolCalls) = %v, want %v", len(message.ToolCalls), 1)
	}

	if message.ToolCallID != "call_1" {
		t.Errorf("Message.ToolCallID = %v, want %v", message.ToolCallID, "call_1")
	}

	if !message.Timestamp.Equal(now) {
		t.Errorf("Message.Timestamp = %v, want %v", message.Timestamp, now)
	}

	if message.Tokens != 15 {
		t.Errorf("Message.Tokens = %v, want %v", message.Tokens, 15)
	}

	if message.Name != "test_user" {
		t.Errorf("Message.Name = %v, want %v", message.Name, "test_user")
	}

	if message.Metadata["source"] != "test" {
		t.Errorf("Message.Metadata[\"source\"] = %v, want %v", message.Metadata["source"], "test")
	}

	if message.Metadata["count"] != 42 {
		t.Errorf("Message.Metadata[\"count\"] = %v, want %v", message.Metadata["count"], 42)
	}
}

func TestMessage_Roles(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want string
	}{
		{"system role", RoleSystem, "system"},
		{"user role", RoleUser, "user"},
		{"assistant role", RoleAssistant, "assistant"},
		{"tool role", RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := Message{Role: tt.role}
			if got := message.GetRole(); got != tt.want {
				t.Errorf("Message.GetRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_GetContent(t *testing.T) {
	message := Message{Content: "Test content"}
	if got := message.GetContent(); got != "Test content" {
		t.Errorf("Message.GetContent() = %v, want %v", got, "Test content")
	}
}

func TestMessage_GetTimestamp(t *testing.T) {
	now := time.Now()
	message := Message{Timestamp: now}
	if got := message.GetTimestamp(); !got.Equal(now) {
		t.Errorf("Message.GetTimestamp() = %v, want %v", got, now)
	}
}

func TestMessage_EmptyValues(t *testing.T) {
	message := Message{}

	if message.ID != "" {
		t.Errorf("Message.ID = %v, want %v", message.ID, "")
	}

	if message.Role != "" {
		t.Errorf("Message.Role = %v, want %v", message.Role, "")
	}

	if message.Content != "" {
		t.Errorf("Message.Content = %v, want %v", message.Content, "")
	}

	if len(message.ToolCalls) != 0 {
		t.Errorf("len(Message.ToolCalls) = %v, want %v", len(message.ToolCalls), 0)
	}

	if message.ToolCallID != "" {
		t.Errorf("Message.ToolCallID = %v, want %v", message.ToolCallID, "")
	}

	if !message.Timestamp.IsZero() {
		t.Errorf("Message.Timestamp = %v, want zero time", message.Timestamp)
	}

	if message.Tokens != 0 {
		t.Errorf("Message.Tokens = %v, want %v", message.Tokens, 0)
	}

	if message.Name != "" {
		t.Errorf("Message.Name = %v, want %v", message.Name, "")
	}

	if message.Metadata != nil {
		t.Errorf("Message.Metadata = %v, want %v", message.Metadata, nil)
	}
}

func TestMessage_MultipleToolCalls(t *testing.T) {
	message := Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "tool1",
					Arguments: `{"arg1": "value1"}`,
				},
			},
			{
				ID:   "call_2",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "tool2",
					Arguments: `{"arg2": "value2"}`,
				},
			},
		},
	}

	if len(message.ToolCalls) != 2 {
		t.Errorf("len(Message.ToolCalls) = %v, want %v", len(message.ToolCalls), 2)
	}

	if message.ToolCalls[0].ID != "call_1" {
		t.Errorf("Message.ToolCalls[0].ID = %v, want %v", message.ToolCalls[0].ID, "call_1")
	}

	if message.ToolCalls[1].ID != "call_2" {
		t.Errorf("Message.ToolCalls[1].ID = %v, want %v", message.ToolCalls[1].ID, "call_2")
	}
}

func TestMessage_ToolRole(t *testing.T) {
	message := Message{
		Role:       RoleTool,
		Content:    "Tool execution result",
		ToolCallID: "call_123",
	}

	if message.Role != RoleTool {
		t.Errorf("Message.Role = %v, want %v", message.Role, RoleTool)
	}

	if message.Content != "Tool execution result" {
		t.Errorf("Message.Content = %v, want %v", message.Content, "Tool execution result")
	}

	if message.ToolCallID != "call_123" {
		t.Errorf("Message.ToolCallID = %v, want %v", message.ToolCallID, "call_123")
	}
}

func TestMessage_Metadata(t *testing.T) {
	message := Message{
		Metadata: map[string]interface{}{
			"string_value": "test",
			"int_value":    123,
			"bool_value":   true,
			"float_value":  3.14,
		},
	}

	if message.Metadata["string_value"] != "test" {
		t.Errorf("Message.Metadata[\"string_value\"] = %v, want %v", message.Metadata["string_value"], "test")
	}

	if message.Metadata["int_value"] != 123 {
		t.Errorf("Message.Metadata[\"int_value\"] = %v, want %v", message.Metadata["int_value"], 123)
	}

	if message.Metadata["bool_value"] != true {
		t.Errorf("Message.Metadata[\"bool_value\"] = %v, want %v", message.Metadata["bool_value"], true)
	}

	if message.Metadata["float_value"] != 3.14 {
		t.Errorf("Message.Metadata[\"float_value\"] = %v, want %v", message.Metadata["float_value"], 3.14)
	}
}

func TestMessage_ConversationFlow(t *testing.T) {
	now := time.Now()

	// System message
	systemMsg := Message{
		ID:        "sys_1",
		Role:      RoleSystem,
		Content:   "You are a helpful assistant.",
		Timestamp: now,
		Tokens:    8,
	}

	// User message
	userMsg := Message{
		ID:        "user_1",
		Role:      RoleUser,
		Content:   "Hello, can you help me?",
		Timestamp: now.Add(time.Second),
		Tokens:    7,
	}

	// Assistant message with tool call
	assistantMsg := Message{
		ID:      "assistant_1",
		Role:    RoleAssistant,
		Content: "I'll help you with that.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "get_info",
					Arguments: `{"query": "help"}`,
				},
			},
		},
		Timestamp: now.Add(2 * time.Second),
		Tokens:    12,
	}

	// Tool result message
	toolMsg := Message{
		ID:         "tool_1",
		Role:       RoleTool,
		Content:    "Here's the information you requested.",
		ToolCallID: "call_1",
		Timestamp:  now.Add(3 * time.Second),
		Tokens:     8,
	}

	// Verify the conversation flow
	messages := []Message{systemMsg, userMsg, assistantMsg, toolMsg}

	if len(messages) != 4 {
		t.Errorf("len(messages) = %v, want %v", len(messages), 4)
	}

	// Check message order and relationships
	if messages[0].Role != RoleSystem {
		t.Errorf("messages[0].Role = %v, want %v", messages[0].Role, RoleSystem)
	}

	if messages[1].Role != RoleUser {
		t.Errorf("messages[1].Role = %v, want %v", messages[1].Role, RoleUser)
	}

	if messages[2].Role != RoleAssistant {
		t.Errorf("messages[2].Role = %v, want %v", messages[2].Role, RoleAssistant)
	}

	if messages[3].Role != RoleTool {
		t.Errorf("messages[3].Role = %v, want %v", messages[3].Role, RoleTool)
	}

	// Check tool call relationship
	if messages[2].ToolCalls[0].ID != messages[3].ToolCallID {
		t.Errorf("Tool call ID mismatch: %v != %v", messages[2].ToolCalls[0].ID, messages[3].ToolCallID)
	}
}
