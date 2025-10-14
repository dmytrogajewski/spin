package status

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/core"
)

func TestAggregator_ProcessEvent(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Test turn start
	event := &core.Event{Type: core.EventTurnStart}
	aggregator.ProcessEvent(event)

	metrics := manager.GetMetrics()
	if metrics.AgentState != "Starting" {
		t.Errorf("Expected agent state 'Starting', got %q", metrics.AgentState)
	}
	if metrics.TurnCount != 1 {
		t.Errorf("Expected turn count 1 (incremented on turn start), got %d", metrics.TurnCount)
	}

	// Test content generation
	event = &core.Event{Type: core.EventContentDelta}
	aggregator.ProcessEvent(event)

	metrics = manager.GetMetrics()
	if metrics.AgentState != "Thinking" {
		t.Errorf("Expected agent state 'Thinking', got %q", metrics.AgentState)
	}

	// Test tool execution
	event = &core.Event{Type: core.EventToolCallStart}
	aggregator.ProcessEvent(event)

	metrics = manager.GetMetrics()
	if metrics.AgentState != "Calling tools" {
		t.Errorf("Expected agent state 'Calling tools', got %q", metrics.AgentState)
	}

	// Test content complete
	event = &core.Event{Type: core.EventContentComplete}
	aggregator.ProcessEvent(event)

	metrics = manager.GetMetrics()
	if metrics.AgentState != "Ready" {
		t.Errorf("Expected agent state 'Ready', got %q", metrics.AgentState)
	}
}

func TestAggregator_ProcessEvent_Disabled(t *testing.T) {
	manager := NewManager()
	manager.Disable() // Disable the manager
	aggregator := NewAggregator(manager)

	// Process an event
	event := &core.Event{Type: core.EventToolCallStart}
	aggregator.ProcessEvent(event)

	// Agent state should not change because manager is disabled
	metrics := manager.GetMetrics()
	if metrics.AgentState != "" {
		t.Errorf("Expected empty agent state (manager disabled), got %q", metrics.AgentState)
	}
}

func TestAggregator_SetMaxTokens(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Add some tokens first
	manager.AddTokens(100, 50)

	// Set max tokens
	aggregator.SetMaxTokens(1000)

	metrics := manager.GetMetrics()
	if metrics.MaxTokens != 1000 {
		t.Errorf("Expected max tokens 1000, got %d", metrics.MaxTokens)
	}
	if metrics.TokenUsage != 15.0 { // 150/1000 * 100
		t.Errorf("Expected token usage 15.0%%, got %.1f%%", metrics.TokenUsage)
	}
}

func TestAggregator_UnknownEvent(t *testing.T) {
	manager := NewManager()
	manager.SetAgentState("InitialState") // Set an initial state
	aggregator := NewAggregator(manager)

	// Process unknown event type (use a high number that doesn't exist)
	event := &core.Event{Type: core.EventType(999)}
	aggregator.ProcessEvent(event)

	// Unknown events should NOT change the state (new behavior)
	metrics := manager.GetMetrics()
	if metrics.AgentState != "InitialState" {
		t.Errorf("Expected agent state to remain 'InitialState' for unknown event, got %q", metrics.AgentState)
	}
}
