package search

import "strings"

// MapNormalizedOffset maps an offset in a normalized string back to the
// corresponding position in the original string. Used when searching in
// whitespace-normalized content and mapping results back to originals.
func MapNormalizedOffset(original, normalized string, normOffset int) int {
	origIdx := 0
	normIdx := 0

	for normIdx < normOffset && origIdx < len(original) {
		if normIdx < len(normalized) && original[origIdx] == normalized[normIdx] {
			origIdx++
			normIdx++
		} else {
			origIdx++
		}
	}

	return origIdx
}

// FindAllNormalized finds all occurrences of needle in normalizedHaystack
// and maps the offsets back to the original haystack.
// Returns (start, end) pairs in original coordinates.
func FindAllNormalized(original, normalizedHaystack, normalizedNeedle string) [][2]int {
	var results [][2]int

	searchFrom := 0

	for {
		idx := strings.Index(normalizedHaystack[searchFrom:], normalizedNeedle)
		if idx < 0 {
			break
		}

		absIdx := searchFrom + idx
		origStart := MapNormalizedOffset(original, normalizedHaystack, absIdx)
		origEnd := MapNormalizedOffset(original, normalizedHaystack, absIdx+len(normalizedNeedle))

		results = append(results, [2]int{origStart, origEnd})
		searchFrom = absIdx + 1
	}

	return results
}

// MatchesAt checks if target matches source starting at position pos.
// Uses the provided equality function for comparison.
func MatchesAt[Elem any](source, target []Elem, pos int, eq func(Elem, Elem) bool) bool {
	if pos+len(target) > len(source) {
		return false
	}

	for idx, val := range target {
		if !eq(source[pos+idx], val) {
			return false
		}
	}

	return true
}

// LineOffset returns the byte offset of the line at the given index.
// Each line is assumed to be followed by a newline character.
func LineOffset(lines []string, lineIdx int) int {
	offset := 0

	for idx := range lineIdx {
		offset += len(lines[idx]) + 1 // +1 for newline.
	}

	return offset
}

// LineOffsetEnd returns the byte offset at the end of the line at the given index.
func LineOffsetEnd(lines []string, lineIdx int) int {
	return LineOffset(lines, lineIdx) + len(lines[lineIdx])
}
