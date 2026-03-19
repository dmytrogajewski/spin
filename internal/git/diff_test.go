package git

// Journey: specs/journeys/JOURNEY-T5.md.

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// testLogger returns a no-op logger for tests.
func testLogger() *slog.Logger {
	return slog.Default()
}

// TestGetDiff_StagedChanges_ShowsDiff reproduces the bug where git_operation(get_diff)
// returned empty after staging a new file. The root cause was using "git diff" (unstaged
// only) instead of "git diff HEAD" (both staged and unstaged).
func TestGetDiff_StagedChanges_ShowsDiff(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := setupTestRepo(t)

	integration := NewIntegration(true, repoDir, testLogger())
	require.NoError(t, integration.Initialize(context.Background()))

	// Create and stage a new file.
	newFile := filepath.Join(repoDir, "feature.md")
	require.NoError(t, os.WriteFile(newFile, []byte("new feature\n"), 0o600))

	stageCmd := exec.CommandContext(context.Background(), "git", "add", "feature.md")
	stageCmd.Dir = repoDir

	out, err := stageCmd.CombinedOutput()
	require.NoError(t, err, "git add failed: %s", out)

	// GetDiff must show the staged change — this was empty before the fix.
	diff, diffErr := integration.GetDiff(context.Background(), "")
	require.NoError(t, diffErr)
	require.Contains(t, diff, "feature.md", "diff should include the staged file")
	require.Contains(t, diff, "+new feature", "diff should include the added content")
}

// TestGetDiff_UnstagedChanges_ShowsDiff verifies unstaged changes still appear.
func TestGetDiff_UnstagedChanges_ShowsDiff(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := setupTestRepo(t)

	integration := NewIntegration(true, repoDir, testLogger())
	require.NoError(t, integration.Initialize(context.Background()))

	// Modify an existing tracked file without staging.
	readmePath := filepath.Join(repoDir, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# Modified\n"), 0o600))

	diff, diffErr := integration.GetDiff(context.Background(), "")
	require.NoError(t, diffErr)
	require.Contains(t, diff, "README.md", "diff should include modified file")
	require.Contains(t, diff, "+# Modified", "diff should include the change")
}

// TestGetDiff_SpecificFile_FiltersDiff verifies file path filtering.
func TestGetDiff_SpecificFile_FiltersDiff(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := setupTestRepo(t)

	integration := NewIntegration(true, repoDir, testLogger())
	require.NoError(t, integration.Initialize(context.Background()))

	// Modify README and create a new file.
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Changed\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "other.txt"), []byte("other\n"), 0o600))

	stageCmd := exec.CommandContext(context.Background(), "git", "add", "other.txt")
	stageCmd.Dir = repoDir

	out, err := stageCmd.CombinedOutput()
	require.NoError(t, err, "git add failed: %s", out)

	// Filter to only README.md.
	diff, diffErr := integration.GetDiff(context.Background(), "README.md")
	require.NoError(t, diffErr)
	require.Contains(t, diff, "README.md")
	require.NotContains(t, diff, "other.txt", "filtered diff should not include other files")
}

// TestGetDiff_CleanRepo_ReturnsEmpty verifies clean state returns empty diff.
func TestGetDiff_CleanRepo_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := setupTestRepo(t)

	integration := NewIntegration(true, repoDir, testLogger())
	require.NoError(t, integration.Initialize(context.Background()))

	diff, diffErr := integration.GetDiff(context.Background(), "")
	require.NoError(t, diffErr)
	require.Empty(t, diff, "clean repo should have empty diff")
}
