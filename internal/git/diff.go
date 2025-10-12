package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

// DiffToBranch returns the diff from current HEAD to specified branch.
//
// Example:
//
//	diff, err := repo.DiffToBranch(ctx, "main")
//	if err != nil {
//	    return err
//	}
//	for _, file := range diff.Files {
//	    fmt.Printf("%s %s\n", file.Status, file.Path)
//	}
func (r *Repository) DiffToBranch(ctx context.Context, branch string) (*Diff, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve current HEAD
	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	// Resolve target branch
	branchRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidBranch, branch)
	}

	return r.diffBetweenHashes(ctx, head.Hash(), branchRef.Hash())
}

// DiffToCommit returns the diff from current HEAD to specified commit.
func (r *Repository) DiffToCommit(ctx context.Context, commit string) (*Diff, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve current HEAD
	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	// Parse commit hash
	commitHash := plumbing.NewHash(commit)

	return r.diffBetweenHashes(ctx, head.Hash(), commitHash)
}

// DiffBetween returns the diff between two refs/commits.
func (r *Repository) DiffBetween(ctx context.Context, from, to string) (*Diff, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve from ref
	fromRef, err := r.repo.Reference(plumbing.ReferenceName(from), true)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRef, from)
	}

	// Resolve to ref
	toRef, err := r.repo.Reference(plumbing.ReferenceName(to), true)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRef, to)
	}

	return r.diffBetweenHashes(ctx, fromRef.Hash(), toRef.Hash())
}

// diffBetweenHashes computes diff between two commit hashes
func (r *Repository) diffBetweenHashes(ctx context.Context, from, to plumbing.Hash) (*Diff, error) {
	// Get commits
	fromCommit, err := r.repo.CommitObject(from)
	if err != nil {
		return nil, fmt.Errorf("get from commit: %w", err)
	}

	toCommit, err := r.repo.CommitObject(to)
	if err != nil {
		return nil, fmt.Errorf("get to commit: %w", err)
	}

	// Compute patch
	patch, err := fromCommit.Patch(toCommit)
	if err != nil {
		return nil, fmt.Errorf("compute patch: %w", err)
	}

	// Parse file changes from patch
	fileChanges := make([]FileChange, 0)
	for _, filePatch := range patch.FilePatches() {
		fromFile, toFile := filePatch.Files()

		var status string
		var path string
		var oldPath string

		if fromFile == nil && toFile != nil {
			// Added
			status = "A"
			path = toFile.Path()
		} else if fromFile != nil && toFile == nil {
			// Deleted
			status = "D"
			path = fromFile.Path()
		} else if fromFile != nil && toFile != nil {
			if fromFile.Path() != toFile.Path() {
				// Renamed
				status = "R"
				path = toFile.Path()
				oldPath = fromFile.Path()
			} else {
				// Modified
				status = "M"
				path = toFile.Path()
			}
		}

		fileChanges = append(fileChanges, FileChange{
			Status:  status,
			Path:    path,
			OldPath: oldPath,
			Patch:   "", // Patch text can be added later if needed
		})
	}

	return &Diff{
		From:  from.String(),
		To:    to.String(),
		Files: fileChanges,
	}, nil
}
