package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/session"
)

// Simple integration tests for end-to-end flows

// TestIntegration_ManagerConversationFlow tests basic manager + conversation interaction
func TestIntegration_ManagerConversationFlow(t *testing.T) {
	// Setup
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock-model"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mockLLM := llm.NewMockProvider("Test response")
	emitter := NewEventEmitter(100)

	mgr, err := NewManager(cfg, WithLLM(mockLLM), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create conversation
	conv, err := mgr.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	// Run a turn
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = conv.RunTurn(ctx, "List files")
	if err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}

	// Verify state
	if conv.State() == StateFailed {
		t.Error("Conversation should not be in failed state")
	}
}

// TestIntegration_SessionPersistence tests session save and load
func TestIntegration_SessionPersistence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mockLLM := llm.NewMockProvider("Response")
	emitter := NewEventEmitter(100)

	// Create manager and conversation
	mgr, err := NewManager(cfg, WithLLM(mockLLM), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	conv, err := mgr.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	// Run a turn
	err = conv.RunTurn(context.Background(), "Test input")
	if err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}

	// Create and save session
	sess := session.NewSession(cfg.WorkDir)
	sessionID := sess.ID

	if err := mgr.storage.Save(sess); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load session
	loaded, err := mgr.storage.Load(sessionID)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.ID != sessionID {
		t.Errorf("Loaded session ID = %s, want %s", loaded.ID, sessionID)
	}
}

// TestIntegration_ListAndArchive tests conversation listing and archiving
func TestIntegration_ListAndArchive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mockLLM := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	mgr, err := NewManager(cfg, WithLLM(mockLLM), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create and save a session
	sess := session.NewSession(cfg.WorkDir)
	if err := mgr.storage.Save(sess); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// List sessions
	sessions, err := mgr.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	// Archive session
	if err := mgr.ArchiveConversation(context.Background(), sess.ID); err != nil {
		t.Fatalf("ArchiveConversation() error: %v", err)
	}

	// Verify archived
	loaded, err := mgr.storage.Load(sess.ID)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.State != session.StateArchived {
		t.Errorf("expected archived state, got %v", loaded.State)
	}
}

// TestIntegration_ConcurrentConversations tests creating multiple conversations
func TestIntegration_ConcurrentConversations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mockLLM := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)

	mgr, err := NewManager(cfg, WithLLM(mockLLM), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create multiple conversations
	numConvs := 3
	convs := make([]*Conversation, numConvs)

	for i := 0; i < numConvs; i++ {
		conv, err := mgr.NewConversation(context.Background(), cfg.WorkDir)
		if err != nil {
			t.Fatalf("NewConversation(%d) error: %v", i, err)
		}
		convs[i] = conv
	}

	// Verify all created
	if len(convs) != numConvs {
		t.Errorf("expected %d conversations, got %d", numConvs, len(convs))
	}
}
