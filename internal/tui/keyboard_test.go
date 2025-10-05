package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// TestCtrlC_CancelTurn tests Ctrl+C during active AI generation
func TestCtrlC_CancelTurn(t *testing.T) {
	m := NewModel()
	m.state = StateWaitingResponse

	// Simulate Ctrl+C
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	// Should transition to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle, got %v", m.state)
	}

	// Should have cancellation message in transcript
	messages := m.chat.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected cancellation message in transcript")
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != ui.RoleSystem {
		t.Errorf("Expected system message, got %v", lastMsg.Role)
	}
	if lastMsg.Content != "Turn cancelled by user" {
		t.Errorf("Expected cancellation message, got: %s", lastMsg.Content)
	}

	// No command should be returned (state transition only)
	if cmd != nil {
		t.Error("Expected nil command after cancellation")
	}
}

// TestCtrlC_ExitFromIdle tests Ctrl+C when idle (should exit)
func TestCtrlC_ExitFromIdle(t *testing.T) {
	m := NewModel()
	m.state = StateIdle
	m.input.Clear() // Ensure input is empty

	// Simulate Ctrl+C
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	// Should transition to Exiting
	if m.state != StateExiting {
		t.Errorf("Expected state StateExiting, got %v", m.state)
	}

	// Should set quitting flag
	if !m.quitting {
		t.Error("Expected quitting flag to be true")
	}

	// Should return Quit command
	if cmd == nil {
		t.Error("Expected Quit command")
	}
}

// TestCtrlC_DenyApproval tests Ctrl+C during tool approval
func TestCtrlC_DenyApproval(t *testing.T) {
	m := NewModel()
	m.state = StateToolApproval

	// Simulate Ctrl+C
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	// Should transition to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after approval cancellation, got %v", m.state)
	}

	// Should have denial message in transcript
	messages := m.chat.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected denial message in transcript")
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != ui.RoleSystem {
		t.Errorf("Expected system message, got %v", lastMsg.Role)
	}
}

// TestCtrlC_ExitBacktrack tests Ctrl+C during backtrack mode
func TestCtrlC_ExitBacktrack(t *testing.T) {
	m := NewModel()

	// Setup backtrack mode with messages
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "First message"})
	m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "Response"})
	m.state = StateBacktrackMode
	m.backtrackIdx = 0

	// Simulate Ctrl+C
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	// Should transition to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle, got %v", m.state)
	}

	// Should reset backtrack index
	if m.backtrackIdx != -1 {
		t.Errorf("Expected backtrackIdx -1, got %d", m.backtrackIdx)
	}

	// Should clear highlight
	messages := m.chat.GetMessages()
	for i, msg := range messages {
		if msg.Highlighted {
			t.Errorf("Message %d should not be highlighted after exiting backtrack", i)
		}
	}
}

// TestCtrlD_GracefulExit tests Ctrl+D from any state
func TestCtrlD_GracefulExit(t *testing.T) {
	states := []AppState{
		StateIdle,
		StateWaitingResponse,
		StateToolApproval,
		StateBacktrackMode,
		StateFilePickerOpen,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			m := NewModel()
			m.state = state

			// Simulate Ctrl+D
			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
			m = updated.(Model)

			// Should transition to Exiting
			if m.state != StateExiting {
				t.Errorf("Expected state StateExiting, got %v", m.state)
			}

			// Should set quitting flag
			if !m.quitting {
				t.Error("Expected quitting flag to be true")
			}

			// Should return Quit command
			if cmd == nil {
				t.Error("Expected Quit command")
			}
		})
	}
}

// TestCtrlL_ClearScreen tests Ctrl+L screen clear
func TestCtrlL_ClearScreen(t *testing.T) {
	m := NewModel()
	m.state = StateIdle

	// Add some messages to transcript
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Message 1"})
	m.chat.AddMessage(ui.Message{Role: ui.RoleAssistant, Content: "Response 1"})
	m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Message 2"})

	// Simulate Ctrl+L
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	m = updated.(Model)

	// Transcript should be preserved (messages not cleared)
	messages := m.chat.GetMessages()
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages preserved, got %d", len(messages))
	}

	// Viewport should be scrolled to bottom (tested via chat component behavior)
	// State should remain Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after Ctrl+L, got %v", m.state)
	}
}

// TestCtrlL_OnlyInIdle tests that Ctrl+L only works in Idle state
func TestCtrlL_OnlyInIdle(t *testing.T) {
	states := []AppState{
		StateWaitingResponse,
		StateToolApproval,
		StateBacktrackMode,
		StateFilePickerOpen,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			m := NewModel()
			m.state = state
			m.chat.AddMessage(ui.Message{Role: ui.RoleUser, Content: "Message"})

			// Simulate Ctrl+L
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
			m = updated.(Model)

			// State should not change
			if m.state != state {
				t.Errorf("Expected state %v, got %v", state, m.state)
			}

			// Messages should be preserved
			if len(m.chat.GetMessages()) != 1 {
				t.Error("Messages should be preserved even when Ctrl+L is ignored")
			}
		})
	}
}

