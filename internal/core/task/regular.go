package task

import (
	"errors"
	"fmt"
)

// Constants for Regular mode configuration
const (
	// DefaultMaxTokens is the default token budget for Regular mode
	DefaultMaxTokens = 16384

	// MaxAllowedTokens is the maximum allowed token budget
	MaxAllowedTokens = 100000

	// MinPromptLength is the minimum length for custom system prompts
	MinPromptLength = 50
)

// Regular implements the standard interactive coding mode.
// This is the default mode for Spin, providing full access to all tools
// including file operations, shell commands, Git, and code search.
//
// Regular mode is designed for:
//   - Interactive coding sessions
//   - Full-featured development workflows
//   - Complex multi-step tasks
//   - Tasks requiring all available tools
//
// Example usage:
//
//	regular := task.NewRegular()
//	fmt.Println("Mode:", regular.Name())
//	fmt.Println("Tools:", regular.AllowedTools())
//	fmt.Println("Max Tokens:", regular.MaxTokens())
type Regular struct {
	config *RegularConfig
}

// RegularConfig contains configuration options for Regular mode.
//
// All fields are optional. Zero values will use sensible defaults:
//   - MaxTokens: 0 uses DefaultMaxTokens (16384)
//   - ExcludedTools: empty allows all tools
//   - CustomSystemPrompt: empty uses default prompt
type RegularConfig struct {
	// MaxTokens overrides the default token budget.
	// If 0, uses DefaultMaxTokens (16384 tokens).
	// Maximum allowed is MaxAllowedTokens (100000).
	MaxTokens int

	// ExcludedTools optionally restricts certain tools.
	// Empty slice means all tools are allowed.
	// Tool names must match registered tool names.
	ExcludedTools []string

	// CustomSystemPrompt optionally overrides the default system prompt.
	// Empty string uses the default prompt.
	// Must be at least MinPromptLength (50) characters.
	CustomSystemPrompt string
}

// NewRegular creates a new Regular task mode with default configuration.
// The returned task uses all default settings:
//   - Token budget: 16384
//   - All tools enabled
//   - Default system prompt
func NewRegular() *Regular {
	return &Regular{
		config: nil,
	}
}

// NewRegularWithConfig creates a new Regular task mode with custom configuration.
// Pass nil config to use all defaults (equivalent to NewRegular()).
func NewRegularWithConfig(config *RegularConfig) *Regular {
	return &Regular{
		config: config,
	}
}

// Name returns the unique identifier for this task mode.
// Always returns "regular".
func (r *Regular) Name() string {
	return "regular"
}

// SystemPrompt returns the mode-specific system prompt that defines
// the agent's behavior and constraints for Regular mode.
//
// If a custom prompt is configured, it will be used instead of the default.
// The default prompt provides comprehensive guidance on capabilities,
// behavior, constraints, and workflow.
func (r *Regular) SystemPrompt() string {
	if r.config != nil && r.config.CustomSystemPrompt != "" {
		return r.config.CustomSystemPrompt
	}
	return defaultSystemPrompt
}

// defaultSystemPrompt is the comprehensive system prompt for Regular mode
const defaultSystemPrompt = `You are Spin, an autonomous coding agent designed to help developers with their coding tasks.

CAPABILITIES:
- Read and write files in the workspace
- Execute shell commands (with user approval for dangerous operations)
- Use Git for version control operations
- Search codebase for patterns and symbols
- Analyze code structure and dependencies
- List directory contents and search for files
- Gather environment and project context

BEHAVIOR:
- Be helpful, precise, and efficient
- Always explain what you're doing and why
- Ask for clarification when requirements are unclear
- Suggest best practices and improvements
- Handle errors gracefully and provide clear feedback
- Work incrementally and verify results at each step
- Respect existing code style and conventions

CONSTRAINTS:
- Only operate within the specified workspace directory
- Request approval for potentially dangerous operations (rm, sudo, etc.)
- Never expose sensitive information (API keys, passwords, etc.)
- Follow security best practices
- Respect .gitignore and other exclusion patterns
- Do not modify system files or directories outside workspace

WORKFLOW:
1. Understand the user's request thoroughly
2. Plan your approach (use planning for complex tasks)
3. Execute step-by-step with clear communication
4. Verify results and handle errors appropriately
5. Provide a summary of what was accomplished

AGENTIC EXECUTION GUIDELINES:
- Prefer making decisions and taking action over asking permission for routine steps
- Default to writing and editing code directly using available tools when the task implies changes
- Propose concrete edits and then apply them via tools unless expressly told not to
- When ambiguity exists, state assumptions briefly and proceed with a sensible default
- After each meaningful step, validate via tests, lints, or quick checks and iterate

OUTPUT STYLE:
- Communicate succinctly; prioritize code and decisions over long prose
- Use minimal explanations that are necessary to understand choices and next actions
- Summarize the impact of changes at the end

Remember: You are a decisive builder. Optimize for shipping working code with safety and verification.`

