package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mockErrorDisplay creates a test ErrorDisplay for testing.
func mockErrorDisplay(msg, code string, severity int) ErrorDisplay {
	return ErrorDisplay{
		Message:     msg,
		Code:        code,
		Details:     "Test details for " + msg,
		Operation:   "Test.Operation",
		Severity:    severity,
		Timestamp:   time.Now().Format(time.RFC3339),
		Dismissible: true,
		Dismissed:   false,
		AutoDismiss: 0,
	}
}

// TestNewErrorModal tests modal creation.
func TestNewErrorModal(t *testing.T) {
	m := NewErrorModal(80, 24)

	if m.Visible {
		t.Error("Modal should not be visible initially")
	}
	if len(m.Errors) != 0 {
		t.Errorf("Errors slice should be empty, got %d errors", len(m.Errors))
	}
	if m.CurrentIdx != 0 {
		t.Errorf("CurrentIdx should be 0, got %d", m.CurrentIdx)
	}
	if m.Width != 80 {
		t.Errorf("Width should be 80, got %d", m.Width)
	}
	if m.Height != 24 {
		t.Errorf("Height should be 24, got %d", m.Height)
	}
}

// TestErrorModal_Show tests showing an error in the modal.
func TestErrorModal_Show(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Critical error", "internal", 3)

	m.Show(err)

	if !m.Visible {
		t.Error("Modal should be visible after Show()")
	}
	if len(m.Errors) != 1 {
		t.Errorf("Should have 1 error, got %d", len(m.Errors))
	}
	if m.CurrentIdx != 0 {
		t.Errorf("CurrentIdx should be 0, got %d", m.CurrentIdx)
	}
	if m.Errors[0].Message != "Critical error" {
		t.Errorf("Error message = %q, want %q", m.Errors[0].Message, "Critical error")
	}
}

// TestErrorModal_ShowMultiple tests showing multiple errors.
func TestErrorModal_ShowMultiple(t *testing.T) {
	m := NewErrorModal(80, 24)

	err1 := mockErrorDisplay("Error 1", "code1", 2)
	err2 := mockErrorDisplay("Error 2", "code2", 3)
	err3 := mockErrorDisplay("Error 3", "code3", 2)

	m.Show(err1)
	m.Show(err2)
	m.Show(err3)

	if len(m.Errors) != 3 {
		t.Errorf("Should have 3 errors, got %d", len(m.Errors))
	}
	if m.CurrentIdx != 2 {
		t.Errorf("CurrentIdx should be 2 (latest), got %d", m.CurrentIdx)
	}
	if !m.Visible {
		t.Error("Modal should be visible")
	}
}

// TestErrorModal_Hide tests hiding the modal.
func TestErrorModal_Hide(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Error", "code", 2)

	m.Show(err)
	if !m.Visible {
		t.Error("Modal should be visible before Hide()")
	}

	m.Hide()
	if m.Visible {
		t.Error("Modal should not be visible after Hide()")
	}

	// Errors should still be preserved
	if len(m.Errors) != 1 {
		t.Error("Errors should be preserved after Hide()")
	}
}

