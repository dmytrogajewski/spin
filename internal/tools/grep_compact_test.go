package tools

// Journey: specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
)

const grepLongLine = "this is a very long matching line that should be truncated because it exceeds the compact line limit!!"

func writeGrepFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "pkg"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pkg", "a.go"), []byte(grepLongLine+"\nshort match\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pkg", "b.go"), []byte("other file match\n"), 0o600))

	return root
}

func TestGrepTool_GroupsAndTruncates(t *testing.T) {
	t.Parallel()

	root := writeGrepFixture(t)
	tool := NewGrepTool(root)

	params, err := FromMap(map[string]any{"pattern": "match"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !result.Success {
		t.Fatalf("expected success, got %s", result.Error)
	}

	if !strings.Contains(result.Output, "pkg/a.go") || !strings.Contains(result.Output, "pkg/b.go") {
		t.Fatalf("Output %q missing file groups", result.Output)
	}

	if strings.Contains(result.Output, grepLongLine) {
		t.Fatalf("Output %q still has untruncated line", result.Output)
	}

	if !strings.Contains(result.Output, "...") {
		t.Fatalf("Output %q want truncated ellipsis", result.Output)
	}
}

func TestGrepTool_DisabledSkipsCompact(t *testing.T) {
	t.Parallel()

	root := writeGrepFixture(t)
	tool := NewGrepTool(root)
	tool.SetCompactEnabled(false)

	params, err := FromMap(map[string]any{"pattern": "match"})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !strings.Contains(result.Output, grepLongLine) {
		t.Fatalf("disabled Output %q want raw long line", result.Output)
	}

	want := string(compact.Default().Apply("grep", []byte(result.Output), nil, 0).Stdout)
	if result.Output == want {
		t.Fatalf("disabled output was compacted")
	}
}
