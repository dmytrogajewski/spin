package tools

import (
	"context"
	"fmt"
	"strings"
)

const (
	fetchToolName      = "fetch_url"
	paramURL           = "url"
	maxOutputChars     = 50_000
	httpErrorThreshold = 400
	httpScheme         = "http://"
	httpsScheme        = "https://"
	contentTypeHTML    = "text/html"
	contentTypeJSON    = "application/json"
	contentTypeText    = "text/"
)

// FetchResponse holds the result of an HTTP GET request.
type FetchResponse struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// ContentType is the Content-Type header value.
	ContentType string
	// Body is the response body.
	Body []byte
}

// PageFetcher retrieves a web page by URL.
type PageFetcher func(ctx context.Context, url string) (*FetchResponse, error)

// FetchURLTool retrieves web content and converts HTML to markdown.
type FetchURLTool struct {
	fetch   PageFetcher
	convert HTMLConverter
}

// NewFetchURLTool creates a fetch_url tool backed by the given fetcher and converter.
func NewFetchURLTool(fetch PageFetcher, convert HTMLConverter) *FetchURLTool {
	return &FetchURLTool{fetch: fetch, convert: convert}
}

// Name returns the tool name.
func (t *FetchURLTool) Name() string {
	return fetchToolName
}

// Description returns a human-readable description.
func (t *FetchURLTool) Description() string {
	return "Fetch a URL and return its content as markdown"
}

// Schema returns the parameter schema.
func (t *FetchURLTool) Schema() ToolSchema {
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
						Description: "The URL to fetch (must start with http:// or https://)",
					},
				},
				Required: []string{paramURL},
			},
		},
	}
}

// Execute fetches the URL, converts HTML to markdown, and returns the content.
func (t *FetchURLTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	rawURL, _ := params.GetString(paramURL)
	if rawURL == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if !isValidURL(rawURL) {
		return NewToolError(fmt.Errorf(
			"%w: URL must start with %s or %s",
			ErrInvalidParameters, httpScheme, httpsScheme,
		)), nil
	}

	if t.fetch == nil {
		return NewToolError(fmt.Errorf("%w: no fetcher configured", ErrInvalidParameters)), nil
	}

	resp, fetchErr := t.fetch(ctx, rawURL)
	if fetchErr != nil {
		return ErrToResultf("fetch: %s", fetchErr)
	}

	// Report HTTP error status codes to the agent.
	if resp.StatusCode >= httpErrorThreshold {
		body := string(resp.Body)
		if body == "" {
			body = "(empty response body)"
		}

		return NewToolError(fmt.Errorf("HTTP %d: %s: %w", resp.StatusCode, capOutput(body, maxOutputChars), errHTTPError)), nil
	}

	if !isTextContent(resp.ContentType) {
		return NewToolError(fmt.Errorf(
			"%w: non-text content type %q", ErrInvalidParameters, resp.ContentType,
		)), nil
	}

	content := convertResponse(resp, t.convert)

	return NewToolResult(capOutput(content, maxOutputChars)), nil
}

// isValidURL checks if the URL starts with http:// or https://.
func isValidURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, httpScheme) || strings.HasPrefix(rawURL, httpsScheme)
}

// isTextContent checks if the Content-Type header indicates text content.
func isTextContent(contentType string) bool {
	lower := strings.ToLower(contentType)

	return strings.HasPrefix(lower, contentTypeText) ||
		strings.HasPrefix(lower, contentTypeJSON) ||
		strings.HasPrefix(lower, contentTypeHTML)
}

// convertResponse converts the response body based on content type.
func convertResponse(resp *FetchResponse, convert HTMLConverter) string {
	lower := strings.ToLower(resp.ContentType)

	if strings.HasPrefix(lower, contentTypeHTML) && convert != nil {
		return convert(resp.Body)
	}

	return string(resp.Body)
}

// capOutput truncates text to maxChars, appending a marker if truncated.
func capOutput(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}

	return text[:maxChars] + "\n\n... [truncated]"
}
