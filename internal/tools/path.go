package tools

import "github.com/dmytrogajewski/spin/pkg/alg/pathx"

// resolvePath resolves a file path relative to workDir.
// If path is already absolute or workDir is empty, path is returned unchanged.
func resolvePath(path, workDir string) string {
	return pathx.ResolvePath(workDir, path)
}
