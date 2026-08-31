package tools

// Journey: specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
)

func writeListFixture(t *testing.T) (dir, rawListing string) {
	t.Helper()

	dir = t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("a"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("b"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o750))

	return dir, "./file1.txt\n./file2.txt\n./subdir/\n"
}

func TestListDirectoryTool_TreeCompression(t *testing.T) {
	t.Parallel()

	dir, raw := writeListFixture(t)
	tool := NewListDirectoryTool()

	params, err := FromMap(map[string]any{"path": dir})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !result.Success {
		t.Fatalf("expected success, got %s", result.Error)
	}

	want := string(compact.Default().Apply("ls", []byte(raw), nil, 0).Stdout)
	if result.Output != want {
		t.Fatalf("Output = %q, want R10 %q", result.Output, want)
	}
}

func TestListDirectoryTool_DisabledSkipsCompact(t *testing.T) {
	t.Parallel()

	dir, raw := writeListFixture(t)
	tool := NewListDirectoryTool()
	tool.SetCompactEnabled(false)

	params, err := FromMap(map[string]any{"path": dir})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != raw {
		t.Fatalf("disabled Output = %q, want raw %q", result.Output, raw)
	}
}
