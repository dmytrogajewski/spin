package similarity

// Journey: specs/journeys/JOURNEY-R14.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLevenshtein(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "identical", a: "hello", b: "hello", want: 0},
		{name: "kitten_sitting", a: "kitten", b: "sitting", want: 3},
		{name: "empty_both", a: "", b: "", want: 0},
		{name: "empty_first", a: "", b: "abc", want: 3},
		{name: "empty_second", a: "abc", b: "", want: 3},
		{name: "single_insert", a: "cat", b: "cats", want: 1},
		{name: "single_delete", a: "cats", b: "cat", want: 1},
		{name: "single_replace", a: "cat", b: "car", want: 1},
		{name: "completely_different", a: "abc", b: "xyz", want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Levenshtein(tt.a, tt.b)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		a         string
		b         string
		wantDelta float64
		wantEps   float64
	}{
		{name: "identical_strings", a: "hello world", b: "hello world", wantDelta: 1.0, wantEps: 1e-10},
		{name: "completely_disjoint", a: "hello world", b: "goodbye universe", wantDelta: 0.0, wantEps: 1e-10},
		{name: "partial_overlap", a: "the quick brown fox", b: "the slow brown dog", wantDelta: 0.333, wantEps: 0.05},
		{name: "both_empty", a: "", b: "", wantDelta: 1.0, wantEps: 1e-10},
		{name: "one_empty", a: "hello", b: "", wantDelta: 0.0, wantEps: 1e-10},
		{name: "short_words_filtered", a: "a b c", b: "a b c", wantDelta: 1.0, wantEps: 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := JaccardSimilarity(tt.a, tt.b)
			require.InDelta(t, tt.wantDelta, got, tt.wantEps)
		})
	}
}

func TestJaccardSimilarity_commutative(t *testing.T) {
	t.Parallel()

	a := "the quick brown fox jumps"
	b := "the lazy brown dog sleeps"

	require.InDelta(t, JaccardSimilarity(a, b), JaccardSimilarity(b, a), 1e-10)
}

func TestNGrams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		words []string
		size  int
		want  []string
	}{
		{name: "trigrams", words: []string{"a", "b", "c", "d"}, size: 3, want: []string{"a b c", "b c d"}},
		{name: "unigrams", words: []string{"a", "b", "c"}, size: 1, want: []string{"a", "b", "c"}},
		{name: "fewer_than_n", words: []string{"a", "b"}, size: 3, want: nil},
		{name: "exact_n", words: []string{"a", "b", "c"}, size: 3, want: []string{"a b c"}},
		{name: "empty", words: nil, size: 3, want: nil},
		{name: "n_zero", words: []string{"a"}, size: 0, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NGrams(tt.words, tt.size)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMaxByFrequency(t *testing.T) {
	t.Parallel()

	t.Run("clear_winner", func(t *testing.T) {
		t.Parallel()

		item, count := MaxByFrequency([]string{"a", "b", "a", "c", "a"})
		require.Equal(t, "a", item)
		require.Equal(t, 3, count)
	})

	t.Run("single_item", func(t *testing.T) {
		t.Parallel()

		item, count := MaxByFrequency([]string{"only"})
		require.Equal(t, "only", item)
		require.Equal(t, 1, count)
	})

	t.Run("empty_returns_zero", func(t *testing.T) {
		t.Parallel()

		item, count := MaxByFrequency([]string{})
		require.Empty(t, item)
		require.Equal(t, 0, count)
	})

	t.Run("nil_returns_zero", func(t *testing.T) {
		t.Parallel()

		item, count := MaxByFrequency[string](nil)
		require.Empty(t, item)
		require.Equal(t, 0, count)
	})

	t.Run("int_items", func(t *testing.T) {
		t.Parallel()

		item, count := MaxByFrequency([]int{1, 2, 2, 3, 2})
		require.Equal(t, 2, item)
		require.Equal(t, 3, count)
	})
}

// Journey: specs/journeys/JOURNEY-S5.md.

func TestLongestCommonSubstring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		first     string
		second    string
		wantStart int
		wantLen   int
	}{
		{name: "identical", first: "hello", second: "hello", wantStart: 0, wantLen: 5},
		{name: "disjoint", first: "abc", second: "xyz", wantStart: 0, wantLen: 0},
		{name: "partial_overlap", first: "abcdef", second: "xbcdey", wantStart: 1, wantLen: 4},
		{name: "empty_first", first: "", second: "abc", wantStart: 0, wantLen: 0},
		{name: "empty_second", first: "abc", second: "", wantStart: 0, wantLen: 0},
		{name: "both_empty", first: "", second: "", wantStart: 0, wantLen: 0},
		{name: "single_char_match", first: "axb", second: "cxd", wantStart: 1, wantLen: 1},
		{name: "prefix_match", first: "abcxyz", second: "abcqqq", wantStart: 0, wantLen: 3},
		{name: "suffix_match", first: "xxxabc", second: "yyyabc", wantStart: 3, wantLen: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotStart, gotLen := LongestCommonSubstring(tt.first, tt.second)
			require.Equal(t, tt.wantStart, gotStart, "start position")
			require.Equal(t, tt.wantLen, gotLen, "length")
		})
	}
}
