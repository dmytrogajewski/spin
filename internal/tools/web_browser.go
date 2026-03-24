package tools

import (
	"context"
	"fmt"
	"strings"
)

const (
	browserToolName = "open_browser"
	fileScheme      = "file://"
	localPathPrefix = "/"
)

// BrowserOpener opens a URL in the system's default browser.
type BrowserOpener func(ctx context.Context, url string) error

// OpenBrowserTool opens a URL in the user's default browser.
type OpenBrowserTool struct {
	open BrowserOpener
}

// NewOpenBrowserTool creates an open_browser tool backed by the given opener.
func NewOpenBrowserTool(open BrowserOpener) *OpenBrowserTool {
	return &OpenBrowserTool{open: open}
}

// Name returns the tool name.
func (t *OpenBrowserTool) Name() string {
	return browserToolName
}

// Description returns a human-readable description.
func (t *OpenBrowserTool) Description() string {
	return "Open a URL in the user's default browser"
}

// Schema returns the parameter schema.
func (t *OpenBrowserTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					paramURL: {
						Type:        "string",
						Description: "The URL or local file path to open",
					},
				},
				Required: []string{paramURL},
			},
		},
	}
}

// Execute opens the URL in the default browser.
func (t *OpenBrowserTool) Execute(
	ctx context.Context, params ToolParameters,
) (ToolResult, error) {
	rawURL := params.GetStringOr(paramURL, "")
	if rawURL == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if t.open == nil {
		return NewToolError(fmt.Errorf(
			"%w: no opener configured", ErrInvalidParameters,
		)), nil
	}

	target := normalizeTarget(rawURL)

	openErr := t.open(ctx, target)
	if openErr != nil {
		return ErrToResultf("open: %s", openErr)
	}

	return NewToolResult(fmt.Sprintf("Opened in browser: %s", target)), nil
}

// normalizeTarget converts local file paths to file:// URIs.
func normalizeTarget(rawURL string) string {
	if strings.HasPrefix(rawURL, httpScheme) ||
		strings.HasPrefix(rawURL, httpsScheme) ||
		strings.HasPrefix(rawURL, fileScheme) {
		return rawURL
	}

	if strings.HasPrefix(rawURL, localPathPrefix) {
		return fileScheme + rawURL
	}

	return rawURL
}