// TestErrorModal_Navigation tests up/down navigation.
func TestErrorModal_Navigation(t *testing.T) {
	m := NewErrorModal(80, 24)

	m.Show(mockErrorDisplay("Error 0", "code", 2))
	m.Show(mockErrorDisplay("Error 1", "code", 2))
	m.Show(mockErrorDisplay("Error 2", "code", 2))

	// Should start at latest (index 2)
	if m.CurrentIdx != 2 {
		t.Fatalf("Initial CurrentIdx = %d, want 2", m.CurrentIdx)
	}

	// Navigate up (to older error)
	m.PrevError()
	if m.CurrentIdx != 1 {
		t.Errorf("After PrevError, CurrentIdx = %d, want 1", m.CurrentIdx)
	}

	m.PrevError()
	if m.CurrentIdx != 0 {
		t.Errorf("After PrevError, CurrentIdx = %d, want 0", m.CurrentIdx)
	}

	// At top, should stay at 0
	m.PrevError()
	if m.CurrentIdx != 0 {
		t.Errorf("At top, CurrentIdx = %d, should stay at 0", m.CurrentIdx)
	}

	// Navigate down (to newer error)
	m.NextError()
	if m.CurrentIdx != 1 {
		t.Errorf("After NextError, CurrentIdx = %d, want 1", m.CurrentIdx)
	}

	m.NextError()
	if m.CurrentIdx != 2 {
		t.Errorf("After NextError, CurrentIdx = %d, want 2", m.CurrentIdx)
	}

	// At bottom, should stay at 2
	m.NextError()
	if m.CurrentIdx != 2 {
		t.Errorf("At bottom, CurrentIdx = %d, should stay at 2", m.CurrentIdx)
	}
}

// TestErrorModal_CurrentError tests getting the current error.
func TestErrorModal_CurrentError(t *testing.T) {
	m := NewErrorModal(80, 24)

	// No errors - should return nil
	if m.CurrentError() != nil {
		t.Error("CurrentError should be nil when no errors")
	}

	err1 := mockErrorDisplay("Error 1", "code1", 2)
	err2 := mockErrorDisplay("Error 2", "code2", 2)

	m.Show(err1)
	m.Show(err2)

	// Should return latest error
	current := m.CurrentError()
	if current == nil {
		t.Fatal("CurrentError should not be nil")
	}
	if current.Message != "Error 2" {
		t.Errorf("Current error message = %q, want %q", current.Message, "Error 2")
	}

	// Navigate to previous
	m.PrevError()
	current = m.CurrentError()
	if current == nil {
		t.Fatal("CurrentError should not be nil")
	}
	if current.Message != "Error 1" {
		t.Errorf("Current error message = %q, want %q", current.Message, "Error 1")
	}
}

// TestErrorModal_Update_KeyboardDismiss tests dismissal via keyboard.
func TestErrorModal_Update_KeyboardDismiss(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Error", "code", 2)
	m.Show(err)

	// Test Esc key
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedM.Visible {
		t.Error("Modal should be hidden after Esc")
	}

	// Show again and test Enter key
	m.Show(err)
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updatedM.Visible {
		t.Error("Modal should be hidden after Enter")
	}
}

// TestErrorModal_Update_Navigation tests navigation via keyboard.
func TestErrorModal_Update_Navigation(t *testing.T) {
	m := NewErrorModal(80, 24)
	m.Show(mockErrorDisplay("Error 1", "code", 2))
	m.Show(mockErrorDisplay("Error 2", "code", 2))

	// Start at index 1 (latest)
	if m.CurrentIdx != 1 {
		t.Fatalf("Initial CurrentIdx = %d, want 1", m.CurrentIdx)
	}

	// Press Up (previous error)
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updatedM.CurrentIdx != 0 {
		t.Errorf("After Up, CurrentIdx = %d, want 0", updatedM.CurrentIdx)
	}

	// Press Down (next error)
	updatedM, _ = updatedM.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updatedM.CurrentIdx != 1 {
		t.Errorf("After Down, CurrentIdx = %d, want 1", updatedM.CurrentIdx)
	}
}

// TestErrorModal_Update_IgnoreWhenHidden tests that hidden modal ignores input.
func TestErrorModal_Update_IgnoreWhenHidden(t *testing.T) {
	m := NewErrorModal(80, 24)
	m.Hide()

	// Ensure modal is hidden
	if m.Visible {
		t.Fatal("Modal should be hidden for this test")
	}

	// Try to navigate - should be no-op
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updatedM.Visible {
		t.Error("Modal should remain hidden")
	}

	// Try to dismiss - should be no-op
	updatedM, _ = updatedM.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updatedM.Visible {
		t.Error("Modal should remain hidden")
	}
}

