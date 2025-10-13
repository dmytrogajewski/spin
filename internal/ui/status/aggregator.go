package status

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// Aggregator processes core events and updates the status manager.
// This is a simple event processor with no rendering responsibilities.
type Aggregator struct {
	manager   *Manager
	startTime time.Time
}

// NewAggregator creates a new status aggregator.
func NewAggregator(manager *Manager) *Aggregator {
	return &Aggregator{
		manager:   manager,
		startTime: time.Now(),
	}
}

// ProcessEvent processes a core event and updates status metrics.
func (a *Aggregator) ProcessEvent(event *core.Event) {
	if !a.manager.IsEnabled() {
		return
	}

	// Simple status updates based on event type
	switch event.Type {
	case core.EventContentDelta:
		a.manager.SetStatus("Generating content...")

	case core.EventContentComplete:
		a.manager.SetStatus("Content complete")

	case core.EventToolCallStart:
		a.manager.SetStatus("Executing tool...")

	case core.EventToolCallComplete:
		a.manager.SetStatus("Tool complete")
		a.manager.IncrementTurn()

	case core.EventTurnStart:
		a.manager.SetStatus("Starting turn...")

	case core.EventTurnComplete:
		a.manager.SetStatus("Turn complete")

	default:
		// For now, just update status text for any event
		a.manager.SetStatus("Processing...")
	}
}

// SetMaxTokens sets the maximum token limit from configuration.
func (a *Aggregator) SetMaxTokens(maxTokens int64) {
	a.manager.SetMaxTokens(maxTokens)
}
