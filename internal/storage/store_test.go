package storage

import (
	"testing"
)

// TestData is a simple struct for testing.
type TestData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestFileStore_SaveLoad(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	data := TestData{ID: "test-1", Name: "Test", Value: 42}

	// Save
	if err := store.Save("test-1", data); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := store.Load("test-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != data.ID || loaded.Name != data.Name || loaded.Value != data.Value {
		t.Errorf("Load() = %+v, want %+v", loaded, data)
	}
}

func TestFileStore_Delete(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	data := TestData{ID: "test-1", Name: "Test", Value: 42}
	store.Save("test-1", data)

	// Delete
	if err := store.Delete("test-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Should not exist
	exists, _ := store.Exists("test-1")
	if exists {
		t.Error("Exists() = true after Delete(), want false")
	}
}

func TestFileStore_Exists(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	// Should not exist initially
	exists, err := store.Exists("test-1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}

	// Save and check
	store.Save("test-1", TestData{ID: "test-1"})

	exists, err = store.Exists("test-1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}
}

func TestFileStore_List(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	// Save multiple items
	for i := 0; i < 5; i++ {
		store.Save(string(rune('a'+i)), TestData{ID: string(rune('a' + i))})
	}

	keys, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(keys) != 5 {
		t.Errorf("List() returned %d keys, want 5", len(keys))
	}
}

func TestFileStore_CustomSuffix(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
		Suffix:  ".history.json",
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	data := TestData{ID: "test-1", Name: "Test", Value: 42}
	store.Save("test-1", data)

	// Load should work
	loaded, err := store.Load("test-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != data.ID {
		t.Errorf("Load() ID = %s, want %s", loaded.ID, data.ID)
	}
}

func TestFileStore_EmptyKey(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	// All operations should fail with empty key
	if err := store.Save("", TestData{}); err == nil {
		t.Error("Save() with empty key should error")
	}

	if _, err := store.Load(""); err == nil {
		t.Error("Load() with empty key should error")
	}

	if err := store.Delete(""); err == nil {
		t.Error("Delete() with empty key should error")
	}

	if _, err := store.Exists(""); err == nil {
		t.Error("Exists() with empty key should error")
	}
}

func TestFileStore_NotFound(t *testing.T) {
	store, err := NewFileStore[TestData](FileStoreConfig{
		BaseDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	_, err = store.Load("nonexistent")
	if err == nil {
		t.Error("Load() nonexistent key should error")
	}
}
