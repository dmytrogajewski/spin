package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/vram"
)

// TestNewProvider tests provider construction
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
		checkFn func(t *testing.T, p *Provider)
	}{
		{
			name: "valid config",
			cfg: Config{
				BaseURL: "http://localhost:11434",
				Model:   "llama2",
			},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				if p.model != "llama2" {
					t.Errorf("model = %q, want %q", p.model, "llama2")
				}
			},
		},
		{
			name: "empty base URL uses default",
			cfg: Config{
				Model: "llama2",
			},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				if p.baseURL != "http://localhost:11434" {
					t.Errorf("baseURL = %q, want default", p.baseURL)
				}
			},
		},
		{
			name: "missing model",
			cfg: Config{
				BaseURL: "http://localhost:11434",
			},
			wantErr: true,
			errMsg:  "model",
		},
		{
			name: "URL with trailing slash",
			cfg: Config{
				BaseURL: "http://localhost:11434/",
				Model:   "llama2",
			},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				if strings.HasSuffix(p.baseURL, "/") {
					t.Errorf("baseURL should not have trailing slash: %q", p.baseURL)
				}
			},
		},
		{
			name: "default timeout applied",
			cfg: Config{
				Model: "llama2",
			},
			wantErr: false,
			checkFn: func(t *testing.T, p *Provider) {
				if p.timeout == 0 {
					t.Error("timeout should be set")
				}
			},
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

			if tt.checkFn != nil {
				tt.checkFn(t, p)
			}
		})
	}
}

// TestProvider_Name tests Name() method
func TestProvider_Name(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	name := p.Name()
	want := "ollama"

	if name != want {
		t.Errorf("Name() = %q, want %q", name, want)
	}
}

// TestProvider_Capabilities tests Capabilities() method
func TestProvider_Capabilities(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	caps := p.Capabilities()

	if !caps.Streaming {
		t.Error("Capabilities().Streaming = false, want true")
	}
	// Ollama now supports function calling (tool use)
	if !caps.FunctionCalling {
		t.Error("Capabilities().FunctionCalling = false, want true")
	}
	if caps.Vision {
		t.Error("Capabilities().Vision = true, want false")
	}
}

