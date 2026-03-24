// Package textwidth provides Unicode-aware text width calculation and
// ellipsization utilities using grapheme cluster segmentation.
package textwidth

import (
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
	graphemes := ExtractGraphemes(input)
	total := TotalWidth(graphemes)

	if total <= maxWidth {
		return input
	}

	leftWidth, rightWidth := splitWidths(maxWidth)
	left := buildLeft(graphemes, leftWidth)
	right := buildRight(graphemes, rightWidth)

	return strings.Join(left, "") + "…" + strings.Join(right, "")
}

// GutterWidth returns the appropriate gutter width for displaying line numbers.
func GutterWidth(lineCount int) int {
	const (
		threshold10   = 10
		threshold100  = 100
		threshold1000 = 1000
		width3        = 3
		width4        = 4
		width5        = 5
		width6        = 6
	)

	switch {
	case lineCount < threshold10:
		return width3
	case lineCount < threshold100:
		return width4
	case lineCount < threshold1000:
		return width5
	default:
		return width6
	}
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
	var result []string

	currentWidth := 0

	for i := len(graphemes) - 1; i >= 0; i-- {
		cluster := graphemes[i]

		cw := uniseg.StringWidth(cluster)
		if currentWidth+cw > maxWidth {
			break
		}

		result = append([]string{cluster}, result...)
		currentWidth += cw
	}

	return result
}
