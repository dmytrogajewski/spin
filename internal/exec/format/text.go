// Package format provides output formatters for exec mode.
package format

import (
	"fmt"
	"os"
	"strings"
)

// TextFormatter formats output as human-readable text.
type TextFormatter struct {
	noColor bool
}

// NewTextFormatter creates a new text formatter.
// Respects NO_COLOR environment variable.
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		noColor: os.Getenv("NO_COLOR") != "",
	}
}

// FormatStart formats the initial message.
func (f *TextFormatter) FormatStart(prompt string) string {
	if prompt == "" {
		return "[Spin] Starting task\n"
	}
	return fmt.Sprintf("[Spin] Starting task: %s\n", prompt)
}

// FormatDelta formats a streaming chunk.
func (f *TextFormatter) FormatDelta(delta string) string {
	if delta == "" {
		return ""
	}

	// If delta contains newlines, only prefix the first line
	if strings.Contains(delta, "\n") {
		lines := strings.Split(delta, "\n")
		lines[0] = "[Spin] " + lines[0]
		return strings.Join(lines, "\n") + "\n"
	}

	return fmt.Sprintf("[Spin] %s\n", delta)
}

// FormatComplete formats the completion message.
func (f *TextFormatter) FormatComplete(result *ExecResult) string {
	var b strings.Builder

	// Status line
	b.WriteString(f.formatStatusLine(result.Status))

	// Summary
	b.WriteString("\nSummary:\n")
	b.WriteString(fmt.Sprintf("  Status: %s\n", result.Status))

	// Duration
	if result.Duration > 0 {
		b.WriteString(fmt.Sprintf("  Duration: %s\n", result.Duration))
	}

	// Tokens
	if result.TokensUsed > 0 {
		b.WriteString(fmt.Sprintf("  Tokens: %s\n", formatNumber(result.TokensUsed)))
	}

	// Error
	if result.Error != nil {
		b.WriteString(fmt.Sprintf("  Error: %v\n", result.Error))
	}

	// Files modified
	if len(result.FilesModified) > 0 {
		b.WriteString(fmt.Sprintf("  Files modified: %d\n", len(result.FilesModified)))
		for _, file := range result.FilesModified {
			b.WriteString(fmt.Sprintf("    - %s\n", file))
		}
	}

	// Commands executed
	if len(result.CommandsRun) > 0 {
		b.WriteString(fmt.Sprintf("  Commands executed: %d\n", len(result.CommandsRun)))
		for _, cmd := range result.CommandsRun {
			b.WriteString(fmt.Sprintf("    - %s (exit %d)\n", cmd.Command, cmd.ExitCode))
		}
	}

	return b.String()
}

// FormatError formats an error message.
func (f *TextFormatter) FormatError(err error) string {
	return fmt.Sprintf("[Spin] Error: %v\n", err)
}

// formatStatusLine creates the status line based on result status
func (f *TextFormatter) formatStatusLine(status string) string {
	switch status {
	case "complete":
		return "[Spin] Task complete\n"
	case "failed":
		return "[Spin] Task failed\n"
	case "timeout":
		return "[Spin] Task timed out\n"
	case "cancelled":
		return "[Spin] Task cancelled\n"
	default:
		return fmt.Sprintf("[Spin] Task %s\n", status)
	}
}

// formatNumber formats large numbers with thousands separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Convert to string
	s := fmt.Sprintf("%d", n)

	// Add commas from right to left
	var result []rune
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, digit)
	}

	return string(result)
}
