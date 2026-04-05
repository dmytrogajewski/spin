package patchapply

import (
	"strings"
	"testing"
)

// matcherTestCase defines a test case for FindContext tests.
type matcherTestCase struct {
	name         string
	fileLines    []string
	contextLines []string
	header       string
	threshold    float64
	want         int
}

// runMatcherTests runs a slice of matcher test cases.
func runMatcherTests(t *testing.T, tests []matcherTestCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMatcher(tt.fileLines)
			if tt.threshold != 0 {
				m.threshold = tt.threshold
			}

			got := m.FindContext(tt.contextLines, tt.header)
			if got != tt.want {
				t.Errorf("FindContext() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fileLines []string
	}{
		{
			name:      "empty file",
			fileLines: []string{},
		},
		{
			name:      "single line",
			fileLines: []string{"hello world"},
		},
		{
			name: "multiple lines",
			fileLines: []string{
				"package main",
				"",
				"func main() {",
				"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMatcher(tt.fileLines)
			if m == nil {
				t.Fatal("NewMatcher() returned nil")
			}

			if m.threshold != 0.85 {
				t.Errorf("NewMatcher() threshold = %.2f, want 0.85", m.threshold)
			}

			if len(m.fileLines) != len(tt.fileLines) {
				t.Errorf("NewMatcher() fileLines length = %d, want %d", len(m.fileLines), len(tt.fileLines))
			}

			if len(m.normalizedLines) != len(tt.fileLines) {
				t.Errorf("NewMatcher() normalizedLines length = %d, want %d", len(m.normalizedLines), len(tt.fileLines))
			}
		})
	}
}

func TestMatcher_FindContext_ExactMatch(t *testing.T) {
	t.Parallel()

	runMatcherTests(t, []matcherTestCase{
		{
			name:      "empty context matches at start",
			fileLines: []string{"line1", "line2"}, contextLines: []string{}, want: 0,
		},
		{
			name:         "exact match at start",
			fileLines:    []string{"func main() {", "    fmt.Println(\"hello\")", "}"},
			contextLines: []string{"func main() {", "    fmt.Println(\"hello\")"}, want: 0,
		},
		{
			name:         "exact match in middle",
			fileLines:    []string{"package main", "", "func main() {", "    return nil", "}"},
			contextLines: []string{"func main() {", "    return nil"}, want: 2,
		},
		{
			name:         "exact match at end",
			fileLines:    []string{"package main", "", "func main() {"},
			contextLines: []string{"func main() {"}, want: 2,
		},
		{
			name:         "single line exact match",
			fileLines:    []string{"line1", "line2", "line3"},
			contextLines: []string{"line2"}, want: 1,
		},
		{
			name:         "multi-line exact match",
			fileLines:    []string{"line1", "line2", "line3", "line4"},
			contextLines: []string{"line2", "line3"}, want: 1,
		},
	})
}

func TestMatcher_FindContext_FuzzyMatch(t *testing.T) {
	t.Parallel()

	runMatcherTests(t, []matcherTestCase{
		{
			name:         "whitespace variation - tabs vs spaces",
			fileLines:    []string{"func main() {", "  fmt.Println(\"hello\")", "}"},
			contextLines: []string{"func main() {", "    fmt.Println(\"hello\")"},
			threshold:    0.85, want: 0,
		},
		{
			name:         "trailing whitespace",
			fileLines:    []string{"func foo() {   ", "}"},
			contextLines: []string{"func foo() {", "}"}, threshold: 0.85, want: 0,
		},
		{
			name:         "leading whitespace",
			fileLines:    []string{"   func bar() {", "}"},
			contextLines: []string{"func bar() {", "}"}, threshold: 0.85, want: 0,
		},
		{
			name:         "minor text difference within threshold",
			fileLines:    []string{"func Calculate(a, b int) int {", "    return a + b", "}"},
			contextLines: []string{"func Calculate(x, y int) int {", "    return x + y"},
			threshold:    0.80, want: 0,
		},
		{
			name:         "below threshold - no match",
			fileLines:    []string{"func totally_different() {", "}"},
			contextLines: []string{"func original() {"}, threshold: 0.85, want: -1,
		},
		{
			name:         "mixed whitespace - tabs and spaces",
			fileLines:    []string{"func test() {", "\t\treturn 0"},
			contextLines: []string{"func test() {", "        return 0"},
			threshold:    0.85, want: 0,
		},
	})
}

func TestMatcher_FindContext_HeaderMatching(t *testing.T) {
	t.Parallel()

	runMatcherTests(t, []matcherTestCase{
		{
			name: "header helps disambiguate - exact function name",
			fileLines: []string{
				"func ProcessA(x int) {", "    return x + 1", "}",
				"", "func ProcessB(x int) {", "    return x + 1", "}",
			},
			contextLines: []string{"    return x + 1"},
			header:       "func ProcessB", want: 5,
		},
		{
			name: "header helps disambiguate - partial match",
			fileLines: []string{
				"// Handler for GET requests", "func handleGet() {",
				"    return nil", "}", "",
				"// Handler for POST requests", "func handlePost() {",
				"    return nil", "}",
			},
			contextLines: []string{"    return nil"}, header: "POST", want: 7,
		},
		{
			name:         "header not found - fallback to full search",
			fileLines:    []string{"func main() {", "    return 0", "}"},
			contextLines: []string{"    return 0"},
			header:       "func nonexistent", want: 1,
		},
		{
			name:         "empty header - full search",
			fileLines:    []string{"line1", "line2", "line3"},
			contextLines: []string{"line2"}, want: 1,
		},
		{
			name: "header matches but context doesn't - fallback",
			fileLines: []string{
				"func ProcessA(x int) {", "    return x + 1", "}",
				"", "func ProcessB(y int) {", "    return y * 2", "}",
			},
			contextLines: []string{"    return x + 1"},
			header:       "func ProcessB", want: 1,
		},
		{
			name: "multiple header occurrences - use first",
			fileLines: []string{
				"// Process data", "func Process(x int) {",
				"    return x + 1", "}", "",
				"// Process batch", "func ProcessBatch() {",
				"    // Call Process", "    return Process(0)", "}",
			},
			contextLines: []string{"    return x + 1"},
			header:       "Process", want: 2,
		},
	})
}

func TestMatcher_FindContext_EdgeCases(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("x", 1000)

	runMatcherTests(t, []matcherTestCase{
		{
			name:      "empty file - empty context",
			fileLines: []string{}, contextLines: []string{}, want: 0,
		},
		{
			name:      "empty file - non-empty context",
			fileLines: []string{}, contextLines: []string{"something"}, want: -1,
		},
		{
			name:         "context larger than file",
			fileLines:    []string{"line1", "line2"},
			contextLines: []string{"line1", "line2", "line3", "line4"}, want: -1,
		},
		{
			name:         "unicode content",
			fileLines:    []string{"func Process() {", "    msg := \"Hello, 世界\"", "}"},
			contextLines: []string{"    msg := \"Hello, 世界\""}, want: 1,
		},
		{
			name:         "very long line",
			fileLines:    []string{"short line", longLine, "another short line"},
			contextLines: []string{longLine}, want: 1,
		},
		{
			name:         "special characters",
			fileLines:    []string{"if (a && b) || (c && d) {", "    return true", "}"},
			contextLines: []string{"if (a && b) || (c && d) {"}, want: 0,
		},
		{
			name:         "regex special characters",
			fileLines:    []string{"pattern := `^[a-zA-Z0-9]+$`", "match := regexp.MatchString(pattern, input)"},
			contextLines: []string{"pattern := `^[a-zA-Z0-9]+$`"}, want: 0,
		},
		{
			name:         "empty lines",
			fileLines:    []string{"", "", "func test() {", "}"},
			contextLines: []string{"", "func test() {"}, want: 1,
		},
		{
			name:         "context at exact file end",
			fileLines:    []string{"line1", "line2", "line3"},
			contextLines: []string{"line3"}, want: 2,
		},
		{
			name:         "context would overflow file end",
			fileLines:    []string{"line1", "line2"},
			contextLines: []string{"line2", "line3"}, want: -1,
		},
	})
}

func TestMatcher_FindContext_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	runMatcherTests(t, []matcherTestCase{
		{
			name: "go code - function signature changed slightly",
			fileLines: []string{
				"package handler", "", "import \"context\"", "",
				"// Process handles incoming requests",
				"func Process(ctx context.Context, req *Request) error {",
				"    if req == nil {", "        return ErrNilRequest",
				"    }", "    return process(ctx, req)", "}",
			},
			contextLines: []string{
				"func Process(ctx context.Context, req Request) error {",
				"    if req == nil {",
			},
			header: "func Process", threshold: 0.80, want: 5,
		},
		{
			name:         "formatted code - indentation changed",
			fileLines:    []string{"type Config struct {", "  Host string", "  Port int", "  Timeout int64", "}"},
			contextLines: []string{"type Config struct {", "    Host string", "    Port int", "    Timeout int64"},
			header:       "type Config", threshold: 0.85, want: 0,
		},
		{
			name:         "comment added between code",
			fileLines:    []string{"func Calculate(x, y int) int {", "    // TODO: add validation", "    return x + y", "}"},
			contextLines: []string{"func Calculate(x, y int) int {", "    return x + y"},
			header:       "func Calculate", threshold: 0.70, want: -1,
		},
		{
			name: "multiple similar functions - use header",
			fileLines: []string{
				"func (s *Service) Create(data Data) error {",
				"    return s.repo.Insert(data)", "}", "",
				"func (s *Service) Update(data Data) error {",
				"    return s.repo.Update(data)", "}", "",
				"func (s *Service) Delete(id ID) error {",
				"    return s.repo.Delete(id)", "}",
			},
			contextLines: []string{"    return s.repo.Delete(id)"},
			header:       "Delete", threshold: 0.85, want: 9,
		},
	})
}

