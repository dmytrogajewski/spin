package git

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Test helpers

// setupTestRepo creates a test Git repository with initial commit
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize repository
	repo, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create initial file
	filename := filepath.Join(tmpDir, "README.md")
	if err := ioutil.WriteFile(filename, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Add and commit
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if _, err := w.Add("README.md"); err != nil {
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

// setupTestRepoWithBranches creates a test repo with multiple branches
func setupTestRepoWithBranches(t *testing.T, branchNames []string) string {
	t.Helper()

	tmpDir := setupTestRepo(t)

	repo, err := gogit.PlainOpen(tmpDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}

	// Create branches
	for _, name := range branchNames {
		ref := head.Name().Short() + "_" + name
		if err := w.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(ref),
			Create: true,
		}); err != nil {
			t.Fatalf("failed to create branch %s: %v", ref, err)
		}
	}

	// Checkout back to main/master
	if err := w.Checkout(&gogit.CheckoutOptions{
		Branch: head.Name(),
	}); err != nil {
		t.Fatalf("failed to checkout main: %v", err)
	}

	return tmpDir
}

// setupTestRepoWithModifications creates a test repo with modified files
func setupTestRepoWithModifications(t *testing.T) string {
	t.Helper()

	tmpDir := setupTestRepo(t)

	// Modify existing file
	filename := filepath.Join(tmpDir, "README.md")
	if err := ioutil.WriteFile(filename, []byte("# Test\nModified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Create untracked file
	newFile := filepath.Join(tmpDir, "untracked.txt")
	if err := ioutil.WriteFile(newFile, []byte("untracked\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	return tmpDir
}

// setupNonRepo creates a directory that is not a Git repository
func setupNonRepo(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// Test Discovery

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
				if err := os.MkdirAll(nestedDir, 0755); err != nil {
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
				// Check if error is or wraps the expected error
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

			// Verify root is absolute path
			if !filepath.IsAbs(repo.Root()) {
				t.Errorf("expected absolute path, got %s", repo.Root())
			}
		})
	}
}

func TestDiscoverCancellation(t *testing.T) {
	tmpDir := setupTestRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := Discover(ctx, tmpDir)
	if err == nil {
		t.Error("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRepositoryRoot(t *testing.T) {
	tmpDir := setupTestRepo(t)
	nestedDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
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

	// Root should be the original tmpDir, not the nested dir
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}
}

// Benchmark tests

func BenchmarkDiscover(b *testing.B) {
	tmpDir := setupTestRepo(&testing.T{})
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Discover(ctx, tmpDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiscoverNested(b *testing.B) {
	tmpDir := setupTestRepo(&testing.T{})
	defer os.RemoveAll(tmpDir)

	// Create deep nesting
	nestedDir := tmpDir
	for i := 0; i < 10; i++ {
		nestedDir = filepath.Join(nestedDir, "subdir")
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Discover(ctx, nestedDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper function to check if error is or wraps expected error
func isError(err, target error) bool {
	return errors.Is(err, target)
}
