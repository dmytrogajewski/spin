package tools

// Journey: specs/journeys/JOURNEY-R4.2.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEditFileTool_SuccessfulEdit(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")
	originalContent := "func hello() {\n\tfmt.Println(\"hello\")\n}"

	require.NoError(t, os.WriteFile(filePath, []byte(originalContent), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "fmt.Println(\"hello\")",
		"new_content": "fmt.Println(\"world\")",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success, "expected success, got: %s", result.Error)

	content, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	require.Contains(t, string(content), "world")
	require.NotContains(t, string(content), "\"hello\"")
}

func TestEditFileTool_StaleReadRejection(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	require.NoError(t, os.WriteFile(filePath, []byte("original"), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	// Wait and modify to make it stale.
	time.Sleep(sleepBeyondTolerance)

	require.NoError(t, os.WriteFile(filePath, []byte("modified by user"), 0o600))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "original",
		"new_content": "replacement",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "modified since last read")
}

func TestEditFileTool_AmbiguousMatchRejection(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")
	content := "foo\nbar\nfoo\nbaz"

	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "foo",
		"new_content": "qux",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "ambiguous")
	require.Contains(t, result.Error, "2 occurrences")
}

func TestEditFileTool_ReturnsDiff(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "hello",
		"new_content": "goodbye",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "exact")
	require.Contains(t, result.Output, "-hello")
	require.Contains(t, result.Output, "+goodbye")
}

func TestEditFileTool_NoMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")

	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "this does not exist anywhere in the file and has no overlap with anything",
		"new_content": "replacement",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "no match found")
}

func TestEditFileTool_FuzzyMatchIndent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.go")
	content := "\t\treturn nil\n\t}"

	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	tracker := NewFileTracker()
	require.NoError(t, tracker.RecordRead(filePath))

	tool := NewEditFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "    return nil\n}",
		"new_content": "    return err\n}",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success, "expected success, got: %s", result.Error)
	require.Contains(t, result.Output, "indent")

	written, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	require.Contains(t, string(written), "return err")
}

func TestEditFileTool_MissingParams(t *testing.T) {
	t.Parallel()

	tool := NewEditFileTool()

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing path", map[string]any{"old_content": "a", "new_content": "b"}},
		{"missing old_content", map[string]any{"path": "/tmp/x", "new_content": "b"}},
		{"missing new_content", map[string]any{"path": "/tmp/x", "old_content": "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			require.NoError(t, err)
			require.False(t, result.Success)
		})
	}
}

func TestEditFileTool_CheckApproval(t *testing.T) {
	t.Parallel()

	tool := NewEditFileTool()

	params, _ := FromMap(map[string]any{
		"path":        "/tmp/test.go",
		"old_content": "a",
		"new_content": "b",
	})

	needs := tool.CheckApproval(params)
	require.True(t, needs.Required)
	require.Equal(t, RiskHigh, needs.Risk)
	require.Contains(t, needs.Reason, "test.go")
}

func TestEditFileTool_WithoutTracker(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o600))

	tool := NewEditFileTool()

	params, _ := FromMap(map[string]any{
		"path":        filePath,
		"old_content": "hello",
		"new_content": "goodbye",
	})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success)
}
