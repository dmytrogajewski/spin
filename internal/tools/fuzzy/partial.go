package fuzzy

// partialMatchThreshold is the minimum ratio of matched length to old content length.
const partialMatchThreshold = 0.6

// PartialFind finds the longest common substring between oldContent and fileContent.
// Only returns a match if the common substring is at least 60% of oldContent length.
func PartialFind(fileContent, oldContent string) []MatchResult {
	if oldContent == "" || fileContent == "" {
		return nil
	}

	lcsStart, lcsLen := longestCommonSubstring(fileContent, oldContent)
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

// longestCommonSubstring finds the longest common substring.
// Returns start position in first string and length.
func longestCommonSubstring(str1, str2 string) (int, int) {
	if str1 == "" || str2 == "" {
		return 0, 0
	}

	// Use rolling row to save memory: O(min(m,n)) space.
	prev := make([]int, len(str2)+1)
	curr := make([]int, len(str2)+1)

	maxLen := 0
	maxEnd := 0

	for idx1 := 1; idx1 <= len(str1); idx1++ {
		for idx2 := 1; idx2 <= len(str2); idx2++ {
			if str1[idx1-1] == str2[idx2-1] {
				curr[idx2] = prev[idx2-1] + 1
				if curr[idx2] > maxLen {
					maxLen = curr[idx2]
					maxEnd = idx1
				}
			} else {
				curr[idx2] = 0
			}
		}

		prev, curr = curr, prev
		// Clear curr for next iteration.
		clear(curr)
	}

	return maxEnd - maxLen, maxLen
}
