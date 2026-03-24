package search

// minAlternatingLen is the minimum number of items needed to detect A→B→A→B.
const minAlternatingLen = 4

// DetectRepeat returns true if all items are equal according to eq.
// Returns true for empty or single-element slices (vacuous truth).
func DetectRepeat[Elem any](items []Elem, eq func(Elem, Elem) bool) bool {
	if len(items) <= 1 {
		return true
	}

	for i := 1; i < len(items); i++ {
		if !eq(items[0], items[i]) {
			return false
		}
	}

	return true
}

// DetectAlternating returns true if items follow an A→B→A→B pattern
// where A != B, checking the last 4 elements.
// Returns false if fewer than 4 items.
func DetectAlternating[Elem comparable](items []Elem) bool {
	if len(items) < minAlternatingLen {
		return false
	}

	tail := items[len(items)-minAlternatingLen:]

	// Pattern: [0]==A, [1]==B, [2]==A, [3]==B where A != B.
	return tail[0] == tail[2] && tail[1] == tail[3] && tail[0] != tail[1]
}
