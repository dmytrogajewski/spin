package ansi_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/pkg/ansi"
)

func TestStyle_SingleColor(t *testing.T) {
	tests := []struct {
		name  string
		build func() string
		want  string
	}{
		{
			name:  "red",
			build: func() string { return ansi.New("text").Red().String() },
			want:  "\x1b[31mtext\x1b[0m",
		},
		{
			name:  "green",
			build: func() string { return ansi.New("text").Green().String() },
			want:  "\x1b[32mtext\x1b[0m",
		},
		{
			name:  "yellow",
			build: func() string { return ansi.New("text").Yellow().String() },
			want:  "\x1b[33mtext\x1b[0m",
		},
		{
			name:  "blue",
			build: func() string { return ansi.New("text").Blue().String() },
			want:  "\x1b[34mtext\x1b[0m",
		},
		{
			name:  "magenta",
			build: func() string { return ansi.New("text").Magenta().String() },
			want:  "\x1b[35mtext\x1b[0m",
		},
		{
			name:  "cyan",
			build: func() string { return ansi.New("text").Cyan().String() },
			want:  "\x1b[36mtext\x1b[0m",
		},
		{
			name:  "white",
			build: func() string { return ansi.New("text").White().String() },
			want:  "\x1b[37mtext\x1b[0m",
		},
		{
			name:  "black",
			build: func() string { return ansi.New("text").Black().String() },
			want:  "\x1b[30mtext\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.build()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyle_SingleStyle(t *testing.T) {
	tests := []struct {
		name  string
		build func() string
		want  string
	}{
		{
			name:  "bold",
			build: func() string { return ansi.New("text").Bold().String() },
			want:  "\x1b[1mtext\x1b[0m",
		},
		{
			name:  "dim",
			build: func() string { return ansi.New("text").Dim().String() },
			want:  "\x1b[2mtext\x1b[0m",
		},
		{
			name:  "italic",
			build: func() string { return ansi.New("text").Italic().String() },
			want:  "\x1b[3mtext\x1b[0m",
		},
		{
			name:  "underline",
			build: func() string { return ansi.New("text").Underline().String() },
			want:  "\x1b[4mtext\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.build()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyle_Combined(t *testing.T) {
	tests := []struct {
		name  string
		build func() string
		want  string
	}{
		{
			name:  "red bold",
			build: func() string { return ansi.New("text").Red().Bold().String() },
			want:  "\x1b[31m\x1b[1mtext\x1b[0m",
		},
		{
			name:  "bold red (order matters)",
			build: func() string { return ansi.New("text").Bold().Red().String() },
			want:  "\x1b[1m\x1b[31mtext\x1b[0m",
		},
		{
			name:  "green underline",
			build: func() string { return ansi.New("text").Green().Underline().String() },
			want:  "\x1b[32m\x1b[4mtext\x1b[0m",
		},
		{
			name:  "yellow bold italic",
			build: func() string { return ansi.New("text").Yellow().Bold().Italic().String() },
			want:  "\x1b[33m\x1b[1m\x1b[3mtext\x1b[0m",
		},
		{
			name:  "all styles",
			build: func() string { return ansi.New("text").Red().Bold().Dim().Italic().Underline().String() },
			want:  "\x1b[31m\x1b[1m\x1b[2m\x1b[3m\x1b[4mtext\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.build()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyle_NoStyle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"simple", "text", "text"},
		{"with spaces", "hello world", "hello world"},
		{"multiline", "line1\nline2", "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ansi.New(tt.input).String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStyle_EmptyText(t *testing.T) {
	// Even with styles, empty text should return empty or just codes
	result := ansi.New("").Red().Bold().String()

	// Should not panic
	if result == "" {
		// OK: empty text returns empty
		return
	}

	// OR: returns just the codes with reset
	if result == "\x1b[31m\x1b[1m\x1b[0m" {
		// Also OK: includes codes
		return
	}

	t.Errorf("unexpected result for empty styled text: %q", result)
}

func TestStyle_Chaining(t *testing.T) {
	// Verify chaining returns *Style for fluent API
	s := ansi.New("test")

	s1 := s.Red()
	if s1 != s {
		t.Error("Red() should return self for chaining")
	}

	s2 := s.Bold()
	if s2 != s {
		t.Error("Bold() should return self for chaining")
	}

	s3 := s.Underline()
	if s3 != s {
		t.Error("Underline() should return self for chaining")
	}
}

func TestStyle_MultipleColors(t *testing.T) {
	// Applying multiple colors should keep all (terminal will use last)
	result := ansi.New("text").Red().Green().Blue().String()

	// Should contain all color codes
	expected := "\x1b[31m\x1b[32m\x1b[34mtext\x1b[0m"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestStyle_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		style func(*ansi.Style) *ansi.Style
	}{
		{"newline", "line1\nline2", func(s *ansi.Style) *ansi.Style { return s.Red() }},
		{"tab", "col1\tcol2", func(s *ansi.Style) *ansi.Style { return s.Bold() }},
		{"unicode", "你好世界", func(s *ansi.Style) *ansi.Style { return s.Green() }},
		{"emoji", "🔥🚀💯", func(s *ansi.Style) *ansi.Style { return s.Yellow() }},
		{"mixed", "Hello 世界 🌍", func(s *ansi.Style) *ansi.Style { return s.Blue().Bold() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styled := tt.style(ansi.New(tt.text))
			result := styled.String()

			// Should contain original text
			plain := ansi.Strip(result)
			if plain != tt.text {
				t.Errorf("styled text doesn't preserve original: got %q, want %q", plain, tt.text)
			}

			// Should start with ANSI code
			if result[0] != '\x1b' {
				t.Errorf("styled text doesn't start with ANSI: %q", result)
			}

			// Should end with reset
			if !endsWithReset(result) {
				t.Errorf("styled text doesn't end with reset: %q", result)
			}
		})
	}
}

func endsWithReset(s string) bool {
	return len(s) >= 4 && s[len(s)-4:] == "\x1b[0m"
}

func TestStyle_LongText(t *testing.T) {
	// Test with long text to ensure no issues
	const longText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
		"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. " +
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris."

	result := ansi.New(longText).Red().Bold().String()

	// Should preserve entire text
	plain := ansi.Strip(result)
	if plain != longText {
		t.Error("Long text not preserved correctly")
	}

	// Should have proper formatting
	if !endsWithReset(result) {
		t.Error("Long text doesn't end with reset")
	}
}
