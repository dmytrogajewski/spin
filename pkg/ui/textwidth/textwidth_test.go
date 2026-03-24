package textwidth

// Journey: specs/journeys/JOURNEY-R-REF-23.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractGraphemes(t *testing.T) {
	t.Parallel()

	t.Run("ascii", func(t *testing.T) {
		t.Parallel()

		got := ExtractGraphemes("abc")
		require.Equal(t, []string{"a", "b", "c"}, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		got := ExtractGraphemes("")
		require.Empty(t, got)
	})

	t.Run("emoji", func(t *testing.T) {
		t.Parallel()

		got := ExtractGraphemes("a👨‍👩‍👧b")
		require.Len(t, got, 3) // a, family emoji, b.
	})
}

func TestTotalWidth(t *testing.T) {
	t.Parallel()

	t.Run("ascii", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 3, TotalWidth([]string{"a", "b", "c"}))
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 0, TotalWidth(nil))
	})

	t.Run("wide_chars", func(t *testing.T) {
		t.Parallel()

		// Wide characters (e.g. fullwidth A) are width 2.
		require.Equal(t, 2, TotalWidth([]string{"\uff21"}))
	})
}

func TestMidEllipsize(t *testing.T) {
	t.Parallel()

	t.Run("fits", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("short", 10)
		require.Equal(t, "short", got)
	})

	t.Run("truncates", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("abcdefghij", 5)
		require.Contains(t, got, "…")
		require.NotEmpty(t, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("", 5)
		require.Empty(t, got)
	})

	t.Run("zero_width_returns_empty", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("abc", 0)
		require.Empty(t, got, "maxWidth=0 should return empty string")
	})

	t.Run("negative_width_returns_empty", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("abc", -5)
		require.Empty(t, got, "negative maxWidth should return empty string")
	})

	t.Run("exact_fit_no_ellipsis", func(t *testing.T) {
		t.Parallel()

		got := MidEllipsize("abc", 3)
		require.Equal(t, "abc", got, "string that exactly fits should not be ellipsized")
	})

	t.Run("wide_chars_reclaim_unused_width", func(t *testing.T) {
		t.Parallel()

		// "\uff21\uff22\uff23" = 3 fullwidth chars (width 6), maxWidth=5.
		// Without redistribution: leftWidth=2(can't fit w=2 char), rightWidth=2 → "…Ｃ" (width 3).
		// With redistribution: left unused donated to right → "…ＢＣ" or similar using full 5 cols.
		got := MidEllipsize("\uff21\uff22\uff23", 5)
		require.Contains(t, got, "…")
		w := TotalWidth(ExtractGraphemes(got))
		require.LessOrEqual(t, w, 5, "result should not exceed maxWidth")
		require.GreaterOrEqual(t, w, 4, "result should reclaim unused width from wide chars")
	})

	t.Run("wide_chars_small_maxwidth", func(t *testing.T) {
		t.Parallel()

		// "\uff21\uff22" = 2 fullwidth chars (width 4), maxWidth=3.
		// Best possible: "…Ｂ" (width 3).
		got := MidEllipsize("\uff21\uff22", 3)
		require.Contains(t, got, "…")
		w := TotalWidth(ExtractGraphemes(got))
		require.Equal(t, 3, w, "should use full width by redistributing unused space")
	})
}

func TestGutterWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lineCount int
		want      int
	}{
		{name: "single_digit", lineCount: 5, want: 3},
		{name: "double_digit", lineCount: 50, want: 4},
		{name: "triple_digit", lineCount: 500, want: 5},
		{name: "four_digit", lineCount: 5000, want: 6},
		{name: "zero", lineCount: 0, want: 3},
		{name: "five_digit", lineCount: 50000, want: 7},
		{name: "six_digit", lineCount: 500000, want: 8},
		{name: "negative", lineCount: -1, want: 3},
		{name: "boundary_10", lineCount: 10, want: 4},
		{name: "boundary_100", lineCount: 100, want: 5},
		{name: "boundary_1000", lineCount: 1000, want: 6},
		{name: "boundary_10000", lineCount: 10000, want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GutterWidth(tt.lineCount)
			require.Equal(t, tt.want, got)
		})
	}
}
