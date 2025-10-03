package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/core/turn"
)

// Test Session Creation

func TestNewSession(t *testing.T) {
	cfg := core.DefaultConfig()
	workDir := "/test/workdir"

	session := NewSession(workDir, cfg)

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

	if session.Turns == nil {
		t.Error("Turns slice is nil")
	}

	if len(session.Turns) != 0 {
		t.Errorf("Turns length = %d, want 0", len(session.Turns))
	}

	if session.Version != CurrentSchemaVersion {
		t.Errorf("Version = %d, want %d", session.Version, CurrentSchemaVersion)
	}
}

func TestNewSession_GeneratesUniqueIDs(t *testing.T) {
	cfg := core.DefaultConfig()
	workDir := "/test/workdir"

	s1 := NewSession(workDir, cfg)
	s2 := NewSession(workDir, cfg)

	if s1.ID == s2.ID {
		t.Error("NewSession() generated duplicate IDs")
	}
}

func TestNewSession_CopiesConfig(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.MaxTurns = 10
	workDir := "/test/workdir"

	session := NewSession(workDir, cfg)

	// Modify original config
	cfg.MaxTurns = 20

	// Session config should be unchanged
	if session.Config.MaxTurns != 10 {
		t.Errorf("Config was not copied, MaxTurns = %d, want 10", session.Config.MaxTurns)
	}
}

func TestNewSession_InitializesMetadata(t *testing.T) {
	cfg := core.DefaultConfig()
	workDir := "/test/workdir"

	session := NewSession(workDir, cfg)

	if session.Metadata.TotalTurns != 0 {
		t.Errorf("Metadata.TotalTurns = %d, want 0", session.Metadata.TotalTurns)
	}

	if session.Metadata.TokensUsed != 0 {
		t.Errorf("Metadata.TokensUsed = %d, want 0", session.Metadata.TokensUsed)
	}
}

// Test Turn Management

func TestSession_AddTurn(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	originalUpdatedAt := session.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	turn := &turn.Turn{
		ID:        "turn-1",
		SessionID: session.ID,
		UserInput: "Test input",
		Tokens:    turn.TokenUsage{TotalTokens: 100},
	}

	err := session.AddTurn(turn)
	if err != nil {
		t.Fatalf("AddTurn() error = %v", err)
	}

	if len(session.Turns) != 1 {
		t.Errorf("Turns length = %d, want 1", len(session.Turns))
	}

	if session.Turns[0] != turn {
		t.Error("Added turn does not match")
	}

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

func TestSession_AddTurn_NilTurn(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.AddTurn(nil)
	if err == nil {
		t.Error("AddTurn(nil) should return error")
	}

	if len(session.Turns) != 0 {
		t.Errorf("Turns length = %d, want 0", len(session.Turns))
	}
}

func TestSession_AddTurn_Multiple(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	for i := 1; i <= 5; i++ {
		turn := &turn.Turn{
			ID:        fmt.Sprintf("turn-%d", i),
			SessionID: session.ID,
			Tokens:    turn.TokenUsage{TotalTokens: i * 100},
		}

		err := session.AddTurn(turn)
		if err != nil {
			t.Fatalf("AddTurn() error = %v", err)
		}
	}

	if len(session.Turns) != 5 {
		t.Errorf("Turns length = %d, want 5", len(session.Turns))
	}

	if session.Metadata.TotalTurns != 5 {
		t.Errorf("Metadata.TotalTurns = %d, want 5", session.Metadata.TotalTurns)
	}

	expectedTokens := 100 + 200 + 300 + 400 + 500
	if session.Metadata.TokensUsed != expectedTokens {
		t.Errorf("Metadata.TokensUsed = %d, want %d", session.Metadata.TokensUsed, expectedTokens)
	}
}

func TestSession_GetTurn(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	turn1 := &turn.Turn{ID: "turn-1", SessionID: session.ID}
	turn2 := &turn.Turn{ID: "turn-2", SessionID: session.ID}

	_ = session.AddTurn(turn1)
	_ = session.AddTurn(turn2)

	got, err := session.GetTurn("turn-1")
	if err != nil {
		t.Fatalf("GetTurn() error = %v", err)
	}

	if got != turn1 {
		t.Error("GetTurn() returned wrong turn")
	}
}

func TestSession_GetTurn_NotFound(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	_, err := session.GetTurn("nonexistent")
	if err == nil {
		t.Error("GetTurn() should return error for nonexistent turn")
	}
}

func TestSession_LastTurn(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	turn1 := &turn.Turn{ID: "turn-1", SessionID: session.ID}
	turn2 := &turn.Turn{ID: "turn-2", SessionID: session.ID}

	_ = session.AddTurn(turn1)
	_ = session.AddTurn(turn2)

	last := session.LastTurn()
	if last != turn2 {
		t.Error("LastTurn() returned wrong turn")
	}
}

func TestSession_LastTurn_EmptySession(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	last := session.LastTurn()
	if last != nil {
		t.Error("LastTurn() should return nil for empty session")
	}
}

func TestSession_TurnCount(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	if session.TurnCount() != 0 {
		t.Errorf("TurnCount() = %d, want 0", session.TurnCount())
	}

	session.AddTurn(&turn.Turn{ID: "turn-1", SessionID: session.ID})
	session.AddTurn(&turn.Turn{ID: "turn-2", SessionID: session.ID})

	if session.TurnCount() != 2 {
		t.Errorf("TurnCount() = %d, want 2", session.TurnCount())
	}
}

// Test Metadata Operations

func TestSession_UpdateMetadata(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

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
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.SetState(StateCompleted)
	if err != nil {
		t.Fatalf("SetState() error = %v", err)
	}

	if session.State != StateCompleted {
		t.Errorf("State = %v, want %v", session.State, StateCompleted)
	}
}

func TestSession_SetState_InvalidTransition(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	// Archive the session
	err := session.SetState(StateArchived)
	if err != nil {
		t.Fatalf("SetState(StateArchived) error = %v", err)
	}

	// Try to transition back to Active (should fail)
	err = session.SetState(StateActive)
	if err == nil {
		t.Error("SetState() should return error for invalid transition from Archived")
	}
}

func TestSession_AddTag(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

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
	session := NewSession("/test/workdir", core.DefaultConfig())

	session.AddTag("test-tag")
	session.AddTag("test-tag") // Add same tag again

	if len(session.Metadata.Tags) != 1 {
		t.Errorf("Tags length = %d, want 1 (duplicates should be ignored)", len(session.Metadata.Tags))
	}
}

func TestSession_RemoveTag(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

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
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.SetTitle("My Session")
	if err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if session.Metadata.Title != "My Session" {
		t.Errorf("Title = %s, want 'My Session'", session.Metadata.Title)
	}
}

// Test Validation

func TestSession_Validate_Valid(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	err := session.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestSession_Validate_EmptyID(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	session.ID = ""

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty ID")
	}
}

func TestSession_Validate_EmptyWorkDir(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	session.WorkDir = ""

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for empty WorkDir")
	}
}

