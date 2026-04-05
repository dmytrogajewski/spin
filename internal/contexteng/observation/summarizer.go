// Package observation provides per-tool-type summarization for tool
// result outputs. It applies compact summaries to large outputs and
// passes short outputs through verbatim.
package observation

import (
	"fmt"
	"strings"
)

// Thresholds for output handling.
const (
	// ShortOutputMax is the maximum character count for verbatim pass-through.
	ShortOutputMax = 100

	// ErrorTruncateMax is the maximum character count for error messages.
	ErrorTruncateMax = 200
)

// Known tool name prefixes for strategy selection.
const (
	ToolReadFile      = "read_file"
	ToolFileSearch    = "file_search"
	ToolListDirectory = "list_directory"
	ToolShellCommand  = "shell_command"
)

// Strategy defines how a specific tool type's output is summarized.
type Strategy interface {
	Summarize(toolName, output string) string
}

// Summarizer applies per-tool-type summarization to tool outputs.
type Summarizer struct {
	strategies map[string]Strategy
	fallback   Strategy
}

// NewSummarizer creates a Summarizer with default per-tool strategies.
func NewSummarizer() *Summarizer {
	return &Summarizer{
		strategies: map[string]Strategy{
			ToolReadFile:      &fileReadStrategy{},
			ToolFileSearch:    &searchStrategy{},
			ToolListDirectory: &directoryStrategy{},
			ToolShellCommand:  &commandStrategy{},
		},
		fallback: &defaultStrategy{},
	}
}

// Summarize applies the appropriate strategy based on tool name.
// Empty outputs are returned unchanged.
func (s *Summarizer) Summarize(toolName, output string) string {
	if output == "" {
		return ""
	}

	strategy, ok := s.strategies[toolName]
	if !ok {
		strategy = s.fallback
	}

	return strategy.Summarize(toolName, output)
}

// fileReadStrategy summarizes file read outputs.
type fileReadStrategy struct{}

// Summarize returns a compact line/char count for file content.
func (f *fileReadStrategy) Summarize(_, output string) string {
	lines := strings.Count(output, "\n") + 1
	chars := len(output)

	return fmt.Sprintf("Read file (%d lines, %d chars)", lines, chars)
}

// searchStrategy summarizes file search outputs.
type searchStrategy struct{}

// Summarize returns a match count summary for search results.
func (s *searchStrategy) Summarize(_, output string) string {
	matches := strings.Count(output, "\n") + 1
	if output == "" {
		matches = 0
	}

	return fmt.Sprintf("Search completed (%d matches found)", matches)
}

// directoryStrategy summarizes directory listing outputs.
type directoryStrategy struct{}

// Summarize returns an item count summary for directory listings.
func (d *directoryStrategy) Summarize(_, output string) string {
	items := strings.Count(output, "\n") + 1
	if output == "" {
		items = 0
	}

	return fmt.Sprintf("Listed directory (%d items)", items)
}

// commandStrategy summarizes command execution outputs.
type commandStrategy struct{}

// Summarize returns short output verbatim or a line count summary for long output.
func (c *commandStrategy) Summarize(_, output string) string {
	if len(output) <= ShortOutputMax {
		return output
	}

	lines := strings.Count(output, "\n") + 1

	return fmt.Sprintf("Command executed (%d lines of output)", lines)
}

// defaultStrategy handles unknown tool types.
type defaultStrategy struct{}

// Summarize returns short output verbatim or a line/char count summary for long output.
func (d *defaultStrategy) Summarize(_, output string) string {
	if len(output) <= ShortOutputMax {
		return output
	}

	lines := strings.Count(output, "\n") + 1

	return fmt.Sprintf("Tool output (%d lines, %d chars)", lines, len(output))
}

// errorPrefix is the standard prefix for error tool results.
const errorPrefix = "Error: "

// SummarizeError truncates error output to a maximum length with a classified prefix.
// If the output already starts with "Error: ", it is not double-prefixed.
func SummarizeError(output string) string {
	// Avoid cascading error prefixes when re-summarizing across turns.
	if !strings.HasPrefix(output, errorPrefix) {
		output = errorPrefix + output
	}

	if len(output) <= ErrorTruncateMax {
		return output
	}

	return output[:ErrorTruncateMax] + "..."
}
