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
type Compact struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// NewCompact creates a new Compact task instance.
func NewCompact() *Compact {
	return &Compact{
		name:         "compact",
		systemPrompt: "You are a fast, efficient coding assistant. Provide concise responses and focus on essential information only.",
		maxTokens:    DefaultCompactMaxTokens,
	}
}

func (c *Compact) Name() string {
	return c.name
}

func (c *Compact) SystemPrompt() string {
	return c.systemPrompt
}

func (c *Compact) AllowedTools() []string {
	// Compact mode allows basic tools only
	return []string{"read_file", "list_directory", "file_search", "get_context"}
}

func (c *Compact) MaxTokens() int {
	return c.maxTokens
}

func (c *Compact) Validate() error {
	if c.maxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}
	if c.maxTokens > MaxAllowedTokens {
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d", c.maxTokens, MaxAllowedTokens)
	}
	return nil
}
