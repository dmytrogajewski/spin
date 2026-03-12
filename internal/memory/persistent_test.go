package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewPersistentStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewPersistentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewPersistentStore failed: %v", err)
	}

	if store == nil {
		t.Fatal("NewPersistentStore returned nil")
	}

	if store.Count() != 0 {
		t.Errorf("New store should have 0 entries, got %d", store.Count())
	}
}

func TestNewPersistentStoreCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir", "nested")

	store, err := NewPersistentStore(subDir)
	if err != nil {
		t.Fatalf("NewPersistentStore failed: %v", err)
	}

	if store == nil {
		t.Fatal("NewPersistentStore returned nil")
	}

	// Verify directory was created.
	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Path should be a directory")
	}
}

func TestPersistentStorePutAndGet(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// Test Put.
	err := store.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Test Get.
	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Key != "key1" {
		t.Errorf("Key mismatch: got %q", entry.Key)
	}

	if entry.Value != "value1" {
		t.Errorf("Value mismatch: got %q", entry.Value)
	}

	if entry.Namespace != DefaultNamespace {
		t.Errorf("Namespace should be default: got %q", entry.Namespace)
	}
}

func TestPersistentStorePutWithNamespace(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Put(ctx, "key1", "value1", PutOptions{Namespace: "custom"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Namespace != "custom" {
		t.Errorf("Namespace should be 'custom': got %q", entry.Namespace)
	}

	// Verify file was created in namespace subdirectory.
	expectedPath := filepath.Join(tmpDir, "custom", "key1.json")
	_, err = os.Stat(expectedPath)
	if os.IsNotExist(err) {
		t.Errorf("File should exist at %s", expectedPath)
	}
}

func TestPersistentStorePutWithTags(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Put(ctx, "key1", "value1", PutOptions{
		Tags: []string{"tag1", "tag2"},
	})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(entry.Tags))
	}
}

func TestPersistentStorePutUpdateExisting(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// First put.
	err := store.Put(ctx, "key1", "value1", PutOptions{Overwrite: true})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Update with same key.
	err = store.Put(ctx, "key1", "value2", PutOptions{Overwrite: true})
	if err != nil {
		t.Fatalf("Put update failed: %v", err)
	}

	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Value != "value2" {
		t.Errorf("Value should be updated: got %q", entry.Value)
	}

	if store.Count() != 1 {
		t.Errorf("Count should be 1 after update: got %d", store.Count())
	}
}

func TestPersistentStorePutWithoutOverwrite(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// First put.
	err := store.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Try to put again without overwrite - should fail.
	err = store.Put(ctx, "key1", "value2", PutOptions{Overwrite: false})
	if err != ErrKeyExists {
		t.Errorf("Expected ErrKeyExists, got %v", err)
	}
}

func TestPersistentStoreGetNotFound(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	_, err := store.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestPersistentStorePutEmptyKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Put(ctx, "", "value", PutOptions{})
	if err != ErrEmptyKey {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestPersistentStoreGetEmptyKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	_, err := store.Get(ctx, "")
	if err != ErrEmptyKey {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestPersistentStoreDelete(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// Add entry.
	err := store.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Delete it.
	err = store.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone.
	_, err = store.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Count should be 0 after delete: got %d", store.Count())
	}
}

func TestPersistentStoreDeleteNonexistent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// Delete nonexistent key should be idempotent (no error).
	err := store.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete nonexistent should not error, got %v", err)
	}
}

func TestPersistentStoreDeleteEmptyKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Delete(ctx, "")
	if err != ErrEmptyKey {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestPersistentStoreCount(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	if store.Count() != 0 {
		t.Errorf("Initial count should be 0")
	}

	store.Put(ctx, "key1", "value1", PutOptions{})

	if store.Count() != 1 {
		t.Errorf("Count should be 1")
	}

	store.Put(ctx, "key2", "value2", PutOptions{})

	if store.Count() != 2 {
		t.Errorf("Count should be 2")
	}

	store.Delete(ctx, "key1")

	if store.Count() != 1 {
		t.Errorf("Count should be 1 after delete")
	}
}

func TestPersistentStoreList(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "prefix_a", "value1", PutOptions{})
	store.Put(ctx, "prefix_b", "value2", PutOptions{})
	store.Put(ctx, "other", "value3", PutOptions{})

	// List all.
	keys, err := store.List(ctx, "*")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// List with prefix.
	keys, err = store.List(ctx, "prefix_*")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with prefix, got %d", len(keys))
	}
}

