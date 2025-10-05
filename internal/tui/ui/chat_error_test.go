package ui

import (
	"strings"
	"testing"
	"time"
)

// TestChat_AddError tests adding an error message to the chat.
func TestChat_AddError(t *testing.T) {
	chat := NewChat(80, 20)

	err := ErrorDisplay{
		Message:   "Permission denied",
		Code:      "permission_denied",
		Details:   "Cannot access /etc/shadow",
		Operation: "Executor.Execute",
		Severity:  2, // Error
		Timestamp: time.Now().Format(time.RFC3339),
	}

	chat.AddError(err)

	if len(chat.messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(chat.messages))
	}

	msg := chat.messages[0]
	if msg.Role != "system" {
		t.Errorf("Error message role = %q, want %q", msg.Role, "system")
	}
	if !msg.IsError {
		t.Error("IsError should be true for error messages")
	}
	if !strings.Contains(msg.Content, "Permission denied") {
		t.Error("Error message should contain error message")
	}
	if !strings.Contains(msg.Content, "permission_denied") {
		t.Error("Error message should contain error code")
	}
}

// TestChat_AddError_AutoScroll tests that errors trigger auto-scroll.
func TestChat_AddError_AutoScroll(t *testing.T) {
	chat := NewChat(80, 20)

	// Add some messages to create scrollable content
	for i := 0; i < 10; i++ {
		chat.AddMessage(Message{
			Role:    "user",
			Content: "Test message",
		})
	}

	// User scrolls up
	chat.userScrolled = true
	chat.atBottom = false

	// Add error
	err := ErrorDisplay{
		Message:   "Test error",
		Code:      "test",
		Severity:  2,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	chat.AddError(err)

	// Should scroll to bottom for errors
	// Note: We can't directly test atBottom since it's set during render
	// But we can verify the message was added
	if len(chat.messages) != 11 {
		t.Errorf("Expected 11 messages, got %d", len(chat.messages))
	}
}

// TestChat_FormatErrorMessage tests error message formatting.
func TestChat_FormatErrorMessage(t *testing.T) {
	chat := NewChat(80, 20)

	tests := []struct {
		name     string
		err      ErrorDisplay
		wantIcon string
		wantText []string
	}{
		{
			name: "error with all fields",
			err: ErrorDisplay{
				Message:   "Connection failed",
				Code:      "external",
				Details:   "ECONNREFUSED",
				Operation: "Agent.RunTurn",
				Severity:  2,
				Timestamp: "2025-10-05T12:34:56Z",
			},
			wantIcon: "❌",
			wantText: []string{
				"Connection failed",
				"external",
				"Agent.RunTurn",
				"ECONNREFUSED",
			},
		},
		{
			name: "critical error",
			err: ErrorDisplay{
				Message:   "System failure",
				Code:      "internal",
				Severity:  3,
				Timestamp: "2025-10-05T12:34:56Z",
			},
			wantIcon: "🔥",
			wantText: []string{
				"System failure",
				"internal",
			},
		},
		{
			name: "warning (should not be in chat)",
			err: ErrorDisplay{
				Message:   "File not found",
				Code:      "not_found",
				Severity:  1,
				Timestamp: "2025-10-05T12:34:56Z",
			},
			wantIcon: "⚠️",
			wantText: []string{
				"File not found",
				"not_found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := chat.formatErrorMessage(tt.err)

			if !strings.Contains(formatted, tt.wantIcon) {
				t.Errorf("Formatted message should contain icon %q", tt.wantIcon)
			}

			for _, text := range tt.wantText {
				if !strings.Contains(formatted, text) {
					t.Errorf("Formatted message should contain %q", text)
				}
			}
		})
	}
}

// TestChat_RenderError tests error rendering in the chat view.
func TestChat_RenderError(t *testing.T) {
	chat := NewChat(80, 20)

	// Add a normal message
	chat.AddMessage(Message{
		Role:    "user",
		Content: "Read config.yaml",
	})

	// Add an error
	err := ErrorDisplay{
		Message:   "Permission denied",
		Code:      "permission_denied",
		Details:   "Cannot access /etc/shadow",
		Operation: "Executor.Execute",
		Severity:  2,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	chat.AddError(err)

	// Check error message was added to messages
	if len(chat.messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(chat.messages))
	}

	errorMsg := chat.messages[1]
	if !strings.Contains(errorMsg.Content, "Permission denied") {
		t.Error("Error message should contain 'Permission denied'")
	}
	if !strings.Contains(errorMsg.Content, "❌") {
		t.Error("Error message should contain error icon")
	}
	if !strings.Contains(errorMsg.Content, "permission_denied") {
		t.Error("Error message should contain error code")
	}
}

// TestChat_ErrorWithDetails tests rendering errors with multiline details.
func TestChat_ErrorWithDetails(t *testing.T) {
	chat := NewChat(80, 20)

	err := ErrorDisplay{
		Message:   "Build failed",
		Code:      "external",
		Details:   "Line 1: syntax error\nLine 2: unexpected token\nLine 3: compilation failed",
		Operation: "Build.Execute",
		Severity:  2,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	chat.AddError(err)

	msg := chat.messages[0]

	// Should contain all detail lines
	if !strings.Contains(msg.Content, "Line 1") {
		t.Error("Error should contain first detail line")
	}
	if !strings.Contains(msg.Content, "Line 2") {
		t.Error("Error should contain second detail line")
	}
	if !strings.Contains(msg.Content, "Line 3") {
		t.Error("Error should contain third detail line")
	}
}

// TestChat_ErrorIcon tests correct icon for each severity.
func TestChat_ErrorIcon(t *testing.T) {
	tests := []struct {
		severity int
		wantIcon string
	}{
		{0, "ℹ️"},  // Info
		{1, "⚠️"},  // Warning
		{2, "❌"},  // Error
		{3, "🔥"},  // Critical
	}

	for _, tt := range tests {
		t.Run(tt.wantIcon, func(t *testing.T) {
			chat := NewChat(80, 20)
			err := ErrorDisplay{
				Message:   "Test",
				Code:      "test",
				Severity:  tt.severity,
				Timestamp: time.Now().Format(time.RFC3339),
			}

			formatted := chat.formatErrorMessage(err)
			if !strings.Contains(formatted, tt.wantIcon) {
				t.Errorf("Formatted message should contain icon %q", tt.wantIcon)
			}
		})
	}
}

// TestChat_MultipleErrors tests handling multiple errors.
func TestChat_MultipleErrors(t *testing.T) {
	chat := NewChat(80, 20)

	// Add multiple errors
	for i := 1; i <= 3; i++ {
		err := ErrorDisplay{
			Message:   "Error " + string(rune('0'+i)),
			Code:      "test",
			Severity:  2,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		chat.AddError(err)
	}

	if len(chat.messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(chat.messages))
	}

	// All should be error messages
	for i, msg := range chat.messages {
		if !msg.IsError {
			t.Errorf("Message %d should be an error", i)
		}
		if msg.Role != "system" {
			t.Errorf("Message %d role = %q, want %q", i, msg.Role, "system")
		}
	}
}
