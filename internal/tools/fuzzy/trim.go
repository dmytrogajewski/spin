package fuzzy

import "strings"

// TrimFind trims trailing whitespace from each line and compares.
func TrimFind(fileContent, oldContent string) []MatchResult {
	return findByLineTransform(fileContent, oldContent, func(line string) string {
		return strings.TrimRight(line, " \t")
	})
}
