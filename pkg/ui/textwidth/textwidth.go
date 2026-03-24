// Package textwidth provides Unicode-aware text width calculation and
// ellipsization utilities using grapheme cluster segmentation.
package textwidth

import (
	"math"
	"strings"

	"github.com/rivo/uniseg"
)

// ellipsisSplitRatio is the proportion of width allocated to the left side.
const ellipsisSplitRatio = 0.6

// ExtractGraphemes splits a string into its Unicode grapheme clusters.
func ExtractGraphemes(input string) []string {
	gr := uniseg.NewGraphemes(input)

	var graphemes []string
	for gr.Next() {
		graphemes = append(graphemes, gr.Str())
	}

	return graphemes
}

// TotalWidth returns the display width of a sequence of grapheme clusters.
func TotalWidth(graphemes []string) int {
	width := 0
	for _, g := range graphemes {
		width += uniseg.StringWidth(g)
	}

	return width
}

// MidEllipsize truncates a string in the middle, preserving the start and end,
// with an ellipsis ("…") in between. Returns the original string if it fits.
func MidEllipsize(input string, maxWidth int) string {
	if maxWidth < 1 {
		return ""
	}

	graphemes := ExtractGraphemes(input)
	total := TotalWidth(graphemes)

	if total <= maxWidth {
		return input
	}

	leftWidth, rightWidth := splitWidths(maxWidth)
	left := buildLeft(graphemes, leftWidth)
	right := buildRight(graphemes, rightWidth)

	// Reclaim unused width from either side and give to the other.
	// This prevents wasted space when wide characters (e.g. CJK) don't fit
	// in the initially allocated width.
	leftUsed := TotalWidth(left)
	rightUsed := TotalWidth(right)

	if leftUsed < leftWidth {
		// Left has unused width, expand right.
		right = buildRight(graphemes, maxWidth-1-leftUsed)
	} else if rightUsed < rightWidth {
		// Right has unused width, expand left.
		left = buildLeft(graphemes, maxWidth-1-rightUsed)
	}

	return strings.Join(left, "") + "…" + strings.Join(right, "")
}

// GutterWidth returns the appropriate gutter width for displaying line numbers.
// Scales logarithmically to support any line count.
func GutterWidth(lineCount int) int {
	const minWidth = 3

	if lineCount < 1 {
		lineCount = 1
	}

	const paddingWidth = 2

	digits := int(math.Log10(float64(lineCount))) + 1
	width := digits + paddingWidth

	return max(width, minWidth)
}

// splitWidths calculates left and right widths for mid-ellipsis splitting.
func splitWidths(maxWidth int) (leftWidth, rightWidth int) {
	leftWidth = int(float64(maxWidth-1) * ellipsisSplitRatio) // -1 for ellipsis character.
	rightWidth = maxWidth - leftWidth - 1

	return leftWidth, rightWidth
}

// buildLeft builds the left portion of the ellipsized string.
func buildLeft(graphemes []string, maxWidth int) []string {
	var result []string

	currentWidth := 0

	for _, g := range graphemes {
		w := uniseg.StringWidth(g)
		if currentWidth+w > maxWidth {
			break
		}

		result = append(result, g)
		currentWidth += w
	}

	return result
}

// buildRight builds the right portion of the ellipsized string.
func buildRight(graphemes []string, maxWidth int) []string {
	var reversed []string

	currentWidth := 0

	for i := len(graphemes) - 1; i >= 0; i-- {
		cluster := graphemes[i]

		cw := uniseg.StringWidth(cluster)
		if currentWidth+cw > maxWidth {
			break
		}

		reversed = append(reversed, cluster)
		currentWidth += cw
	}

	// Reverse to restore original order.
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	return reversed
}
