package filesearch

import (
	"os"
	"path/filepath"
)

// Scanner scans directories for files with gitignore support.
type Scanner struct {
	baseDir       string
	ignoreGit     bool // Deprecated: use IgnoreHandler instead
	maxDepth      int
	ignoreHandler *IgnoreHandler // Handles .gitignore and .spinignore patterns
}

// NewScanner creates a new file scanner with gitignore support.
// If ignoreGit is true, basic .git exclusion is enabled (for backward compatibility).
// The Scanner will automatically create an IgnoreHandler to respect .gitignore and .spinignore files.
func NewScanner(baseDir string, ignoreGit bool) *Scanner {
	return &Scanner{
		baseDir:   baseDir,
		ignoreGit: ignoreGit,
		maxDepth:  20, // Reasonable default to prevent deep recursion
		// ignoreHandler will be lazily created in Scan()
	}
}

// NewScannerWithIgnore creates a scanner with a custom IgnoreHandler.
// This allows advanced configuration and testing.
func NewScannerWithIgnore(baseDir string, handler *IgnoreHandler) *Scanner {
	return &Scanner{
		baseDir:       baseDir,
		ignoreGit:     false, // Not needed when using custom handler
		maxDepth:      20,
		ignoreHandler: handler,
	}
}

// Scan returns all files in the directory recursively.
// Returns relative paths from baseDir.
// Files matching .gitignore or .spinignore patterns are excluded.
func (s *Scanner) Scan() ([]string, error) {
	var files []string

	// Auto-create IgnoreHandler if not provided
	if s.ignoreHandler == nil && s.baseDir != "" {
		handler, _ := NewIgnoreHandler(s.baseDir)
		s.ignoreHandler = handler
	}

	err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip errors (permission denied, etc.)
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			// Skip if can't get relative path
			return nil
		}

		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Check if path should be ignored
		if s.ignoreHandler != nil && s.ignoreHandler.IsIgnored(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Legacy ignoreGit support (for backward compatibility)
		// This is now redundant since IgnoreHandler has .git/** by default,
		// but we keep it for explicit backward compatibility
		if d.IsDir() {
			if s.ignoreGit && d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Add file to results
		files = append(files, relPath)
		return nil
	})

	return files, err
}
