package filesearch

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// IgnoreHandler handles .gitignore and .spinignore pattern matching.
// It loads patterns from .gitignore, .spinignore, and includes sensible defaults
// to exclude common directories like .git, node_modules, vendor, etc.
type IgnoreHandler struct {
	patterns []string // All loaded ignore patterns.
	rootDir  string   // Root directory for pattern resolution.
}

// NewIgnoreHandler creates a new ignore handler for the given root directory.
// It loads .gitignore and .spinignore files if they exist, and includes default patterns.
// Returns an error only on critical failures (file exists but cannot be read).
// Missing .gitignore or .spinignore files are not considered errors.
func NewIgnoreHandler(rootDir string) (*IgnoreHandler, error) {
	h := &IgnoreHandler{
		rootDir:  rootDir,
		patterns: make([]string, 0),
	}

	// Load default patterns first.
	h.patterns = append(h.patterns, defaultPatterns()...)

	// Try to load .gitignore.
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	err := h.loadIgnoreFile(gitignorePath)
	if err != nil {
		// Only return error if file exists but cannot be read.
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Try to load .spinignore.
	spinignorePath := filepath.Join(rootDir, ".spinignore")
	err = h.loadIgnoreFile(spinignorePath)
	if err != nil {
		// Only return error if file exists but cannot be read.
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return h, nil
}

// IsIgnored checks if a relative path should be ignored based on loaded patterns.
// The path parameter should be relative to rootDir.
// The isDir parameter indicates whether the path is a directory.
// Returns true if the path matches any ignore pattern.
func (h *IgnoreHandler) IsIgnored(relPath string, isDir bool) bool {
	// Empty path is not ignored.
	if relPath == "" || relPath == "." || relPath == "./" {
		return false
	}

	// Check each pattern.
	for _, pattern := range h.patterns {
		// Try exact match.
		matched, err := doublestar.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}

		// For directories, also check with trailing slash
		// This handles patterns like "build/" which should only match directories.
		if isDir {
			matched, err = doublestar.Match(pattern, relPath+"/")
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// loadIgnoreFile loads patterns from an ignore file (.gitignore or .spinignore).
// Returns an error if the file exists but cannot be read.
// Missing files return os.ErrNotExist which should be handled by the caller.
func (h *IgnoreHandler) loadIgnoreFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines.
		if line == "" {
			continue
		}

		// Skip comment lines.
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Add pattern.
		h.patterns = append(h.patterns, line)
	}

	return scanner.Err()
}

// defaultPatterns returns the default ignore patterns that are always included.
// These cover common directories and files that should typically be ignored:
// - .git: Git internal files
// - node_modules: Node.js dependencies
// - .spin: Spin internal directory
// - vendor: Go vendor directory
// - __pycache__: Python cache
// - .vscode, .idea: IDE settings
// - *.pyc, *.pyo: Python bytecode
// - .DS_Store, Thumbs.db: OS-specific files
// - .gitignore, .spinignore: Ignore files themselves.
func defaultPatterns() []string {
	return []string{
		".git/**",
		".gitignore",
		".spinignore",
		"node_modules/**",
		".spin/**",
		"vendor/**",
		"__pycache__/**",
		".vscode/**",
		".idea/**",
		"*.pyc",
		"*.pyo",
		".DS_Store",
		"Thumbs.db",
	}
}
