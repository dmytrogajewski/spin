package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Contain joins rel onto pluginRoot and returns an absolute path that stays
// inside the plugin root. rel must start with "./". Bare names and escapes fail.
func Contain(pluginRoot, rel string) (string, error) {
	if rel == "" {
		return "", ErrEmptyPath
	}

	if !strings.HasPrefix(rel, relPathPrefix) {
		return "", fmt.Errorf("%w: %q must start with %s", ErrNotPluginRelative, rel, relPathPrefix)
	}

	absRoot, err := filepath.Abs(pluginRoot)
	if err != nil {
		return "", fmt.Errorf("plugin root: %w", err)
	}

	joined := filepath.Join(absRoot, filepath.Clean(rel))

	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("plugin path: %w", err)
	}

	if !insideRoot(absRoot, absJoined) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}

	return resolveIfPresent(absRoot, absJoined, rel)
}

func resolveIfPresent(absRoot, absJoined, rel string) (string, error) {
	if !pathExists(absJoined) {
		return absJoined, nil
	}

	resolved, resolveErr := filepath.EvalSymlinks(absJoined)
	if resolveErr != nil {
		return "", fmt.Errorf("resolve %s: %w", rel, resolveErr)
	}

	if !insideRoot(absRoot, resolved) {
		return "", fmt.Errorf("%w: %s", ErrPathEscape, rel)
	}

	return resolved, nil
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)

	return err == nil
}

func insideRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}

	return rel != pathDotDot && !strings.HasPrefix(rel, pathDotDot+string(os.PathSeparator))
}