// Benchmark tests.
func BenchmarkMatcher_FindContext_ExactMatch_Small(b *testing.B) {
	fileLines := generateTestLines(100)
	contextLines := fileLines[50:55] // 5 lines of exact context.
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "")
	}
}

func BenchmarkMatcher_FindContext_ExactMatch_1kLines(b *testing.B) {
	fileLines := generateTestLines(1000)
	contextLines := fileLines[500:505]
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "")
	}
}

func BenchmarkMatcher_FindContext_ExactMatch_10kLines(b *testing.B) {
	fileLines := generateTestLines(10000)
	contextLines := fileLines[5000:5005]
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "")
	}
}

func BenchmarkMatcher_FindContext_FuzzyMatch_Small(b *testing.B) {
	fileLines := generateTestLines(100)
	contextLines := generateSimilarLines(fileLines[50:55], 0.90)
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "")
	}
}

func BenchmarkMatcher_FindContext_FuzzyMatch_10kLines(b *testing.B) {
	fileLines := generateTestLines(10000)
	contextLines := generateSimilarLines(fileLines[5000:5005], 0.90)
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "")
	}
}

func BenchmarkMatcher_FindContext_WithHeader_10kLines(b *testing.B) {
	fileLines := generateTestLines(10000)
	fileLines[5000] = "func TargetFunction() {" // Distinctive header.
	contextLines := fileLines[5001:5005]
	m := NewMatcher(fileLines)

	b.ResetTimer()

	for range b.N {
		m.FindContext(contextLines, "TargetFunction")
	}
}

// Helper functions for tests.

// generateTestLines generates n lines of realistic Go code.
func generateTestLines(n int) []string {
	lines := make([]string, n)
	templates := []string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func process() error {",
		"    return nil",
		"}",
		"",
		"type Data struct {",
		"    ID   int",
		"    Name string",
		"}",
	}

	for i := range n {
		lines[i] = templates[i%len(templates)]
	}

	return lines
}

// generateSimilarLines generates lines similar to the input with given similarity ratio.
func generateSimilarLines(original []string, similarity float64) []string {
	similar := make([]string, len(original))
	for i, line := range original {
		if similarity >= 1.0 {
			similar[i] = line
		} else {
			// Add some variation by changing whitespace.
			similar[i] = "  " + strings.TrimSpace(line) + "  "
		}
	}

	return similar
}
