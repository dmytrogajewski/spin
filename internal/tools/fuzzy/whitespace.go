package fuzzy

import (
	"regexp"
	"strings"
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
	var results []MatchResult

	searchFrom := 0

	for {
		idx := strings.Index(normalizedFile[searchFrom:], normalizedOld)
		if idx < 0 {
			break
		}

		absIdx := searchFrom + idx

		// Map normalized offsets back to original offsets.
		origStart := mapNormalizedOffset(original, normalizedFile, absIdx)
		origEnd := mapNormalizedOffset(original, normalizedFile, absIdx+len(normalizedOld))

		results = append(results, MatchResult{
			Start:    origStart,
			End:      origEnd,
			Original: original[origStart:origEnd],
		})

		searchFrom = absIdx + 1
	}

	return results
}

// mapNormalizedOffset maps an offset in normalized string back to the original string.
func mapNormalizedOffset(original, normalized string, normOffset int) int {
	origIdx := 0
	normIdx := 0

	for normIdx < normOffset && origIdx < len(original) {
		if normIdx < len(normalized) && original[origIdx] == normalized[normIdx] {
			origIdx++
			normIdx++
		} else {
			// Original has extra whitespace that was collapsed.
			origIdx++
		}
	}

	return origIdx
}
