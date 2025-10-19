package task

import (
	"testing"
)

func TestNewReview(t *testing.T) {
	review := NewReview()

	if review == nil {
		t.Fatal("NewReview() returned nil")
	}

	if review.Name() != "review" {
		t.Errorf("NewReview().Name() = %v, want %v", review.Name(), "review")
	}
}

func TestReview_Name(t *testing.T) {
	review := NewReview()

	if review.Name() != "review" {
		t.Errorf("Review.Name() = %v, want %v", review.Name(), "review")
	}
}

func TestReview_SystemPrompt(t *testing.T) {
	review := NewReview()
	result := review.SystemPrompt()

	if len(result) == 0 {
		t.Error("Review.SystemPrompt() returned empty string")
	}
}

func TestReview_AllowedTools(t *testing.T) {
	review := NewReview()
	tools := review.AllowedTools()

	if tools == nil {
		t.Error("Review.AllowedTools() returned nil")
	}

	if len(tools) == 0 {
		t.Error("Review.AllowedTools() returned empty slice")
	}
}

func TestReview_MaxTokens(t *testing.T) {
	review := NewReview()
	result := review.MaxTokens()

	if result <= 0 {
		t.Errorf("Review.MaxTokens() = %v, want > 0", result)
	}
}

func TestReview_Validate(t *testing.T) {
	review := NewReview()
	err := review.Validate()

	if err != nil {
		t.Errorf("Review.Validate() unexpected error: %v", err)
	}
}

func TestReview_Validate_WithConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *ReviewConfig
		wantError bool
	}{
		{
			name:      "nil config is valid",
			config:    nil,
			wantError: false,
		},
		{
			name: "valid config",
			config: &ReviewConfig{
				MaxTokens:   12000,
				TargetFiles: []string{"*.go", "*.md"},
			},
			wantError: false,
		},
		{
			name: "negative max tokens",
			config: &ReviewConfig{
				MaxTokens: -100,
			},
			wantError: true,
		},
		{
			name: "max tokens exceeds limit",
			config: &ReviewConfig{
				MaxTokens: MaxAllowedTokens + 1000,
			},
			wantError: true,
		},
		{
			name: "empty target file pattern",
			config: &ReviewConfig{
				TargetFiles: []string{"*.go", "", "*.md"},
			},
			wantError: true,
		},
		{
			name: "custom prompt too short",
			config: &ReviewConfig{
				CustomSystemPrompt: "Hi",
			},
			wantError: true,
		},
		{
			name: "valid custom prompt",
			config: &ReviewConfig{
				CustomSystemPrompt: "This is a sufficiently long custom system prompt for review mode that meets the minimum requirements",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review := &Review{config: tt.config}
			err := review.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Review.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestReview_SystemPrompt_WithCustom(t *testing.T) {
	customPrompt := "Custom review system prompt that is sufficiently long to meet minimum requirements for the system"
	review := &Review{
		config: &ReviewConfig{
			CustomSystemPrompt: customPrompt,
		},
	}

	result := review.SystemPrompt()
	if result != customPrompt {
		t.Errorf("Review.SystemPrompt() = %v, want %v", result, customPrompt)
	}
}

func TestReview_MaxTokens_WithCustom(t *testing.T) {
	customTokens := 15000
	review := &Review{
		config: &ReviewConfig{
			MaxTokens: customTokens,
		},
	}

	result := review.MaxTokens()
	if result != customTokens {
		t.Errorf("Review.MaxTokens() = %v, want %v", result, customTokens)
	}
}

func TestReview_Concurrency(t *testing.T) {
	review := NewReview()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = review.Name()
			_ = review.SystemPrompt()
			_ = review.AllowedTools()
			_ = review.MaxTokens()
			_ = review.Validate()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
