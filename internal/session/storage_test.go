package session

import (
	"os"
	"path/filepath"
	"testing"
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

	err = storage.Save(session.ID, *session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify session exists
	exists, err := storage.Exists(session.ID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Session was not saved")
	}
}

func TestFileStorage_Save_Overwrite(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Save session
	err = storage.Save(session.ID, *session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Modify and save again
	session.SetTitle("Updated Title")
	err = storage.Save(session.ID, *session)
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

	// Empty key should fail
	err = storage.Save("", *session)
	if err == nil {
		t.Error("Save() should return error for empty key")
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
	err = storage.Save(original.ID, *original)
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

// Test Delete

func TestFileStorage_Delete(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	session := NewSession("/test/workdir")

	// Save session
	err = storage.Save(session.ID, *session)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Delete session
	err = storage.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify session was deleted
	exists, _ := storage.Exists(session.ID)
	if exists {
		t.Error("Session was not deleted")
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
	err = storage.Save(session.ID, *session)
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
		err = storage.Save(session.ID, *session)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// List all sessions (returns keys)
	keys, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(keys) != 5 {
		t.Errorf("List() returned %d sessions, want 5", len(keys))
	}
}

func TestFileStorage_List_EmptyStorage(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	keys, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("List() returned %d sessions, want 0", len(keys))
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
			err := storage.Save(session.ID, *session)
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
	keys, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(keys) != 10 {
		t.Errorf("List() returned %d sessions, want 10", len(keys))
	}
}

func TestFileStorage_ConcurrentReads(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create a session
	session := NewSession("/test/workdir")
	err = storage.Save(session.ID, *session)
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
