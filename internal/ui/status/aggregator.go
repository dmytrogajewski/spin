// Package status provides UI status aggregation.
package status

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// Aggregator processes core events and updates the status manager.
// This is a simple event processor with no rendering responsibilities.
type Aggregator struct {
	manager      *Manager
	startTime    time.Time
	streamStart  time.Time // When current stream started.
	streamTokens int64     // Tokens in current stream.
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

	a.applyEvent(event)
}

func (a *Aggregator) applyEvent(event *events.Event) {
	switch event.Type {
	case events.EventTurnStart:
		a.handleTurnStart()
	case events.EventThinkingDelta:
		a.manager.SetAgentState("Thinking")
	case events.EventContentDelta:
		a.handleContentDelta(event)
	case events.EventContentComplete:
		a.handleContentComplete()
	case events.EventToolCallStart:
		a.handleToolCallStart(event)
	case events.EventToolCallProgress:
		a.manager.SetAgentState("Executing")
	case events.EventToolCallComplete:
		a.handleToolCallComplete(event)
	case events.EventTurnComplete:
		a.manager.SetAgentState("Idle")
	case events.EventTurnFailed, events.EventError:
		a.manager.SetAgentState("Error")
	case events.EventCommandApproval:
		a.manager.SetAgentState("Waiting approval")
	case events.EventCommandApproved:
		a.manager.SetAgentState("Approved")
	case events.EventCommandDenied:
		a.manager.SetAgentState("Denied")
	case events.EventWarning:
		a.manager.SetAgentState("Warning")
	default:
		a.applyTaskCountEvent(event.Type)
	}
}

func (a *Aggregator) handleTurnStart() {
	a.manager.SetAgentState("Starting")
	a.manager.IncrementTurn()
}

func (a *Aggregator) handleContentDelta(event *events.Event) {
	a.manager.SetAgentState("Thinking")

	if a.streamStart.IsZero() {
		a.streamStart = time.Now()
		a.streamTokens = 0
	}

	data, ok := event.Data.(events.ContentDeltaData)
	if !ok || data.Content == "" {
		return
	}

	estimatedTokens := max(int64(len(data.Content)/stringsx.CharsPerToken), 1)
	a.streamTokens += estimatedTokens

	duration := time.Since(a.streamStart)
	if duration > 0 {
		a.manager.CalculateTPS(a.streamTokens, duration)
	}
}

func (a *Aggregator) handleContentComplete() {
	// Don't change agent state here — EventTurnComplete will set "Idle".
	// Setting "Ready" here causes a redundant status bar render between
	// ContentComplete and TurnComplete.
	a.streamStart = time.Time{}
	a.streamTokens = 0
	a.manager.CalculateTPS(0, 1)
}

func (a *Aggregator) handleToolCallStart(event *events.Event) {
	if data, ok := event.Data.(events.ToolCallStartData); ok {
		a.manager.SetAgentState("Calling: " + data.ToolName)
	} else {
		a.manager.SetAgentState("Calling tools")
	}
}

func (a *Aggregator) handleToolCallComplete(event *events.Event) {
	a.manager.SetAgentState("Complete")

	data, ok := event.Data.(events.ToolCallCompleteData)
	if !ok || data.Metadata == nil {
		return
	}

	in, inOK := compactMetaInt(data.Metadata[compact.MetaBytesIn])
	out, outOK := compactMetaInt(data.Metadata[compact.MetaBytesOut])

	if inOK && outOK {
		a.manager.SetCompactSavings(in, out)
	}
}

func compactMetaInt(v any) (int, bool) {
	n, ok := v.(int)

	return n, ok
}

// SetMaxTokens sets the maximum token limit from configuration.
func (a *Aggregator) SetMaxTokens(maxTokens int64) {
	a.manager.SetMaxTokens(maxTokens)
}

func (a *Aggregator) applyTaskCountEvent(eventType events.EventType) {
	switch eventType {
	case events.EventBackgroundTaskStarted:
		a.adjustTaskCounts(1, 1)
	case events.EventBackgroundTaskStopped:
		a.adjustTaskCounts(-1, 0)
	default:
		// Keep current state for unknown events.
	}
}

func (a *Aggregator) adjustTaskCounts(activeDelta, totalDelta int) {
	cur := a.manager.GetMetrics()
	active := max(0, cur.TasksActive+activeDelta)
	total := max(0, cur.TasksTotal+totalDelta)
	a.manager.SetTaskCounts(active, total)
}