// AllowedTools returns the list of tool names that are permitted
// in Regular mode.
//
// By default, all available tools are allowed. If ExcludedTools
// is configured, those tools will be filtered out.
//
// Available tools include (8 total):
//   - File operations: read_file, write_file, list_directory
//   - Command execution: execute_command
//   - Context and search: get_context, file_search
//   - Advanced operations: apply_patch, git_context
func (r *Regular) AllowedTools() []string {
	// Default tool set for Regular mode - all available tools
	tools := []string{
		// File operations
		"read_file",
		"write_file",
		"list_directory",

		// Command execution
		"execute_command",

		// Context and search
		"get_context",
		"file_search",

		// Advanced operations
		"apply_patch",
		"git_context",
	}

	// Filter out excluded tools if configured
	if r.config != nil && len(r.config.ExcludedTools) > 0 {
		return filterTools(tools, r.config.ExcludedTools)
	}

	return tools
}

// MaxTokens returns the maximum token budget for Regular mode.
//
// Returns the configured MaxTokens if set and greater than 0,
// otherwise returns DefaultMaxTokens (16384).
//
// The token budget affects context window size and history truncation.
func (r *Regular) MaxTokens() int {
	if r.config != nil && r.config.MaxTokens > 0 {
		return r.config.MaxTokens
	}
	return DefaultMaxTokens
}

// Validate validates the Regular task configuration and returns an error
// if the configuration is invalid.
//
// Validation checks:
//   - MaxTokens must not be negative
//   - MaxTokens must not exceed MaxAllowedTokens
//   - ExcludedTools must not contain empty strings
//   - CustomSystemPrompt must be at least MinPromptLength if provided
//
// A nil config is always valid (uses defaults).
// Multiple validation errors are joined together.
//
//nolint:dupl // Validation logic is similar but validates different config fields per mode
func (r *Regular) Validate() error {
	if r.config == nil {
		return nil // Default config is always valid
	}

	var errs []error

	// Validate max tokens
	if r.config.MaxTokens < 0 {
		errs = append(errs, fmt.Errorf("max tokens cannot be negative: %d", r.config.MaxTokens))
	}
	if r.config.MaxTokens > MaxAllowedTokens {
		errs = append(errs, fmt.Errorf("max tokens (%d) exceeds maximum allowed (%d)", r.config.MaxTokens, MaxAllowedTokens))
	}

	// Validate excluded tools
	for i, tool := range r.config.ExcludedTools {
		if tool == "" {
			errs = append(errs, fmt.Errorf("excluded tool at index %d cannot be empty", i))
		}
	}

	// Validate custom system prompt
	if r.config.CustomSystemPrompt != "" && len(r.config.CustomSystemPrompt) < MinPromptLength {
		errs = append(errs, fmt.Errorf("custom system prompt too short (%d characters, minimum %d)", len(r.config.CustomSystemPrompt), MinPromptLength))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// filterTools removes excluded tools from the tool list.
// Returns a new slice with excluded tools removed, preserving order.
// If a tool in excluded is not in tools, it is silently ignored.
func filterTools(tools []string, excluded []string) []string {
	// Build exclusion map for O(1) lookup
	excludeMap := make(map[string]bool, len(excluded))
	for _, tool := range excluded {
		excludeMap[tool] = true
	}

	// Filter tools
	filtered := make([]string, 0, len(tools))
	for _, tool := range tools {
		if !excludeMap[tool] {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}
