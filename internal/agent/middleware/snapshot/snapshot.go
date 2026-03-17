// Package snapshot provides a harness middleware that captures working-tree
// snapshots after each agent execution phase for full-tree undo support.
package snapshot

import (
	"context"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent/harness"
)

// Snapshotter captures working-tree state. Implemented by [undo.Service].
type Snapshotter interface {
	TakeSnapshot() (string, error)
}

// Middleware implements [harness.Middleware] to take snapshots
// after each harness execution phase.
type Middleware struct {
	snapshotter Snapshotter
	logger      *slog.Logger
}

// NewMiddleware creates a new snapshot middleware.
func NewMiddleware(snapshotter Snapshotter, logger *slog.Logger) *Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return &Middleware{
		snapshotter: snapshotter,
		logger:      logger,
	}
}

// BeforeTurn is a no-op for snapshot middleware.
func (m *Middleware) BeforeTurn(_ context.Context, _ *harness.IterationContext) {}

// AfterExecution takes a snapshot of the working tree after execution completes.
func (m *Middleware) AfterExecution(
	ctx context.Context, _ *harness.IterationContext, _ *harness.Response,
) {
	hash, err := m.snapshotter.TakeSnapshot()
	if err != nil {
		m.logger.WarnContext(ctx, "failed to take snapshot after execution", "error", err)

		return
	}

	m.logger.DebugContext(ctx, "snapshot taken after execution", "hash", hash)
}
