package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
	"github.com/dmytrogajewski/spin/pkg/storage"
)

// persistedEntry represents the JSON structure stored in files.
type persistedEntry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Namespace string    `json:"namespace"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TTL       int64     `json:"ttl_seconds,omitempty"` // TTL in seconds, 0 = no expiry.
}

// IndexEntry tracks metadata for a persistent entry.
type IndexEntry struct {
	Key         string
	Namespace   string
	Tags        []string
	FilePath    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	AccessCount int
	Size        int64
}

// PersistentStore provides file-based cross-session memory.
//
// It implements the Store interface and persists entries to
// the filesystem as JSON files, organized by namespace.
type PersistentStore struct {
	basePath string
	index    map[string]*IndexEntry
	mu       sync.RWMutex
}

// NewPersistentStore creates a persistent store at the given path.
//
// The directory is created if it doesn't exist. On startup,
// the store scans the directory to rebuild its index of existing entries.
// The context controls the startup index rebuild.
func NewPersistentStore(ctx context.Context, basePath string) (*PersistentStore, error) {
	// Expand home directory if needed.
	expanded, err := pathx.ExpandHome(basePath)
	if err != nil {
		return nil, fmt.Errorf("expand home directory: %w", err)
	}

	basePath = expanded

	// Create directory if it doesn't exist.
	err = os.MkdirAll(basePath, 0o700)
	if err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	store := &PersistentStore{
		basePath: basePath,
		index:    make(map[string]*IndexEntry),
	}

	// Rebuild index from existing files.
	err = store.rebuildIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("rebuild index: %w", err)
	}

	return store, nil
}

// Put stores a value to the filesystem.
func (s *PersistentStore) Put(ctx context.Context, key, value string, opts PutOptions) error {
	if key == "" {
		return ErrEmptyKey
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine namespace.
	namespace := opts.Namespace
	if namespace == "" {
		namespace = DefaultNamespace
	}

	// Check for existing entry.
	indexKey := s.indexKey(namespace, key)

	existing, exists := s.index[indexKey]
	if exists && !opts.Overwrite {
		return ErrKeyExists
	}

	now := time.Now()

	// Prepare entry.
	entry := persistedEntry{
		Key:       key,
		Value:     value,
		Namespace: namespace,
		Tags:      opts.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if opts.TTL > 0 {
		entry.TTL = int64(opts.TTL.Seconds())
	}

	if exists {
		entry.CreatedAt = existing.CreatedAt
	}

	// Create namespace directory.
	namespaceDir := filepath.Join(s.basePath, namespace)

	err := os.MkdirAll(namespaceDir, 0o700)
	if err != nil {
		return fmt.Errorf("create namespace directory: %w", err)
	}

	// Serialize to JSON.
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize entry: %w", err)
	}

	// Atomic write.
	filePath := s.filePath(namespace, key)

	err = storage.AtomicWriteFile(ctx, filePath, data, storage.DefaultFilePerm)
	if err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	// Update index.
	s.index[indexKey] = &IndexEntry{
		Key:       key,
		Namespace: namespace,
		Tags:      opts.Tags,
		FilePath:  filePath,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: now,
		Size:      int64(len(data)),
	}

	return nil
}

// Get retrieves an entry from the filesystem.
func (s *PersistentStore) Get(ctx context.Context, key string) (*Entry, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find in index (search all namespaces).
	_, indexEntry := s.findByKey(key)
	if indexEntry == nil {
		return nil, ErrNotFound
	}

	// Read file.
	data, err := os.ReadFile(indexEntry.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Index is stale, remove entry.
			delete(s.index, s.indexKey(indexEntry.Namespace, key))

			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("read file: %w", err)
	}

	// Deserialize.
	var entry persistedEntry

	err = json.Unmarshal(data, &entry)
	if err != nil {
		return nil, fmt.Errorf("deserialize entry: %w", err)
	}

	// Increment access count.
	indexEntry.AccessCount++

	var ttl time.Duration
	if entry.TTL > 0 {
		ttl = time.Duration(entry.TTL) * time.Second
	}

	return &Entry{
		Key:       entry.Key,
		Value:     entry.Value,
		Namespace: entry.Namespace,
		Tags:      entry.Tags,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		TTL:       ttl,
	}, nil
}

// Delete removes an entry from the filesystem.
func (s *PersistentStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find in index.
	indexKey, indexEntry := s.findByKey(key)
	if indexEntry == nil {
		return nil // Idempotent - no error if doesn't exist.
	}

	// Delete file.
	err := os.Remove(indexEntry.FilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}

	// Remove from index.
	delete(s.index, indexKey)

	return nil
}

// List returns keys matching the pattern.
func (s *PersistentStore) List(ctx context.Context, pattern string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.index))
	for _, entry := range s.index {
		if matchPattern(pattern, entry.Key) {
			keys = append(keys, entry.Key)
		}
	}

	return keys, nil
}

// Search finds entries containing the query string.
func (s *PersistentStore) Search(ctx context.Context, query string, topK int) ([]Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]Entry, 0)

	for _, indexEntry := range s.index {
		// Check context between iterations (each may read a file).
		if err := ctx.Err(); err != nil {
			return matches, fmt.Errorf("search: %w", err)
		}

		// Check key match.
		if stringsx.ContainsIgnoreCase(indexEntry.Key, query) {
			entry, err := s.readEntryUnsafe(indexEntry.FilePath)
			if err == nil {
				matches = append(matches, *entry)
			}

			continue
		}

		// Check value match (need to read file).
		entry, err := s.readEntryUnsafe(indexEntry.FilePath)
		if err == nil && stringsx.ContainsIgnoreCase(entry.Value, query) {
			matches = append(matches, *entry)
		}
	}

	// Limit to topK.
	if len(matches) > topK {
		matches = matches[:topK]
	}

	return matches, nil
}

// Count returns the total number of entries.
func (s *PersistentStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.index)
}

// Close persists the index and releases resources.
func (s *PersistentStore) Close() error {
	// Index is rebuilt on startup from files, no need to persist.
	return nil
}

// rebuildIndex scans the directory structure and rebuilds the in-memory index.
func (s *PersistentStore) rebuildIndex(ctx context.Context) error {
	// Walk directory looking for .json files.
	walkErr := filepath.WalkDir(s.basePath, func(path string, dirEntry os.DirEntry, walkDirErr error) error {
		if walkDirErr != nil {
			return walkDirErr
		}

		// Check context between files.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("rebuild index: %w", err)
		}

		if dirEntry.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".tmp") {
			return nil
		}

		s.indexFileEntry(path, dirEntry)

		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("walk memory store: %w", walkErr)
	}

	return nil
}

// indexFileEntry reads and indexes a single file entry. Silently skips unreadable or invalid files.
func (s *PersistentStore) indexFileEntry(path string, dirEntry os.DirEntry) {
	cleanPath := filepath.Clean(path)

	data, readErr := os.ReadFile(cleanPath)
	if readErr != nil || len(data) == 0 {
		return
	}

	var entry persistedEntry
	if unmarshalErr := json.Unmarshal(data, &entry); unmarshalErr != nil || entry.Key == "" {
		return
	}

	info, infoErr := dirEntry.Info()
	if infoErr != nil {
		return
	}

	indexKey := s.indexKey(entry.Namespace, entry.Key)
	s.index[indexKey] = &IndexEntry{
		Key:       entry.Key,
		Namespace: entry.Namespace,
		Tags:      entry.Tags,
		FilePath:  cleanPath,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		Size:      info.Size(),
	}
}

// readEntryUnsafe reads an entry without locking. Must be called with lock held.
func (s *PersistentStore) readEntryUnsafe(filePath string) (*Entry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read entry file: %w", err)
	}

	var entry persistedEntry

	err = json.Unmarshal(data, &entry)
	if err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}

	var ttl time.Duration
	if entry.TTL > 0 {
		ttl = time.Duration(entry.TTL) * time.Second
	}

	return &Entry{
		Key:       entry.Key,
		Value:     entry.Value,
		Namespace: entry.Namespace,
		Tags:      entry.Tags,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		TTL:       ttl,
	}, nil
}

// findByKey locates an index entry by its key across all namespaces.
// Must be called with lock held. Returns ("", nil) if not found.
func (s *PersistentStore) findByKey(key string) (string, *IndexEntry) {
	for k, entry := range s.index {
		if entry.Key == key {
			return k, entry
		}
	}

	return "", nil
}

// indexKey creates a unique key for the index from namespace and key.
func (s *PersistentStore) indexKey(namespace, key string) string {
	return namespace + "/" + key
}

// filePath returns the file path for an entry.
func (s *PersistentStore) filePath(namespace, key string) string {
	return filepath.Join(s.basePath, namespace, key+".json")
}
