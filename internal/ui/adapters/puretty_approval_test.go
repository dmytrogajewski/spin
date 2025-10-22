package adapters

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// TestApprovalDialog_KeyboardInput tests that keyboard input is routed
// to the approval dialog when it's active.
// This is a regression test for the bug where arrow keys and A/D keys
// didn't work because all keyboard events were consumed by the prompt loop.
func TestApprovalDialog_KeyboardInput(t *testing.T) {
	// Create a buffer for output
	var buf bytes.Buffer

	// Create a keyboard event channel
	keyCh := make(chan term.KeyEvent, 10)

	// Create mock TTY
	mockTTY := &mockTerminalController{
		width:  80,
		height: 24,
	}

	// Create PureTTY with test options
	ui, err := NewPureTTY(&buf,
		WithTTY(mockTTY),
		withKeyboardEvents(keyCh),
	)
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	// Start UI in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ui.Run(ctx)
	}()

	// Give UI time to start
	time.Sleep(50 * time.Millisecond)

	// Create approval request
	cmd := &security.Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		Raw:     "rm -rf /",
	}
	req := security.ApprovalRequest{
		ID:      "test-123",
		Command: cmd,
		Reason:  "Testing approval dialog keyboard input",
		WorkDir: "/tmp",
	}

	// Start ShowApprovalDialog in background (it blocks)
	responseCh := make(chan security.ApprovalResponse, 1)
	go func() {
		resp := ui.ShowApprovalDialog(req)
		responseCh <- resp
	}()

	// Give dialog time to render
	time.Sleep(50 * time.Millisecond)

	// Verify mode is ModeApproval
	ui.mu.Lock()
	mode := ui.mode
	ui.mu.Unlock()

	if mode != ModeApproval {
		t.Fatalf("Expected mode to be ModeApproval, got %v", mode)
	}

	// Test 1: Send 'A' key to approve
	keyCh <- term.KeyEvent{
		Kind: term.KeyRune,
		Rune: 'A',
	}

	// Wait for response with timeout
	select {
	case resp := <-responseCh:
		if !resp.Approved {
			t.Error("Expected approval to be approved after pressing 'A'")
		}
		if resp.Reason != "user approved" {
			t.Errorf("Expected reason 'user approved', got '%s'", resp.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BUG: Approval dialog did not respond to 'A' key press - keyboard events not routed to dialog")
	}

	// Cleanup
	cancel()
}

// TestApprovalDialog_DenyKey tests that 'D' key denies the request.
func TestApprovalDialog_DenyKey(t *testing.T) {
	var buf bytes.Buffer
	keyCh := make(chan term.KeyEvent, 10)
	mockTTY := &mockTerminalController{width: 80, height: 24}

	ui, err := NewPureTTY(&buf, WithTTY(mockTTY), withKeyboardEvents(keyCh))
	if err != nil {
		t.Fatalf("NewPureTTY failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ui.Run(ctx)
	}()
	time.Sleep(50 * time.Millisecond)

	req := security.ApprovalRequest{
		ID:      "test-456",
		Command: &security.Command{Program: "rm", Raw: "rm file.txt"},
		Reason:  "Testing deny",
		WorkDir: "/tmp",
	}

	responseCh := make(chan security.ApprovalResponse, 1)
	go func() {
		resp := ui.ShowApprovalDialog(req)
		responseCh <- resp
	}()
	time.Sleep(50 * time.Millisecond)

	// Send 'D' key to deny
	keyCh <- term.KeyEvent{
		Kind: term.KeyRune,
		Rune: 'D',
	}

	select {
	case resp := <-responseCh:
		if resp.Approved {
			t.Error("Expected approval to be denied after pressing 'D'")
		}
		if resp.Reason != "user denied" {
			t.Errorf("Expected reason 'user denied', got '%s'", resp.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BUG: Approval dialog did not respond to 'D' key press")
	}

	cancel()
}

// TestApprovalDialog_ArrowKeys tests that arrow keys don't work (current bug).
// After the fix, this should be updated to test arrow key navigation.
func TestApprovalDialog_ArrowKeys(t *testing.T) {
	t.Skip("Arrow key navigation not yet implemented - waiting for keyboard routing fix")

	// TODO: After fixing keyboard routing, implement this test:
	// 1. Show approval dialog
	// 2. Press right arrow to move to "Deny" button
	// 3. Verify selection changed
	// 4. Press Enter to confirm
	// 5. Verify denial
}

// mockTerminalController is a mock implementation for testing.
type mockTerminalController struct {
	width      int
	height     int
	entered    bool
	resizeFunc func(int, int)
}

func (m *mockTerminalController) Enter() error {
	m.entered = true
	return nil
}

func (m *mockTerminalController) Exit() error {
	m.entered = false
	return nil
}

func (m *mockTerminalController) Size() (int, int) {
	return m.width, m.height
}

func (m *mockTerminalController) OnResize(f func(int, int)) {
	m.resizeFunc = f
}

// withKeyboardEvents is a test option to inject keyboard events.
func withKeyboardEvents(keyCh <-chan term.KeyEvent) PureTTYOption {
	return func(p *PureTTY) error {
		p.keyboardEvents = keyCh
		return nil
	}
}
