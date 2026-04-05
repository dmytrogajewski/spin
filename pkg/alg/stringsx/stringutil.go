// Package stringsx provides zero-dependency text utilities that consolidate
// duplicated string operations across the codebase.
package stringsx

import (
	"regexp"
	"strings"
)

// CharsPerToken is a rough estimate of how many characters correspond to one
// LLM token. Used across the codebase for quick token-count approximations.
const CharsPerToken = 4

// ellipsis is the suffix appended when truncating.
const ellipsis = "..."

// ellipsisLen is the byte length of the ellipsis suffix.
const ellipsisLen = len(ellipsis)

// consecutiveBlankLines matches 3+ consecutive newlines.
var consecutiveBlankLines = regexp.MustCompile(`\n{3,}`)

// doubleNewline is the replacement for collapsed blank lines.
const doubleNewline = "\n\n"

// backtickFence is the marker for code fences.
const backtickFence = "```"

// minListPrefixLen is the minimum line length to check for list prefixes.
const minListPrefixLen = 2

// CollapseWhitespace replaces all runs of whitespace (spaces, tabs, newlines)
// with a single space and trims leading/trailing whitespace.
func CollapseWhitespace(input string) string {
	return strings.Join(strings.Fields(input), " ")
}

// CollapseBlankLines replaces 3+ consecutive newlines with exactly two newlines.
// Single and double newlines are preserved.
func CollapseBlankLines(input string) string {
	return consecutiveBlankLines.ReplaceAllString(input, doubleNewline)
}

// TrimTrailingPerLine trims trailing spaces and tabs from each line.
// Leading whitespace is preserved.
func TrimTrailingPerLine(input string) string {
	lines := strings.Split(input, "\n")

	for idx, line := range lines {
		lines[idx] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// TruncateWithEllipsis truncates the string to maxLen bytes.
// If the string exceeds maxLen and maxLen >= ellipsisLen, the result
// ends with "...". If maxLen < ellipsisLen, it simply slices the string.
// Returns the original string if it fits within maxLen.
func TruncateWithEllipsis(input string, maxLen int) string {
	if len(input) <= maxLen {
		return input
	}

	if maxLen <= 0 {
		return ""
	}

	if maxLen < ellipsisLen {
		return input[:maxLen]
	}

	return input[:maxLen-ellipsisLen] + ellipsis
}

// ContainsAnyKeyword returns true if any keyword is found as a
// case-insensitive substring in the input.
func ContainsAnyKeyword(input string, keywords []string) bool {
	lower := strings.ToLower(input)

	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// CountLines splits input on newlines and counts lines where the predicate
// returns true. A nil predicate counts all lines.
func CountLines(input string, predicate func(string) bool) int {
	if input == "" {
		return 0
	}

	lines := strings.Split(input, "\n")
	count := 0

	for _, line := range lines {
		if predicate == nil || predicate(line) {
			count++
		}
	}

	return count
}

// StripCodeFence removes markdown code fences (``` delimiters) from the input.
// Returns the inner content and the detected language tag (empty if none).
func StripCodeFence(input string) (content, lang string) {
	trimmed := strings.TrimSpace(input)

	// Try to strip opening fence with language tag.
	trimmed, lang = stripOpeningFence(trimmed)

	// Strip closing fence.
	if after, ok := strings.CutSuffix(trimmed, backtickFence); ok {
		trimmed = strings.TrimSpace(after)
	}

	return trimmed, lang
}

// stripOpeningFence removes a leading ``` fence and extracts the language tag.
func stripOpeningFence(input string) (content, lang string) {
	if !strings.HasPrefix(input, backtickFence) {
		return input, ""
	}

	// Extract remainder after opening backticks.
	rest := input[len(backtickFence):]

	firstLine, body, hasNewline := strings.Cut(rest, "\n")
	if !hasNewline {
		// No newline — the whole remainder is the language tag.
		return "", strings.TrimSpace(firstLine)
	}

	return strings.TrimSpace(body), strings.TrimSpace(firstLine)
}

// StripListPrefix removes common list prefixes from a line.
// Recognized prefixes: "- ", "* ", single-digit "N.", single-digit "N)".
// Returns the original line if no prefix is found.
func StripListPrefix(line string) string {
	if len(line) <= minListPrefixLen {
		return line
	}

	// Numbered items: "1. content" or "1) content".
	if line[0] >= '0' && line[0] <= '9' && (line[1] == '.' || line[1] == ')') {
		return strings.TrimSpace(line[minListPrefixLen:])
	}

	// Bullet items: "- content" or "* content".
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return strings.TrimSpace(line[minListPrefixLen:])
	}

	return line
}
