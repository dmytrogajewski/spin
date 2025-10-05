package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNewProvider tests provider construction
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
				Model:   "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "empty base URL",
			cfg: Config{
				APIKey: "test-key",
				Model:  "gpt-4",
			},
			wantErr: true,
			errMsg:  "base URL",
		},
		{
			name: "empty API key allowed",
			cfg: Config{
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama2",
			},
			wantErr: false,
		},
		{
			name: "URL with trailing slash",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1/",
				APIKey:  "test-key",
				Model:   "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "default timeout applied",
			cfg: Config{
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewProvider() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewProvider() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewProvider() unexpected error = %v", err)
				return
			}

			if p == nil {
				t.Fatal("NewProvider() returned nil provider")
			}

			// Verify URL normalization (trailing slash removed)
			if strings.HasSuffix(p.baseURL, "/") {
				t.Errorf("Provider baseURL = %q, should not have trailing slash", p.baseURL)
			}

			// Verify default timeout
			if p.timeout == 0 {
				t.Error("Provider timeout not set")
			}
		})
	}
}

// TestProvider_Name tests Name() method
func TestProvider_Name(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	name := p.Name()
	want := "openai-compatible"

	if name != want {
		t.Errorf("Name() = %q, want %q", name, want)
	}
}

// TestProvider_Capabilities tests Capabilities() method
func TestProvider_Capabilities(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	caps := p.Capabilities()

	if !caps.Streaming {
		t.Error("Capabilities().Streaming = false, want true")
	}
	if !caps.FunctionCalling {
		t.Error("Capabilities().FunctionCalling = false, want true")
	}
	if caps.Vision {
		t.Error("Capabilities().Vision = true, want false")
	}
}

