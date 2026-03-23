package similarity

import "strings"

// NGrams produces all contiguous n-word combinations from the input slice.
// Returns nil if words has fewer than n elements or if n <= 0.
func NGrams(words []string, n int) []string {
	if n <= 0 || len(words) < n {
		return nil
	}

	count := len(words) - n + 1
	result := make([]string, 0, count)

	for idx := range count {
		result = append(result, strings.Join(words[idx:idx+n], " "))
	}

	return result
}

// MaxByFrequency returns the most frequent item and its count.
// Returns (zero, 0) for empty or nil input.
func MaxByFrequency[Item comparable](items []Item) (best Item, count int) {
	if len(items) == 0 {
		return best, 0
	}

	counts := make(map[Item]int, len(items))

	for _, item := range items {
		counts[item]++

		if counts[item] > count {
			count = counts[item]
			best = item
		}
	}

	return best, count
}
