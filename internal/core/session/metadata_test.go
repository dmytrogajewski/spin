package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/core/turn"
)

// Test Metadata Initialization

func TestMetadata_DefaultValues(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	if session.Metadata.Title != "" {
		t.Errorf("Title = %s, want empty string", session.Metadata.Title)
	}

	if session.Metadata.Description != "" {
		t.Errorf("Description = %s, want empty string", session.Metadata.Description)
	}

	if len(session.Metadata.Tags) != 0 {
		t.Errorf("Tags length = %d, want 0", len(session.Metadata.Tags))
	}

	if session.Metadata.TotalTurns != 0 {
		t.Errorf("TotalTurns = %d, want 0", session.Metadata.TotalTurns)
	}

	if session.Metadata.TokensUsed != 0 {
		t.Errorf("TokensUsed = %d, want 0", session.Metadata.TokensUsed)
	}

	if session.Metadata.LastError != "" {
		t.Errorf("LastError = %s, want empty string", session.Metadata.LastError)
	}
}

// Test Token Tracking

func TestMetadata_TokenTracking(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	// Add turns with different token counts
	_ = session.AddTurn(&turn.Turn{
		ID:        "turn-1",
		SessionID: session.ID,
		Tokens:    turn.TokenUsage{TotalTokens: 100},
	})

	_ = session.AddTurn(&turn.Turn{
		ID:        "turn-2",
		SessionID: session.ID,
		Tokens:    turn.TokenUsage{TotalTokens: 250},
	})

	_ = session.AddTurn(&turn.Turn{
		ID:        "turn-3",
		SessionID: session.ID,
		Tokens:    turn.TokenUsage{TotalTokens: 150},
	})

	expectedTokens := 100 + 250 + 150
	if session.Metadata.TokensUsed != expectedTokens {
		t.Errorf("TokensUsed = %d, want %d", session.Metadata.TokensUsed, expectedTokens)
	}
}

// Test Turn Count Consistency

func TestMetadata_TurnCountConsistency(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	// Add multiple turns
	for i := 0; i < 10; i++ {
		_ = session.AddTurn(&turn.Turn{
			ID:        fmt.Sprintf("turn-%d", i),
			SessionID: session.ID,
		})
	}

	// Metadata should match actual turn count
	if session.Metadata.TotalTurns != len(session.Turns) {
		t.Errorf("Metadata.TotalTurns = %d, actual turns = %d",
			session.Metadata.TotalTurns, len(session.Turns))
	}

	if session.TurnCount() != session.Metadata.TotalTurns {
		t.Errorf("TurnCount() = %d, Metadata.TotalTurns = %d",
			session.TurnCount(), session.Metadata.TotalTurns)
	}
}

// Test Title Management

func TestMetadata_SetTitle_Valid(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.SetTitle("My Project Session")
	if err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if session.Metadata.Title != "My Project Session" {
		t.Errorf("Title = %s, want 'My Project Session'", session.Metadata.Title)
	}
}

func TestMetadata_SetTitle_UpdatesTimestamp(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	originalUpdatedAt := session.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	err := session.SetTitle("Updated Title")
	if err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if session.UpdatedAt.Equal(originalUpdatedAt) {
		t.Error("UpdatedAt was not updated after SetTitle()")
	}
}

// Test Tag Management

func TestMetadata_AddTag_Single(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.AddTag("backend")
	if err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1", len(session.Metadata.Tags))
	}

	if session.Metadata.Tags[0] != "backend" {
		t.Errorf("Tag = %s, want 'backend'", session.Metadata.Tags[0])
	}
}

func TestMetadata_AddTag_Multiple(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	tags := []string{"backend", "api", "database", "auth"}

	for _, tag := range tags {
		err := session.AddTag(tag)
		if err != nil {
			t.Fatalf("AddTag() error = %v", err)
		}
	}

	if len(session.Metadata.Tags) != len(tags) {
		t.Errorf("Tags length = %d, want %d", len(session.Metadata.Tags), len(tags))
	}
}

func TestMetadata_AddTag_Duplicate(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	// Add same tag twice
	_ = session.AddTag("backend")
	_ = session.AddTag("backend")

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1 (duplicates should be ignored)",
			len(session.Metadata.Tags))
	}
}

func TestMetadata_RemoveTag_Existing(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	_ = session.AddTag("backend")
	_ = session.AddTag("frontend")
	_ = session.AddTag("database")

	err := session.RemoveTag("frontend")
	if err != nil {
		t.Fatalf("RemoveTag() error = %v", err)
	}

	if len(session.Metadata.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(session.Metadata.Tags))
	}

	// Verify frontend was removed
	for _, tag := range session.Metadata.Tags {
		if tag == "frontend" {
			t.Error("Tag 'frontend' was not removed")
		}
	}
}

func TestMetadata_RemoveTag_NonExistent(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	_ = session.AddTag("backend")

	// Removing non-existent tag should not error
	err := session.RemoveTag("nonexistent")
	if err != nil {
		t.Errorf("RemoveTag() error = %v, want nil", err)
	}

	// Original tag should still be there
	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1", len(session.Metadata.Tags))
	}
}

// Test Last Error Tracking

func TestMetadata_LastError(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	// Initially empty
	if session.Metadata.LastError != "" {
		t.Errorf("LastError = %s, want empty string", session.Metadata.LastError)
	}

	// Set error
	err := session.UpdateMetadata(func(m *Metadata) {
		m.LastError = "command execution failed"
	})

	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if session.Metadata.LastError != "command execution failed" {
		t.Errorf("LastError = %s, want 'command execution failed'",
			session.Metadata.LastError)
	}
}

// Test Metadata Update Callback

func TestMetadata_UpdateMetadata_Multiple(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.UpdateMetadata(func(m *Metadata) {
		m.Title = "Test Session"
		m.Description = "Testing metadata updates"
		m.Tags = []string{"test", "metadata"}
	})

	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if session.Metadata.Title != "Test Session" {
		t.Errorf("Title = %s, want 'Test Session'", session.Metadata.Title)
	}

	if session.Metadata.Description != "Testing metadata updates" {
		t.Errorf("Description = %s, want 'Testing metadata updates'",
			session.Metadata.Description)
	}

	if len(session.Metadata.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(session.Metadata.Tags))
	}
}

func TestMetadata_UpdateMetadata_UpdatesTimestamp(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	originalUpdatedAt := session.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	err := session.UpdateMetadata(func(m *Metadata) {
		m.Description = "Updated"
	})

	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if session.UpdatedAt.Equal(originalUpdatedAt) {
		t.Error("UpdatedAt was not updated after UpdateMetadata()")
	}
}

// Test Description

func TestMetadata_SetDescription(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	description := "This session implements user authentication"

	err := session.UpdateMetadata(func(m *Metadata) {
		m.Description = description
	})

	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if session.Metadata.Description != description {
		t.Errorf("Description = %s, want %s", session.Metadata.Description, description)
	}
}
