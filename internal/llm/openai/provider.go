// Package openai provides an OpenAI-compatible LLM provider implementation.
// It supports the OpenAI Chat Completions API format, making it compatible with
// OpenAI, Azure OpenAI, and other services that implement the same API.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// Config configures the OpenAI provider.
type Config struct {
	// BaseURL is the API endpoint URL (e.g., https://api.openai.com/v1)
	BaseURL string

	// APIKey is the API key for authentication (optional for local providers)
	APIKey string

	// Model is the default model name to use
	Model string

	// Timeout is the request timeout (defaults to 5 minutes)
	Timeout time.Duration
}

// Provider implements the OpenAI-compatible LLM provider.
type Provider struct {
	client  *llm.HTTPClient
	baseURL string
	apiKey  string
	model   string
	timeout time.Duration
}

// NewProvider creates a new OpenAI-compatible provider.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Normalize URL (remove trailing slash)
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	// Default timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Create HTTP client with timeout
	client := llm.NewHTTPClient(
		llm.WithTimeout(timeout),
		llm.WithMaxRetries(3),
	)

	return &Provider{
		client:  client,
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		timeout: timeout,
	}, nil
}

// Complete performs a synchronous completion request.
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// Build request body
	reqBody := p.buildRequest(req, false)

	// Create HTTP request
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/chat/completions", reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Make HTTP request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	// Parse response
	var apiResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to common format
	return p.convertResponse(&apiResp), nil
}

// Stream performs a streaming completion request.
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Build streaming request
	reqBody := p.buildRequest(req, true)

	// Create HTTP request
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/chat/completions", reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Make HTTP request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.handleError(resp)
	}

	// Create channel for chunks
	chunks := make(chan llm.StreamChunk, 10)

	// Start streaming in goroutine
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		if err := p.streamResponse(ctx, resp.Body, chunks); err != nil {
			chunks <- llm.StreamChunk{
				Type:  llm.ChunkTypeError,
				Error: err,
			}
		}
	}()

	return chunks, nil
}

// Models returns the list of available models.
func (p *Provider) Models(ctx context.Context) ([]llm.Model, error) {
	// Create HTTP request
	req, err := p.newRequest(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Make HTTP request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	// Parse response
	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to common format
	models := make([]llm.Model, len(result.Data))
	for i, m := range result.Data {
		models[i] = llm.Model{
			ID:   m.ID,
			Name: m.ID,
		}
	}

	return models, nil
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
		Vision:          false,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai-compatible"
}

// Close closes the provider and releases resources.
func (p *Provider) Close() error {
	return nil
}

// buildRequest builds the OpenAI API request body.
func (p *Provider) buildRequest(req llm.CompletionRequest, stream bool) map[string]interface{} {
	// Convert messages
	messages := make([]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		m := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}

		// Add tool calls if present
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]map[string]interface{}, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = map[string]interface{}{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]interface{}{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
			}
			m["tool_calls"] = toolCalls
		}

		// Add tool call ID if present (for tool responses)
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}

		messages[i] = m
	}

	// Build request body
	body := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": messages,
		"stream":   stream,
	}

	// Add optional parameters
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"type": tool.Type,
				"function": map[string]interface{}{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				},
			}
		}
		body["tools"] = tools
	}

	return body
}

// convertResponse converts OpenAI response to common format.
func (p *Provider) convertResponse(resp *chatCompletionResponse) *llm.CompletionResponse {
	if len(resp.Choices) == 0 {
		return &llm.CompletionResponse{
			ID:    resp.ID,
			Model: resp.Model,
		}
	}

	choice := resp.Choices[0]
	result := &llm.CompletionResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]llm.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = llm.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: llm.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return result
}

// streamResponse processes streaming response using the shared StreamSSE function.
func (p *Provider) streamResponse(ctx context.Context, r io.Reader, chunks chan<- llm.StreamChunk) error {
	return llm.StreamSSE(ctx, r, chunks, p.parseChunk)
}

// parseChunk parses OpenAI SSE event data into a StreamChunk.
func (p *Provider) parseChunk(data []byte) (*llm.StreamChunk, error) {
	var chunk chatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, nil // Skip malformed chunks
	}
	return p.convertChunk(&chunk), nil
}

// convertChunk converts OpenAI chunk to common format.
func (p *Provider) convertChunk(chunk *chatCompletionChunk) *llm.StreamChunk {
	if len(chunk.Choices) == 0 {
		return nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// Content delta
	if delta.Content != "" {
		return &llm.StreamChunk{
			Type:    llm.ChunkTypeContentDelta,
			Content: delta.Content,
		}
	}

	// Tool call delta
	if len(delta.ToolCalls) > 0 {
		tc := delta.ToolCalls[0]
		return &llm.StreamChunk{
			Type: llm.ChunkTypeToolCallDelta,
			ToolCall: &llm.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: llm.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			},
		}
	}

	// Finish reason
	if choice.FinishReason != nil && *choice.FinishReason != "" {
		return &llm.StreamChunk{
			Type:         llm.ChunkTypeDone,
			FinishReason: *choice.FinishReason,
		}
	}

	return nil
}

// newRequest creates an HTTP request.
func (p *Provider) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// handleError parses and returns an error from HTTP response.
func (p *Provider) handleError(resp *http.Response) error {
	var errResp errorResponse

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	// Try to parse error response
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Map to common error types
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: %s", errResp.Error.Message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", llm.ErrRateLimited, errResp.Error.Message)
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		return fmt.Errorf("server error: %s", errResp.Error.Message)
	default:
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error.Message)
	}
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(model string) string {
	if model != "" {
		return model
	}
	return p.model
}
