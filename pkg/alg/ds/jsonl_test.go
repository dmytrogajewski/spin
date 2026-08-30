package ds

// Journey: specs/journeys/JOURNEY-R18.md.

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFilePermissions is the permission for test files.
const testFilePermissions = 0o600

// testItem is a typed struct for JSONL testing.
type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestJSONLWriter_append_and_read(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	writer, err := NewJSONLWriter[testItem](path)
	require.NoError(t, err)

	require.NoError(t, writer.Append(testItem{Name: "first", Value: 1}))
	require.NoError(t, writer.Append(testItem{Name: "second", Value: 2}))

	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "first", items[0].Name)
	require.Equal(t, "second", items[1].Name)
	require.Equal(t, 2, writer.Count())

	require.NoError(t, writer.Close())
}

func TestJSONLWriter_strings(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	writer, err := NewJSONLWriter[string](path)
	require.NoError(t, err)

	require.NoError(t, writer.Append("hello"))
	require.NoError(t, writer.Append("world"))

	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{"hello", "world"}, items)

	require.NoError(t, writer.Close())
}

func TestJSONLWriter_corrupt_lines_skipped(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)

	// Write valid JSON, corrupt data, then more valid JSON.
	content := "{\"name\":\"good\",\"value\":1}\nnot-json\n{\"name\":\"also-good\",\"value\":2}\n"
	require.NoError(t, os.WriteFile(path, []byte(content), testFilePermissions))

	writer, err := NewJSONLWriter[testItem](path)
	require.NoError(t, err)

	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "good", items[0].Name)
	require.Equal(t, "also-good", items[1].Name)

	require.NoError(t, writer.Close())
}

func TestJSONLWriter_empty_file(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	require.NoError(t, os.WriteFile(path, nil, testFilePermissions))

	writer, err := NewJSONLWriter[testItem](path)
	require.NoError(t, err)

	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Empty(t, items)

	require.NoError(t, writer.Close())
}

func TestJSONLWriter_close_prevents_append(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	writer, err := NewJSONLWriter[testItem](path)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	err = writer.Append(testItem{Name: "late", Value: 99})
	require.ErrorIs(t, err, ErrWriterClosed)
}

func TestJSONLWriter_concurrent_append(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	writer, err := NewJSONLWriter[int](path)
	require.NoError(t, err)

	concurrency := 50

	var wg sync.WaitGroup

	wg.Add(concurrency)

	for idx := range concurrency {
		go func() {
			defer wg.Done()

			assert.NoError(t, writer.Append(idx))
		}()
	}

	wg.Wait()

	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Len(t, items, concurrency)
	require.Equal(t, concurrency, writer.Count())

	require.NoError(t, writer.Close())
}

func TestJSONLWriter_nonexistent_read(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	writer, err := NewJSONLWriter[testItem](path)
	require.NoError(t, err)

	// ReadAll on a freshly-created (empty) file should return empty.
	items, err := writer.ReadAll()
	require.NoError(t, err)
	require.Empty(t, items)

	require.NoError(t, writer.Close())
}

func TestReadJSONL_MissingFile(t *testing.T) {
	t.Parallel()

	items, err := ReadJSONL[testItem](filepath.Join(t.TempDir(), "nope.jsonl"))
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestReadJSONL_ValidLines(t *testing.T) {
	t.Parallel()

	path := testFilePath(t)
	require.NoError(t, os.WriteFile(path, []byte("{\"name\":\"a\",\"value\":1}\n"), testFilePermissions))

	items, err := ReadJSONL[testItem](path)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "a", items[0].Name)
}

// testFilePath returns a unique temp file path for a test.
func testFilePath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "test.jsonl")
}
