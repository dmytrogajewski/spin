package filesearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMatcher(t *testing.T) {
	m := NewMatcher(false)

	assert.NotNil(t, m)
	assert.False(t, m.caseSensitive)
}

func TestNewMatcher_CaseSensitive(t *testing.T) {
	m := NewMatcher(true)

	assert.NotNil(t, m)
	assert.True(t, m.caseSensitive)
}

func TestMatcher_Score_EmptyQuery(t *testing.T) {
	m := NewMatcher(false)

	score, indices := m.Score("", "path/to/file.go")

	assert.Equal(t, 0, score)
	assert.Nil(t, indices)
}

func TestMatcher_Score_SimpleMatch(t *testing.T) {
	m := NewMatcher(false)

	score, indices := m.Score("abc", "a/b/c.txt")

	assert.Greater(t, score, 0)
	assert.Len(t, indices, 3)
	assert.Contains(t, indices, 0) // 'a'
	assert.Contains(t, indices, 2) // 'b'
	assert.Contains(t, indices, 4) // 'c'
}

func TestMatcher_Score_NoMatch(t *testing.T) {
	m := NewMatcher(false)

	score, indices := m.Score("xyz", "a/b/c.txt")

	assert.Equal(t, -1, score)
	assert.Nil(t, indices)
}

func TestMatcher_Score_PartialMatch(t *testing.T) {
	m := NewMatcher(false)

	// "ab" exists but "abc" doesn't fully match
	score, _ := m.Score("abcd", "a/b/c.txt")

	assert.Equal(t, -1, score) // Not all query chars matched
}

func TestMatcher_Score_ConsecutiveBonus(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("abc", "a/b/c.txt")
	score2, _ := m.Score("abc", "abc.txt")

	// "abc" consecutive should score higher
	assert.Greater(t, score2, score1)
}

func TestMatcher_Score_SeparatorBonus(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("ft", "file_test.go")
	score2, _ := m.Score("ft", "first.go")

	// Match after separator should score higher
	assert.Greater(t, score1, score2)
}

func TestMatcher_Score_PathLengthBonus(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("f", "file.go")
	score2, _ := m.Score("f", "very/long/path/to/file.go")

	// Shorter path should score higher
	assert.Greater(t, score1, score2)
}

func TestMatcher_Score_CaseInsensitive(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("abc", "ABC.txt")
	score2, _ := m.Score("ABC", "abc.txt")

	// Case-insensitive: both should match
	assert.Greater(t, score1, 0)
	assert.Greater(t, score2, 0)
}

func TestMatcher_Match_EmptyQuery(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{"file1.go", "file2.go"}

	matches := m.Match("", paths)

	assert.Len(t, matches, 0)
}

func TestMatcher_Match_EmptyPaths(t *testing.T) {
	m := NewMatcher(false)

	matches := m.Match("test", []string{})

	assert.Len(t, matches, 0)
}

func TestMatcher_Match_SinglePath(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{"test.go"}

	matches := m.Match("test", paths)

	assert.Len(t, matches, 1)
	assert.Equal(t, "test.go", matches[0].Path)
	assert.Greater(t, matches[0].Score, 0)
}

func TestMatcher_Match_MultiplePaths(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{
		"src/app.go",
		"src/main.go",
		"internal/app/app.go",
		"test/app_test.go",
	}

	matches := m.Match("app", paths)

	// Should find at least 3 matches containing "app"
	assert.GreaterOrEqual(t, len(matches), 3)

	// First match should be highest scored
	assert.Equal(t, "src/app.go", matches[0].Path)
}

func TestMatcher_Match_Sorting(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{
		"very/long/path/test.go",
		"test.go",
		"t/e/s/t.go",
	}

	matches := m.Match("test", paths)

	// Shortest path with best match should be first
	assert.Equal(t, "test.go", matches[0].Path)
}

func TestMatcher_Match_NoMatches(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{
		"file1.go",
		"file2.go",
	}

	matches := m.Match("xyz", paths)

	assert.Len(t, matches, 0)
}

func TestMatcher_Match_PartialMatches(t *testing.T) {
	m := NewMatcher(false)
	paths := []string{
		"readme.md",
		"internal/reader/reader.go",
		"src/reactive.go",
	}

	matches := m.Match("rea", paths)

	// All three have "r", "e", "a" in sequence
	assert.GreaterOrEqual(t, len(matches), 2)
}

func BenchmarkMatcher_Score(b *testing.B) {
	m := NewMatcher(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Score("test", "path/to/test_file.go")
	}
}

func BenchmarkMatcher_Match_100(b *testing.B) {
	m := NewMatcher(false)

	// Generate 100 paths
	paths := make([]string, 100)
	for i := 0; i < 100; i++ {
		paths[i] = "src/package/file_" + string(rune(i)) + ".go"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("file", paths)
	}
}

func BenchmarkMatcher_Match_1000(b *testing.B) {
	m := NewMatcher(false)

	// Generate 1000 paths
	paths := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		paths[i] = "src/package/module/file_" + string(rune(i%100)) + ".go"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("file", paths)
	}
}
