package memory

import "errors"

// Sentinel errors for memory operations.
var (
	// ErrNotFound indicates the requested key does not exist.
	ErrNotFound = errors.New("memory: key not found")

	// ErrKeyExists indicates the key already exists (when Overwrite is false).
	ErrKeyExists = errors.New("memory: key already exists")

	// ErrEmptyKey indicates an empty key was provided.
	ErrEmptyKey = errors.New("memory: key cannot be empty")
)
