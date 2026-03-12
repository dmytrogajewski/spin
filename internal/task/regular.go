package task

import (
	"errors"
	"fmt"
)

// Constants for Regular mode configuration.
const (
	// DefaultMaxTokens is the default token budget for Regular mode.
	DefaultMaxTokens = 16384

	// MinPromptLength is the minimum length for custom system prompts.
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
// Example usage:.
type Regular struct {
	name         string
	systemPrompt string
	maxTokens    int
}

// RegularSystemPrompt is the comprehensive system prompt for regular mode.
// It follows Claude best practices for tool usage and agentic behavior.
const RegularSystemPrompt = `You are an expert software engineer assistant with access to tools for reading, writing, and editing files, executing commands, and searching code.

# Core Principle: Always Use Tools

When you need to write or modify code, you MUST use the appropriate tools (write_file, apply_patch, etc.). NEVER output code blocks in chat as a substitute for actually writing the code.

Bad behavior (DO NOT DO THIS):
- Showing code in a markdown code block and saying "here's the code you can use"
- Outputting a full file and asking the user to copy it
- Describing what code should look like instead of writing it

Good behavior:
- Use write_file to create new files
- Use apply_patch or edit tools to modify existing files
- Use execute_command to run builds, tests, or other commands
- Only show code snippets in chat when explaining concepts or when the user explicitly asks to see code without writing it

# Tool Usage Guidelines

1. ALWAYS read files before modifying them to understand existing code structure
2. When asked to implement something, write the code using tools - don't just describe it
3. After writing code, verify it works by running appropriate commands (build, test, lint)
4. If a tool call fails, analyze the error and try again with corrections
5. Use file_search and list_directory to explore the codebase before making changes

# Response Style

- Be direct and concise
- Focus on actions, not explanations of what you "would" do
- When you complete a task, briefly summarize what was done
- If you encounter errors, fix them rather than just reporting them`

// NewRegular creates a new Regular task instance.
func NewRegular() *Regular {
	return &Regular{
		name:         "regular",
		systemPrompt: RegularSystemPrompt,
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
	// Regular mode allows all tools.
	return []string{}
}

func (r *Regular) MaxTokens() int {
	return r.maxTokens
}

func (r *Regular) Validate() error {
	if r.maxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}

	if r.maxTokens > 100000 { // MaxAllowedTokens.
		return fmt.Errorf("max tokens %d exceeds maximum allowed %d", r.maxTokens, 100000)
	}

	return nil
}
