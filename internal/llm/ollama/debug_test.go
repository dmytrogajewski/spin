package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
)

func TestOllamaStreaming(t *testing.T) {
	const testModel = "qwen2.5-coder:1.5b"

	// Define the streaming responses the mock server will return.
	streamResponses := []api.ChatResponse{
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "1"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ", "}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "2"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ", "}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: "3"}, Done: false},
		{Model: testModel, Message: api.Message{Role: "assistant", Content: ""}, Done: true, DoneReason: "stop"},
	}

	// Create an httptest server that mimics Ollama's API.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			// Return model info with a context_length field.
			resp := api.ShowResponse{
				ModelInfo: map[string]any{
					"general.context_length": float64(4096),
				},
			}

			w.Header().Set("Content-Type", "application/json")

			err := json.NewEncoder(w).Encode(resp)
			if err != nil {
				t.Errorf("failed to encode show response: %v", err)
			}

		case "/api/chat":
			// Return newline-delimited JSON responses.
			w.Header().Set("Content-Type", "application/x-ndjson")
			flusher, ok := w.(http.Flusher)

			for _, resp := range streamResponses {
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

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Create a Provider pointing at the test server.
	baseURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	provider := &Provider{
		client:  api.NewClient(baseURL, srv.Client()),
		model:   testModel,
		baseURL: srv.URL,
		timeout: 10 * time.Second,
	}
	defer provider.Close()

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Count to 3"),
		}),
	}

	ctx := context.Background()

	chunks, err := provider.Stream(ctx, params)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	content := ""

	chunkCount := 0
	var contentSb103 strings.Builder
	for chunk := range chunks {
		chunkCount++

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				contentSb103.WriteString(delta.Content)
			}

			if chunk.Choices[0].FinishReason != "" {
				t.Logf("Finish reason: %s", chunk.Choices[0].FinishReason)
			}
		}
	}
	content += contentSb103.String()

	t.Logf("Total chunks: %d", chunkCount)
	t.Logf("Content: %q", content)

	if content == "" {
		t.Fatal("No content received from stream")
	}

	const expected = "1, 2, 3"
	if content != expected {
		t.Fatalf("unexpected content: got %q, want %q", content, expected)
	}
}
