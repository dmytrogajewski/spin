package stringsx

// Journey: specs/journeys/JOURNEY-R3.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollapseWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "only_spaces", input: "   ", want: ""},
		{name: "only_tabs", input: "\t\t", want: ""},
		{name: "single_word", input: "hello", want: "hello"},
		{name: "normal_spacing", input: "hello world", want: "hello world"},
		{name: "multiple_spaces", input: "hello   world", want: "hello world"},
		{name: "tabs_and_spaces", input: "hello \t world", want: "hello world"},
		{name: "leading_trailing", input: "  hello  ", want: "hello"},
		{name: "newlines_collapse", input: "hello\n\nworld", want: "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CollapseWhitespace(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCollapseBlankLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "no_newlines", input: "hello world", want: "hello world"},
		{name: "single_newline", input: "a\nb", want: "a\nb"},
		{name: "double_newline_preserved", input: "a\n\nb", want: "a\n\nb"},
		{name: "triple_collapsed", input: "a\n\n\nb", want: "a\n\nb"},
		{name: "many_newlines", input: "a\n\n\n\n\nb", want: "a\n\nb"},
		{name: "multiple_groups", input: "a\n\n\nb\n\n\nc", want: "a\n\nb\n\nc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CollapseBlankLines(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTrimTrailingPerLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "no_trailing", input: "hello\nworld", want: "hello\nworld"},
		{name: "trailing_spaces", input: "hello  \nworld  ", want: "hello\nworld"},
		{name: "trailing_tabs", input: "hello\t\nworld\t", want: "hello\nworld"},
		{name: "leading_preserved", input: "  hello\n  world", want: "  hello\n  world"},
		{name: "mixed_trailing", input: "hello \t\nworld\t ", want: "hello\nworld"},
		{name: "all_whitespace_line", input: "hello\n   \nworld", want: "hello\n\nworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TrimTrailingPerLine(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{name: "short_unchanged", input: "hi", maxLen: 10, want: "hi"},
		{name: "exact_boundary", input: "hello", maxLen: 5, want: "hello"},
		{name: "truncated", input: "hello world", maxLen: 8, want: "hello..."},
		{name: "maxlen_equals_ellipsis", input: "hello", maxLen: 3, want: "..."},
		{name: "maxlen_below_ellipsis", input: "hello", maxLen: 2, want: "he"},
		{name: "empty_string", input: "", maxLen: 10, want: ""},
		{name: "maxlen_zero", input: "hello", maxLen: 0, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateWithEllipsis(tt.input, tt.maxLen)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestContainsAnyKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		keywords []string
		want     bool
	}{
		{name: "empty_string", input: "", keywords: []string{"error"}, want: false},
		{name: "empty_keywords", input: "hello error", keywords: nil, want: false},
		{name: "exact_match", input: "error occurred", keywords: []string{"error"}, want: true},
		{name: "case_insensitive", input: "Fatal Error", keywords: []string{"error"}, want: true},
		{name: "substring_match", input: "unfailed test", keywords: []string{"failed"}, want: true},
		{name: "no_match", input: "all good", keywords: []string{"error", "failed"}, want: false},
		{name: "multiple_keywords_second_matches", input: "panic at disco", keywords: []string{"error", "panic"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContainsAnyKeyword(tt.input, tt.keywords)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()

	nonEmpty := func(line string) bool { return line != "" }

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty", input: "", want: 0},
		{name: "single_line", input: "hello", want: 1},
		{name: "two_lines", input: "hello\nworld", want: 2},
		{name: "blank_lines_skipped", input: "hello\n\nworld", want: 2},
		{name: "all_blank", input: "\n\n\n", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CountLines(tt.input, nonEmpty)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCountLines_nil_predicate(t *testing.T) {
	t.Parallel()

	// Nil predicate counts all lines.
	got := CountLines("a\nb\nc", nil)
	require.Equal(t, 3, got)
}

func TestStripCodeFence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantBody string
		wantLang string
	}{
		{name: "no_fences", input: "hello world", wantBody: "hello world", wantLang: ""},
		{name: "json_fence", input: "```json\n{\"a\":1}\n```", wantBody: "{\"a\":1}", wantLang: "json"},
		{name: "plain_fence", input: "```\nhello\n```", wantBody: "hello", wantLang: ""},
		{name: "yaml_fence", input: "```yaml\nkey: val\n```", wantBody: "key: val", wantLang: "yaml"},
		{name: "no_closing_fence", input: "```json\n{\"a\":1}", wantBody: "{\"a\":1}", wantLang: "json"},
		{name: "only_closing_fence", input: "hello\n```", wantBody: "hello", wantLang: ""},
		{name: "whitespace_around", input: "  ```json\n{}\n```  ", wantBody: "{}", wantLang: "json"},
		{name: "empty_content", input: "```json\n```", wantBody: "", wantLang: "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotBody, gotLang := StripCodeFence(tt.input)
			require.Equal(t, tt.wantBody, gotBody)
			require.Equal(t, tt.wantLang, gotLang)
		})
	}
}

func TestStripListPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no_prefix", input: "hello world", want: "hello world"},
		{name: "dash_prefix", input: "- item one", want: "item one"},
		{name: "asterisk_prefix", input: "* item two", want: "item two"},
		{name: "numbered_dot", input: "1. first item", want: "first item"},
		{name: "numbered_paren", input: "3) third item", want: "third item"},
		{name: "short_string", input: "hi", want: "hi"},
		{name: "empty", input: "", want: ""},
		{name: "dash_no_space", input: "-nospace", want: "-nospace"},
		{name: "only_prefix", input: "1. ", want: ""},
		{name: "multi_digit", input: "10. tenth", want: "10. tenth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StripListPrefix(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
