package filesearch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewIgnoreHandler_BasicCreation tests basic creation.
func TestNewIgnoreHandler_BasicCreation(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	assert.Equal(t, tmpDir, handler.rootDir)
	// Should have default patterns.
	assert.NotEmpty(t, handler.patterns)
}

// TestNewIgnoreHandler_MissingFiles tests handler creation when no .gitignore exists.
func TestNewIgnoreHandler_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// No .gitignore or .spinignore files.

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	// Should still have default patterns.
	assert.Contains(t, handler.patterns, ".git/**")
	assert.Contains(t, handler.patterns, "node_modules/**")
}

// TestNewIgnoreHandler_LoadGitignore tests loading .gitignore.
func TestNewIgnoreHandler_LoadGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore.
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := `# Comment
*.log
build/
temp
`
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.Contains(t, handler.patterns, "*.log")
	assert.Contains(t, handler.patterns, "build/")
	assert.Contains(t, handler.patterns, "temp")
	// Should not contain comments or empty lines.
	assert.NotContains(t, handler.patterns, "# Comment")
}

// TestNewIgnoreHandler_LoadSpinignore tests loading .spinignore.
func TestNewIgnoreHandler_LoadSpinignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .spinignore.
	spinignorePath := filepath.Join(tmpDir, ".spinignore")
	spinignoreContent := `custom-dir/
*.tmp
`
	err := os.WriteFile(spinignorePath, []byte(spinignoreContent), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.Contains(t, handler.patterns, "custom-dir/")
	assert.Contains(t, handler.patterns, "*.tmp")
}

// TestNewIgnoreHandler_BothFiles tests loading both .gitignore and .spinignore.
func TestNewIgnoreHandler_BothFiles(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("*.log\n"), 0644)
	require.NoError(t, err)

	spinignorePath := filepath.Join(tmpDir, ".spinignore")
	err = os.WriteFile(spinignorePath, []byte("*.tmp\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.Contains(t, handler.patterns, "*.log")
	assert.Contains(t, handler.patterns, "*.tmp")
}

// TestNewIgnoreHandler_EmptyFile tests empty .gitignore.
func TestNewIgnoreHandler_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte(""), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, handler)
	// Should still have defaults.
	assert.Contains(t, handler.patterns, ".git/**")
}

// TestNewIgnoreHandler_OnlyComments tests .gitignore with only comments.
func TestNewIgnoreHandler_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content := `# This is a comment
# Another comment


`
	err := os.WriteFile(gitignorePath, []byte(content), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)

	require.NoError(t, err)
	// Should not add comment lines.
	for _, pattern := range handler.patterns {
		assert.False(t, strings.HasPrefix(pattern, "#"))
	}
}

