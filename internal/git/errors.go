package git

import "errors"

// Sentinel errors for common failure modes
var (
	// ErrNotRepository indicates the path is not a Git repository
	ErrNotRepository = errors.New("not a git repository")

	// ErrInvalidRemote indicates the specified remote doesn't exist
	ErrInvalidRemote = errors.New("remote not found")

	// ErrInvalidBranch indicates the specified branch doesn't exist
	ErrInvalidBranch = errors.New("branch not found")

	// ErrInvalidRef indicates the specified reference is invalid
	ErrInvalidRef = errors.New("invalid reference")

	// ErrDetachedHead indicates operation requires a branch but HEAD is detached
	ErrDetachedHead = errors.New("detached HEAD state")
)
