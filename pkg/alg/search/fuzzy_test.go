package search

// Journey: specs/journeys/JOURNEY-R-REF-21.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapNormalizedOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		original   string
		normalized string
		normOffset int
		want       int
	}{
		{name: "identical", original: "abc", normalized: "abc", normOffset: 2, want: 2},
		{name: "extra_spaces", original: "a  b", normalized: "a b", normOffset: 2, want: 2},
		{name: "start", original: "abc", normalized: "abc", normOffset: 0, want: 0},
		{name: "end", original: "abc", normalized: "abc", normOffset: 3, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MapNormalizedOffset(tt.original, tt.normalized, tt.normOffset)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFindAllNormalized(t *testing.T) {
	t.Parallel()

	t.Run("single_match", func(t *testing.T) {
		t.Parallel()

		results := FindAllNormalized("hello world", "hello world", "world")
		require.Len(t, results, 1)
		require.Equal(t, 6, results[0][0])
		require.Equal(t, 11, results[0][1])
	})

	t.Run("no_match", func(t *testing.T) {
		t.Parallel()

		results := FindAllNormalized("hello", "hello", "xyz")
		require.Empty(t, results)
	})

	t.Run("multiple_matches", func(t *testing.T) {
		t.Parallel()

		results := FindAllNormalized("aa bb aa", "aa bb aa", "aa")
		require.Len(t, results, 2)
	})
}

func TestMatchesAt(t *testing.T) {
	t.Parallel()

	strEq := func(a, b string) bool { return a == b }

	t.Run("match_at_start", func(t *testing.T) {
		t.Parallel()

		got := MatchesAt([]string{"a", "b", "c"}, []string{"a", "b"}, 0, strEq)
		require.True(t, got)
	})

	t.Run("match_at_offset", func(t *testing.T) {
		t.Parallel()

		got := MatchesAt([]string{"a", "b", "c"}, []string{"b", "c"}, 1, strEq)
		require.True(t, got)
	})

	t.Run("no_match", func(t *testing.T) {
		t.Parallel()

		got := MatchesAt([]string{"a", "b", "c"}, []string{"b", "d"}, 1, strEq)
		require.False(t, got)
	})

	t.Run("out_of_bounds", func(t *testing.T) {
		t.Parallel()

		got := MatchesAt([]string{"a"}, []string{"a", "b"}, 0, strEq)
		require.False(t, got)
	})

	t.Run("empty_target", func(t *testing.T) {
		t.Parallel()

		got := MatchesAt([]string{"a"}, []string{}, 0, strEq)
		require.True(t, got)
	})
}

func TestLineOffset(t *testing.T) {
	t.Parallel()

	lines := []string{"hello", "world", "foo"}

	require.Equal(t, 0, LineOffset(lines, 0))
	require.Equal(t, 6, LineOffset(lines, 1))  // 5 chars + newline.
	require.Equal(t, 12, LineOffset(lines, 2)) // Two lines of 5 chars + 2 newlines.
}

func TestLineOffsetEnd(t *testing.T) {
	t.Parallel()

	lines := []string{"hello", "world"}

	require.Equal(t, 5, LineOffsetEnd(lines, 0))  // "hello" ends at 5.
	require.Equal(t, 11, LineOffsetEnd(lines, 1)) // "hello\nworld" ends at 11.
}
