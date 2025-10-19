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
//
//	review := task.NewReview()
//	fmt.Println("Mode:", review.Name())
//	fmt.Println("Tools:", review.AllowedTools()) // Read-only tools only
//	fmt.Println("Max Tokens:", review.MaxTokens())
type Review struct {
	config *ReviewConfig
}

// ReviewConfig contains configuration options for Review mode.
//
// All fields are optional. Zero values will use sensible defaults:
//   - MaxTokens: 0 uses DefaultReviewMaxTokens (12288)
//   - TargetFiles: empty reviews all files in workspace
//   - CustomSystemPrompt: empty uses default review prompt
//   - IncludeGitOps: defaults to true
type ReviewConfig struct {
	// TargetFiles optionally restricts review to specific files.
	// Empty slice means review all files in workspace.
	// Supports glob patterns (e.g., "*.go", "src/**/*.ts").
	TargetFiles []string

	// MaxTokens overrides the default token budget.
	// If 0, uses DefaultReviewMaxTokens (12288 tokens).
	// Maximum allowed is MaxAllowedTokens (100000).
	MaxTokens int

	// CustomSystemPrompt optionally overrides the default system prompt.
	// Empty string uses the default review-focused prompt.
	// Must be at least MinPromptLength (50) characters.
	CustomSystemPrompt string

	// IncludeGitOps enables read-only Git operations.
	// Default: true (allows git_status, git_diff, git_log).
	// Set to false to disable all Git operations.
	IncludeGitOps bool
}

// NewReview creates a new Review task mode with default configuration.
// The returned task uses all default settings:
//   - Token budget: 12288 (smaller than Regular)
//   - All read-only tools enabled
//   - Git operations enabled
//   - Default review-focused system prompt
func NewReview() *Review {
	return &Review{
		config: nil,
	}
}

// Name returns the unique identifier for this task mode.
// Always returns "review".
func (r *Review) Name() string {
	return "review"
}

// SystemPrompt returns the mode-specific system prompt that defines
// the agent's behavior and constraints for Review mode.
//
// If a custom prompt is configured, it will be used instead of the default.
// The default prompt emphasizes read-only analysis and structured feedback.
func (r *Review) SystemPrompt() string {
	if r.config != nil && r.config.CustomSystemPrompt != "" {
		return r.config.CustomSystemPrompt
	}
	return defaultReviewPrompt
}

// defaultReviewPrompt is the comprehensive system prompt for Review mode
const defaultReviewPrompt = `You are Spin in Review Mode, a code review and analysis assistant.

YOUR ROLE:
You are operating in read-only mode. Your purpose is to analyze, review, and provide insights about code without making any modifications.

CAPABILITIES (Read-Only):
- Read files and examine code structure
- Search codebase for patterns and symbols
- List directory contents
- View Git status, diffs, and history (if enabled)
- Analyze code quality and potential issues

YOU CANNOT modify any files:
- Cannot modify any files
- Cannot execute shell commands
- Cannot write or create files
- Cannot commit or stage changes

REVIEW FOCUS AREAS:
1. Code Quality
   - Readability and maintainability
   - Adherence to best practices
   - Code style and conventions
   - DRY and SOLID principles

2. Potential Issues
   - Logic errors and edge cases
   - Performance concerns
   - Security vulnerabilities
   - Memory leaks and resource management

3. Architecture & Design
   - Code organization and structure
   - Separation of concerns
   - Appropriate use of patterns
   - API design and interfaces

4. Testing & Documentation
   - Test coverage and quality
   - Documentation completeness
   - Code comments clarity
   - Example usage

BEHAVIOR:
- Provide constructive, actionable feedback
- Explain the reasoning behind suggestions
- Prioritize issues by severity (Critical, High, Medium, Low)
- Suggest specific improvements with examples
- Be thorough but concise
- Acknowledge good practices when found

OUTPUT FORMAT:
Structure your review with clear sections:
- Summary: High-level overview
- Critical Issues: Must-fix problems
- Suggestions: Improvements and best practices
- Positive Notes: Good practices to acknowledge
- Recommendations: Next steps

Remember: Your goal is to help improve code quality through insightful analysis, not to make changes directly.`

// AllowedTools returns the list of tool names that are permitted
// in Review mode.
//
// Review mode only allows read-only tools:
//   - File operations: read_file, list_dir, search_files
//   - Code operations: search_code, get_context
//   - Git operations: git_status, git_diff, git_log (if enabled)
//
// Write operations are explicitly excluded:
//   - No write_file
//   - No shell
//   - No git_add or git_commit
func (r *Review) AllowedTools() []string {
	// Read-only tools for code review (5 tools)
	tools := []string{
		// File operations (read-only)
		"read_file",
		"list_directory",

		// Context and search (read-only)
		"get_context",
		"file_search",
		"git_context",
	}

	// Review mode excludes write operations:
	// - No write_file
	// - No execute_command
	// - No apply_patch

	return tools
}

// MaxTokens returns the maximum token budget for Review mode.
//
// Returns the configured MaxTokens if set and greater than 0,
// otherwise returns DefaultReviewMaxTokens (12288).
//
// Review mode uses a smaller default budget than Regular mode
// since reviews typically focus on specific files or changes.
func (r *Review) MaxTokens() int {
	if r.config != nil && r.config.MaxTokens > 0 {
		return r.config.MaxTokens
	}
	return DefaultReviewMaxTokens
}

// Validate validates the Review task configuration and returns an error
// if the configuration is invalid.
//
// Validation checks:
//   - MaxTokens must not be negative
//   - MaxTokens must not exceed MaxAllowedTokens
//   - TargetFiles patterns must not be empty strings
//   - CustomSystemPrompt must be at least MinPromptLength if provided
//
// A nil config is always valid (uses defaults).
// Multiple validation errors are joined together.
func (r *Review) Validate() error {
	if r.config == nil {
		return nil // Default config is always valid
	}

	var errs []error
	errs = append(errs, r.validateMaxTokens()...)
	errs = append(errs, r.validateTargetFiles()...)
	errs = append(errs, r.validateCustomSystemPrompt()...)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// validateMaxTokens validates the max tokens configuration.
func (r *Review) validateMaxTokens() []error {
	var errs []error
	if r.config.MaxTokens < 0 {
		errs = append(errs, fmt.Errorf("max tokens cannot be negative: %d", r.config.MaxTokens))
	}
	if r.config.MaxTokens > MaxAllowedTokens {
		errs = append(errs, fmt.Errorf("max tokens (%d) exceeds maximum allowed (%d)", r.config.MaxTokens, MaxAllowedTokens))
	}
	return errs
}

// validateTargetFiles validates the target files configuration.
func (r *Review) validateTargetFiles() []error {
	var errs []error
	for i, pattern := range r.config.TargetFiles {
		if pattern == "" {
			errs = append(errs, fmt.Errorf("target file pattern at index %d cannot be empty", i))
		}
	}
	return errs
}

// validateCustomSystemPrompt validates the custom system prompt configuration.
func (r *Review) validateCustomSystemPrompt() []error {
	var errs []error
	if r.config.CustomSystemPrompt != "" && len(r.config.CustomSystemPrompt) < MinPromptLength {
		errs = append(errs, fmt.Errorf("custom system prompt too short (%d characters, minimum %d)", len(r.config.CustomSystemPrompt), MinPromptLength))
	}
	return errs
}
