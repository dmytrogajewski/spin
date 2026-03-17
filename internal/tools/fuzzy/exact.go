package fuzzy

import "strings"

// ExactFind performs exact string matching.
func ExactFind(fileContent, oldContent string) []MatchResult {
	var results []MatchResult

	searchFrom := 0

	for {
		idx := strings.Index(fileContent[searchFrom:], oldContent)
		if idx < 0 {
			break
		}

		absIdx := searchFrom + idx
		results = append(results, MatchResult{
			Start:    absIdx,
			End:      absIdx + len(oldContent),
			Original: fileContent[absIdx : absIdx+len(oldContent)],
		})

		searchFrom = absIdx + 1
	}

	return results
}
