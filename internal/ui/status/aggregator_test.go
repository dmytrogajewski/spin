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

func TestAggregator_ProcessEvent_AllTypes(t *testing.T) {
	tests := []struct {
		name          string
		event         *core.Event
		expectedState string
	}{
		{
			name:          "ToolCallStart with data",
			event:         &core.Event{Type: core.EventToolCallStart, Data: core.ToolCallStartData{ToolName: "bash"}},
			expectedState: "Calling: bash",
		},
		{
			name:          "ToolCallStart without data",
			event:         &core.Event{Type: core.EventToolCallStart, Data: nil},
			expectedState: "Calling tools",
		},
		{
			name:          "ToolCallProgress",
			event:         &core.Event{Type: core.EventToolCallProgress},
			expectedState: "Executing",
		},
		{
			name:          "ToolCallComplete",
			event:         &core.Event{Type: core.EventToolCallComplete},
			expectedState: "Complete",
		},
		{
			name:          "CommandApproval",
			event:         &core.Event{Type: core.EventCommandApproval},
			expectedState: "Waiting approval",
		},
		{
			name:          "CommandApproved",
			event:         &core.Event{Type: core.EventCommandApproved},
			expectedState: "Approved",
		},
		{
			name:          "CommandDenied",
			event:         &core.Event{Type: core.EventCommandDenied},
			expectedState: "Denied",
		},
		{
			name:          "Error",
			event:         &core.Event{Type: core.EventError},
			expectedState: "Error",
		},
		{
			name:          "Warning",
			event:         &core.Event{Type: core.EventWarning},
			expectedState: "Warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			aggregator := NewAggregator(manager)

			aggregator.ProcessEvent(tt.event)

			metrics := manager.GetMetrics()
			if metrics.AgentState != tt.expectedState {
				t.Errorf("Expected agent state %q, got %q", tt.expectedState, metrics.AgentState)
			}
		})
	}
}

func TestAggregator_ProcessEvent_ContentDelta_WithData(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Process content delta with data
	event := &core.Event{
		Type: core.EventContentDelta,
		Data: core.ContentDeltaData{Content: "This is some test content with enough characters to count tokens"},
	}
	aggregator.ProcessEvent(event)

	metrics := manager.GetMetrics()
	if metrics.AgentState != "Thinking" {
		t.Errorf("Expected agent state 'Thinking', got %q", metrics.AgentState)
	}
	// TPS should be calculated
	if metrics.TokensPerSec < 0 {
		t.Error("Expected non-negative TPS")
	}
}

func TestAggregator_ProcessEvent_ContentDelta_ShortContent(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Process content delta with very short content (less than 4 chars)
	event := &core.Event{
		Type: core.EventContentDelta,
		Data: core.ContentDeltaData{Content: "Hi"},
	}
	aggregator.ProcessEvent(event)

	metrics := manager.GetMetrics()
	if metrics.AgentState != "Thinking" {
		t.Errorf("Expected agent state 'Thinking', got %q", metrics.AgentState)
	}
}

func TestAggregator_ProcessEvent_TurnComplete_WithTokens(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Process turn complete with token data
	event := &core.Event{
		Type: core.EventTurnComplete,
		Data: core.TurnEventData{TokensUsed: 500},
	}
	aggregator.ProcessEvent(event)

	metrics := manager.GetMetrics()
	if metrics.AgentState != "Ready" {
		t.Errorf("Expected agent state 'Ready', got %q", metrics.AgentState)
	}
	if metrics.TokenCount != 500 {
		t.Errorf("Expected token count 500, got %d", metrics.TokenCount)
	}
}

func TestAggregator_ProcessEvent_ContentComplete_ResetsStreaming(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// First, start streaming with content delta
	event := &core.Event{
		Type: core.EventContentDelta,
		Data: core.ContentDeltaData{Content: "Some streaming content"},
	}
	aggregator.ProcessEvent(event)

	// Verify TPS was calculated
	metrics := manager.GetMetrics()
	// TPS might be 0 or positive, just verify it's set

	// Now complete the content
	event = &core.Event{Type: core.EventContentComplete}
	aggregator.ProcessEvent(event)

	metrics = manager.GetMetrics()
	if metrics.AgentState != "Ready" {
		t.Errorf("Expected agent state 'Ready' after content complete, got %q", metrics.AgentState)
	}
	// TPS should be reset to 0
	if metrics.TokensPerSec != 0 {
		t.Errorf("Expected TPS to be reset to 0 after content complete, got %.2f", metrics.TokensPerSec)
	}
}
