package strutil

import (
	"strings"
	"testing"
)

// TestSplitLines tests line splitting with various line ending formats
func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Unix LF",
			input: "a\nb\nc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Windows CRLF",
			input: "a\r\nb\r\nc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Old Mac CR",
			input: "a\rb\rc",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Mixed endings",
			input: "a\nb\r\nc\rd",
			want:  []string{"a", "b", "c", "d"},
		},
		{
			name:  "Empty string",
			input: "",
			want:  []string{""},
		},
		{
			name:  "No newlines",
			input: "abc",
			want:  []string{"abc"},
		},
		{
			name:  "Trailing newline",
			input: "a\nb\n",
			want:  []string{"a", "b", ""},
		},
		{
			name:  "Empty lines",
			input: "a\n\nb",
			want:  []string{"a", "", "b"},
		},
		{
			name:  "Only newlines",
			input: "\n\n\n",
			want:  []string{"", "", "", ""},
		},
		{
			name:  "Single line with LF",
			input: "single\n",
			want:  []string{"single", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitLines(tt.input)
			if !equalSlices(got, tt.want) {
				t.Errorf("SplitLines() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJoinLines tests line joining
func TestJoinLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{
			name:  "Multiple lines",
			input: []string{"a", "b", "c"},
			want:  "a\nb\nc",
		},
		{
			name:  "Single line",
			input: []string{"single"},
			want:  "single",
		},
		{
			name:  "Empty slice",
			input: []string{},
			want:  "",
		},
		{
			name:  "With empty lines",
			input: []string{"a", "", "b"},
			want:  "a\n\nb",
		},
		{
			name:  "Only empty strings",
			input: []string{"", "", ""},
			want:  "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinLines(tt.input)
			if got != tt.want {
				t.Errorf("JoinLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTrimEmptyLines tests trimming of leading and trailing empty lines
func TestTrimEmptyLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "Leading empty",
			input: []string{"", "", "a", "b"},
			want:  []string{"a", "b"},
		},
		{
			name:  "Trailing empty",
			input: []string{"a", "b", "", ""},
			want:  []string{"a", "b"},
		},
		{
			name:  "Both ends empty",
			input: []string{"", "a", "b", ""},
			want:  []string{"a", "b"},
		},
		{
			name:  "Middle empty preserved",
			input: []string{"a", "", "b"},
			want:  []string{"a", "", "b"},
		},
		{
			name:  "All empty",
			input: []string{"", "", ""},
			want:  []string{},
		},
		{
			name:  "No empty",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "Empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "Whitespace lines preserved",
			input: []string{"", " ", "a", " ", ""},
			want:  []string{" ", "a", " "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimEmptyLines(tt.input)
			if !equalSlices(got, tt.want) {
				t.Errorf("TrimEmptyLines() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectIndentation tests indentation detection
func TestDetectIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTabs bool
		wantSize int
	}{
		{
			name:     "Spaces 2",
			input:    "  a\n  b\n    c\n",
			wantTabs: false,
			wantSize: 2,
		},
		{
			name:     "Spaces 4",
			input:    "    a\n    b\n        c\n",
			wantTabs: false,
			wantSize: 4,
		},
		{
			name:     "Tabs",
			input:    "\ta\n\tb\n\t\tc\n",
			wantTabs: true,
			wantSize: 1,
		},
		{
			name:     "Mixed favor tabs",
			input:    "\ta\n  b\n\tc\n\td\n",
			wantTabs: true,
			wantSize: 1,
		},
		{
			name:     "Mixed favor spaces",
			input:    "    a\n    b\n\tc\n    d\n",
			wantTabs: false,
			wantSize: 4,
		},
		{
			name:     "Empty",
			input:    "",
			wantTabs: false,
			wantSize: 4, // default
		},
		{
			name:     "No indentation",
			input:    "a\nb\nc\n",
			wantTabs: false,
			wantSize: 4, // default
		},
		{
			name:     "Real Go code",
			input:    "func main() {\n\tfmt.Println(\"hello\")\n}\n",
			wantTabs: true,
			wantSize: 1,
		},
		{
			name:     "Real Python code",
			input:    "def main():\n    print('hello')\n    return 0\n",
			wantTabs: false,
			wantSize: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTabs, gotSize := DetectIndentation(tt.input)
			if gotTabs != tt.wantTabs || gotSize != tt.wantSize {
				t.Errorf("DetectIndentation() = (%v, %v), want (%v, %v)",
					gotTabs, gotSize, tt.wantTabs, tt.wantSize)
			}
		})
	}
}

// TestNormalizeWhitespace tests whitespace normalization
func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Multiple spaces",
			input: "a  b   c",
			want:  "a b c",
		},
		{
			name:  "Leading spaces",
			input: "  a b",
			want:  "a b",
		},
		{
			name:  "Trailing spaces",
			input: "a b  ",
			want:  "a b",
		},
		{
			name:  "Tabs",
			input: "a\tb\tc",
			want:  "a b c",
		},
		{
			name:  "Newlines",
			input: "a\nb\nc",
			want:  "a b c",
		},
		{
			name:  "Mixed whitespace",
			input: "  a \t\n b  \r\n  c  ",
			want:  "a b c",
		},
		{
			name:  "Empty",
			input: "",
			want:  "",
		},
		{
			name:  "Only whitespace",
			input: "   \t\n  ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTrimWhitespace tests whitespace trimming
func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Leading spaces",
			input: "  abc",
			want:  "abc",
		},
		{
			name:  "Trailing spaces",
			input: "abc  ",
			want:  "abc",
		},
		{
			name:  "Both ends",
			input: "  abc  ",
			want:  "abc",
		},
		{
			name:  "Internal preserved",
			input: "  a  b  c  ",
			want:  "a  b  c",
		},
		{
			name:  "Empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("TrimWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLevenshteinDistance tests edit distance calculation
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{
			name: "Empty strings",
			a:    "",
			b:    "",
			want: 0,
		},
		{
			name: "One empty",
			a:    "abc",
			b:    "",
			want: 3,
		},
		{
			name: "Other empty",
			a:    "",
			b:    "abc",
			want: 3,
		},
		{
			name: "Identical",
			a:    "abc",
			b:    "abc",
			want: 0,
		},
		{
			name: "One substitution",
			a:    "abc",
			b:    "abd",
			want: 1,
		},
		{
			name: "One insertion",
			a:    "abc",
			b:    "abcd",
			want: 1,
		},
		{
			name: "One deletion",
			a:    "abcd",
			b:    "abc",
			want: 1,
		},
		{
			name: "Kitten to sitting",
			a:    "kitten",
			b:    "sitting",
			want: 3,
		},
		{
			name: "Saturday to Sunday",
			a:    "Saturday",
			b:    "Sunday",
			want: 3,
		},
		{
			name: "Completely different",
			a:    "abc",
			b:    "def",
			want: 3,
		},
		{
			name: "Case sensitive",
			a:    "ABC",
			b:    "abc",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d",
					tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestSimilarity tests similarity ratio calculation
func TestSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{
			name: "Identical",
			a:    "abc",
			b:    "abc",
			want: 1.0,
		},
		{
			name: "Empty strings",
			a:    "",
			b:    "",
			want: 1.0,
		},
		{
			name: "Completely different",
			a:    "abc",
			b:    "def",
			want: 0.0,
		},
		{
			name: "One change",
			a:    "abc",
			b:    "abd",
			want: 0.666, // 2/3 ≈ 0.666
		},
		{
			name: "Kitten to sitting",
			a:    "kitten",
			b:    "sitting",
			want: 0.571, // (7-3)/7 ≈ 0.571
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Similarity(tt.a, tt.b)
			// Allow small floating point error
			if abs(got-tt.want) > 0.01 {
				t.Errorf("Similarity(%q, %q) = %.3f, want %.3f",
					tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestFuzzyMatch tests fuzzy matching score
func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		target string
		want   float64 // Approximate score
	}{
		{
			name:   "Exact match",
			query:  "abc",
			target: "abc",
			want:   100.0,
		},
		{
			name:   "Substring",
			query:  "abc",
			target: "xabcx",
			want:   80.0, // Contains
		},
		{
			name:   "Scattered",
			query:  "abc",
			target: "axbxc",
			want:   70.0, // Fuzzy match score
		},
		{
			name:   "No match",
			query:  "xyz",
			target: "abc",
			want:   0.0,
		},
		{
			name:   "Case insensitive",
			query:  "ABC",
			target: "abc",
			want:   100.0,
		},
		{
			name:   "Empty query",
			query:  "",
			target: "abc",
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FuzzyMatch(tt.query, tt.target)
			// Allow some variance in scoring
			if abs(got-tt.want) > 20.0 {
				t.Errorf("FuzzyMatch(%q, %q) = %.1f, want %.1f",
					tt.query, tt.target, got, tt.want)
			}
		})
	}
}

// TestToSnakeCase tests snake_case conversion
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "PascalCase",
			input: "MyVariableName",
			want:  "my_variable_name",
		},
		{
			name:  "camelCase",
			input: "myVariableName",
			want:  "my_variable_name",
		},
		{
			name:  "Already snake_case",
			input: "my_variable_name",
			want:  "my_variable_name",
		},
		{
			name:  "With numbers",
			input: "MyVariable123Name",
			want:  "my_variable123_name",
		},
		{
			name:  "Acronym",
			input: "HTTPServer",
			want:  "http_server",
		},
		{
			name:  "Single word",
			input: "Variable",
			want:  "variable",
		},
		{
			name:  "Empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("ToSnakeCase(%q) = %q, want %q",
					tt.input, got, tt.want)
			}
		})
	}
}

// TestToCamelCase tests camelCase conversion
func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "snake_case",
			input: "my_variable_name",
			want:  "myVariableName",
		},
		{
			name:  "Already camelCase",
			input: "myVariableName",
			want:  "myVariableName",
		},
		{
			name:  "PascalCase",
			input: "MyVariableName",
			want:  "myVariableName",
		},
		{
			name:  "With numbers",
			input: "my_variable_123",
			want:  "myVariable123",
		},
		{
			name:  "Single word",
			input: "variable",
			want:  "variable",
		},
		{
			name:  "Empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("ToCamelCase(%q) = %q, want %q",
					tt.input, got, tt.want)
			}
		})
	}
}

