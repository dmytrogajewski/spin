package session

import (
	"testing"
	"time"
)

// Test Session Creation.

func TestNewSession(t *testing.T) {
	workDir := "/test/workdir"

	session := NewSession(workDir)

	if session == nil {
		t.Fatal("NewSession() returned nil")
	}

	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.WorkDir != workDir {
		t.Errorf("WorkDir = %s, want %s", session.WorkDir, workDir)
	}

	if session.State != StateActive {
		t.Errorf("State = %v, want %v", session.State, StateActive)
	}

	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if session.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}

	if session.Version != CurrentSchemaVersion {
		t.Errorf("Version = %d, want %d", session.Version, CurrentSchemaVersion)
	}
}

func TestNewSession_GeneratesUniqueIDs(t *testing.T) {
	workDir := "/test/workdir"

	s1 := NewSession(workDir)
	s2 := NewSession(workDir)

	if s1.ID == s2.ID {
		t.Error("NewSession() generated duplicate IDs")
	}
}

func TestNewSession_InitializesMetadata(t *testing.T) {
	workDir := "/test/workdir"

	session := NewSession(workDir)

	if session.Metadata.TotalTurns != 0 {
		t.Errorf("Metadata.TotalTurns = %d, want 0", session.Metadata.TotalTurns)
	}

	if session.Metadata.TokensUsed != 0 {
		t.Errorf("Metadata.TokensUsed = %d, want 0", session.Metadata.TokensUsed)
	}
}

// Test IncrementTurnCount.

func TestSession_IncrementTurnCount(t *testing.T) {
	session := NewSession("/test/workdir")
	originalUpdatedAt := session.UpdatedAt

	// Wait a bit to ensure timestamp difference.
	time.Sleep(10 * time.Millisecond)

	session.IncrementTurnCount(100)

	if session.UpdatedAt.Equal(originalUpdatedAt) {
		t.Error("UpdatedAt was not updated")
	}

	if session.Metadata.TotalTurns != 1 {
		t.Errorf("Metadata.TotalTurns = %d, want 1", session.Metadata.TotalTurns)
	}

	if session.Metadata.TokensUsed != 100 {
		t.Errorf("Metadata.TokensUsed = %d, want 100", session.Metadata.TokensUsed)
	}
}

func TestSession_IncrementTurnCount_Multiple(t *testing.T) {
	session := NewSession("/test/workdir")

	for i := 1; i <= 5; i++ {
		session.IncrementTurnCount(i * 100)
	}

	if session.Metadata.TotalTurns != 5 {
		t.Errorf("Metadata.TotalTurns = %d, want 5", session.Metadata.TotalTurns)
	}

	expectedTokens := 100 + 200 + 300 + 400 + 500
	if session.Metadata.TokensUsed != expectedTokens {
		t.Errorf("Metadata.TokensUsed = %d, want %d", session.Metadata.TokensUsed, expectedTokens)
	}
}

// Test Metadata Operations.

func TestSession_UpdateMetadata(t *testing.T) {
	session := NewSession("/test/workdir")

	err := session.UpdateMetadata(func(m *Metadata) {
		m.Title = "Test Session"
		m.Description = "Test Description"
	})
	if err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if session.Metadata.Title != "Test Session" {
		t.Errorf("Title = %s, want 'Test Session'", session.Metadata.Title)
	}

	if session.Metadata.Description != "Test Description" {
		t.Errorf("Description = %s, want 'Test Description'", session.Metadata.Description)
	}
}

func TestSession_SetState(t *testing.T) {
	session := NewSession("/test/workdir")

	err := session.SetState(StateCompleted)
	if err != nil {
		t.Fatalf("SetState() error = %v", err)
	}

	if session.State != StateCompleted {
		t.Errorf("State = %v, want %v", session.State, StateCompleted)
	}
}

