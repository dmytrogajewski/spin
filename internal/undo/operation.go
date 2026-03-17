// Package undo provides an in-memory operation log for file mutations,
// enabling single-step undo of agent file operations.
package undo

import "time"

// OperationType represents the kind of file mutation.
type OperationType int

const (
	// OpCreate indicates a new file was created.
	OpCreate OperationType = iota
	// OpModify indicates an existing file was modified.
	OpModify
	// OpDelete indicates a file was deleted.
	OpDelete
)

// String returns the string representation of the operation type.
func (o OperationType) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpModify:
		return "modify"
	case OpDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Operation records a single file mutation with before-content for undo.
type Operation struct {
	// Type is the kind of file mutation.
	Type OperationType
	// Path is the absolute file path that was mutated.
	Path string
	// BeforeContent holds the file content before the mutation.
	// Nil for create operations (file did not exist).
	BeforeContent []byte
	// Timestamp is when the operation was recorded.
	Timestamp time.Time
}
