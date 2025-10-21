package overlay

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/security"
)

func TestNewApprovalDialog(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 60*time.Second)

	if dialog == nil {
		t.Fatal("NewApprovalDialog returned nil")
	}

	if dialog.request.ID != req.ID {
		t.Errorf("Expected request ID %s, got %s", req.ID, dialog.request.ID)
	}

	if dialog.timeout != 60*time.Second {
		t.Errorf("Expected timeout %v, got %v", 60*time.Second, dialog.timeout)
	}

	if dialog.IsVisible() {
		t.Error("Dialog should not be visible initially")
	}
}

func TestApprovalDialog_Show_Approve(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 5*time.Second)

	// Start dialog in background
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)
	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait for dialog to be ready (use channel-based synchronization)
	select {
	case <-time.After(50 * time.Millisecond):
		// Dialog should be ready by now
	case <-ctx.Done():
		t.Fatal("Context cancelled before dialog ready")
	}

	// Approve the dialog
	dialog.Approve()

	// Wait for result
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
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 5*time.Second)

	// Start dialog in background
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)
	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait a bit for dialog to start
	time.Sleep(10 * time.Millisecond)

	// Deny the dialog
	dialog.Deny()

	// Wait for result
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

func TestApprovalDialog_Show_Timeout(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 100*time.Millisecond)

	// Start dialog and wait for timeout
	ctx := context.Background()
	result := dialog.Show(ctx)

	if result.Approved {
		t.Error("Expected denial on timeout, got approval")
	}
	if result.RequestID != req.ID {
		t.Errorf("Expected request ID %s, got %s", req.ID, result.RequestID)
	}
	if result.Reason != "timeout" {
		t.Errorf("Expected reason 'timeout', got %s", result.Reason)
	}
}

func TestApprovalDialog_HandleKey(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 5*time.Second)

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
			closed := dialog.HandleKey(string(tt.key))
			if closed != tt.shouldClose {
				t.Errorf("HandleKey(%c) returned %v, expected %v", tt.key, closed, tt.shouldClose)
			}
		})
	}
}

func TestApprovalDialog_GetRemainingTime(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 1*time.Second)

	// Initially not visible
	if dialog.GetRemainingTime() != 0 {
		t.Error("Expected 0 remaining time when not visible")
	}

	// Start dialog
	ctx := context.Background()
	go dialog.Show(ctx)
	time.Sleep(10 * time.Millisecond)

	// Should have remaining time
	remaining := dialog.GetRemainingTime()
	if remaining <= 0 || remaining > 1*time.Second {
		t.Errorf("Expected remaining time between 0 and 1s, got %v", remaining)
	}

	// Wait for timeout
	time.Sleep(1100 * time.Millisecond)
	if dialog.GetRemainingTime() != 0 {
		t.Error("Expected 0 remaining time after timeout")
	}
}

func TestApprovalDialog_Render(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user/project",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 60*time.Second)

	// Not visible - should return empty string
	if dialog.Render(80, 24) != "" {
		t.Error("Expected empty string when dialog not visible")
	}

	// Dialog is visible by default after creation
	// (removed private field access)

	// Render dialog
	output := dialog.Render(80, 24)
	if output == "" {
		t.Error("Expected non-empty output when dialog visible")
	}

	// Check for key elements
	if !strings.Contains(output, "Approval Required") {
		t.Error("Expected 'Approval Required' in output")
	}
	if !strings.Contains(output, "rm -rf /tmp/test") {
		t.Error("Expected command in output")
	}
	if !strings.Contains(output, "Destructive file operation") {
		t.Error("Expected reason in output")
	}
	if !strings.Contains(output, "/home/user/project") {
		t.Error("Expected workdir in output")
	}
	if !strings.Contains(output, "[A]pprove") {
		t.Error("Expected approve shortcut in output")
	}
	if !strings.Contains(output, "[D]eny") {
		t.Error("Expected deny shortcut in output")
	}
}

func TestApprovalDialog_Render_LongContent(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /very/long/path/that/exceeds/normal/terminal/width/and/should/be/truncated"},
		Reason:    "This is a very long reason that should be truncated when it exceeds the dialog width",
		WorkDir:   "/home/user/very/long/project/path/that/might/also/be/truncated",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 60*time.Second)

	// Dialog is visible by default after creation
	// (removed private field access)

	// Render with small width
	output := dialog.Render(40, 24)
	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Should contain truncated content
	if !strings.Contains(output, "...") {
		t.Error("Expected truncated content with '...'")
	}
}

func TestApprovalDialog_ConcurrentAccess(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 5*time.Second)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test IsVisible
			_ = dialog.IsVisible()

			// Test GetRemainingTime
			_ = dialog.GetRemainingTime()

			// Test HandleKey
			_ = dialog.HandleKey("x")
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
}

func TestApprovalDialog_MultipleResponses(t *testing.T) {
	req := security.ApprovalRequest{
		ID:        "test-id",
		Command:   &security.Command{Raw: "rm -rf /tmp/test"},
		Reason:    "Destructive file operation",
		WorkDir:   "/home/user",
		Timestamp: time.Now(),
	}

	dialog := NewApprovalDialog(req, 5*time.Second)

	// Start dialog
	ctx := context.Background()
	resultCh := make(chan security.ApprovalResponse, 1)
	go func() {
		resultCh <- dialog.Show(ctx)
	}()

	// Wait for dialog to be ready
	select {
	case <-time.After(50 * time.Millisecond):
		// Dialog should be ready by now
	case <-ctx.Done():
		t.Fatal("Context cancelled before dialog ready")
	}

	// Send multiple responses
	dialog.Approve()
	dialog.Deny()
	dialog.Approve()

	// Should only get first response
	select {
	case result := <-resultCh:
		if !result.Approved {
			t.Error("Expected first response (approve) to be returned")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Dialog did not complete within timeout")
	}
}
