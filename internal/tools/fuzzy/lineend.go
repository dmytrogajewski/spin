package fuzzy

import "strings"

// LineEndFind normalizes CRLF to LF and finds matches.
func LineEndFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := strings.ReplaceAll(oldContent, "\r\n", "\n")
	normalizedFile := strings.ReplaceAll(fileContent, "\r\n", "\n")

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}
