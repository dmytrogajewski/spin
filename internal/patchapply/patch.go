package patchapply

import "errors"

// Errors.
var (
	// ErrInvalidThreshold indicates an invalid threshold value.
	ErrInvalidThreshold = errors.New("threshold must be between 0.0 and 1.0")
)

// Patch represents a complete patch with one or more file operations.
type Patch struct {
	Operations []FileOperation
}

// FileOperation is a union type for all file operations.
// All file operations must implement this interface.
type FileOperation interface {
	// Path returns the file path for this operation.
	Path() string
}

// AddFile represents an operation to create a new file with content.
type AddFile struct {
	FilePath string
	Lines    []string
}

func (a *AddFile) Path() string { return a.FilePath }

// DeleteFile represents an operation to delete an existing file.
type DeleteFile struct {
	FilePath string
}

func (d *DeleteFile) Path() string { return d.FilePath }

// UpdateFile represents an operation to modify an existing file.
// It may optionally include a new path for rename/move operations.
type UpdateFile struct {
	FilePath string
	NewPath  string // Optional, for move operations.
	Hunks    []Hunk
}

func (u UpdateFile) Path() string { return u.FilePath }

// Hunk represents a change section within a file update.
// Each hunk has an optional context header (e.g., "func MyFunc") and a list of changes.
type Hunk struct {
	Header  string // Optional context (e.g., "func MyFunc").
	Changes []LineChange
}

// LineChange represents a single line operation within a hunk.
type LineChange struct {
	Type LineChangeType
	Text string
}

// LineChangeType indicates the type of line change.
type LineChangeType int

const (
	// LineContext represents a context line (prefix: " ").
	LineContext LineChangeType = iota
	// LineDelete represents a line to be deleted (prefix: "-").
	LineDelete
	// LineInsert represents a line to be inserted (prefix: "+").
	LineInsert
)

// String returns a string representation of the LineChangeType for debugging.
func (t LineChangeType) String() string {
	switch t {
	case LineContext:
		return "context"
	case LineDelete:
		return "delete"
	case LineInsert:
		return "insert"
	default:
		return "unknown"
	}
}
