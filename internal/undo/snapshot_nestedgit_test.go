package undo_test

// Journey: specs/journeys/JOURNEY-2.1.md.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/undo"
)

// TestSnapshotManager_NestedGitNoCommit_SnapshotSucceeds reproduces the bug
// where `git add -A` fails when the work tree contains a subdirectory with
// a .git directory but no commits (e.g., `cargo new` creates a git-initialized
// project with no initial commit).
//
// Error before fix:
//
//	git add: error: 'sub_project/' does not have a commit checked out
//	fatal: adding files failed
//	exit status 128
func TestSnapshotManager_NestedGitNoCommit_SnapshotSucceeds(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Set up a work directory with a file.
	workDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "root.txt"), []byte("root"), 0o600))

	// Create a nested directory with its own .git but NO commits.
	// This is what `cargo new` produces — git init with no commit.
	nestedDir := filepath.Join(workDir, "sub_project")
	require.NoError(t, os.MkdirAll(nestedDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "main.rs"), []byte("fn main() {}"), 0o600))

	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = nestedDir

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init failed: %s", out)

	// Now try to snapshot — this MUST NOT fail.
	mgr := undo.NewSnapshotManager(workDir)
	require.NoError(t, mgr.Init(context.Background()))

	hash, snapErr := mgr.Snapshot()
	require.NoError(t, snapErr, "Snapshot should succeed even with nested .git dir that has no commits")
	require.NotEmpty(t, hash)
}

// TestSnapshotManager_NestedGitWithCommit_TracksRootFiles verifies that
// root-level files are still tracked correctly when a nested git repo exists.
// Note: git treats nested repos with commits as submodule pointers — individual
// files inside them are NOT tracked by the shadow repo. This is expected.
func TestSnapshotManager_NestedGitWithCommit_TracksRootFiles(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	workDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(workDir, "root.txt"), []byte("original"), 0o600))

	// Create nested dir with committed git repo.
	nestedDir := filepath.Join(workDir, "sub_project")
	require.NoError(t, os.MkdirAll(nestedDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "main.rs"), []byte("fn main() {}"), 0o600))

	initGitWithCommit(t, nestedDir)

	mgr := undo.NewSnapshotManager(workDir)
	require.NoError(t, mgr.Init(context.Background()))

	// Snapshot 1.
	hash1, err := mgr.Snapshot()
	require.NoError(t, err)

	// Modify ROOT file (not nested).
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "root.txt"), []byte("modified"), 0o600))

	// Snapshot 2 — should differ because root file changed.
	hash2, err := mgr.Snapshot()
	require.NoError(t, err)
	require.NotEqual(t, hash1, hash2, "snapshots should differ after modifying root file")

	// Restore to snapshot 1.
	require.NoError(t, mgr.Restore(hash1))

	// Verify root file is restored.
	content, readErr := os.ReadFile(filepath.Join(workDir, "root.txt"))
	require.NoError(t, readErr)
	require.Equal(t, "original", string(content), "root file should be restored to original")
}

// TestSnapshotManager_UnsafeWorkDir_RejectsHome verifies that Init() refuses
// to snapshot the user's home directory to prevent catastrophic git add on $HOME.
func TestSnapshotManager_UnsafeWorkDir_RejectsHome(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	mgr := undo.NewSnapshotManager(homeDir)
	initErr := mgr.Init(context.Background())

	require.ErrorIs(t, initErr, undo.ErrUnsafeWorkDir)
}

// TestSnapshotManager_UnsafeWorkDir_RejectsRoot verifies that Init() refuses /.
func TestSnapshotManager_UnsafeWorkDir_RejectsRoot(t *testing.T) {
	t.Parallel()

	mgr := undo.NewSnapshotManager("/")
	initErr := mgr.Init(context.Background())

	require.ErrorIs(t, initErr, undo.ErrUnsafeWorkDir)
}

// initGitWithCommit initializes a git repo with an initial commit.
func initGitWithCommit(t *testing.T, dir string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "init"},
	}

	for _, args := range cmds {
		cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
		cmd.Dir = dir

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git command %v failed: %s", args, out)
	}
}
