package filesearch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScanner(t *testing.T) {
	s := NewScanner(".", true)

	assert.NotNil(t, s)
	assert.Equal(t, ".", s.baseDir)
	assert.True(t, s.ignoreGit)
}

func TestScanner_Scan_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestScanner_Scan_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file.
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "test.txt")
}

func TestScanner_Scan_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files.
	files := []string{"file1.txt", "file2.go", "file3.md"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte(""), 0644)
		require.NoError(t, err)
	}

	s := NewScanner(tmpDir, false)
	scanned, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, scanned, 3)
	assert.ElementsMatch(t, files, scanned)
}

func TestScanner_Scan_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir2"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dir1", "file1.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dir1", "dir2", "file2.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Contains(t, files, "root.txt")
	assert.Contains(t, files, "dir1/file1.txt")
	assert.Contains(t, files, "dir1/dir2/file2.txt")
}

func TestScanner_Scan_IgnoreGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory with files.
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "regular.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, true)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "regular.txt")
	assert.NotContains(t, files, ".git/config")
}

func TestScanner_Scan_IncludeGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory with files.
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "regular.txt"), []byte(""), 0644))

	// NOTE: With IgnoreHandler, .git is ALWAYS ignored by default patterns
	// This test now verifies that default ignore patterns are applied.
	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "regular.txt")
	// .git is in default ignore patterns, so it should NOT be included.
	assert.NotContains(t, files, ".git/config")
}

func TestScanner_Scan_RelativePaths(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "file.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	// Paths should be relative to baseDir.
	assert.Contains(t, files, "subdir/file.txt")
	// Should NOT contain absolute path.
	for _, file := range files {
		assert.False(t, filepath.IsAbs(file))
	}
}

func TestScanner_Scan_SymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file.
	realFile := filepath.Join(tmpDir, "real.txt")
	require.NoError(t, os.WriteFile(realFile, []byte(""), 0644))

	// Create a symlink.
	linkFile := filepath.Join(tmpDir, "link.txt")

	err := os.Symlink(realFile, linkFile)
	if err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	// Should include both real file and symlink.
	assert.GreaterOrEqual(t, len(files), 1)
}

func TestScanner_Scan_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden file.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, ".hidden")
	assert.Contains(t, files, "visible.txt")
}

func BenchmarkScanner_Scan_100Files(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 100 files.
	for i := range 100 {
		path := filepath.Join(tmpDir, "file_"+string(rune(i))+".txt")
		_ = os.WriteFile(path, []byte(""), 0644)
	}

	s := NewScanner(tmpDir, false)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Scan()
	}
}

func BenchmarkScanner_Scan_1000Files(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 1000 files in subdirectories.
	for i := range 10 {
		dir := filepath.Join(tmpDir, "dir"+string(rune(i)))
		_ = os.MkdirAll(dir, 0755)

		for j := range 100 {
			path := filepath.Join(dir, "file_"+string(rune(j))+".txt")
			_ = os.WriteFile(path, []byte(""), 0644)
		}
	}

	s := NewScanner(tmpDir, false)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Scan()
	}
}

// Integration tests for IgnoreHandler with Scanner.

func TestScanner_WithGitignore_SimplePattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore.
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("*.log\n"), 0644)
	require.NoError(t, err)

	// Create files.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "debug.log"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "app.txt")
	assert.NotContains(t, files, "debug.log")
}

func TestScanner_WithGitignore_DirectoryPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore.
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("build/\n"), 0644)
	require.NoError(t, err)

	// Create directories and files.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "build"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "build", "out.txt"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "src/main.go")
	assert.NotContains(t, files, "build/out.txt")
}

func TestScanner_WithGitignore_NodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// No need to create .gitignore - node_modules is in defaults.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "pkg", "index.js"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "app.js")
	assert.NotContains(t, files, "node_modules/pkg/index.js")
}

func TestScanner_WithSpinignore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .spinignore.
	spinignorePath := filepath.Join(tmpDir, ".spinignore")
	err := os.WriteFile(spinignorePath, []byte("*.tmp\n"), 0644)
	require.NoError(t, err)

	// Create files.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.tmp"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "test.txt")
	assert.NotContains(t, files, "test.tmp")
}

func TestScanner_WithBothIgnoreFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore.
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte("*.log\n"), 0644)
	require.NoError(t, err)

	// Create .spinignore.
	spinignorePath := filepath.Join(tmpDir, ".spinignore")
	err = os.WriteFile(spinignorePath, []byte("*.tmp\n"), 0644)
	require.NoError(t, err)

	// Create files.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "debug.log"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.tmp"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "app.txt")
	assert.NotContains(t, files, "debug.log")
	assert.NotContains(t, files, "test.tmp")
}

func TestScanner_RealWorldNodeProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate a real Node.js project structure.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "index.js"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "node_modules", "express"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "node_modules", "express", "index.js"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".DS_Store"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "src/index.js")
	assert.Contains(t, files, "package.json")
	assert.NotContains(t, files, "node_modules/express/index.js")
	assert.NotContains(t, files, ".DS_Store")
}

func TestScanner_RealWorldGoProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate a real Go project structure.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "cmd", "app"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "cmd", "app", "main.go"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "vendor", "lib"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "vendor", "lib", "lib.go"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git", "objects"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "cmd/app/main.go")
	assert.Contains(t, files, "go.mod")
	assert.NotContains(t, files, "vendor/lib/lib.go")
	assert.NotContains(t, files, ".git/config")
}

func TestScanner_RealWorldPythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate a real Python project structure.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.py"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "__pycache__"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "__pycache__", "main.cpython-39.pyc"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "module.pyc"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(""), 0644))

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "src/main.py")
	assert.Contains(t, files, "requirements.txt")
	assert.NotContains(t, files, "__pycache__/main.cpython-39.pyc")
	assert.NotContains(t, files, "module.pyc")
}

func TestScanner_BackwardCompatibility_IgnoreGitFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory.
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(""), 0644))

	// Test with ignoreGit=true (old behavior should still work).
	s := NewScanner(tmpDir, true)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Contains(t, files, "README.md")
	assert.NotContains(t, files, ".git/config")
}

func BenchmarkScanner_WithIgnore_10kFiles(b *testing.B) {
	tmpDir := b.TempDir()

	// Create .gitignore with a few patterns.
	_ = os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("*.log\n*.tmp\nbuild/\n"), 0644)

	// Create 10k files.
	for i := range 100 {
		dir := filepath.Join(tmpDir, "dir_"+string(rune(i)))
		_ = os.MkdirAll(dir, 0755)

		for j := range 100 {
			path := filepath.Join(dir, "file_"+string(rune(j))+".txt")
			_ = os.WriteFile(path, []byte(""), 0644)
		}
	}

	// Create some files to be ignored.
	_ = os.MkdirAll(filepath.Join(tmpDir, "build"), 0755)

	for i := range 50 {
		path := filepath.Join(tmpDir, "build", "file_"+string(rune(i))+".txt")
		_ = os.WriteFile(path, []byte(""), 0644)
	}

	s := NewScanner(tmpDir, false)

	b.ResetTimer()

	for range b.N {
		_, _ = s.Scan()
	}
}
