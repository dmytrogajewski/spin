package agent

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/message"
)

// TestRequest_UsesCanonicalMessage verifies Request uses message.Message.
func TestRequest_UsesCanonicalMessage(t *testing.T) {
	t.Parallel()
	msg := message.Message{
		ID:        "test-1",
		Role:      message.RoleUser,
		Content:   "test content",
		Timestamp: time.Now(),
	}

	req := Request{
		Input:   "test input",
		History: []message.Message{msg},
	}

	if req.Input != "test input" {
		t.Errorf("expected input 'test input', got %q", req.Input)
	}

	if len(req.History) != 1 {
		t.Fatalf("expected 1 message in history, got %d", len(req.History))
	}

	if req.History[0].ID != "test-1" {
		t.Errorf("expected message ID 'test-1', got %q", req.History[0].ID)
	}

	if req.History[0].Content != "test content" {
		t.Errorf("expected content 'test content', got %q", req.History[0].Content)
	}
}
