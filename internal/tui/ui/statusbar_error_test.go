package ui

import (
	"strings"
	"testing"
	"time"
)

// TestStatusBar_ShowError tests displaying an error in the status bar.
func TestStatusBar_ShowError(t *testing.T) {
	sb := NewStatusBar(120)

	err := ErrorDisplay{
		Message:     "File not found",
		Code:        "not_found",
		Severity:    1, // Warning
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)

	if sb.errorMsg == nil {
		t.Fatal("errorMsg should not be nil after ShowError")
	}
	if sb.errorMsg.Message != "File not found" {
		t.Errorf("errorMsg.Message = %q, want %q", sb.errorMsg.Message, "File not found")
	}
	if sb.errorShownAt.IsZero() {
		t.Error("errorShownAt should be set")
	}
}

// TestStatusBar_ClearError tests clearing an error from the status bar.
func TestStatusBar_ClearError(t *testing.T) {
	sb := NewStatusBar(120)

	err := ErrorDisplay{
		Message:     "Test error",
		Code:        "test",
		Severity:    1,
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)
	if sb.errorMsg == nil {
		t.Fatal("Setup failed: errorMsg should be set")
	}

	sb.ClearError()

	if sb.errorMsg != nil {
		t.Error("errorMsg should be nil after ClearError")
	}
}

// TestStatusBar_HasError tests checking if an error is present.
func TestStatusBar_HasError(t *testing.T) {
	sb := NewStatusBar(120)

	if sb.HasError() {
		t.Error("HasError should be false initially")
	}

	err := ErrorDisplay{
		Message:     "Test error",
		Code:        "test",
		Severity:    1,
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)

	if !sb.HasError() {
		t.Error("HasError should be true after ShowError")
	}

	sb.ClearError()

	if sb.HasError() {
		t.Error("HasError should be false after ClearError")
	}
}

// TestStatusBar_ErrorExpired tests checking if error has expired.
func TestStatusBar_ErrorExpired(t *testing.T) {
	sb := NewStatusBar(120)

	// No error - should not be expired
	if sb.ErrorExpired() {
		t.Error("ErrorExpired should be false when no error")
	}

	// Error with 5 second auto-dismiss, just shown
	err := ErrorDisplay{
		Message:     "Test error",
		Code:        "test",
		Severity:    1,
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)

	// Should not be expired yet
	if sb.ErrorExpired() {
		t.Error("ErrorExpired should be false immediately after ShowError")
	}

	// Simulate passage of time by setting errorShownAt in the past
	sb.errorShownAt = time.Now().Add(-6 * time.Second)

	// Now should be expired
	if !sb.ErrorExpired() {
		t.Error("ErrorExpired should be true after duration passed")
	}
}

// TestStatusBar_TimeUntilErrorDismiss tests remaining time calculation.
func TestStatusBar_TimeUntilErrorDismiss(t *testing.T) {
	sb := NewStatusBar(120)

	// No error - should be 0
	remaining := sb.TimeUntilErrorDismiss()
	if remaining != 0 {
		t.Errorf("TimeUntilErrorDismiss = %v, want 0 when no error", remaining)
	}

	// Error with 5 second auto-dismiss
	err := ErrorDisplay{
		Message:     "Test error",
		Code:        "test",
		Severity:    1,
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)

	// Should have ~5 seconds remaining
	remaining = sb.TimeUntilErrorDismiss()
	if remaining < 4*time.Second || remaining > 5*time.Second {
		t.Errorf("TimeUntilErrorDismiss = %v, expected ~5 seconds", remaining)
	}

	// Simulate 3 seconds passed
	sb.errorShownAt = time.Now().Add(-3 * time.Second)

	// Should have ~2 seconds remaining
	remaining = sb.TimeUntilErrorDismiss()
	if remaining < 1*time.Second || remaining > 2*time.Second {
		t.Errorf("TimeUntilErrorDismiss = %v, expected ~2 seconds", remaining)
	}

	// Simulate error expired
	sb.errorShownAt = time.Now().Add(-6 * time.Second)

	// Should be 0
	remaining = sb.TimeUntilErrorDismiss()
	if remaining != 0 {
		t.Errorf("TimeUntilErrorDismiss = %v, want 0 when expired", remaining)
	}
}

