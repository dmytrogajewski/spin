package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// SmitheryAPIClient provides access to Smithery's tool discovery API.
type SmitheryAPIClient struct {
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// SmitheryAPIConfig holds configuration for creating a Smithery API client.
type SmitheryAPIConfig struct {
	APIKey  string
	Timeout time.Duration
	Logger  *slog.Logger
}

// NewSmitheryAPIClient creates a new Smithery API client.
func NewSmitheryAPIClient(config SmitheryAPIConfig) (*SmitheryAPIClient, error) {
	if config.APIKey == "" {
		return nil, ErrSmitheryAPIKeyRequired
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &SmitheryAPIClient{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: config.Logger,
	}, nil
}

// SmitheryToolSearchResult represents a tool found in Smithery API search.
type SmitheryToolSearchResult struct {
	Tool   SmitheryToolInfo   `json:"tool"`
	Server SmitheryServerInfo `json:"server"`
}

// SmitheryToolInfo contains tool metadata from Smithery API.
type SmitheryToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SmitheryServerInfo contains server metadata from Smithery API.
type SmitheryServerInfo struct {
	QualifiedName string `json:"qualifiedName"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	HomepageURL   string `json:"homepageUrl"`
	Verified      bool   `json:"verified"`
}

// SmitherySearchResponse is the response from Smithery tools search API.
type SmitherySearchResponse struct {
	Tools      []SmitheryToolSearchResult `json:"tools"`
	TotalCount int                        `json:"totalCount"`
	Page       int                        `json:"page"`
	PageSize   int                        `json:"pageSize"`
}

// SearchTools searches Smithery API for tools matching the query.
func (c *SmitheryAPIClient) SearchTools(ctx context.Context, query string, limit int) (*SmitherySearchResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	apiURL := fmt.Sprintf("https://api.smithery.ai/tools?q=%s&pageSize=%d",
		url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	if c.logger != nil {
		c.logger.DebugContext(ctx, "searching Smithery API", "query", query, "limit", limit)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

return nil, fmt.Errorf("API error (status %d): %s: %w", resp.StatusCode, string(body), ErrAPIErrorStatus)
	}

	var result SmitherySearchResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if c.logger != nil {
		c.logger.DebugContext(ctx, "Smithery search complete", "query", query, "found", len(result.Tools))
	}

	return &result, nil
}

// GetServerMCPURL returns the MCP URL for connecting to a Smithery server.
// serverPath is the qualified name like "gmail" or "@namespace/server-name".
// Returns URL in format: https://server.smithery.ai/qualifiedName/mcp
// Note: The @ prefix is preserved if present - some servers require it.
func (c *SmitheryAPIClient) GetServerMCPURL(serverPath string) string {
	// Keep the serverPath as-is - the API returns the correct format
	// Some servers are like "asana", others are like "@namespace/server".
	return fmt.Sprintf("https://server.smithery.ai/%s/mcp", serverPath)
}
