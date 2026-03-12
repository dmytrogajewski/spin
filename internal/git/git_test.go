package git

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Test helpers.

// setupTestRepo creates a test Git repository with initial commit.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize repository.
	repo, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create initial file.
	filename := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(filename, []byte("# Test\n"), 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Add and commit.
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	_, err = w.Add("README.md")
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	_, err = w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	return tmpDir
}

// setupTestRepoWithBranches creates a test repo with multiple branches.

// setupTestRepoWithModifications creates a test repo with modified files.
func setupTestRepoWithModifications(t *testing.T) string {
	t.Helper()

	tmpDir := setupTestRepo(t)

	// Modify existing file.
	filename := filepath.Join(tmpDir, "README.md")
	err := os.WriteFile(filename, []byte("# Test\nModified\n"), 0644)
	if err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Create untracked file.
	newFile := filepath.Join(tmpDir, "untracked.txt")
	err = os.WriteFile(newFile, []byte("untracked\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	return tmpDir
}

// setupNonRepo creates a directory that is not a Git repository.
func setupNonRepo(t *testing.T) string {
	t.Helper()

	return t.TempDir()
}

// Test Discovery.

func TestDiscover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr error
	}{
		{
			name:    "valid repo root",
			setup:   setupTestRepo,
			wantErr: nil,
		},
		{
			name:    "nested directory",
			setup:   setupNestedRepo,
			wantErr: nil,
		},
		{
			name:    "not a repo",
			setup:   setupNonRepo,
			wantErr: ErrNotRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			startPath := tt.setup(t)
			repo, err := Discover(context.Background(), startPath)

			if tt.wantErr != nil {
				assertDiscoverError(t, err, tt.wantErr)
				return
			}

			assertDiscoverSuccess(t, repo, err)
		})
	}
}

func setupNestedRepo(t *testing.T) string {
	t.Helper()

	tmpDir := setupTestRepo(t)

	nestedDir := filepath.Join(tmpDir, "subdir", "deep")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	return nestedDir
}

func assertDiscoverError(t *testing.T, err, wantErr error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error %v, got nil", wantErr)
		return
	}

	if !isError(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}

func assertDiscoverSuccess(t *testing.T, repo *Repository, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo == nil {
		t.Fatal("expected repo, got nil")
	}

	if repo.Root() == "" {
		t.Error("expected non-empty root path")
	}

	if !filepath.IsAbs(repo.Root()) {
		t.Errorf("expected absolute path, got %s", repo.Root())
	}
}

