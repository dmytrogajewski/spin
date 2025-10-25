package output

import (
	"fmt"
	"strings"
	"time"
)

// ThinkFilter processes streaming content to detect and format <think> blocks.
// It strips XML tags and applies dim gray formatting to thinking content in real-time.
type ThinkFilter struct {
	inThink     bool            // Currently inside <think> block
	thinkStart  time.Time       // When current think block started
	thinkTokens int             // Token count in current think block
	tagBuffer   strings.Builder // Buffer for detecting partial tags across chunks
}

// NewThinkFilter creates a new think block filter.
func NewThinkFilter() *ThinkFilter {
	return &ThinkFilter{}
}

// Process filters a content chunk, handling <think> tags.
// Returns the formatted output to display, or empty string if content is buffered.
func (f *ThinkFilter) Process(chunk string) string {
	// Prepend any buffered tag fragments from previous chunk
	if f.tagBuffer.Len() > 0 {
		chunk = f.tagBuffer.String() + chunk
		f.tagBuffer.Reset()
	}

	var out strings.Builder
	i := 0

	for i < len(chunk) {
		// Check for <think> or <think> tag
		if !f.inThink {
			if strings.HasPrefix(chunk[i:], "<think>") {
				f.inThink = true
				f.thinkStart = time.Now()
				f.thinkTokens = 0
				out.WriteString("\x1b[2m")        // Dim
				out.WriteString("\x1b[38;5;244m") // Gray
				i += len("<think>")
				continue
			}
			if strings.HasPrefix(chunk[i:], "<think>") {
				f.inThink = true
				f.thinkStart = time.Now()
				f.thinkTokens = 0
				out.WriteString("\x1b[2m")        // Dim
				out.WriteString("\x1b[38;5;244m") // Gray
				i += len("<think>")
				continue
			}
		}

		// Check for </think> or </think> tag
		if f.inThink {
			if strings.HasPrefix(chunk[i:], "</think>") {
				f.inThink = false
				duration := time.Since(f.thinkStart)
				out.WriteString("\x1b[0m") // Reset dim gray
				out.WriteString("\x1b[2m\x1b[38;5;242m")
				out.WriteString(fmt.Sprintf(" [thought for %.2fs, ~%d tokens]",
					duration.Seconds(), f.thinkTokens))
				out.WriteString("\x1b[0m\n")
				i += len("</think>")
				continue
			}
			if strings.HasPrefix(chunk[i:], "</think>") {
				f.inThink = false
				duration := time.Since(f.thinkStart)
				out.WriteString("\x1b[0m") // Reset dim gray
				out.WriteString("\x1b[2m\x1b[38;5;242m")
				out.WriteString(fmt.Sprintf(" [thought for %.2fs, ~%d tokens]",
					duration.Seconds(), f.thinkTokens))
				out.WriteString("\x1b[0m\n")
				i += len("</think>")
				continue
			}
		}

		// Check if we're at end of chunk and might have a partial tag
		remaining := chunk[i:]
		if !f.inThink {
			// Check if remaining starts with incomplete opening tag
			if len(remaining) > 0 && remaining[0] == '<' && !strings.HasPrefix(remaining, "<think>") && !strings.HasPrefix(remaining, "<think>") {
				f.tagBuffer.WriteString(remaining)
				break
			}
		}
		if f.inThink {
			// Check if remaining starts with incomplete closing tag
			if len(remaining) > 0 && remaining[0] == '<' && !strings.HasPrefix(remaining, "</think>") && !strings.HasPrefix(remaining, "</think>") {
				f.tagBuffer.WriteString(remaining)
				break
			}
		}

		// Inside think block: stream content immediately in dim gray
		if f.inThink {
			char := chunk[i]
			out.WriteByte(char) // Stream in real-time!
			// Rough token estimation: whitespace-delimited words
			if char == ' ' || char == '\n' || char == '\t' {
				f.thinkTokens++
			}
			i++
			continue
		}

		// Outside think block: pass through as-is
		out.WriteByte(chunk[i])
		i++
	}

	return out.String()
}

// Flush returns formatting reset and summary if still in a think block (for stream end).
// Returns empty string if not in a think block.
func (f *ThinkFilter) Flush() string {
	if !f.inThink {
		return ""
	}

	var out strings.Builder
	duration := time.Since(f.thinkStart)

	// Reset dim gray formatting
	out.WriteString("\x1b[0m") // Reset

	// Add incomplete thinking summary
	out.WriteString("\x1b[2m\x1b[38;5;242m")
	out.WriteString(fmt.Sprintf(" [thought for %.2fs, ~%d tokens, incomplete]",
		duration.Seconds(), f.thinkTokens))
	out.WriteString("\x1b[0m\n")

	f.inThink = false

	return out.String()
}

// Reset clears the filter state.
func (f *ThinkFilter) Reset() {
	f.inThink = false
	f.thinkStart = time.Time{}
	f.thinkTokens = 0
	f.tagBuffer.Reset()
}
