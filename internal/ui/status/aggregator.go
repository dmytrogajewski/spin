package status

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// Aggregator processes core events and updates the status manager.
// This is a simple event processor with no rendering responsibilities.
type Aggregator struct {
	manager      *Manager
	startTime    time.Time
	streamStart  time.Time // When current stream started
	streamTokens int64     // Tokens in current stream
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

	// Map events to user-friendly agent states
	switch event.Type {
	case core.EventTurnStart:
		a.manager.SetAgentState("Starting")
		a.manager.IncrementTurn()

	case core.EventContentDelta:
		a.manager.SetAgentState("Thinking")
		// Track streaming for TPS calculation
		if a.streamStart.IsZero() {
			a.streamStart = time.Now()
			a.streamTokens = 0
		}
		// Estimate tokens from characters (rough: 1 token ≈ 4 chars)
		if data, ok := event.Data.(core.ContentDeltaData); ok {
			if data.Content != "" {
				estimatedTokens := int64(len(data.Content) / 4)
				if estimatedTokens < 1 {
					estimatedTokens = 1
				}
				a.streamTokens += estimatedTokens

				// Calculate TPS based on stream duration
				duration := time.Since(a.streamStart)
				if duration > 0 {
					a.manager.CalculateTPS(a.streamTokens, duration)
				}
			}
		}

	case core.EventContentComplete:
		a.manager.SetAgentState("Ready")
		// Reset TPS and streaming state on content complete
		a.streamStart = time.Time{}
		a.streamTokens = 0
		a.manager.CalculateTPS(0, 1) // Reset TPS to 0

	case core.EventToolCallStart:
		// Extract tool name for more specific status
		if data, ok := event.Data.(core.ToolCallStartData); ok {
			a.manager.SetAgentState("Calling: " + data.ToolName)
		} else {
			a.manager.SetAgentState("Calling tools")
		}

	case core.EventToolCallProgress:
		a.manager.SetAgentState("Executing")

	case core.EventToolCallComplete:
		a.manager.SetAgentState("Complete")

	case core.EventTurnComplete:
		a.manager.SetAgentState("Ready")
		// Extract token usage from turn event
		if data, ok := event.Data.(core.TurnEventData); ok {
			if data.TokensUsed > 0 {
				a.manager.AddTokens(int64(data.TokensUsed), 0)
			}
		}

	case core.EventCommandApproval:
		a.manager.SetAgentState("Waiting approval")

	case core.EventCommandApproved:
		a.manager.SetAgentState("Approved")

	case core.EventCommandDenied:
		a.manager.SetAgentState("Denied")

	case core.EventError:
		a.manager.SetAgentState("Error")

	case core.EventWarning:
		a.manager.SetAgentState("Warning")

	default:
		// For unknown events, keep current state
	}
}

// SetMaxTokens sets the maximum token limit from configuration.
func (a *Aggregator) SetMaxTokens(maxTokens int64) {
	a.manager.SetMaxTokens(maxTokens)
}
