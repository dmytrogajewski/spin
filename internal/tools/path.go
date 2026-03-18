package tools

import "path/filepath"

// resolvePath resolves a file path relative to workDir.
// If path is already absolute or workDir is empty, path is returned unchanged.
func resolvePath(path, workDir string) string {
	if filepath.IsAbs(path) || workDir == "" {
		return path
	}

	return filepath.Join(workDir, path)
}
