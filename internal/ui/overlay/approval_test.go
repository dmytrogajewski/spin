package overlay

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/security"
)

func TestNewApprovalDialog(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	if dialog == nil {
		t.Fatal("NewApprovalDialog returned nil")
	}

	if dialog.request.ID != req.ID {
		t.Errorf("Expected request ID %s, got %s", req.ID, dialog.request.ID)
	}

	if dialog.IsVisible() {
		t.Error("Dialog should not be visible initially")
	}
}

func TestApprovalDialog_Show_Approve(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Start dialog in background.
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)

	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait for dialog to be ready (use channel-based synchronization).
	select {
	case <-time.After(50 * time.Millisecond):
		// Dialog should be ready by now.
	case <-ctx.Done():
		t.Fatal("Context canceled before dialog ready")
	}

	// Approve the dialog.
	dialog.Approve()

	// Wait for result.
	select {
	case result := <-resultCh:
		if !result.Approved {
			t.Error("Expected approval, got denial")
		}

		if result.RequestID != req.ID {
			t.Errorf("Expected request ID %s, got %s", req.ID, result.RequestID)
		}

		if result.Reason != "user approved" {
			t.Errorf("Expected reason 'user approved', got %s", result.Reason)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Dialog did not complete within timeout")
	}
}

func TestApprovalDialog_Show_Deny(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Start dialog in background.
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)

	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait a bit for dialog to start.
	time.Sleep(10 * time.Millisecond)

	// Deny the dialog.
	dialog.Deny()

	// Wait for result.
	select {
	case result := <-resultCh:
		if result.Approved {
			t.Error("Expected denial, got approval")
		}

		if result.RequestID != req.ID {
			t.Errorf("Expected request ID %s, got %s", req.ID, result.RequestID)
		}

		if result.Reason != "user denied" {
			t.Errorf("Expected reason 'user denied', got %s", result.Reason)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Dialog did not complete within timeout")
	}
}

func TestApprovalDialog_HandleKey(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	tests := []struct {
		name        string
		key         rune
		shouldClose bool
	}{
		{"Approve A", 'A', true},
		{"Approve a", 'a', true},
		{"Deny D", 'D', true},
		{"Deny d", 'd', true},
		{"Cancel ESC", '\x1b', true},
		{"Help ?", '?', false},
		{"Other key", 'x', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			closed := dialog.HandleKey(string(tt.key))
			if closed != tt.shouldClose {
				t.Errorf("HandleKey(%c) returned %v, expected %v", tt.key, closed, tt.shouldClose)
			}
		})
	}
}

func TestApprovalDialog_Render(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user/project",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Render dialog - should return empty string since we now use status bar.
	output := dialog.Render(80, 24)
	if output != "" {
		t.Error("Expected empty string since we now use status bar for approval display")
	}
}

func TestApprovalDialog_Render_LongContent(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /very/long/path/that/exceeds/normal/terminal/width/and/should/be/truncated"},
		Reason:    "This is a very long reason that should be truncated when it exceeds the dialog width",
		WorkDir:   "/home/user/very/long/project/path/that/might/also/be/truncated",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Render with small width - should return empty string since we now use status bar.
	output := dialog.Render(40, 24)
	if output != "" {
		t.Error("Expected empty string since we now use status bar for approval display")
	}
}

func TestApprovalDialog_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Test concurrent access.
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			// Test IsVisible.
			_ = dialog.IsVisible()

			// Test HandleKey.
			_ = dialog.HandleKey("x")
		}()
	}

	// Wait for all goroutines.
	for range 10 {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
}

func TestApprovalDialog_MultipleResponses(t *testing.T) {
	t.Parallel()
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req)

	// Start dialog.
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)

	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait for dialog to be ready.
	select {
	case <-time.After(50 * time.Millisecond):
		// Dialog should be ready by now.
	case <-ctx.Done():
		t.Fatal("Context canceled before dialog ready")
	}

	// Send multiple responses.
	dialog.Approve()
	dialog.Deny()
	dialog.Approve()

	// Should only get first response.
	select {
	case result := <-resultCh:
		if !result.Approved {
			t.Error("Expected first response (approve) to be returned")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Dialog did not complete within timeout")
	}
}
