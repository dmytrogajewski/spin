package ollama

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
)

// newMockOllamaServer creates a mock Ollama server that responds to /api/show and /api/chat.
func newMockOllamaServer(t *testing.T, _ string, chatResponses []api.ChatResponse) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			handleShowRequest(t, w)
		case "/api/chat":
			handleChatRequest(t, w, chatResponses)
		default:
			http.NotFound(w, r)
		}
	}))
}

func handleShowRequest(t *testing.T, w http.ResponseWriter) {
	t.Helper()

	resp := api.ShowResponse{
		ModelInfo: map[string]any{"general.context_length": float64(4096)},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Errorf("failed to encode show response: %v", err)
	}
}

func handleChatRequest(t *testing.T, w http.ResponseWriter, responses []api.ChatResponse) {
	t.Helper()

	w.Header().Set("Content-Type", "application/x-ndjson")
	flusher, ok := w.(http.Flusher)

	for _, resp := range responses {
		data, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("failed to marshal chat response: %v", err)

			return
		}

		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n"))

		if ok {
			flusher.Flush()
		}
	}
}

// newTestProvider creates a Provider connected to a test server.
func newTestProvider(t *testing.T, srv *httptest.Server, model string) *Provider {
	t.Helper()

	baseURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	return &Provider{
		client:  api.NewClient(baseURL, srv.Client()),
		model:   model,
		baseURL: srv.URL,
		timeout: 10 * time.Second,
		logger:  slog.Default(),
	}
}

// collectStreamedContent reads all chunks and returns the concatenated content.
func collectStreamedContent(t *testing.T, chunks <-chan openai.ChatCompletionChunk) string {
	t.Helper()

	var content strings.Builder

	for chunk := range chunks {
		if len(chunk.Choices) == 0 {
			continue
		}

		if chunk.Choices[0].Delta.Content != "" {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	return content.String()
}

func TestOllamaStreaming(t *testing.T) {
	t.Parallel()

	const testModel = "qwen2.5-coder:1.5b"

	streamResponses := []api.ChatResponse{
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "1"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ", "}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "2"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ", "}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "3"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ""}, Done: true, DoneReason: "stop"},
	}

	srv := newMockOllamaServer(t, testModel, streamResponses)
	defer srv.Close()

	provider := newTestProvider(t, srv, testModel)
	defer provider.Close()

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Count to 3"),
		},
	}

	chunks, err := provider.Stream(context.Background(), params)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	content := collectStreamedContent(t, chunks)
	if content == "" {
		t.Fatal("No content received from stream")
	}

	const expected = "1, 2, 3"
	if content != expected {
		t.Fatalf("unexpected content: got %q, want %q", content, expected)
	}
}
