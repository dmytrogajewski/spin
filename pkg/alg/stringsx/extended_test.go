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

	t.Run("maxLen_smaller_than_suffix", func(t *testing.T) {
		t.Parallel()

		got := TruncateLines("a very long line here", 5)
		require.Equal(t, "a ver", got)
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

// Journey: specs/journeys/JOURNEY-R-REF-6.md.

func TestTruncateWithSuffix(t *testing.T) {
	t.Parallel()

	const (
		maxLen = 10
		suffix = "\n\n... [truncated]"
	)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "under_limit", input: "short", want: "short"},
		{name: "at_limit", input: "0123456789", want: "0123456789"},
		{name: "over_limit", input: "0123456789EXTRA", want: "0123456789" + suffix},
		{name: "empty_input", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateWithSuffix(tt.input, maxLen, suffix)
			require.Equal(t, tt.want, got)
		})
	}
}

// Journey: specs/journeys/JOURNEY-R-REF-17.md.

func TestNormalizeEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "newline", input: `hello\nworld`, want: "hello\nworld"},
		{name: "tab", input: `col1\tcol2`, want: "col1\tcol2"},
		{name: "quote", input: `say \"hi\"`, want: `say "hi"`},
		{name: "backslash", input: `path\\to\\file`, want: `path\to\file`},
		{name: "mixed", input: `line1\nline2\t\"ok\"`, want: "line1\nline2\t\"ok\""},
		{name: "no_escapes", input: "plain text", want: "plain text"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NormalizeEscapes(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMaskSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		key          string
		visibleChars int
		want         string
	}{
		{name: "normal_key", key: "sk-1234567890abcdef", visibleChars: 4, want: "sk-1...cdef"},
		{name: "short_key_masked", key: "abcd1234", visibleChars: 4, want: "***"},
		{name: "empty_key", key: "", visibleChars: 4, want: "***"},
		{name: "very_short_key", key: "abc", visibleChars: 4, want: "***"},
		{name: "zero_visible", key: "sk-1234567890abcdef", visibleChars: 0, want: "***"},
		{name: "visible_too_large", key: "abcdefghij", visibleChars: 5, want: "***"},
		{name: "one_visible_char", key: "abcdefghij", visibleChars: 1, want: "a...j"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MaskSecret(tt.key, tt.visibleChars)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		sep   string
		want  map[string]string
	}{
		{
			name:  "simple_pairs",
			items: []string{"Content-Type=application/json", "Authorization=Bearer token"},
			sep:   "=",
			want:  map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
		},
		{
			name:  "colon_separator",
			items: []string{"host:example.com", "port:8080"},
			sep:   ":",
			want:  map[string]string{"host": "example.com", "port": "8080"},
		},
		{
			name:  "value_with_separator",
			items: []string{"url=https://example.com"},
			sep:   "=",
			want:  map[string]string{"url": "https://example.com"},
		},
		{
			name:  "skip_invalid",
			items: []string{"valid=pair", "noseparator"},
			sep:   "=",
			want:  map[string]string{"valid": "pair"},
		},
		{
			name:  "empty_input",
			items: []string{},
			sep:   "=",
			want:  nil,
		},
		{
			name:  "nil_input",
			items: nil,
			sep:   "=",
			want:  nil,
		},
		{
			name:  "all_invalid",
			items: []string{"nosep1", "nosep2"},
			sep:   "=",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseKeyValuePairs(tt.items, tt.sep)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDetectTruncation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "balanced", input: `func main() { fmt.Println("hi") }`, want: ""},
		{name: "unclosed_brace", input: `func main() {`, want: "1 unclosed delimiter(s)"},
		{name: "unclosed_string", input: `var s = "hello`, want: "unclosed string literal"},
		{name: "unclosed_paren", input: `foo(bar(`, want: "2 unclosed delimiter(s)"},
		{name: "nested_balanced", input: `a(b[c{d}e]f)`, want: ""},
		{name: "string_with_braces", input: `fmt.Println("{")`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectTruncation(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
