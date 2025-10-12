package strutil

import (
	"strings"
	"testing"
)

func BenchmarkSplitLines(b *testing.B) {
	text := strings.Repeat("line content here\n", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLines(text)
	}
}

func BenchmarkSplitLinesMixed(b *testing.B) {
	text := strings.Repeat("line\r\n", 500) + strings.Repeat("line\n", 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SplitLines(text)
	}
}

func BenchmarkJoinLines(b *testing.B) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "line content here"
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = JoinLines(lines)
	}
}

func BenchmarkTrimEmptyLines(b *testing.B) {
	lines := make([]string, 1000)
	for i := range lines {
		if i < 10 || i >= 990 {
			lines[i] = ""
		} else {
			lines[i] = "line content"
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TrimEmptyLines(lines)
	}
}

func BenchmarkDetectIndentation(b *testing.B) {
	text := strings.Repeat("    line\n        nested\n", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DetectIndentation(text)
	}
}

func BenchmarkNormalizeWhitespace(b *testing.B) {
	text := "  some   text  with   irregular    spacing  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NormalizeWhitespace(text)
	}
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	a := "The quick brown fox jumps over the lazy dog"
	c := "The quick brown dog jumps over the lazy fox"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LevenshteinDistance(a, c)
	}
}

func BenchmarkLevenshteinDistanceShort(b *testing.B) {
	a := "kitten"
	c := "sitting"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LevenshteinDistance(a, c)
	}
}

func BenchmarkLevenshteinDistanceLong(b *testing.B) {
	a := strings.Repeat("a", 100)
	c := strings.Repeat("b", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LevenshteinDistance(a, c)
	}
}

func BenchmarkSimilarity(b *testing.B) {
	a := "The quick brown fox"
	c := "The quick brown dog"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Similarity(a, c)
	}
}

func BenchmarkFuzzyMatch(b *testing.B) {
	query := "abc"
	target := "alphabet characters"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FuzzyMatch(query, target)
	}
}

func BenchmarkToSnakeCase(b *testing.B) {
	input := "MyVariableName"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToSnakeCase(input)
	}
}

func BenchmarkToCamelCase(b *testing.B) {
	input := "my_variable_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToCamelCase(input)
	}
}

func BenchmarkToPascalCase(b *testing.B) {
	input := "my_variable_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ToPascalCase(input)
	}
}

func BenchmarkNormalizeIndentation(b *testing.B) {
	text := strings.Repeat("    line\n        nested\n", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NormalizeIndentation(text, true, 4)
	}
}
