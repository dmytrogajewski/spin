package undo_test

// Journey: specs/journeys/JOURNEY-R5.2.md.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/undo"
)

const (
	testFileName    = "hello.txt"
	testFileContent = "hello world"
	testFileUpdated = "hello updated"
	testNewFile     = "new.txt"
	testNewContent  = "new file content"
)

// setupTestWorkDir creates a temp directory with a git repo for snapshot testing.
func setupTestWorkDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize a real git repo so shadow repo operations work.
	runGitInit(t, dir)

	// Create a test file.
	writeTestFile(t, dir, testFileName, testFileContent)

	return dir
}

// runGitInit initializes a git repo in the given directory.
func runGitInit(t *testing.T, dir string) {
	t.Helper()

	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "git", "init", dir)

	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init failed: %s", out)

	// Configure user for commits.
	configCmd := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "test@test.com")

	_, configErr := configCmd.CombinedOutput()
	require.NoError(t, configErr)

	nameCmd := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "Test")

	_, nameErr := nameCmd.CombinedOutput()
	require.NoError(t, nameErr)
}

// writeTestFile writes content to a file in the given directory.
func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	require.NoError(t, err)
}

// readTestFile reads content from a file in the given directory.
func readTestFile(t *testing.T, dir, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)

	return string(data)
}

func TestSnapshotManager_Init_CreatesShadowRepo(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	err := mgr.Init()
	require.NoError(t, err)

	// Shadow dir should exist with a HEAD file (bare repo marker).
	shadowDir := mgr.ShadowDir()
	require.DirExists(t, shadowDir)

	headPath := filepath.Join(shadowDir, "HEAD")
	require.FileExists(t, headPath)
}

func TestSnapshotManager_Snapshot_ReturnsHash(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	hash, err := mgr.Snapshot()
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Git tree hashes are 40 hex characters.
	const gitHashLen = 40

	require.Len(t, hash, gitHashLen)
}

func TestSnapshotManager_Snapshot_CapturesFileChanges(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Take snapshot of initial state.
	hash1, err := mgr.Snapshot()
	require.NoError(t, err)

	// Modify the file.
	writeTestFile(t, dir, testFileName, testFileUpdated)

	// Take another snapshot.
	hash2, err := mgr.Snapshot()
	require.NoError(t, err)

	// Hashes should differ because content changed.
	require.NotEqual(t, hash1, hash2)
}

func TestSnapshotManager_Restore_RevertsChanges(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Take snapshot of initial state.
	hash1, err := mgr.Snapshot()
	require.NoError(t, err)

	// Modify the file.
	writeTestFile(t, dir, testFileName, testFileUpdated)

	// Take snapshot of modified state (so shadow index is current).
	_, snapErr := mgr.Snapshot()
	require.NoError(t, snapErr)

	// Restore to initial state.
	require.NoError(t, mgr.Restore(hash1))

	// File should be back to original content.
	content := readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)
}

func TestSnapshotManager_Restore_HandlesNewFiles(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Take snapshot before new file.
	hash1, err := mgr.Snapshot()
	require.NoError(t, err)

	// Add a new file.
	writeTestFile(t, dir, testNewFile, testNewContent)

	// Take snapshot with new file (so index sees it).
	_, snapErr := mgr.Snapshot()
	require.NoError(t, snapErr)

	// Restore to before the new file.
	require.NoError(t, mgr.Restore(hash1))

	// New file should be gone.
	_, statErr := os.Stat(filepath.Join(dir, testNewFile))
	require.True(t, os.IsNotExist(statErr))
}

func TestSnapshotManager_Restore_HandlesDeletedFiles(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Take snapshot with the file.
	hash1, err := mgr.Snapshot()
	require.NoError(t, err)

	// Delete the file.
	require.NoError(t, os.Remove(filepath.Join(dir, testFileName)))

	// Take snapshot without the file.
	_, snapErr := mgr.Snapshot()
	require.NoError(t, snapErr)

	// Restore to when the file existed.
	require.NoError(t, mgr.Restore(hash1))

	// File should be back.
	content := readTestFile(t, dir, testFileName)
	require.Equal(t, testFileContent, content)
}

func TestSnapshotManager_SyncsGitignore(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	// Create a .gitignore.
	gitignoreContent := "*.log\ntmp/\n"
	writeTestFile(t, dir, ".gitignore", gitignoreContent)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Check that info/exclude has the gitignore content.
	excludePath := filepath.Join(mgr.ShadowDir(), "info", "exclude")
	content := readTestFile(t, filepath.Dir(excludePath), filepath.Base(excludePath))
	require.Equal(t, gitignoreContent, content)
}

func TestSnapshotManager_Cleanup_Succeeds(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	// Take a snapshot so there's something to GC.
	_, err := mgr.Snapshot()
	require.NoError(t, err)

	// Cleanup should not error.
	require.NoError(t, mgr.Cleanup())
}

func TestSnapshotManager_NotInitialized_ReturnsError(t *testing.T) {
	t.Parallel()

	mgr := undo.NewSnapshotManager(t.TempDir())

	_, err := mgr.Snapshot()
	require.ErrorIs(t, err, undo.ErrShadowRepoNotInitialized)

	require.ErrorIs(t, mgr.Restore("abc"), undo.ErrShadowRepoNotInitialized)
	require.ErrorIs(t, mgr.Cleanup(), undo.ErrShadowRepoNotInitialized)
}

func TestProjectHash_Deterministic(t *testing.T) {
	t.Parallel()

	hash1 := undo.ProjectHash("/some/path")
	hash2 := undo.ProjectHash("/some/path")

	require.Equal(t, hash1, hash2)
}

func TestProjectHash_DifferentPaths(t *testing.T) {
	t.Parallel()

	hash1 := undo.ProjectHash("/path/a")
	hash2 := undo.ProjectHash("/path/b")

	require.NotEqual(t, hash1, hash2)
}

func TestProjectHash_Length(t *testing.T) {
	t.Parallel()

	const expectedHashLen = 16

	hash := undo.ProjectHash("/any/path")

	require.Len(t, hash, expectedHashLen)
}

func TestSnapshotManager_SnapshotCount(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())
	require.Equal(t, 0, mgr.SnapshotCount())

	_, err := mgr.Snapshot()
	require.NoError(t, err)
	require.Equal(t, 1, mgr.SnapshotCount())
}

func TestSnapshotManager_GetSnapshot(t *testing.T) {
	t.Parallel()

	dir := setupTestWorkDir(t)

	mgr := undo.NewSnapshotManager(dir)
	require.NoError(t, mgr.Init())

	hash, err := mgr.Snapshot()
	require.NoError(t, err)

	got, getErr := mgr.GetSnapshot(0)
	require.NoError(t, getErr)
	require.Equal(t, hash, got)

	_, badErr := mgr.GetSnapshot(1)
	require.ErrorIs(t, badErr, undo.ErrSnapshotNotFound)
}
