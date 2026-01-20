package agentsmd

import (
	"context"
	"os"
	"path/filepath"
)

const (
	// FileName is the standard name for agent instruction files.
	FileName = "AGENTS.md"
)

// Discoverer finds AGENTS.md files in the filesystem.
type Discoverer interface {
	// Discover finds AGENTS.md starting from workDir.
	// Returns the path to the file, or empty string if not found.
	// Returns an error only for filesystem errors, not for missing files.
	Discover(ctx context.Context, workDir string) (string, error)
}

// DefaultDiscoverer implements Discoverer using filesystem walking.
type DefaultDiscoverer struct {
	// gitRoot is the git repository root, if known.
	// This is checked after workDir but before parent directories.
	gitRoot string
}

// NewDiscoverer creates a new DefaultDiscoverer.
// gitRoot is optional; pass empty string if not in a git repository.
func NewDiscoverer(gitRoot string) *DefaultDiscoverer {
	return &DefaultDiscoverer{
		gitRoot: gitRoot,
	}
}

// Discover finds AGENTS.md using this search order:
//  1. Working directory
//  2. Git repository root (if in a repo and different from workDir)
//  3. Parent directories up to filesystem root
//
// Returns empty string if not found (this is not an error).
func (d *DefaultDiscoverer) Discover(ctx context.Context, workDir string) (string, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return "", err
	}

	// Normalize workDir to absolute path
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}

	// 1. Check working directory first
	path := filepath.Join(absWorkDir, FileName)
	if fileExists(path) {
		return path, nil
	}

	// 2. Check git root if available and different from workDir
	if d.gitRoot != "" {
		absGitRoot, err := filepath.Abs(d.gitRoot)
		if err == nil && absGitRoot != absWorkDir {
			path = filepath.Join(absGitRoot, FileName)
			if fileExists(path) {
				return path, nil
			}
		}
	}

	// 3. Walk up to filesystem root (but stop at gitRoot if specified)
	var stopAt string
	if d.gitRoot != "" {
		stopAt, _ = filepath.Abs(d.gitRoot)
	}

	current := absWorkDir
	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}

		// Stop at git root if specified (already checked git root above)
		if stopAt != "" && parent == stopAt {
			break
		}

		// Check context cancellation periodically
		if err := ctx.Err(); err != nil {
			return "", err
		}

		path = filepath.Join(parent, FileName)
		if fileExists(path) {
			return path, nil
		}
		current = parent
	}

	// Not found (this is not an error)
	return "", nil
}

// fileExists checks if a file exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
