package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/storage"
)

// ErrScannerRequired is returned when Rebuild is called with a nil scanner.
var ErrScannerRequired = errors.New("metadata scanner is required for rebuild")

// IndexEntry represents a single session in the index.
type IndexEntry struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	LastModified time.Time `json:"last_modified"`
	WorkDir      string    `json:"work_dir"`
}

// IndexData is the on-disk representation of the session index.
type IndexData struct {
	Version int          `json:"version"`
	Entries []IndexEntry `json:"entries"`
}

// MetadataScanner scans session metadata files to build index entries.
type MetadataScanner interface {
	ScanSessions(ctx context.Context) ([]IndexEntry, error)
}

// RebuildCallback is called when the index is rebuilt from scratch.
type RebuildCallback func()

// IndexOption configures an Index instance.
type IndexOption func(*Index)

// WithRebuildCallback sets a callback invoked when the index is auto-rebuilt.
func WithRebuildCallback(callback RebuildCallback) IndexOption {
	return func(idx *Index) {
		idx.onRebuilt = callback
	}
}

// indexVersion is the current index schema version.
const indexVersion = 1

// Index provides fast session listing backed by a single JSON file.
// Auto-rebuilds from metadata files when the index is missing or corrupted.
// Thread-safe via [sync.RWMutex].
type Index struct {
	mu        sync.RWMutex
	path      string
	entries   map[string]IndexEntry
	scanner   MetadataScanner
	onRebuilt RebuildCallback
}

// NewSessionIndex creates or loads a session index from the given path.
// If the index file is missing or corrupted and a scanner is provided, it rebuilds automatically.
func NewSessionIndex(ctx context.Context, path string, scanner MetadataScanner, opts ...IndexOption) (*Index, error) {
	idx := &Index{
		path:    path,
		entries: make(map[string]IndexEntry),
		scanner: scanner,
	}

	for _, opt := range opts {
		opt(idx)
	}

	loadErr := idx.load(ctx)
	if loadErr == nil {
		return idx, nil
	}

	// Auto-rebuild if scanner is available.
	if scanner != nil {
		if rebuildErr := idx.Rebuild(ctx, scanner); rebuildErr != nil {
			return nil, fmt.Errorf("auto-rebuild index: %w", rebuildErr)
		}

		return idx, nil
	}

	// No scanner — start with empty index (missing file is OK).
	if !errors.Is(loadErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("load index: %w", loadErr)
	}

	return idx, nil
}

// Update upserts an entry in the index and persists to disk.
func (idx *Index) Update(ctx context.Context, entry IndexEntry) error {
	idx.mu.Lock()
	idx.entries[entry.ID] = entry
	idx.mu.Unlock()

	return idx.save(ctx)
}

// Remove deletes an entry from the index and persists to disk.
func (idx *Index) Remove(ctx context.Context, sessionID string) error {
	idx.mu.Lock()
	delete(idx.entries, sessionID)
	idx.mu.Unlock()

	return idx.save(ctx)
}

// List returns index entries, optionally filtered by workDir.
// Results are sorted by LastModified descending (newest first).
// An empty workDir returns all entries.
func (idx *Index) List(workDir string) []IndexEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]IndexEntry, 0, len(idx.entries))

	for _, entry := range idx.entries {
		if workDir == "" || entry.WorkDir == workDir {
			result = append(result, entry)
		}
	}

	// Sort by LastModified descending.
	slices.SortFunc(result, func(a, b IndexEntry) int {
		if a.LastModified.After(b.LastModified) {
			return -1
		}

		if a.LastModified.Before(b.LastModified) {
			return 1
		}

		return 0
	})

	return result
}

// Count returns the total number of entries in the index.
func (idx *Index) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.entries)
}

// Rebuild replaces all entries by scanning metadata files via the provided scanner.
func (idx *Index) Rebuild(ctx context.Context, scanner MetadataScanner) error {
	if scanner == nil {
		return ErrScannerRequired
	}

	scanned, err := scanner.ScanSessions(ctx)
	if err != nil {
		return fmt.Errorf("scan sessions: %w", err)
	}

	idx.mu.Lock()
	idx.entries = make(map[string]IndexEntry, len(scanned))

	for _, entry := range scanned {
		idx.entries[entry.ID] = entry
	}

	idx.mu.Unlock()

	if saveErr := idx.save(ctx); saveErr != nil {
		return fmt.Errorf("save after rebuild: %w", saveErr)
	}

	if idx.onRebuilt != nil {
		idx.onRebuilt()
	}

	return nil
}

// load reads the index from disk.
func (idx *Index) load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("load index: %w", err)
	}

	data, readErr := os.ReadFile(idx.path)
	if readErr != nil {
		return fmt.Errorf("read index file: %w", readErr)
	}

	var indexData IndexData
	if jsonErr := json.Unmarshal(data, &indexData); jsonErr != nil {
		return fmt.Errorf("unmarshal index: %w", jsonErr)
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.entries = make(map[string]IndexEntry, len(indexData.Entries))

	for _, entry := range indexData.Entries {
		idx.entries[entry.ID] = entry
	}

	return nil
}

// save persists the index to disk atomically.
func (idx *Index) save(ctx context.Context) error {
	idx.mu.RLock()

	entries := make([]IndexEntry, 0, len(idx.entries))

	for _, entry := range idx.entries {
		entries = append(entries, entry)
	}

	idx.mu.RUnlock()

	indexData := IndexData{
		Version: indexVersion,
		Entries: entries,
	}

	data, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	if writeErr := storage.AtomicWriteFile(ctx, idx.path, data, storage.DefaultFilePerm); writeErr != nil {
		return fmt.Errorf("write index file: %w", writeErr)
	}

	return nil
}
