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
			if maxLen <= suffixLen {
				lines[idx] = line[:maxLen]
			} else {
				lines[idx] = line[:maxLen-suffixLen] + truncatedSuffix
			}

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
	return ContainsAnyKeyword(input, []string{substr})
}

// TruncateWithSuffix truncates input to maxLen bytes and appends suffix.
// The suffix is added beyond maxLen (total may exceed maxLen).
// Returns input unchanged if len(input) <= maxLen.
func TruncateWithSuffix(input string, maxLen int, suffix string) string {
	if len(input) <= maxLen {
		return input
	}

	return input[:maxLen] + suffix
}

// escapeBackslashPlaceholder is a sentinel used during escape normalization
// to avoid double-replacement of backslash sequences.
const escapeBackslashPlaceholder = "\x00ESCAPE_BACKSLASH\x00"

// NormalizeEscapes replaces literal escape sequences (\\n, \\t, \\", \\\\)
// with their actual characters (\n, \t, ", \).
func NormalizeEscapes(input string) string {
	// Replace \\\\ first with placeholder to avoid double-replacement.
	result := strings.ReplaceAll(input, `\\`, escapeBackslashPlaceholder)
	result = strings.ReplaceAll(result, `\n`, "\n")
	result = strings.ReplaceAll(result, `\t`, "\t")
	result = strings.ReplaceAll(result, `\"`, `"`)

	return strings.ReplaceAll(result, escapeBackslashPlaceholder, `\`)
}

// maskedPlaceholder is returned when a secret is too short to safely reveal.
const maskedPlaceholder = "***"

// minSecretLen is the minimum length a secret must have before any
// characters are revealed; shorter secrets are fully masked.
const minSecretLen = 8

// MaskSecret masks a secret string for display, showing only the first and last
// visibleChars characters with "..." in between. If the string is too short
// (8 characters or fewer), it returns "***" to avoid leaking the secret.
func MaskSecret(key string, visibleChars int) string {
	if len(key) <= minSecretLen {
		return maskedPlaceholder
	}

	if visibleChars <= 0 {
		return maskedPlaceholder
	}

	// Ensure we don't try to show more characters than available.
	if visibleChars*2 >= len(key) {
		return maskedPlaceholder
	}

	return key[:visibleChars] + "..." + key[len(key)-visibleChars:]
}

// ParseKeyValuePairs splits each item in items around the first occurrence of
// sep and returns the resulting key-value map. Items that do not contain sep
// are silently skipped. Returns nil for an empty input slice.
func ParseKeyValuePairs(items []string, sep string) map[string]string {
	if len(items) == 0 {
		return nil
	}

	result := make(map[string]string, len(items))

	for _, item := range items {
		key, value, ok := strings.Cut(item, sep)
		if ok {
			result[key] = value
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// truncationState tracks delimiter and string-literal state during truncation detection.
type truncationState struct {
	opens    int
	inString bool
	escaped  bool
}

// processByte updates state for a single byte during truncation scanning.
func (ts *truncationState) processByte(ch byte) {
	if ts.escaped {
		ts.escaped = false

		return
	}

	if ch == '\\' && ts.inString {
		ts.escaped = true

		return
	}

	if ts.inString {
		if ch == '"' {
			ts.inString = false
		}

		return
	}

	ts.processOutsideString(ch)
}

// processOutsideString handles delimiter counting outside string literals.
func (ts *truncationState) processOutsideString(ch byte) {
	switch ch {
	case '"':
		ts.inString = true
	case '{', '(', '[':
		ts.opens++
	case '}', ')', ']':
		ts.opens--
	}
}

// DetectTruncation checks if content appears to be truncated source code.
// Returns a reason string describing the truncation, or empty if not truncated.
// Detects unclosed string literals and unmatched delimiters ({, (, [).
func DetectTruncation(content string) string {
	if content == "" {
		return ""
	}

	var state truncationState

	for i := range len(content) {
		state.processByte(content[i])
	}

	if state.inString {
		return "unclosed string literal"
	}

	if state.opens > 0 {
		return fmt.Sprintf("%d unclosed delimiter(s)", state.opens)
	}

	return ""
}
