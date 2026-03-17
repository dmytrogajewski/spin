package tools_test

// Journey: specs/journeys/JOURNEY-R8.3.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var errSearchFailed = errors.New("search unavailable")

func TestWebSearchTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewWebSearchTool(nil)

	require.Equal(t, "web_search", tool.Name())
}

func TestWebSearchTool_ReturnsResults(t *testing.T) {
	t.Parallel()

	search := func(_ context.Context, _ string, _ int) ([]tools.SearchResult, error) {
		return []tools.SearchResult{
			{Title: "Go Documentation", URL: "https://go.dev/doc", Snippet: "Official Go docs"},
			{Title: "Go Blog", URL: "https://go.dev/blog", Snippet: "The Go Blog"},
		}, nil
	}

	tool := tools.NewWebSearchTool(search)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": "golang documentation"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "2 result(s)")
	require.Contains(t, result.Output, "Go Documentation")
	require.Contains(t, result.Output, "https://go.dev/doc")
	require.Contains(t, result.Output, "Official Go docs")
	require.Contains(t, result.Output, "Go Blog")
}

func TestWebSearchTool_DomainFilter(t *testing.T) {
	t.Parallel()

	var capturedQuery string

	search := func(_ context.Context, query string, _ int) ([]tools.SearchResult, error) {
		capturedQuery = query

		return []tools.SearchResult{
			{Title: "Result", URL: "https://golang.org/result", Snippet: "Found it"},
		}, nil
	}

	tool := tools.NewWebSearchTool(search)
	ctx := context.Background()

	params := makeParams(map[string]any{
		"query":  "concurrency",
		"domain": "golang.org",
	})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, capturedQuery, "site:golang.org")
	require.Contains(t, capturedQuery, "concurrency")
}

func TestWebSearchTool_NoResults(t *testing.T) {
	t.Parallel()

	search := func(_ context.Context, _ string, _ int) ([]tools.SearchResult, error) {
		return []tools.SearchResult{}, nil
	}

	tool := tools.NewWebSearchTool(search)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": "asdfghjklzxcvbnm"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "No results found")
}

func TestWebSearchTool_EmptyQuery(t *testing.T) {
	t.Parallel()

	tool := tools.NewWebSearchTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": ""})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestWebSearchTool_MissingQuery(t *testing.T) {
	t.Parallel()

	tool := tools.NewWebSearchTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestWebSearchTool_NilSearcher(t *testing.T) {
	t.Parallel()

	tool := tools.NewWebSearchTool(nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": "test"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestWebSearchTool_SearchError(t *testing.T) {
	t.Parallel()

	search := func(_ context.Context, _ string, _ int) ([]tools.SearchResult, error) {
		return nil, errSearchFailed
	}

	tool := tools.NewWebSearchTool(search)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": "test"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "search unavailable")
}

func TestWebSearchTool_ResultWithoutSnippet(t *testing.T) {
	t.Parallel()

	search := func(_ context.Context, _ string, _ int) ([]tools.SearchResult, error) {
		return []tools.SearchResult{
			{Title: "Title Only", URL: "https://example.com"},
		}, nil
	}

	tool := tools.NewWebSearchTool(search)
	ctx := context.Background()

	params := makeParams(map[string]any{"query": "test"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "Title Only")
	require.Contains(t, result.Output, "https://example.com")
}
