package agentsmd

import (
	"context"
	"fmt"
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
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("discover agents.md: %w", err)
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	// 1. Check working directory first.
	if path := filepath.Join(absWorkDir, FileName); fileExists(path) {
		return path, nil
	}

	// 2. Check git root if available and different from workDir.
	if path := d.checkGitRoot(absWorkDir); path != "" {
		return path, nil
	}

	// 3. Walk parent directories.
	return d.walkParents(ctx, absWorkDir)
}

// checkGitRoot checks the git root directory for AGENTS.md if it differs from workDir.
func (d *DefaultDiscoverer) checkGitRoot(absWorkDir string) string {
	if d.gitRoot == "" {
		return ""
	}

	absGitRoot, err := filepath.Abs(d.gitRoot)
	if err != nil || absGitRoot == absWorkDir {
		return ""
	}

	if path := filepath.Join(absGitRoot, FileName); fileExists(path) {
		return path
	}

	return ""
}

// walkParents walks parent directories looking for AGENTS.md, stopping at the
// git root or filesystem root.
func (d *DefaultDiscoverer) walkParents(ctx context.Context, absWorkDir string) (string, error) {
	stopAt := d.absoluteGitRoot()

	current := absWorkDir
	for {
		parent := filepath.Dir(current)
		if parent == current {
			break
		}

		if stopAt != "" && parent == stopAt {
			break
		}

		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("discover agents.md walk: %w", err)
		}

		if path := filepath.Join(parent, FileName); fileExists(path) {
			return path, nil
		}

		current = parent
	}

	return "", nil
}

// absoluteGitRoot returns the absolute path of the git root, or empty string.
func (d *DefaultDiscoverer) absoluteGitRoot() string {
	if d.gitRoot == "" {
		return ""
	}

	abs, err := filepath.Abs(d.gitRoot)
	if err != nil {
		return ""
	}

	return abs
}

// fileExists checks if a file exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