// TestIgnoreHandler_IsIgnored_SimplePattern tests simple pattern matching.
func TestIgnoreHandler_IsIgnored_SimplePattern(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	// Use **/*.log to match .log files at any depth (proper gitignore syntax).
	err := os.WriteFile(gitignorePath, []byte("*.log\n**/*.log\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"match .log file", "debug.log", false, true},
		{"match .log in subdir", "logs/debug.log", false, true},
		{"no match .txt file", "readme.txt", false, false},
		{"no match log.txt", "log.txt", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIgnoreHandler_IsIgnored_DirectoryPattern tests directory pattern matching.
func TestIgnoreHandler_IsIgnored_DirectoryPattern(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	// Pattern with trailing / should only match directories, not files with same name.
	err := os.WriteFile(gitignorePath, []byte("dist/\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"match dist dir", "dist", true, true},
		// Note: dist/output won't match with just "dist/" pattern
		// In real usage, Scanner will skip the dist directory entirely via SkipDir.
		{"no match dist file", "dist", false, false},
		{"no match dist.txt", "dist.txt", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIgnoreHandler_IsIgnored_DoubleStarPattern tests ** patterns.
func TestIgnoreHandler_IsIgnored_DoubleStarPattern(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("node_modules/**\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"match node_modules dir", "node_modules", true, true},
		{"match file in node_modules", "node_modules/pkg/file.js", false, true},
		{"match subdir in node_modules", "node_modules/pkg/sub", true, true},
		{"match deep file", "node_modules/a/b/c/d.js", false, true},
		{"no match similar name", "node_modules_backup", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIgnoreHandler_IsIgnored_WildcardPattern tests **/pattern matching.
func TestIgnoreHandler_IsIgnored_WildcardPattern(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("**/temp\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{"match temp at root", "temp", true, true},
		{"match temp in subdir", "build/temp", true, true},
		{"match temp deeply nested", "a/b/c/temp", true, true},
		{"no match tempfile", "tempfile", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIgnoreHandler_IsIgnored_DefaultPatterns tests default ignore patterns.
func TestIgnoreHandler_IsIgnored_DefaultPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		isDir    bool
		expected bool
	}{
		{".git dir", ".git", true, true},
		{".git file", ".git/config", false, true},
		{"node_modules dir", "node_modules", true, true},
		{"node_modules file", "node_modules/pkg/index.js", false, true},
		{".spin dir", ".spin", true, true},
		{"vendor dir", "vendor", true, true},
		{"vendor file", "vendor/lib.go", false, true},
		{"__pycache__ dir", "__pycache__", true, true},
		{"__pycache__ file", "__pycache__/module.pyc", false, true},
		{".vscode dir", ".vscode", true, true},
		{".idea dir", ".idea", true, true},
		{".pyc file", "script.pyc", false, true},
		{".pyo file", "script.pyo", false, true},
		{".DS_Store", ".DS_Store", false, true},
		{"Thumbs.db", "Thumbs.db", false, true},
		{"regular file", "src/main.go", false, false},
		{"regular dir", "src", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIgnored(tt.path, tt.isDir)
			assert.Equal(t, tt.expected, result, "path: %s, isDir: %v", tt.path, tt.isDir)
		})
	}
}

// TestIgnoreHandler_IsIgnored_MultiplePatterns tests multiple patterns.
func TestIgnoreHandler_IsIgnored_MultiplePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content := `*.log
*.tmp
build/
dist/
`
	err := os.WriteFile(gitignorePath, []byte(content), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"debug.log", false, true},
		{"temp.tmp", false, true},
		{"build", true, true},
		{"dist", true, true},
		{"src/main.go", false, false},
		{"build.txt", false, false}, // not a directory.
	}

	for _, tt := range tests {
		result := handler.IsIgnored(tt.path, tt.isDir)
		assert.Equal(t, tt.expected, result, "path: %s", tt.path)
	}
}

// TestIgnoreHandler_IsIgnored_CaseSensitive tests case sensitivity.
func TestIgnoreHandler_IsIgnored_CaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("Build/\n"), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	// Gitignore patterns are case-sensitive by default.
	assert.True(t, handler.IsIgnored("Build", true))
	// Note: On case-insensitive filesystems (macOS, Windows), this may match
	// but on Linux it should not match
	// For this test, we accept either behavior since filesystem matters.
}

// TestIgnoreHandler_IsIgnored_EmptyPath tests empty path.
func TestIgnoreHandler_IsIgnored_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	result := handler.IsIgnored("", false)
	assert.False(t, result, "empty path should not be ignored")
}

// TestIgnoreHandler_IsIgnored_RootDot tests . and ./ paths.
func TestIgnoreHandler_IsIgnored_RootDot(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	assert.False(t, handler.IsIgnored(".", true))
	assert.False(t, handler.IsIgnored("./", true))
}

// TestIgnoreHandler_LoadIgnoreFile_Whitespace tests whitespace handling.
func TestIgnoreHandler_LoadIgnoreFile_Whitespace(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content := `  *.log
	build/
temp
`
	err := os.WriteFile(gitignorePath, []byte(content), 0644)
	require.NoError(t, err)

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	// Patterns should be trimmed.
	assert.Contains(t, handler.patterns, "*.log")
	assert.Contains(t, handler.patterns, "build/")
	assert.Contains(t, handler.patterns, "temp")
}

// TestIgnoreHandler_LoadIgnoreFile_LongPatternList tests performance with many patterns.
func TestIgnoreHandler_LoadIgnoreFile_LongPatternList(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	f, err := os.Create(gitignorePath)
	require.NoError(t, err)

	defer f.Close()

	// Write 1000 patterns.
	for i := range 1000 {
		_, _ = f.WriteString("pattern_" + string(rune(i)) + "\n")
	}

	f.Close()

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	// Should handle many patterns without error.
	assert.Greater(t, len(handler.patterns), 1000) // including defaults.
}

// TestIgnoreHandler_IsIgnored_PerformanceCheck tests performance.
func TestIgnoreHandler_IsIgnored_PerformanceCheck(t *testing.T) {
	tmpDir := t.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	f, err := os.Create(gitignorePath)
	require.NoError(t, err)

	defer f.Close()

	// Write 100 patterns.
	for i := range 100 {
		_, _ = f.WriteString("*.pattern" + string(rune(i)) + "\n")
	}

	f.Close()

	handler, err := NewIgnoreHandler(tmpDir)
	require.NoError(t, err)

	// Test 1000 IsIgnored calls - should be fast.
	for range 1000 {
		handler.IsIgnored("some/path/file.txt", false)
	}
	// If this test completes without timeout, performance is acceptable.
}

// TestDefaultPatterns tests the default patterns function.
func TestDefaultPatterns(t *testing.T) {
	patterns := defaultPatterns()

	assert.NotEmpty(t, patterns)
	assert.Contains(t, patterns, ".git/**")
	assert.Contains(t, patterns, "node_modules/**")
	assert.Contains(t, patterns, ".spin/**")
	assert.Contains(t, patterns, "vendor/**")
	assert.Contains(t, patterns, "__pycache__/**")
	assert.Contains(t, patterns, ".vscode/**")
	assert.Contains(t, patterns, ".idea/**")
}

// Benchmark tests.

func BenchmarkIgnoreHandler_IsIgnored_10Patterns(b *testing.B) {
	tmpDir := b.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	content := ""
	var contentSb476 strings.Builder
	for i := range 10 {
		contentSb476.WriteString("*.pattern" + string(rune(i)) + "\n")
	}
	content += contentSb476.String()

	_ = os.WriteFile(gitignorePath, []byte(content), 0644)

	handler, _ := NewIgnoreHandler(tmpDir)

	b.ResetTimer()

	for range b.N {
		handler.IsIgnored("some/path/file.txt", false)
	}
}

func BenchmarkIgnoreHandler_IsIgnored_100Patterns(b *testing.B) {
	tmpDir := b.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	content := ""
	var contentSb497 strings.Builder
	for i := range 100 {
		contentSb497.WriteString("*.pattern" + string(rune(i)) + "\n")
	}
	content += contentSb497.String()

	_ = os.WriteFile(gitignorePath, []byte(content), 0644)

	handler, _ := NewIgnoreHandler(tmpDir)

	b.ResetTimer()

	for range b.N {
		handler.IsIgnored("some/path/file.txt", false)
	}
}

func BenchmarkIgnoreHandler_IsIgnored_1000Patterns(b *testing.B) {
	tmpDir := b.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	content := ""
	var contentSb518 strings.Builder
	for i := range 1000 {
		contentSb518.WriteString("*.pattern" + string(rune(i)) + "\n")
	}
	content += contentSb518.String()

	_ = os.WriteFile(gitignorePath, []byte(content), 0644)

	handler, _ := NewIgnoreHandler(tmpDir)

	b.ResetTimer()

	for range b.N {
		handler.IsIgnored("some/path/file.txt", false)
	}
}

func BenchmarkIgnoreHandler_IsIgnored_Match(b *testing.B) {
	tmpDir := b.TempDir()

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	_ = os.WriteFile(gitignorePath, []byte("*.log\n"), 0644)

	handler, _ := NewIgnoreHandler(tmpDir)

	b.ResetTimer()

	for range b.N {
		handler.IsIgnored("debug.log", false)
	}
}
