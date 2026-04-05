package undo

import (
	"fmt"
	"sync"
)

// Service combines OperationLog (fine-grained file undo) and SnapshotManager
// (full working-tree undo) into a unified undo interface.
type Service struct {
	mu       sync.Mutex
	log      *OperationLog
	snapshot *SnapshotManager
}

// NewService creates a new undo service.
func NewService(log *OperationLog, snapshot *SnapshotManager) *Service {
	return &Service{
		log:      log,
		snapshot: snapshot,
	}
}

// UndoLast reverses the most recent file operation via the operation log.
func (s *Service) UndoLast() (Operation, error) {
	return s.log.Undo()
}

// TakeSnapshot captures the current working tree state.
// Returns the tree hash or an error if snapshots are unavailable.
func (s *Service) TakeSnapshot() (string, error) {
	if s.snapshot == nil {
		return "", ErrShadowRepoNotInitialized
	}

	return s.snapshot.Snapshot()
}

// UndoToStep restores the working tree to the state at the given step (zero-based).
func (s *Service) UndoToStep(step int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.snapshot == nil {
		return ErrShadowRepoNotInitialized
	}

	hash, err := s.snapshot.GetSnapshot(step)
	if err != nil {
		return fmt.Errorf("get snapshot at step %d: %w", step, err)
	}

	if restoreErr := s.snapshot.Restore(hash); restoreErr != nil {
		return fmt.Errorf("restore to step %d: %w", step, restoreErr)
	}

	return nil
}

// SnapshotCount returns the number of snapshots taken.
func (s *Service) SnapshotCount() int {
	if s.snapshot == nil {
		return 0
	}

	return s.snapshot.SnapshotCount()
}

// OperationLog returns the underlying operation log.
func (s *Service) OperationLog() *OperationLog {
	return s.log
}
