package jsonrpc

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }
func numPtr(n int64) *int64   { return &n }

func TestRequestID_String(t *testing.T) {
	strID := RequestID{Str: strPtr("req-123")}
	if strID.String() != "req-123" {
		t.Errorf("Expected 'req-123', got '%s'", strID.String())
	}

	numID := RequestID{Num: numPtr(42)}
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
			id:       RequestID{Str: strPtr("req-123")},
			expected: `"req-123"`,
		},
		{
			name:     "Number ID",
			id:       RequestID{Num: numPtr(42)},
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
			expected: RequestID{Str: strPtr("req-123")},
		},
		{
			name:     "Number ID",
			input:    `42`,
			expected: RequestID{Num: numPtr(42)},
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
	reqID := RequestID{Str: strPtr("1")}
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
		ID:      RequestID{Str: strPtr("1")},
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
		ID:      RequestID{Str: strPtr("1")},
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
		Config:        json.RawMessage(`{"key":"value"}`),
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

	if string(decoded.Config) != `{"key":"value"}` {
		t.Errorf("Config mismatch: got %s", string(decoded.Config))
	}
}

func TestInitializeParams_ParseConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    json.RawMessage
		expectErr bool
		validate  func(t *testing.T, target map[string]string)
	}{
		{
			name:      "Valid config",
			config:    json.RawMessage(`{"key":"value","foo":"bar"}`),
			expectErr: false,
			validate: func(t *testing.T, target map[string]string) {
				if target["key"] != "value" {
					t.Errorf("Expected key=value, got %s", target["key"])
				}
				if target["foo"] != "bar" {
					t.Errorf("Expected foo=bar, got %s", target["foo"])
				}
			},
		},
		{
			name:      "Empty config",
			config:    json.RawMessage{},
			expectErr: false,
			validate: func(t *testing.T, target map[string]string) {
				if len(target) != 0 {
					t.Errorf("Expected empty target, got %v", target)
				}
			},
		},
		{
			name:      "Nil config",
			config:    nil,
			expectErr: false,
			validate: func(t *testing.T, target map[string]string) {
				if len(target) != 0 {
					t.Errorf("Expected empty target, got %v", target)
				}
			},
		},
		{
			name:      "Invalid JSON",
			config:    json.RawMessage(`{invalid}`),
			expectErr: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := InitializeParams{
				WorkspacePath: "/workspace",
				Config:        tt.config,
			}

			var target map[string]string
			err := params.ParseConfig(&target)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, target)
				}
			}
		})
	}
}

func TestInitializeParams_ParseConfig_ComplexStruct(t *testing.T) {
	type ComplexConfig struct {
		Name    string            `json:"name"`
		Enabled bool              `json:"enabled"`
		Count   int               `json:"count"`
		Meta    map[string]string `json:"meta"`
	}

	params := InitializeParams{
		WorkspacePath: "/workspace",
		Config:        json.RawMessage(`{"name":"test","enabled":true,"count":42,"meta":{"a":"1"}}`),
	}

	var target ComplexConfig
	err := params.ParseConfig(&target)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if target.Name != "test" {
		t.Errorf("Expected name=test, got %s", target.Name)
	}
	if !target.Enabled {
		t.Error("Expected enabled=true")
	}
	if target.Count != 42 {
		t.Errorf("Expected count=42, got %d", target.Count)
	}
	if target.Meta["a"] != "1" {
		t.Errorf("Expected meta.a=1, got %s", target.Meta["a"])
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

// Task Mode Tests

func TestSendMessageParams_WithTaskMode(t *testing.T) {
	mode := "review"
	params := SendMessageParams{
		Message:  "Hello",
		TaskMode: &mode,
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SendMessageParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TaskMode == nil {
		t.Fatal("TaskMode should not be nil")
	}
	if *decoded.TaskMode != "review" {
		t.Errorf("Expected task mode 'review', got '%s'", *decoded.TaskMode)
	}
}

func TestSendMessageParams_WithoutTaskMode(t *testing.T) {
	params := SendMessageParams{
		Message: "Hello",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SendMessageParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TaskMode != nil {
		t.Errorf("TaskMode should be nil, got %v", *decoded.TaskMode)
	}
}

func TestSendMessageResult_WithTaskMode(t *testing.T) {
	result := SendMessageResult{
		ConversationID: "conv-123",
		TurnID:         "turn-456",
		TaskMode:       "compact",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SendMessageResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TaskMode != "compact" {
		t.Errorf("Expected task mode 'compact', got '%s'", decoded.TaskMode)
	}
}

func TestValidateTaskMode_Valid(t *testing.T) {
	tests := []string{"regular", "review", "compact", "planning", ""}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			err := ValidateTaskMode(mode)
			if err != nil {
				t.Errorf("ValidateTaskMode(%q) returned error: %v", mode, err)
			}
		})
	}
}

func TestValidateTaskMode_Invalid(t *testing.T) {
	tests := []string{"invalid", "REGULAR", "Review", "compact-mode", "plan"}

	for _, mode := range tests {
		t.Run(mode, func(t *testing.T) {
			err := ValidateTaskMode(mode)
			if err == nil {
				t.Errorf("ValidateTaskMode(%q) should return error", mode)
			}
		})
	}
}

func TestSendMessageParams_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantMode *string
	}{
		{
			name:     "with task_mode",
			json:     `{"message":"Hello","task_mode":"review"}`,
			wantMode: strPtr("review"),
		},
		{
			name:     "without task_mode",
			json:     `{"message":"Hello"}`,
			wantMode: nil,
		},
		{
			name:     "task_mode null",
			json:     `{"message":"Hello","task_mode":null}`,
			wantMode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params SendMessageParams
			err := json.Unmarshal([]byte(tt.json), &params)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if tt.wantMode == nil {
				if params.TaskMode != nil {
					t.Errorf("Expected TaskMode to be nil, got %v", *params.TaskMode)
				}
			} else {
				if params.TaskMode == nil {
					t.Fatal("TaskMode should not be nil")
				}
				if *params.TaskMode != *tt.wantMode {
					t.Errorf("Expected task mode %q, got %q", *tt.wantMode, *params.TaskMode)
				}
			}
		})
	}
}
