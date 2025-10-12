package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

// CurrentBranch returns information about the current branch.
// Returns ErrDetachedHead if HEAD is detached.
//
// Example:
//
//	branch, err := repo.CurrentBranch(ctx)
//	if err != nil {
//	    if errors.Is(err, git.ErrDetachedHead) {
//	        // Handle detached HEAD
//	    }
//	    return err
//	}
//	fmt.Printf("Current branch: %s\n", branch.Name)
func (r *Repository) CurrentBranch(ctx context.Context) (*BranchInfo, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return nil, ErrDetachedHead
	}

	branch := &BranchInfo{
		Name:     head.Name().Short(),
		FullName: head.Name().String(),
		Hash:     head.Hash().String(),
	}

	return branch, nil
}

// ListBranches returns all local branches.
//
// Example:
//
//	branches, err := repo.ListBranches(ctx)
//	if err != nil {
//	    return err
//	}
//	for _, branch := range branches {
//	    fmt.Printf("Branch: %s (hash: %s)\n", branch.Name, branch.Hash)
//	}
func (r *Repository) ListBranches(ctx context.Context) ([]*BranchInfo, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	branches := make([]*BranchInfo, 0)

	refs, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		branch := &BranchInfo{
			Name:     ref.Name().Short(),
			FullName: ref.Name().String(),
			Hash:     ref.Hash().String(),
		}

		branches = append(branches, branch)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("iterate branches: %w", err)
	}

	return branches, nil
}

// ListRemoteBranches returns all remote branches.
func (r *Repository) ListRemoteBranches(ctx context.Context) ([]*BranchInfo, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	branches := make([]*BranchInfo, 0)

	refs, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Only include remote branches
		if ref.Name().IsRemote() {
			branch := &BranchInfo{
				Name:     ref.Name().Short(),
				FullName: ref.Name().String(),
				Hash:     ref.Hash().String(),
			}
			branches = append(branches, branch)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("iterate references: %w", err)
	}

	return branches, nil
}
