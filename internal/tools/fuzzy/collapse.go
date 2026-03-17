package fuzzy

import (
	"regexp"
	"strings"
)

var consecutiveBlankLines = regexp.MustCompile(`\n{3,}`)

const doubleNewline = "\n\n"

// collapseBlankLines replaces consecutive blank lines with a single blank line.
func collapseBlankLines(str string) string {
	return consecutiveBlankLines.ReplaceAllString(str, doubleNewline)
}

// CollapseFind collapses consecutive blank lines and finds matches.
func CollapseFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := collapseBlankLines(oldContent)
	normalizedFile := collapseBlankLines(fileContent)

	matches := findByNormalized(fileContent, normalizedFile, normalizedOld)
	if len(matches) > 0 {
		return matches
	}

	// Also try with stripped trailing whitespace per line + collapsed blanks.
	normalizedOld = collapseBlankLines(trimLines(oldContent))
	normalizedFile = collapseBlankLines(trimLines(fileContent))

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}

// trimLines trims trailing whitespace from each line.
func trimLines(str string) string {
	lines := strings.Split(str, "\n")
	for idx, line := range lines {
		lines[idx] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}
