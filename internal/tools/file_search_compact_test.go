package tools

// Journey: specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeSearchFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "pkg"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alpha.go"), []byte("a"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pkg", "alpha_test.go"), []byte("b"), 0o600))

	return dir
}

func TestFileSearchTool_TreeCompression(t *testing.T) {
	t.Parallel()

	dir := writeSearchFixture(t)
	tool := NewFileSearchTool(dir)

	params, err := FromMap(map[string]any{"query": "alpha", "limit": 10})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !result.Success {
		t.Fatalf("expected success, got %s", result.Error)
	}

	if !strings.Contains(result.Output, "alpha.go") {
		t.Fatalf("Output %q missing fixture name", result.Output)
	}

	if !strings.Contains(result.Output, ". (") {
		t.Fatalf("Output %q want R10 tree header", result.Output)
	}

	if strings.Contains(result.Output, "score:") {
		t.Fatalf("Output %q still raw ranked list", result.Output)
	}
}

func TestFileSearchTool_DisabledSkipsCompact(t *testing.T) {
	t.Parallel()

	dir := writeSearchFixture(t)
	tool := NewFileSearchTool(dir)
	tool.SetCompactEnabled(false)

	params, err := FromMap(map[string]any{"query": "alpha", "limit": 10})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if strings.Contains(result.Output, ". (") {
		t.Fatalf("disabled Output = %q, want raw paths", result.Output)
	}

	if !strings.Contains(result.Output, "alpha.go") {
		t.Fatalf("disabled Output %q missing fixture name", result.Output)
	}
}
