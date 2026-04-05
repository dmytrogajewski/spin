// Package adapter bridges contexteng implementations to harness interfaces.
package adapter

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/contexteng/compactor"
	"github.com/dmytrogajewski/spin/internal/message"
)

// CompactorAdapter adapts compactor.Compactor to the harness.ContextCompactor interface.
// It converts the Stage return value to a bool indicating whether messages were modified.
type CompactorAdapter struct {
	inner *compactor.Compactor
}

// NewCompactorAdapter creates a CompactorAdapter wrapping the given compactor.
// A nil compactor produces a no-op adapter that returns messages unchanged.
func NewCompactorAdapter(c *compactor.Compactor) *CompactorAdapter {
	return &CompactorAdapter{inner: c}
}

// Compact delegates to the inner compactor and converts Stage to bool.
// Returns true when the compaction stage actually modified messages
// (observation mask or fast prune), false for no-op stages (none, warning).
func (a *CompactorAdapter) Compact(
	ctx context.Context, messages []message.Message,
) ([]message.Message, bool, error) {
	if a.inner == nil {
		return messages, false, nil
	}

	result, stage, err := a.inner.Compact(ctx, messages)
	if err != nil {
		return nil, false, err
	}

	modified := stage >= compactor.StageObservationMask

	return result, modified, nil
}
