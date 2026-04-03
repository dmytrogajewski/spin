package similarity

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// stringSimilarity is a simple scorer for testing: fraction of matching characters.
func stringSimilarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}

	maxLen := max(len(a), len(b))
	matching := 0

	for i := range min(len(a), len(b)) {
		if a[i] == b[i] {
			matching++
		}
	}

	return float64(matching) / float64(maxLen)
}

func TestFindPairs(t *testing.T) {
	t.Parallel()

	t.Run("finds_similar_pairs", func(t *testing.T) {
		t.Parallel()

		items := []string{"hello", "hella", "world", "worle"}
		pairs := FindPairs(items, stringSimilarity, 0.8)

		// hello/hella and world/worle.
		require.Len(t, pairs, 2)
	})

	t.Run("no_pairs_above_threshold", func(t *testing.T) {
		t.Parallel()

		items := []string{"abc", "xyz", "123"}
		pairs := FindPairs(items, stringSimilarity, 0.9)

		require.Empty(t, pairs)
	})

	t.Run("empty_input", func(t *testing.T) {
		t.Parallel()

		pairs := FindPairs([]string{}, stringSimilarity, 0.5)
		require.Empty(t, pairs)
	})

	t.Run("single_item", func(t *testing.T) {
		t.Parallel()

		pairs := FindPairs([]string{"one"}, stringSimilarity, 0.5)
		require.Empty(t, pairs)
	})

	t.Run("no_duplicates", func(t *testing.T) {
		t.Parallel()

		items := []string{"aa", "aa"}
		pairs := FindPairs(items, stringSimilarity, 0.9)

		// Should find exactly one pair (a,b) not (a,b) and (b,a).
		require.Len(t, pairs, 1)
		require.InDelta(t, 1.0, pairs[0].Score, 0.001)
	})
}

func TestTopK(t *testing.T) {
	t.Parallel()

	t.Run("returns_top_k", func(t *testing.T) {
		t.Parallel()

		items := []string{"hello", "hella", "world", "howdy"}
		results := TopK("hello", items, stringSimilarity, 2)

		require.Len(t, results, 2)
		// First result should be "hello" itself (score 1.0).
		require.Equal(t, "hello", results[0].Item)
		require.InDelta(t, 1.0, results[0].Score, 0.001)
	})

	t.Run("k_greater_than_items", func(t *testing.T) {
		t.Parallel()

		items := []string{"a", "b"}
		results := TopK("a", items, stringSimilarity, 10)

		require.Len(t, results, 2)
	})

	t.Run("k_zero", func(t *testing.T) {
		t.Parallel()

		results := TopK("a", []string{"a"}, stringSimilarity, 0)
		require.Empty(t, results)
	})

	t.Run("empty_items", func(t *testing.T) {
		t.Parallel()

		results := TopK("a", []string{}, stringSimilarity, 5)
		require.Empty(t, results)
	})

	t.Run("sorted_by_score", func(t *testing.T) {
		t.Parallel()

		items := []string{"abc", "abd", "xyz"}
		results := TopK("abc", items, stringSimilarity, 3)

		for i := 1; i < len(results); i++ {
			require.GreaterOrEqual(t, results[i-1].Score, results[i].Score)
		}
	})
}

func TestFilterAbove(t *testing.T) {
	t.Parallel()

	t.Run("filters_by_threshold", func(t *testing.T) {
		t.Parallel()

		items := []string{"hello", "hella", "world"}
		results := FilterAbove("hello", items, stringSimilarity, 0.8)

		// hello (1.0) and hella (0.8).
		require.Len(t, results, 2)
	})

	t.Run("none_above_threshold", func(t *testing.T) {
		t.Parallel()

		items := []string{"xyz"}
		results := FilterAbove("abc", items, stringSimilarity, 0.9)

		require.Empty(t, results)
	})

	t.Run("empty_items", func(t *testing.T) {
		t.Parallel()

		results := FilterAbove("a", []string{}, stringSimilarity, 0.5)
		require.Empty(t, results)
	})
}

func TestScoredItem_Score(t *testing.T) {
	t.Parallel()

	items := []string{"hello"}
	results := TopK("hello", items, stringSimilarity, 1)

	require.Len(t, results, 1)
	require.False(t, math.IsNaN(results[0].Score))
}
