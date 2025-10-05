package session

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test FileStorage Creation

func TestNewFileStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	if storage == nil {
		t.Fatal("NewFileStorage() returned nil")
	}
}

func TestNewFileStorage_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")

	_, err := NewFileStorage(sessionDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Check directory was created
	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Path is not a directory")
	}
}

func TestNewFileStorage_InvalidPath(t *testing.T) {
	// Try to create storage in a file (not directory)
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0600)

	_, err := NewFileStorage(tmpFile)
	if err == nil {
		t.Error("NewFileStorage() should return error for invalid path")
	}
}

// Test Save

func TestFileStorage_Save(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file was created
	path := storage.sessionPath(session.ID)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Session file was not created: %v", err)
	}
}

func TestFileStorage_Save_AtomicWrite(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Save session
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check no temp files remain
	tmpPath := storage.sessionPath(session.ID) + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temporary file was not cleaned up")
	}
}

func TestFileStorage_Save_Overwrite(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Save session
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Modify and save again
	session.SetTitle("Updated Title")
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load and verify
	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Metadata.Title != "Updated Title" {
		t.Errorf("Title = %s, want 'Updated Title'", loaded.Metadata.Title)
	}
}

func TestFileStorage_Save_InvalidSession(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")
	session.ID = "" // Invalid

	err = storage.Save(session)
	if err == nil {
		t.Error("Save() should return error for invalid session")
	}
}

// Test Load

func TestFileStorage_Load(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	original := NewSession("/test/workdir")
	original.SetTitle("Test Session")

	// Save session
	err = storage.Save(original)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load session
	loaded, err := storage.Load(original.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != original.ID {
		t.Errorf("ID = %s, want %s", loaded.ID, original.ID)
	}

	if loaded.WorkDir != original.WorkDir {
		t.Errorf("WorkDir = %s, want %s", loaded.WorkDir, original.WorkDir)
	}

	if loaded.Metadata.Title != original.Metadata.Title {
		t.Errorf("Title = %s, want %s", loaded.Metadata.Title, original.Metadata.Title)
	}
}

func TestFileStorage_Load_NotFound(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	_, err = storage.Load("nonexistent-id")
	if err == nil {
		t.Error("Load() should return error for nonexistent session")
	}
}

func TestFileStorage_Load_CorruptedData(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create corrupted session file
	sessionID := "corrupted-session"
	path := storage.sessionPath(sessionID)
	os.WriteFile(path, []byte("invalid json {{{"), 0600)

	_, err = storage.Load(sessionID)
	if err == nil {
		t.Error("Load() should return error for corrupted data")
	}
}

// Test Delete

func TestFileStorage_Delete(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Save session
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Delete session
	err = storage.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify file was deleted
	path := storage.sessionPath(session.ID)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Session file was not deleted")
	}
}

func TestFileStorage_Delete_NotFound(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Deleting non-existent session should not error
	err = storage.Delete("nonexistent-id")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil", err)
	}
}

// Test Exists

func TestFileStorage_Exists(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Should not exist initially
	exists, err := storage.Exists(session.ID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}

	// Save session
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Should exist now
	exists, err = storage.Exists(session.ID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}
}

// Test List

