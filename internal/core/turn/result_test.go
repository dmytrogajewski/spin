package turn

import (
	"testing"
	"time"
)

func TestResult(t *testing.T) {
	result := Result{
		Success:     true,
		FinalState:  StateCompleted,
		Error:       nil,
		Response:    "Task completed successfully",
		Duration:    time.Minute,
		Tokens:      TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		ToolCount:   3,
		ContextSize: 1024,
		Truncated:   false,
	}

	// Test basic fields
	if !result.Success {
		t.Errorf("Result.Success = %v, want true", result.Success)
	}

	if result.FinalState != StateCompleted {
		t.Errorf("Result.FinalState = %v, want %v", result.FinalState, StateCompleted)
	}

	if result.Error != nil {
		t.Errorf("Result.Error = %v, want nil", result.Error)
	}

	if result.Response != "Task completed successfully" {
		t.Errorf("Result.Response = %v, want 'Task completed successfully'", result.Response)
	}

	if result.Duration != time.Minute {
		t.Errorf("Result.Duration = %v, want %v", result.Duration, time.Minute)
	}

	if result.Tokens.TotalTokens != 150 {
		t.Errorf("Result.Tokens.TotalTokens = %d, want 150", result.Tokens.TotalTokens)
	}

	if result.ToolCount != 3 {
		t.Errorf("Result.ToolCount = %d, want 3", result.ToolCount)
	}

	if result.ContextSize != 1024 {
		t.Errorf("Result.ContextSize = %d, want 1024", result.ContextSize)
	}

	if result.Truncated {
		t.Errorf("Result.Truncated = %v, want false", result.Truncated)
	}
}

func TestResult_WithError(t *testing.T) {
	err := &TestError{Message: "test error"}
	result := Result{
		Success:     false,
		FinalState:  StateFailed,
		Error:       err,
		Response:    "",
		Duration:    time.Second * 30,
		Tokens:      TokenUsage{TotalTokens: 50},
		ToolCount:   0,
		ContextSize: 512,
		Truncated:   true,
	}

	// Test error case
	if result.Success {
		t.Errorf("Result.Success = %v, want false", result.Success)
	}

	if result.FinalState != StateFailed {
		t.Errorf("Result.FinalState = %v, want %v", result.FinalState, StateFailed)
	}

	if result.Error != err {
		t.Errorf("Result.Error = %v, want %v", result.Error, err)
	}

	if result.Response != "" {
		t.Errorf("Result.Response = %v, want empty string", result.Response)
	}

	if result.Duration != time.Second*30 {
		t.Errorf("Result.Duration = %v, want %v", result.Duration, time.Second*30)
	}

	if result.Tokens.TotalTokens != 50 {
		t.Errorf("Result.Tokens.TotalTokens = %d, want 50", result.Tokens.TotalTokens)
	}

	if result.ToolCount != 0 {
		t.Errorf("Result.ToolCount = %d, want 0", result.ToolCount)
	}

	if result.ContextSize != 512 {
		t.Errorf("Result.ContextSize = %d, want 512", result.ContextSize)
	}

	if !result.Truncated {
		t.Errorf("Result.Truncated = %v, want true", result.Truncated)
	}
}

// TestError is a simple error type for testing
type TestError struct {
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}