func TestPersistentStoreSearch(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "api_response", "status ok", PutOptions{})
	store.Put(ctx, "error_log", "status error", PutOptions{})
	store.Put(ctx, "config", "database url", PutOptions{})

	// Search by keyword in value.
	results, err := store.Search(ctx, "status", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Search by keyword in key.
	results, err = store.Search(ctx, "api", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search with topK limit.
	results, err = store.Search(ctx, "status", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result with topK=1, got %d", len(results))
	}
}

func TestPersistentStoreSearchNoMatches(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "key1", "value1", PutOptions{})

	results, err := store.Search(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestPersistentStorePersistence(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create store and add entries.
	store1, _ := NewPersistentStore(tmpDir)
	store1.Put(ctx, "key1", "value1", PutOptions{Namespace: "ns1"})
	store1.Put(ctx, "key2", "value2", PutOptions{Namespace: "ns2"})
	store1.Close()

	// Create new store at same path - should load existing data.
	store2, _ := NewPersistentStore(tmpDir)

	if store2.Count() != 2 {
		t.Errorf("Reloaded store should have 2 entries, got %d", store2.Count())
	}

	entry, err := store2.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed after reload: %v", err)
	}

	if entry.Value != "value1" {
		t.Errorf("Value mismatch after reload: got %q", entry.Value)
	}

	if entry.Namespace != "ns1" {
		t.Errorf("Namespace mismatch after reload: got %q", entry.Namespace)
	}
}

func TestPersistentStoreConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	const (
		numWriters = 5
		numOps     = 10
	)

	done := make(chan bool, numWriters*2)

	// Writers.
	for i := range numWriters {
		go func(writerID int) {
			for j := range numOps {
				key := "key" + string(rune('A'+writerID)) + string(rune('0'+j%10))
				store.Put(ctx, key, "value", PutOptions{Overwrite: true})
			}

			done <- true
		}(i)
	}

	// Readers.
	for range numWriters {
		go func() {
			for range numOps {
				store.List(ctx, "*")
				store.Search(ctx, "key", 5)
			}

			done <- true
		}()
	}

	// Wait for all goroutines.
	for range numWriters * 2 {
		<-done
	}

	// Basic sanity check.
	if store.Count() == 0 {
		t.Error("Store should have entries after concurrent access")
	}
}

func TestPersistentStoreClose(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestPersistentStoreWithTTL(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	err := store.Put(ctx, "key1", "value1", PutOptions{
		TTL: 1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.TTL != 1*time.Hour {
		t.Errorf("TTL mismatch: got %v", entry.TTL)
	}
}

func TestPersistentStoreGetByNamespaceKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	// Put in different namespaces.
	store.Put(ctx, "key1", "value1", PutOptions{Namespace: "ns1"})
	store.Put(ctx, "key1", "value2", PutOptions{Namespace: "ns2"})

	// Get should find the first match.
	entry, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// Either value is acceptable since we don't specify namespace.
	if entry.Value != "value1" && entry.Value != "value2" {
		t.Errorf("Unexpected value: %q", entry.Value)
	}
}

func TestPersistentStoreDeleteByNamespaceKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "key1", "value1", PutOptions{Namespace: "ns1"})

	err := store.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(ctx, "key1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}

func TestPersistentStoreSearchCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "Key1", "VALUE1", PutOptions{})

	// Search with different case.
	results, err := store.Search(ctx, "key1", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (case-insensitive key), got %d", len(results))
	}

	results, err = store.Search(ctx, "value1", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (case-insensitive value), got %d", len(results))
	}
}

func TestPersistentStoreExactPatternMatch(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	store, _ := NewPersistentStore(tmpDir)

	store.Put(ctx, "exact_key", "value1", PutOptions{})
	store.Put(ctx, "another_key", "value2", PutOptions{})

	// Exact match.
	keys, err := store.List(ctx, "exact_key")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 1 || keys[0] != "exact_key" {
		t.Errorf("Expected exactly 'exact_key', got %v", keys)
	}
}

func TestPersistentStoreRebuildIndexWithInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid JSON file.
	invalidPath := filepath.Join(tmpDir, "default", "invalid.json")
	os.MkdirAll(filepath.Dir(invalidPath), 0700)
	os.WriteFile(invalidPath, []byte("not valid json"), 0600)

	// Create a tmp file (should be ignored).
	tmpFilePath := filepath.Join(tmpDir, "default", "temp.json.tmp")
	os.WriteFile(tmpFilePath, []byte("{}"), 0600)

	// Should not fail, just skip invalid files.
	store, err := NewPersistentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewPersistentStore should not fail: %v", err)
	}

	// Invalid and tmp files should be ignored.
	if store.Count() != 0 {
		t.Errorf("Store should have 0 valid entries, got %d", store.Count())
	}
}
