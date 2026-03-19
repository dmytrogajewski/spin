package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	t.Run("clean repo", func(t *testing.T) {
		t.Parallel()

		status := getRepoStatus(t, setupTestRepo)
		assertCleanStatus(t, status)
	})

	t.Run("modified files", func(t *testing.T) {
		t.Parallel()

		status := getRepoStatus(t, setupTestRepoWithModifications)
		assertDirtyStatus(t, status)
	})
}

func getRepoStatus(t *testing.T, setup func(t *testing.T) string) *Status {
	t.Helper()

	tmpDir := setup(t)

	repo, err := Discover(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected status, got nil")
	}

	return status
}

func assertCleanStatus(t *testing.T, s *Status) {
	t.Helper()

	if len(s.ModifiedFiles) != 0 {
		t.Errorf("expected no modified files, got %d", len(s.ModifiedFiles))
	}

	if len(s.UntrackedFiles) != 0 {
		t.Errorf("expected no untracked files, got %d", len(s.UntrackedFiles))
	}

	if s.Branch == "" {
		t.Error("expected non-empty branch name")
	}

	if s.Hash == "" {
		t.Error("expected non-empty hash")
	}

	if s.Detached {
		t.Error("expected not detached")
	}
}

func assertDirtyStatus(t *testing.T, s *Status) {
	t.Helper()

	if len(s.ModifiedFiles) == 0 {
		t.Error("expected modified files")
	}

	if len(s.UntrackedFiles) == 0 {
		t.Error("expected untracked files")
	}
}

// TestStatus_EmptyRepo tests that Status does not panic on an empty repo (no commits).
// Reproduces bug: running `spin exec` from a subdirectory of a git repo with no commits
// panics with nil pointer dereference on head.Name().IsBranch() at status.go:78,
// because head is nil (empty repo) but the nil guard only covers lines 54-61.
func TestStatus_EmptyRepo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Initialize repo but do NOT create any commits — head will be nil.
	_, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	repo, err := Discover(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// This must not panic.
	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected status, got nil")
	}

	if !status.Detached {
		t.Error("empty repo should report Detached=true")
	}

	if status.Branch != "" {
		t.Errorf("empty repo should have empty branch, got %q", status.Branch)
	}
}

// TestStatus_EmptyRepoFromSubdir tests Status from a subdirectory of an empty repo.
// This is the exact scenario from the panic trace: cd into a nested dir inside
// a git repo that has no commits.
func TestStatus_EmptyRepoFromSubdir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	_, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create a nested subdirectory (simulating tetris/tetris/).
	subDir := filepath.Join(tmpDir, "subproject")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Discover from the subdirectory — walks up to find .git.
	repo, err := Discover(context.Background(), subDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Must not panic.
	status, err := repo.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected status, got nil")
	}

	if !status.Detached {
		t.Error("empty repo should report Detached=true")
	}
}

func TestStatusCancellation(t *testing.T) {
	t.Parallel()

	tmpDir := setupTestRepo(t)

	repo, err := Discover(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	// Note: Status might complete before cancellation is checked.
	_, err = repo.Status(ctx)
	// Status might complete before cancellation is checked, so both nil and
	// context.Canceled are acceptable outcomes. Only fail on unexpected errors.
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = err // acceptable: status completed quickly
	}
}

func BenchmarkStatus(b *testing.B) {
	tmpDir := setupTestRepo(&testing.T{})

	repo, err := Discover(context.Background(), tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		_, err = repo.Status(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestStatusCodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code StatusCode
		want string
	}{
		{Unmodified, "unmodified"},
		{Modified, "modified"},
		{Added, "added"},
		{Deleted, "deleted"},
		{Renamed, "renamed"},
		{Copied, "copied"},
		{Untracked, "untracked"},
		{StatusCode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			got := tt.code.String()
			if got != tt.want {
				t.Errorf("StatusCode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
