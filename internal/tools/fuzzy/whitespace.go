package fuzzy

import (
	"regexp"

	"github.com/dmytrogajewski/spin/pkg/alg/search"
)

var whitespaceRunPattern = regexp.MustCompile(`[ \t]+`)

// collapseWhitespace replaces runs of spaces/tabs with a single space.
func collapseWhitespace(str string) string {
	return whitespaceRunPattern.ReplaceAllString(str, " ")
}

// WhitespaceFind collapses whitespace runs and finds matches.
func WhitespaceFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := collapseWhitespace(oldContent)
	normalizedFile := collapseWhitespace(fileContent)

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}

// findByNormalized maps matches found in normalized content back to original content.
func findByNormalized(original, normalizedFile, normalizedOld string) []MatchResult {
	offsets := search.FindAllNormalized(original, normalizedFile, normalizedOld)
	results := make([]MatchResult, 0, len(offsets))

	for _, pair := range offsets {
		results = append(results, MatchResult{
			Start:    pair[0],
			End:      pair[1],
			Original: original[pair[0]:pair[1]],
		})
	}

	return results
}
