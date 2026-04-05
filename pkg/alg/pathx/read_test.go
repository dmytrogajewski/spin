package pathx

// Journey: specs/journeys/JOURNEY-R6.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// smallContent is test content that fits within any reasonable limit.
const smallContent = "hello world"

// maxTestBytes is the byte limit used in truncation tests.
const maxTestBytes = 5

// testFilePermissions is the permission mode for test files.
const testFilePermissions = 0o600

func TestReadFileWithLimit(t *testing.T) {
	t.Parallel()

	t.Run("small_file_not_truncated", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, smallContent)

		content, truncated, err := ReadFileWithLimit(path, 1024)
		require.NoError(t, err)
		require.False(t, truncated)
		require.Equal(t, smallContent, content)
	})

	t.Run("large_file_truncated", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "abcdefghij")

		content, truncated, err := ReadFileWithLimit(path, maxTestBytes)
		require.NoError(t, err)
		require.True(t, truncated)
		require.Equal(t, "abcde", content)
	})

	t.Run("zero_limit_reads_all", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, smallContent)

		content, truncated, err := ReadFileWithLimit(path, 0)
		require.NoError(t, err)
		require.False(t, truncated)
		require.Equal(t, smallContent, content)
	})

	t.Run("negative_limit_reads_all", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, smallContent)

		content, truncated, err := ReadFileWithLimit(path, -1)
		require.NoError(t, err)
		require.False(t, truncated)
		require.Equal(t, smallContent, content)
	})

	t.Run("missing_file_errors", func(t *testing.T) {
		t.Parallel()

		_, _, err := ReadFileWithLimit("/nonexistent/file.txt", 1024)
		require.Error(t, err)
	})

	t.Run("exact_size_not_truncated", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "12345")

		content, truncated, err := ReadFileWithLimit(path, maxTestBytes)
		require.NoError(t, err)
		require.False(t, truncated)
		require.Equal(t, "12345", content)
	})
}

func TestReadLastLines(t *testing.T) {
	t.Parallel()

	t.Run("fewer_lines_than_n", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "line1\nline2")

		got, err := ReadLastLines(path, 5)
		require.NoError(t, err)
		require.Equal(t, "line1\nline2", got)
	})

	t.Run("exactly_n_lines", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "a\nb\nc")

		got, err := ReadLastLines(path, 3)
		require.NoError(t, err)
		require.Equal(t, "a\nb\nc", got)
	})

	t.Run("more_than_n_lines", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "a\nb\nc\nd\ne")

		got, err := ReadLastLines(path, 2)
		require.NoError(t, err)
		require.Equal(t, "d\ne", got)
	})

	t.Run("empty_file", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "")

		got, err := ReadLastLines(path, 5)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("single_line", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "only line")

		got, err := ReadLastLines(path, 1)
		require.NoError(t, err)
		require.Equal(t, "only line", got)
	})

	t.Run("missing_file_errors", func(t *testing.T) {
		t.Parallel()

		_, err := ReadLastLines("/nonexistent/file.txt", 5)
		require.Error(t, err)
	})

	t.Run("trailing_newline_handled", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "a\nb\nc\n")

		got, err := ReadLastLines(path, 2)
		require.NoError(t, err)
		require.Contains(t, got, "c")
	})
}

// writeTestFile creates a temp file with the given content and returns its path.
func writeTestFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	require.NoError(t, os.WriteFile(path, []byte(content), testFilePermissions))

	return path
}
