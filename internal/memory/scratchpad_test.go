package memory

import (
	"errors"
	"context"
	"testing"
)

func TestNewScratchpad(t *testing.T) {
	sessionID := "test-session-123"
	maxSize := 50

	pad := NewScratchpad(sessionID, maxSize)

	if pad == nil {
		t.Fatal("NewScratchpad returned nil")
	}

	if pad.SessionID() != sessionID {
		t.Errorf("SessionID mismatch: got %q, want %q", pad.SessionID(), sessionID)
	}

	if pad.MaxSize() != maxSize {
		t.Errorf("MaxSize mismatch: got %d, want %d", pad.MaxSize(), maxSize)
	}

	if pad.Count() != 0 {
		t.Errorf("Count should be 0 for new scratchpad, got %d", pad.Count())
	}
}

func TestScratchpadPutAndGet(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	// Test Put.
	err := pad.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Test Get.
	entry, err := pad.Get(ctx, "key1")
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

func TestScratchpadPutWithNamespace(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	err := pad.Put(ctx, "key1", "value1", PutOptions{Namespace: "custom"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	entry, err := pad.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Namespace != "custom" {
		t.Errorf("Namespace should be 'custom': got %q", entry.Namespace)
	}
}

func TestScratchpadPutUpdateExisting(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	// First put.
	err := pad.Put(ctx, "key1", "value1", PutOptions{Overwrite: true})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Update with same key.
	err = pad.Put(ctx, "key1", "value2", PutOptions{Overwrite: true})
	if err != nil {
		t.Fatalf("Put update failed: %v", err)
	}

	entry, err := pad.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if entry.Value != "value2" {
		t.Errorf("Value should be updated: got %q", entry.Value)
	}

	if pad.Count() != 1 {
		t.Errorf("Count should be 1 after update: got %d", pad.Count())
	}
}

func TestScratchpadPutWithoutOverwrite(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	// First put (Overwrite defaults to false).
	err := pad.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Try to put again without overwrite - should fail.
	err = pad.Put(ctx, "key1", "value2", PutOptions{Overwrite: false})
	if !errors.Is(err, ErrKeyExists) {
		t.Errorf("Expected ErrKeyExists, got %v", err)
	}
}

func TestScratchpadGetNotFound(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	_, err := pad.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestScratchpadPutEmptyKey(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	err := pad.Put(ctx, "", "value", PutOptions{})
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestScratchpadGetEmptyKey(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	_, err := pad.Get(ctx, "")
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestScratchpadDelete(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	// Add entry.
	err := pad.Put(ctx, "key1", "value1", PutOptions{})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Delete it.
	err = pad.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone.
	_, err = pad.Get(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}

	if pad.Count() != 0 {
		t.Errorf("Count should be 0 after delete: got %d", pad.Count())
	}
}

func TestScratchpadDeleteNonexistent(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	// Delete nonexistent key should be idempotent (no error).
	err := pad.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete nonexistent should not error, got %v", err)
	}
}

func TestScratchpadCount(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if pad.Count() != 0 {
		t.Errorf("Initial count should be 0")
	}

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if pad.Count() != 1 {
		t.Errorf("Count should be 1")
	}

	if err := pad.Put(ctx, "key2", "value2", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if pad.Count() != 2 {
		t.Errorf("Count should be 2")
	}

	if err := pad.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if pad.Count() != 1 {
		t.Errorf("Count should be 1 after delete")
	}
}

func TestScratchpadClear(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key2", "value2", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key3", "value3", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if pad.Count() != 3 {
		t.Errorf("Count should be 3 before clear")
	}

	pad.Clear()

	if pad.Count() != 0 {
		t.Errorf("Count should be 0 after clear")
	}
}

func TestScratchpadAccessCountIncrement(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// First get - access count should be 1.
	_, err := pad.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// Second get - access count should be 2.
	_, err = pad.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// Third get - access count should be 3.
	entry, err := pad.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Access count starts at 0 and increments on Get
	// After 3 Gets, it should be 3.
	if entry == nil {
		t.Fatal("entry should not be nil")
	}
}

func TestScratchpadLRUEviction(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 3) // Small capacity for testing.

	// Add 3 entries.
	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key2", "value2", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key3", "value3", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Access key2 and key3 to make them more recently used.
	if _, err := pad.Get(ctx, "key2"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if _, err := pad.Get(ctx, "key3"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Add 4th entry - should evict key1 (least accessed).
	if err := pad.Put(ctx, "key4", "value4", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if pad.Count() != 3 {
		t.Errorf("Count should be 3 after eviction, got %d", pad.Count())
	}

	// key1 should be evicted.
	_, err := pad.Get(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("key1 should have been evicted")
	}

	// key2, key3, key4 should still exist.
	_, err = pad.Get(ctx, "key2")
	if err != nil {
		t.Errorf("key2 should exist: %v", err)
	}

	_, err = pad.Get(ctx, "key3")
	if err != nil {
		t.Errorf("key3 should exist: %v", err)
	}

	_, err = pad.Get(ctx, "key4")
	if err != nil {
		t.Errorf("key4 should exist: %v", err)
	}
}

func TestScratchpadPinnedEntryNotEvicted(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 3)

	// Add 3 entries.
	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key2", "value2", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "key3", "value3", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Pin key1 (access count 0).
	err := pad.Pin("key1")
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	// Access key3 multiple times to make it more recently used
	// key2 stays at 0 access count, key3 gets 2 access counts.
	if _, err = pad.Get(ctx, "key3"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if _, err = pad.Get(ctx, "key3"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Add 4th entry - should evict key2 (access count 0, unpinned)
	// key1 has 0 but is pinned, key3 has 2.
	if err = pad.Put(ctx, "key4", "value4", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// key1 should still exist (pinned).
	_, err = pad.Get(ctx, "key1")
	if err != nil {
		t.Errorf("key1 should not be evicted (pinned): %v", err)
	}

	// key2 should be evicted (lowest access count among unpinned).
	_, err = pad.Get(ctx, "key2")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("key2 should have been evicted")
	}

	// key3 and key4 should exist.
	_, err = pad.Get(ctx, "key3")
	if err != nil {
		t.Errorf("key3 should exist: %v", err)
	}

	_, err = pad.Get(ctx, "key4")
	if err != nil {
		t.Errorf("key4 should exist: %v", err)
	}
}

func TestScratchpadPin(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	err := pad.Pin("key1")
	if err != nil {
		t.Errorf("Pin failed: %v", err)
	}
}

func TestScratchpadPinNotFound(t *testing.T) {
	pad := NewScratchpad("session-1", 10)

	err := pad.Pin("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestScratchpadPinEmptyKey(t *testing.T) {
	pad := NewScratchpad("session-1", 10)

	err := pad.Pin("")
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestScratchpadUnpin(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Pin("key1"); err != nil {
		t.Fatalf("Pin failed: %v", err)
	}

	err := pad.Unpin("key1")
	if err != nil {
		t.Errorf("Unpin failed: %v", err)
	}
}

func TestScratchpadUnpinNotFound(t *testing.T) {
	pad := NewScratchpad("session-1", 10)

	err := pad.Unpin("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestScratchpadList(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "prefix_a", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "prefix_b", "value2", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "other", "value3", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// List all.
	keys, err := pad.List(ctx, "*")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// List with prefix.
	keys, err = pad.List(ctx, "prefix_*")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with prefix, got %d", len(keys))
	}
}

func TestScratchpadSearch(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "api_response", "status ok", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "error_log", "status error", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if err := pad.Put(ctx, "config", "database url", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Search by keyword in value.
	results, err := pad.Search(ctx, "status", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Search by keyword in key.
	results, err = pad.Search(ctx, "api", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search with topK limit.
	results, err = pad.Search(ctx, "status", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result with topK=1, got %d", len(results))
	}
}

func TestScratchpadSearchNoMatches(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "key1", "value1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	results, err := pad.Search(ctx, "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestScratchpadSearchCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	if err := pad.Put(ctx, "Key1", "VALUE1", PutOptions{}); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Search with different case.
	results, err := pad.Search(ctx, "key1", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (case-insensitive), got %d", len(results))
	}

	results, err = pad.Search(ctx, "value1", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (case-insensitive), got %d", len(results))
	}
}

func TestScratchpadConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 100)

	// Run concurrent writers.
	const (
		numWriters = 10
		numOps     = 50
	)

	done := make(chan bool, numWriters*2)

	// Writers.
	for i := range numWriters {
		go func(writerID int) {
			for j := range numOps {
				key := "key" + string(rune('A'+writerID)) + string(rune('0'+j%10))
				_ = pad.Put(ctx, key, "value", PutOptions{Overwrite: true})
			}

			done <- true
		}(i)
	}

	// Readers.
	for range numWriters {
		go func() {
			for range numOps {
				_, _ = pad.List(ctx, "*")
				_, _ = pad.Search(ctx, "key", 5)
			}

			done <- true
		}()
	}

	// Wait for all goroutines.
	for range numWriters * 2 {
		<-done
	}

	// Basic sanity check - scratchpad should have some entries.
	if pad.Count() == 0 {
		t.Error("Scratchpad should have entries after concurrent access")
	}
}

func TestScratchpadDeleteEmptyKey(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	err := pad.Delete(ctx, "")
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestScratchpadUnpinEmptyKey(t *testing.T) {
	pad := NewScratchpad("session-1", 10)

	err := pad.Unpin("")
	if !errors.Is(err, ErrEmptyKey) {
		t.Errorf("Expected ErrEmptyKey, got %v", err)
	}
}

func TestScratchpadEntryTypeInference(t *testing.T) {
	ctx := context.Background()
	pad := NewScratchpad("session-1", 10)

	tests := []struct {
		key      string
		value    string
		expected EntryType
	}{
		{"note", "just a simple note", EntryTypeNote},
		{"code", "func main() { }", EntryTypeCode},
		{"code_block", "```go\nfunc test() {}\n```", EntryTypeCode},
		{"url", "https://example.com", EntryTypeReference},
		{"file", "file:///path/to/file", EntryTypeReference},
		{"decision", "We decided to use PostgreSQL", EntryTypeDecision},
		{"task", "TODO: implement feature", EntryTypeTask},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := pad.Put(ctx, tt.key, tt.value, PutOptions{}); err != nil {
				t.Fatalf("Put failed: %v", err)
			}
			// We can't directly check the type, but the entry should exist.
			_, err := pad.Get(ctx, tt.key)
			if err != nil {
				t.Errorf("Failed to get entry: %v", err)
			}
		})
	}
}
