package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/google/uuid"
)

func makeApprovalEvent(command, reason, workdir string) core.ApprovalEventData {
	return core.ApprovalEventData{
		RequestID: uuid.New().String(),
		Command:   command,
		Reason:    reason,
		WorkDir:   workdir,
		Timestamp: time.Now(),
	}
}

// TestApprovalModal_Create tests modal creation.

func TestApprovalModal_Create(t *testing.T) {
	req := core.ApprovalEventData{
		RequestID: uuid.New().String(),
		Command:   "rm -rf /tmp/cache",
		Reason:    "destructive operation",
		WorkDir:   "/home/user/project",
		Timestamp: time.Now(),
	}

	modal := NewApprovalModal(req, 80, 24)

	if modal.request.ID != req.RequestID {
		t.Errorf("expected request ID %s, got %s", req.RequestID, modal.request.ID)
	}

	if modal.editing {
		t.Error("expected editing to be false initially")
	}

	if modal.editValue != "" {
		t.Error("expected editValue to be empty initially")
	}

	if modal.width != 80 {
		t.Errorf("expected width 80, got %d", modal.width)
	}

	if modal.height != 24 {
		t.Errorf("expected height 24, got %d", modal.height)
	}
}

// TestApprovalModal_ApproveKey tests approval with 'a' key.
func TestApprovalModal_ApproveKey(t *testing.T) {
	req := makeApprovalEvent("ls -la", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'a'
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

// TestApprovalModal_ApproveKeyUppercase tests approval with 'A' key.
func TestApprovalModal_ApproveKeyUppercase(t *testing.T) {
	req := makeApprovalEvent("ls -la", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'A'
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})

	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

// TestApprovalModal_DenyKey tests denial with 'd' key.
func TestApprovalModal_DenyKey(t *testing.T) {
	req := makeApprovalEvent("rm -rf /", "extremely dangerous", "/")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'd'
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

// TestApprovalModal_DenyKeyUppercase tests denial with 'D' key.
func TestApprovalModal_DenyKeyUppercase(t *testing.T) {
	req := makeApprovalEvent("rm -rf /", "extremely dangerous", "/")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'D'
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

// TestApprovalModal_ModifyKey tests entering edit mode with 'm' key.
func TestApprovalModal_ModifyKey(t *testing.T) {
	req := makeApprovalEvent("git push --force", "force push", "/home/user/repo")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'm'
	updatedModal, _ := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	if !updatedModal.editing {
		t.Error("expected editing to be true after pressing 'm'")
	}

	if updatedModal.editValue != req.Command {
		t.Errorf("expected editValue to be %s, got %s", req.Command, updatedModal.editValue)
	}
}

// TestApprovalModal_ModifyKeyUppercase tests entering edit mode with 'M' key.
func TestApprovalModal_ModifyKeyUppercase(t *testing.T) {
	req := makeApprovalEvent("git push --force", "force push", "/home/user/repo")

	modal := NewApprovalModal(req, 80, 24)

	// Simulate pressing 'M'
	updatedModal, _ := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})

	if !updatedModal.editing {
		t.Error("expected editing to be true after pressing 'M'")
	}

	if updatedModal.editValue != req.Command {
		t.Errorf("expected editValue to be %s, got %s", req.Command, updatedModal.editValue)
	}
}

// TestApprovalModal_EditModeEnter tests confirming edit with Enter.
func TestApprovalModal_EditModeEnter(t *testing.T) {
	req := makeApprovalEvent("git push", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	// Enter edit mode
	modal, _ = modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	// Modify command
	modal.editValue = "git push origin main"

	// Press Enter to confirm
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Error("expected command to be returned after Enter")
	}
}

// TestApprovalModal_EditModeEscape tests canceling edit with Escape.
func TestApprovalModal_EditModeEscape(t *testing.T) {
	req := makeApprovalEvent("git push", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	// Enter edit mode
	modal, _ = modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	// Modify command
	modal.editValue = "git push origin main"

	// Press Escape to cancel
	updatedModal, _ := modal.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if updatedModal.editing {
		t.Error("expected editing to be false after Escape")
	}

	if updatedModal.editValue != "" {
		t.Error("expected editValue to be cleared after Escape")
	}
}

// TestApprovalModal_Resize tests modal resizing.
func TestApprovalModal_Resize(t *testing.T) {
	req := makeApprovalEvent("test", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	newWidth, newHeight := 100, 30
	modal.SetSize(newWidth, newHeight)

	if modal.width != newWidth {
		t.Errorf("expected width %d, got %d", newWidth, modal.width)
	}

	if modal.height != newHeight {
		t.Errorf("expected height %d, got %d", newHeight, modal.height)
	}
}

// TestApprovalModal_View tests view rendering.
func TestApprovalModal_View(t *testing.T) {
	req := makeApprovalEvent("rm -rf /tmp", "destructive", "/home/user")

	modal := NewApprovalModal(req, 80, 24)

	view := modal.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check that view contains key information
	if !contains(view, "rm -rf /tmp") {
		t.Error("expected view to contain command")
	}

	if !contains(view, "destructive") {
		t.Error("expected view to contain reason")
	}

	if !contains(view, "/home/user") {
		t.Error("expected view to contain working directory")
	}
}

// TestApprovalModal_ViewEditMode tests view rendering in edit mode.
func TestApprovalModal_ViewEditMode(t *testing.T) {
	req := makeApprovalEvent("git push", "test", "/tmp")

	modal := NewApprovalModal(req, 80, 24)

	// Enter edit mode
	modal, _ = modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	view := modal.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check that view shows edit mode UI
	if !contains(view, "git push") {
		t.Error("expected view to contain command being edited")
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
