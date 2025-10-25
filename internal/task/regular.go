package task

import (
	"errors"
	"fmt"
)

// Constants for Regular mode configuration
const (
	// DefaultMaxTokens is the default token budget for Regular mode
	DefaultMaxTokens = 16384

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
type Regular struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// NewRegular creates a new Regular task instance.
func NewRegular() *Regular {
	return &Regular{
		name:         "regular",
		systemPrompt: "You are a helpful coding assistant. You have access to all available tools and can help with any coding task.",
		maxTokens:    DefaultMaxTokens,
	}
}

func (r *Regular) Name() string {
	return r.name
}

func (r *Regular) SystemPrompt() string {
	return r.systemPrompt
}

func (r *Regular) AllowedTools() []string {
	// Regular mode allows all tools
	return []string{}
}

func (r *Regular) MaxTokens() int {
	return r.maxTokens
}

func (r *Regular) Validate() error {
	if r.maxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}
	if r.maxTokens > 100000 { // MaxAllowedTokens
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d", r.maxTokens, 100000)
	}
	return nil
}
