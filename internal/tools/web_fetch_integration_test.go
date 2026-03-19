package tools_test

// Journey: specs/journeys/JOURNEY-4.2.md.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// realFetcher returns a PageFetcher that performs actual HTTP requests.
func realFetcher() tools.PageFetcher {
	return func(ctx context.Context, url string) (*tools.FetchResponse, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		body := make([]byte, 0, 1024)
		buf := make([]byte, 4096)

		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}

			if readErr != nil {
				break
			}
		}

		return &tools.FetchResponse{
			StatusCode:  resp.StatusCode,
			ContentType: resp.Header.Get("Content-Type"),
			Body:        body,
		}, nil
	}
}

func TestFetchURLTool_WithHTMLConversion(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Test</title></head>
<body>
<h1>Welcome</h1>
<p>This is a <strong>test</strong> page with a <a href="https://example.com">link</a>.</p>
<ul><li>Item one</li><li>Item two</li></ul>
</body></html>`)
	}))
	defer srv.Close()

	tool := tools.NewFetchURLTool(realFetcher(), tools.ConvertHTML)
	params := makeParams(map[string]any{"url": srv.URL})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Should contain converted markdown, not raw HTML tags.
	require.Contains(t, result.Output, "# Welcome")
	require.Contains(t, result.Output, "**test**")
	require.Contains(t, result.Output, "[link](https://example.com)")
	require.Contains(t, result.Output, "- Item one")
	require.NotContains(t, result.Output, "<h1>")
	require.NotContains(t, result.Output, "<p>")
	require.NotContains(t, result.Output, "</strong>")
}

func TestFetchURLTool_PlainTextPassthrough(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Hello, this is plain text.\nNo conversion needed.")
	}))
	defer srv.Close()

	tool := tools.NewFetchURLTool(realFetcher(), tools.ConvertHTML)
	params := makeParams(map[string]any{"url": srv.URL})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.True(t, result.Success)

	// Plain text should pass through verbatim without HTML conversion.
	require.Contains(t, result.Output, "Hello, this is plain text.")
	require.Contains(t, result.Output, "No conversion needed.")
}

func TestFetchURLTool_ServerError_ReturnsError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer srv.Close()

	tool := tools.NewFetchURLTool(realFetcher(), tools.ConvertHTML)
	params := makeParams(map[string]any{"url": srv.URL})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	// HTTP 500 should be reported as a tool error with status code.
	require.False(t, result.Success, "HTTP 500 should be an error")
	require.Contains(t, result.Error, "HTTP 500")
	require.Contains(t, result.Error, "Internal Server Error")
}

// TestFetchURLTool_NotFound_ReportsStatusCode reproduces the bug where HTTP 404
// returned an empty successful result instead of reporting the error status code.
func TestFetchURLTool_NotFound_ReportsStatusCode(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tool := tools.NewFetchURLTool(realFetcher(), tools.ConvertHTML)
	params := makeParams(map[string]any{"url": srv.URL})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)
	require.False(t, result.Success, "HTTP 404 should be an error")
	require.Contains(t, result.Error, "HTTP 404")
	require.Contains(t, result.Error, "empty response body")
}

func TestFetchURLTool_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "too late")
	}))
	defer srv.Close()

	tool := tools.NewFetchURLTool(realFetcher(), tools.ConvertHTML)
	params := makeParams(map[string]any{"url": srv.URL})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)

	// The fetcher should propagate the context cancellation as a tool error.
	require.False(t, result.Success)
	require.NotEmpty(t, result.Error)
}
