package agentsmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscover_WorkDir(t *testing.T) {
	t.Parallel()

	// Create temp directory with AGENTS.md.
	tempDir := t.TempDir()

	agentsPath := filepath.Join(tempDir, FileName)

	err := os.WriteFile(agentsPath, []byte("# Test Instructions"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	d := NewDiscoverer("")
	ctx := context.Background()

	path, err := d.Discover(ctx, tempDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if path != agentsPath {
		t.Errorf("Discover() = %v, want %v", path, agentsPath)
	}
}

func TestDiscover_GitRoot(t *testing.T) {
	t.Parallel()

	// Create temp directory structure.
	tempDir := t.TempDir()

	workDir := filepath.Join(tempDir, "subdir")

	err := os.MkdirAll(workDir, 0o750)
	if err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}

	// Put AGENTS.md in git root (tempDir), not workDir.
	agentsPath := filepath.Join(tempDir, FileName)

	err = os.WriteFile(agentsPath, []byte("# Git Root Instructions"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	d := NewDiscoverer(tempDir) // gitRoot = tempDir.
	ctx := context.Background()

	path, err := d.Discover(ctx, workDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if path != agentsPath {
		t.Errorf("Discover() = %v, want %v", path, agentsPath)
	}
}

func TestDiscover_ParentDir(t *testing.T) {
	t.Parallel()

	// Create temp directory structure.
	tempDir := t.TempDir()

	workDir := filepath.Join(tempDir, "level1", "level2")

	err := os.MkdirAll(workDir, 0o750)
	if err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}

	// Put AGENTS.md in parent (level1), not workDir.
	agentsPath := filepath.Join(tempDir, "level1", FileName)

	err = os.WriteFile(agentsPath, []byte("# Parent Instructions"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	d := NewDiscoverer("") // no git root.
	ctx := context.Background()

	path, err := d.Discover(ctx, workDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if path != agentsPath {
		t.Errorf("Discover() = %v, want %v", path, agentsPath)
	}
}

func TestDiscover_NotFound(t *testing.T) {
	t.Parallel()

	// Create temp directory without AGENTS.md
	// Use a deeply nested path to avoid accidentally finding a real AGENTS.md
	// when walking up the directory tree during tests.
	tempDir := t.TempDir()

	isolatedDir := filepath.Join(tempDir, "isolated", "deep", "path")

	err := os.MkdirAll(isolatedDir, 0o750)
	if err != nil {
		t.Fatalf("failed to create isolated dir: %v", err)
	}

	// Create a marker file at tempDir to stop the walk-up
	// by making tempDir look like a git root.
	d := NewDiscoverer(tempDir) // Set gitRoot to tempDir to limit walk-up.
	ctx := context.Background()

	path, err := d.Discover(ctx, isolatedDir)
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil", err)
	}

	if path != "" {
		t.Errorf("Discover() = %v, want empty string", path)
	}
}

func TestDiscover_ContextCanceled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	d := NewDiscoverer("")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := d.Discover(ctx, tempDir)
	if err == nil {
		t.Error("Discover() error = nil, want context.Canceled")
	}
}

func TestDiscover_WorkDirPriority(t *testing.T) {
	t.Parallel()

	// Create temp directory structure with AGENTS.md in both locations.
	tempDir := t.TempDir()

	workDir := filepath.Join(tempDir, "subdir")

	err := os.MkdirAll(workDir, 0o750)
	if err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}

	// AGENTS.md in both workDir and git root.
	workDirAgents := filepath.Join(workDir, FileName)
	gitRootAgents := filepath.Join(tempDir, FileName)

	err = os.WriteFile(workDirAgents, []byte("# WorkDir"), 0o600)
	if err != nil {
		t.Fatalf("failed to create workdir file: %v", err)
	}

	err = os.WriteFile(gitRootAgents, []byte("# GitRoot"), 0o600)
	if err != nil {
		t.Fatalf("failed to create git root file: %v", err)
	}

	d := NewDiscoverer(tempDir) // gitRoot = tempDir.
	ctx := context.Background()

	// WorkDir should take priority.
	path, err := d.Discover(ctx, workDir)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if path != workDirAgents {
		t.Errorf("Discover() = %v, want %v (workDir should have priority)", path, workDirAgents)
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Test file that exists.
	existingFile := filepath.Join(tempDir, "exists.txt")

	err := os.WriteFile(existingFile, []byte("content"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() = false for existing file")
	}

	// Test file that doesn't exist.
	if fileExists(filepath.Join(tempDir, "nonexistent.txt")) {
		t.Error("fileExists() = true for non-existent file")
	}

	// Test directory (should return false).
	if fileExists(tempDir) {
		t.Error("fileExists() = true for directory")
	}
}
