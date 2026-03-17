package tools

import (
	"context"
	"fmt"
	"strings"
)

const (
	screenshotToolName = "capture_web_screenshot"
	paramWidth         = "width"
	paramHeight        = "height"
	paramFullPage      = "full_page"
	defaultWidth       = 1920
	defaultHeight      = 1080
	maxWidth           = 3840
	maxHeight          = 2160
)

// ScreenshotCapturer captures a screenshot of a web page.
// It returns the file path to the saved PNG image.
type ScreenshotCapturer func(
	ctx context.Context, url string, width, height int, fullPage bool,
) (string, error)

// ScreenshotTool captures a web page screenshot using a headless browser.
type ScreenshotTool struct {
	capture ScreenshotCapturer
}

// NewScreenshotTool creates a capture_web_screenshot tool backed by the given capturer.
func NewScreenshotTool(capture ScreenshotCapturer) *ScreenshotTool {
	return &ScreenshotTool{capture: capture}
}

// Name returns the tool name.
func (t *ScreenshotTool) Name() string {
	return screenshotToolName
}

// Description returns a human-readable description.
func (t *ScreenshotTool) Description() string {
	return "Capture a screenshot of a web page using a headless browser"
}

// Schema returns the parameter schema.
func (t *ScreenshotTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type:       "object",
				Properties: screenshotProperties(),
				Required:   []string{paramURL},
			},
		},
	}
}

// Execute captures the screenshot and returns the file path.
func (t *ScreenshotTool) Execute(
	ctx context.Context, params ToolParameters,
) (ToolResult, error) {
	rawURL, _ := params.GetString(paramURL)
	if rawURL == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if !isScreenshotURL(rawURL) {
		return NewToolError(fmt.Errorf(
			"%w: URL must start with %s or %s",
			ErrInvalidParameters, httpScheme, httpsScheme,
		)), nil
	}

	if t.capture == nil {
		return NewToolError(fmt.Errorf(
			"%w: no capturer configured", ErrInvalidParameters,
		)), nil
	}

	width := clampViewport(params.GetIntOr(paramWidth, defaultWidth), maxWidth)
	height := clampViewport(params.GetIntOr(paramHeight, defaultHeight), maxHeight)
	fullPage := params.GetBoolOr(paramFullPage, false)

	filePath, captureErr := t.capture(ctx, rawURL, width, height, fullPage)
	if captureErr != nil {
		return ErrToResultf("screenshot: %s", captureErr)
	}

	return NewToolResult(fmt.Sprintf("Screenshot saved to: %s", filePath)), nil
}

// screenshotProperties returns the property definitions for screenshot parameters.
func screenshotProperties() map[string]PropertyDefinition {
	return map[string]PropertyDefinition{
		paramURL: {
			Type:        "string",
			Description: "The URL to capture (must start with http:// or https://)",
		},
		paramWidth: {
			Type:        "number",
			Description: "Viewport width in pixels (default 1920, max 3840)",
		},
		paramHeight: {
			Type:        "number",
			Description: "Viewport height in pixels (default 1080, max 2160)",
		},
		paramFullPage: {
			Type:        "boolean",
			Description: "Capture the full scrollable page (default false)",
		},
	}
}

// isScreenshotURL checks if the URL is valid for screenshot capture.
func isScreenshotURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, httpScheme) || strings.HasPrefix(rawURL, httpsScheme)
}

// clampViewport clamps a viewport dimension to a maximum value.
func clampViewport(value, maxValue int) int {
	if value > maxValue {
		return maxValue
	}

	if value <= 0 {
		return maxValue
	}

	return value
}
