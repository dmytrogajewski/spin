package tools_test

// Journey: specs/journeys/JOURNEY-R8.3.md.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var errFetchFailed = errors.New("connection refused")

func TestFetchURLTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewFetchURLTool(nil, nil)

	require.Equal(t, "fetch_url", tool.Name())
}

func TestFetchURLTool_ReturnsHTML(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, _ string) (*tools.FetchResponse, error) {
		return &tools.FetchResponse{
			StatusCode:  200,
			ContentType: "text/html",
			Body:        []byte("<h1>Hello</h1><p>World</p>"),
		}, nil
	}
	convert := func(_ []byte) string {
		return "# Hello\n\nWorld"
	}

	tool := tools.NewFetchURLTool(fetch, convert)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "# Hello")
	require.Contains(t, result.Output, "World")
}

func TestFetchURLTool_ReturnsNonHTMLText(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		contentType string
		body        string
		url         string
		expected    string
	}{
		{
			name:        "plain text",
			contentType: "text/plain",
			body:        "plain content",
			url:         "https://example.com/robots.txt",
			expected:    "plain content",
		},
		{
			name:        "JSON",
			contentType: "application/json",
			body:        `{"key": "value"}`,
			url:         "https://api.example.com/data",
			expected:    `"key": "value"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := &tools.FetchResponse{
				StatusCode:  200,
				ContentType: tc.contentType,
				Body:        []byte(tc.body),
			}
			fetch := func(_ context.Context, _ string) (*tools.FetchResponse, error) {
				return resp, nil
			}

			tool := tools.NewFetchURLTool(fetch, nil)
			params := makeParams(map[string]any{"url": tc.url})

			result, execErr := tool.Execute(context.Background(), params)
			require.NoError(t, execErr)
			require.True(t, result.Success)
			require.Contains(t, result.Output, tc.expected)
		})
	}
}

func TestFetchURLTool_BlocksBinaryContent(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, _ string) (*tools.FetchResponse, error) {
		return &tools.FetchResponse{
			StatusCode:  200,
			ContentType: "image/png",
			Body:        []byte{0x89, 0x50, 0x4E, 0x47},
		}, nil
	}

	tool := tools.NewFetchURLTool(fetch, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "https://example.com/image.png"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "non-text content type")
}

func TestFetchURLTool_TruncatesLargeOutput(t *testing.T) {
	t.Parallel()

	largeBody := strings.Repeat("x", 60_000)
	fetch := func(_ context.Context, _ string) (*tools.FetchResponse, error) {
		return &tools.FetchResponse{
			StatusCode:  200,
			ContentType: "text/plain",
			Body:        []byte(largeBody),
		}, nil
	}

	tool := tools.NewFetchURLTool(fetch, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "https://example.com/large"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, "... [truncated]")
	require.LessOrEqual(t, len(result.Output), 60_000)
}

func TestFetchURLTool_InvalidURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewFetchURLTool(nil, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "ftp://example.com"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "http://")
}

func TestFetchURLTool_MissingURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewFetchURLTool(nil, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFetchURLTool_NilFetcher(t *testing.T) {
	t.Parallel()

	tool := tools.NewFetchURLTool(nil, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestFetchURLTool_FetchError(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, _ string) (*tools.FetchResponse, error) {
		return nil, errFetchFailed
	}

	tool := tools.NewFetchURLTool(fetch, nil)
	ctx := context.Background()

	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(ctx, params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "connection refused")
}
