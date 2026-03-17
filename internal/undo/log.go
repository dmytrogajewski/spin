package undo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MaxEntries is the maximum number of operations retained before FIFO eviction.
const MaxEntries = 50

// ErrEmptyLog is returned when undo is called on an empty log.
var ErrEmptyLog = errors.New("operation log is empty; nothing to undo")

// OperationLog tracks file mutations and supports single-step undo.
// Thread-safe via [sync.Mutex].
type OperationLog struct {
	mu      sync.Mutex
	entries []Operation
}

// NewOperationLog creates a new empty operation log.
func NewOperationLog() *OperationLog {
	return &OperationLog{
		entries: make([]Operation, 0, MaxEntries),
	}
}

// Record adds an operation to the log with FIFO eviction at MaxEntries.
func (l *OperationLog) Record(op Operation) {
	l.mu.Lock()
	defer l.mu.Unlock()

	op.Timestamp = time.Now()

	if len(l.entries) >= MaxEntries {
		// FIFO eviction: drop the oldest entry.
		copy(l.entries, l.entries[1:])
		l.entries[len(l.entries)-1] = op
	} else {
		l.entries = append(l.entries, op)
	}
}

// Undo reverses the most recent operation and removes it from the log.
func (l *OperationLog) Undo() (Operation, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		return Operation{}, ErrEmptyLog
	}

	// Pop the last entry.
	lastIdx := len(l.entries) - 1
	op := l.entries[lastIdx]
	l.entries = l.entries[:lastIdx]

	if err := reverseOperation(op); err != nil {
		return op, fmt.Errorf("undo %s %s: %w", op.Type, op.Path, err)
	}

	return op, nil
}

// Len returns the number of recorded operations.
func (l *OperationLog) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return len(l.entries)
}

// reverseOperation applies the reverse of the given operation to the filesystem.
func reverseOperation(op Operation) error {
	switch op.Type {
	case OpCreate:
		// Reverse of create is delete.
		if err := os.Remove(op.Path); err != nil {
			return fmt.Errorf("remove created file: %w", err)
		}
	case OpModify:
		// Reverse of modify is restore the before-content.
		if err := os.WriteFile(op.Path, op.BeforeContent, 0o600); err != nil {
			return fmt.Errorf("restore modified file: %w", err)
		}
	case OpDelete:
		// Reverse of delete is recreate with before-content.
		dir := filepath.Dir(op.Path)

		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create parent dirs for deleted file: %w", err)
		}

		if err := os.WriteFile(op.Path, op.BeforeContent, 0o600); err != nil {
			return fmt.Errorf("restore deleted file: %w", err)
		}
	}

	return nil
}

// RecordFileChange is a convenience method that detects whether a path
// is a create or modify operation and records it with the appropriate before-content.
func (l *OperationLog) RecordFileChange(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist yet — record as create operation.
			l.Record(Operation{
				Type: OpCreate,
				Path: path,
			})

			return nil
		}

		return fmt.Errorf("read before-content: %w", err)
	}

	// File exists — record as modify operation.
	l.Record(Operation{
		Type:          OpModify,
		Path:          path,
		BeforeContent: content,
	})

	return nil
}
