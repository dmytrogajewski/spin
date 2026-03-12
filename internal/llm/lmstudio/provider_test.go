package lmstudio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openai/openai-go"
)

// TestNewProvider tests provider construction.
func TestNewProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		checkFn func(t *testing.T, p *Provider)
	}{
		{
			name:    "valid config",
			cfg:     Config{Model: "llama2"},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				t.Helper()

				if p.Provider == nil {
					t.Error("Provider.Provider should not be nil")
				}
			},
		},
		{
			name: "custom base URL",
			cfg: Config{
				BaseURL: "http://custom:1234/v1",
				Model:   "mistral",
			},
			wantErr: false,
		},
		{
			name:    "empty config uses defaults",
			cfg:     Config{},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				t.Helper()

				if p.Provider == nil {
					t.Error("Provider.Provider should not be nil")
				}
			},
		},
		{
			name: "with timeout",
			cfg: Config{
				Model:   "llama2",
				Timeout: 60,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := NewProvider(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("NewProvider() expected error, got nil")
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

			if tt.checkFn != nil {
				tt.checkFn(t, p)
			}
		})
	}
}

// TestProvider_Name tests Name() method.
func TestProvider_Name(t *testing.T) {
	t.Parallel()

	p, _ := NewProvider(Config{})

	name := p.Name()
	want := "lmstudio"

	if name != want {
		t.Errorf("Name() = %q, want %q", name, want)
	}
}

// TestProvider_Capabilities tests Capabilities() method.
func TestProvider_Capabilities(t *testing.T) {
	t.Parallel()

	p, _ := NewProvider(Config{})

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

// TestProvider_Complete tests completion delegation.
func TestProvider_Complete(t *testing.T) {
	t.Parallel()

	// Create test server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Request method = %s, want POST", r.Method)
		}

		if r.URL.Path != "/chat/completions" {
			t.Errorf("Request path = %s, want /chat/completions", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "llama2",
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
		}`))
	}))
	defer server.Close()

	// Create provider.
	p, err := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}

	// Call Complete.
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello"),
		}),
	}

	resp, err := p.Complete(context.Background(), params)
	if err != nil {
		t.Errorf("Complete() unexpected error = %v", err)

		return
	}

	if resp == nil {
		t.Fatal("Complete() returned nil response")
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello! How can I help you?" {
		content := ""
		if len(resp.Choices) > 0 {
			content = resp.Choices[0].Message.Content
		}

		t.Errorf("Response content = %q, want %q", content, "Hello! How can I help you?")
	}
}

// TestProvider_Stream tests streaming delegation.
func TestProvider_Stream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Request path = %s, want /chat/completions", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		data := "data: " + `{"id":"1","choices":[{"delta":{"content":"Hello"}}]}` + "\n\n" +
			"data: " + `{"id":"1","choices":[{"delta":{"content":" there"}}]}` + "\n\n" +
			"data: [DONE]\n\n"

		_, _ = w.Write([]byte(data))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Hello"),
		}),
	}

	chunks, err := p.Stream(context.Background(), params)
	if err != nil {
		t.Errorf("Stream() unexpected error = %v", err)

		return
	}

	// Collect chunks.
	var collected []openai.ChatCompletionChunk
	for chunk := range chunks {
		collected = append(collected, chunk)
	}

	if len(collected) == 0 {
		t.Error("Stream() returned no chunks")
	}
}

// TestProvider_Models tests model listing delegation.
func TestProvider_Models(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("Request path = %s, want /models", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "llama2", "created": 1677652288},
				{"id": "mistral", "created": 1677652289}
			]
		}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
	})

	models, err := p.Models(context.Background())
	if err != nil {
		t.Errorf("Models() unexpected error = %v", err)

		return
	}

	if len(models) != 2 {
		t.Errorf("Models() count = %d, want 2", len(models))
	}
}

// TestProvider_Close tests cleanup delegation.
func TestProvider_Close(t *testing.T) {
	t.Parallel()

	p, _ := NewProvider(Config{})

	err := p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

// TestProvider_DefaultBaseURL tests default base URL.
func TestProvider_DefaultBaseURL(t *testing.T) {
	t.Parallel()

	// We can't easily test the internal baseURL, but we can verify
	// that the provider is created successfully without a baseURL.
	p, err := NewProvider(Config{})
	if err != nil {
		t.Errorf("NewProvider() with empty baseURL should succeed, got error: %v", err)
	}

	if p == nil {
		t.Error("NewProvider() with empty baseURL returned nil")
	}
}

// TestProvider_ErrorHandling tests error propagation.
func TestProvider_ErrorHandling(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Unauthorized"}}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
	})

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("test"),
		}),
	}

	_, err := p.Complete(context.Background(), params)
	if err == nil {
		t.Error("Complete() with 401 response should return error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
		t.Errorf("Error should contain 'unauthorized', got: %v", err)
	}
}

// TestProvider_ContextCancellation tests context cancellation propagation.
func TestProvider_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// This should not be reached due to cancellation.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"test"}}]}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("test"),
		}),
	}

	_, err := p.Complete(ctx, params)
	if err == nil {
		t.Error("Complete() with canceled context should return error")
	}
}
