# Package: internal/llm

**Path:** `internal/llm`  
**Purpose:** Vendor-agnostic LLM provider interfaces and implementations

---

## Overview

The `llm` package provides a unified abstraction layer for interacting with Large Language Model (LLM) providers. It supports OpenAI, Ollama, LMStudio, and any OpenAI-compatible API through a common interface, enabling zero vendor lock-in and seamless provider switching.

## Key Features

- **Vendor Agnostic**: Single interface for multiple LLM providers
- **Streaming Support**: Server-Sent Events (SSE) streaming
- **Retry Logic**: Built-in HTTP client with exponential backoff
- **Token Counting**: Accurate tokenization for different models
- **Error Handling**: Comprehensive error types and recovery
- **Testability**: MockProvider for easy testing
- **Thread Safe**: All providers safe for concurrent use

## Package Structure

```
internal/llm/
├── provider.go         # Provider interface
├── types.go            # Core types
├── client.go           # HTTP client with retry
├── stream.go           # SSE streaming
├── tokenizer.go        # Token counting
├── errors.go           # Error types
├── mock.go             # Mock provider
├── factory/            # Provider factory
├── openai/             # OpenAI provider
├── ollama/             # Ollama provider
└── lmstudio/           # LMStudio provider
```

## Provider Interface

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models(ctx context.Context) ([]Model, error)
    Capabilities() Capabilities
    Name() string
    Close() error
}
```

## Supported Providers

### 1. OpenAI Provider

Compatible with OpenAI, Azure OpenAI, and OpenAI-compatible APIs.

```go
import "github.com/dmytrogajewski/spin/internal/llm/openai"

provider, err := openai.NewProvider(openai.Config{
    BaseURL: "https://api.openai.com/v1",
    APIKey:  "sk-...",
    Model:   "gpt-4",
})
```

### 2. Ollama Provider

Local LLM execution via Ollama.

```go
import "github.com/dmytrogajewski/spin/internal/llm/ollama"

provider, err := ollama.NewProvider(ollama.Config{
    BaseURL: "http://localhost:11434",
    Model:   "llama2",
})
```

#### VRAM Auto-Tuning (Ollama)

Spin can auto-tune local model settings based on available VRAM. When enabled, it detects VRAM and selects best-fit context length (num_ctx) and GPU layers to avoid OOM while preserving quality.

Configuration (YAML):

```yaml
llm:
  provider: ollama
  model: llama3.1
  # auto_tune defaults to true; set false to disable
  auto_tune: true
  vram:
    # headroom_mib defaults to 1024 if omitted
    headroom_mib: 1024  # reserve 1GiB for system
```

Notes:
- Quantization choice is inferred from model tag (e.g., q4_0); auto-tune primarily sets num_ctx and GPU layers.
- Auto-tune is best-effort and will not block if VRAM detection is unavailable.

### 3. LMStudio Provider

Local models via LMStudio's OpenAI-compatible API.

```go
import "github.com/dmytrogajewski/spin/internal/llm/lmstudio"

provider, err := lmstudio.NewProvider(lmstudio.Config{
    BaseURL: "http://localhost:1234/v1",
    Model:   "local-model",
})
```

## HTTP Client

The package provides a robust HTTP client with retry logic:

```go
client := llm.NewHTTPClient(llm.HTTPClientConfig{
    Timeout:     30 * time.Second,
    MaxRetries:  3,
    RetryDelay:  1 * time.Second,
})
```

**Features:**
- Exponential backoff
- Transient error detection
- Context cancellation support
- Custom timeout per request

## SSE Streaming

Server-Sent Events streaming for real-time responses:

```go
chunks, err := provider.Stream(ctx, req)
for chunk := range chunks {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        continue
    }
    fmt.Print(chunk.Content)
}
```

**Implementation Details:**
- Buffered SSE scanning
- Delta content accumulation
- Tool call streaming support
- Automatic cleanup on context cancellation

## Tokenizer

Accurate token counting for different model families:

```go
tokenizer := llm.NewTokenizer("gpt-4")
count := tokenizer.Count("Hello, world!")
```

**Supported Models:**
- GPT-3.5/GPT-4 (cl100k_base)
- GPT-3/Codex (p50k_base)
- Llama models (approximation)

## Error Handling

```go
// Common error types
var (
    ErrInvalidRequest    = errors.New("invalid request")
    ErrProviderError     = errors.New("provider error")
    ErrRateLimited       = errors.New("rate limited")
    ErrContextCanceled   = errors.New("context canceled")
)

// Check error type
if errors.Is(err, llm.ErrRateLimited) {
    time.Sleep(backoff)
    retry()
}
```

## Factory Pattern

Centralized provider creation with secure credential management:

```go
import (
    "github.com/dmytrogajewski/spin/internal/llm/factory"
    "github.com/dmytrogajewski/spin/internal/auth"
)

// Create auth manager with keystore
authMgr := auth.NewManager(auth.NewKeystore())
defer authMgr.Close()

// Create factory with auth support
f := factory.NewFactory(authMgr)

// Create provider (credentials from keystore)
provider, err := f.NewProvider(ctx, factory.ProviderConfig{
    Type:    "openai",
    Model:   "gpt-4",
    BaseURL: "https://api.openai.com/v1",
    KeyName: "openai-api-key", // credential name in keystore
})

// Legacy: direct API key (deprecated, use KeyName instead)
provider, err := f.NewProvider(ctx, factory.ProviderConfig{
    Type:   "openai",
    Model:  "gpt-4",
    APIKey: "sk-...", // DEPRECATED: use KeyName with keystore
})
```

Supported provider types:
- `openai` - OpenAI and compatible APIs
- `ollama` - Ollama local models
- `lmstudio` - LMStudio local models
- `openai-compatible` - Generic OpenAI-compatible endpoints

## Testing

MockProvider for unit tests:

```go
mock := llm.NewMockProvider("test",
    llm.WithResponse("Hello!"),
    llm.WithToolCalls([]llm.ToolCall{...}),
)

resp, err := mock.Complete(ctx, req)
assert.Equal(t, "Hello!", resp.Content)
```

## Thread Safety

All provider implementations are thread-safe:

```go
// Safe concurrent usage
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        provider.Complete(ctx, req)
    }()
}
wg.Wait()
```

## Performance

**Benchmarks:**
- Token counting: ~100ns per message
- HTTP client creation: ~200ns
- Stream chunk processing: ~50ns

**Best Practices:**
- Reuse provider instances
- Use streaming for long responses
- Set appropriate timeouts
- Enable retry logic for production

## Configuration

```go
type Config struct {
    Provider    string        // "openai", "ollama", "lmstudio"
    BaseURL     string        // Provider base URL
    APIKey      string        // API key (if required)
    Model       string        // Model identifier
    Timeout     time.Duration // Request timeout
    MaxRetries  int           // Maximum retry attempts
    Temperature float64       // 0.0-2.0
    MaxTokens   int           // Max output tokens
}
```

## Related Documentation

- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Ollama API Docs](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [internal/auth](auth.md) - Authentication module
- [internal/core](core.md) - Core business logic

---

**Last Updated:** 2025-10-05  
**Test Coverage:** 94.8% (verified 2025-10-05)
- internal/llm: 94.8%
- factory: 94.4%
- lmstudio: 90.9%
- ollama: 91.7%
- openai: 89.5%  
**Status:** ✅ Production Ready
