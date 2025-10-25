package status

import (
	"bytes"
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
)

// TestIntegration_StatusDisplayWithEvents tests the complete flow:
// Event → Aggregator → Manager → FormatCompact
func TestIntegration_StatusDisplayWithEvents(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Simulate content generation event
	event := &events.Event{
		Type: events.EventContentDelta,
		Data: events.ContentDeltaData{
			Content: "Hello",
		},
	}
	aggregator.ProcessEvent(event)

	// Check status was updated
	status := manager.FormatCompact(80)
	// New behavior: AgentState set to "Thinking"
	if !contains(status, "Thinking") {
		t.Errorf("Expected 'Thinking', got '%s'", status)
	}
}

// TestIntegration_StatusFormatting tests that the formatted status is compact and useful
func TestIntegration_StatusFormatting(t *testing.T) {
	manager := NewManager()

	// Set some realistic data
	manager.SetProvider("ollama", "llama3.1")
	manager.IncrementTurn()
	manager.AddTokens(100, 200)
	manager.SetResponseTime(1000000000, 300) // 1 second, 300 tokens

	// Get formatted status
	status := manager.FormatCompact(80)

	// Check it contains expected parts
	if status == "" {
		t.Error("Expected non-empty status")
	}

	// New format should contain activity indicator
	if !contains(status, "[●]") && !contains(status, "[○]") {
		t.Errorf("Expected status to contain activity indicator, got: %s", status)
	}

	// Should contain context percentage (if we set maxTokens)
	// (Note: we didn't set maxTokens in this test, so percentage might be 0%)

	// Should contain agent state (default is "Ready")
	if !contains(status, "Ready") {
		t.Errorf("Expected status to contain 'Ready', got: %s", status)
	}

	t.Logf("Formatted status: %s", status)
}

// TestIntegration_StatusDisabled tests that disabled manager returns empty string
func TestIntegration_StatusDisabled(t *testing.T) {
	manager := NewManager()
	manager.SetProvider("ollama", "llama3.1")
	manager.IncrementTurn()

	// Disable manager
	manager.Disable()

	// Should return empty status
	status := manager.FormatCompact(80)
	if status != "" {
		t.Errorf("Expected empty status when disabled, got: %s", status)
	}
}

// TestIntegration_StatusPriority tests that explicit status text takes priority over metrics
func TestIntegration_StatusPriority(t *testing.T) {
	manager := NewManager()
	manager.SetProvider("ollama", "llama3.1")
	manager.SetStatus("Waiting for user input...")

	// Should return explicit status text, not metrics
	status := manager.FormatCompact(80)
	if status != "Waiting for user input..." {
		t.Errorf("Expected 'Waiting for user input...', got: %s", status)
	}
}

// TestIntegration_EmptyStatus tests that empty manager returns empty string
func TestIntegration_EmptyStatus(t *testing.T) {
	manager := NewManager()

	// No data set, should show default "[○] Ready"
	status := manager.FormatCompact(80)
	if !contains(status, "Ready") {
		t.Errorf("Expected default 'Ready' status, got: %s", status)
	}
}

// Helper function
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
