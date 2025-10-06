// Package ollama provides an Ollama LLM provider implementation.
// Ollama is a tool for running large language models locally with a simple REST API.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

const (
	// DefaultBaseURL is the default Ollama API endpoint.
	DefaultBaseURL = "http://localhost:11434"
)

// Config configures the Ollama provider.
type Config struct {
	// BaseURL is the API endpoint URL (default: http://localhost:11434)
	BaseURL string

	// Model is the model name to use (required)
	Model string

	// Timeout is the request timeout (defaults to 5 minutes)
	Timeout time.Duration
}

// Provider implements the Ollama LLM provider.
type Provider struct {
	client  *llm.HTTPClient
	baseURL string
	model   string
	timeout time.Duration
}

// NewProvider creates a new Ollama provider with automatic retry logic.
//
// The provider automatically retries failed requests on:
//   - Rate limit errors (429)
//   - Service unavailable errors (503)
//   - Gateway timeout errors (504)
//
// Retry behavior:
//   - Maximum retries: 3
//   - Exponential backoff starting at 1 second
//   - Respects Retry-After header
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Default base URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Normalize URL (remove trailing slash)
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Default timeout
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Create HTTP client with retry logic
	client := llm.NewHTTPClient(
		llm.WithTimeout(timeout),
		llm.WithMaxRetries(3),
		llm.WithRetryDelay(time.Second),
	)

	return &Provider{
		client:  client,
		baseURL: baseURL,
		model:   cfg.Model,
		timeout: timeout,
	}, nil
}

// Complete performs a synchronous completion request.
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// Build prompt from messages
	prompt := p.buildPrompt(req.Messages)

	// Build request
	genReq := generateRequest{
		Model:       p.getModel(req.Model),
		Prompt:      prompt,
		Stream:      false,
		Temperature: req.Temperature,
	}

	// Create HTTP request
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/generate", genReq)
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
	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to common format
	return p.convertResponse(&genResp), nil
}

// Stream performs a streaming completion request.
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Use chat API if tools are present, otherwise use generate API
	if len(req.Tools) > 0 {
		return p.streamChat(ctx, req)
	}

	// Build prompt from messages
	prompt := p.buildPrompt(req.Messages)

	// Build streaming request
	genReq := generateRequest{
		Model:       p.getModel(req.Model),
		Prompt:      prompt,
		Stream:      true,
		Temperature: req.Temperature,
	}

	// Create HTTP request
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/generate", genReq)
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
	req, err := p.newRequest(ctx, http.MethodGet, "/api/tags", nil)
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
	var tagsResp tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to common format
	models := make([]llm.Model, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = llm.Model{
			ID:   m.Name,
			Name: m.Name,
		}
	}

	return models, nil
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true, // Supported via /api/chat endpoint
		Vision:          false,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Close closes the provider and releases resources.
func (p *Provider) Close() error {
	return nil
}

// buildPrompt converts structured messages to Ollama's text prompt format.
func (p *Provider) buildPrompt(messages []llm.Message) string {
	var parts []string

	for _, msg := range messages {
		var role string
		switch msg.Role {
		case "system":
			role = "System"
		case "user":
			role = "User"
		case "assistant":
			role = "Assistant"
		case "tool":
			role = "Tool"
		default:
			role = "User"
		}

		if msg.Content != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", role, msg.Content))
		}
	}

	// Join with double newlines for readability
	prompt := strings.Join(parts, "\n\n")

	// Always end with Assistant: to prompt a response
	if prompt != "" {
		prompt += "\n\n"
	}
	prompt += "Assistant:"

	return prompt
}

// convertResponse converts Ollama response to common format.
func (p *Provider) convertResponse(resp *generateResponse) *llm.CompletionResponse {
	result := &llm.CompletionResponse{
		Model:   resp.Model,
		Content: resp.Response,
		Usage: llm.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}

	if resp.Done {
		result.FinishReason = "stop"
	}

	return result
}

// streamResponse processes streaming response.
func (p *Provider) streamResponse(ctx context.Context, r io.Reader, chunks chan<- llm.StreamChunk) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()

		// Parse JSON chunk
		var chunk generateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue // Skip malformed chunks
		}

		// Send content chunk if there's content
		if chunk.Response != "" {
			chunks <- llm.StreamChunk{
				Type:    llm.ChunkTypeContentDelta,
				Content: chunk.Response,
			}
		}

		// Send done chunk if completed
		if chunk.Done {
			chunks <- llm.StreamChunk{
				Type:         llm.ChunkTypeDone,
				FinishReason: "stop",
			}
			return nil
		}
	}

	return scanner.Err()
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
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", llm.ErrModelNotFound, errResp.Error)
	default:
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error)
	}
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(model string) string {
	if model != "" {
		return model
	}
	return p.model
}