// TestCtrlH_ShowHelp tests Ctrl+H help display
func TestCtrlH_ShowHelp(t *testing.T) {
	m := NewModel()
	m.state = StateIdle

	// Simulate Ctrl+H
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
	m = updated.(Model)

	// Should transition to Help state
	if m.state != StateHelp {
		t.Errorf("Expected state StateHelp, got %v", m.state)
	}
}

// TestQuestionMark_ShowHelp tests ? help display
func TestQuestionMark_ShowHelp(t *testing.T) {
	m := NewModel()
	m.state = StateIdle

	// Simulate ?
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)

	// Should transition to Help state
	if m.state != StateHelp {
		t.Errorf("Expected state StateHelp, got %v", m.state)
	}
}

// TestHelp_Dismiss tests dismissing help modal
func TestHelp_Dismiss(t *testing.T) {
	m := NewModel()
	m.state = StateHelp

	// Simulate any key (e.g., 'a')
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	// Should return to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after dismissing help, got %v", m.state)
	}
}

// TestHelp_DismissWithEsc tests dismissing help with Esc
func TestHelp_DismissWithEsc(t *testing.T) {
	m := NewModel()
	m.state = StateHelp

	// Simulate Esc
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Should return to Idle
	if m.state != StateIdle {
		t.Errorf("Expected state StateIdle after Esc in help, got %v", m.state)
	}
}

// TestKeyboardShortcutConflicts ensures no conflicts
func TestKeyboardShortcutConflicts(t *testing.T) {
	// This test documents all keyboard shortcuts and ensures they don't conflict

	globalShortcuts := map[string]string{
		"ctrl+c": "Cancel/Exit",
		"ctrl+d": "Exit",
		"ctrl+h": "Help",
		"?":      "Help",
	}

	idleShortcuts := map[string]string{
		"enter":  "Submit",
		"esc":    "Clear input or enter backtrack",
		"@":      "File picker",
		"ctrl+l": "Clear screen",
	}

	approvalShortcuts := map[string]string{
		"a": "Approve",
		"d": "Deny",
		"m": "Modify",
	}

	// Ensure no overlap
	allShortcuts := make(map[string]bool)

	for key := range globalShortcuts {
		allShortcuts[key] = true
	}

	for key := range idleShortcuts {
		if key == "?" || key == "ctrl+c" || key == "ctrl+d" || key == "ctrl+h" {
			continue // Expected overlap with global
		}
		if allShortcuts[key] {
			t.Errorf("Conflict detected: %s used in multiple contexts", key)
		}
		allShortcuts[key] = true
	}

	// Approval shortcuts are state-specific, so they can overlap with others
	// but we document them here
	for key := range approvalShortcuts {
		_ = key // Just document, no conflict check needed for state-specific
	}

	t.Logf("Total unique shortcuts documented: %d", len(allShortcuts))
}

// TestCtrlC_NoDoubleCancel tests that Ctrl+C doesn't double-cancel
func TestCtrlC_NoDoubleCancel(t *testing.T) {
	m := NewModel()
	m.state = StateWaitingResponse

	// First Ctrl+C
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if m.state != StateIdle {
		t.Fatalf("Expected StateIdle after first Ctrl+C, got %v", m.state)
	}

	messageCount := len(m.chat.GetMessages())

	// Second Ctrl+C (should exit, not add another cancellation message)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if m.state != StateExiting {
		t.Errorf("Expected StateExiting after second Ctrl+C, got %v", m.state)
	}

	// Should not add another cancellation message
	if len(m.chat.GetMessages()) != messageCount {
		t.Error("Second Ctrl+C should not add another message")
	}

	// Should return Quit command
	if cmd == nil {
		t.Error("Expected Quit command on second Ctrl+C")
	}
}

// TestEscPressCount_Reset tests that Esc counter resets on other keys
func TestEscPressCount_Reset(t *testing.T) {
	m := NewModel()
	m.state = StateIdle
	m.input.Clear()

	// Press Esc once
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.escPressCount != 1 {
		t.Fatalf("Expected escPressCount=1, got %d", m.escPressCount)
	}

	// Press a different key (e.g., 'a')
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	// Esc counter should reset
	if m.escPressCount != 0 {
		t.Errorf("Expected escPressCount=0 after typing, got %d", m.escPressCount)
	}

	// Should still be in Idle (not backtrack)
	if m.state != StateIdle {
		t.Errorf("Expected StateIdle, got %v", m.state)
	}
}

// TestStateExiting_IgnoresInput tests that StateExiting ignores all input
func TestStateExiting_IgnoresInput(t *testing.T) {
	m := NewModel()
	m.state = StateExiting
	m.quitting = true

	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyCtrlL},
	}

	for _, key := range keys {
		updated, _ := m.Update(key)
		m = updated.(Model)

		if m.state != StateExiting {
			t.Errorf("StateExiting should ignore all input, but state changed to %v", m.state)
		}
	}
}

// TestHelp_FromDifferentStates tests help access from different states
func TestHelp_FromDifferentStates(t *testing.T) {
	states := []AppState{
		StateIdle,
		StateWaitingResponse,
		StateBacktrackMode,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			m := NewModel()
			m.state = state

			// Simulate Ctrl+H
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlH})
			m = updated.(Model)

			// Should transition to Help from any state
			if m.state != StateHelp {
				t.Errorf("Expected StateHelp from %v, got %v", state, m.state)
			}
		})
	}
}
