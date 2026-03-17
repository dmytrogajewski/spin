package tools

import (
	"fmt"
	"strings"
)

// Output truncation constants.
const (
	// MaxOutputChars is the maximum total characters in truncated output.
	MaxOutputChars = 30_000

	// HeadChars is the number of characters preserved from the start.
	HeadChars = 10_000

	// TailChars is the number of characters preserved from the end.
	TailChars = 10_000

	// MaxLineChars is the maximum length of a single line.
	MaxLineChars = 2_000

	// TruncatedSuffix is appended to lines that exceed MaxLineChars.
	TruncatedSuffix = "... [truncated]"
)

// TruncateHeadTail truncates s to at most maxTotal characters,
// preserving the first head and last tail characters with an
// omission marker between them.
func TruncateHeadTail(s string, maxTotal, head, tail int) string {
	if len(s) <= maxTotal {
		return s
	}

	omitted := len(s) - head - tail
	marker := fmt.Sprintf("\n... [%d characters omitted] ...\n", omitted)

	var sb strings.Builder

	sb.Grow(head + len(marker) + tail)
	sb.WriteString(s[:head])
	sb.WriteString(marker)
	sb.WriteString(s[len(s)-tail:])

	return sb.String()
}

// TruncateLines truncates individual lines exceeding maxLen.
func TruncateLines(s string, maxLen int) string {
	if s == "" {
		return ""
	}

	suffixLen := len(TruncatedSuffix)
	lines := strings.Split(s, "\n")
	changed := false

	for idx, line := range lines {
		if len(line) > maxLen {
			lines[idx] = line[:maxLen-suffixLen] + TruncatedSuffix
			changed = true
		}
	}

	if !changed {
		return s
	}

	return strings.Join(lines, "\n")
}

// TruncateOutput applies both line truncation and head-tail truncation
// using the default constants.
func TruncateOutput(s string) string {
	s = TruncateLines(s, MaxLineChars)

	return TruncateHeadTail(s, MaxOutputChars, HeadChars, TailChars)
}
