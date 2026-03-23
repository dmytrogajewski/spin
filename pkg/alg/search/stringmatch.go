package search

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/similarity"
)

// String scoring constants for multi-strategy matching.
const (
	scoreExact        = 1.0
	scorePrefix       = 0.9
	scoreContains     = 0.7
	scoreWordBoundary = 0.75
	fuzzyThreshold    = 0.5
	fuzzyWeight       = 0.6
)

// ScoreString calculates a relevance score for a string given a query.
// Tries strategies in priority order: exact → prefix → contains → word-boundary → fuzzy.
// Returns 0.0 if no strategy matches. Fuzzy matching uses Levenshtein distance
// and is only attempted when fuzzy is true.
func ScoreString(input, query string, fuzzy bool) float64 {
	if input == query {
		return scoreExact
	}

	if strings.HasPrefix(input, query) {
		return scorePrefix
	}

	if strings.Contains(input, query) {
		return scoreContains
	}

	// Word boundary match (query matches start of a word).
	words := strings.FieldsFunc(input, isWordSeparator)
	for _, word := range words {
		if strings.HasPrefix(word, query) {
			return scoreWordBoundary
		}
	}

	if fuzzy {
		return scoreFuzzy(input, query)
	}

	return 0.0
}

// isWordSeparator returns true for characters that separate words.
func isWordSeparator(char rune) bool {
	return char == '_' || char == '-' || char == ' ' || char == '.'
}

// scoreFuzzy computes a Levenshtein-based fuzzy score.
func scoreFuzzy(input, query string) float64 {
	distance := similarity.Levenshtein(input, query)

	maxLen := max(len(input), len(query))
	if maxLen == 0 {
		return 0.0
	}

	score := 1.0 - float64(distance)/float64(maxLen)
	if score >= fuzzyThreshold {
		return score * fuzzyWeight
	}

	return 0.0
}
