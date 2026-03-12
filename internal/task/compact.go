// Package task provides task execution and compaction.
package task

import (
	"fmt"
)

// Constants for Compact mode configuration.
const (
	// DefaultCompactMaxTokens is the default token budget for Compact mode
	// Smallest budget of all modes for maximum efficiency.
	DefaultCompactMaxTokens = 4096
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
// Example usage:.
type Compact struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// CompactSystemPrompt is the system prompt for compact mode.
// Optimized for quick, focused tasks with minimal token usage.
const CompactSystemPrompt = `You are a fast, efficient coding assistant optimized for quick tasks.

# Constraints

- Limited tool access: read_file, list_directory, file_search, get_context
- No file modification or command execution in this mode
- Keep responses brief and focused

# Guidelines

1. Answer questions directly without lengthy explanations
2. When searching, use precise patterns to minimize results
3. Provide code references (file:line) rather than full code blocks
4. If a task requires file modification, inform the user to switch to regular mode`

// NewCompact creates a new Compact task instance.
func NewCompact() *Compact {
	return &Compact{
		name:         TaskNameCompact,
		systemPrompt: CompactSystemPrompt,
		maxTokens:    DefaultCompactMaxTokens,
	}
}

// Name implements the Name operation.
func (c *Compact) Name() string {
	return c.name
}

// SystemPrompt implements the SystemPrompt operation.
func (c *Compact) SystemPrompt() string {
	return c.systemPrompt
}

// AllowedTools implements the AllowedTools operation.
func (c *Compact) AllowedTools() []string {
	// Compact mode allows basic tools only.
	return []string{"read_file", "list_directory", "file_search", "get_context"}
}

// MaxTokens implements the MaxTokens operation.
func (c *Compact) MaxTokens() int {
	return c.maxTokens
}

// Validate implements the Validate operation.
func (c *Compact) Validate() error {
	if c.maxTokens <= 0 {
		return ErrMaxTokensMustBePositive
	}

	if c.maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d: %w", c.maxTokens, MaxAllowedTokens, ErrMaxTokensExceedsMaximumAllowed)
	}

	return nil
}
