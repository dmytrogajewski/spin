package similarity

// LongestCommonSubstring finds the longest common substring between two strings.
// Returns the start position in the first string and the length of the match.
// Returns (0, 0) if either string is empty or there is no common substring.
// Uses O(min(m,n)) space via rolling rows.
func LongestCommonSubstring(first, second string) (start, length int) {
	if first == "" || second == "" {
		return 0, 0
	}

	prev := make([]int, len(second)+1)
	curr := make([]int, len(second)+1)

	maxLen := 0
	maxEnd := 0

	for idx1 := 1; idx1 <= len(first); idx1++ {
		for idx2 := 1; idx2 <= len(second); idx2++ {
			if first[idx1-1] == second[idx2-1] {
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

		clear(curr)
	}

	return maxEnd - maxLen, maxLen
}