// TestProvider_Complete tests synchronous completions
func TestProvider_Complete(t *testing.T) {
	tests := []struct {
		name       string
		req        llm.CompletionRequest
		serverResp string
		statusCode int
		wantErr    bool
		checkResp  func(t *testing.T, resp *llm.CompletionResponse)
	}{
		{
			name: "successful completion",
			req: llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			serverResp: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! How can I help you?"
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			checkResp: func(t *testing.T, resp *llm.CompletionResponse) {
				if resp.Content != "Hello! How can I help you?" {
					t.Errorf("Response content = %q, want %q", resp.Content, "Hello! How can I help you?")
				}
				if resp.Usage.TotalTokens != 30 {
					t.Errorf("Response usage = %d, want 30", resp.Usage.TotalTokens)
				}
			},
		},
		{
			name: "completion with tool calls",
			req: llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: "user", Content: "What's the weather?"},
				},
				Tools: []llm.Tool{
					{
						Type: "function",
						Function: llm.Function{
							Name:        "get_weather",
							Description: "Get weather",
						},
					},
				},
			},
			serverResp: `{
				"id": "chatcmpl-124",
				"object": "chat.completion",
				"created": 1677652289,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "",
						"tool_calls": [{
							"id": "call_1",
							"type": "function",
							"function": {
								"name": "get_weather",
								"arguments": "{\"location\":\"London\"}"
							}
						}]
					},
					"finish_reason": "tool_calls"
				}],
				"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			checkResp: func(t *testing.T, resp *llm.CompletionResponse) {
				if len(resp.ToolCalls) != 1 {
					t.Errorf("ToolCalls count = %d, want 1", len(resp.ToolCalls))
					return
				}
				if resp.ToolCalls[0].Function.Name != "get_weather" {
					t.Errorf("ToolCall name = %q, want %q", resp.ToolCalls[0].Function.Name, "get_weather")
				}
			},
		},
		{
			name: "401 unauthorized",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverResp: `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "429 rate limited",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverResp: `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_exceeded"}}`,
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
		},
		{
			name: "500 server error",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverResp: `{"error": {"message": "Internal server error", "type": "server_error"}}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Request method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Request path = %s, want /chat/completions", r.URL.Path)
				}

				// Verify Content-Type
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %s, want application/json", ct)
				}

				// Write response
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResp))
			}))
			defer server.Close()

			// Create provider
			p, err := NewProvider(Config{
				BaseURL: server.URL,
				APIKey:  "test-key",
				Model:   "gpt-4",
			})
			if err != nil {
				t.Fatalf("NewProvider() error = %v", err)
			}

			// Call Complete
			resp, err := p.Complete(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("Complete() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Complete() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Complete() returned nil response")
			}

			if tt.checkResp != nil {
				tt.checkResp(t, resp)
			}
		})
	}
}

// TestProvider_Complete_ContextCancellation tests context cancellation
func TestProvider_Complete_ContextCancellation(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"message":{"content":"test"}}]}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "gpt-4",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	}

	_, err := p.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() with cancelled context should return error")
	}
}

// TestProvider_Stream tests streaming completions
func TestProvider_Stream(t *testing.T) {
	tests := []struct {
		name       string
		req        llm.CompletionRequest
		serverData string
		wantChunks int
		wantErr    bool
		checkChunk func(t *testing.T, chunks []llm.StreamChunk)
	}{
		{
			name: "successful stream",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "Hello"}},
			},
			serverData: "data: " + `{"id":"1","choices":[{"delta":{"content":"Hello"}}]}` + "\n\n" +
				"data: " + `{"id":"1","choices":[{"delta":{"content":" there"}}]}` + "\n\n" +
				"data: " + `{"id":"1","choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
				"data: [DONE]\n\n",
			wantChunks: 4,
			checkChunk: func(t *testing.T, chunks []llm.StreamChunk) {
				// Check content chunks
				if chunks[0].Content != "Hello" {
					t.Errorf("First chunk content = %q, want %q", chunks[0].Content, "Hello")
				}
				if chunks[1].Content != " there" {
					t.Errorf("Second chunk content = %q, want %q", chunks[1].Content, " there")
				}
				// Last chunk should be done
				if chunks[len(chunks)-1].Type != llm.ChunkTypeDone {
					t.Error("Last chunk should be ChunkTypeDone")
				}
			},
		},
		{
			name: "stream with tool call",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverData: "data: " + `{"id":"1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"test_func","arguments":"{}"}}]}}]}` + "\n\n" +
				"data: [DONE]\n\n",
			wantChunks: 2,
			checkChunk: func(t *testing.T, chunks []llm.StreamChunk) {
				if chunks[0].ToolCall == nil {
					t.Error("Expected tool call chunk")
					return
				}
				if chunks[0].ToolCall.Function.Name != "test_func" {
					t.Errorf("Tool call name = %q, want %q", chunks[0].ToolCall.Function.Name, "test_func")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify streaming request
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Request path = %s, want /chat/completions", r.URL.Path)
				}

				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				// Write SSE data
				w.Write([]byte(tt.serverData))
			}))
			defer server.Close()

			p, _ := NewProvider(Config{
				BaseURL: server.URL,
				Model:   "gpt-4",
			})

			// Call Stream
			chunks, err := p.Stream(context.Background(), tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("Stream() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Stream() unexpected error = %v", err)
				return
			}

			// Collect chunks
			var collected []llm.StreamChunk
			for chunk := range chunks {
				if chunk.Type == llm.ChunkTypeError {
					t.Errorf("Received error chunk: %v", chunk.Error)
				}
				collected = append(collected, chunk)
			}

			if len(collected) != tt.wantChunks {
				t.Errorf("Stream() got %d chunks, want %d", len(collected), tt.wantChunks)
			}

			if tt.checkChunk != nil {
				tt.checkChunk(t, collected)
			}
		})
	}
}

