package acp

import (
	"strings"
	"unicode"
)

const (
	// maxTitleLength is the maximum length for a generated session title.
	maxTitleLength = 80
)

// generateSessionTitle extracts a short title from agent response content.
// Returns empty string if content is empty or whitespace-only.
func generateSessionTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Strip markdown heading prefix.
	content = stripMarkdownHeading(content)

	// Find first sentence boundary.
	title := extractFirstSentence(content)

	// Truncate at word boundary if too long.
	if len(title) > maxTitleLength {
		title = truncateAtWord(title, maxTitleLength)
	}

	return title
}

// stripMarkdownHeading removes leading markdown heading characters.
func stripMarkdownHeading(content string) string {
	trimmed := strings.TrimLeft(content, "# ")
	if trimmed == "" {
		return content
	}

	return trimmed
}

// extractFirstSentence returns content up to the first line or sentence boundary.
func extractFirstSentence(content string) string {
	// First, limit to first line.
	if idx := strings.IndexByte(content, '\n'); idx > 0 {
		content = strings.TrimSpace(content[:idx])
	}

	// Then find first sentence boundary within that line.
	for i, ch := range content {
		if !isSentenceEnd(ch) {
			continue
		}

		// Check if next char is space or end of string.
		nextIdx := i + len(string(ch))
		if nextIdx >= len(content) || content[nextIdx] == ' ' {
			return content[:nextIdx]
		}
	}

	return content
}

// isSentenceEnd reports whether ch is a sentence-ending punctuation mark.
func isSentenceEnd(ch rune) bool {
	return ch == '.' || ch == '!' || ch == '?'
}

// truncateAtWord truncates content to maxLen at the nearest word boundary.
func truncateAtWord(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	// Find last space before maxLen.
	truncated := content[:maxLen]
	lastSpace := strings.LastIndexFunc(truncated, unicode.IsSpace)

	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(truncated) + "..."
}
