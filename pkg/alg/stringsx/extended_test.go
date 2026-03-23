package stringsx

// Journey: specs/journeys/JOURNEY-S3.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateHeadTail(t *testing.T) {
	t.Parallel()

	t.Run("short_unchanged", func(t *testing.T) {
		t.Parallel()

		got := TruncateHeadTail("hello", 10, 3, 3)
		require.Equal(t, "hello", got)
	})

	t.Run("truncated_with_marker", func(t *testing.T) {
		t.Parallel()

		input := "abcdefghij"

		got := TruncateHeadTail(input, 5, 2, 2)
		require.Contains(t, got, "ab")
		require.Contains(t, got, "ij")
		require.Contains(t, got, "characters omitted")
	})
}

func TestTruncateLines(t *testing.T) {
	t.Parallel()

	t.Run("no_truncation", func(t *testing.T) {
		t.Parallel()

		got := TruncateLines("short\nline", 100)
		require.Equal(t, "short\nline", got)
	})

	t.Run("long_line_truncated", func(t *testing.T) {
		t.Parallel()

		input := "a very long line that exceeds the limit"

		got := TruncateLines(input, 25)
		require.Contains(t, got, truncatedSuffix)
		require.LessOrEqual(t, len(got), 25)
	})

	t.Run("empty_string", func(t *testing.T) {
		t.Parallel()

		got := TruncateLines("", 10)
		require.Empty(t, got)
	})
}

func TestIsPartialPrefix(t *testing.T) {
	t.Parallel()

	candidates := []string{"</tool_call>", "</thinking>"}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "strict_prefix", input: "</tool", want: true},
		{name: "full_match_is_not_partial", input: "</tool_call>", want: false},
		{name: "no_match", input: "xyz", want: false},
		{name: "empty_input", input: "", want: true},
		{name: "single_char_prefix", input: "<", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsPartialPrefix(tt.input, candidates)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFindMatchingClose(t *testing.T) {
	t.Parallel()

	t.Run("simple_close", func(t *testing.T) {
		t.Parallel()

		content := "<open>hello</close>"

		got := FindMatchingClose(content, 6, "<open>", "</close>")
		require.Equal(t, 11, got)
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		content := "<a><a>inner</a></a>"

		got := FindMatchingClose(content, 3, "<a>", "</a>")
		require.Equal(t, 15, got)
	})

	t.Run("no_close", func(t *testing.T) {
		t.Parallel()

		got := FindMatchingClose("<a>hello", 3, "<a>", "</a>")
		require.Equal(t, -1, got)
	})

	t.Run("empty_content", func(t *testing.T) {
		t.Parallel()

		got := FindMatchingClose("", 0, "<a>", "</a>")
		require.Equal(t, -1, got)
	})
}

func TestContainsIgnoreCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		substr string
		want   bool
	}{
		{name: "exact", input: "Hello World", substr: "hello", want: true},
		{name: "upper", input: "hello", substr: "HELLO", want: true},
		{name: "mixed", input: "FooBar", substr: "oob", want: true},
		{name: "no_match", input: "hello", substr: "xyz", want: false},
		{name: "empty_substr", input: "hello", substr: "", want: true},
		{name: "empty_both", input: "", substr: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContainsIgnoreCase(tt.input, tt.substr)
			require.Equal(t, tt.want, got)
		})
	}
}