// TestStatusBar_ViewWithError tests rendering with an error displayed.
func TestStatusBar_ViewWithError(t *testing.T) {
	sb := NewStatusBar(120)

	info := StatusInfo{
		Model:         "llama3.1",
		SandboxMode:   "read-only",
		WorkingDir:    "/home/user/project",
		Status:        StatusIdle,
		TurnTokens:    0,
		SessionTokens: 0,
	}
	sb.SetInfo(info)

	// Render without error
	viewNormal := sb.View()
	if strings.Contains(viewNormal, "File not found") {
		t.Error("Normal view should not contain error message")
	}

	// Add error
	err := ErrorDisplay{
		Message:     "File not found",
		Code:        "not_found",
		Severity:    1, // Warning
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}
	sb.ShowError(err)

	// Render with error
	viewWithError := sb.View()
	if !strings.Contains(viewWithError, "File not found") {
		t.Error("View with error should contain error message")
	}
	if !strings.Contains(viewWithError, "⚠️") {
		t.Error("View with error should contain warning icon")
	}
}

// TestStatusBar_ErrorPriority tests error display takes priority over normal content.
func TestStatusBar_ErrorPriority(t *testing.T) {
	sb := NewStatusBar(120)

	info := StatusInfo{
		Model:       "llama3.1",
		SandboxMode: "read-only",
		WorkingDir:  "/home/user/project",
		Status:      StatusActive, // Active status
	}
	sb.SetInfo(info)

	// Add error
	err := ErrorDisplay{
		Message:     "Critical error",
		Code:        "internal",
		Severity:    3, // Critical
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 0, // No auto-dismiss
	}
	sb.ShowError(err)

	view := sb.View()

	// Error should be displayed
	if !strings.Contains(view, "Critical error") {
		t.Error("View should display error message even when status is active")
	}
	// Critical error icon should be present
	if !strings.Contains(view, "🔥") {
		t.Error("View should contain critical error icon")
	}
}

// TestStatusBar_ErrorColorCoding tests error severity affects styling.
func TestStatusBar_ErrorColorCoding(t *testing.T) {
	tests := []struct {
		name     string
		severity int
		wantIcon string
	}{
		{"info shows info icon", 0, "ℹ️"},
		{"warning shows warning icon", 1, "⚠️"},
		{"error shows error icon", 2, "❌"},
		{"critical shows critical icon", 3, "🔥"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar(120)

			err := ErrorDisplay{
				Message:     "Test message",
				Code:        "test",
				Severity:    tt.severity,
				Timestamp:   time.Now().Format(time.RFC3339),
				AutoDismiss: 5,
			}

			sb.ShowError(err)
			view := sb.View()

			if !strings.Contains(view, tt.wantIcon) {
				t.Errorf("View should contain icon %q for severity %d", tt.wantIcon, tt.severity)
			}
		})
	}
}

// TestStatusBar_AutoDismissCountdown tests countdown display.
func TestStatusBar_AutoDismissCountdown(t *testing.T) {
	sb := NewStatusBar(120)

	err := ErrorDisplay{
		Message:     "Test error",
		Code:        "test",
		Severity:    1,
		Timestamp:   time.Now().Format(time.RFC3339),
		AutoDismiss: 5,
	}

	sb.ShowError(err)

	// Simulate 2 seconds passed
	sb.errorShownAt = time.Now().Add(-2 * time.Second)

	view := sb.View()

	// Should show "dismiss in Xs" where X is 3 or less
	if !strings.Contains(view, "dismiss in") {
		t.Error("View should contain auto-dismiss countdown")
	}
}
