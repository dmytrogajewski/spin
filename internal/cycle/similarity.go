package cycle

import (
	"strings"
	"unicode"
)

// calculateSimilarity computes the Jaccard similarity between two strings.
// Jaccard similarity is defined as |A ∩ B| / |A ∪ B| where A and B are sets of words.
//
// This is a simple but effective measure for detecting when LLM responses
// are repeating similar content without being identical strings.
func calculateSimilarity(a, b string) float64 {
	wordsA := extractWords(a)
	wordsB := extractWords(b)

	if similarity := handleEmptyCases(wordsA, wordsB); similarity >= 0 {
		return similarity
	}

	setA := createWordSet(wordsA)
	setB := createWordSet(wordsB)

	intersection := calculateIntersection(setA, setB)
	union := calculateUnion(setA, setB, intersection)

	return calculateJaccardSimilarity(intersection, union)
}

// handleEmptyCases handles cases where one or both word sets are empty.
func handleEmptyCases(wordsA, wordsB []string) float64 {
	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0 // Both empty strings are identical.
	}

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0 // One empty, one non-empty are dissimilar.
	}

	return -1 // Not an empty case.
}

// createWordSet creates a set from a slice of words.
func createWordSet(words []string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, word := range words {
		set[word] = true
	}

	return set
}

// calculateIntersection calculates the size of the intersection of two sets.
func calculateIntersection(setA, setB map[string]bool) int {
	intersection := 0

	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	return intersection
}

// calculateUnion calculates the size of the union of two sets.
func calculateUnion(setA, setB map[string]bool, intersection int) int {
	return len(setA) + len(setB) - intersection
}

// calculateJaccardSimilarity calculates the Jaccard similarity coefficient.
func calculateJaccardSimilarity(intersection, union int) float64 {
	if union == 0 {
		return 1.0 // Both sets are empty (shouldn't happen due to earlier check).
	}

	return float64(intersection) / float64(union)
}

// extractWords extracts individual words from a string.
// Words are converted to lowercase and punctuation is removed.
// This provides a normalized word set for similarity comparison.
func extractWords(text string) []string {
	// Split on whitespace and punctuation.
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	})

	words := make([]string, 0, len(fields))
	for _, field := range fields {
		// Convert to lowercase.
		word := strings.ToLower(field)

		// Skip empty words and very short words (likely not meaningful).
		if len(word) > 2 {
			words = append(words, word)
		}
	}

	return words
}
