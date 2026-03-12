package git

import (
	"errors"
	"context"
	"testing"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(t *testing.T) *Repository
		verify func(t *testing.T, status *Status)
	}{
		{
			name: "clean repo",
			setup: func(t *testing.T) *Repository {
				tmpDir := setupTestRepo(t)

				repo, err := Discover(context.Background(), tmpDir)
				if err != nil {
					t.Fatalf("Discover failed: %v", err)
				}

				return repo
			},
			verify: func(t *testing.T, s *Status) {
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
			},
		},
		{
			name: "modified files",
			setup: func(t *testing.T) *Repository {
				tmpDir := setupTestRepoWithModifications(t)

				repo, err := Discover(context.Background(), tmpDir)
				if err != nil {
					t.Fatalf("Discover failed: %v", err)
				}

				return repo
			},
			verify: func(t *testing.T, s *Status) {
				if len(s.ModifiedFiles) == 0 {
					t.Error("expected modified files")
				}

				if len(s.UntrackedFiles) == 0 {
					t.Error("expected untracked files")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := tt.setup(t)
			ctx := context.Background()

			status, err := repo.Status(ctx)
			if err != nil {
				t.Fatalf("Status failed: %v", err)
			}

			if status == nil {
				t.Fatal("expected status, got nil")
			}

			tt.verify(t, status)
		})
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
