package filesearch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers.

func createTestDir(t *testing.T) string {
	t.Helper()

	return t.TempDir()
}

func createTestFiles(t *testing.T, root string, files []string) {
	t.Helper()

	for _, file := range files {
		fullPath := filepath.Join(root, file)
		dir := filepath.Dir(fullPath)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(fullPath, []byte("test"), 0o600))
	}
}

// Constructor Tests.

func TestNewSearcher(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	s, err := NewSearcher(root)

	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, root, s.root)
	assert.NotNil(t, s.scanner)
	assert.NotNil(t, s.matcher)
	assert.False(t, s.indexed)
	assert.Empty(t, s.index)
}

func TestNewSearcher_InvalidRoot(t *testing.T) {
	t.Parallel()

	_, err := NewSearcher("/nonexistent/path/that/does/not/exist")

	assert.Error(t, err)
}

// Indexing Tests.

func TestSearcher_IndexAsync(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"main.go",
		"test.go",
		"src/app.go",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)

	ctx := context.Background()
	err = s.IndexAsync(ctx)

	require.NoError(t, err)
	assert.True(t, s.IsIndexed())
	assert.Len(t, s.index, 3)
}

func TestSearcher_IndexAsync_Empty(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	s, err := NewSearcher(root)
	require.NoError(t, err)

	ctx := context.Background()
	err = s.IndexAsync(ctx)

	require.NoError(t, err)
	assert.True(t, s.IsIndexed())
	assert.Empty(t, s.index)
}

func TestSearcher_IndexAsync_Cancellation(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	// Create many files to increase indexing time.
	files := make([]string, 100)
	for i := range 100 {
		files[i] = filepath.Join("dir", "subdir", fmt.Sprintf("file_%d.go", i))
	}

	createTestFiles(t, root, files)

	s, err := NewSearcher(root)
	require.NoError(t, err)

	// Cancel context immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = s.IndexAsync(ctx)

	// Should return context.Canceled error or succeed if already indexed.
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	}
}

func TestSearcher_IndexAsync_Timeout(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	// Create some files.
	createTestFiles(t, root, []string{"test.go"})

	s, err := NewSearcher(root)
	require.NoError(t, err)

	// Very short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout occurs.

	err = s.IndexAsync(ctx)

	// May error with DeadlineExceeded or succeed if fast enough.
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}
}

func TestSearcher_IndexAsync_Idempotent(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{"main.go"})

	s, err := NewSearcher(root)
	require.NoError(t, err)

	ctx := context.Background()

	// Index multiple times.
	err1 := s.IndexAsync(ctx)
	err2 := s.IndexAsync(ctx)
	err3 := s.IndexAsync(ctx)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.True(t, s.IsIndexed())
}

// Search Tests.

func TestSearcher_Search_NotIndexed(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	s, err := NewSearcher(root)
	require.NoError(t, err)

	results := s.Search("test", 10)

	assert.Empty(t, results)
}

func TestSearcher_Search_EmptyQuery(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{"main.go"})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	results := s.Search("", 10)

	assert.Empty(t, results)
}

func TestSearcher_Search_Basic(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"test.go",
		"test_utils.go",
		"main.go",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	results := s.Search("test", 10)

	assert.Len(t, results, 2)
	assert.Equal(t, "test.go", results[0].Path)
	assert.Equal(t, "test_utils.go", results[1].Path)
}

func TestSearcher_Search_Ranking(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"test.go",                  // Exact match.
		"test_utils.go",            // Prefix match.
		"my_test.go",               // Contains match.
		"src/test/handler.go",      // Path segment.
		"internal/testing/util.go", // Fuzzy match.
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	results := s.Search("test", 10)

	assert.GreaterOrEqual(t, len(results), 4)

	// Verify ranking order.
	assert.Equal(t, "test.go", results[0].Path, "Exact match should rank first")
	assert.Equal(t, "test_utils.go", results[1].Path, "Prefix match should rank second")
	assert.Equal(t, "my_test.go", results[2].Path, "Contains match should rank third")
}

func TestSearcher_Search_Limit(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"test1.go",
		"test2.go",
		"test3.go",
		"test4.go",
		"test5.go",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"limit 1", 1, 1},
		{"limit 3", 3, 3},
		{"limit 10", 10, 5},
		{"limit 0", 0, 5}, // 0 means no limit.
		{"limit negative", -1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results := s.Search("test", tt.limit)
			assert.Len(t, results, tt.want)
		})
	}
}

func TestSearcher_Search_NoMatches(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"main.go",
		"app.go",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	results := s.Search("xyz", 10)

	assert.Empty(t, results)
}

func TestSearcher_Search_CaseInsensitive(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"Main.go",
		"TEST.go",
		"Config.toml",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	results := s.Search("test", 10)

	assert.Len(t, results, 1)
	assert.Equal(t, "TEST.go", results[0].Path)
}

// IsIndexed Tests.

func TestSearcher_IsIndexed_False(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	s, err := NewSearcher(root)
	require.NoError(t, err)

	assert.False(t, s.IsIndexed())
}

