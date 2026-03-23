package collections

import "time"

// TailNOrAll returns the last n elements of a slice.
// Unlike [TailN], returns the full slice (copy) when n <= 0.
// Returns nil if input is nil or empty.
func TailNOrAll[Elem any](input []Elem, n int) []Elem {
	if len(input) == 0 {
		return nil
	}

	if n <= 0 || n >= len(input) {
		result := make([]Elem, len(input))
		copy(result, input)

		return result
	}

	result := make([]Elem, n)
	copy(result, input[len(input)-n:])

	return result
}

// DiffMaps compares two maps and returns the keys that were added, removed,
// or modified. Uses the equal function to compare values.
func DiffMaps[Key comparable, Val any](before, after map[Key]Val, equal func(Val, Val) bool) (added, removed, modified []Key) {
	for key, afterVal := range after {
		beforeVal, exists := before[key]
		if !exists {
			added = append(added, key)
		} else if !equal(beforeVal, afterVal) {
			modified = append(modified, key)
		}
	}

	for key := range before {
		if _, exists := after[key]; !exists {
			removed = append(removed, key)
		}
	}

	return added, removed, modified
}

// FilterSince returns elements whose timestamp (extracted by ts) is after since.
func FilterSince[Elem any](items []Elem, ts func(Elem) time.Time, since time.Time) []Elem {
	var result []Elem

	for _, item := range items {
		if ts(item).After(since) {
			result = append(result, item)
		}
	}

	return result
}
