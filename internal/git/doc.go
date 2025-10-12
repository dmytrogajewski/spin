// Package git provides Git repository operations for the Spin AI coding agent.
//
// This package uses go-git (https://github.com/go-git/go-git) to provide
// pure-Go Git operations without requiring the git command-line tool.
//
// # Features
//
//   - Repository discovery by walking up directory tree
//   - Status queries (branch, modified/staged/untracked files, ahead/behind)
//   - Branch operations (list, current, remote tracking)
//   - Diff operations (between branches, commits, or working tree)
//   - Remote operations (list remotes, get URLs)
//   - Commit log queries
//   - Context-aware operations with cancellation support
//
// # Example Usage
//
//	// Discover repository
//	repo, err := git.Discover(ctx, "/path/to/project")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get status
//	status, err := repo.Status(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Branch: %s\n", status.Branch)
//	fmt.Printf("Modified: %d files\n", len(status.ModifiedFiles))
//
//	// List branches
//	branches, err := repo.ListBranches(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, branch := range branches {
//	    fmt.Printf("Branch: %s (hash: %s)\n", branch.Name, branch.Hash)
//	}
//
// # Thread Safety
//
// All operations are safe for concurrent read access. The underlying go-git
// library handles locking internally. Write operations (when added in future)
// will require additional synchronization.
//
// # Performance
//
// Operations are designed to be efficient for typical repositories:
//   - Discovery: <10ms for typical depth (<10 directories)
//   - Status: <100ms for repos with <1000 files
//   - ListBranches: <50ms for repos with <100 branches
//   - Diff: <200ms for diffs with <500 files
//
// For large repositories or large diffs, operations may take longer. All
// operations respect context cancellation and timeouts.
//
// # Error Handling
//
// The package defines sentinel errors for common failure modes:
//   - ErrNotRepository: path is not a Git repository
//   - ErrInvalidRemote: specified remote doesn't exist
//   - ErrInvalidBranch: specified branch doesn't exist
//   - ErrInvalidRef: specified reference is invalid
//   - ErrDetachedHead: operation requires branch but HEAD is detached
//
// All errors are wrapped with context for debugging.
package git
