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
// It uses os.MkdirTemp with the system temp dir to ensure the directory
// is outside any parent git repository (since GOTMPDIR may be set to
// a directory inside the project).
func setupNonRepo(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "spin-test-nonrepo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	return tmpDir
}

// Test Discovery.

func TestDiscover(t *testing.T) {
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
			name: "nested directory",
			setup: func(t *testing.T) string {
				tmpDir := setupTestRepo(t)

				nestedDir := filepath.Join(tmpDir, "subdir", "deep")
				err := os.MkdirAll(nestedDir, 0755)
				if err != nil {
					t.Fatalf("failed to create nested dir: %v", err)
				}

				return nestedDir
			},
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
			startPath := tt.setup(t)
			ctx := context.Background()

			repo, err := Discover(ctx, startPath)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)

					return
				}
				// Check if error is or wraps the expected error.
				if !isError(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if repo == nil {
				t.Fatal("expected repo, got nil")
			}

			if repo.Root() == "" {
				t.Error("expected non-empty root path")
			}

			// Verify root is absolute path.
			if !filepath.IsAbs(repo.Root()) {
				t.Errorf("expected absolute path, got %s", repo.Root())
			}
		})
	}
}

func TestDiscoverCancellation(t *testing.T) {
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
	tests := []struct {
		name string
		info ContextInfo
	}{
		{
			name: "not a repository",
			info: ContextInfo{
				GitEnabled: false,
				IsRepo:     false,
			},
		},
		{
			name: "clean repository",
			info: ContextInfo{
				GitEnabled: true,
				IsRepo:     true,
				Branch:     "main",
				Commit:     "abc123",
				IsClean:    true,
			},
		},
		{
			name: "dirty repository",
			info: ContextInfo{
				GitEnabled:     true,
				IsRepo:         true,
				Branch:         "feature/test",
				Remote:         "origin/feature/test",
				Commit:         "def456",
				IsClean:        false,
				ModifiedFiles:  3,
				UntrackedFiles: 2,
				Ahead:          1,
				Behind:         0,
				Detached:       false,
			},
		},
		{
			name: "detached HEAD",
			info: ContextInfo{
				GitEnabled: true,
				IsRepo:     true,
				Commit:     "xyz789",
				Detached:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON.
			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal back.
			var decoded ContextInfo
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Verify fields match.
			if decoded.GitEnabled != tt.info.GitEnabled {
				t.Errorf("GitEnabled mismatch: got %v, want %v", decoded.GitEnabled, tt.info.GitEnabled)
			}

			if decoded.IsRepo != tt.info.IsRepo {
				t.Errorf("IsRepo mismatch: got %v, want %v", decoded.IsRepo, tt.info.IsRepo)
			}

			if decoded.Branch != tt.info.Branch {
				t.Errorf("Branch mismatch: got %q, want %q", decoded.Branch, tt.info.Branch)
			}

			if decoded.Remote != tt.info.Remote {
				t.Errorf("Remote mismatch: got %q, want %q", decoded.Remote, tt.info.Remote)
			}

			if decoded.Commit != tt.info.Commit {
				t.Errorf("Commit mismatch: got %q, want %q", decoded.Commit, tt.info.Commit)
			}

			if decoded.IsClean != tt.info.IsClean {
				t.Errorf("IsClean mismatch: got %v, want %v", decoded.IsClean, tt.info.IsClean)
			}

			if decoded.ModifiedFiles != tt.info.ModifiedFiles {
				t.Errorf("ModifiedFiles mismatch: got %d, want %d", decoded.ModifiedFiles, tt.info.ModifiedFiles)
			}

			if decoded.UntrackedFiles != tt.info.UntrackedFiles {
				t.Errorf("UntrackedFiles mismatch: got %d, want %d", decoded.UntrackedFiles, tt.info.UntrackedFiles)
			}

			if decoded.Ahead != tt.info.Ahead {
				t.Errorf("Ahead mismatch: got %d, want %d", decoded.Ahead, tt.info.Ahead)
			}

			if decoded.Behind != tt.info.Behind {
				t.Errorf("Behind mismatch: got %d, want %d", decoded.Behind, tt.info.Behind)
			}

			if decoded.Detached != tt.info.Detached {
				t.Errorf("Detached mismatch: got %v, want %v", decoded.Detached, tt.info.Detached)
			}
		})
	}
}
