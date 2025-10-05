package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestRequestID_String(t *testing.T) {
	strID := StringID("req-123")
	if strID.String() != "req-123" {
		t.Errorf("Expected 'req-123', got '%s'", strID.String())
	}

	numID := NumberID(42)
	if numID.String() != "42" {
		t.Errorf("Expected '42', got '%s'", numID.String())
	}
}

func TestRequestID_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		id       RequestID
		expected string
	}{
		{
			name:     "String ID",
			id:       StringID("req-123"),
			expected: `"req-123"`,
		},
		{
			name:     "Number ID",
			id:       NumberID(42),
			expected: `42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestRequestID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RequestID
	}{
		{
			name:     "String ID",
			input:    `"req-123"`,
			expected: StringID("req-123"),
		},
		{
			name:     "Number ID",
			input:    `42`,
			expected: NumberID(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id RequestID
			if err := json.Unmarshal([]byte(tt.input), &id); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if id.String() != tt.expected.String() {
				t.Errorf("Expected %s, got %s", tt.expected.String(), id.String())
			}
		})
	}
}

func TestRequestID_UnmarshalJSON_Invalid(t *testing.T) {
	var id RequestID
	err := json.Unmarshal([]byte(`true`), &id)
	if err == nil {
		t.Error("Expected error for invalid request ID")
	}
}

func TestRequest_Marshal(t *testing.T) {
	reqID := StringID("1")
	req := Request{
		JSONRPC: "2.0",
		ID:      &reqID,
		Method:  "test_method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got '%s'", decoded.JSONRPC)
	}
	if decoded.Method != "test_method" {
		t.Errorf("Expected method 'test_method', got '%s'", decoded.Method)
	}
}

func TestResponse_Success(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      StringID("1"),
		Result:  json.RawMessage(`{"status":"ok"}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Error != nil {
		t.Error("Error should be nil for success response")
	}
}

func TestResponse_Error(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      StringID("1"),
		Error:   NewError(MethodNotFound, "method not found"),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Error == nil {
		t.Error("Error should not be nil")
	}
	if decoded.Error.Code != MethodNotFound {
		t.Errorf("Expected code %d, got %d", MethodNotFound, decoded.Error.Code)
	}
}

func TestNewError(t *testing.T) {
	err := NewError(InvalidParams, "invalid parameters")

	if err.Code != InvalidParams {
		t.Errorf("Expected code %d, got %d", InvalidParams, err.Code)
	}
	if err.Message != "invalid parameters" {
		t.Errorf("Expected message 'invalid parameters', got '%s'", err.Message)
	}
	if err.Data != nil {
		t.Error("Data should be nil")
	}
}

func TestNewErrorWithData(t *testing.T) {
	data := map[string]string{"detail": "missing field"}
	err := NewErrorWithData(InvalidParams, "invalid parameters", data)

	if err.Code != InvalidParams {
		t.Errorf("Expected code %d, got %d", InvalidParams, err.Code)
	}
	if err.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestNotification_Marshal(t *testing.T) {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  "notification_method",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Notification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Method != "notification_method" {
		t.Errorf("Expected method 'notification_method', got '%s'", decoded.Method)
	}
}

func TestErrorCodes(t *testing.T) {
	codes := map[string]int{
		"ParseError":     ParseError,
		"InvalidRequest": InvalidRequest,
		"MethodNotFound": MethodNotFound,
		"InvalidParams":  InvalidParams,
		"InternalError":  InternalError,
	}

	for name, code := range codes {
		if code == 0 {
			t.Errorf("%s should not be 0", name)
		}
	}
}

func TestInitializeParams(t *testing.T) {
	params := InitializeParams{
		WorkspacePath: "/path/to/workspace",
		Config:        map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.WorkspacePath != params.WorkspacePath {
		t.Errorf("WorkspacePath mismatch")
	}
}

func TestSendMessageParams(t *testing.T) {
	convID := "conv-123"
	params := SendMessageParams{
		ConversationID: &convID,
		Message:        "Hello",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SendMessageParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ConversationID == nil || *decoded.ConversationID != convID {
		t.Error("ConversationID mismatch")
	}
	if decoded.Message != params.Message {
		t.Error("Message mismatch")
	}
}

func TestFileMatch(t *testing.T) {
	match := FileMatch{
		Path:  "src/main.go",
		Score: 0.95,
	}

	data, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded FileMatch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Path != match.Path {
		t.Error("Path mismatch")
	}
	if decoded.Score != match.Score {
		t.Error("Score mismatch")
	}
}
