package memory

import (
	"errors"
	"testing"
	"time"
)

func TestMemoryScopeConstants(t *testing.T) {
	tests := []struct {
		name     string
		scope    MemoryScope
		expected string
	}{
		{"session scope", ScopeSession, "session"},
		{"thread scope", ScopeThread, "thread"},
		{"persistent scope", ScopePersistent, "persistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.scope) != tt.expected {
				t.Errorf("got %q, want %q", tt.scope, tt.expected)
			}
		})
	}
}

func TestEntryTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		entryType EntryType
		expected  string
	}{
		{"note type", EntryTypeNote, "note"},
		{"code type", EntryTypeCode, "code"},
		{"reference type", EntryTypeReference, "reference"},
		{"decision type", EntryTypeDecision, "decision"},
		{"task type", EntryTypeTask, "task"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.entryType) != tt.expected {
				t.Errorf("got %q, want %q", tt.entryType, tt.expected)
			}
		})
	}
}

func TestPutOptionsDefaults(t *testing.T) {
	opts := PutOptions{}

	if opts.TTL != 0 {
		t.Errorf("default TTL should be 0, got %v", opts.TTL)
	}

	if opts.Namespace != "" {
		t.Errorf("default Namespace should be empty, got %q", opts.Namespace)
	}

	if opts.Tags != nil {
		t.Errorf("default Tags should be nil, got %v", opts.Tags)
	}

	if opts.Overwrite != false {
		t.Errorf("default Overwrite should be false, got %v", opts.Overwrite)
	}
}

func TestMemoryEntryFields(t *testing.T) {
	now := time.Now()
	entry := MemoryEntry{
		Key:       "test-key",
		Value:     "test-value",
		Namespace: "test-namespace",
		Tags:      []string{"tag1", "tag2"},
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       time.Hour,
	}

	if entry.Key != "test-key" {
		t.Errorf("Key mismatch: got %q", entry.Key)
	}

	if entry.Value != "test-value" {
		t.Errorf("Value mismatch: got %q", entry.Value)
	}

	if entry.Namespace != "test-namespace" {
		t.Errorf("Namespace mismatch: got %q", entry.Namespace)
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Tags length mismatch: got %d", len(entry.Tags))
	}

	if entry.TTL != time.Hour {
		t.Errorf("TTL mismatch: got %v", entry.TTL)
	}
}

func TestDefaultNamespace(t *testing.T) {
	if DefaultNamespace != "default" {
		t.Errorf("DefaultNamespace should be 'default', got %q", DefaultNamespace)
	}
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound
	if err == nil {
		t.Fatal("ErrNotFound should not be nil")
	}

	if err.Error() != "memory: key not found" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestErrKeyExists(t *testing.T) {
	err := ErrKeyExists
	if err == nil {
		t.Fatal("ErrKeyExists should not be nil")
	}

	if err.Error() != "memory: key already exists" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestErrEmptyKey(t *testing.T) {
	err := ErrEmptyKey
	if err == nil {
		t.Fatal("ErrEmptyKey should not be nil")
	}

	if err.Error() != "memory: key cannot be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"not found error", ErrNotFound, false},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, ErrNotFound)
			if tt.err == ErrNotFound && !got {
				t.Errorf("errors.Is should return true for ErrNotFound")
			}
		})
	}
}