func TestSession_SetState_InvalidTransition(t *testing.T) {
	session := NewSession("/test/workdir")

	// Archive the session.
	err := session.SetState(StateArchived)
	if err != nil {
		t.Fatalf("SetState(StateArchived) error = %v", err)
	}

	// Try to transition back to Active (should fail).
	err = session.SetState(StateActive)
	if err == nil {
		t.Error("SetState() should return error for invalid transition from Archived")
	}
}

func TestSession_AddTag(t *testing.T) {
	session := NewSession("/test/workdir")

	err := session.AddTag("test-tag")
	if err != nil {
		t.Fatalf("AddTag() error = %v", err)
	}

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1", len(session.Metadata.Tags))
	}

	if session.Metadata.Tags[0] != "test-tag" {
		t.Errorf("Tag = %s, want 'test-tag'", session.Metadata.Tags[0])
	}
}

func TestSession_AddTag_Duplicate(t *testing.T) {
	session := NewSession("/test/workdir")

	session.AddTag("test-tag")
	session.AddTag("test-tag") // Add same tag again.

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1 (duplicates should be ignored)", len(session.Metadata.Tags))
	}
}

func TestSession_RemoveTag(t *testing.T) {
	session := NewSession("/test/workdir")

	session.AddTag("tag1")
	session.AddTag("tag2")

	err := session.RemoveTag("tag1")
	if err != nil {
		t.Fatalf("RemoveTag() error = %v", err)
	}

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1", len(session.Metadata.Tags))
	}

	if session.Metadata.Tags[0] != "tag2" {
		t.Errorf("Tag = %s, want 'tag2'", session.Metadata.Tags[0])
	}
}

func TestSession_SetTitle(t *testing.T) {
	session := NewSession("/test/workdir")

	err := session.SetTitle("My Session")
	if err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if session.Metadata.Title != "My Session" {
		t.Errorf("Title = %s, want 'My Session'", session.Metadata.Title)
	}
}

// Test Validation.

func TestSession_Validate_Valid(t *testing.T) {
	session := NewSession("/test/workdir")

	err := session.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestSession_Validate_EmptyID(t *testing.T) {
	session := NewSession("/test/workdir")
	session.ID = ""

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty ID")
	}
}

func TestSession_Validate_EmptyWorkDir(t *testing.T) {
	session := NewSession("/test/workdir")
	session.WorkDir = ""

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty WorkDir")
	}
}

func TestSession_Validate_InvalidTimestamps(t *testing.T) {
	session := NewSession("/test/workdir")
	session.UpdatedAt = session.CreatedAt.Add(-1 * time.Hour) // UpdatedAt before CreatedAt.

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error when UpdatedAt < CreatedAt")
	}
}

func TestSession_Validate_InvalidState(t *testing.T) {
	session := NewSession("/test/workdir")
	session.State = "invalid-state" // Invalid state value.

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid state")
	}
}

// Test Concurrent Access.

func TestSession_ConcurrentMetadataUpdates(t *testing.T) {
	session := NewSession("/test/workdir")

	done := make(chan bool)

	for i := range 10 {
		go func(n int) {
			for range 10 {
				session.IncrementTurnCount(100)
			}

			done <- true
		}(i)
	}

	// Wait for all writers to complete.
	for range 10 {
		<-done
	}

	// Should have exactly 100 turn increments.
	if session.Metadata.TotalTurns != 100 {
		t.Errorf("TotalTurns = %d, want 100", session.Metadata.TotalTurns)
	}

	if session.Metadata.TokensUsed != 10000 {
		t.Errorf("TokensUsed = %d, want 10000", session.Metadata.TokensUsed)
	}
}

func TestSession_ConcurrentTagOperations(t *testing.T) {
	session := NewSession("/test/workdir")

	done := make(chan bool)

	// Add tags concurrently.
	for i := range 5 {
		go func(n int) {
			for range 10 {
				session.AddTag("tag")
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines.
	for range 5 {
		<-done
	}

	// Should have exactly 1 tag (duplicates ignored).
	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1", len(session.Metadata.Tags))
	}
}