// TestToPascalCase tests PascalCase conversion
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "snake_case",
			input: "my_variable_name",
			want:  "MyVariableName",
		},
		{
			name:  "camelCase",
			input: "myVariableName",
			want:  "MyVariableName",
		},
		{
			name:  "Already PascalCase",
			input: "MyVariableName",
			want:  "MyVariableName",
		},
		{
			name:  "With numbers",
			input: "my_variable_123",
			want:  "MyVariable123",
		},
		{
			name:  "Single word",
			input: "variable",
			want:  "Variable",
		},
		{
			name:  "Empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("ToPascalCase(%q) = %q, want %q",
					tt.input, got, tt.want)
			}
		})
	}
}

// Helper functions

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestNormalizeIndentation tests indentation normalization
func TestNormalizeIndentation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		useTabs  bool
		size     int
		want     string
	}{
		{
			name:     "Spaces to tabs",
			input:    "    line1\n        line2\n",
			useTabs:  true,
			size:     4,
			want:     "\tline1\n\t\tline2\n",
		},
		{
			name:     "Tabs to spaces",
			input:    "\tline1\n\t\tline2\n",
			useTabs:  false,
			size:     4,
			want:     "    line1\n        line2\n",
		},
		{
			name:     "Already correct",
			input:    "\tline1\n\t\tline2\n",
			useTabs:  true,
			size:     1,
			want:     "\tline1\n\t\tline2\n",
		},
		{
			name:     "Empty",
			input:    "",
			useTabs:  true,
			size:     4,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeIndentation(tt.input, tt.useTabs, tt.size)
			if got != tt.want {
				t.Errorf("NormalizeIndentation() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Edge case tests

func TestSplitLinesVeryLong(t *testing.T) {
	// Test with a very long input
	input := strings.Repeat("line\n", 10000)
	lines := SplitLines(input)
	if len(lines) != 10001 { // 10000 lines + 1 empty at end
		t.Errorf("SplitLines() long input: got %d lines, want 10001", len(lines))
	}
}

func TestLevenshteinDistanceVeryLong(t *testing.T) {
	// Test with long strings
	a := strings.Repeat("a", 100)
	b := strings.Repeat("b", 100)
	dist := LevenshteinDistance(a, b)
	if dist != 100 {
		t.Errorf("LevenshteinDistance() long strings: got %d, want 100", dist)
	}
}
