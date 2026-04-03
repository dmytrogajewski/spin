package similarity

import (
	"strings"
	"unicode"

	"github.com/dmytrogajewski/spin/pkg/alg/collections"
)

// minWordLength is the minimum character count for a word to be included
// in the Jaccard word-set comparison.
const minWordLength = 2

// JaccardSimilarity computes the Jaccard similarity between two strings
// based on their word sets. Words shorter than minWordLength are filtered.
// Returns 1.0 for two empty strings, 0.0 if only one is empty,
// and the Jaccard index (|A intersect B| / |A union B|) otherwise.
func JaccardSimilarity(a, b string) float64 {
	wordsA := ExtractWords(a)
	wordsB := ExtractWords(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0
	}

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	setA := collections.ToSet[string](wordsA)
	setB := collections.ToSet[string](wordsB)

	intersection := countIntersection(setA, setB)
	union := len(setA) + len(setB) - intersection

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// ExtractWords splits text into lowercase words, filtering out short tokens.
// Words shorter than minWordLength characters are excluded.
func ExtractWords(text string) []string {
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})

	var result []string

	for _, token := range tokens {
		word := strings.ToLower(token)
		if len(word) > minWordLength {
			result = append(result, word)
		}
	}

	return result
}

// countIntersection counts elements present in both sets.
func countIntersection(setA, setB map[string]bool) int {
	count := 0

	for word := range setA {
		if setB[word] {
			count++
		}
	}

	return count
}
