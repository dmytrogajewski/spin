package status

import (
	"bytes"
	"testing"

	"github.com/dmytrogajewski/spin/internal/core"
)

// TestIntegration_StatusDisplayWithEvents tests the complete flow:
// Event → Aggregator → Manager → FormatCompact
func TestIntegration_StatusDisplayWithEvents(t *testing.T) {
	manager := NewManager()
	aggregator := NewAggregator(manager)

	// Simulate content generation event
	event := &core.Event{
		Type: core.EventContentDelta,
		Data: core.ContentDeltaData{
			Content: "Hello",
		},
	}
	aggregator.ProcessEvent(event)

	// Check status was updated
	status := manager.FormatCompact()
	if status != "Generating content..." {
		t.Errorf("Expected 'Generating content...', got '%s'", status)
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
	status := manager.FormatCompact()

	// Check it contains expected parts
	if status == "" {
		t.Error("Expected non-empty status")
	}

	// Should contain provider
	if !contains(status, "ollama") {
		t.Errorf("Expected status to contain 'ollama', got: %s", status)
	}

	// Should contain turn count
	if !contains(status, "T:1") {
		t.Errorf("Expected status to contain 'T:1', got: %s", status)
	}

	// Should contain token count
	if !contains(status, "Tok:300") {
		t.Errorf("Expected status to contain 'Tok:300', got: %s", status)
	}

	// Should contain TPS
	if !contains(status, "TPS:") {
		t.Errorf("Expected status to contain 'TPS:', got: %s", status)
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
	status := manager.FormatCompact()
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
	status := manager.FormatCompact()
	if status != "Waiting for user input..." {
		t.Errorf("Expected 'Waiting for user input...', got: %s", status)
	}
}

// TestIntegration_EmptyStatus tests that empty manager returns empty string
func TestIntegration_EmptyStatus(t *testing.T) {
	manager := NewManager()

	// No data set, should return empty
	status := manager.FormatCompact()
	if status != "" {
		t.Errorf("Expected empty status with no data, got: %s", status)
	}
}

// Helper function
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
