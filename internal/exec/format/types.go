// Package format provides output formatters for exec mode.
package format

import (
	"time"
)

// OutputFormat represents the output format type.
type OutputFormat string

const (
	// FormatText is human-readable text output
	FormatText OutputFormat = "text"
	// FormatJSON is structured JSON output
	FormatJSON OutputFormat = "json"
)

// Formatter is the interface for output formatters.
type Formatter interface {
	// FormatStart formats the initial message
	FormatStart(prompt string) string

	// FormatDelta formats a streaming chunk
	FormatDelta(delta string) string

	// FormatComplete formats the completion message
	FormatComplete(result *ExecResult) string

	// FormatError formats an error message
	FormatError(err error) string
}

// ExecResult represents the result of an exec run.
type ExecResult struct {
	// Status indicates completion state: "complete", "failed", "timeout", "cancelled"
	Status string

	// Messages contains the conversation history
	Messages []Message

	// FilesModified contains paths of files that were modified
	FilesModified []string

	// CommandsRun contains commands that were executed
	CommandsRun []CommandLog

	// TokensUsed is the total number of tokens consumed
	TokensUsed int

	// Duration is the total execution time
	Duration time.Duration

	// Error contains any error that occurred
	Error error
}

// Message represents a single message in the conversation.
type Message struct {
	// Role is the message sender: "user", "assistant", "system"
	Role string

	// Content is the message text
	Content string

	// Timestamp is when the message was created
	Timestamp time.Time
}

// CommandLog represents a command that was executed.
type CommandLog struct {
	// Command is the command that was run
	Command string

	// ExitCode is the exit code from the command
	ExitCode int

	// Output is the command output (may be truncated)
	Output string
}
