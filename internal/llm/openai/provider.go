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
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// Validate validates the OpenAI configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %v", c.Timeout)
	}
	return nil
}

// Provider implements the OpenAI-compatible LLM provider.
type Provider struct {
	client      *llm.HTTPClient
	errorMapper *llm.ErrorMapper
	baseURL     string
	apiKey      string
	model       string
	timeout     time.Duration
}

// NewProvider creates a new OpenAI-compatible provider.
func NewProvider(cfg Config) (*Provider, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Normalize URL (remove trailing slash)
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	// Use timeout from config or default
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = llm.DefaultTimeout
	}

	// Create HTTP client with timeout
	client := llm.NewHTTPClient(
		llm.WithTimeout(timeout),
		llm.WithMaxRetries(3),
	)

	// Create error mapper for standardized error handling
	errorMapper := llm.NewErrorMapper("openai")

	return &Provider{
		client:      client,
		errorMapper: errorMapper,
		baseURL:     baseURL,
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		timeout:     timeout,
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
		return nil, p.errorMapper.MapError(fmt.Errorf("http request: %w", err))
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, p.errorMapper.MapError(p.handleError(resp))
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
		return nil, p.errorMapper.MapError(fmt.Errorf("http request: %w", err))
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.errorMapper.MapError(p.handleError(resp))
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
	messages := p.convertMessages(req.Messages)

	body := map[string]interface{}{
		"model":    p.getModel(req.Model),
		"messages": messages,
		"stream":   stream,
	}

	p.addOptionalParameters(body, req)
	p.addTools(body, req.Tools)

	return body
}

// convertMessages converts LLM messages to OpenAI format.
func (p *Provider) convertMessages(messages []llm.Message) []interface{} {
	result := make([]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = p.convertMessage(msg)
	}
	return result
}

// convertMessage converts a single LLM message to OpenAI format.
func (p *Provider) convertMessage(msg llm.Message) map[string]interface{} {
	m := map[string]interface{}{
		"role":    msg.Role,
		"content": msg.Content,
	}

	if len(msg.ToolCalls) > 0 {
		m["tool_calls"] = p.convertToolCalls(msg.ToolCalls)
	}

	if msg.ToolCallID != "" {
		m["tool_call_id"] = msg.ToolCallID
	}

	return m
}

// convertToolCalls converts LLM tool calls to OpenAI format.
func (p *Provider) convertToolCalls(toolCalls []llm.ToolCall) []map[string]interface{} {
	result := make([]map[string]interface{}, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = map[string]interface{}{
			"id":   tc.ID,
			"type": tc.Type,
			"function": map[string]interface{}{
				"name":      tc.Function.Name,
				"arguments": tc.Function.Arguments,
			},
		}
	}
	return result
}

// addOptionalParameters adds optional parameters to the request body.
func (p *Provider) addOptionalParameters(body map[string]interface{}, req llm.CompletionRequest) {
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
}

// addTools adds tools to the request body.
func (p *Provider) addTools(body map[string]interface{}, tools []llm.Tool) {
	if len(tools) == 0 {
		return
	}

	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"type": tool.Type,
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		}
	}
	body["tools"] = result
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
	result := p.buildBaseResponse(resp, choice)
	result.ToolCalls = p.convertResponseToolCalls(choice.Message.ToolCalls)

	return result
}

// buildBaseResponse builds the base response structure.
func (p *Provider) buildBaseResponse(resp *chatCompletionResponse, choice chatCompletionChoice) *llm.CompletionResponse {
	return &llm.CompletionResponse{
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
}

// convertResponseToolCalls converts OpenAI tool calls to LLM format.
func (p *Provider) convertResponseToolCalls(toolCalls []chatToolCall) []llm.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]llm.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = llm.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: llm.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
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
