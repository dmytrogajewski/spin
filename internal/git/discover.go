package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
)

// Discover finds a git repository starting from the given path.
// It walks up the directory tree until a .git directory is found.
// Returns ErrNotRepository if no repository is found.
//
// The search respects context cancellation and will return ctx.Err() if
// the context is canceled during the search.
//
// Example:
//
//	repo, err := git.Discover(ctx, "/path/to/project/subdir")
//	if err != nil {
//	    if errors.Is(err, git.ErrNotRepository) {
//
// Not a Git repository
//
//	    }
//	    return err
//	}
//	fmt.Printf("Repository root: %s\n", repo.Root())
func Discover(ctx context.Context, startPath string) (*Repository, error) {
	// Normalize path to absolute.
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Walk up directory tree looking for .git.
	for {
		// Check context cancellation.
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("discover git repository: %w", ctx.Err())
		default:
		}

		// Try to open repository at current path.
		repo, openErr := gogit.PlainOpen(absPath)
		if openErr == nil {
			// Successfully opened repository.
			worktree, wtErr := repo.Worktree()
			if wtErr != nil {
				return nil, fmt.Errorf("get worktree: %w", wtErr)
			}

			return &Repository{
				repo: repo,
				root: worktree.Filesystem.Root(),
			}, nil
		}

		// Check if this was a "not a repository" error or something else.
		if !errors.Is(openErr, gogit.ErrRepositoryNotExists) {
			// Some other error occurred, return it.
			return nil, fmt.Errorf("open repository: %w", openErr)
		}

		// Not a repo at this level, try parent directory.
		parent := filepath.Dir(absPath)
		if parent == absPath {
			// Reached filesystem root without finding repository.
			return nil, fmt.Errorf("%w: %s", ErrNotRepository, startPath)
		}

		absPath = parent
	}
}

// Root returns the repository root path (absolute path).
func (r *Repository) Root() string {
	return r.root
}
