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

	indexMu sync.RWMutex
	index   []string
	indexed bool
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
	// Check if already indexed
	s.indexMu.RLock()
	if s.indexed {
		s.indexMu.RUnlock()
		return s.indexErr
	}
	s.indexMu.RUnlock()

	// Acquire write lock for indexing
	s.indexMu.Lock()

	// Double-check after acquiring lock (another goroutine may have indexed)
	if s.indexed {
		err := s.indexErr
		s.indexMu.Unlock()
		return err
	}

	// Run scanning with context support
	files, err := s.scanWithContext(ctx)

	if err != nil {
		s.indexErr = err
		s.indexMu.Unlock()
		return err
	}

	s.index = files
	s.indexed = true
	s.indexErr = nil
	s.indexMu.Unlock()

	return nil
}

// scanWithContext performs file scanning with context cancellation support.
func (s *Searcher) scanWithContext(ctx context.Context) ([]string, error) {
	// Check if context is already canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// For now, we use the synchronous scanner
	// TODO: Enhance Scanner.Scan() to support context cancellation
	files, err := s.scanner.Scan()
	if err != nil {
		return nil, err
	}

	// Check context again after scanning
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return files, nil
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
