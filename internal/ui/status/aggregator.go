package status

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
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
func (a *Aggregator) ProcessEvent(event *events.Event) {
	if !a.manager.IsEnabled() {
		return
	}

	// Map events to user-friendly agent states
	switch event.Type {
	case events.EventTurnStart:
		a.manager.SetAgentState("Starting")
		a.manager.IncrementTurn()

	case events.EventContentDelta:
		a.manager.SetAgentState("Thinking")
		// Track streaming for TPS calculation
		if a.streamStart.IsZero() {
			a.streamStart = time.Now()
			a.streamTokens = 0
		}
		// Estimate tokens from characters (rough: 1 token ≈ 4 chars)
		if data, ok := event.Data.(events.ContentDeltaData); ok {
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

	case events.EventContentComplete:
		a.manager.SetAgentState("Ready")
		// Reset TPS and streaming state on content complete
		a.streamStart = time.Time{}
		a.streamTokens = 0
		a.manager.CalculateTPS(0, 1) // Reset TPS to 0

	case events.EventToolCallStart:
		// Extract tool name for more specific status
		if data, ok := event.Data.(events.ToolCallStartData); ok {
			a.manager.SetAgentState("Calling: " + data.ToolName)
		} else {
			a.manager.SetAgentState("Calling tools")
		}

	case events.EventToolCallProgress:
		a.manager.SetAgentState("Executing")

	case events.EventToolCallComplete:
		a.manager.SetAgentState("Complete")

	case events.EventTurnComplete:
		a.manager.SetAgentState("Idle")
		// Extract token usage from turn event
		if data, ok := event.Data.(events.TurnEventData); ok {
			if data.TokensUsed > 0 {
				a.manager.AddTokens(int64(data.TokensUsed), 0)
			}
		}

	case events.EventTurnFailed:
		a.manager.SetAgentState("Error")

	case events.EventCommandApproval:
		a.manager.SetAgentState("Waiting approval")

	case events.EventCommandApproved:
		a.manager.SetAgentState("Approved")

	case events.EventCommandDenied:
		a.manager.SetAgentState("Denied")

	case events.EventError:
		a.manager.SetAgentState("Error")

	case events.EventWarning:
		a.manager.SetAgentState("Warning")

	default:
		// For unknown events, keep current state
	}
}

// SetMaxTokens sets the maximum token limit from configuration.
func (a *Aggregator) SetMaxTokens(maxTokens int64) {
	a.manager.SetMaxTokens(maxTokens)
}
