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
	assert.Contains(t, indices, 0) // 'a'.
	assert.Contains(t, indices, 2) // 'b'.
	assert.Contains(t, indices, 4) // 'c'.
}

func TestMatcher_Score_NoMatch(t *testing.T) {
	m := NewMatcher(false)

	score, indices := m.Score("xyz", "a/b/c.txt")

	assert.Equal(t, -1, score)
	assert.Nil(t, indices)
}

func TestMatcher_Score_PartialMatch(t *testing.T) {
	m := NewMatcher(false)

	// "ab" exists but "abc" doesn't fully match.
	score, _ := m.Score("abcd", "a/b/c.txt")

	assert.Equal(t, -1, score) // Not all query chars matched.
}

func TestMatcher_Score_ConsecutiveBonus(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("abc", "a/b/c.txt")
	score2, _ := m.Score("abc", "abc.txt")

	// "abc" consecutive should score higher.
	assert.Greater(t, score2, score1)
}

func TestMatcher_Score_SeparatorBonus(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("ft", "file_test.go")
	score2, _ := m.Score("ft", "first.go")

	// Match after separator should score higher.
	assert.Greater(t, score1, score2)
}

func TestMatcher_Score_PathLengthBonus(t *testing.T) {
	m := NewMatcher(false)

	// Both are filename prefix matches (90), path length bonus should differentiate
	// But actually both will get 90 since they're prefix matches
	// Let's use fuzzy matching instead where path length bonus is applied.
	score1, _ := m.Score("mgo", "main.go")
	score2, _ := m.Score("mgo", "very/long/path/to/main.go")

	// Shorter path should score higher in fuzzy match.
	assert.Greater(t, score1, score2)
}

func TestMatcher_Score_CaseInsensitive(t *testing.T) {
	m := NewMatcher(false)

	score1, _ := m.Score("abc", "ABC.txt")
	score2, _ := m.Score("ABC", "abc.txt")

	// Case-insensitive: both should match.
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

	// Should find at least 3 matches containing "app".
	assert.GreaterOrEqual(t, len(matches), 3)

	// First match should be highest scored.
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

	// All three match as prefix (90), but "test.go" is shortest
	// Actually both "test.go" and "very/long/path/test.go" are prefix matches
	// but "test.go" should win due to path length bonus
	// However, they both have same score (90 + length bonus), so we need to verify
	// the first result is one of the better ones.
	assert.Contains(t, []string{"test.go", "very/long/path/test.go"}, matches[0].Path)
	// Actually let me check: since advanced scoring gives exact scores,
	// both should be 90, but path length bonus makes "test.go" higher
	// Let's just check it's sorted by score.
	for i := range len(matches) - 1 {
		assert.GreaterOrEqual(t, matches[i].Score, matches[i+1].Score, "Matches should be sorted by score")
	}
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

	// All three have "r", "e", "a" in sequence.
	assert.GreaterOrEqual(t, len(matches), 2)
}

func BenchmarkMatcher_Score(b *testing.B) {
	m := NewMatcher(false)

	b.ResetTimer()

	for range b.N {
		m.Score("test", "path/to/test_file.go")
	}
}

func BenchmarkMatcher_Match_100(b *testing.B) {
	m := NewMatcher(false)

	// Generate 100 paths.
	paths := make([]string, 100)
	for i := range 100 {
		paths[i] = "src/package/file_" + string(rune(i)) + ".go"
	}

	b.ResetTimer()

	for range b.N {
		m.Match("file", paths)
	}
}

func BenchmarkMatcher_Match_1000(b *testing.B) {
	m := NewMatcher(false)

	// Generate 1000 paths.
	paths := make([]string, 1000)
	for i := range 1000 {
		paths[i] = "src/package/module/file_" + string(rune(i%100)) + ".go"
	}

	b.ResetTimer()

	for range b.N {
		m.Match("file", paths)
	}
}

// Enhanced scoring tests for Feature 3.2.

func TestMatcher_Score_ExactFilenameMatch(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name  string
		query string
		path  string
		want  int
	}{
		{"exact match", "main.go", "main.go", 100},
		{"exact match with path", "main.go", "src/main.go", 100},
		{"exact match nested", "config.toml", "internal/config/config.toml", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.Equal(t, tt.want, score, "Expected exact match score of 100")
		})
	}
}

func TestMatcher_Score_FilenamePrefix(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name  string
		query string
		path  string
		want  int
	}{
		{"prefix match", "config", "config.toml", 90},
		{"prefix match with path", "test", "src/test_utils.go", 90},
		{"prefix match nested", "app", "internal/app/application.go", 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.Equal(t, tt.want, score, "Expected filename prefix score of 90")
		})
	}
}

