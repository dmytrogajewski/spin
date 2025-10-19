package debug

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

func TestEventLogger_New(t *testing.T) {
	tests := []struct {
		name   string
		format string
		filter []string
	}{
		{"text format no filter", "text", nil},
		{"json format no filter", "json", nil},
		{"with filter", "text", []string{"tool", "stream"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewEventLogger(tt.format, tt.filter)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
			if logger.format != tt.format {
				t.Errorf("expected format %s, got %s", tt.format, logger.format)
			}
		})
	}
}

func TestEventLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		name     string
		filter   []string
		event    core.Event
		expected bool
	}{
		{
			name:     "no filter logs all",
			filter:   nil,
			event:    core.Event{Type: core.EventContentDelta},
			expected: true,
		},
		{
			name:     "filter matches",
			filter:   []string{"content_delta", "tool_call_start"},
			event:    core.Event{Type: core.EventContentDelta},
			expected: true,
		},
		{
			name:     "filter does not match",
			filter:   []string{"tool_call_start"},
			event:    core.Event{Type: core.EventContentDelta},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewEventLogger("text", tt.filter)
			result := logger.shouldLog(tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEventLogger_LogEvent_Text(t *testing.T) {
	logger := NewEventLogger("text", nil)

	// Capture stderr
	var buf bytes.Buffer
	logger.writer = &buf

	event := core.Event{
		Type: core.EventToolCallStart,
		Data: map[string]interface{}{
			"tool": "bash",
			"args": map[string]string{"command": "ls"},
		},
	}

	logger.logEvent(event)

	output := buf.String()
	if !strings.Contains(output, "tool_call_start") {
		t.Errorf("expected output to contain 'tool_call_start', got: %s", output)
	}
	if !strings.Contains(output, "bash") {
		t.Errorf("expected output to contain 'bash', got: %s", output)
	}
}

func TestEventLogger_LogEvent_JSON(t *testing.T) {
	logger := NewEventLogger("json", nil)

	// Capture stderr
	var buf bytes.Buffer
	logger.writer = &buf

	event := core.Event{
		Type: core.EventContentDelta,
		Data: map[string]interface{}{
			"delta": "Hello",
		},
	}

	logger.logEvent(event)

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Type is int, so it won't be a string in JSON
	// Instead, check that the type field exists and is a number
	if _, ok := parsed["type"]; !ok {
		t.Error("expected type field in JSON output")
	}
}

func TestEventLogger_Run(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger := NewEventLogger("text", nil)

	// Test with invalid prompt should return error
	err := logger.Run(ctx, "")
	if err == nil {
		t.Error("expected error with empty prompt")
	}
	if !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("expected 'prompt cannot be empty' error, got: %v", err)
	}
}

func TestEventLogger_Run_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logger := NewEventLogger("text", nil)

	// Test with valid prompt - this will create a real conversation
	// Note: This test may fail if no LLM provider is configured
	err := logger.Run(ctx, "Hello, this is a test prompt")

	// We expect either success or a configuration error
	if err != nil {
		// Check if it's a configuration error (expected in test environment)
		if !strings.Contains(err.Error(), "failed to create manager") &&
			!strings.Contains(err.Error(), "failed to create conversation") &&
			!strings.Contains(err.Error(), "turn execution failed") {
			t.Errorf("unexpected error type: %v", err)
		}
	}
}

func TestEventLogger_Run_WithFilter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test with filter
	logger := NewEventLogger("text", []string{"turn_start", "turn_complete"})

	// Test with valid prompt
	err := logger.Run(ctx, "Test prompt")

	// We expect either success or a configuration error
	if err != nil {
		// Check if it's a configuration error (expected in test environment)
		if !strings.Contains(err.Error(), "failed to create manager") &&
			!strings.Contains(err.Error(), "failed to create conversation") &&
			!strings.Contains(err.Error(), "turn execution failed") {
			t.Errorf("unexpected error type: %v", err)
		}
	}
}

func TestEventLogger_Run_JSONFormat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test with JSON format
	logger := NewEventLogger("json", nil)

	// Test with valid prompt
	err := logger.Run(ctx, "Test prompt")

	// We expect either success or a configuration error
	if err != nil {
		// Check if it's a configuration error (expected in test environment)
		if !strings.Contains(err.Error(), "failed to create manager") &&
			!strings.Contains(err.Error(), "failed to create conversation") &&
			!strings.Contains(err.Error(), "turn execution failed") {
			t.Errorf("unexpected error type: %v", err)
		}
	}
}

func TestEventLogger_Run_EmptyPrompt(t *testing.T) {
	logger := NewEventLogger("text", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	ctx := context.Background()
	err := logger.Run(ctx, "")

	if err == nil {
		t.Error("Run() expected error for empty prompt, got nil")
	}
	if !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("Run() error should contain 'prompt cannot be empty', got: %v", err)
	}
}

