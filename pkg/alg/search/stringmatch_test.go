package search

// Journey: specs/journeys/JOURNEY-S9.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScoreString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		query string
		fuzzy bool
		want  float64
	}{
		{name: "exact_match", input: "read_file", query: "read_file", fuzzy: false, want: scoreExact},
		{name: "prefix_match", input: "read_file", query: "read", fuzzy: false, want: scorePrefix},
		{name: "contains_match", input: "read_file", query: "file", fuzzy: false, want: scoreContains},
		{name: "word_boundary", input: "read_file", query: "file", fuzzy: false, want: scoreContains},
		{name: "no_match", input: "read_file", query: "xyz", fuzzy: false, want: 0.0},
		{name: "fuzzy_disabled", input: "read_file", query: "reed_file", fuzzy: false, want: 0.0},
		{name: "fuzzy_close_match", input: "read_file", query: "reed_file", fuzzy: true, want: fuzzyWeight * (1.0 - 1.0/9.0)},
		{name: "fuzzy_too_different", input: "abc", query: "xyz", fuzzy: true, want: 0.0},
		{name: "empty_query", input: "hello", query: "", fuzzy: false, want: scorePrefix},
		{name: "both_empty", input: "", query: "", fuzzy: false, want: scoreExact},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ScoreString(tt.input, tt.query, tt.fuzzy)
			require.InDelta(t, tt.want, got, 0.001)
		})
	}
}
