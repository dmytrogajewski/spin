package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToolResult_Success(t *testing.T) {
	result := NewSuccessResult("output data")

	if result.Success == nil {
		t.Error("Success should not be nil")
	}
	if result.Error != nil {
		t.Error("Error should be nil")
	}
	if result.Success.Output != "output data" {
		t.Errorf("Expected output 'output data', got '%s'", result.Success.Output)
	}
}

func TestToolResult_Error(t *testing.T) {
	result := NewErrorResult("error message")

	if result.Error == nil {
		t.Error("Error should not be nil")
	}
	if result.Success != nil {
		t.Error("Success should be nil")
	}
	if result.Error.Message != "error message" {
		t.Errorf("Expected message 'error message', got '%s'", result.Error.Message)
	}
}


func TestParseMessage_UnknownType(t *testing.T) {
	msg := Message{
		Type: "unknown_type",
		Data: json.RawMessage(`{}`),
	}

	_, err := ParseMessage(msg)
	if err == nil {
		t.Error("Expected error for unknown message type")
	}
}

func TestStatusLevel_Constants(t *testing.T) {
	levels := []StatusLevel{
		StatusLevelInfo,
		StatusLevelWarning,
		StatusLevelError,
	}

	for _, level := range levels {
		if level == "" {
			t.Errorf("StatusLevel should not be empty")
		}
	}
}

func TestUserMessage_JSON(t *testing.T) {
	msg := UserMessage{
		Content:   "Test message",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded UserMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Content != msg.Content {
		t.Errorf("Expected content '%s', got '%s'", msg.Content, decoded.Content)
	}
}

func TestToolApproval_JSON(t *testing.T) {
	args := json.RawMessage(`{"modified":"args"}`)
	approval := ToolApproval{
		ToolCallID:   "call-123",
		Approved:     true,
		ModifiedArgs: &args,
	}

	data, err := json.Marshal(approval)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ToolApproval
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ToolCallID != approval.ToolCallID {
		t.Errorf("Expected tool_call_id '%s', got '%s'", approval.ToolCallID, decoded.ToolCallID)
	}
	if decoded.Approved != approval.Approved {
		t.Errorf("Expected approved %v, got %v", approval.Approved, decoded.Approved)
	}
}

