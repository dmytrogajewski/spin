// Package memory provides context offloading storage for the Spin agent.
//
// The memory package enables the agent to store information outside the LLM's
// immediate context window, providing both session-scoped ephemeral storage
// (Scratchpad) and cross-session persistent storage (PersistentStore).
//
// Key concepts:
//   - MemoryStore: Unified interface for all memory operations
//   - Scratchpad: In-memory session-scoped store with LRU eviction
//   - PersistentStore: File-based cross-session store
//
// Example usage:
//
//	// Create a scratchpad for the current session
//	pad := memory.NewScratchpad("session-123", 50)
//	pad.Put(ctx, "api_response", `{"status": "ok"}`, memory.PutOptions{})
//	entry, _ := pad.Get(ctx, "api_response")
//
//	// Create a persistent store for cross-session memory
//	store, _ := memory.NewPersistentStore("~/.spin/memory")
//	store.Put(ctx, "preference", "tabs over spaces", memory.PutOptions{
//	    Namespace: "preferences",
//	})
package memory

import (
	"context"
	"time"
)

// DefaultNamespace is used when no namespace is specified.
const DefaultNamespace = "default"

// MemoryScope defines where memory is stored.
type MemoryScope string

// Memory scope constants.
const (
	// ScopeSession indicates memory exists only for the current session.
	ScopeSession MemoryScope = "session"

	// ScopeThread indicates memory exists for the current conversation thread.
	ScopeThread MemoryScope = "thread"

	// ScopePersistent indicates memory persists across sessions.
	ScopePersistent MemoryScope = "persistent"
)

// EntryType categorizes memory entries.
type EntryType string

// Entry type constants.
const (
	// EntryTypeNote represents free-form notes.
	EntryTypeNote EntryType = "note"

	// EntryTypeCode represents code snippets.
	EntryTypeCode EntryType = "code"

	// EntryTypeReference represents file/URL references.
	EntryTypeReference EntryType = "reference"

	// EntryTypeDecision represents decisions made during the session.
	EntryTypeDecision EntryType = "decision"

	// EntryTypeTask represents pending tasks.
	EntryTypeTask EntryType = "task"
)

// PutOptions configures the Put operation.
type PutOptions struct {
	// TTL specifies how long the entry should be kept. 0 means no expiry.
	TTL time.Duration

	// Namespace is a logical grouping for the entry. Defaults to "default".
	Namespace string

	// Tags are arbitrary labels for filtering.
	Tags []string

	// Overwrite controls whether to replace an existing entry.
	// If false and key exists, Put returns an error.
	Overwrite bool
}

// MemoryEntry represents a stored memory item.
type MemoryEntry struct {
	// Key is the unique identifier within the namespace.
	Key string

	// Value is the stored content.
	Value string

	// Namespace is the logical grouping.
	Namespace string

	// Tags are arbitrary labels.
	Tags []string

	// CreatedAt is when the entry was first created.
	CreatedAt time.Time

	// UpdatedAt is when the entry was last modified.
	UpdatedAt time.Time

	// TTL is the time-to-live for the entry.
	TTL time.Duration
}

// MemoryStore defines the interface for context offloading storage.
//
// Implementations include Scratchpad (session-scoped) and PersistentStore
// (cross-session). All methods are safe for concurrent use.
type MemoryStore interface {
	// Put stores a value with optional configuration.
	// If opts.Namespace is empty, DefaultNamespace is used.
	Put(ctx context.Context, key string, value string, opts PutOptions) error

	// Get retrieves an entry by key.
	// Returns ErrNotFound if the key does not exist.
	Get(ctx context.Context, key string) (*MemoryEntry, error)

	// Delete removes an entry by key.
	// Returns nil if key does not exist (idempotent).
	Delete(ctx context.Context, key string) error

	// List returns keys matching the pattern.
	// Pattern supports * as wildcard (e.g., "prefix/*", "*").
	List(ctx context.Context, pattern string) ([]string, error)

	// Search finds entries containing the query string.
	// Returns up to topK results, sorted by relevance.
	Search(ctx context.Context, query string, topK int) ([]MemoryEntry, error)
}
