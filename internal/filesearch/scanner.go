package filesearch

import (
	"os"
	"path/filepath"
)

// Scanner scans directories for files.
type Scanner struct {
	baseDir   string
	ignoreGit bool
	maxDepth  int
}

// NewScanner creates a new file scanner.
func NewScanner(baseDir string, ignoreGit bool) *Scanner {
	return &Scanner{
		baseDir:   baseDir,
		ignoreGit: ignoreGit,
		maxDepth:  20, // Reasonable default to prevent deep recursion
	}
}

// Scan returns all files in the directory recursively.
// Returns relative paths from baseDir.
func (s *Scanner) Scan() ([]string, error) {
	var files []string

	err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip errors (permission denied, etc.)
			return nil
		}

		// Skip directories (we only want files)
		if d.IsDir() {
			// Skip .git directories if ignoreGit is true
			if s.ignoreGit && d.Name() == ".git" {
				return filepath.SkipDir
			}
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

		files = append(files, relPath)
		return nil
	})

	return files, err
}
