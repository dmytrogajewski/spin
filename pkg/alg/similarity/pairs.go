package similarity

// Pair represents two items found to be similar.
type Pair[Item any] struct {
	// Left is the index of the first item.
	Left int
	// Right is the index of the second item.
	Right int
	// Score is the similarity score between the two items.
	Score float64
}

// FindSimilarPairs performs O(n²) pairwise comparison and returns all pairs
// with similarity >= threshold. Uses getText to extract comparable text from items.
func FindSimilarPairs[Item any](
	items []Item,
	getText func(Item) string,
	threshold float64,
	sim Strategy,
) []Pair[Item] {
	var pairs []Pair[Item]

	for i := range items {
		for j := i + 1; j < len(items); j++ {
			score := sim(getText(items[i]), getText(items[j]))
			if score >= threshold {
				pairs = append(pairs, Pair[Item]{
					Left:  i,
					Right: j,
					Score: score,
				})
			}
		}
	}

	return pairs
}
