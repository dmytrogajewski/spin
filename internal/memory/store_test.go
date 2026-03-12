package memory

import (
	"errors"
	"testing"
	"time"
)

var errOther = errors.New("other")

func TestScopeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scope    Scope
		expected string
	}{
		{"session scope", ScopeSession, "session"},
		{"thread scope", ScopeThread, "thread"},
		{"persistent scope", ScopePersistent, "persistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.scope) != tt.expected {
				t.Errorf("got %q, want %q", tt.scope, tt.expected)
			}
		})
	}
}

func TestEntryTypeConstants(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			if string(tt.entryType) != tt.expected {
				t.Errorf("got %q, want %q", tt.entryType, tt.expected)
			}
		})
	}
}

func TestPutOptionsDefaults(t *testing.T) {
	t.Parallel()

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

	if opts.Overwrite {
		t.Errorf("default Overwrite should be false, got %v", opts.Overwrite)
	}
}

func TestEntryFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entry := Entry{
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

	if !entry.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", entry.CreatedAt, now)
	}

	if !entry.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", entry.UpdatedAt, now)
	}
}

func TestDefaultNamespace(t *testing.T) {
	t.Parallel()

	if DefaultNamespace != "default" {
		t.Errorf("DefaultNamespace should be 'default', got %q", DefaultNamespace)
	}
}

func TestErrNotFound(t *testing.T) {
	t.Parallel()

	err := ErrNotFound
	if err == nil {
		t.Fatal("ErrNotFound should not be nil")
	}

	if err.Error() != "memory: key not found" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestErrKeyExists(t *testing.T) {
	t.Parallel()

	err := ErrKeyExists
	if err == nil {
		t.Fatal("ErrKeyExists should not be nil")
	}

	if err.Error() != "memory: key already exists" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestErrEmptyKey(t *testing.T) {
	t.Parallel()

	err := ErrEmptyKey
	if err == nil {
		t.Fatal("ErrEmptyKey should not be nil")
	}

	if err.Error() != "memory: key cannot be empty" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"not found error", ErrNotFound, false},
		{"other error", errOther, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errors.Is(tt.err, ErrNotFound)
			if errors.Is(tt.err, ErrNotFound) && !got {
				t.Errorf("errors.Is should return true for ErrNotFound")
			}
		})
	}
}
