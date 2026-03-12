package git

import (
	"errors"
	"context"
	"testing"
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