// TestErrorModal_View_Empty tests rendering when no errors.
func TestErrorModal_View_Empty(t *testing.T) {
	m := NewErrorModal(80, 24)

	view := m.View()
	if view != "" {
		t.Error("View should be empty string when no errors")
	}
}

// TestErrorModal_View_Hidden tests rendering when hidden.
func TestErrorModal_View_Hidden(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Error", "code", 2)
	m.Show(err)
	m.Hide()

	view := m.View()
	if view != "" {
		t.Error("View should be empty string when hidden")
	}
}

// TestErrorModal_View_Visible tests rendering when visible.
func TestErrorModal_View_Visible(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Critical error occurred", "internal", 3)
	m.Show(err)

	view := m.View()
	if view == "" {
		t.Fatal("View should not be empty when modal is visible")
	}

	// Check for key content
	if !strings.Contains(view, "Critical error occurred") {
		t.Error("View should contain error message")
	}
	if !strings.Contains(view, "internal") {
		t.Error("View should contain error code")
	}
	if !strings.Contains(view, "[Esc] Dismiss") {
		t.Error("View should contain dismiss instruction")
	}
}

// TestErrorModal_View_Multiline tests rendering with multiline details.
func TestErrorModal_View_Multiline(t *testing.T) {
	m := NewErrorModal(80, 24)
	err := mockErrorDisplay("Error", "code", 2)
	err.Details = "Line 1\nLine 2\nLine 3"
	m.Show(err)

	view := m.View()
	if !strings.Contains(view, "Line 1") {
		t.Error("View should contain first line of details")
	}
	if !strings.Contains(view, "Line 2") {
		t.Error("View should contain second line of details")
	}
}

// TestErrorModal_Resize tests modal resizing.
func TestErrorModal_Resize(t *testing.T) {
	m := NewErrorModal(80, 24)

	m.Resize(100, 30)

	if m.Width != 100 {
		t.Errorf("Width = %d, want 100", m.Width)
	}
	if m.Height != 30 {
		t.Errorf("Height = %d, want 30", m.Height)
	}
}

// TestErrorModal_ErrorCount tests error count tracking.
func TestErrorModal_ErrorCount(t *testing.T) {
	m := NewErrorModal(80, 24)

	if m.ErrorCount() != 0 {
		t.Errorf("ErrorCount = %d, want 0", m.ErrorCount())
	}

	m.Show(mockErrorDisplay("E1", "c1", 2))
	if m.ErrorCount() != 1 {
		t.Errorf("ErrorCount = %d, want 1", m.ErrorCount())
	}

	m.Show(mockErrorDisplay("E2", "c2", 2))
	if m.ErrorCount() != 2 {
		t.Errorf("ErrorCount = %d, want 2", m.ErrorCount())
	}

	m.Show(mockErrorDisplay("E3", "c3", 2))
	if m.ErrorCount() != 3 {
		t.Errorf("ErrorCount = %d, want 3", m.ErrorCount())
	}
}

// TestErrorModal_Clear tests clearing all errors.
func TestErrorModal_Clear(t *testing.T) {
	m := NewErrorModal(80, 24)
	m.Show(mockErrorDisplay("E1", "c1", 2))
	m.Show(mockErrorDisplay("E2", "c2", 2))

	if m.ErrorCount() != 2 {
		t.Fatalf("Setup failed: ErrorCount = %d, want 2", m.ErrorCount())
	}

	m.Clear()

	if m.ErrorCount() != 0 {
		t.Errorf("After Clear, ErrorCount = %d, want 0", m.ErrorCount())
	}
	if m.Visible {
		t.Error("Modal should be hidden after Clear")
	}
	if m.CurrentIdx != 0 {
		t.Errorf("CurrentIdx should be reset to 0, got %d", m.CurrentIdx)
	}
}
