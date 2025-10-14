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
	// Extract words from both strings (case-insensitive, ignore punctuation)
	wordsA := extractWords(a)
	wordsB := extractWords(b)

	if len(wordsA) == 0 && len(wordsB) == 0 {
		return 1.0 // Both empty strings are identical
	}

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0 // One empty, one non-empty are dissimilar
	}

	// Create sets for intersection and union calculation
	setA := make(map[string]bool, len(wordsA))
	for _, word := range wordsA {
		setA[word] = true
	}

	setB := make(map[string]bool, len(wordsB))
	for _, word := range wordsB {
		setB[word] = true
	}

	// Calculate intersection size
	intersection := 0
	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	// Calculate union size
	union := len(setA) + len(setB) - intersection

	if union == 0 {
		return 1.0 // Both sets are empty (shouldn't happen due to earlier check)
	}

	return float64(intersection) / float64(union)
}

// extractWords extracts individual words from a string.
// Words are converted to lowercase and punctuation is removed.
// This provides a normalized word set for similarity comparison.
func extractWords(text string) []string {
	// Simple word extraction: split on whitespace and filter
	fields := strings.Fields(text)

	words := make([]string, 0, len(fields))
	for _, field := range fields {
		// Convert to lowercase and remove punctuation
		word := strings.ToLower(removePunctuation(field))

		// Skip empty words and very short words (likely not meaningful)
		if len(word) > 2 {
			words = append(words, word)
		}
	}

	return words
}

// removePunctuation removes common punctuation characters from a word.
// This helps normalize words for better similarity comparison.
func removePunctuation(word string) string {
	return strings.Map(func(r rune) rune {
		// Remove common punctuation characters
		if unicode.IsPunct(r) {
			return -1 // Remove this character
		}
		return r
	}, word)
}
