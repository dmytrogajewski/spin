package tools_test

// Journey: specs/journeys/JOURNEY-R8.2.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const (
	testNewName    = "HandleResponse"
	testRenameLine = 10
	testRenameChar = 5
)

func TestRenameSymbolTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewRenameSymbolTool(nil)

	require.Equal(t, "rename_symbol", tool.Name())
}

func TestRenameSymbolTool_RenamesAllUsages(t *testing.T) {
	t.Parallel()

	rename := func(_ context.Context, _ string, _, _ int, _ string) (*lsp.WorkspaceEdit, error) {
		return &lsp.WorkspaceEdit{
			Changes: map[string][]lsp.TextEdit{
				"file:///a.go": {
					{Range: lsp.Range{Start: lsp.Position{Line: 4, Character: 5}}, NewText: testNewName},
					{Range: lsp.Range{Start: lsp.Position{Line: 19, Character: 0}}, NewText: testNewName},
				},
				"file:///b.go": {
					{Range: lsp.Range{Start: lsp.Position{Line: 9, Character: 2}}, NewText: testNewName},
				},
			},
		}, nil
	}

	tool := tools.NewRenameSymbolTool(rename)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
		"new_name":  testNewName,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "2 file(s)")
	require.Contains(t, result.Output, "3 edit(s)")
}

func TestRenameSymbolTool_NoChanges(t *testing.T) {
	t.Parallel()

	rename := func(_ context.Context, _ string, _, _ int, _ string) (*lsp.WorkspaceEdit, error) {
		return &lsp.WorkspaceEdit{}, nil
	}

	tool := tools.NewRenameSymbolTool(rename)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
		"new_name":  testNewName,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No changes")
}

func TestRenameSymbolTool_InvalidIdentifier(t *testing.T) {
	t.Parallel()

	tool := tools.NewRenameSymbolTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
		"new_name":  "123invalid",
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "not a valid identifier")
}

func TestRenameSymbolTool_RequiresApproval(t *testing.T) {
	t.Parallel()

	tool := tools.NewRenameSymbolTool(nil)

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"new_name":  testNewName,
	})

	needs := tool.CheckApproval(params)
	require.True(t, needs.Required)
	require.Equal(t, tools.RiskHigh, needs.Risk)
	require.Contains(t, needs.Reason, testGoFile)
	require.Contains(t, needs.Reason, testNewName)
}

func TestRenameSymbolTool_MissingParams(t *testing.T) {
	t.Parallel()

	tool := tools.NewRenameSymbolTool(nil)
	ctx := context.Background()

	// Missing new_name.
	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestRenameSymbolTool_NilFinder(t *testing.T) {
	t.Parallel()

	tool := tools.NewRenameSymbolTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
		"new_name":  testNewName,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestRenameSymbolTool_FinderError(t *testing.T) {
	t.Parallel()

	rename := func(_ context.Context, _ string, _, _ int, _ string) (*lsp.WorkspaceEdit, error) {
		return nil, errServerUnavailable
	}

	tool := tools.NewRenameSymbolTool(rename)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"file_path": testGoFile,
		"line":      testRenameLine,
		"character": testRenameChar,
		"new_name":  testNewName,
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

// Verify RenameSymbolTool implements ToolWithApproval at compile time.
var _ tools.ToolWithApproval = (*tools.RenameSymbolTool)(nil)
