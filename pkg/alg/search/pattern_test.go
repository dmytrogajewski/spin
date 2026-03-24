package search

// Journey: specs/journeys/JOURNEY-R-REF-11.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectRepeat(t *testing.T) {
	t.Parallel()

	intEq := func(a, b int) bool { return a == b }

	tests := []struct {
		name  string
		items []int
		want  bool
	}{
		{name: "nil_slice", items: nil, want: true},
		{name: "empty_slice", items: []int{}, want: true},
		{name: "single_element", items: []int{5}, want: true},
		{name: "all_same", items: []int{3, 3, 3}, want: true},
		{name: "first_differs", items: []int{1, 3, 3}, want: false},
		{name: "last_differs", items: []int{3, 3, 1}, want: false},
		{name: "two_same", items: []int{7, 7}, want: true},
		{name: "two_different", items: []int{7, 8}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectRepeat(tt.items, intEq)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDetectRepeat_CustomEq(t *testing.T) {
	t.Parallel()

	type item struct {
		key   string
		value int
	}

	byKey := func(a, b item) bool { return a.key == b.key }

	items := []item{
		{key: "read", value: 1},
		{key: "read", value: 2},
		{key: "read", value: 3},
	}

	require.True(t, DetectRepeat(items, byKey))

	items[2].key = "write"
	require.False(t, DetectRepeat(items, byKey))
}

func TestDetectAlternating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  bool
	}{
		{name: "nil_slice", items: nil, want: false},
		{name: "empty_slice", items: []string{}, want: false},
		{name: "one_element", items: []string{"a"}, want: false},
		{name: "two_elements", items: []string{"a", "b"}, want: false},
		{name: "three_elements", items: []string{"a", "b", "a"}, want: false},
		{name: "abab_pattern", items: []string{"a", "b", "a", "b"}, want: true},
		{name: "aaaa_pattern", items: []string{"a", "a", "a", "a"}, want: false},
		{name: "abba_pattern", items: []string{"a", "b", "b", "a"}, want: false},
		{name: "longer_with_abab_tail", items: []string{"x", "y", "a", "b", "a", "b"}, want: true},
		{name: "longer_without_pattern", items: []string{"a", "b", "a", "b", "c", "d"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectAlternating(tt.items)
			require.Equal(t, tt.want, got)
		})
	}
}