func TestDiscoverCancellation(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := Discover(ctx, tmpDir)
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRepositoryRoot(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepo(t)

	nestedDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	ctx := context.Background()

	repo, err := Discover(ctx, nestedDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	root := repo.Root()
	if root == "" {
		t.Error("expected non-empty root")
	}

	// Root should be the original tmpDir, not the nested dir.
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}
}

// Benchmark tests.

func BenchmarkDiscover(b *testing.B) {
	tmpDir := setupTestRepo(&testing.T{})
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		_, err := Discover(ctx, tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiscoverNested(b *testing.B) {
	tmpDir := setupTestRepo(&testing.T{})
	defer os.RemoveAll(tmpDir)

	// Create deep nesting.
	nestedDir := tmpDir
	for range 10 {
		nestedDir = filepath.Join(nestedDir, "subdir")
	}

	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		_, err = Discover(ctx, nestedDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper function to check if error is or wraps expected error.
func isError(err, target error) bool {
	return errors.Is(err, target)
}

// Test GetContextInfo.

func TestGetContextInfo_NotRepository(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	gi := NewIntegration(true, t.TempDir(), logger)

	// Don't initialize - should not be a repository.
	info := gi.GetContextInfo()

	// Verify basic fields.
	if info.GitEnabled {
		t.Errorf("expected GitEnabled=false, got %v", info.GitEnabled)
	}

	if info.IsRepo {
		t.Errorf("expected IsRepo=false, got %v", info.IsRepo)
	}

	// Verify optional fields are empty.
	if info.Branch != "" {
		t.Errorf("expected empty Branch, got %q", info.Branch)
	}

	if info.Remote != "" {
		t.Errorf("expected empty Remote, got %q", info.Remote)
	}

	if info.Commit != "" {
		t.Errorf("expected empty Commit, got %q", info.Commit)
	}
}

func TestGetContextInfo_Repository(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepo(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	gi := NewIntegration(true, tmpDir, logger)

	ctx := context.Background()
	err := gi.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	info := gi.GetContextInfo()

	// Verify basic fields.
	if !info.GitEnabled {
		t.Errorf("expected GitEnabled=true, got %v", info.GitEnabled)
	}

	if !info.IsRepo {
		t.Errorf("expected IsRepo=true, got %v", info.IsRepo)
	}

	// Verify branch is set (default branch varies, but should not be empty).
	if info.Branch == "" {
		t.Error("expected non-empty Branch")
	}

	// Verify commit is set.
	if info.Commit == "" {
		t.Error("expected non-empty Commit")
	}

	// Verify clean repository.
	if !info.IsClean {
		t.Errorf("expected IsClean=true for clean repo, got %v", info.IsClean)
	}

	if info.ModifiedFiles != 0 {
		t.Errorf("expected ModifiedFiles=0, got %d", info.ModifiedFiles)
	}

	if info.UntrackedFiles != 0 {
		t.Errorf("expected UntrackedFiles=0, got %d", info.UntrackedFiles)
	}
}

func TestGetContextInfo_WithModifications(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepoWithModifications(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	gi := NewIntegration(true, tmpDir, logger)

	ctx := context.Background()
	err := gi.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	info := gi.GetContextInfo()

	// Verify repository fields.
	if !info.GitEnabled {
		t.Errorf("expected GitEnabled=true, got %v", info.GitEnabled)
	}

	if !info.IsRepo {
		t.Errorf("expected IsRepo=true, got %v", info.IsRepo)
	}

	// Verify dirty repository.
	if info.IsClean {
		t.Errorf("expected IsClean=false for dirty repo, got %v", info.IsClean)
	}

	// Should have at least 1 modified file.
	if info.ModifiedFiles == 0 {
		t.Error("expected ModifiedFiles > 0 for dirty repo")
	}

	// Should have at least 1 untracked file.
	if info.UntrackedFiles == 0 {
		t.Error("expected UntrackedFiles > 0 for dirty repo")
	}
}

func TestGetContextInfo_AllFields(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepoWithModifications(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	gi := NewIntegration(true, tmpDir, logger)

	ctx := context.Background()
	err := gi.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	info := gi.GetContextInfo()

	// Verify all boolean fields are set.
	if !info.GitEnabled {
		t.Error("expected GitEnabled=true")
	}

	if !info.IsRepo {
		t.Error("expected IsRepo=true")
	}

	// Verify string fields are set (when available).
	if info.Branch == "" {
		t.Error("expected Branch to be set")
	}

	if info.Commit == "" {
		t.Error("expected Commit to be set")
	}

	// Verify int fields make sense.
	if info.ModifiedFiles < 0 {
		t.Errorf("expected ModifiedFiles >= 0, got %d", info.ModifiedFiles)
	}

	if info.UntrackedFiles < 0 {
		t.Errorf("expected UntrackedFiles >= 0, got %d", info.UntrackedFiles)
	}

	if info.Ahead < 0 {
		t.Errorf("expected Ahead >= 0, got %d", info.Ahead)
	}

	if info.Behind < 0 {
		t.Errorf("expected Behind >= 0, got %d", info.Behind)
	}
}

func TestContextInfo_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info ContextInfo
	}{
		{
			name: "not a repository",
			info: ContextInfo{GitEnabled: false, IsRepo: false},
		},
		{
			name: "clean repository",
			info: ContextInfo{GitEnabled: true, IsRepo: true, Branch: "main", Commit: "abc123", IsClean: true},
		},
		{
			name: "dirty repository",
			info: ContextInfo{
				GitEnabled: true, IsRepo: true, Branch: "feature/test", Remote: "origin/feature/test",
				Commit: "def456", ModifiedFiles: 3, UntrackedFiles: 2, Ahead: 1,
			},
		},
		{
			name: "detached HEAD",
			info: ContextInfo{GitEnabled: true, IsRepo: true, Commit: "xyz789", Detached: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded ContextInfo
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			assertContextInfoEqual(t, decoded, tt.info)
		})
	}
}

func assertContextInfoEqual(t *testing.T, got, want ContextInfo) {
	t.Helper()

	if got.GitEnabled != want.GitEnabled {
		t.Errorf("GitEnabled mismatch: got %v, want %v", got.GitEnabled, want.GitEnabled)
	}

	if got.IsRepo != want.IsRepo {
		t.Errorf("IsRepo mismatch: got %v, want %v", got.IsRepo, want.IsRepo)
	}

	if got.Branch != want.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", got.Branch, want.Branch)
	}

	if got.Remote != want.Remote {
		t.Errorf("Remote mismatch: got %q, want %q", got.Remote, want.Remote)
	}

	if got.Commit != want.Commit {
		t.Errorf("Commit mismatch: got %q, want %q", got.Commit, want.Commit)
	}

	if got.IsClean != want.IsClean {
		t.Errorf("IsClean mismatch: got %v, want %v", got.IsClean, want.IsClean)
	}

	if got.ModifiedFiles != want.ModifiedFiles {
		t.Errorf("ModifiedFiles mismatch: got %d, want %d", got.ModifiedFiles, want.ModifiedFiles)
	}

	if got.UntrackedFiles != want.UntrackedFiles {
		t.Errorf("UntrackedFiles mismatch: got %d, want %d", got.UntrackedFiles, want.UntrackedFiles)
	}

	if got.Ahead != want.Ahead {
		t.Errorf("Ahead mismatch: got %d, want %d", got.Ahead, want.Ahead)
	}

	if got.Behind != want.Behind {
		t.Errorf("Behind mismatch: got %d, want %d", got.Behind, want.Behind)
	}

	if got.Detached != want.Detached {
		t.Errorf("Detached mismatch: got %v, want %v", got.Detached, want.Detached)
	}
}
