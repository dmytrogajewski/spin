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

const readCompactFixture = "package demo\n\n// Hello greets.\nfunc Hello() string {\n\treturn \"hi\"\n}\n"

func writeReadFixture(t *testing.T) (path, raw string) {
	t.Helper()

	dir := t.TempDir()
	path = filepath.Join(dir, "demo.go")
	require.NoError(t, os.WriteFile(path, []byte(readCompactFixture), 0o600))

	return path, readCompactFixture
}

func TestReadFileTool_DefaultMinimal(t *testing.T) {
	t.Parallel()

	path, raw := writeReadFixture(t)
	tool := NewReadFileTool()

	params, err := FromMap(map[string]any{"path": path})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !result.Success {
		t.Fatalf("expected success, got %s", result.Error)
	}

	want := string(compact.Default().Apply("read", []byte(raw), nil, 0).Stdout)
	if result.Output != want {
		t.Fatalf("Output = %q, want compact minimal %q", result.Output, want)
	}
}

func TestReadFileTool_LevelNone(t *testing.T) {
	t.Parallel()

	path, raw := writeReadFixture(t)
	tool := NewReadFileTool()

	params, err := FromMap(map[string]any{"path": path, "level": compact.LevelNone})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != raw {
		t.Fatalf("level=none Output = %q, want raw", result.Output)
	}
}

func TestReadFileTool_EnvOffSkipsCompact(t *testing.T) {
	t.Setenv(compact.EnvName, compact.EnvOff)

	path, raw := writeReadFixture(t)
	tool := NewReadFileTool()

	params, err := FromMap(map[string]any{"path": path})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != raw {
		t.Fatalf("SPIN_COMPACT=0 Output = %q, want raw", result.Output)
	}
}

func TestReadFileTool_DisabledSkipsCompact(t *testing.T) {
	t.Parallel()

	path, raw := writeReadFixture(t)
	tool := NewReadFileTool()
	tool.SetCompactEnabled(false)

	params, err := FromMap(map[string]any{"path": path})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != raw {
		t.Fatalf("disabled Output = %q, want raw", result.Output)
	}
}

func TestReadFileTool_ConfigReadLevelNone(t *testing.T) {
	t.Parallel()

	path, raw := writeReadFixture(t)
	tool := NewReadFileTool()
	tool.SetReadLevel(compact.LevelNone)

	params, err := FromMap(map[string]any{"path": path})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if result.Output != raw {
		t.Fatalf("config none Output = %q, want raw", result.Output)
	}
}
