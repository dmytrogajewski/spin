package task

import (
	"testing"
)

func TestNewRegular(t *testing.T) {
	regular := NewRegular()

	if regular == nil {
		t.Fatal("NewRegular() returned nil")
	}

	if regular.Name() != "regular" {
		t.Errorf("NewRegular().Name() = %v, want %v", regular.Name(), "regular")
	}

	// Test that it implements the Task interface.
	var _ Task = regular
}

func TestRegular_Name(t *testing.T) {
	regular := NewRegular()

	if regular.Name() != "regular" {
		t.Errorf("Regular.Name() = %v, want %v", regular.Name(), "regular")
	}
}

func TestRegular_SystemPrompt(t *testing.T) {
	regular := NewRegular()
	result := regular.SystemPrompt()

	if len(result) == 0 {
		t.Error("Regular.SystemPrompt() returned empty string")
	}

	// Should be a reasonable length.
	if len(result) < 50 {
		t.Errorf("Regular.SystemPrompt() too short: %d characters", len(result))
	}
}

func TestRegular_AllowedTools(t *testing.T) {
	regular := NewRegular()
	tools := regular.AllowedTools()

	// Regular mode allows all tools (empty slice means no restrictions).
	if tools == nil {
		t.Error("Regular.AllowedTools() returned nil")
	}

	// Empty slice means all tools are allowed.
	if len(tools) != 0 {
		t.Errorf("Regular.AllowedTools() should return empty slice for all tools, got %d tools", len(tools))
	}
}

func TestRegular_MaxTokens(t *testing.T) {
	regular := NewRegular()
	result := regular.MaxTokens()

	if result <= 0 {
		t.Errorf("Regular.MaxTokens() = %v, want > 0", result)
	}

	if result != DefaultMaxTokens {
		t.Errorf("Regular.MaxTokens() = %v, want %v", result, DefaultMaxTokens)
	}
}

func TestRegular_Validate(t *testing.T) {
	regular := NewRegular()

	err := regular.Validate()
	if err != nil {
		t.Errorf("Regular.Validate() unexpected error: %v", err)
	}
}

func TestRegular_Concurrency(_ *testing.T) {
	regular := NewRegular()

	// Test concurrent access.
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			_ = regular.Name()
			_ = regular.SystemPrompt()
			_ = regular.AllowedTools()
			_ = regular.MaxTokens()
			_ = regular.Validate()

			done <- true
		}()
	}

	// Wait for all goroutines to complete.
	for range 10 {
		<-done
	}
}

func TestRegular_TaskInterface(t *testing.T) {
	regular := NewRegular()

	// Verify all interface methods work.
	if regular.Name() == "" {
		t.Error("Regular.Name() returned empty string")
	}

	if regular.SystemPrompt() == "" {
		t.Error("Regular.SystemPrompt() returned empty string")
	}

	if regular.AllowedTools() == nil {
		t.Error("Regular.AllowedTools() returned nil")
	}

	if regular.MaxTokens() <= 0 {
		t.Error("Regular.MaxTokens() returned non-positive value")
	}

	err := regular.Validate()
	if err != nil {
		t.Errorf("Regular.Validate() returned error: %v", err)
	}
}

func TestRegular_Constants(t *testing.T) {
	if DefaultMaxTokens != 16384 {
		t.Errorf("DefaultMaxTokens = %v, want 16384", DefaultMaxTokens)
	}

	if MinPromptLength != 50 {
		t.Errorf("MinPromptLength = %v, want 50", MinPromptLength)
	}
}

func TestRegular_MaxTokensInRange(t *testing.T) {
	regular := NewRegular()
	maxTokens := regular.MaxTokens()

	// Should be within reasonable bounds.
	if maxTokens < 1000 {
		t.Errorf("Regular.MaxTokens() = %d, seems too small", maxTokens)
	}

	if maxTokens > 100000 {
		t.Errorf("Regular.MaxTokens() = %d, exceeds maximum allowed", maxTokens)
	}
}
