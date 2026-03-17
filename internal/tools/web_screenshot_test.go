package tools_test

// Journey: specs/journeys/JOURNEY-R8.4.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var errScreenshotFailed = errors.New("headless browser not available")

const testScreenshotPath = "/tmp/screenshot.png"

func TestScreenshotTool_Name(t *testing.T) {
	t.Parallel()

	tool := tools.NewScreenshotTool(nil)

	require.Equal(t, "capture_web_screenshot", tool.Name())
}

func TestScreenshotTool_ExecuteWithCapturer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		capture   tools.ScreenshotCapturer
		wantOK    bool
		wantMatch string
	}{
		{
			name: "success",
			capture: func(_ context.Context, _ string, _, _ int, _ bool) (string, error) {
				return "/tmp/screenshot-abc123.png", nil
			},
			wantOK:    true,
			wantMatch: "/tmp/screenshot-abc123.png",
		},
		{
			name: "capture error",
			capture: func(_ context.Context, _ string, _, _ int, _ bool) (string, error) {
				return "", errScreenshotFailed
			},
			wantOK:    false,
			wantMatch: "headless browser not available",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tool := tools.NewScreenshotTool(tc.capture)
			params := makeParams(map[string]any{"url": "https://example.com"})

			result, execErr := tool.Execute(context.Background(), params)
			require.NoError(t, execErr)
			require.Equal(t, tc.wantOK, result.Success)

			if tc.wantOK {
				require.Contains(t, result.Output, tc.wantMatch)
			} else {
				require.Contains(t, result.Error, tc.wantMatch)
			}
		})
	}
}

func TestScreenshotTool_Viewport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		inputWidth     int
		inputHeight    int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "custom viewport",
			inputWidth:     1280,
			inputHeight:    720,
			expectedWidth:  1280,
			expectedHeight: 720,
		},
		{
			name:           "default viewport",
			inputWidth:     0,
			inputHeight:    0,
			expectedWidth:  1920,
			expectedHeight: 1080,
		},
		{
			name:           "capped viewport",
			inputWidth:     9999,
			inputHeight:    9999,
			expectedWidth:  3840,
			expectedHeight: 2160,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var capturedWidth, capturedHeight int

			capture := func(_ context.Context, _ string, w, h int, _ bool) (string, error) {
				capturedWidth = w
				capturedHeight = h

				return testScreenshotPath, nil
			}

			tool := tools.NewScreenshotTool(capture)

			p := map[string]any{"url": "https://example.com"}
			if tc.inputWidth > 0 {
				p["width"] = tc.inputWidth
			}

			if tc.inputHeight > 0 {
				p["height"] = tc.inputHeight
			}

			params := makeParams(p)

			result, execErr := tool.Execute(context.Background(), params)
			require.NoError(t, execErr)
			require.True(t, result.Success)
			require.Equal(t, tc.expectedWidth, capturedWidth)
			require.Equal(t, tc.expectedHeight, capturedHeight)
		})
	}
}

func TestScreenshotTool_FullPage(t *testing.T) {
	t.Parallel()

	var capturedFullPage bool

	capture := func(_ context.Context, _ string, _, _ int, fullPage bool) (string, error) {
		capturedFullPage = fullPage

		return testScreenshotPath, nil
	}

	tool := tools.NewScreenshotTool(capture)
	params := makeParams(map[string]any{
		"url":       "https://example.com",
		"full_page": true,
	})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.True(t, capturedFullPage)
}

func TestScreenshotTool_InvalidURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewScreenshotTool(nil)
	params := makeParams(map[string]any{"url": "ftp://example.com"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "http://")
}

func TestScreenshotTool_MissingURL(t *testing.T) {
	t.Parallel()

	tool := tools.NewScreenshotTool(nil)
	params := makeParams(map[string]any{})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
}

func TestScreenshotTool_NilCapturer(t *testing.T) {
	t.Parallel()

	tool := tools.NewScreenshotTool(nil)
	params := makeParams(map[string]any{"url": "https://example.com"})

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "no capturer configured")
}
