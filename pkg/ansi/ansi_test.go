package ansi_test

import (
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/pkg/ansi"
)

func TestStrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no ansi",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "simple color",
			input: "\x1b[31mred\x1b[0m",
			want:  "red",
		},
		{
			name:  "bold",
			input: "\x1b[1mbold\x1b[0m",
			want:  "bold",
		},
		{
			name:  "combined styles",
			input: "\x1b[1;32mbold green\x1b[0m",
			want:  "bold green",
		},
		{
			name:  "multiple sequences",
			input: "\x1b[31mred\x1b[0m normal \x1b[1mbold\x1b[0m",
			want:  "red normal bold",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "only ansi",
			input: "\x1b[31m\x1b[0m",
			want:  "",
		},
		{
			name:  "partial match stripped",
			input: "\x1b[invalid text",
			want:  "nvalid text", // \x1b[i is a valid sequence (ESC [ i)
		},
		{
			name:  "incomplete sequence",
			input: "text\x1b[31",
			want:  "text\x1b[31",
		},
		{
			name:  "cursor control codes",
			input: "\x1b[2Ktext\x1b[1Gmore",
			want:  "textmore",
		},
		{
			name:  "dec save/restore",
			input: "\x1b7text\x1b8more",
			want:  "textmore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Strip(tt.input)
			if got != tt.want {
				t.Errorf("Strip() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "plain ascii",
			input: "hello",
			want:  5,
		},
		{
			name:  "with ansi",
			input: "\x1b[31mhello\x1b[0m",
			want:  5,
		},
		{
			name:  "bold with ansi",
			input: "\x1b[1mhello\x1b[0m",
			want:  5,
		},
		{
			name:  "utf8 characters",
			input: "你好",
			want:  2,
		},
		{
			name:  "utf8 with ansi",
			input: "\x1b[1m你好\x1b[0m",
			want:  2,
		},
		{
			name:  "mixed utf8 and ascii",
			input: "Hello 世界",
			want:  8,
		},
		{
			name:  "emoji",
			input: "🔥🚀",
			want:  2,
		},
		{
			name:  "emoji with ansi",
			input: "\x1b[32m🔥\x1b[0m",
			want:  1,
		},
		{
			name:  "empty",
			input: "",
			want:  0,
		},
		{
			name:  "only ansi",
			input: "\x1b[31m\x1b[0m",
			want:  0,
		},
		{
			name:  "spaces",
			input: "   ",
			want:  3,
		},
		{
			name:  "complex formatting",
			input: "\x1b[1m\x1b[32mBold Green Text\x1b[0m",
			want:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Length(tt.input)
			if got != tt.want {
				t.Errorf("Length() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are valid ANSI sequences
	tests := []struct {
		name     string
		constant string
		wantLen  int // Expected length of ANSI sequence
	}{
		{"Reset", ansi.Reset, 4},
		{"Black", ansi.Black, 5},
		{"Red", ansi.Red, 5},
		{"Green", ansi.Green, 5},
		{"Yellow", ansi.Yellow, 5},
		{"Blue", ansi.Blue, 5},
		{"Magenta", ansi.Magenta, 5},
		{"Cyan", ansi.Cyan, 5},
		{"White", ansi.White, 5},
		{"Bold", ansi.Bold, 4},
		{"Dim", ansi.Dim, 4},
		{"Italic", ansi.Italic, 4},
		{"Underline", ansi.Underline, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.constant) != tt.wantLen {
				t.Errorf("%s has length %d, want %d", tt.name, len(tt.constant), tt.wantLen)
			}

			// Verify it starts with ESC [
			if !strings.HasPrefix(tt.constant, "\x1b[") {
				t.Errorf("%s doesn't start with ESC [: %q", tt.name, tt.constant)
			}

			// Verify it ends with m (SGR code)
			if !strings.HasSuffix(tt.constant, "m") {
				t.Errorf("%s doesn't end with 'm': %q", tt.name, tt.constant)
			}
		})
	}
}

func TestStripPreservesNonANSI(t *testing.T) {
	// Ensure Strip doesn't modify non-ANSI control characters
	input := "line1\nline2\tindented\rcarriage"
	want := input

	got := ansi.Strip(input)
	if got != want {
		t.Errorf("Strip() modified non-ANSI control chars: got %q, want %q", got, want)
	}
}

func TestLengthWithControlChars(t *testing.T) {
	// Length should count control chars like \n, \t, \r
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"newline", "a\nb", 3},
		{"tab", "a\tb", 3},
		{"carriage return", "a\rb", 3},
		{"mixed", "a\n\tb\r", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Length(tt.input)
			if got != tt.want {
				t.Errorf("Length() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStripLargeInput(t *testing.T) {
	// Test with large input to ensure performance
	const size = 10000
	var b strings.Builder
	for i := 0; i < size; i++ {
		b.WriteString("\x1b[31mtext\x1b[0m ")
	}

	input := b.String()
	result := ansi.Strip(input)

	// Should strip all ANSI codes
	if strings.Contains(result, "\x1b") {
		t.Error("Strip() didn't remove all ANSI codes from large input")
	}

	// Should contain all the text
	expectedCount := size
	actualCount := strings.Count(result, "text")
	if actualCount != expectedCount {
		t.Errorf("Strip() text count = %d, want %d", actualCount, expectedCount)
	}
}