// TestBuildPrompt tests message to prompt conversion
func TestBuildPrompt(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	tests := []struct {
		name     string
		messages []llm.Message
		want     string
	}{
		{
			name: "single user message",
			messages: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			want: "User: Hello\n\nAssistant:",
		},
		{
			name: "with system message",
			messages: []llm.Message{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
			},
			want: "System: You are helpful\n\nUser: Hello\n\nAssistant:",
		},
		{
			name: "multi-turn conversation",
			messages: []llm.Message{
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "Hello!"},
				{Role: "user", Content: "How are you?"},
			},
			want: "User: Hi\n\nAssistant: Hello!\n\nUser: How are you?\n\nAssistant:",
		},
		{
			name: "with tool response",
			messages: []llm.Message{
				{Role: "user", Content: "What's the weather?"},
				{Role: "assistant", Content: "Let me check"},
				{Role: "tool", Content: "Temperature: 20°C", ToolCallID: "call_1"},
			},
			want: "User: What's the weather?\n\nAssistant: Let me check\n\nTool: Temperature: 20°C\n\nAssistant:",
		},
		{
			name:     "empty messages",
			messages: []llm.Message{},
			want:     "Assistant:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.buildPrompt(tt.messages)
			if got != tt.want {
				t.Errorf("buildPrompt() = %q, want %q", got, tt.want)
			}
		})
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
			serverResp: `{"model":"llama2","response":"Hi there!","done":true,"prompt_eval_count":10,"eval_count":5}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			checkResp: func(t *testing.T, resp *llm.CompletionResponse) {
				if resp.Content != "Hi there!" {
					t.Errorf("Response content = %q, want %q", resp.Content, "Hi there!")
				}
				if resp.Usage.PromptTokens != 10 {
					t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
				}
				if resp.Usage.CompletionTokens != 5 {
					t.Errorf("CompletionTokens = %d, want 5", resp.Usage.CompletionTokens)
				}
			},
		},
		{
			name: "404 model not found",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverResp: `{"error":"model not found"}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name: "500 server error",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverResp: `{"error":"internal error"}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Request method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/api/generate" {
					t.Errorf("Request path = %s, want /api/generate", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResp))
			}))
			defer server.Close()

			// Create provider
			p, err := NewProvider(Config{
				BaseURL: server.URL,
				Model:   "llama2",
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response":"test","done":true}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
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
			serverData: `{"model":"llama2","response":"Hello","done":false}` + "\n" +
				`{"model":"llama2","response":" there","done":false}` + "\n" +
				`{"model":"llama2","response":"","done":true,"prompt_eval_count":10,"eval_count":5}` + "\n",
			wantChunks: 3,
			checkChunk: func(t *testing.T, chunks []llm.StreamChunk) {
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
			name: "empty response with done",
			req: llm.CompletionRequest{
				Messages: []llm.Message{{Role: "user", Content: "test"}},
			},
			serverData: `{"model":"llama2","response":"","done":true}` + "\n",
			wantChunks: 1,
			checkChunk: func(t *testing.T, chunks []llm.StreamChunk) {
				if chunks[0].Type != llm.ChunkTypeDone {
					t.Error("Chunk should be ChunkTypeDone")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/generate" {
					t.Errorf("Request path = %s, want /api/generate", r.URL.Path)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.serverData))
			}))
			defer server.Close()

			p, _ := NewProvider(Config{
				BaseURL: server.URL,
				Model:   "llama2",
			})

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

// TestProvider_Stream_ErrorResponse tests streaming with error response
func TestProvider_Stream_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	_, err := p.Stream(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err == nil {
		t.Error("Stream() with error response should return error")
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
				"models": [
					{"name": "llama2", "size": 3800000000, "modified_at": "2024-01-01T00:00:00Z"},
					{"name": "mistral", "size": 4100000000, "modified_at": "2024-01-02T00:00:00Z"}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  2,
		},
		{
			name:       "empty model list",
			serverResp: `{"models": []}`,
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "500 error",
			serverResp: `{"error": "internal error"}`,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("Request path = %s, want /api/tags", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResp))
			}))
			defer server.Close()

			p, _ := NewProvider(Config{
				BaseURL: server.URL,
				Model:   "llama2",
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
	p, _ := NewProvider(Config{Model: "llama2"})

	err := p.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}
}

// TestProvider_ParseError tests response parsing errors
func TestProvider_Complete_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
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
		Model:   "llama2",
	})

	_, err := p.Models(context.Background())

	if err == nil {
		t.Error("Models() with invalid JSON should return error")
	}
}

// TestGetModel tests getModel function
func TestGetModel(t *testing.T) {
	p, _ := NewProvider(Config{
		Model: "llama2",
	})

	tests := []struct {
		name  string
		model string
		want  string
	}{
		{
			name:  "empty model uses default",
			model: "",
			want:  "llama2",
		},
		{
			name:  "specified model used",
			model: "mistral",
			want:  "mistral",
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

// TestStreamResponse_ContextCancellation tests context cancellation during streaming
func TestStreamResponse_ContextCancellation(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	// Create a reader that will stream slowly
	data := `{"response":"test","done":false}` + "\n" +
		`{"response":"more","done":false}` + "\n"

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	chunks := make(chan llm.StreamChunk, 10)
	err := p.streamResponse(ctx, strings.NewReader(data), chunks)

	if err == nil {
		t.Error("streamResponse() with cancelled context should return error")
	}
}

// TestStreamResponse_MalformedJSON tests handling of malformed JSON
func TestStreamResponse_MalformedJSON(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	// Mix of valid and invalid JSON
	data := `{"response":"valid","done":false}` + "\n" +
		`{invalid json}` + "\n" +
		`{"response":"also valid","done":true}` + "\n"

	chunks := make(chan llm.StreamChunk, 10)
	go func() {
		p.streamResponse(context.Background(), strings.NewReader(data), chunks)
		close(chunks)
	}()

	var collected []llm.StreamChunk
	for chunk := range chunks {
		collected = append(collected, chunk)
	}

	// Should get 2 valid chunks (malformed one skipped)
	if len(collected) < 2 {
		t.Errorf("Got %d chunks, want 2 (malformed JSON should be skipped)", len(collected))
	}
}

// TestHandleError tests error handling with different status codes
func TestHandleError(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "model not found",
			statusCode: http.StatusNotFound,
			body:       `{"error":"model not found"}`,
			wantErr:    "model not found",
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"error":"bad request"}`,
			wantErr:    "HTTP 400",
		},
		{
			name:       "malformed error json",
			statusCode: http.StatusBadRequest,
			body:       `{invalid}`,
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

// --- VRAM AutoTune tests ---

type fakeDetector struct {
	total int64
	free  int64
	name  string
}

func (f *fakeDetector) TotalVRAM() (int64, error)     { return f.total, nil }
func (f *fakeDetector) AvailableVRAM() (int64, error) { return f.free, nil }
func (f *fakeDetector) GPUName() (string, error)      { return f.name, nil }

func TestAutoTune_LowVRAMFallback_Warning(t *testing.T) {
	// Override seams
	oldNewDetector := vramNewDetector
	oldNewCalc := newRequirementsCalculator
	t.Cleanup(func() { vramNewDetector = oldNewDetector; newRequirementsCalculator = oldNewCalc })

	vramNewDetector = func(_ vram.CommandRunner) vram.Detector {
		return &fakeDetector{total: 2 << 30, free: 512 << 20, name: "nvidia"}
	}
	newRequirementsCalculator = func(d vram.Detector, headroom int64) *vram.RequirementsCalculator {
		return vram.NewRequirementsCalculator(d, headroom)
	}

	// Fake /api/tags to provide model size
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"models":[{"name":"llama2","size":%d}]}`, 7<<30) // 7 GiB model
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p, err := NewProvider(Config{BaseURL: server.URL, Model: "llama2"})
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}

	// Headroom 1GiB to force aggressive fallback
	if err := p.AutoTune(context.Background(), 1<<30); err != nil {
		t.Fatalf("AutoTune error: %v", err)
	}

	warn := p.GetAutoTuneWarning()
	if warn == "" {
		t.Errorf("expected warning for low VRAM fallback, got empty")
	}
}

func TestAutoTune_CPUFallback_Warning(t *testing.T) {
	oldNewDetector := vramNewDetector
	oldNewCalc := newRequirementsCalculator
	t.Cleanup(func() { vramNewDetector = oldNewDetector; newRequirementsCalculator = oldNewCalc })

	vramNewDetector = func(_ vram.CommandRunner) vram.Detector { return &fakeDetector{total: 0, free: 0, name: "cpu"} }
	newRequirementsCalculator = func(d vram.Detector, headroom int64) *vram.RequirementsCalculator {
		return vram.NewRequirementsCalculator(d, headroom)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"models":[{"name":"llama2","size":%d}]}`, 2<<30)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p, err := NewProvider(Config{BaseURL: server.URL, Model: "llama2"})
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}

	if err := p.AutoTune(context.Background(), 0); err != nil {
		t.Fatalf("AutoTune error: %v", err)
	}

	warn := p.GetAutoTuneWarning()
	if !strings.Contains(strings.ToLower(warn), "cpu") {
		t.Errorf("expected CPU fallback warning, got: %q", warn)
	}
}

