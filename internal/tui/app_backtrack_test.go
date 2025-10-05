package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// Helper to update and assert the model
func updateModel(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	newModel, _ := m.Update(msg)
	model, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Update returned wrong type: %T", newModel)
	}
	return model
}

// TestBacktrackModeEntry tests entering backtrack mode with Esc-Esc.
func TestBacktrackModeEntry(t *testing.T) {
	tests := []struct {
		name          string
		inputValue    string
		escPresses    int
		expectedState AppState
		expectedIdx   int // Expected backtrack index (-1 = not set)
	}{
		{
			name:          "Esc-Esc with empty input enters backtrack",
			inputValue:    "",
			escPresses:    2,
			expectedState: StateBacktrackMode,
			expectedIdx:   4, // Points to last user message (index 4)
		},
		{
			name:          "Single Esc with empty input stays idle",
			inputValue:    "",
			escPresses:    1,
			expectedState: StateIdle,
			expectedIdx:   -1,
		},
		{
			name:          "Esc with non-empty input clears input",
			inputValue:    "some text",
			escPresses:    1,
			expectedState: StateIdle,
			expectedIdx:   -1,
		},
		{
			name:          "Three Esc presses navigates in backtrack",
			inputValue:    "",
			escPresses:    3,
			expectedState: StateBacktrackMode,
			expectedIdx:   2, // Should navigate to previous user message (0->2->4, then 4->2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel()

			// Add test messages (user messages at indices 0, 2, 4)
			m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "First user message", Timestamp: time.Now()})
			m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "First assistant response", Timestamp: time.Now()})
			m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Second user message", Timestamp: time.Now()})
			m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "Second assistant response", Timestamp: time.Now()})
			m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Third user message", Timestamp: time.Now()})

			// Set input value
			if tt.inputValue != "" {
				m.input.SetValue(tt.inputValue)
			}

			// Press Esc multiple times
			for i := 0; i < tt.escPresses; i++ {
				m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
			}

			// Check state
			if m.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, m.state)
			}

			// Check backtrack index (only if in backtrack mode)
			if tt.expectedState == StateBacktrackMode {
				if m.backtrackIdx != tt.expectedIdx {
					t.Errorf("Expected backtrackIdx %d, got %d", tt.expectedIdx, m.backtrackIdx)
				}
			}
		})
	}
}

// TestBacktrackNavigation tests navigating through messages with Esc in backtrack mode.
func TestBacktrackNavigation(t *testing.T) {
	m := NewModel()

	// Add messages (user at 0, 2, 4)
	messages := []ui.Message{
		{Role: ui.RoleUser, Content: "Message 1", Timestamp: time.Now()},
		{Role: ui.RoleAssistant, Content: "Response 1", Timestamp: time.Now()},
		{Role: ui.RoleUser, Content: "Message 2", Timestamp: time.Now()},
		{Role: ui.RoleAssistant, Content: "Response 2", Timestamp: time.Now()},
		{Role: ui.RoleUser, Content: "Message 3", Timestamp: time.Now()},
	}
	for _, msg := range messages {
		m.chat.AddMessage(msg)
	}

	// Enter backtrack mode (Esc-Esc)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Should start at last user message (index 4)
	if m.backtrackIdx != 4 {
		t.Fatalf("Expected initial backtrackIdx 4, got %d", m.backtrackIdx)
	}

	// Press Esc to go to previous user message (index 2)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.backtrackIdx != 2 {
		t.Errorf("Expected backtrackIdx 2 after 1 nav, got %d", m.backtrackIdx)
	}

	// Press Esc again to go to first user message (index 0)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.backtrackIdx != 0 {
		t.Errorf("Expected backtrackIdx 0 after 2 navs, got %d", m.backtrackIdx)
	}

	// Press Esc again - should stay at 0 (no wrap-around)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.backtrackIdx != 0 {
		t.Errorf("Expected backtrackIdx to stay at 0, got %d", m.backtrackIdx)
	}

	// State should still be StateBacktrackMode
	if m.state != StateBacktrackMode {
		t.Errorf("Expected state to remain StateBacktrackMode, got %v", m.state)
	}
}

// TestBacktrackSelection tests loading selected message into input with Enter.
func TestBacktrackSelection(t *testing.T) {
	m := NewModel()

	// Add messages
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "First message", Timestamp: time.Now()})
	m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "Response", Timestamp: time.Now()})
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Second message", Timestamp: time.Now()})

	// Enter backtrack (Esc-Esc)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Should be at index 2 ("Second message")
	if m.backtrackIdx != 2 {
		t.Fatalf("Expected backtrackIdx 2, got %d", m.backtrackIdx)
	}

	// Navigate to previous message (index 0, "First message")
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Press Enter to select
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Check state transitioned to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after selection, got %v", m.state)
	}

	// Check input contains selected message
	if m.input.GetValue() != "First message" {
		t.Errorf("Expected input 'First message', got '%s'", m.input.GetValue())
	}

	// Check backtrackIdx NOT reset yet (kept for forking)
	if m.backtrackIdx == -1 {
		t.Errorf("Expected backtrackIdx to be kept for forking, got -1")
	}
}

// TestBacktrackCancel tests cancelling backtrack mode with Ctrl+C.
func TestBacktrackCancel(t *testing.T) {
	m := NewModel()

	// Add messages
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Test message", Timestamp: time.Now()})

	// Enter backtrack mode
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.state != StateBacktrackMode {
		t.Fatalf("Failed to enter backtrack mode")
	}

	// Press Ctrl+C to cancel (Phase 3.9: exits backtrack to Idle, not app)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)

	// Should transition to StateIdle (Phase 3.9 behavior)
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after Ctrl+C, got %v", m.state)
	}

	// Backtrack index should be reset
	if m.backtrackIdx != -1 {
		t.Errorf("Expected backtrackIdx -1, got %d", m.backtrackIdx)
	}

	// No quit command (not exiting app, just exiting backtrack)
	if cmd != nil {
		t.Error("Expected nil command after Ctrl+C (not exiting app)")
	}
}

