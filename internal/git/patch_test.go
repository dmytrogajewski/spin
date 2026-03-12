package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPatch_SimplePatch(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create initial file.
	testFile := filepath.Join(tmpDir, "hello.txt")
	err := os.WriteFile(testFile, []byte("Hello World\n"), 0644)
	require.NoError(t, err)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	patchText := `diff --git a/hello.txt b/hello.txt
index 557db03..980a0d5 100644
--- a/hello.txt
+++ b/hello.txt
@@ -1 +1 @@
-Hello World
+Hello Spin
`

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	result, err := repo.ApplyPatch(context.Background(), patchText, ApplyPatchOptions{})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "successfully")

	// Verify file changed.
	content, _ := os.ReadFile(testFile)
	assert.Equal(t, "Hello Spin\n", string(content))
}

func TestApplyPatch_AddFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	patchText := `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..ce01362
--- /dev/null
+++ b/new.txt
@@ -0,0 +1 @@
+hello
`

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	result, err := repo.ApplyPatch(context.Background(), patchText, ApplyPatchOptions{})

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify file was created.
	newFile := filepath.Join(tmpDir, "new.txt")
	assert.FileExists(t, newFile)
	content, _ := os.ReadFile(newFile)
	assert.Contains(t, string(content), "hello")
}

func TestApplyPatch_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create file to delete.
	testFile := filepath.Join(tmpDir, "delete.txt")
	os.WriteFile(testFile, []byte("to be deleted\n"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "add file").Run()

	patchText := `diff --git a/delete.txt b/delete.txt
deleted file mode 100644
index abc123..0000000
--- a/delete.txt
+++ /dev/null
@@ -1 +0,0 @@
-to be deleted
`

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	result, err := repo.ApplyPatch(context.Background(), patchText, ApplyPatchOptions{})

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify file was deleted.
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestApplyPatch_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line 1\nline 2\n"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	patchText := `diff --git a/test.txt b/test.txt
index abc..def 100644
--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
-line 1
+modified line 1
 line 2
`

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	result, err := repo.ApplyPatch(context.Background(), patchText, ApplyPatchOptions{
		DryRun: true,
	})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "dry-run")

	// Verify file was NOT modified.
	content, _ := os.ReadFile(testFile)
	assert.Equal(t, "line 1\nline 2\n", string(content))
}

func TestApplyPatch_EmptyPatch(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	result, err := repo.ApplyPatch(context.Background(), "", ApplyPatchOptions{})

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "Empty patch")
}

func TestApplyPatch_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	repo, err := Discover(context.Background(), tmpDir)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err = repo.ApplyPatch(ctx, "some patch", ApplyPatchOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestPatchError_Error(t *testing.T) {
	tests := []struct {
		name      string
		patchErr  *PatchError
		wantMatch string
	}{
		{
			name: "with file and line",
			patchErr: &PatchError{
				Message:  "Patch failed",
				FilePath: "src/main.go",
				Line:     42,
				Reason:   "Context not found",
			},
			wantMatch: "Patch failed (file: src/main.go, line: 42): Context not found",
		},
		{
			name: "with file only",
			patchErr: &PatchError{
				Message:  "Patch failed",
				FilePath: "test.txt",
				Reason:   "File does not exist",
			},
			wantMatch: "Patch failed (file: test.txt): File does not exist",
		},
		{
			name: "without file",
			patchErr: &PatchError{
				Message: "Patch failed",
				Reason:  "Invalid syntax",
			},
			wantMatch: "Patch failed: Invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.patchErr.Error()
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

// setupGitRepo initializes a git repository in the given directory.
func setupGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "git init failed")

	exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test User").Run()

	// Create initial commit.
	readmeFile := filepath.Join(dir, "README.md")
	os.WriteFile(readmeFile, []byte("# Test Repo\n"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Initial commit").Run()
}
