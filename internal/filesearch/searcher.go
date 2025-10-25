package filesearch

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Searcher provides high-level file search with async indexing and advanced ranking.
type Searcher struct {
	root    string
	scanner *Scanner
	matcher *Matcher

	indexMu  sync.RWMutex
	index    []string
	indexed  bool
	indexErr error
}

// NewSearcher creates a new searcher for the given root directory.
// Returns an error if the root directory does not exist.
func NewSearcher(root string) (*Searcher, error) {
	// Validate root exists
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("invalid root directory: %w", err)
	}

	return &Searcher{
		root:    root,
		scanner: NewScanner(root, false), // Will auto-load .gitignore/.spinignore
		matcher: NewMatcher(false),       // Case-insensitive by default
		indexed: false,
	}, nil
}

// IndexAsync indexes files asynchronously with cancellation support.
// This method is idempotent - calling it multiple times is safe.
// If indexing is already complete, returns immediately.
// If indexing is in progress, waits for it to complete.
func (s *Searcher) IndexAsync(ctx context.Context) error {
	if s.isAlreadyIndexed() {
		return s.getIndexError()
	}

	return s.performIndexing(ctx)
}

// isAlreadyIndexed checks if the searcher is already indexed.
func (s *Searcher) isAlreadyIndexed() bool {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.indexed
}

// getIndexError returns the current index error.
func (s *Searcher) getIndexError() error {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.indexErr
}

// performIndexing performs the actual indexing operation.
func (s *Searcher) performIndexing(ctx context.Context) error {
	s.indexMu.Lock()
	defer s.indexMu.Unlock()

	// Double-check after acquiring lock (another goroutine may have indexed)
	if s.indexed {
		return s.indexErr
	}

	files, err := s.scanWithContext(ctx)
	if err != nil {
		s.indexErr = err
		return err
	}

	s.index = files
	s.indexed = true
	s.indexErr = nil
	return nil
}

// scanWithContext performs file scanning with context cancellation support.
func (s *Searcher) scanWithContext(ctx context.Context) ([]string, error) {
	// Use the context-aware scanning method
	return s.scanner.ScanWithContext(ctx)
}

// Search performs ranked search on the indexed files.
// Returns top matches sorted by score (highest first).
// If limit is 0 or negative, returns all matches.
// Returns empty slice if not indexed or query is empty.
func (s *Searcher) Search(query string, limit int) []Match {
	if query == "" {
		return []Match{}
	}

	// Read lock for concurrent search
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()

	if !s.indexed {
		return []Match{}
	}

	// Perform fuzzy matching with advanced scoring
	matches := s.matcher.Match(query, s.index)

	// Apply limit if specified
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

// IsIndexed returns true if indexing is complete.
func (s *Searcher) IsIndexed() bool {
	s.indexMu.RLock()
	defer s.indexMu.RUnlock()
	return s.indexed
}
