package status

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/core"
)

func TestAggregator_ProcessEvent(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Test content generation
	event := &core.Event{Type: core.EventContentDelta}
	aggregator.ProcessEvent(event)

	status := manager.GetStatus()
	if status.Text != "Generating content..." {
		t.Errorf("Expected status 'Generating content...', got %q", status.Text)
	}

	// Test tool execution
	event = &core.Event{Type: core.EventToolCallStart}
	aggregator.ProcessEvent(event)

	status = manager.GetStatus()
	if status.Text != "Executing tool..." {
		t.Errorf("Expected status 'Executing tool...', got %q", status.Text)
	}

	// Test tool complete (increments turn)
	event = &core.Event{Type: core.EventToolCallComplete}
	aggregator.ProcessEvent(event)

	status = manager.GetStatus()
	if status.Text != "Tool complete" {
		t.Errorf("Expected status 'Tool complete', got %q", status.Text)
	}

	metrics := manager.GetMetrics()
	if metrics.TurnCount != 1 {
		t.Errorf("Expected turn count 1, got %d", metrics.TurnCount)
	}

	// Test content complete
	event = &core.Event{Type: core.EventContentComplete}
	aggregator.ProcessEvent(event)

	status = manager.GetStatus()
	if status.Text != "Content complete" {
		t.Errorf("Expected status 'Content complete', got %q", status.Text)
	}
}

func TestAggregator_ProcessEvent_Disabled(t *testing.T) {
	manager := NewManager()
	manager.Disable() // Disable the manager
	aggregator := NewAggregator(manager)

	// Process an event
	event := &core.Event{Type: core.EventToolCallStart}
	aggregator.ProcessEvent(event)

	// Status should not change because manager is disabled
	status := manager.GetStatus()
	if status.Text != "" {
		t.Errorf("Expected empty status (manager disabled), got %q", status.Text)
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
	aggregator := NewAggregator(manager)

	// Process unknown event type (use a high number that doesn't exist)
	event := &core.Event{Type: core.EventType(999)}
	aggregator.ProcessEvent(event)

	// Should set default status for unknown events
	status := manager.GetStatus()
	if status.Text != "Processing..." {
		t.Errorf("Expected status 'Processing...' (unknown event), got %q", status.Text)
	}
}