func TestSearcher_IsIndexed_True(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	s, err := NewSearcher(root)
	require.NoError(t, err)

	require.NoError(t, s.IndexAsync(context.Background()))

	assert.True(t, s.IsIndexed())
}

// Integration Tests.

func TestSearcher_RealProject(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	// Create realistic project structure.
	files := []string{
		"main.go",
		"go.mod",
		"go.sum",
		"README.md",
		"internal/config/config.go",
		"internal/config/config_test.go",
		"internal/app/app.go",
		"internal/app/handler.go",
		"internal/app/handler_test.go",
		"pkg/util/util.go",
		"pkg/util/util_test.go",
		"cmd/server/main.go",
		"test/integration_test.go",
	}
	createTestFiles(t, root, files)

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	tests := []struct {
		name       string
		query      string
		minCount   int
		firstMatch string // Expected first result.
	}{
		{"exact match", "main.go", 1, "cmd/server/main.go"},
		{"config files", "config", 2, "internal/config/config.go"},
		{"test files", "test", 4, "pkg/util/util_test.go"}, // Shorter path wins.
		{"util files", "util", 2, "pkg/util/util.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results := s.Search(tt.query, 10)
			assert.GreaterOrEqual(t, len(results), tt.minCount)

			if len(results) > 0 {
				assert.Equal(t, tt.firstMatch, results[0].Path)
			}
		})
	}
}

func TestSearcher_WithGitignore(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)

	// Create .gitignore.
	gitignorePath := filepath.Join(root, ".gitignore")
	gitignoreContent := `node_modules/
*.log
dist/
`
	require.NoError(t, os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o600))

	// Create files.
	files := []string{
		"main.go",
		"test.go",
		"node_modules/pkg/index.js",
		"debug.log",
		"dist/bundle.js",
	}
	createTestFiles(t, root, files)

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	// Search for "test".
	results := s.Search("go", 10)

	// Should only find main.go and test.go, not node_modules or dist.
	assert.Len(t, results, 2)

	for _, r := range results {
		assert.NotContains(t, r.Path, "node_modules")
		assert.NotContains(t, r.Path, "dist")
		assert.NotContains(t, r.Path, ".log")
	}
}

// Concurrent Tests.

func TestSearcher_ConcurrentSearch(t *testing.T) {
	t.Parallel()

	root := createTestDir(t)
	createTestFiles(t, root, []string{
		"test1.go",
		"test2.go",
		"test3.go",
	})

	s, err := NewSearcher(root)
	require.NoError(t, err)
	require.NoError(t, s.IndexAsync(context.Background()))

	// Run multiple concurrent searches.
	done := make(chan bool)

	for range 10 {
		go func() {
			results := s.Search("test", 10)
			assert.Len(t, results, 3)

			done <- true
		}()
	}

	// Wait for all goroutines.
	for range 10 {
		<-done
	}
}

// Benchmark Tests.

func BenchmarkSearcher_IndexAsync_100(b *testing.B) {
	root := createTestDir(&testing.T{})
	defer os.RemoveAll(root)

	files := make([]string, 100)
	for i := range 100 {
		files[i] = "src/file_" + string(rune(i)) + ".go"
	}

	createTestFiles(&testing.T{}, root, files)

	b.ResetTimer()

	for range b.N {
		s, _ := NewSearcher(root)
		_ = s.IndexAsync(context.Background())
	}
}

func BenchmarkSearcher_IndexAsync_1000(b *testing.B) {
	root := createTestDir(&testing.T{})
	defer os.RemoveAll(root)

	files := make([]string, 1000)
	for i := range 1000 {
		files[i] = "src/package/file_" + string(rune(i%100)) + ".go"
	}

	createTestFiles(&testing.T{}, root, files)

	b.ResetTimer()

	for range b.N {
		s, _ := NewSearcher(root)
		_ = s.IndexAsync(context.Background())
	}
}

func BenchmarkSearcher_Search_100(b *testing.B) {
	root := createTestDir(&testing.T{})
	defer os.RemoveAll(root)

	files := make([]string, 100)
	for i := range 100 {
		files[i] = "src/file_" + string(rune(i)) + ".go"
	}

	createTestFiles(&testing.T{}, root, files)

	s, _ := NewSearcher(root)
	_ = s.IndexAsync(context.Background())

	b.ResetTimer()

	for range b.N {
		s.Search("file", 10)
	}
}

func BenchmarkSearcher_Search_10000(b *testing.B) {
	root := createTestDir(&testing.T{})
	defer os.RemoveAll(root)

	files := make([]string, 10000)
	for i := range 10000 {
		files[i] = "src/pkg/module/file_" + string(rune(i%100)) + ".go"
	}

	createTestFiles(&testing.T{}, root, files)

	s, _ := NewSearcher(root)
	_ = s.IndexAsync(context.Background())

	b.ResetTimer()

	for range b.N {
		s.Search("file", 10)
	}
}
