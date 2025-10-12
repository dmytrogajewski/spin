package strutil

import (
	"strings"
	"unicode"
)

// SplitLines splits text into lines, handling different line endings (CRLF, LF, CR).
// All line endings are normalized to LF before splitting.
//
// Example:
//
//	lines := strutil.SplitLines("line1\r\nline2\nline3\r")
//	// Returns: ["line1", "line2", "line3"]
func SplitLines(text string) []string {
	// Normalize line endings: CRLF → LF, CR → LF
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

// JoinLines joins a slice of lines into a single string with LF line endings.
//
// Example:
//
//	text := strutil.JoinLines([]string{"line1", "line2", "line3"})
//	// Returns: "line1\nline2\nline3"
func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// TrimEmptyLines removes leading and trailing empty lines from a slice.
// Empty lines in the middle are preserved. A line is considered empty only
// if it's exactly an empty string, not if it contains only whitespace.
//
// Example:
//
//	lines := strutil.TrimEmptyLines([]string{"", "", "a", "", "b", "", ""})
//	// Returns: ["a", "", "b"]
func TrimEmptyLines(lines []string) []string {
	if len(lines) == 0 {
		return []string{}
	}

	// Find first non-empty line (empty means exactly "")
	start := 0
	for start < len(lines) && lines[start] == "" {
		start++
	}

	// Find last non-empty line
	end := len(lines)
	for end > start && lines[end-1] == "" {
		end--
	}

	if start >= end {
		return []string{}
	}

	return lines[start:end]
}

// DetectIndentation analyzes text to determine indentation style and size.
// Returns (useTabs, size) where useTabs indicates tab vs space preference,
// and size is the indentation width (1 for tabs, typically 2 or 4 for spaces).
//
// The function samples the first 100 lines and returns the most common pattern.
// Defaults to (false, 4) if no clear pattern is found.
//
// Example:
//
//	useTabs, size := strutil.DetectIndentation("    line1\n    line2\n")
//	// Returns: (false, 4)
func DetectIndentation(text string) (useTabs bool, size int) {
	lines := SplitLines(text)

	tabCount := 0
	spaceCount := 0
	spaceSizes := make(map[int]int)

	// Sample first 100 lines
	maxLines := len(lines)
	if maxLines > 100 {
		maxLines = 100
	}

	for i := 0; i < maxLines; i++ {
		line := lines[i]
		if len(line) == 0 {
			continue
		}

		// Count leading whitespace
		indent := 0
		foundTab := false

		for j, ch := range line {
			if ch == '\t' {
				tabCount++
				foundTab = true
				break
			} else if ch == ' ' {
				indent++
			} else {
				// Non-whitespace character reached
				if indent > 0 {
					spaceCount++
					spaceSizes[indent]++
				}
				break
			}

			// Don't check too far into the line
			if j > 20 {
				break
			}
		}

		if foundTab {
			continue
		}
	}

	// Determine if tabs or spaces
	if tabCount > spaceCount {
		return true, 1
	}

	// Find most common space indentation size
	maxCount := 0
	commonSize := 4 // Default to 4 spaces

	for size, count := range spaceSizes {
		if count > maxCount {
			maxCount = count
			commonSize = size
		}
	}

	return false, commonSize
}

// NormalizeIndentation converts indentation in text to the specified style.
// If useTabs is true, converts spaces to tabs. Otherwise, converts tabs to spaces.
// The size parameter specifies the number of spaces per indent level.
//
// Example:
//
//	text := strutil.NormalizeIndentation("\tline1\n\t\tline2\n", false, 4)
//	// Returns: "    line1\n        line2\n"
func NormalizeIndentation(text string, useTabs bool, size int) string {
	if text == "" {
		return ""
	}

	lines := SplitLines(text)
	result := make([]string, len(lines))

	for i, line := range lines {
		if len(line) == 0 {
			result[i] = line
			continue
		}

		// Count leading whitespace
		leadingTabs := 0
		leadingSpaces := 0
		firstNonSpace := 0

		for j, ch := range line {
			if ch == '\t' {
				leadingTabs++
			} else if ch == ' ' {
				leadingSpaces++
			} else {
				firstNonSpace = j
				break
			}
		}

		// Calculate indent level
		indentLevel := leadingTabs
		if leadingSpaces > 0 {
			indentLevel += leadingSpaces / size
		}

		// Build new indentation
		var newIndent string
		if useTabs {
			newIndent = strings.Repeat("\t", indentLevel)
		} else {
			newIndent = strings.Repeat(" ", indentLevel*size)
		}

		// Reconstruct line
		if firstNonSpace > 0 {
			result[i] = newIndent + line[firstNonSpace:]
		} else {
			result[i] = line
		}
	}

	return JoinLines(result)
}

// NormalizeWhitespace replaces all whitespace sequences with single spaces
// and trims leading/trailing whitespace. Useful for fuzzy matching.
//
// Example:
//
//	text := strutil.NormalizeWhitespace("  a  b   c  ")
//	// Returns: "a b c"
func NormalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// TrimWhitespace removes leading and trailing whitespace from text.
// This is a convenience wrapper around strings.TrimSpace.
//
// Example:
//
//	text := strutil.TrimWhitespace("  hello  ")
//	// Returns: "hello"
func TrimWhitespace(text string) string {
	return strings.TrimSpace(text)
}

// LevenshteinDistance calculates the edit distance between two strings.
// It uses the Wagner-Fischer dynamic programming algorithm with space optimization.
//
// The edit distance is the minimum number of single-character edits (insertions,
// deletions, or substitutions) required to change one string into another.
//
// Time complexity: O(n*m) where n and m are the lengths of the strings.
// Space complexity: O(min(n,m)) with optimization.
//
// Example:
//
//	dist := strutil.LevenshteinDistance("kitten", "sitting")
//	// Returns: 3
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Ensure a is the shorter string for space optimization
	if len(a) > len(b) {
		a, b = b, a
	}

	// Use single array instead of full matrix
	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)

	// Initialize first row
	for i := 0; i <= len(a); i++ {
		prev[i] = i
	}

	// Fill matrix row by row
	for j := 1; j <= len(b); j++ {
		curr[0] = j

		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			curr[i] = min3(
				curr[i-1]+1,      // insertion
				prev[i]+1,        // deletion
				prev[i-1]+cost,   // substitution
			)
		}

		// Swap arrays
		prev, curr = curr, prev
	}

	return prev[len(a)]
}

