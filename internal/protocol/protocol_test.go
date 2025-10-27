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

func TestParsedMessageInterface(t *testing.T) {
	// Verify all message types implement ParsedMessage
	var _ ParsedMessage = TurnStart{}
	var _ ParsedMessage = AssistantDelta{}
	var _ ParsedMessage = ToolCallProposed{}
	var _ ParsedMessage = ToolCallExecuting{}
	var _ ParsedMessage = ToolCallResult{}
	var _ ParsedMessage = TurnComplete{}
	var _ ParsedMessage = StatusUpdate{}
}

func TestParseMessage_AllTypes(t *testing.T) {
	tests := []struct {
		name        string
		message     Message
		wantType    string
		checkResult func(t *testing.T, parsed ParsedMessage)
	}{
		{
			name: "TurnStart",
			message: Message{
				Type: "turn_start",
				Data: json.RawMessage(`{"turn_id":"turn-123","user_message":"Hello"}`),
			},
			wantType: "TurnStart",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				ts, ok := parsed.(TurnStart)
				if !ok {
					t.Fatalf("Expected TurnStart, got %T", parsed)
				}
				if ts.TurnID != "turn-123" {
					t.Errorf("Expected turn_id 'turn-123', got '%s'", ts.TurnID)
				}
				if ts.UserMessage != "Hello" {
					t.Errorf("Expected user_message 'Hello', got '%s'", ts.UserMessage)
				}
			},
		},
		{
			name: "AssistantDelta",
			message: Message{
				Type: "assistant_delta",
				Data: json.RawMessage(`{"delta":"Hello "}`),
			},
			wantType: "AssistantDelta",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				ad, ok := parsed.(AssistantDelta)
				if !ok {
					t.Fatalf("Expected AssistantDelta, got %T", parsed)
				}
				if ad.Delta != "Hello " {
					t.Errorf("Expected delta 'Hello ', got '%s'", ad.Delta)
				}
			},
		},
		{
			name: "ToolCallProposed",
			message: Message{
				Type: "tool_call_proposed",
				Data: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"bash","arguments":{},"requires_approval":true}`),
			},
			wantType: "ToolCallProposed",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				tcp, ok := parsed.(ToolCallProposed)
				if !ok {
					t.Fatalf("Expected ToolCallProposed, got %T", parsed)
				}
				if tcp.ToolCallID != "call-1" {
					t.Errorf("Expected tool_call_id 'call-1', got '%s'", tcp.ToolCallID)
				}
				if tcp.ToolName != "bash" {
					t.Errorf("Expected tool_name 'bash', got '%s'", tcp.ToolName)
				}
				if !tcp.RequiresApproval {
					t.Error("Expected requires_approval true")
				}
			},
		},
		{
			name: "ToolCallExecuting",
			message: Message{
				Type: "tool_call_executing",
				Data: json.RawMessage(`{"tool_call_id":"call-2"}`),
			},
			wantType: "ToolCallExecuting",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				tce, ok := parsed.(ToolCallExecuting)
				if !ok {
					t.Fatalf("Expected ToolCallExecuting, got %T", parsed)
				}
				if tce.ToolCallID != "call-2" {
					t.Errorf("Expected tool_call_id 'call-2', got '%s'", tce.ToolCallID)
				}
			},
		},
		{
			name: "ToolCallResult_Success",
			message: Message{
				Type: "tool_call_result",
				Data: json.RawMessage(`{"tool_call_id":"call-3","result":{"success":{"output":"Done"}}}`),
			},
			wantType: "ToolCallResult",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				tcr, ok := parsed.(ToolCallResult)
				if !ok {
					t.Fatalf("Expected ToolCallResult, got %T", parsed)
				}
				if tcr.ToolCallID != "call-3" {
					t.Errorf("Expected tool_call_id 'call-3', got '%s'", tcr.ToolCallID)
				}
				if tcr.Result.Success == nil {
					t.Fatal("Expected success result")
				}
				if tcr.Result.Success.Output != "Done" {
					t.Errorf("Expected output 'Done', got '%s'", tcr.Result.Success.Output)
				}
			},
		},
		{
			name: "ToolCallResult_Error",
			message: Message{
				Type: "tool_call_result",
				Data: json.RawMessage(`{"tool_call_id":"call-4","result":{"error":{"message":"Failed"}}}`),
			},
			wantType: "ToolCallResult",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				tcr, ok := parsed.(ToolCallResult)
				if !ok {
					t.Fatalf("Expected ToolCallResult, got %T", parsed)
				}
				if tcr.Result.Error == nil {
					t.Fatal("Expected error result")
				}
				if tcr.Result.Error.Message != "Failed" {
					t.Errorf("Expected message 'Failed', got '%s'", tcr.Result.Error.Message)
				}
			},
		},
		{
			name: "TurnComplete",
			message: Message{
				Type: "turn_complete",
				Data: json.RawMessage(`{"turn_id":"turn-456","final_message":"Completed"}`),
			},
			wantType: "TurnComplete",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				tc, ok := parsed.(TurnComplete)
				if !ok {
					t.Fatalf("Expected TurnComplete, got %T", parsed)
				}
				if tc.TurnID != "turn-456" {
					t.Errorf("Expected turn_id 'turn-456', got '%s'", tc.TurnID)
				}
				if tc.FinalMessage != "Completed" {
					t.Errorf("Expected final_message 'Completed', got '%s'", tc.FinalMessage)
				}
			},
		},
		{
			name: "StatusUpdate",
			message: Message{
				Type: "status_update",
				Data: json.RawMessage(`{"message":"Processing","level":"info"}`),
			},
			wantType: "StatusUpdate",
			checkResult: func(t *testing.T, parsed ParsedMessage) {
				su, ok := parsed.(StatusUpdate)
				if !ok {
					t.Fatalf("Expected StatusUpdate, got %T", parsed)
				}
				if su.Message != "Processing" {
					t.Errorf("Expected message 'Processing', got '%s'", su.Message)
				}
				if su.Level != StatusLevelInfo {
					t.Errorf("Expected level 'info', got '%s'", su.Level)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseMessage(tt.message)
			if err != nil {
				t.Fatalf("ParseMessage() error = %v", err)
			}
			tt.checkResult(t, parsed)
		})
	}
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	msg := Message{
		Type: "turn_start",
		Data: json.RawMessage(`{invalid json}`),
	}

	_, err := ParseMessage(msg)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestMessageConstructors(t *testing.T) {
	t.Run("NewAssistantDeltaMessage", func(t *testing.T) {
		delta := AssistantDelta{Delta: "test"}
		msg := NewAssistantDeltaMessage(delta)
		if msg.Type != "assistant_delta" {
			t.Errorf("Expected type 'assistant_delta', got '%s'", msg.Type)
		}
		parsed, err := ParseMessage(msg)
		if err != nil {
			t.Fatalf("ParseMessage error: %v", err)
		}
		ad, ok := parsed.(AssistantDelta)
		if !ok {
			t.Fatalf("Expected AssistantDelta, got %T", parsed)
		}
		if ad.Delta != "test" {
			t.Errorf("Expected delta 'test', got '%s'", ad.Delta)
		}
	})

	t.Run("NewToolCallProposedMessage", func(t *testing.T) {
		tcp := ToolCallProposed{
			ToolCallID:       "call-1",
			ToolName:         "bash",
			Arguments:        json.RawMessage(`{}`),
			RequiresApproval: true,
		}
		msg := NewToolCallProposedMessage(tcp)
		if msg.Type != "tool_call_proposed" {
			t.Errorf("Expected type 'tool_call_proposed', got '%s'", msg.Type)
		}
	})

	t.Run("NewToolCallExecutingMessage", func(t *testing.T) {
		tce := ToolCallExecuting{ToolCallID: "call-2"}
		msg := NewToolCallExecutingMessage(tce)
		if msg.Type != "tool_call_executing" {
			t.Errorf("Expected type 'tool_call_executing', got '%s'", msg.Type)
		}
	})

	t.Run("NewToolCallResultMessage", func(t *testing.T) {
		tcr := ToolCallResult{
			ToolCallID: "call-3",
			Result:     NewSuccessResult("output"),
		}
		msg := NewToolCallResultMessage(tcr)
		if msg.Type != "tool_call_result" {
			t.Errorf("Expected type 'tool_call_result', got '%s'", msg.Type)
		}
	})

	t.Run("NewStatusUpdateMessage", func(t *testing.T) {
		su := StatusUpdate{Message: "status", Level: StatusLevelInfo}
		msg := NewStatusUpdateMessage(su)
		if msg.Type != "status_update" {
			t.Errorf("Expected type 'status_update', got '%s'", msg.Type)
		}
	})
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
