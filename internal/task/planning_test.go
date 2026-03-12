package task

import (
	"strings"
	"testing"
)

func TestNewPlanning(t *testing.T) {
	planning := NewPlanning()

	if planning == nil {
		t.Fatal("NewPlanning() returned nil")
	}

	if planning.Name() != "planning" {
		t.Errorf("NewPlanning().Name() = %v, want 'planning'", planning.Name())
	}

	// Test that it implements the Task interface.
	var _ Task = planning
}

func TestPlanning_Name(t *testing.T) {
	planning := NewPlanning()

	name := planning.Name()

	if name != "planning" {
		t.Errorf("Planning.Name() = %s, want 'planning'", name)
	}
}

func TestPlanning_SystemPrompt(t *testing.T) {
	planning := NewPlanning()
	prompt := planning.SystemPrompt()

	if len(prompt) == 0 {
		t.Error("Planning.SystemPrompt() returned empty string")
	}

	// Check that prompt contains key planning elements.
	expectedElements := []string{
		"planning",
		"step",
	}

	for _, element := range expectedElements {
		if !strings.Contains(strings.ToLower(prompt), element) {
			t.Errorf("Planning.SystemPrompt() should contain '%s'", element)
		}
	}

	// Should be a reasonable length.
	if len(prompt) < 50 {
		t.Errorf("Planning.SystemPrompt() too short: %d characters", len(prompt))
	}
}

func TestPlanning_AllowedTools(t *testing.T) {
	planning := NewPlanning()
	tools := planning.AllowedTools()

	expectedTools := []string{"list_directory", "file_search", "git_context", "get_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Planning.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Planning.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestPlanning_MaxTokens(t *testing.T) {
	planning := NewPlanning()
	maxTokens := planning.MaxTokens()

	if maxTokens != PlanningMaxTokens {
		t.Errorf("Planning.MaxTokens() = %d, want %d", maxTokens, PlanningMaxTokens)
	}

	if maxTokens <= 0 {
		t.Errorf("Planning.MaxTokens() = %d, want > 0", maxTokens)
	}
}

func TestPlanning_Validate(t *testing.T) {
	planning := NewPlanning()

	err := planning.Validate()
	if err != nil {
		t.Errorf("Planning.Validate() unexpected error: %v", err)
	}
}

func TestPlanning_DefaultPrompt(t *testing.T) {
	planning := NewPlanning()
	prompt := planning.SystemPrompt()

	// Check that default prompt is reasonable.
	if len(prompt) < 50 {
		t.Errorf("Planning default prompt too short: %d characters", len(prompt))
	}
}

func TestPlanning_DefaultTools(t *testing.T) {
	planning := NewPlanning()
	tools := planning.AllowedTools()

	expectedTools := []string{"list_directory", "file_search", "git_context", "get_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Planning.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Planning.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestPlanning_DefaultMaxTokens(t *testing.T) {
	planning := NewPlanning()
	maxTokens := planning.MaxTokens()

	if maxTokens != PlanningMaxTokens {
		t.Errorf("Planning.MaxTokens() = %d, want %d", maxTokens, PlanningMaxTokens)
	}
}

func TestPlanning_Concurrency(_ *testing.T) {
	planning := NewPlanning()

	// Test concurrent access to methods.
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			// These methods should be safe for concurrent access.
			planning.Name()
			planning.SystemPrompt()
			planning.AllowedTools()
			planning.MaxTokens()
			_ = planning.Validate()

			done <- true
		}()
	}

	// Wait for all goroutines to complete.
	for range 10 {
		<-done
	}
}

func TestPlanning_TaskInterface(t *testing.T) {
	planning := NewPlanning()

	// Verify all interface methods work.
	if planning.Name() == "" {
		t.Error("Planning.Name() returned empty string")
	}

	if planning.SystemPrompt() == "" {
		t.Error("Planning.SystemPrompt() returned empty string")
	}

	if planning.AllowedTools() == nil {
		t.Error("Planning.AllowedTools() returned nil")
	}

	if planning.MaxTokens() <= 0 {
		t.Error("Planning.MaxTokens() returned non-positive value")
	}

	err := planning.Validate()
	if err != nil {
		t.Errorf("Planning.Validate() returned error: %v", err)
	}
}

func TestPlanning_Constants(t *testing.T) {
	// Test that constants are properly defined.
	if PlanningMaxTokens <= 0 {
		t.Errorf("PlanningMaxTokens = %d, want > 0", PlanningMaxTokens)
	}

	if PlanningMaxTokens != 4096 {
		t.Errorf("PlanningMaxTokens = %d, want 4096", PlanningMaxTokens)
	}

	if PlanningMinSteps <= 0 {
		t.Errorf("PlanningMinSteps = %d, want > 0", PlanningMinSteps)
	}

	if PlanningMaxSteps <= 0 {
		t.Errorf("PlanningMaxSteps = %d, want > 0", PlanningMaxSteps)
	}

	if PlanningMinEstimate <= 0 {
		t.Errorf("PlanningMinEstimate = %d, want > 0", PlanningMinEstimate)
	}

	// Test relationships.
	if PlanningMinSteps > PlanningMaxSteps {
		t.Errorf("PlanningMinSteps (%d) > PlanningMaxSteps (%d)", PlanningMinSteps, PlanningMaxSteps)
	}
}
