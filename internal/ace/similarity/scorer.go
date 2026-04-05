// Package similarity provides generic similarity-based operations for ACE subsystems.
//
// It replaces hardcoded similarity logic in curator/dedup, refine/merge,
// playbook/search, and retrieval with composable, testable abstractions.
package similarity

// Scorer computes the similarity between two items of type T.
// Returns a value in [0.0, 1.0] where 1.0 means identical.
type Scorer[T any] func(a, b T) float64

// ScoredItem pairs an item with its similarity score.
type ScoredItem[T any] struct {
	Item  T
	Score float64
}

// Pair represents two items found to be similar.
type Pair[T any] struct {
	A     T
	B     T
	Score float64
}

// FindPairs returns all pairs of items with similarity above the threshold.
// Each pair appears at most once (i.e., (a,b) but not (b,a)).
func FindPairs[T any](items []T, scorer Scorer[T], threshold float64) []Pair[T] {
	var pairs []Pair[T]

	for i := range items {
		for j := i + 1; j < len(items); j++ {
			score := scorer(items[i], items[j])
			if score >= threshold {
				pairs = append(pairs, Pair[T]{
					A:     items[i],
					B:     items[j],
					Score: score,
				})
			}
		}
	}

	return pairs
}

// TopK returns the k items most similar to the query, sorted by score descending.
// If fewer than k items exist, returns all items.
func TopK[T any](query T, items []T, scorer Scorer[T], k int) []ScoredItem[T] {
	if k <= 0 || len(items) == 0 {
		return nil
	}

	scored := make([]ScoredItem[T], len(items))
	for i, item := range items {
		scored[i] = ScoredItem[T]{
			Item:  item,
			Score: scorer(query, item),
		}
	}

	// Simple selection sort for top-k (efficient for small k).
	for i := range min(k, len(scored)) {
		maxIdx := i

		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[maxIdx].Score {
				maxIdx = j
			}
		}

		scored[i], scored[maxIdx] = scored[maxIdx], scored[i]
	}

	if k > len(scored) {
		k = len(scored)
	}

	return scored[:k]
}

// FilterAbove returns items with similarity to the query above the threshold.
func FilterAbove[T any](query T, items []T, scorer Scorer[T], threshold float64) []ScoredItem[T] {
	var result []ScoredItem[T]

	for _, item := range items {
		score := scorer(query, item)
		if score >= threshold {
			result = append(result, ScoredItem[T]{
				Item:  item,
				Score: score,
			})
		}
	}

	return result
}
