package tools_test

// Journey: specs/journeys/JOURNEY-R8.2.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const (
	testGoFile  = "/workspace/main.go"
	testGoURI   = "file:///workspace/main.go"
	testSymName = "HandleRequest"
)

var errServerUnavailable = errors.New("server unavailable")

func makeParams(kv map[string]any) tools.ToolParameters {
	params, _ := tools.FromMap(kv)

	return params
}

func TestFindSymbolTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindSymbolTool(nil)

	require.Equal(t, "find_symbol", tool.Name())
}

func TestFindSymbolTool_ReturnsLocations(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return []lsp.Location{
			{URI: testGoURI, Range: lsp.Range{Start: lsp.Position{Line: 9, Character: 4}}},
		}, nil
	}

	tool := tools.NewFindSymbolTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"name":      testSymName,
		"file_path": testGoFile,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "1 location(s)")
	require.Contains(t, result.Output, "/workspace/main.go:10:5")
}

func TestFindSymbolTool_NoResults(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return []lsp.Location{}, nil
	}

	tool := tools.NewFindSymbolTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"name":      "NonExistent",
		"file_path": testGoFile,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No symbols found")
}

func TestFindSymbolTool_MissingName(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindSymbolTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"file_path": testGoFile})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFindSymbolTool_MissingFilePath(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindSymbolTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"name": testSymName})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFindSymbolTool_NilFinder(t *testing.T) {
	t.Parallel()

	tool := tools.NewFindSymbolTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"name":      testSymName,
		"file_path": testGoFile,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFindSymbolTool_FinderError(t *testing.T) {
	t.Parallel()

	find := func(_ context.Context, _ string, _, _ int) ([]lsp.Location, error) {
		return nil, errServerUnavailable
	}

	tool := tools.NewFindSymbolTool(find)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"name":      testSymName,
		"file_path": testGoFile,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "server unavailable")
}