// TestBacktrackWithEmptyTranscript tests backtrack with no messages.
func TestBacktrackWithEmptyTranscript(t *testing.T) {
	m := NewModel()

	// No messages added

	// Try to enter backtrack (Esc-Esc)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Should stay in Idle (no user messages to backtrack)
	if m.state != StateIdle {
		t.Errorf("Expected state Idle with empty transcript, got %v", m.state)
	}

	if m.backtrackIdx != -1 {
		t.Errorf("Expected backtrackIdx -1 with empty transcript, got %d", m.backtrackIdx)
	}
}

// TestBacktrackWithSingleMessage tests backtrack with only one user message.
func TestBacktrackWithSingleMessage(t *testing.T) {
	m := NewModel()

	// Add single user message
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Only message", Timestamp: time.Now()})

	// Enter backtrack
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Should enter backtrack mode and point to index 0
	if m.state != StateBacktrackMode {
		t.Fatalf("Expected StateBacktrackMode, got %v", m.state)
	}

	if m.backtrackIdx != 0 {
		t.Errorf("Expected backtrackIdx 0, got %d", m.backtrackIdx)
	}

	// Try to navigate backward (should stay at 0)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.backtrackIdx != 0 {
		t.Errorf("Expected backtrackIdx to stay at 0, got %d", m.backtrackIdx)
	}

	// Select the message
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.input.GetValue() != "Only message" {
		t.Errorf("Expected input 'Only message', got '%s'", m.input.GetValue())
	}
}

// TestConversationForking tests that editing and resubmitting truncates conversation.
func TestConversationForking(t *testing.T) {
	m := NewModel()

	// Add conversation
	messages := []ui.Message{
		{Role: ui.RoleUser, Content: "Message 1", Timestamp: time.Now()},
		{Role: ui.RoleAssistant, Content: "Response 1", Timestamp: time.Now()},
		{Role: ui.RoleUser, Content: "Message 2", Timestamp: time.Now()},
		{Role: ui.RoleAssistant, Content: "Response 2", Timestamp: time.Now()},
		{Role: ui.RoleUser, Content: "Message 3", Timestamp: time.Now()},
		{Role: ui.RoleAssistant, Content: "Response 3", Timestamp: time.Now()},
	}
	for _, msg := range messages {
		m.chat.AddMessage(msg)
	}

	// Enter backtrack and navigate to "Message 2" (index 2)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc}) // Navigate back once

	if m.backtrackIdx != 2 {
		t.Fatalf("Expected backtrackIdx 2, got %d", m.backtrackIdx)
	}

	// Select the message
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Edit the message
	m.input.SetValue("Edited message 2")

	// Submit the edited message (Enter key)
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	// Check that conversation was truncated:
	// Should have: Message 1, Response 1, Edited message 2
	// Should NOT have: Response 2, Message 3, Response 3
	allMessages := m.chat.GetMessages()

	expectedCount := 3 // Message 1, Response 1, Edited message 2
	if len(allMessages) != expectedCount {
		t.Errorf("Expected %d messages after fork, got %d", expectedCount, len(allMessages))
	}

	// Check that the message was replaced
	if len(allMessages) >= 3 {
		if allMessages[2].Content != "Edited message 2" {
			t.Errorf("Expected message content 'Edited message 2', got '%s'", allMessages[2].Content)
		}
		if allMessages[2].Role != ui.RoleUser {
			t.Errorf("Expected role User, got %v", allMessages[2].Role)
		}
	}
}

// TestBacktrackOnlySelectsUserMessages tests that only user messages are selectable.
func TestBacktrackOnlySelectsUserMessages(t *testing.T) {
	m := NewModel()

	// Add messages with no user messages, only system/assistant
	m.chat.AddMessage(ui.Message{Role: ui.RoleSystem, Content: "System message", Timestamp: time.Now()})
	m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "Assistant message", Timestamp: time.Now()})

	// Try to enter backtrack
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})

	// Should stay in Idle (no user messages)
	if m.state != StateIdle {
		t.Errorf("Expected state Idle when no user messages, got %v", m.state)
	}

	if m.backtrackIdx != -1 {
		t.Errorf("Expected backtrackIdx -1 when no user messages, got %d", m.backtrackIdx)
	}
}

// TestBacktrackStateTransitions tests state transition validity.
func TestBacktrackStateTransitions(t *testing.T) {
	// Test that StateBacktrackMode can transition to StateIdle
	if !StateBacktrackMode.CanTransitionTo(StateIdle) {
		t.Error("StateBacktrackMode should allow transition to StateIdle")
	}

	// Test that StateBacktrackMode can transition to StateExiting
	if !StateBacktrackMode.CanTransitionTo(StateExiting) {
		t.Error("StateBacktrackMode should allow transition to StateExiting")
	}

	// Test that StateBacktrackMode cannot transition to invalid states
	invalidStates := []AppState{
		StateWaitingResponse,
		StateToolApproval,
		StateFilePickerOpen,
	}

	for _, invalidState := range invalidStates {
		if StateBacktrackMode.CanTransitionTo(invalidState) {
			t.Errorf("StateBacktrackMode should not allow transition to %v", invalidState)
		}
	}
}
