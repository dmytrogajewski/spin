package collections

// Journey: specs/journeys/JOURNEY-R2.md.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTailN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		count int
		want  []int
	}{
		{name: "nil_slice", input: nil, count: 3, want: nil},
		{name: "empty_slice", input: []int{}, count: 3, want: nil},
		{name: "n_zero", input: []int{1, 2, 3}, count: 0, want: nil},
		{name: "n_negative", input: []int{1, 2, 3}, count: -1, want: nil},
		{name: "n_greater_than_len", input: []int{1, 2}, count: 5, want: []int{1, 2}},
		{name: "n_equals_len", input: []int{1, 2, 3}, count: 3, want: []int{1, 2, 3}},
		{name: "n_less_than_len", input: []int{1, 2, 3, 4, 5}, count: 2, want: []int{4, 5}},
		{name: "n_one", input: []int{10, 20, 30}, count: 1, want: []int{30}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TailN(tt.input, tt.count)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTailN_strings(t *testing.T) {
	t.Parallel()

	got := TailN([]string{"a", "b", "c"}, 2)
	require.Equal(t, []string{"b", "c"}, got)
}

func TestToSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  map[string]bool
	}{
		{name: "nil_slice", input: nil, want: nil},
		{name: "empty_slice", input: []string{}, want: map[string]bool{}},
		{name: "unique_items", input: []string{"a", "b", "c"}, want: map[string]bool{"a": true, "b": true, "c": true}},
		{name: "duplicate_items", input: []string{"a", "b", "a"}, want: map[string]bool{"a": true, "b": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToSet(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestToSet_ints(t *testing.T) {
	t.Parallel()

	got := ToSet([]int{1, 2, 2, 3})
	require.Equal(t, map[int]bool{1: true, 2: true, 3: true}, got)
}

func TestAllSame(t *testing.T) {
	t.Parallel()

	identity := func(s string) string { return s }

	tests := []struct {
		name  string
		input []string
		want  bool
	}{
		{name: "empty", input: nil, want: true},
		{name: "single", input: []string{"a"}, want: true},
		{name: "all_same", input: []string{"x", "x", "x"}, want: true},
		{name: "one_different", input: []string{"x", "y", "x"}, want: false},
		{name: "first_different", input: []string{"y", "x", "x"}, want: false},
		{name: "last_different", input: []string{"x", "x", "y"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := AllSame(tt.input, identity)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAllSame_with_key_extractor(t *testing.T) {
	t.Parallel()

	type item struct {
		name  string
		value int
	}

	items := []item{
		{name: "a", value: 1},
		{name: "b", value: 1},
		{name: "c", value: 1},
	}

	// All same value via key extractor.
	got := AllSame(items, func(it item) item {
		return item{value: it.value}
	})
	require.True(t, got)
}

func TestMean(t *testing.T) {
	t.Parallel()

	t.Run("empty_returns_zero", func(t *testing.T) {
		t.Parallel()

		got := Mean([]float64(nil))
		require.InDelta(t, 0.0, got, 1e-10)
	})

	t.Run("single_value", func(t *testing.T) {
		t.Parallel()

		got := Mean([]float64{5.0})
		require.InDelta(t, 5.0, got, 1e-10)
	})

	t.Run("typical_values", func(t *testing.T) {
		t.Parallel()

		got := Mean([]float64{2.0, 4.0, 6.0})
		require.InDelta(t, 4.0, got, 1e-10)
	})

	t.Run("float32", func(t *testing.T) {
		t.Parallel()

		got := Mean([]float32{1.0, 3.0})
		require.InDelta(t, float32(2.0), got, 1e-5)
	})
}

func TestRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		before      int
		after       int
		wantApprox  float64
		wantEpsilon float64
	}{
		{name: "zero_denominator", before: 0, after: 5, wantApprox: 0, wantEpsilon: 1e-10},
		{name: "equal_values", before: 10, after: 10, wantApprox: 1.0, wantEpsilon: 1e-10},
		{name: "half", before: 100, after: 50, wantApprox: 0.5, wantEpsilon: 1e-10},
		{name: "double", before: 50, after: 100, wantApprox: 2.0, wantEpsilon: 1e-10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Ratio(tt.before, tt.after)
			require.InDelta(t, tt.wantApprox, got, tt.wantEpsilon)
		})
	}
}

func TestEnsureMap(t *testing.T) {
	t.Parallel()

	t.Run("nil_returns_non_nil", func(t *testing.T) {
		t.Parallel()

		got := EnsureMap[string, int](nil)
		require.NotNil(t, got)
		require.Empty(t, got)
	})

	t.Run("non_nil_returns_same", func(t *testing.T) {
		t.Parallel()

		existing := map[string]int{"a": 1}
		got := EnsureMap(existing)
		require.Equal(t, existing, got)

		// Verify it is the same map instance by mutating.
		got["b"] = 2

		require.Equal(t, 2, existing["b"])
	})
}

func TestPtr(t *testing.T) {
	t.Parallel()

	t.Run("int_value", func(t *testing.T) {
		t.Parallel()

		got := Ptr(42)
		require.NotNil(t, got)
		require.Equal(t, 42, *got)
	})

	t.Run("zero_value", func(t *testing.T) {
		t.Parallel()

		got := Ptr(0)
		require.NotNil(t, got)
		require.Equal(t, 0, *got)
	})

	t.Run("string_value", func(t *testing.T) {
		t.Parallel()

		got := Ptr("hello")
		require.NotNil(t, got)
		require.Equal(t, "hello", *got)
	})
}

// Journey: specs/journeys/JOURNEY-S2.md.

func TestTailNOrAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		count int
		want  []int
	}{
		{name: "nil_slice", input: nil, count: 3, want: nil},
		{name: "empty_slice", input: []int{}, count: 3, want: nil},
		{name: "n_zero_returns_all", input: []int{1, 2, 3}, count: 0, want: []int{1, 2, 3}},
		{name: "n_negative_returns_all", input: []int{1, 2, 3}, count: -1, want: []int{1, 2, 3}},
		{name: "n_greater_than_len", input: []int{1, 2}, count: 5, want: []int{1, 2}},
		{name: "n_equals_len", input: []int{1, 2, 3}, count: 3, want: []int{1, 2, 3}},
		{name: "n_less_than_len", input: []int{1, 2, 3, 4, 5}, count: 2, want: []int{4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TailNOrAll(tt.input, tt.count)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDiffMaps(t *testing.T) {
	t.Parallel()

	intEqual := func(a, b int) bool { return a == b }

	t.Run("added_keys", func(t *testing.T) {
		t.Parallel()

		before := map[string]int{"a": 1}
		after := map[string]int{"a": 1, "b": 2}

		added, removed, modified := DiffMaps(before, after, intEqual)
		require.Equal(t, []string{"b"}, added)
		require.Empty(t, removed)
		require.Empty(t, modified)
	})

	t.Run("removed_keys", func(t *testing.T) {
		t.Parallel()

		before := map[string]int{"a": 1, "b": 2}
		after := map[string]int{"a": 1}

		added, removed, modified := DiffMaps(before, after, intEqual)
		require.Empty(t, added)
		require.Equal(t, []string{"b"}, removed)
		require.Empty(t, modified)
	})

	t.Run("modified_keys", func(t *testing.T) {
		t.Parallel()

		before := map[string]int{"a": 1}
		after := map[string]int{"a": 99}

		added, removed, modified := DiffMaps(before, after, intEqual)
		require.Empty(t, added)
		require.Empty(t, removed)
		require.Equal(t, []string{"a"}, modified)
	})

	t.Run("both_empty", func(t *testing.T) {
		t.Parallel()

		added, removed, modified := DiffMaps(map[string]int{}, map[string]int{}, intEqual)
		require.Empty(t, added)
		require.Empty(t, removed)
		require.Empty(t, modified)
	})

	t.Run("both_nil", func(t *testing.T) {
		t.Parallel()

		added, removed, modified := DiffMaps[string, int](nil, nil, intEqual)
		require.Empty(t, added)
		require.Empty(t, removed)
		require.Empty(t, modified)
	})
}

func TestFilterSince(t *testing.T) {
	t.Parallel()

	type entry struct {
		name string
		at   time.Time
	}

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	items := []entry{
		{name: "old", at: base},
		{name: "mid", at: base.Add(time.Hour)},
		{name: "new", at: base.Add(2 * time.Hour)},
	}

	tsFunc := func(item entry) time.Time { return item.at }

	t.Run("filter_keeps_recent", func(t *testing.T) {
		t.Parallel()

		got := FilterSince(items, tsFunc, base.Add(90*time.Minute))
		require.Len(t, got, 1)
		require.Equal(t, "new", got[0].name)
	})

	t.Run("filter_all_old", func(t *testing.T) {
		t.Parallel()

		got := FilterSince(items, tsFunc, base.Add(3*time.Hour))
		require.Empty(t, got)
	})

	t.Run("filter_all_new", func(t *testing.T) {
		t.Parallel()

		got := FilterSince(items, tsFunc, base.Add(-time.Hour))
		require.Len(t, got, 3)
	})

	t.Run("nil_input", func(t *testing.T) {
		t.Parallel()

		got := FilterSince[entry](nil, tsFunc, base)
		require.Empty(t, got)
	})
}
