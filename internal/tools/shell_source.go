package tools

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

// ShellAdapter adapts TaskManager to the unified-view shell surface.
type ShellAdapter struct {
	manager TaskManager
}

// AsShellSource adapts TaskManager to the unified-view shell surface.
func AsShellSource(manager TaskManager) *ShellAdapter {
	if manager == nil {
		return nil
	}

	return &ShellAdapter{manager: manager}
}

// List returns shell snapshots for the unified /tasks view.
func (s *ShellAdapter) List(ctx context.Context) []tasks.ShellSnapshot {
	if s == nil {
		return nil
	}

	snaps := s.manager.List(ctx)
	out := make([]tasks.ShellSnapshot, 0, len(snaps))

	for _, snap := range snaps {
		out = append(out, tasks.ShellSnapshot{
			ID:      snap.ID,
			Command: snap.Command,
			State:   snap.Status.String(),
		})
	}

	return out
}

// Kill terminates a shell task by raw id (SIGTERM then SIGKILL).
func (s *ShellAdapter) Kill(ctx context.Context, id string) error {
	if s == nil {
		return nil
	}

	return s.manager.Kill(ctx, id)
}