// TestProvider_Models tests model listing
func TestProvider_Models(t *testing.T) {
	tests := []struct {
		name       string
		serverResp string
		statusCode int
		wantErr    bool
		wantCount  int
	}{
		{
			name: "successful model list",
			serverResp: `{
				"data": [
					{"id": "gpt-4", "created": 1677652288},
					{"id": "gpt-3.5-turbo", "created": 1677652288}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  2,
		},
		{
			name:       "empty model list",
			serverResp: `{"data": []}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "401 error",
			serverResp: `{"error": {"message": "Invalid API key"}}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/models" {
					t.Errorf("Request path = %s, want /models", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResp))
			}))
			defer server.Close()

			p, _ := NewProvider(Config{
				BaseURL: server.URL,
				Model:   "gpt-4",
			})

			models, err := p.Models(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("Models() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Models() unexpected error = %v", err)
				return
			}

			if len(models) != tt.wantCount {
				t.Errorf("Models() count = %d, want %d", len(models), tt.wantCount)
			}
		})
	}
}

// TestProvider_Close tests resource cleanup
func TestProvider_Close(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	err := p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

// TestProvider_Stream_ContextCancellation tests context cancellation during streaming
func TestProvider_Stream_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Write indefinitely (will be cancelled by context)
		for i := 0; i < 100; i++ {
			w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"delta\":{\"content\":\"test\"}}]}\n\n"))
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "gpt-4",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	chunks, err := p.Stream(ctx, llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err != nil {
		// Context may have already expired by the time request is made
		// This is acceptable
		return
	}

	// Collect chunks until cancelled
	count := 0
	for chunk := range chunks {
		if chunk.Type == llm.ChunkTypeError {
			// Expected context error
			break
		}
		count++
		if count > 5 {
			// Got enough chunks, cancel now
			cancel()
		}
	}

	// We should have received some chunks before cancellation
	if count == 0 {
		t.Error("Expected at least some chunks before cancellation")
	}
}

// TestProvider_Stream_ErrorResponse tests streaming with error response
func TestProvider_Stream_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "gpt-4",
	})

	_, err := p.Stream(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err == nil {
		t.Error("Stream() with error response should return error")
	}
}

// TestGetModel tests getModel function
func TestGetModel(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "default-model",
	})

	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "empty model uses default",
			model: "",
			want:  "default-model",
		},
		{
			name:  "specified model used",
			model: "custom-model",
			want:  "custom-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.getModel(tt.model)
			if got != tt.want {
				t.Errorf("getModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestHandleError tests error handling
func TestHandleError(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"Invalid API key"}}`,
			wantErr:    "unauthorized",
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":{"message":"Rate limit exceeded"}}`,
			wantErr:    "429",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":{"message":"Internal error"}}`,
			wantErr:    "server error",
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"Bad request"}}`,
			wantErr:    "HTTP 400",
		},
		{
			name:       "malformed error json",
			statusCode: http.StatusBadRequest,
			body:       `{invalid json}`,
			wantErr:    "HTTP 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			p.baseURL = server.URL

			_, err := p.Complete(context.Background(), llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			})

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestStreamResponse_ScannerError tests scanner error handling
func TestStreamResponse_ScannerError(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	// Create a reader that will cause scanner errors by returning very long lines
	largeChunk := strings.Repeat("x", 100000) // Very large line
	data := "data: " + largeChunk + "\n\n"

	chunks := make(chan llm.StreamChunk, 10)
	err := p.streamResponse(context.Background(), strings.NewReader(data), chunks)
	close(chunks)

	// Should handle scanner error gracefully (or succeed)
	// We're just testing that it doesn't panic
	_ = err
}

// TestConvertChunk tests chunk conversion edge cases
func TestConvertChunk(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	tests := []struct {
		name  string
		chunk *chatCompletionChunk
		want  *llm.StreamChunk
	}{
		{
			name: "empty choices",
			chunk: &chatCompletionChunk{
				Choices: []chatCompletionStreamChoice{},
			},
			want: nil,
		},
		{
			name: "finish reason only",
			chunk: &chatCompletionChunk{
				Choices: []chatCompletionStreamChoice{
					{
						Delta: chatMessageDelta{},
						FinishReason: func() *string {
							s := "stop"
							return &s
						}(),
					},
				},
			},
			want: &llm.StreamChunk{
				Type:         llm.ChunkTypeDone,
				FinishReason: "stop",
			},
		},
		{
			name: "empty delta",
			chunk: &chatCompletionChunk{
				Choices: []chatCompletionStreamChoice{
					{
						Delta: chatMessageDelta{},
					},
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.convertChunk(tt.chunk)

			if tt.want == nil {
				if got != nil {
					t.Errorf("convertChunk() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("convertChunk() returned nil, want non-nil")
			}

			if got.Type != tt.want.Type {
				t.Errorf("convertChunk().Type = %v, want %v", got.Type, tt.want.Type)
			}
		})
	}
}

// TestNewRequest tests HTTP request creation edge cases
func TestNewRequest(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Model:   "gpt-4",
	})

	tests := []struct {
		name    string
		method  string
		path    string
		body    interface{}
		wantErr bool
	}{
		{
			name:    "GET request without body",
			method:  http.MethodGet,
			path:    "/models",
			body:    nil,
			wantErr: false,
		},
		{
			name:    "POST request with body",
			method:  http.MethodPost,
			path:    "/chat/completions",
			body:    map[string]string{"test": "value"},
			wantErr: false,
		},
		{
			name:    "request with API key",
			method:  http.MethodPost,
			path:    "/test",
			body:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := p.newRequest(context.Background(), tt.method, tt.path, tt.body)

			if tt.wantErr {
				if err == nil {
					t.Error("newRequest() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("newRequest() unexpected error = %v", err)
				return
			}

			if req == nil {
				t.Fatal("newRequest() returned nil request")
			}

			// Verify Authorization header if API key is set
			if p.apiKey != "" {
				auth := req.Header.Get("Authorization")
				want := "Bearer " + p.apiKey
				if auth != want {
					t.Errorf("Authorization header = %q, want %q", auth, want)
				}
			}

			// Verify Content-Type
			if ct := req.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// TestProvider_Complete_ParseError tests response parsing errors
func TestProvider_Complete_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "gpt-4",
	})

	_, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err == nil {
		t.Error("Complete() with invalid JSON should return error")
	}
}

// TestProvider_Models_ParseError tests models parsing errors
func TestProvider_Models_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "gpt-4",
	})

	_, err := p.Models(context.Background())

	if err == nil {
		t.Error("Models() with invalid JSON should return error")
	}
}

// TestConvertResponse tests response conversion with empty choices
func TestConvertResponse(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	resp := &chatCompletionResponse{
		ID:      "test-123",
		Model:   "gpt-4",
		Choices: []chatCompletionChoice{},
	}

	converted := p.convertResponse(resp)

	if converted.ID != "test-123" {
		t.Errorf("convertResponse().ID = %q, want %q", converted.ID, "test-123")
	}

	if converted.Content != "" {
		t.Errorf("convertResponse().Content = %q, want empty", converted.Content)
	}
}

// TestBuildRequest tests request building
func TestBuildRequest(t *testing.T) {
	p, _ := NewProvider(Config{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	})

	tests := []struct {
		name   string
		req    llm.CompletionRequest
		stream bool
		check  func(t *testing.T, reqBody map[string]interface{})
	}{
		{
			name: "basic request",
			req: llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			stream: false,
			check: func(t *testing.T, reqBody map[string]interface{}) {
				if reqBody["model"] != "gpt-4" {
					t.Errorf("model = %v, want gpt-4", reqBody["model"])
				}
				if reqBody["stream"] != false {
					t.Errorf("stream = %v, want false", reqBody["stream"])
				}
				messages := reqBody["messages"].([]interface{})
				if len(messages) != 1 {
					t.Errorf("messages count = %d, want 1", len(messages))
				}
			},
		},
		{
			name: "request with model override",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
				Model:    "gpt-3.5-turbo",
			},
			check: func(t *testing.T, reqBody map[string]interface{}) {
				if reqBody["model"] != "gpt-3.5-turbo" {
					t.Errorf("model = %v, want gpt-3.5-turbo", reqBody["model"])
				}
			},
		},
		{
			name: "streaming request",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			stream: true,
			check: func(t *testing.T, reqBody map[string]interface{}) {
				if reqBody["stream"] != true {
					t.Error("stream should be true")
				}
			},
		},
		{
			name: "request with tools",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
				Tools: []llm.Tool{
					{
						Type: "function",
						Function: llm.Function{
							Name:        "test_func",
							Description: "A test function",
						},
					},
				},
			},
			check: func(t *testing.T, reqBody map[string]interface{}) {
				tools := reqBody["tools"]
				if tools == nil {
					t.Error("tools should be present")
				}
			},
		},
		{
			name: "request with parameters",
			req: llm.CompletionRequest{
				Messages:    []llm.Message{{Role: "user", Content: "test"}},
				MaxTokens:   100,
				Temperature: 0.7,
			},
			check: func(t *testing.T, reqBody map[string]interface{}) {
				if reqBody["max_tokens"] != 100 {
					t.Errorf("max_tokens = %v, want 100", reqBody["max_tokens"])
				}
				if reqBody["temperature"] != 0.7 {
					t.Errorf("temperature = %v, want 0.7", reqBody["temperature"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := p.buildRequest(tt.req, tt.stream)
			tt.check(t, reqBody)
		})
	}
}
