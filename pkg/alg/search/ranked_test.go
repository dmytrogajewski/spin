package search

// Journey: specs/journeys/JOURNEY-R15.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testItem is a simple scored item for testing.
type testItem struct {
	name  string
	value float64
}

// scoreByValue returns the item's value as its score.
func scoreByValue(item testItem) float64 {
	return item.value
}

func TestRankedSearch(t *testing.T) {
	t.Parallel()

	items := []testItem{
		{name: "low", value: 0.2},
		{name: "high", value: 0.9},
		{name: "mid", value: 0.5},
		{name: "below", value: 0.1},
	}

	t.Run("sorted_descending_and_filtered", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch(items, scoreByValue, 0.3, 0)
		require.Len(t, got, 2)
		require.Equal(t, "high", got[0].name)
		require.Equal(t, "mid", got[1].name)
	})

	t.Run("limit_caps_results", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch(items, scoreByValue, 0.0, 2)
		require.Len(t, got, 2)
		require.Equal(t, "high", got[0].name)
		require.Equal(t, "mid", got[1].name)
	})

	t.Run("zero_limit_means_unlimited", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch(items, scoreByValue, 0.0, 0)
		require.Len(t, got, 4)
	})

	t.Run("all_below_threshold", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch(items, scoreByValue, 1.0, 0)
		require.Empty(t, got)
	})

	t.Run("empty_input", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch([]testItem{}, scoreByValue, 0.0, 0)
		require.Empty(t, got)
	})

	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch[testItem](nil, scoreByValue, 0.0, 0)
		require.Empty(t, got)
	})

	t.Run("limit_larger_than_results", func(t *testing.T) {
		t.Parallel()

		got := RankedSearch(items, scoreByValue, 0.3, 100)
		require.Len(t, got, 2)
	})
}

func TestRankedSearch_strings(t *testing.T) {
	t.Parallel()

	words := []string{"banana", "apple", "cherry"}

	lengthScore := func(word string) float64 {
		return float64(len(word))
	}

	got := RankedSearch(words, lengthScore, 0.0, 2)
	require.Len(t, got, 2)
	// Both cherry and banana have 6 chars; either can be first (unstable sort).
	// apple (5 chars) must not appear.
	require.NotEqual(t, "apple", got[0])
	require.NotEqual(t, "apple", got[1])
}