func TestFileStorage_List(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create and save multiple sessions
	for i := 0; i < 5; i++ {
		session := NewSession("/test/workdir")
		err = storage.Save(session)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List all sessions
	ids, err := storage.List(Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 5 {
		t.Errorf("List() returned %d sessions, want 5", len(ids))
	}
}

func TestFileStorage_List_EmptyStorage(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	ids, err := storage.List(Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 0 {
		t.Errorf("List() returned %d sessions, want 0", len(ids))
	}
}

// Test ListMetadata

func TestFileStorage_ListMetadata(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create sessions with different metadata
	for i := 0; i < 3; i++ {
		session := NewSession("/test/workdir")
		session.SetTitle(fmt.Sprintf("Session %d", i))
		err = storage.Save(session)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List metadata
	metadata, err := storage.ListMetadata(Filter{})
	if err != nil {
		t.Fatalf("ListMetadata() error = %v", err)
	}

	if len(metadata) != 3 {
		t.Errorf("ListMetadata() returned %d items, want 3", len(metadata))
	}
}

// Test Filtering

func TestFileStorage_List_WithFilter_State(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create sessions with different states
	s1 := NewSession("/test/workdir")
	s1.SetState(StateActive)
	storage.Save(s1)

	s2 := NewSession("/test/workdir")
	s2.SetState(StateCompleted)
	storage.Save(s2)

	s3 := NewSession("/test/workdir")
	s3.SetState(StateCompleted)
	storage.Save(s3)

	// Filter by completed state
	completedState := StateCompleted
	filter := Filter{State: &completedState}

	ids, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("List() returned %d sessions, want 2", len(ids))
	}
}

func TestFileStorage_List_WithFilter_WorkDir(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create sessions with different work directories
	s1 := NewSession("/project/a")
	storage.Save(s1)

	s2 := NewSession("/project/b")
	storage.Save(s2)

	s3 := NewSession("/project/a")
	storage.Save(s3)

	// Filter by work directory
	filter := Filter{WorkDir: "/project/a"}

	ids, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("List() returned %d sessions, want 2", len(ids))
	}
}

func TestFileStorage_List_WithFilter_Date(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	// Create sessions with different timestamps
	s1 := NewSession("/test/workdir")
	s1.CreatedAt = twoHoursAgo
	s1.UpdatedAt = twoHoursAgo
	storage.Save(s1)

	s2 := NewSession("/test/workdir")
	s2.CreatedAt = oneHourAgo
	s2.UpdatedAt = oneHourAgo
	storage.Save(s2)

	s3 := NewSession("/test/workdir")
	// s3 has current timestamp
	storage.Save(s3)

	// Filter by created after one hour ago
	filter := Filter{CreatedAfter: &oneHourAgo}

	ids, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// Should return s2 and s3
	if len(ids) < 1 {
		t.Errorf("List() returned %d sessions, want at least 1", len(ids))
	}
}

func TestFileStorage_List_WithFilter_Tags(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create sessions with different tags
	s1 := NewSession("/test/workdir")
	s1.AddTag("auth")
	storage.Save(s1)

	s2 := NewSession("/test/workdir")
	s2.AddTag("database")
	storage.Save(s2)

	s3 := NewSession("/test/workdir")
	s3.AddTag("auth")
	s3.AddTag("api")
	storage.Save(s3)

	// Filter by auth tag
	filter := Filter{Tags: []string{"auth"}}

	ids, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("List() returned %d sessions, want 2", len(ids))
	}
}

func TestFileStorage_List_WithPagination(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create 10 sessions
	for i := 0; i < 10; i++ {
		session := NewSession("/test/workdir")
		err = storage.Save(session)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Get first page (5 items)
	filter := Filter{Limit: 5, Offset: 0}
	page1, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(page1) != 5 {
		t.Errorf("List() returned %d sessions, want 5", len(page1))
	}

	// Get second page (5 items)
	filter.Offset = 5
	page2, err := storage.List(filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(page2) != 5 {
		t.Errorf("List() returned %d sessions, want 5", len(page2))
	}

	// Pages should not overlap
	for _, id1 := range page1 {
		for _, id2 := range page2 {
			if id1 == id2 {
				t.Error("Pagination returned duplicate session IDs")
			}
		}
	}
}

// Test Concurrent Access

func TestFileStorage_ConcurrentSaves(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	done := make(chan bool)

	// Start multiple concurrent savers
	for i := 0; i < 10; i++ {
		go func() {
			session := NewSession("/test/workdir")
			err := storage.Save(session)
			if err != nil {
				t.Errorf("Save() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all savers to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have 10 sessions
	ids, err := storage.List(Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(ids) != 10 {
		t.Errorf("List() returned %d sessions, want 10", len(ids))
	}
}

func TestFileStorage_ConcurrentReads(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create a session
	session := NewSession("/test/workdir")
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	done := make(chan bool)

	// Start multiple concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			_, err := storage.Load(session.ID)
			if err != nil {
				t.Errorf("Load() error = %v", err)
			}
			done <- true
		}()
	}

	// Wait for all readers to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoad_Standalone(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-session"

	// Create a test session file
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
	}
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test the Load function
	loaded, err := Load(storage, sessionID)
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if loaded.ID != sessionID {
		t.Errorf("Load() ID = %v, want %v", loaded.ID, sessionID)
	}
}

func TestDelete_Standalone(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-session"

	// Create a test session
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
	}
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test the Delete function
	err = Delete(storage, sessionID)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify session is deleted
	_, err = storage.Load(sessionID)
	if err == nil {
		t.Error("Expected error loading deleted session")
	}
}

func TestExists_Standalone(t *testing.T) {
	tmpDir := t.TempDir()
	sessionID := "test-session"

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Initially should not exist
	exists, err := Exists(storage, sessionID)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}

	// Create a session
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
	}
	err = storage.Save(session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Now should exist
	exists, err = Exists(storage, sessionID)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}
}
