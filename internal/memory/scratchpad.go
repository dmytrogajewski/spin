package memory

import (
	"context"
	"sync"
	"time"
)

// ScratchpadEntry represents a session-scoped memory item with tracking.
type ScratchpadEntry struct {
	Key         string
	Value       string
	Type        EntryType
	Namespace   string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AccessCount int
	Pinned      bool
}

// Scratchpad provides session-scoped ephemeral memory with LRU eviction.
//
// It implements the Store interface and provides additional methods
// for session-specific operations like pinning entries.
type Scratchpad struct {
	entries   map[string]*ScratchpadEntry
	maxSize   int
	mu        sync.RWMutex
	sessionID string
}

// NewScratchpad creates a new scratchpad for the given session.
//
// The maxSize parameter controls the maximum number of entries.
// When capacity is reached, the least recently used (LRU) entry
// is evicted unless it is pinned.
func NewScratchpad(sessionID string, maxSize int) *Scratchpad {
	return &Scratchpad{
		entries:   make(map[string]*ScratchpadEntry),
		maxSize:   maxSize,
		sessionID: sessionID,
	}
}

// SessionID returns the session identifier.
func (s *Scratchpad) SessionID() string {
	return s.sessionID
}

// MaxSize returns the maximum number of entries.
func (s *Scratchpad) MaxSize() int {
	return s.maxSize
}

// Count returns the current number of entries.
func (s *Scratchpad) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.entries)
}

// Put stores a value in the scratchpad.
//
// If the scratchpad is at capacity, the least recently used entry
// is evicted (unless pinned) before adding the new entry.
func (s *Scratchpad) Put(_ context.Context, key, value string, opts PutOptions) error {
	if key == "" {
		return ErrEmptyKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if key exists.
	existing, exists := s.entries[key]
	if exists && !opts.Overwrite {
		return ErrKeyExists
	}

	// Determine namespace.
	namespace := opts.Namespace
	if namespace == "" {
		namespace = DefaultNamespace
	}

	now := time.Now()

	if exists {
		// Update existing entry.
		existing.Value = value
		existing.UpdatedAt = now
		existing.Namespace = namespace
		existing.Tags = opts.Tags
	} else {
		// Evict if at capacity.
		if len(s.entries) >= s.maxSize {
			s.evictLRU()
		}

		// Create new entry.
		s.entries[key] = &ScratchpadEntry{
			Key:         key,
			Value:       value,
			Type:        inferEntryType(value),
			Namespace:   namespace,
			Tags:        opts.Tags,
			CreatedAt:   now,
			UpdatedAt:   now,
			AccessCount: 0,
			Pinned:      false,
		}
	}

	return nil
}

// Get retrieves an entry by key and increments its access count.
func (s *Scratchpad) Get(_ context.Context, key string) (*Entry, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if !exists {
		return nil, ErrNotFound
	}

	// Increment access count.
	entry.AccessCount++

	return &Entry{
		Key:       entry.Key,
		Value:     entry.Value,
		Namespace: entry.Namespace,
		Tags:      entry.Tags,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}, nil
}

// Delete removes an entry by key.
//
// Returns nil if the key does not exist (idempotent).
func (s *Scratchpad) Delete(_ context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, key)

	return nil
}

// List returns keys matching the pattern.
//
// Pattern supports * as wildcard. Use "*" to list all keys.
func (s *Scratchpad) List(_ context.Context, pattern string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.entries))
	for key := range s.entries {
		if matchPattern(pattern, key) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Search finds entries containing the query string.
//
// Searches both keys and values. Returns up to topK results,
// sorted by access count (most accessed first).
func (s *Scratchpad) Search(_ context.Context, query string, topK int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect matching entries.
	matches := make([]Entry, 0)

	for _, entry := range s.entries {
		if containsIgnoreCase(entry.Key, query) || containsIgnoreCase(entry.Value, query) {
			matches = append(matches, Entry{
				Key:       entry.Key,
				Value:     entry.Value,
				Namespace: entry.Namespace,
				Tags:      entry.Tags,
				CreatedAt: entry.CreatedAt,
				UpdatedAt: entry.UpdatedAt,
			})
		}
	}

	// Sort by access count (descending) - most accessed first.
	sortByAccessCount(matches, s.entries)

	// Limit to topK.
	if len(matches) > topK {
		matches = matches[:topK]
	}

	return matches, nil
}

// Pin marks an entry as pinned, preventing it from being auto-evicted.
func (s *Scratchpad) Pin(key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if !exists {
		return ErrNotFound
	}

	entry.Pinned = true

	return nil
}

// Unpin removes the pinned flag from an entry.
func (s *Scratchpad) Unpin(key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if !exists {
		return ErrNotFound
	}

	entry.Pinned = false

	return nil
}

// Clear removes all entries from the scratchpad.
func (s *Scratchpad) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]*ScratchpadEntry)
}

// evictLRU removes the least recently used entry that is not pinned.
// Must be called with lock held.
func (s *Scratchpad) evictLRU() {
	var lruKey string

	lruAccessCount := -1

	for key, entry := range s.entries {
		if entry.Pinned {
			continue
		}

		if lruAccessCount == -1 || entry.AccessCount < lruAccessCount {
			lruAccessCount = entry.AccessCount
			lruKey = key
		}
	}

	if lruKey != "" {
		delete(s.entries, lruKey)
	}
}

// entryTypeRule maps keywords to an EntryType for inference.
type entryTypeRule struct {
	entryType EntryType
	keywords  []string
}

// entryTypeRules defines pattern-matching rules for inferring entry types.
// Rules are evaluated in order; the first match wins.
var entryTypeRules = []entryTypeRule{
	{EntryTypeCode, []string{"func ", "class ", "def ", "```"}},
	{EntryTypeReference, []string{"http://", "https://", "file://"}},
	{EntryTypeDecision, []string{"decided", "decision", "will use", "chose"}},
	{EntryTypeTask, []string{"to-do", "task", "need to", "should"}},
}

// inferEntryType guesses the entry type from its value.
func inferEntryType(value string) EntryType {
	for _, rule := range entryTypeRules {
		if matchesAnyKeyword(value, rule.keywords) {
			return rule.entryType
		}
	}

	return EntryTypeNote
}

// matchesAnyKeyword returns true if value contains any of the keywords (case-insensitive).
func matchesAnyKeyword(value string, keywords []string) bool {
	for _, kw := range keywords {
		if containsIgnoreCase(value, kw) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" ||
		findIgnoreCase(s, substr) >= 0)
}

// findIgnoreCase finds substr in s (case-insensitive).
func findIgnoreCase(s, substr string) int {
	if substr == "" {
		return 0
	}

	if len(s) < len(substr) {
		return -1
	}

	// Simple case-insensitive search.
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true

		for j := range len(substr) {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false

				break
			}
		}

		if match {
			return i
		}
	}

	return -1
}

// toLower converts a byte to lowercase.
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}

	return b
}

// matchPattern checks if key matches the glob pattern.
// Supports * as wildcard.
func matchPattern(pattern, key string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}

	// Simple prefix matching for patterns like "prefix*".
	if pattern != "" && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]

		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	// Exact match.
	return pattern == key
}

// sortByAccessCount sorts entries by access count (descending).
func sortByAccessCount(entries []Entry, lookup map[string]*ScratchpadEntry) {
	// Simple bubble sort for small lists.
	for i := range len(entries) - 1 {
		for j := range len(entries) - i - 1 {
			countJ := lookup[entries[j].Key].AccessCount

			countJ1 := lookup[entries[j+1].Key].AccessCount
			if countJ < countJ1 {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}
}
