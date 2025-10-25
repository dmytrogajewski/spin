package git

import (
	"errors"
	"fmt"
)

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

// PatchError represents an error when applying a patch
type PatchError struct {
	Message  string // error message
	FilePath string // file path where error occurred (optional)
	Line     int    // line number where error occurred (optional)
	Reason   string // reason for the error
}

// Error returns the error message
func (e *PatchError) Error() string {
	if e.FilePath != "" {
		if e.Line > 0 {
			return fmt.Sprintf("%s (file: %s, line: %d): %s", e.Message, e.FilePath, e.Line, e.Reason)
		}
		return fmt.Sprintf("%s (file: %s): %s", e.Message, e.FilePath, e.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Reason)
}
