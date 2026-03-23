// Package similarity provides text similarity algorithms including
// Levenshtein edit distance, Jaccard word-set similarity, and n-gram extraction.
package similarity

// Levenshtein computes the edit distance between two strings.
// Returns the minimum number of single-character edits (insertions,
// deletions, substitutions) required to transform a into b.
func Levenshtein(a, b string) int {
	if a == "" {
		return len(b)
	}

	if b == "" {
		return len(a)
	}

	aLen := len(a)
	bLen := len(b)

	// Use a single row for space optimization.
	prev := make([]int, bLen+1)

	for col := range bLen + 1 {
		prev[col] = col
	}

	for row := 1; row <= aLen; row++ {
		curr := make([]int, bLen+1)
		curr[0] = row

		for col := 1; col <= bLen; col++ {
			cost := 1
			if a[row-1] == b[col-1] {
				cost = 0
			}

			deletion := prev[col] + 1
			insertion := curr[col-1] + 1
			substitution := prev[col-1] + cost

			curr[col] = min(deletion, min(insertion, substitution))
		}

		prev = curr
	}

	return prev[bLen]
}
