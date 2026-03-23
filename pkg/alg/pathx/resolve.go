package pathx

import "path/filepath"

// ResolvePath resolves path against workDir. If path is absolute or
// workDir is empty, path is returned unchanged. Otherwise the result
// is [filepath.Join](workDir, path).
func ResolvePath(workDir, path string) string {
	if filepath.IsAbs(path) || workDir == "" {
		return path
	}

	return filepath.Join(workDir, path)
}
