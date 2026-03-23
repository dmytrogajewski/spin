package stringsx

import (
	"fmt"
	"strings"
)

// truncatedSuffix is appended to lines that exceed the max length.
const truncatedSuffix = "... [truncated]"

// TruncateHeadTail keeps the first head bytes and last tail bytes of s,
// inserting an omission marker in the middle. Returns s unchanged if
// len(s) <= maxTotal.
func TruncateHeadTail(input string, maxTotal, head, tail int) string {
	if len(input) <= maxTotal {
		return input
	}

	omitted := len(input) - head - tail
	marker := fmt.Sprintf("\n... [%d characters omitted] ...\n", omitted)

	var builder strings.Builder

	builder.Grow(head + len(marker) + tail)
	builder.WriteString(input[:head])
	builder.WriteString(marker)
	builder.WriteString(input[len(input)-tail:])

	return builder.String()
}

// TruncateLines truncates any line longer than maxLen, appending a
// truncation suffix. Returns the original string if no lines are truncated.
func TruncateLines(input string, maxLen int) string {
	if input == "" {
		return ""
	}

	suffixLen := len(truncatedSuffix)
	lines := strings.Split(input, "\n")
	changed := false

	for idx, line := range lines {
		if len(line) > maxLen {
			lines[idx] = line[:maxLen-suffixLen] + truncatedSuffix
			changed = true
		}
	}

	if !changed {
		return input
	}

	return strings.Join(lines, "\n")
}

// IsPartialPrefix returns true if s is a strict prefix of any candidate
// (i.e., s is shorter than the candidate and the candidate starts with s).
func IsPartialPrefix(input string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, input) && len(input) < len(candidate) {
			return true
		}
	}

	return false
}

// FindMatchingClose finds the position of the matching closing tag,
// accounting for nesting depth. Starts scanning from startPos.
// Returns -1 if no matching close is found.
func FindMatchingClose(content string, startPos int, openTag, closeTag string) int {
	depth := 1
	pos := startPos

	for pos < len(content) && depth > 0 {
		nextOpen := strings.Index(content[pos:], openTag)
		nextClose := strings.Index(content[pos:], closeTag)

		if nextClose == -1 {
			return -1
		}

		nextClose += pos

		if nextOpen != -1 {
			nextOpen += pos

			if nextOpen < nextClose {
				depth++
				pos = nextOpen + len(openTag)

				continue
			}
		}

		depth--

		if depth == 0 {
			return nextClose
		}

		pos = nextClose + len(closeTag)
	}

	return -1
}

// ContainsIgnoreCase checks if s contains substr, ignoring case.
func ContainsIgnoreCase(input, substr string) bool {
	return strings.Contains(strings.ToLower(input), strings.ToLower(substr))
}
