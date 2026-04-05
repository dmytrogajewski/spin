package tools

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

const (
	searchToolName   = "web_search"
	paramQuery       = "query"
	paramDomain      = "domain"
	maxSearchResults = 10
)

// SearchResult represents a single web search result.
type SearchResult struct {
	// Title is the page title.
	Title string
	// URL is the page URL.
	URL string
	// Snippet is a short excerpt from the page.
	Snippet string
}

// WebSearcher performs a web search and returns results.
type WebSearcher func(ctx context.Context, query string, maxResults int) ([]SearchResult, error)

// WebSearchTool searches the web and returns results.
type WebSearchTool struct {
	search WebSearcher
}

// NewWebSearchTool creates a web_search tool backed by the given searcher.
func NewWebSearchTool(search WebSearcher) *WebSearchTool {
	return &WebSearchTool{search: search}
}

// Name returns the tool name.
func (t *WebSearchTool) Name() string {
	return searchToolName
}

// Description returns a human-readable description.
func (t *WebSearchTool) Description() string {
	return "Search the web and return results with titles, URLs, and snippets"
}

// Schema returns the parameter schema.
func (t *WebSearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					paramQuery: {
						Type:        "string",
						Description: "The search query",
					},
					paramDomain: {
						Type:        "string",
						Description: "Optional domain filter (e.g. 'golang.org')",
					},
				},
				Required: []string{paramQuery},
			},
		},
	}
}

// Execute runs the web search and returns formatted results.
func (t *WebSearchTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	query := params.GetStringOr(paramQuery, "")
	if query == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	domain := params.GetStringOr(paramDomain, "")
	if domain != "" {
		query = fmt.Sprintf("site:%s %s", domain, query)
	}

	if t.search == nil {
		return NewToolError(fmt.Errorf("%w: no searcher configured", ErrInvalidParameters)), nil
	}

	results, searchErr := t.search(ctx, query, maxSearchResults)
	if searchErr != nil {
		return ErrToResultf("search: %s", searchErr)
	}

	if len(results) == 0 {
		return NewToolResult("No results found for: " + query), nil
	}

	return NewToolResult(formatSearchResults(results)), nil
}

// formatSearchResults formats search results as a numbered list.
func formatSearchResults(results []SearchResult) string {
	return stringsx.FormatNumberedList(
		results,
		func(n int) string { return fmt.Sprintf("Found %d result(s):\n", n) },
		func(_ int, r SearchResult) string {
			line := fmt.Sprintf("%s\n   %s", r.Title, r.URL)
			if r.Snippet != "" {
				line += fmt.Sprintf("\n   %s", r.Snippet)
			}

			return line
		},
	)
}
