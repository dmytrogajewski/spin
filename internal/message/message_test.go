package message

import (
	"encoding/json"
	"testing"
)

// TestToolCall_Creation tests creating a ToolCall with all fields.
func TestToolCall_Creation(t *testing.T) {
	tc := ToolCall{
		ID:   "call_abc123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "read_file",
			Arguments: `{"path": "test.go"}`,
		},
	}

	if tc.ID != "call_abc123" {
		t.Errorf("ID = %q, want %q", tc.ID, "call_abc123")
	}

	if tc.Type != "function" {
		t.Errorf("Type = %q, want %q", tc.Type, "function")
	}

	if tc.Function.Name != "read_file" {
		t.Errorf("Function.Name = %q, want %q", tc.Function.Name, "read_file")
	}

	if tc.Function.Arguments != `{"path": "test.go"}` {
		t.Errorf("Function.Arguments = %q, want %q", tc.Function.Arguments, `{"path": "test.go"}`)
	}
}

// TestToolCall_JSONMarshaling tests JSON marshaling and unmarshaling.
func TestToolCall_JSONMarshaling(t *testing.T) {
	original := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "execute_command",
			Arguments: `{"command": "ls -la"}`,
		},
	}

	// Marshal to JSON.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal back.
	var decoded ToolCall

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify fields match.
	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}

	if decoded.Function.Name != original.Function.Name {
		t.Errorf("Function.Name = %q, want %q", decoded.Function.Name, original.Function.Name)
	}

	if decoded.Function.Arguments != original.Function.Arguments {
		t.Errorf("Function.Arguments = %q, want %q", decoded.Function.Arguments, original.Function.Arguments)
	}
}

// TestMetadata_StringValues tests that Metadata stores string key-value pairs.
func TestMetadata_StringValues(t *testing.T) {
	meta := Metadata{
		"user":      "test_user",
		"session":   "abc123",
		"timestamp": "2025-10-26",
	}

	if meta["user"] != "test_user" {
		t.Errorf("user = %q, want %q", meta["user"], "test_user")
	}

	if meta["session"] != "abc123" {
		t.Errorf("session = %q, want %q", meta["session"], "abc123")
	}

	if meta["timestamp"] != "2025-10-26" {
		t.Errorf("timestamp = %q, want %q", meta["timestamp"], "2025-10-26")
	}
}

// TestMessage_WithToolCalls tests Message with typed ToolCalls.
func TestMessage_WithToolCalls(t *testing.T) {
	msg := Message{
		ID:      "msg_1",
		Role:    RoleAssistant,
		Content: "I'll read the file for you.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "test.go"}`,
				},
			},
			{
				ID:   "call_2",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "execute_command",
					Arguments: `{"command": "ls"}`,
				},
			},
		},
	}

	// Verify core fields.
	if msg.ID != "msg_1" {
		t.Errorf("ID = %q, want %q", msg.ID, "msg_1")
	}

	if msg.Role != RoleAssistant {
		t.Errorf("Role = %q, want %q", msg.Role, RoleAssistant)
	}

	if msg.Content != "I'll read the file for you." {
		t.Errorf("Content = %q, want %q", msg.Content, "I'll read the file for you.")
	}

	// Verify ToolCalls are typed.
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("len(ToolCalls) = %d, want 2", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCalls[0].ID = %q, want %q", msg.ToolCalls[0].ID, "call_1")
	}

	if msg.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("ToolCalls[0].Function.Name = %q, want %q", msg.ToolCalls[0].Function.Name, "read_file")
	}

	if msg.ToolCalls[1].Function.Name != "execute_command" {
		t.Errorf("ToolCalls[1].Function.Name = %q, want %q", msg.ToolCalls[1].Function.Name, "execute_command")
	}
}

// TestMessage_WithMetadata tests Message with typed Metadata.
func TestMessage_WithMetadata(t *testing.T) {
	msg := Message{
		ID:      "msg_2",
		Role:    RoleUser,
		Content: "Hello",
		Metadata: Metadata{
			"source":    "cli",
			"user_id":   "user_123",
			"timestamp": "2025-10-26T10:00:00Z",
		},
	}

	// Verify core fields.
	if msg.ID != "msg_2" {
		t.Errorf("ID = %q, want %q", msg.ID, "msg_2")
	}

	if msg.Role != RoleUser {
		t.Errorf("Role = %q, want %q", msg.Role, RoleUser)
	}

	if msg.Content != "Hello" {
		t.Errorf("Content = %q, want %q", msg.Content, "Hello")
	}

	// Verify Metadata is typed as map[string]string.
	if msg.Metadata["source"] != "cli" {
		t.Errorf("Metadata[source] = %q, want %q", msg.Metadata["source"], "cli")
	}

	if msg.Metadata["user_id"] != "user_123" {
		t.Errorf("Metadata[user_id] = %q, want %q", msg.Metadata["user_id"], "user_123")
	}

	if msg.Metadata["timestamp"] != "2025-10-26T10:00:00Z" {
		t.Errorf("Metadata[timestamp] = %q, want %q", msg.Metadata["timestamp"], "2025-10-26T10:00:00Z")
	}
}

// TestMessage_JSONSerialization tests complete Message JSON marshaling/unmarshaling.
func TestMessage_JSONSerialization(t *testing.T) {
	original := Message{
		ID:      "msg_123",
		Role:    RoleAssistant,
		Content: "I'll help you with that.",
		ToolCalls: []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "main.go"}`,
				},
			},
		},
		Metadata: Metadata{
			"session": "sess_456",
			"user":    "test_user",
		},
	}

	// Marshal to JSON.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal back.
	var decoded Message

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify all fields.
	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}

	if decoded.Role != original.Role {
		t.Errorf("Role = %q, want %q", decoded.Role, original.Role)
	}

	if decoded.Content != original.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, original.Content)
	}

	if len(decoded.ToolCalls) != len(original.ToolCalls) {
		t.Fatalf("len(ToolCalls) = %d, want %d", len(decoded.ToolCalls), len(original.ToolCalls))
	}

	if decoded.ToolCalls[0].ID != original.ToolCalls[0].ID {
		t.Errorf("ToolCalls[0].ID = %q, want %q", decoded.ToolCalls[0].ID, original.ToolCalls[0].ID)
	}

	if decoded.Metadata["session"] != original.Metadata["session"] {
		t.Errorf("Metadata[session] = %q, want %q", decoded.Metadata["session"], original.Metadata["session"])
	}
}

// TestMessage_InterfaceMethods tests the cycle.Message interface methods.