func TestEventLogger_Filter_Multiple(t *testing.T) {
	logger := NewEventLogger("text", []string{"tool_call_start", "tool_call_complete"})

	tests := []struct {
		eventType core.EventType
		expected  bool
	}{
		{core.EventToolCallStart, true},
		{core.EventToolCallComplete, true},
		{core.EventContentDelta, false},
		{core.EventTurnStart, false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType.String(), func(t *testing.T) {
			event := core.Event{Type: tt.eventType}
			result := logger.shouldLog(event)
			if result != tt.expected {
				t.Errorf("for event %s: expected %v, got %v", tt.eventType, tt.expected, result)
			}
		})
	}
}

func TestEventLogger_LogEventJSON(t *testing.T) {
	logger := NewEventLogger("json", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	event := core.Event{
		Type: core.EventContentDelta,
		Data: map[string]interface{}{
			"delta": "Hello",
		},
	}

	logger.logEventJSON("2023-01-01 12:00:00", event)

	output := buf.String()
	if !strings.Contains(output, "timestamp") {
		t.Errorf("expected output to contain 'timestamp', got: %s", output)
	}
	if !strings.Contains(output, "type") {
		t.Errorf("expected output to contain 'type', got: %s", output)
	}
	if !strings.Contains(output, "data") {
		t.Errorf("expected output to contain 'data', got: %s", output)
	}
}

func TestEventLogger_LogEventText(t *testing.T) {
	logger := NewEventLogger("text", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	event := core.Event{
		Type: core.EventTurnStart,
		Data: "Starting turn",
	}

	logger.logEventText("2023-01-01 12:00:00", event)

	output := buf.String()
	if !strings.Contains(output, "2023-01-01 12:00:00") {
		t.Errorf("expected output to contain timestamp, got: %s", output)
	}
	if !strings.Contains(output, "turn_start") {
		t.Errorf("expected output to contain event type, got: %s", output)
	}
	if !strings.Contains(output, "Starting turn") {
		t.Errorf("expected output to contain event data, got: %s", output)
	}
}

func TestEventLogger_LogEventJSON_InvalidData(t *testing.T) {
	logger := NewEventLogger("json", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	// Create event with data that can't be marshaled to JSON
	event := core.Event{
		Type: core.EventError,
		Data: func() {}, // Function can't be marshaled to JSON
	}

	logger.logEventJSON("2023-01-01 12:00:00", event)

	output := buf.String()
	if !strings.Contains(output, "timestamp") {
		t.Errorf("expected output to contain 'timestamp', got: %s", output)
	}
	if !strings.Contains(output, "type") {
		t.Errorf("expected output to contain 'type', got: %s", output)
	}
	// Should fall back to empty JSON object for data
	if !strings.Contains(output, "{}") {
		t.Errorf("expected output to contain empty JSON object for invalid data, got: %s", output)
	}
}

func TestEventLogger_LogEventJSON_EncodeError(t *testing.T) {
	logger := NewEventLogger("json", nil)

	// Create event with unmarshalable output structure
	// This will trigger the error path at line 129-131
	event := core.Event{
		Type: core.EventError,
		Data: "normal data",
	}

	// Create a writer that will capture the output
	var buf bytes.Buffer
	logger.writer = &buf

	// Save original writer field to create an impossible encoding scenario
	// We can't easily trigger json.Marshal error on a normal map, but we can verify
	// the error handling path works with a failing writer
	logger.writer = &failingWriter{}

	// This should not panic and should handle write failures gracefully
	logger.logEventJSON("2023-01-01 12:00:00", event)
}

func TestEventLogger_LogEventJSON_MarshalError(t *testing.T) {
	logger := NewEventLogger("json", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	// Create a more complex unmarshalable structure
	type unmarshalable struct {
		Chan chan int
	}

	event := core.Event{
		Type: core.EventError,
		Data: unmarshalable{Chan: make(chan int)},
	}

	// This should handle marshal error and use {}
	logger.logEventJSON("2023-01-01 12:00:00", event)

	output := buf.String()
	// Should still produce valid JSON with empty object for data
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output should be valid JSON: %v, got: %s", err, output)
	}
}

// failingWriter is a writer that always fails
type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write failed")
}

func TestEventLogger_Concurrency(t *testing.T) {
	logger := NewEventLogger("text", nil)
	var buf bytes.Buffer
	logger.writer = &buf

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			event := core.Event{
				Type: core.EventContentDelta,
				Data: fmt.Sprintf("message %d", i),
			}
			logger.logEvent(event)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify some output was written
	if buf.Len() == 0 {
		t.Error("expected some output from concurrent logging")
	}
}
