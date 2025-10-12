package git

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
)

// Status returns the current repository status including branch name,
// modified/staged/untracked files, and tracking information.
//
// Example:
//
//	status, err := repo.Status(ctx)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Branch: %s\n", status.Branch)
//	fmt.Printf("Modified: %d files\n", len(status.ModifiedFiles))
//	fmt.Printf("Untracked: %d files\n", len(status.UntrackedFiles))
func (r *Repository) Status(ctx context.Context) (*Status, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}

	// Get worktree status
	gitStatus, err := worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	// Get current HEAD
	head, err := r.repo.Head()
	if err != nil {
		// Detached HEAD or no HEAD (empty repo)
		return &Status{
			Detached:       true,
			ModifiedFiles:  make([]FileStatus, 0),
			UntrackedFiles: make([]string, 0),
		}, nil
	}

	status := &Status{
		Branch:         head.Name().Short(),
		Hash:           head.Hash().String(),
		ModifiedFiles:  make([]FileStatus, 0),
		UntrackedFiles: make([]string, 0),
		Detached:       !head.Name().IsBranch(),
	}

	// Parse file statuses
	for path, fileStatus := range gitStatus {
		if fileStatus.Worktree == gogit.Untracked {
			status.UntrackedFiles = append(status.UntrackedFiles, path)
		} else {
			status.ModifiedFiles = append(status.ModifiedFiles, FileStatus{
				Path:     path,
				Staging:  mapGoGitStatus(fileStatus.Staging),
				Worktree: mapGoGitStatus(fileStatus.Worktree),
			})
		}
	}

	// Get tracking branch and ahead/behind
	// This is a best-effort operation - if it fails, we continue without tracking info
	if head.Name().IsBranch() {
		remoteBranch, ahead, behind := r.getTrackingInfo(head.Name().Short())
		status.RemoteBranch = remoteBranch
		status.Ahead = ahead
		status.Behind = behind
	}

	return status, nil
}

// mapGoGitStatus maps go-git status code to our StatusCode
func mapGoGitStatus(status gogit.StatusCode) StatusCode {
	switch status {
	case gogit.Unmodified:
		return Unmodified
	case gogit.Modified:
		return Modified
	case gogit.Added:
		return Added
	case gogit.Deleted:
		return Deleted
	case gogit.Renamed:
		return Renamed
	case gogit.Copied:
		return Copied
	case gogit.Untracked:
		return Untracked
	default:
		return Unmodified
	}
}

// getTrackingInfo returns tracking branch name and ahead/behind counts
// Returns empty string and 0, 0 if no tracking branch
func (r *Repository) getTrackingInfo(branchName string) (remoteBranch string, ahead, behind int) {
	// This is a simplified implementation
	// Full implementation would query git config for branch.{branchName}.remote
	// and branch.{branchName}.merge, then calculate ahead/behind using log
	// For now, return empty tracking info
	return "", 0, 0
}
