package git

import (
	gogit "github.com/go-git/go-git/v5"
)

// Repository represents a Git repository
type Repository struct {
	repo *gogit.Repository // underlying go-git repository
	root string            // absolute path to repository root
}

// Status represents repository status
type Status struct {
	Branch         string       // current branch name (empty if detached)
	RemoteBranch   string       // upstream branch (e.g., "origin/main")
	Ahead          int          // commits ahead of remote
	Behind         int          // commits behind remote
	ModifiedFiles  []FileStatus // modified/staged files
	UntrackedFiles []string     // untracked files
	Detached       bool         // true if in detached HEAD state
	Hash           string       // current commit hash
}

// FileStatus represents a file's status in the repository
type FileStatus struct {
	Path     string     // file path relative to repository root
	Staging  StatusCode // staging area status
	Worktree StatusCode // working tree status
}

// StatusCode represents file status
type StatusCode int

const (
	// Unmodified indicates no changes
	Unmodified StatusCode = iota
	// Modified indicates file has been modified
	Modified
	// Added indicates file has been added
	Added
	// Deleted indicates file has been deleted
	Deleted
	// Renamed indicates file has been renamed
	Renamed
	// Copied indicates file has been copied
	Copied
	// Untracked indicates file is not tracked by Git
	Untracked
)

// String returns string representation of StatusCode
func (s StatusCode) String() string {
	switch s {
	case Unmodified:
		return "unmodified"
	case Modified:
		return "modified"
	case Added:
		return "added"
	case Deleted:
		return "deleted"
	case Renamed:
		return "renamed"
	case Copied:
		return "copied"
	case Untracked:
		return "untracked"
	default:
		return "unknown"
	}
}

// Diff represents changes between commits/branches
type Diff struct {
	From  string       // source ref/commit
	To    string       // target ref/commit
	Files []FileChange // changed files
}

// FileChange represents a single file change in a diff
type FileChange struct {
	Status  string // A=added, M=modified, D=deleted, R=renamed, C=copied
	Path    string // file path (relative to repo root)
	OldPath string // old path (for renames/copies)
	Patch   string // optional: unified diff patch
}

// BranchInfo represents branch information
type BranchInfo struct {
	Name     string // branch name (short form: "main")
	FullName string // full ref name: "refs/heads/main"
	Hash     string // current commit hash
	Remote   string // upstream remote (if tracking)
}

// RemoteInfo represents remote information
type RemoteInfo struct {
	Name string   // remote name (e.g., "origin")
	URLs []string // remote URLs (fetch/push)
}

// CommitInfo represents commit information
type CommitInfo struct {
	Hash      string // commit hash
	Author    string // author name
	Email     string // author email
	Message   string // commit message
	Timestamp string // commit timestamp (ISO 8601 format)
}
