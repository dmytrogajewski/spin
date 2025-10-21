package agent

import (
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// selectIntervention chooses the appropriate intervention based on cycle type and turn count
func (a *Agent) selectIntervention(cycleType cycle.CycleType, turnCount int) cycle.Intervention {
	// Escalation ladder based on turn count
	switch {
	case turnCount < 10:
		// Early cycles: Use soft intervention (reflection)
		return &cycle.ReflectionIntervention{}

	case turnCount < 30:
		// Mid-stage cycles: Use medium intervention (context summarization)
		// For now, use reflection as fallback since summarization needs compressor integration
		return &cycle.ReflectionIntervention{}

	default:
		// Late-stage/persistent cycles: Escalate to user
		return &cycle.EscalateIntervention{
			Emitter: &eventEmitterAdapter{emitter: a.emitter},
		}
	}
}

// eventEmitterAdapter adapts events.EventEmitter to cycle.EventEmitter interface
type eventEmitterAdapter struct {
	emitter *events.EventEmitter
}

func (a *eventEmitterAdapter) Emit(event cycle.Event) {
	// Convert cycle.Event to events.Event
	// Map event type based on string value
	var eventType events.EventType
	switch event.GetType() {
	case "turn_paused":
		eventType = events.EventTurnPaused
	default:
		eventType = events.EventWarning // fallback
	}

	coreEvent := events.Event{
		Type:      eventType,
		Timestamp: event.GetTimestamp(),
		Data:      event.GetData(),
	}
	a.emitter.Emit(coreEvent)
}

// extractToolNames extracts tool names from LLM tool calls for cycle detection
func extractToolNames(toolCalls []llm.ToolCall) []string {
	names := make([]string, len(toolCalls))
	for i, tc := range toolCalls {
		names[i] = tc.Function.Name
	}
	return names
}
