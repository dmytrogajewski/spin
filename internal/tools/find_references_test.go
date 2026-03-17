package tools_test

// Journey: specs/journeys/JOURNEY-R8.2.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestFindReferencesTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindReferencesTool(nil)

	require.Equal(t, "find_references", tool.Name())
}

func TestFindReferencesTool_CrossFile(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return []lsp.Location{
			{URI: "file:///a.go", Range: lsp.Range{Start: lsp.Position{Line: 4, Character: 0}}},
			{URI: "file:///b.go", Range: lsp.Range{Start: lsp.Position{Line: 9, Character: 2}}},
			{URI: "file:///a.go", Range: lsp.Range{Start: lsp.Position{Line: 19, Character: 0}}},
		}, nil
	}

	tool := tools.NewFindReferencesTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      10,
		"character": 5,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "3 reference(s)")
	require.Contains(t, result.Output, "2 file(s)")
}

func TestFindReferencesTool_NoResults(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return []lsp.Location{}, nil
	}

	tool := tools.NewFindReferencesTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      0,
		"character": 0,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No references found")
}

func TestFindReferencesTool_MissingParams(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindReferencesTool(nil)
	ctx := context.Background()

	// Missing file_path.
	params := makeParams(map[string]any{
		"line":      0,
		"character": 0,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFindReferencesTool_NilFinder(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindReferencesTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      0,
		"character": 0,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFindReferencesTool_FinderError(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return nil, errServerUnavailable
	}

	tool := tools.NewFindReferencesTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      0,
		"character": 0,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}