func TestSession_Validate_InvalidTimestamps(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	session.UpdatedAt = session.CreatedAt.Add(-1 * time.Hour) // UpdatedAt before CreatedAt

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error when UpdatedAt < CreatedAt")
	}
}

func TestSession_Validate_InvalidState(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())
	session.State = State("invalid")

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for invalid state")
	}
}

func TestSession_Validate_DuplicateTurnIDs(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	turn1 := &turn.Turn{ID: "turn-1", SessionID: session.ID}
	turn2 := &turn.Turn{ID: "turn-1", SessionID: session.ID} // Duplicate ID

	_ = session.AddTurn(turn1)
	_ = session.AddTurn(turn2)

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for duplicate turn IDs")
	}
}

func TestSession_Validate_InconsistentMetadata(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	session.AddTurn(&turn.Turn{ID: "turn-1", SessionID: session.ID, Tokens: turn.TokenUsage{TotalTokens: 100}})
	session.AddTurn(&turn.Turn{ID: "turn-2", SessionID: session.ID, Tokens: turn.TokenUsage{TotalTokens: 200}})

	// Manually corrupt metadata
	session.Metadata.TotalTurns = 5 // Wrong count

	err := session.Validate()
	if err == nil {
		t.Error("Validate() should return error for inconsistent metadata")
	}
}

// Test Concurrent Access

func TestSession_ConcurrentReads(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	for i := 0; i < 10; i++ {
		session.AddTurn(&turn.Turn{
			ID:        fmt.Sprintf("turn-%d", i),
			SessionID: session.ID,
		})
	}

	// Start multiple concurrent readers
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = session.TurnCount()
				_ = session.LastTurn()
			}
			done <- true
		}()
	}

	// Wait for all readers to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSession_ConcurrentWrites(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 10; j++ {
				session.AddTurn(&turn.Turn{
					ID:        fmt.Sprintf("turn-%d-%d", n, j),
					SessionID: session.ID,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all writers to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 100 turns
	if session.TurnCount() != 100 {
		t.Errorf("TurnCount() = %d, want 100", session.TurnCount())
	}
}

func TestSession_ConcurrentReadWrite(t *testing.T) {
	session := NewSession("/test/workdir", core.DefaultConfig())

	done := make(chan bool)

	// Writers
	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 20; j++ {
				session.AddTurn(&turn.Turn{
					ID:        fmt.Sprintf("turn-%d-%d", n, j),
					SessionID: session.ID,
				})
			}
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = session.TurnCount()
				_ = session.LastTurn()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly 100 turns
	if session.TurnCount() != 100 {
		t.Errorf("TurnCount() = %d, want 100", session.TurnCount())
	}
}
