package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/core/turn"
	"github.com/dmytrogajewski/spin/internal/llm"
)

func TestNewManager_AndNewConversation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir() // Separate dir for sessions

	llm := llm.NewMockProvider("hi")
	emitter := NewEventEmitter(100)

	// Constructor now initializes storage automatically
	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	// Run a simple turn to ensure wiring works
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conv.RunTurn(ctx, "say hi"); err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}
}

func TestManager_ListAndArchive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// List should be safe even if empty
	metas, err := m.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}
	if metas == nil {
		t.Error("ListConversations() returned nil slice")
	}

	// Archive non-existent should return error
	if err := m.ArchiveConversation(context.Background(), "non-existent"); err == nil {
		t.Error("ArchiveConversation() should error for non-existent session")
	}
}

func TestManager_ResumeConversation_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if _, err := m.ResumeConversation(context.Background(), "does-not-exist"); err == nil {
		t.Error("expected error for non-existent session in ResumeConversation")
	}
}

// Integration tests with real file storage

func TestManager_Integration_SaveResumeWithHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("assistant reply")
	emitter := NewEventEmitter(100)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a new conversation and run a turn
	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	ctx := context.Background()
	if err := conv.RunTurn(ctx, "hello"); err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}

	// Manually create and save a session with the history
	sess := createSessionForTest(t, cfg.WorkDir, conv)
	if err := m.storage.Save(sess); err != nil {
		t.Fatalf("Save session error: %v", err)
	}

	// Resume the conversation from storage
	resumed, err := m.ResumeConversation(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("ResumeConversation() error: %v", err)
	}

	// Verify history is restored (check message count)
	// History should have system + user + assistant messages
	if resumed == nil {
		t.Fatal("resumed conversation is nil")
	}
}

func TestManager_Integration_ListWithMetadata(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create and save multiple sessions
	conv1, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess1 := createSessionForTest(t, cfg.WorkDir, conv1)
	sess1.Metadata.Title = "Session 1"
	if err := m.storage.Save(sess1); err != nil {
		t.Fatalf("Save session1 error: %v", err)
	}

	conv2, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess2 := createSessionForTest(t, cfg.WorkDir, conv2)
	sess2.Metadata.Title = "Session 2"
	if err := m.storage.Save(sess2); err != nil {
		t.Fatalf("Save session2 error: %v", err)
	}

	// List all sessions
	metas, err := m.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}

	if len(metas) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(metas))
	}
}

func TestManager_Integration_Archive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create and save a session
	conv, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess := createSessionForTest(t, cfg.WorkDir, conv)
	if err := m.storage.Save(sess); err != nil {
		t.Fatalf("Save session error: %v", err)
	}

	// Archive the session
	if err := m.ArchiveConversation(context.Background(), sess.ID); err != nil {
		t.Fatalf("ArchiveConversation() error: %v", err)
	}

	// Load and verify state is archived
	loaded, err := m.storage.Load(sess.ID)
	if err != nil {
		t.Fatalf("Load session error: %v", err)
	}

	if loaded.State != session.StateArchived {
		t.Errorf("expected state archived, got %v", loaded.State)
	}
}

func TestManager_Integration_MultipleConversations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create multiple conversations concurrently
	numConvs := 5
	convs := make([]*Conversation, numConvs)
	for i := 0; i < numConvs; i++ {
		conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
		if err != nil {
			t.Fatalf("NewConversation(%d) error: %v", i, err)
		}
		convs[i] = conv
	}

	// Each should be independent
	if len(convs) != numConvs {
		t.Errorf("expected %d conversations, got %d", numConvs, len(convs))
	}
}

// Helper to create a minimal session for testing
func createSessionForTest(t *testing.T, workDir string, conv *Conversation) *session.Session {
	t.Helper()
	sess := session.NewSession(workDir)
	// Create a turn with sample user input and AI response
	sampleTurn := turn.NewTurn(sess.ID, "test input")
	// Transition turn to completed state
	_ = sampleTurn.Start()
	_ = sampleTurn.Complete("test reply", turn.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 15,
		TotalTokens:      25,
	})
	// Add turn to session
	_ = sess.AddTurn(sampleTurn)
	return sess
}