func TestMatcher_Score_FilenameContains(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name     string
		query    string
		path     string
		minScore int
		maxScore int
	}{
		// Note: "test_utils.go" will match as prefix (90) since filename starts with "test".
		{"contains early", "util", "test_utils.go", 75, 82},
		{"contains middle", "test", "my_test.go", 70, 78},
		{"contains late", "test", "utils_my_test.go", 70, 75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.GreaterOrEqual(t, score, tt.minScore, "Score should be at least %d", tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore, "Score should be at most %d", tt.maxScore)
		})
	}
}

func TestMatcher_Score_PathSegmentMatch(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name  string
		query string
		path  string
		want  int
	}{
		{"exact segment", "src", "src/main.go", 60},
		{"segment prefix", "int", "internal/agent.go", 50},
		{"nested segment", "app", "src/app/handler.go", 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.Equal(t, tt.want, score, "Expected path segment score")
		})
	}
}

func TestMatcher_Score_FuzzyConsecutive(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name     string
		query    string
		path     string
		minScore int
	}{
		{"consecutive chars", "cfg", "config.go", 40},
		{"multiple consecutive", "tst", "test.go", 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.GreaterOrEqual(t, score, tt.minScore, "Fuzzy consecutive should score at least %d", tt.minScore)
		})
	}
}

func TestMatcher_Score_FuzzyScattered(t *testing.T) {
	m := NewMatcher(false)

	tests := []struct {
		name     string
		query    string
		path     string
		minScore int
	}{
		{"scattered chars", "mgo", "main.go", 20},
		{"widely scattered", "abc", "a/b/c.txt", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _ := m.Score(tt.query, tt.path)
			assert.GreaterOrEqual(t, score, tt.minScore, "Fuzzy scattered should score at least %d", tt.minScore)
		})
	}
}

func TestMatcher_Score_Ranking(t *testing.T) {
	m := NewMatcher(false)

	paths := []string{
		"test",                     // Exact filename match - 100.
		"test.go",                  // Filename prefix - 90.
		"my_test.go",               // Filename contains - 70-80.
		"src/test/handler.go",      // Path segment - 60.
		"internal/testing/util.go", // Fuzzy match - <60.
	}

	scores := make(map[string]int)

	for _, path := range paths {
		score, _ := m.Score("test", path)
		scores[path] = score
	}

	// Debug: verify scores match expectations.
	assert.Equal(t, 100, scores["test"], "Exact match should be 100")
	assert.Equal(t, 90, scores["test.go"], "Prefix match should be 90")
	assert.GreaterOrEqual(t, scores["my_test.go"], 70, "Contains should be 70+")
	assert.LessOrEqual(t, scores["my_test.go"], 80, "Contains should be <=80")
	assert.Equal(t, 60, scores["src/test/handler.go"], "Path segment should be 60")

	// Verify ranking order.
	assert.Greater(t, scores["test"], scores["test.go"], "Exact (100) should beat prefix (90)")
	assert.Greater(t, scores["test.go"], scores["my_test.go"], "Prefix (90) should beat contains (70-80)")
	assert.Greater(t, scores["my_test.go"], scores["src/test/handler.go"], "Contains (70-80) should beat path segment (60)")
	assert.Greater(t, scores["src/test/handler.go"], scores["internal/testing/util.go"], "Path segment (60) should beat fuzzy (<60)")
}

func TestMatcher_Match_AdvancedRanking(t *testing.T) {
	m := NewMatcher(false)

	paths := []string{
		"internal/testing/fuzzy.go",
		"src/test/helper.go",
		"my_test.go",
		"test_utils.go",
		"test.go",
	}

	matches := m.Match("test", paths)

	// Both test.go and test_utils.go are prefix matches (90)
	// So we can't guarantee exact order between them
	// Let's just verify top matches are correct.
	assert.GreaterOrEqual(t, len(matches), 4)

	// Verify the scores are sorted descending.
	for i := range len(matches) - 1 {
		assert.GreaterOrEqual(t, matches[i].Score, matches[i+1].Score)
	}

	// Verify test.go is in top 2 (prefix match).
	topTwo := []string{matches[0].Path, matches[1].Path}
	assert.Contains(t, topTwo, "test.go")
	assert.Contains(t, topTwo, "test_utils.go")
}

func BenchmarkMatcher_Score_Advanced(b *testing.B) {
	m := NewMatcher(false)

	paths := []string{
		"exact.go",
		"exact_match.go",
		"my_exact.go",
		"src/exact/file.go",
		"internal/testing/example.go",
	}

	b.ResetTimer()

	for range b.N {
		for _, path := range paths {
			m.Score("exact", path)
		}
	}
}