// streamChat performs a streaming chat request (with tool support).
func (p *Provider) streamChat(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Convert messages to chat format
	chatMessages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		chatMsg := chatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = make([]chatToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				// Ollama expects arguments as object, not string
				// Parse JSON string into map[string]interface{}
				var args interface{}
				if tc.Function.Arguments != "" {
					var argsMap map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &argsMap); err == nil {
						args = argsMap
					} else {
						// Fallback to string if parsing fails
						args = tc.Function.Arguments
					}
				} else {
					args = map[string]interface{}{}
				}

				chatMsg.ToolCalls[j] = chatToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: chatToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
		}

		chatMessages = append(chatMessages, chatMsg)
	}

	// Convert tools to chat format
	var chatTools []chatTool
	if len(req.Tools) > 0 {
		chatTools = make([]chatTool, len(req.Tools))
		for i, tool := range req.Tools {
			// Convert parameters (interface{} to map[string]interface{})
			params := make(map[string]interface{})
			if tool.Function.Parameters != nil {
				if p, ok := tool.Function.Parameters.(map[string]interface{}); ok {
					params = p
				}
			}

			chatTools[i] = chatTool{
				Type: tool.Type,
				Function: chatToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  params,
				},
			}
		}
	}

	// Build chat request
	chatReq := chatRequest{
		Model:    p.getModel(req.Model),
		Messages: chatMessages,
		Tools:    chatTools,
		Stream:   true,
	}

	if req.Temperature > 0 {
		chatReq.Options = &chatOptions{
			Temperature: req.Temperature,
		}
	}

	if req.MaxTokens > 0 {
		if chatReq.Options == nil {
			chatReq.Options = &chatOptions{}
		}
		chatReq.Options.NumPredict = req.MaxTokens
	}

	log.Printf("[Ollama] Sending chat request: messages=%d, tools=%d", len(chatMessages), len(chatTools))

	// Debug: log message roles and content lengths
	for i, msg := range chatMessages {
		log.Printf("[Ollama]   Msg %d: role=%s, content_len=%d, tool_calls=%d, tool_call_id=%s",
			i, msg.Role, len(msg.Content), len(msg.ToolCalls), msg.ToolCallID)
	}

	// Create HTTP request
	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/chat", chatReq)
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

		if err := p.streamChatResponse(ctx, resp.Body, chunks); err != nil {
			chunks <- llm.StreamChunk{
				Type:  llm.ChunkTypeError,
				Error: err,
			}
		}
	}()

	return chunks, nil
}

// streamChatResponse processes streaming chat response (with tool calls).
func (p *Provider) streamChatResponse(ctx context.Context, r io.Reader, chunks chan<- llm.StreamChunk) error {
	scanner := bufio.NewScanner(r)
	lineCount := 0

	log.Printf("[Ollama] Starting chat stream processing")

	for scanner.Scan() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		lineCount++

		// Parse JSON chunk
		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			log.Printf("[Ollama] Failed to parse chunk line %d: %v", lineCount, err)
			continue // Skip malformed chunks
		}

		// Log every chunk
		log.Printf("[Ollama] Chunk %d: content_len=%d, tool_calls=%d, done=%v",
			lineCount, len(chunk.Message.Content), len(chunk.Message.ToolCalls), chunk.Done)

		// Send content chunk if there's content
		if chunk.Message.Content != "" {
			log.Printf("[Ollama] Sending content chunk: %q", chunk.Message.Content)
			chunks <- llm.StreamChunk{
				Type:    llm.ChunkTypeContentDelta,
				Content: chunk.Message.Content,
			}
		}

		// Send tool calls if present
		if len(chunk.Message.ToolCalls) > 0 {
			log.Printf("[Ollama] Processing %d tool calls", len(chunk.Message.ToolCalls))
			for i, tc := range chunk.Message.ToolCalls {
				// Generate ID if not provided by Ollama
				id := tc.ID
				if id == "" {
					id = fmt.Sprintf("call_%d", i)
				}

				// Get type, default to "function"
				tcType := tc.Type
				if tcType == "" {
					tcType = "function"
				}

				// Convert arguments to JSON string if it's an object
				var argsStr string
				switch args := tc.Function.Arguments.(type) {
				case string:
					argsStr = args
				case map[string]interface{}, []interface{}:
					argsBytes, _ := json.Marshal(args)
					argsStr = string(argsBytes)
				default:
					argsStr = "{}"
				}

				log.Printf("[Ollama] Sending tool call: name=%s, id=%s", tc.Function.Name, id)
				chunks <- llm.StreamChunk{
					Type: llm.ChunkTypeToolCallStart,
					ToolCall: &llm.ToolCall{
						ID:   id,
						Type: tcType,
						Function: llm.FunctionCall{
							Name:      tc.Function.Name,
							Arguments: argsStr,
						},
					},
				}
			}
		}

		// Send done chunk
		if chunk.Done {
			log.Printf("[Ollama] Received done chunk")
			chunks <- llm.StreamChunk{
				Type: llm.ChunkTypeDone,
			}
		}
	}

	log.Printf("[Ollama] Stream processing ended, total lines: %d", lineCount)
	return scanner.Err()
}
