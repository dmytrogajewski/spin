package compress

import (
	"strings"
)

// MessageImportance represents the priority level of a message for compression.
type MessageImportance int

const (
	// ImportanceLow indicates the message can be compressed first.
	// Examples: verbose reasoning, "thinking" content
	ImportanceLow MessageImportance = 0

	// ImportanceMedium indicates the message has standard importance.
	// Examples: regular assistant responses
	ImportanceMedium MessageImportance = 1

	// ImportanceHigh indicates the message should be preserved if possible.
	// Examples: code changes, decisions
	ImportanceHigh MessageImportance = 2

	// ImportanceCritical indicates the message must be preserved.
	// Examples: user messages, tool results, errors
	ImportanceCritical MessageImportance = 3
)

// Message is the minimal interface needed for compression.
// This avoids circular imports with core package.
type Message interface {
	GetRole() string
	GetContent() string
	GetToolCallCount() int
}

// Role constants (mirror core package)
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// MessageClassifier assigns importance levels to messages based on their content and role.
//
// Classification Rules:
//   - Critical: User messages, tool results, errors, system messages
//   - High: Messages with code blocks or diffs
//   - Medium: Regular assistant responses
//   - Low: Verbose responses (>1000 chars without code), empty messages
//
// The classifier uses deterministic rules (no ML or randomness) for predictable behavior.
type MessageClassifier struct{}

// Classify assigns an importance level to a message.
//
// The classification is deterministic and based on message characteristics:
// role, content length, presence of code/diffs, and keywords.
func (c *MessageClassifier) Classify(msg Message) MessageImportance {
	role := msg.GetRole()
	content := msg.GetContent()
	toolCallCount := msg.GetToolCallCount()

	// Critical: System messages (always preserve system prompts)
	if role == RoleSystem {
		return ImportanceCritical
	}

	// Critical: User messages (always preserve user intent)
	if role == RoleUser {
		return ImportanceCritical
	}

	// Critical: Tool results (preserve command outputs, file contents)
	if role == RoleTool {
		return ImportanceCritical
	}

	// Critical: Messages with tool calls (preserve tool invocations)
	if toolCallCount > 0 {
		return ImportanceCritical
	}

	// Critical: Error messages (check content for error keywords)
	if c.isError(content) {
		return ImportanceCritical
	}

	// For assistant messages, classify by content
	if role == RoleAssistant {
		return c.classifyAssistantMessage(content)
	}

	// Default: Low importance for unknown roles
	return ImportanceLow
}

// isError checks if content represents an error condition.
func (c *MessageClassifier) isError(content string) bool {
	// Check for error keywords in content (case-insensitive)
	contentLower := strings.ToLower(content)
	errorKeywords := []string{"error:", "error ", "failed:", "failed ", "exception:", "exception "}

	for _, keyword := range errorKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// classifyAssistantMessage classifies assistant role messages by content.
func (c *MessageClassifier) classifyAssistantMessage(content string) MessageImportance {

	// Empty messages: Low
	if len(strings.TrimSpace(content)) == 0 {
		return ImportanceLow
	}

	// High: Contains code blocks (markdown fenced code)
	if strings.Contains(content, "```") {
		return ImportanceHigh
	}

	// High: Contains diff/patch markers
	if strings.Contains(content, "@@") {
		return ImportanceHigh
	}

	// Low: Verbose responses (>1000 chars) without code
	// These are typically "thinking" or lengthy reasoning
	if len(content) > 1000 {
		return ImportanceLow
	}

	// Medium: Regular assistant responses
	return ImportanceMedium
}
