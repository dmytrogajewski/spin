package task

import (
	"errors"
	"fmt"
)

// Constants for Review mode configuration
const (
	// DefaultReviewMaxTokens is the default token budget for Review mode
	// Smaller than Regular mode since reviews typically need less context
	DefaultReviewMaxTokens = 12288
)

// Review implements read-only code review mode.
// This mode is designed for code analysis and review scenarios where
// modifications should not be made.
//
// Review mode is designed for:
//   - Code reviews and pull request analysis
//   - Security audits and vulnerability scanning
//   - Documentation review
//   - Learning and exploration without side effects
//   - Safe mode in restricted environments
//
// Example usage:
type Review struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// NewReview creates a new Review task instance.
func NewReview() *Review {
	return &Review{
		name:         "review",
		systemPrompt: "You are a code review assistant. You can analyze code, identify issues, and provide feedback, but you cannot modify files or execute commands.",
		maxTokens:    DefaultReviewMaxTokens,
	}
}

func (r *Review) Name() string {
	return r.name
}

func (r *Review) SystemPrompt() string {
	return r.systemPrompt
}

func (r *Review) AllowedTools() []string {
	// Review mode allows only read-only tools
	return []string{"read_file", "list_directory", "file_search", "git_context", "get_context"}
}

func (r *Review) MaxTokens() int {
	return r.maxTokens
}

func (r *Review) Validate() error {
	if r.maxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}
	if r.maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d", r.maxTokens, MaxAllowedTokens)
	}
	return nil
}
