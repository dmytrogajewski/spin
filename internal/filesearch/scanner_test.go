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

	// Create a file
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

	// Create multiple files
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

	// Create nested structure
	os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir2"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "dir1", "file1.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "dir1", "dir2", "file2.txt"), []byte(""), 0644)

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

	// Create .git directory with files
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "regular.txt"), []byte(""), 0644)

	s := NewScanner(tmpDir, true)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "regular.txt")
	assert.NotContains(t, files, ".git/config")
}

func TestScanner_Scan_IncludeGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory with files
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "regular.txt"), []byte(""), 0644)

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "regular.txt")
	assert.Contains(t, files, ".git/config")
}

func TestScanner_Scan_RelativePaths(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file.txt"), []byte(""), 0644)

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	// Paths should be relative to baseDir
	assert.Contains(t, files, "subdir/file.txt")
	// Should NOT contain absolute path
	for _, file := range files {
		assert.False(t, filepath.IsAbs(file))
	}
}

func TestScanner_Scan_SymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	realFile := filepath.Join(tmpDir, "real.txt")
	os.WriteFile(realFile, []byte(""), 0644)

	// Create a symlink
	linkFile := filepath.Join(tmpDir, "link.txt")
	err := os.Symlink(realFile, linkFile)
	if err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	// Should include both real file and symlink
	assert.GreaterOrEqual(t, len(files), 1)
}

func TestScanner_Scan_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden file
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte(""), 0644)

	s := NewScanner(tmpDir, false)
	files, err := s.Scan()

	assert.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, ".hidden")
	assert.Contains(t, files, "visible.txt")
}

func BenchmarkScanner_Scan_100Files(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 100 files
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, "file_"+string(rune(i))+".txt")
		os.WriteFile(path, []byte(""), 0644)
	}

	s := NewScanner(tmpDir, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Scan()
	}
}

func BenchmarkScanner_Scan_1000Files(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 1000 files in subdirectories
	for i := 0; i < 10; i++ {
		dir := filepath.Join(tmpDir, "dir"+string(rune(i)))
		os.MkdirAll(dir, 0755)
		for j := 0; j < 100; j++ {
			path := filepath.Join(dir, "file_"+string(rune(j))+".txt")
			os.WriteFile(path, []byte(""), 0644)
		}
	}

	s := NewScanner(tmpDir, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Scan()
	}
}
