package task

import (
	"testing"
)

func TestNewReview(t *testing.T) {
	t.Parallel()
	review := NewReview()

	if review == nil {
		t.Fatal("NewReview() returned nil")
	}

	if review.Name() != TaskNameReview {
		t.Errorf("NewReview().Name() = %v, want %v", review.Name(), TaskNameReview)
	}

	// Test that it implements the Task interface.
	var _ Task = review
}

func TestReview_Name(t *testing.T) {
	t.Parallel()
	review := NewReview()

	if review.Name() != TaskNameReview {
		t.Errorf("Review.Name() = %v, want %v", review.Name(), TaskNameReview)
	}
}

func TestReview_SystemPrompt(t *testing.T) {
	t.Parallel()
	review := NewReview()
	result := review.SystemPrompt()

	if result == "" {
		t.Error("Review.SystemPrompt() returned empty string")
	}

	// Should be a reasonable length.
	if len(result) < 50 {
		t.Errorf("Review.SystemPrompt() too short: %d characters", len(result))
	}
}

func TestReview_AllowedTools(t *testing.T) {
	t.Parallel()
	review := NewReview()
	tools := review.AllowedTools()

	if tools == nil {
		t.Error("Review.AllowedTools() returned nil")
	}

	if len(tools) == 0 {
		t.Error("Review.AllowedTools() returned empty slice")
	}

	// Review mode should have read-only tools.
	expectedTools := []string{"read_file", "list_directory", "file_search", "git_context", "get_context"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Review.AllowedTools() length = %d, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range expectedTools {
		if tools[i] != tool {
			t.Errorf("Review.AllowedTools()[%d] = %s, want %s", i, tools[i], tool)
		}
	}
}

func TestReview_MaxTokens(t *testing.T) {
	t.Parallel()
	review := NewReview()
	result := review.MaxTokens()

	if result <= 0 {
		t.Errorf("Review.MaxTokens() = %v, want > 0", result)
	}

	if result != DefaultReviewMaxTokens {
		t.Errorf("Review.MaxTokens() = %v, want %v", result, DefaultReviewMaxTokens)
	}
}

func TestReview_Validate(t *testing.T) {
	t.Parallel()
	review := NewReview()

	err := review.Validate()
	if err != nil {
		t.Errorf("Review.Validate() unexpected error: %v", err)
	}
}

func TestReview_Concurrency(t *testing.T) {
	t.Parallel()
	review := NewReview()

	// Test concurrent access.
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			_ = review.Name()
			_ = review.SystemPrompt()
			_ = review.AllowedTools()
			_ = review.MaxTokens()
			_ = review.Validate()

			done <- true
		}()
	}

	// Wait for all goroutines to complete.
	for range 10 {
		<-done
	}
}

func TestReview_TaskInterface(t *testing.T) {
	t.Parallel()
	review := NewReview()

	// Verify all interface methods work.
	if review.Name() == "" {
		t.Error("Review.Name() returned empty string")
	}

	if review.SystemPrompt() == "" {
		t.Error("Review.SystemPrompt() returned empty string")
	}

	if review.AllowedTools() == nil {
		t.Error("Review.AllowedTools() returned nil")
	}

	if review.MaxTokens() <= 0 {
		t.Error("Review.MaxTokens() returned non-positive value")
	}

	err := review.Validate()
	if err != nil {
		t.Errorf("Review.Validate() returned error: %v", err)
	}
}

func TestReview_Constants(t *testing.T) {
	t.Parallel(
	// Verify constants are reasonable.
	)

	if DefaultReviewMaxTokens <= 0 {
		t.Errorf("DefaultReviewMaxTokens = %d, want > 0", DefaultReviewMaxTokens)
	}

	if DefaultReviewMaxTokens != 12288 {
		t.Errorf("DefaultReviewMaxTokens = %d, want 12288", DefaultReviewMaxTokens)
	}
}

func TestReview_MaxTokensInRange(t *testing.T) {
	t.Parallel()
	review := NewReview()
	maxTokens := review.MaxTokens()

	// Should be within reasonable bounds.
	if maxTokens < 1000 {
		t.Errorf("Review.MaxTokens() = %d, seems too small", maxTokens)
	}

	if maxTokens > 100000 {
		t.Errorf("Review.MaxTokens() = %d, exceeds maximum allowed", maxTokens)
	}
}
