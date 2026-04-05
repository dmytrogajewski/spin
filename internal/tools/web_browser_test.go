package tools_test

// Journey: specs/journeys/JOURNEY-R8.4.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var errBrowserFailed = errors.New("no display available")

func TestOpenBrowserTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewOpenBrowserTool(nil)

	require.Equal(t, "open_browser", tool.Name())
}

func TestOpenBrowserTool_OpensURL(t *testing.T) {
	t.Parallel()

	var openedURL string

	opener := func(_ context.Context, u string) error {
		openedURL = u

		return nil
	}

	tool := tools.NewOpenBrowserTool(opener)
	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "https://example.com")
	require.Equal(t, "https://example.com", openedURL)
}

func TestOpenBrowserTool_LocalPathToFileURI(t *testing.T) {
	t.Parallel()

	var openedURL string

	opener := func(_ context.Context, u string) error {
		openedURL = u

		return nil
	}

	tool := tools.NewOpenBrowserTool(opener)
	params := makeParams(map[string]any{"url": "/tmp/index.html"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Equal(t, "file:///tmp/index.html", openedURL)
}

func TestOpenBrowserTool_MissingURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewOpenBrowserTool(nil)
	params := makeParams(map[string]any{})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestOpenBrowserTool_NilOpener(t *testing.T) {
	t.Parallel()

	tool := tools.NewOpenBrowserTool(nil)
	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "no opener configured")
}

func TestOpenBrowserTool_OpenerError(t *testing.T) {
	t.Parallel()

	opener := func(_ context.Context, _ string) error {
		return errBrowserFailed
	}

	tool := tools.NewOpenBrowserTool(opener)
	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "no display available")
}

func TestOpenBrowserTool_FileSchemePassthrough(t *testing.T) {
	t.Parallel()

	var openedURL string

	opener := func(_ context.Context, u string) error {
		openedURL = u

		return nil
	}

	tool := tools.NewOpenBrowserTool(opener)
	params := makeParams(map[string]any{"url": "file:///tmp/index.html"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Equal(t, "file:///tmp/index.html", openedURL)
}

func TestOpenBrowserTool_EmptyURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewOpenBrowserTool(nil)
	params := makeParams(map[string]any{"url": ""})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}