// Similarity calculates a similarity ratio between two strings based on
// Levenshtein distance. Returns a value between 0.0 (completely different)
// and 1.0 (identical).
//
// Formula: 1.0 - (distance / max(len(a), len(b)))
//
// Example:
//
//	similarity := strutil.Similarity("kitten", "sitting")
//	// Returns: 0.571 (approximately 57% similar)
func Similarity(a, b string) float64 {
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}

	distance := LevenshteinDistance(a, b)
	return 1.0 - float64(distance)/float64(maxLen)
}

// FuzzyMatch calculates a fuzzy match score between a query and target string.
// Returns a score between 0.0 (no match) and 100.0 (exact match).
//
// Scoring algorithm:
//   - Exact match: 100.0
//   - Starts with query: 90.0
//   - Contains query: 80.0 - (position/10)
//   - Fuzzy consecutive match: 40.0+
//
// Matching is case-insensitive.
//
// Example:
//
//	score := strutil.FuzzyMatch("abc", "alphabet")
//	// Returns: ~80.0 (contains "abc")
func FuzzyMatch(query, target string) float64 {
	if len(query) == 0 {
		return 0.0
	}

	queryLower := strings.ToLower(query)
	targetLower := strings.ToLower(target)

	// Exact match
	if queryLower == targetLower {
		return 100.0
	}

	// Starts with query
	if strings.HasPrefix(targetLower, queryLower) {
		return 90.0
	}

	// Contains query
	if idx := strings.Index(targetLower, queryLower); idx >= 0 {
		// Earlier in string is better
		return 80.0 - float64(idx)/10.0
	}

	// Fuzzy match (consecutive characters)
	matches := 0
	targetIdx := 0

	for _, ch := range queryLower {
		found := false
		for targetIdx < len(targetLower) {
			if rune(targetLower[targetIdx]) == ch {
				matches++
				found = true
				targetIdx++
				break
			}
			targetIdx++
		}
		if !found {
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Score based on match percentage
	matchRatio := float64(matches) / float64(len(query))
	return 40.0 + (matchRatio * 30.0)
}

// ToSnakeCase converts a string to snake_case.
// Handles PascalCase, camelCase, and mixed formats.
// Acronyms are handled by inserting underscores between consecutive uppercase letters
// followed by a lowercase letter (e.g., HTTPServer → http_server).
//
// Example:
//
//	snake := strutil.ToSnakeCase("MyVariableName")
//	// Returns: "my_variable_name"
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + len(s)/2) // Pre-allocate with some extra space

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		if unicode.IsUpper(ch) {
			// Add underscore before uppercase (except first char)
			if i > 0 {
				prevRune := runes[i-1]
				// Add underscore if:
				// 1. Previous char was not uppercase and not underscore
				// 2. OR this is end of acronym (prev uppercase, next lowercase)
				needsUnderscore := !unicode.IsUpper(prevRune) && prevRune != '_'

				// Handle acronyms: HTTP in HTTPServer
				if !needsUnderscore && unicode.IsUpper(prevRune) && i+1 < len(runes) {
					nextRune := runes[i+1]
					if unicode.IsLower(nextRune) {
						needsUnderscore = true
					}
				}

				if needsUnderscore {
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(ch))
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// ToCamelCase converts a string to camelCase.
// Handles snake_case, PascalCase, and mixed formats.
//
// Example:
//
//	camel := strutil.ToCamelCase("my_variable_name")
//	// Returns: "myVariableName"
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}

	// Split on underscores
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		// No underscores, just lowercase first char
		return lowercaseFirst(s)
	}

	var result strings.Builder
	result.Grow(len(s))

	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			result.WriteString(lowercaseFirst(part))
		} else {
			result.WriteString(uppercaseFirst(part))
		}
	}

	return result.String()
}

// ToPascalCase converts a string to PascalCase.
// Handles snake_case, camelCase, and mixed formats.
//
// Example:
//
//	pascal := strutil.ToPascalCase("my_variable_name")
//	// Returns: "MyVariableName"
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Split on underscores
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		// No underscores, just uppercase first char
		return uppercaseFirst(s)
	}

	var result strings.Builder
	result.Grow(len(s))

	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(uppercaseFirst(part))
	}

	return result.String()
}

// Helper functions

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// lowercaseFirst converts the first character of a string to lowercase
func lowercaseFirst(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// uppercaseFirst converts the first character of a string to uppercase
func uppercaseFirst(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
