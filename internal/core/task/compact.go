package task

import (
	"errors"
	"fmt"
)

// Constants for Compact mode configuration
const (
	// DefaultCompactMaxTokens is the default token budget for Compact mode
	// Smallest budget of all modes for maximum efficiency
	DefaultCompactMaxTokens = 4096

	// MaxAllowedTokens and MinPromptLength are shared with other task modes
	// and defined at package level (see regular.go)
)

// Compact implements minimal context mode for quick tasks and constrained environments.
// This mode prioritizes speed and efficiency over comprehensive context.
//
// Compact mode is designed for:
//   - Quick, simple tasks (file reads, searches, status checks)
//   - Constrained environments (limited memory/CPU)
//   - Fast response requirements (minimize latency)
//   - Cost optimization (reduce token usage)
//   - Batch operations (many small requests)
//
// Example usage:
//
//	compact := task.NewCompact()
//	fmt.Println("Mode:", compact.Name())
//	fmt.Println("Tools:", compact.AllowedTools()) // Minimal set: 3 tools
//	fmt.Println("Max Tokens:", compact.MaxTokens()) // 4096 - smallest budget
type Compact struct {
	config *CompactConfig
}

// CompactConfig contains configuration options for Compact mode.
//
// All fields are optional. Zero values will use sensible defaults:
//   - MaxTokens: 0 uses DefaultCompactMaxTokens (4096)
//   - AdditionalTools: empty uses only essential tools
//   - CustomSystemPrompt: empty uses default minimal prompt
type CompactConfig struct {
	// MaxTokens overrides the default token budget.
	// If 0, uses DefaultCompactMaxTokens (4096 tokens).
	// Maximum allowed is MaxAllowedTokens (100000).
	MaxTokens int

	// AdditionalTools optionally adds more tools to the minimal set.
	// Default minimal set: read_file, list_dir, search_code.
	// Tools are appended to the base set.
	AdditionalTools []string

	// CustomSystemPrompt optionally overrides the default system prompt.
	// Empty string uses the default minimal, concise prompt.
	// Must be at least MinPromptLength (50) characters.
	CustomSystemPrompt string
}

// NewCompact creates a new Compact task mode with default configuration.
// The returned task uses all default settings:
//   - Token budget: 4096 (smallest)
//   - Essential tools only: read_file, list_dir, search_code
//   - Default minimal system prompt
func NewCompact() *Compact {
	return &Compact{
		config: nil,
	}
}

// NewCompactWithConfig creates a new Compact task mode with custom configuration.
// Pass nil config to use all defaults (equivalent to NewCompact()).
func NewCompactWithConfig(config *CompactConfig) *Compact {
	return &Compact{
		config: config,
	}
}

// Name returns the unique identifier for this task mode.
// Always returns "compact".
func (c *Compact) Name() string {
	return "compact"
}

// SystemPrompt returns the mode-specific system prompt that defines
// the agent's behavior and constraints for Compact mode.
//
// If a custom prompt is configured, it will be used instead of the default.
// The default prompt is minimal and concise, emphasizing speed and directness.
func (c *Compact) SystemPrompt() string {
	if c.config != nil && c.config.CustomSystemPrompt != "" {
		return c.config.CustomSystemPrompt
	}
	return defaultCompactPrompt
}

// defaultCompactPrompt is the minimal, concise system prompt for Compact mode
const defaultCompactPrompt = `You are Spin in Compact Mode - optimized for quick, focused tasks.

MODE: Minimal context, fast responses, essential operations only.

CORE CAPABILITIES:
- Read files and examine code
- List directories and search files
- Search code for patterns

GUIDELINES:
- Be concise and direct
- Focus on the specific task
- Avoid lengthy explanations unless asked
- Use minimal context - only what's needed
- Provide actionable answers quickly

CONSTRAINTS:
- Limited token budget (4096)
- Minimal tool set (3 essential tools)
- Focus on efficiency over comprehensiveness

Remember: Speed and clarity over comprehensiveness. Answer the question directly.`

// AllowedTools returns the list of tool names that are permitted
// in Compact mode.
//
// By default, returns a minimal essential set of 3 tools:
//   - read_file: Read file contents
//   - list_dir: List directory contents
//   - search_code: Search for code patterns
//
// If AdditionalTools are configured, they are appended to the base set.
func (c *Compact) AllowedTools() []string {
	// Minimal essential tool set for Compact mode
	tools := []string{
		"read_file",
		"list_dir",
		"search_code",
	}

	// Add any additional tools if configured
	if c.config != nil && len(c.config.AdditionalTools) > 0 {
		tools = append(tools, c.config.AdditionalTools...)
	}

	return tools
}

// MaxTokens returns the maximum token budget for Compact mode.
//
// Returns the configured MaxTokens if set and greater than 0,
// otherwise returns DefaultCompactMaxTokens (4096).
//
// Compact mode has the smallest token budget of all modes,
// optimized for quick responses and minimal context.
func (c *Compact) MaxTokens() int {
	if c.config != nil && c.config.MaxTokens > 0 {
		return c.config.MaxTokens
	}
	return DefaultCompactMaxTokens
}

// Validate validates the Compact task configuration and returns an error
// if the configuration is invalid.
//
// Validation checks:
//   - MaxTokens must not be negative
//   - MaxTokens must not exceed MaxAllowedTokens
//   - AdditionalTools must not contain empty strings
//   - CustomSystemPrompt must be at least MinPromptLength if provided
//
// A nil config is always valid (uses defaults).
// Multiple validation errors are joined together.
func (c *Compact) Validate() error {
	if c.config == nil {
		return nil // Default config is always valid
	}

	var errs []error

	// Validate max tokens
	if c.config.MaxTokens < 0 {
		errs = append(errs, fmt.Errorf("max tokens cannot be negative: %d", c.config.MaxTokens))
	}
	if c.config.MaxTokens > MaxAllowedTokens {
		errs = append(errs, fmt.Errorf("max tokens (%d) exceeds maximum allowed (%d)", c.config.MaxTokens, MaxAllowedTokens))
	}

	// Validate additional tools
	for i, tool := range c.config.AdditionalTools {
		if tool == "" {
			errs = append(errs, fmt.Errorf("additional tool at index %d cannot be empty", i))
		}
	}

	// Validate custom system prompt
	if c.config.CustomSystemPrompt != "" && len(c.config.CustomSystemPrompt) < MinPromptLength {
		errs = append(errs, fmt.Errorf("custom system prompt too short (%d characters, minimum %d)", len(c.config.CustomSystemPrompt), MinPromptLength))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
