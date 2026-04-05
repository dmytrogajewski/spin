package fuzzy

import "github.com/dmytrogajewski/spin/pkg/alg/stringsx"

// EscapeFind normalizes escape sequences and finds matches.
func EscapeFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := stringsx.NormalizeEscapes(oldContent)

	// Try finding normalized old in original file content.
	matches := ExactFind(fileContent, normalizedOld)
	if len(matches) > 0 {
		return matches
	}

	// Also try normalizing file content.
	normalizedFile := stringsx.NormalizeEscapes(fileContent)

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}
