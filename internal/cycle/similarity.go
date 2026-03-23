package cycle

import "github.com/dmytrogajewski/spin/pkg/alg/similarity"

// calculateSimilarity computes the Jaccard similarity between two strings.
// Jaccard similarity is defined as |A intersect B| / |A union B| where A and B are sets of words.
//
// This is a simple but effective measure for detecting when LLM responses
// are repeating similar content without being identical strings.
func calculateSimilarity(a, b string) float64 {
	return similarity.JaccardSimilarity(a, b)
}

// extractWords splits text into lowercase words for similarity comparison.
// Delegates to the shared similarity package.
func extractWords(text string) []string {
	return similarity.ExtractWords(text)
}
