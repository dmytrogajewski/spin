package task

import (
	"strings"
	"testing"
)

func TestNewCompact(t *testing.T) {
	t.Parallel()
	compact := NewCompact()

	if compact == nil {
		t.Fatal("NewCompact() returned nil")
	}

	// Test that it implements the Task interface.
	var _ Task = compact
}

func TestCompact_Name(t *testing.T) {
	t.Parallel()
	compact := NewCompact()

	name := compact.Name()

	if name != TaskNameCompact {
		t.Errorf("Compact.Name() = %s, want 'compact'", name)
	}
}

func TestCompact_SystemPrompt(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	prompt := compact.SystemPrompt()

	if prompt == "" {
		t.Error("Compact.SystemPrompt() returned empty string")
	}

	// Check that prompt contains key elements.
	if !strings.Contains(prompt, "fast") || !strings.Contains(prompt, "efficient") {
		t.Errorf("Compact.SystemPrompt() should contain key elements about efficiency")
	}
}

func TestCompact_AllowedTools(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	tools := compact.AllowedTools()

	expectedTools := []string{"read_file", "list_directory", "file_search", "get_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Compact.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Compact.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestCompact_MaxTokens(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	maxTokens := compact.MaxTokens()

	if maxTokens != DefaultCompactMaxTokens {
		t.Errorf("Compact.MaxTokens() = %d, want %d", maxTokens, DefaultCompactMaxTokens)
	}

	if maxTokens <= 0 {
		t.Errorf("Compact.MaxTokens() = %d, want > 0", maxTokens)
	}
}

func TestCompact_Validate(t *testing.T) {
	t.Parallel()
	compact := NewCompact()

	err := compact.Validate()
	if err != nil {
		t.Errorf("Compact.Validate() unexpected error: %v", err)
	}
}

func TestCompact_DefaultPrompt(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	prompt := compact.SystemPrompt()

	// Check that default prompt is reasonable.
	if len(prompt) < 50 {
		t.Errorf("Compact default prompt too short: %d characters", len(prompt))
	}
}

func TestCompact_DefaultTools(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	tools := compact.AllowedTools()

	expectedTools := []string{"read_file", "list_directory", "file_search", "get_context"}

	if len(tools) != len(expectedTools) {
		t.Errorf("Compact.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("Compact.AllowedTools() [%d] = %s, want %s", i, tool, expectedTools[i])
		}
	}
}

func TestCompact_DefaultMaxTokens(t *testing.T) {
	t.Parallel()
	compact := NewCompact()
	maxTokens := compact.MaxTokens()

	if maxTokens != DefaultCompactMaxTokens {
		t.Errorf("Compact.MaxTokens() = %d, want %d", maxTokens, DefaultCompactMaxTokens)
	}
}

func TestCompact_Concurrency(t *testing.T) {
	t.Parallel()
	compact := NewCompact()

	// Test concurrent access to methods.
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			// These methods should be safe for concurrent access.
			compact.Name()
			compact.SystemPrompt()
			compact.AllowedTools()
			compact.MaxTokens()
			_ = compact.Validate()

			done <- true
		}()
	}

	// Wait for all goroutines to complete.
	for range 10 {
		<-done
	}
}

func TestCompact_TaskInterface(t *testing.T) {
	t.Parallel()
	compact := NewCompact()

	// Verify all interface methods work.
	if compact.Name() == "" {
		t.Error("Compact.Name() returned empty string")
	}

	if compact.SystemPrompt() == "" {
		t.Error("Compact.SystemPrompt() returned empty string")
	}

	if compact.AllowedTools() == nil {
		t.Error("Compact.AllowedTools() returned nil")
	}

	if compact.MaxTokens() <= 0 {
		t.Error("Compact.MaxTokens() returned non-positive value")
	}

	err := compact.Validate()
	if err != nil {
		t.Errorf("Compact.Validate() returned error: %v", err)
	}
}

func TestCompact_Constants(t *testing.T) {
	t.Parallel(
	// Verify constants are reasonable.
	)

	if DefaultCompactMaxTokens <= 0 {
		t.Errorf("DefaultCompactMaxTokens = %d, want > 0", DefaultCompactMaxTokens)
	}

	if DefaultCompactMaxTokens != 4096 {
		t.Errorf("DefaultCompactMaxTokens = %d, want 4096", DefaultCompactMaxTokens)
	}
}
