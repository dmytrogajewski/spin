package debug

import (
	"bytes"
	"context"
	"encoding/json"
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
	// This test requires a mock core manager
	// For now, we'll test the basic structure
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	logger := NewEventLogger("text", nil)

	// Test with invalid prompt should return error
	err := logger.Run(ctx, "")
	if err == nil {
		t.Error("expected error with empty prompt")
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
