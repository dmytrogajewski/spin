package pathx

import (
	"os"
	"path/filepath"
)

// unsafeDirs is the set of directory paths that are too broad to operate on.
var unsafeDirs = map[string]bool{
	"/":    true,
	"/tmp": true,
	"/var": true,
}

// IsUnsafeWorkDir returns true if workDir is a dangerous directory to
// perform bulk operations on (filesystem root, /tmp, /var, or $HOME).
func IsUnsafeWorkDir(workDir string) bool {
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return false
	}

	if unsafeDirs[absDir] {
		return true
	}

	// Reject user's home directory.
	homeDir, homeErr := os.UserHomeDir()

	return homeErr == nil && absDir == homeDir
}
