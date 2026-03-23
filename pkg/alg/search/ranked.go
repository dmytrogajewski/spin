// Package search provides generic scored search utilities.
package search

import "sort"

// scored pairs an item with its computed score for sorting.
type scored[Item any] struct {
	item  Item
	score float64
}

// RankedSearch scores each item, filters by minScore, sorts descending
// by score, and applies a result limit. A limit of 0 means unlimited.
func RankedSearch[Item any](items []Item, score func(Item) float64, minScore float64, limit int) []Item {
	var results []scored[Item]

	for _, item := range items {
		val := score(item)
		if val >= minScore {
			results = append(results, scored[Item]{item: item, score: val})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	output := make([]Item, len(results))
	for idx, res := range results {
		output[idx] = res.item
	}

	return output
}
