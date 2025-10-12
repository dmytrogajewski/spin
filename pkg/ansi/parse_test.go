package ansi_test

import (
	"reflect"
	"testing"

	"github.com/dmytrogajewski/spin/pkg/ansi"
)

func TestParse_PlainText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "empty",
			input: "",
			want:  []ansi.Segment{},
		},
		{
			name:  "simple text",
			input: "hello",
			want: []ansi.Segment{
				{Text: "hello"},
			},
		},
		{
			name:  "text with spaces",
			input: "hello world",
			want: []ansi.Segment{
				{Text: "hello world"},
			},
		},
		{
			name:  "multiline",
			input: "line1\nline2",
			want: []ansi.Segment{
				{Text: "line1\nline2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_SingleColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "red text",
			input: "\x1b[31mred\x1b[0m",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
			},
		},
		{
			name:  "green text",
			input: "\x1b[32mgreen\x1b[0m",
			want: []ansi.Segment{
				{Text: "green", Foreground: "green"},
			},
		},
		{
			name:  "yellow text",
			input: "\x1b[33myellow\x1b[0m",
			want: []ansi.Segment{
				{Text: "yellow", Foreground: "yellow"},
			},
		},
		{
			name:  "blue text",
			input: "\x1b[34mblue\x1b[0m",
			want: []ansi.Segment{
				{Text: "blue", Foreground: "blue"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_SingleStyle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "bold text",
			input: "\x1b[1mbold\x1b[0m",
			want: []ansi.Segment{
				{Text: "bold", Bold: true},
			},
		},
		{
			name:  "dim text",
			input: "\x1b[2mdim\x1b[0m",
			want: []ansi.Segment{
				{Text: "dim", Dim: true},
			},
		},
		{
			name:  "italic text",
			input: "\x1b[3mitalic\x1b[0m",
			want: []ansi.Segment{
				{Text: "italic", Italic: true},
			},
		},
		{
			name:  "underline text",
			input: "\x1b[4munderline\x1b[0m",
			want: []ansi.Segment{
				{Text: "underline", Underline: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_CombinedStyles(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "bold red",
			input: "\x1b[1;31mbold red\x1b[0m",
			want: []ansi.Segment{
				{Text: "bold red", Foreground: "red", Bold: true},
			},
		},
		{
			name:  "bold then red (separate codes)",
			input: "\x1b[1m\x1b[31mbold red\x1b[0m",
			want: []ansi.Segment{
				{Text: "bold red", Foreground: "red", Bold: true},
			},
		},
		{
			name:  "underline green",
			input: "\x1b[4;32munderline green\x1b[0m",
			want: []ansi.Segment{
				{Text: "underline green", Foreground: "green", Underline: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_MultipleSegments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "red then normal",
			input: "\x1b[31mred\x1b[0m normal",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
				{Text: " normal"},
			},
		},
		{
			name:  "normal then red",
			input: "normal \x1b[31mred\x1b[0m",
			want: []ansi.Segment{
				{Text: "normal "},
				{Text: "red", Foreground: "red"},
			},
		},
		{
			name:  "red then green",
			input: "\x1b[31mred\x1b[0m \x1b[32mgreen\x1b[0m",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
				{Text: " "},
				{Text: "green", Foreground: "green"},
			},
		},
		{
			name:  "three segments",
			input: "\x1b[31mred\x1b[0m normal \x1b[1mbold\x1b[0m",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
				{Text: " normal "},
				{Text: "bold", Bold: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_ResetHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []ansi.Segment
	}{
		{
			name:  "explicit reset",
			input: "\x1b[31mred\x1b[0mplain",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
				{Text: "plain"},
			},
		},
		{
			name:  "no reset at end",
			input: "\x1b[31mred text",
			want: []ansi.Segment{
				{Text: "red text", Foreground: "red"},
			},
		},
		{
			name:  "multiple resets",
			input: "\x1b[31mred\x1b[0m\x1b[32mgreen\x1b[0m",
			want: []ansi.Segment{
				{Text: "red", Foreground: "red"},
				{Text: "green", Foreground: "green"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_StateInheritance(t *testing.T) {
	// When changing color without reset, bold should persist
	input := "\x1b[1mbold \x1b[31mred bold\x1b[0m"
	want := []ansi.Segment{
		{Text: "bold ", Bold: true},
		{Text: "red bold", Foreground: "red", Bold: true},
	}

	got := ansi.Parse(input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() = %+v, want %+v", got, want)
	}
}

func TestParse_MalformedSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"incomplete sequence", "\x1b[31incomplete"},
		{"invalid code", "\x1b[999minvalid"},
		{"no terminator", "\x1b[31"},
		{"double escape", "\x1b\x1b[31mtext"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parse() panicked on malformed input: %v", r)
				}
			}()

			result := ansi.Parse(tt.input)
			// Should return something (best-effort)
			if result == nil {
				t.Error("Parse() returned nil for malformed input")
			}
		})
	}
}

func TestParse_EmptySegments(t *testing.T) {
	// Consecutive codes without text should not create empty segments
	input := "\x1b[31m\x1b[0m\x1b[32mtext\x1b[0m"

	got := ansi.Parse(input)

	// Should not have empty text segments
	for i, seg := range got {
		if seg.Text == "" {
			t.Errorf("Segment %d has empty text: %+v", i, seg)
		}
	}
}

func TestParse_LongText(t *testing.T) {
	// Test with long text containing multiple styles
	input := "\x1b[31mError:\x1b[0m Failed to process \x1b[1mfile.txt\x1b[0m. " +
		"Please check \x1b[33mwarning\x1b[0m messages above."

	got := ansi.Parse(input)

	// Should parse into multiple segments
	if len(got) < 4 {
		t.Errorf("Expected at least 4 segments, got %d", len(got))
	}

	// First segment should be red "Error:"
	if got[0].Foreground != "red" || got[0].Text != "Error:" {
		t.Errorf("First segment incorrect: %+v", got[0])
	}

	// Reconstruct text (should match original minus ANSI)
	var reconstructed string
	for _, seg := range got {
		reconstructed += seg.Text
	}

	expected := "Error: Failed to process file.txt. Please check warning messages above."
	if reconstructed != expected {
		t.Errorf("Reconstructed text = %q, want %q", reconstructed, expected)
	}
}

func TestParse_AllColors(t *testing.T) {
	// Test all 8 basic colors
	tests := []struct {
		code  string
		color string
	}{
		{"30", "black"},
		{"31", "red"},
		{"32", "green"},
		{"33", "yellow"},
		{"34", "blue"},
		{"35", "magenta"},
		{"36", "cyan"},
		{"37", "white"},
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			input := "\x1b[" + tt.code + "mtext\x1b[0m"
			got := ansi.Parse(input)

			if len(got) != 1 {
				t.Fatalf("Expected 1 segment, got %d", len(got))
			}

			if got[0].Foreground != tt.color {
				t.Errorf("Foreground = %q, want %q", got[0].Foreground, tt.color)
			}
		})
	}
}

func TestParse_UTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"chinese", "\x1b[31m你好\x1b[0m"},
		{"japanese", "\x1b[32mこんにちは\x1b[0m"},
		{"emoji", "\x1b[33m🔥🚀\x1b[0m"},
		{"mixed", "\x1b[34mHello 世界 🌍\x1b[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.Parse(tt.input)

			if len(got) == 0 {
				t.Fatal("Parse() returned empty result")
			}

			// Should preserve UTF-8 text
			plain := ansi.Strip(tt.input)
			if got[0].Text != plain {
				t.Errorf("Text = %q, want %q", got[0].Text, plain)
			}
		})
	}
}
