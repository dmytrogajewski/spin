package fuzzy

import "github.com/dmytrogajewski/spin/pkg/alg/similarity"

// partialMatchThreshold is the minimum ratio of matched length to old content length.
const partialMatchThreshold = 0.6

// PartialFind finds the longest common substring between oldContent and fileContent.
// Only returns a match if the common substring is at least 60% of oldContent length.
func PartialFind(fileContent, oldContent string) []MatchResult {
	if oldContent == "" || fileContent == "" {
		return nil
	}

	lcsStart, lcsLen := similarity.LongestCommonSubstring(fileContent, oldContent)
	if lcsLen == 0 {
		return nil
	}

	ratio := float64(lcsLen) / float64(len(oldContent))
	if ratio < partialMatchThreshold {
		return nil
	}

	return []MatchResult{
		{
			Start:    lcsStart,
			End:      lcsStart + lcsLen,
			Original: fileContent[lcsStart : lcsStart+lcsLen],
		},
	}
}
