package task

import (
	"fmt"
)

// Constants for Review mode configuration.
const (
	// DefaultReviewMaxTokens is the default token budget for Review mode
	// Smaller than Regular mode since reviews typically need less context.
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
// Example usage:.
type Review struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// ReviewSystemPrompt is the system prompt for review mode.
// It emphasizes read-only analysis without modification capabilities.
const ReviewSystemPrompt = `You are an expert code reviewer. You have read-only access to analyze code, identify issues, and provide detailed feedback.

# Your Role

You can:
- Read and analyze source code files
- Search for patterns and potential issues
- Review code quality, architecture, and best practices
- Identify bugs, security vulnerabilities, and performance issues
- Suggest improvements with specific, actionable feedback

You cannot:
- Modify files or write code
- Execute commands or run tests

# Review Guidelines

1. Use read_file and file_search to thoroughly examine the code
2. Provide specific line references when discussing issues
3. Categorize feedback by severity (critical, warning, suggestion)
4. Explain WHY something is an issue, not just WHAT the issue is
5. When suggesting fixes, describe the approach clearly since you cannot implement it directly

# Response Style

- Be constructive and specific
- Prioritize critical issues over style nitpicks
- Provide actionable feedback with clear explanations`

// NewReview creates a new Review task instance.
func NewReview() *Review {
	return &Review{
		name:         "review",
		systemPrompt: ReviewSystemPrompt,
		maxTokens:    DefaultReviewMaxTokens,
	}
}

// Name implements the Name operation.
func (r *Review) Name() string {
	return r.name
}

// SystemPrompt implements the SystemPrompt operation.
func (r *Review) SystemPrompt() string {
	return r.systemPrompt
}

// AllowedTools implements the AllowedTools operation.
func (r *Review) AllowedTools() []string {
	// Review mode allows only read-only tools.
	return []string{"read_file", "list_directory", "file_search", "git_context", "get_context"}
}

// MaxTokens implements the MaxTokens operation.
func (r *Review) MaxTokens() int {
	return r.maxTokens
}

// Validate implements the Validate operation.
func (r *Review) Validate() error {
	if r.maxTokens <= 0 {
		return ErrMaxTokensMustBePositive
	}

	if r.maxTokens > MaxAllowedTokens {
return fmt.Errorf("max tokens %d exceeds maximum allowed %d: %w", r.maxTokens, MaxAllowedTokens, ErrMaxTokensExceedsMaximumAllowed)
	}

	return nil
}