// TestNewRequest tests HTTP request creation
func TestNewRequest(t *testing.T) {
	p, _ := NewProvider(Config{Model: "llama2"})

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
			path:    "/api/tags",
			body:    nil,
			wantErr: false,
		},
		{
			name:    "POST request with body",
			method:  http.MethodPost,
			path:    "/api/generate",
			body:    map[string]string{"test": "value"},
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

			// Verify Content-Type
			if ct := req.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// TestProvider_ModelOverride tests request with model override
func TestProvider_ModelOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"model":"mistral","response":"test","done":true}`))
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	// Request with model override
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
		Model:    "mistral",
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Model != "mistral" {
		t.Errorf("Response model = %q, want %q", resp.Model, "mistral")
	}
}

// TestProvider_Complete_RetryOn503 tests retry logic on 503 errors
func TestProvider_Complete_RetryOn503(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on 3rd attempt
		json.NewEncoder(w).Encode(generateResponse{
			Response: "Success after retries",
			Done:     true,
		})
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err != nil {
		t.Errorf("Complete() should succeed after retries, got error: %v", err)
	}

	if resp.Content != "Success after retries" {
		t.Errorf("Content = %q, want %q", resp.Content, "Success after retries")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts (2 retries), got %d", attempts)
	}
}

// TestProvider_Complete_RetryOn429 tests retry logic with Retry-After header
func TestProvider_Complete_RetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		json.NewEncoder(w).Encode(generateResponse{
			Response: "Success",
			Done:     true,
		})
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	start := time.Now()
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Success" {
		t.Errorf("Content = %q, want %q", resp.Content, "Success")
	}

	// Should have waited ~1 second for Retry-After
	if elapsed < time.Second {
		t.Errorf("Should have waited for Retry-After, elapsed = %v", elapsed)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts (1 retry), got %d", attempts)
	}
}

// TestProvider_Complete_MaxRetriesExceeded tests failure after max retries
func TestProvider_Complete_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	_, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	if err == nil {
		t.Error("Complete() should fail after max retries")
	}

	// Should mention 503 in error
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("Error should mention 503, got: %v", err)
	}
}

// TestProvider_Stream_RetryOn503 tests retry logic for streaming on 503 errors
func TestProvider_Stream_RetryOn503(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on 2nd attempt
		fmt.Fprintf(w, `{"response":"Hello","done":false}`)
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, `{"response":" World","done":true}`)
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	chunks, err := p.Stream(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "test"}},
	})

	// Stream request should fail immediately (doesn't retry)
	if err == nil {
		t.Fatal("Expected error from Stream on 503, got nil")
	}
	if !strings.Contains(err.Error(), "503") && !strings.Contains(err.Error(), "Service Unavailable") {
		t.Errorf("Expected 503 error, got: %v", err)
	}

	// Chunks channel should be nil when error occurs
	if chunks != nil {
		t.Error("Expected nil chunks channel on error")
	}
}

// TestProvider_Models_RetryOn504 tests retry logic for Models on 504 errors
func TestProvider_Models_RetryOn504(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		// Succeed on 2nd attempt
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []modelInfo{
				{Name: "llama2", ModifiedAt: time.Now()},
			},
		})
	}))
	defer server.Close()

	p, _ := NewProvider(Config{
		BaseURL: server.URL,
		Model:   "llama2",
	})

	models, err := p.Models(context.Background())

	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}

	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	if attempts < 2 {
		t.Errorf("Expected at least 2 attempts (1 retry), got %d", attempts)
	}
}
