package filesearch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Scanner scans directories for files with gitignore support.
type Scanner struct {
	baseDir       string
	ignoreGit     bool // Deprecated: use IgnoreHandler instead.
	maxDepth      int
	ignoreHandler *IgnoreHandler // Handles .gitignore and .spinignore patterns.
}

// NewScanner creates a new file scanner with gitignore support.
// If ignoreGit is true, basic .git exclusion is enabled (for backward compatibility).
// The Scanner will automatically create an IgnoreHandler to respect .gitignore and .spinignore files.
func NewScanner(baseDir string, ignoreGit bool) *Scanner {
	return &Scanner{
		baseDir:   baseDir,
		ignoreGit: ignoreGit,
		maxDepth:  20, // Reasonable default to prevent deep recursion.
		// ignoreHandler is lazily created in Scan().
	}
}

// Scan returns all files in the directory recursively.
// Returns relative paths from baseDir.
// Files matching .gitignore or .spinignore patterns are excluded.
func (s *Scanner) Scan() ([]string, error) {
	return s.ScanWithContext(context.Background())
}

// ScanWithContext returns all files in the directory recursively with context cancellation support.
// Returns relative paths from baseDir.
// Files matching .gitignore or .spinignore patterns are excluded.
// If the context is canceled, scanning stops and returns an error.
func (s *Scanner) ScanWithContext(ctx context.Context) ([]string, error) {
	var files []string

	s.ensureIgnoreHandler()

	err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, walkErr error) error {
		// Check context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only process entries without walk errors.
		if walkErr == nil {
			relPath, shouldSkip := s.processPath(path, d)
			if shouldSkip {
				return filepath.SkipDir
			}

			if relPath != "" {
				files = append(files, relPath)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return files, nil
}

// ensureIgnoreHandler creates an IgnoreHandler if not provided.
func (s *Scanner) ensureIgnoreHandler() {
	if s.ignoreHandler == nil && s.baseDir != "" {
		handler, _ := NewIgnoreHandler(s.baseDir)
		s.ignoreHandler = handler
	}
}

// processPath processes a single path and returns the relative path and skip flag.
func (s *Scanner) processPath(path string, d os.DirEntry) (string, bool) {
	relPath, err := filepath.Rel(s.baseDir, path)
	if err != nil {
		return "", false // Skip if can't get relative path.
	}

	relPath = filepath.ToSlash(relPath) // Convert to forward slashes for consistency.

	if s.shouldIgnorePath(relPath, d) {
		return "", d.IsDir() // Skip directory if ignored.
	}

	if d.IsDir() {
		return "", false // Don't add directories to results.
	}

	return relPath, false
}

// shouldIgnorePath checks if a path should be ignored.
func (s *Scanner) shouldIgnorePath(relPath string, d os.DirEntry) bool {
	// Check ignore handler first.
	if s.ignoreHandler != nil && s.ignoreHandler.IsIgnored(relPath, d.IsDir()) {
		return true
	}

	// Legacy ignoreGit support (for backward compatibility).
	if d.IsDir() && s.ignoreGit && d.Name() == ".git" {
		return true
	}

	return false
}
