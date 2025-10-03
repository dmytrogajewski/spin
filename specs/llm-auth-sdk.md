# Spin LLM Providers, Authentication & SDK - Technical Documentation

## Overview

This document covers:
1. **LLM Providers** (`internal/llm`) - Vendor-agnostic LLM integration
2. **Authentication** (`internal/auth`) - Credential management
3. **Go SDK** (`pkg/sdk`) - Programmatic Spin integration
4. **Distribution** - Installation and deployment

**Philosophy:** Zero vendor lock-in. Works with any OpenAI-compatible API, Ollama, LMStudio, vLLM, and other open-source LLM backends.

---

## Module 1: LLM Provider System

**Package:** `internal/llm`  
**Purpose:** Vendor-agnostic LLM integration with multiple backend support

### Architecture

```
internal/llm/
├── provider.go              # Provider interface
├── client.go                # HTTP client implementation
├── types.go                 # Common types (Request/Response)
├── stream.go                # Streaming infrastructure
├── tokenizer.go             # Token counting
│
├── openai/                  # OpenAI-compatible API
│   ├── provider.go
│   ├── client.go
│   └── types.go
│
├── ollama/                  # Ollama-specific
│   ├── provider.go
│   ├── client.go
│   └── models.go
│
├── lmstudio/                # LMStudio-specific
│   ├── provider.go
│   └── client.go
│
├── anthropic/               # Anthropic Claude (if compatible)
│   ├── provider.go
│   └── adapter.go
│
└── localai/                 # LocalAI
    ├── provider.go
    └── client.go
```

### Provider Interface

**File:** `provider.go`

```go
package llm

import (
    "context"
    "io"
)

// Provider represents an LLM backend
type Provider interface {
    // Complete performs a completion request
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    
    // Stream performs a streaming completion request
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    
    // Models returns available models
    Models(ctx context.Context) ([]Model, error)
    
    // Capabilities returns provider capabilities
    Capabilities() Capabilities
    
    // Name returns provider name
    Name() string
    
    // Close closes the provider
    Close() error
}

// CompletionRequest represents a completion request
type CompletionRequest struct {
    Messages    []Message
    Model       string
    Tools       []Tool
    MaxTokens   int
    Temperature float64
    Stream      bool
    StopTokens  []string
}

// CompletionResponse represents a completion response
type CompletionResponse struct {
    ID         string
    Model      string
    Content    string
    ToolCalls  []ToolCall
    Usage      Usage
    FinishReason string
}

// StreamChunk represents a streaming chunk
type StreamChunk struct {
    Type         ChunkType
    Content      string
    ToolCall     *ToolCall
    FinishReason string
    Error        error
}

// ChunkType defines chunk types
type ChunkType int

const (
    ChunkTypeContentDelta ChunkType = iota
    ChunkTypeToolCallStart
    ChunkTypeToolCallDelta
    ChunkTypeToolCallComplete
    ChunkTypeDone
    ChunkTypeError
)

// Message represents a conversation message
type Message struct {
    Role       string      // system, user, assistant, tool
    Content    string
    ToolCalls  []ToolCall
    ToolCallID string      // For tool responses
}

// ToolCall represents an AI tool invocation
type ToolCall struct {
    ID        string
    Type      string // function
    Function  FunctionCall
}

// FunctionCall represents a function call
type FunctionCall struct {
    Name      string
    Arguments string // JSON-encoded arguments
}

// Tool represents a tool definition
type Tool struct {
    Type     string // function
    Function Function
}

// Function represents a function definition
type Function struct {
    Name        string
    Description string
    Parameters  interface{} // JSON Schema
}

// Model represents an available model
type Model struct {
    ID          string
    Name        string
    Description string
    ContextSize int
    Capabilities []string
}

// Capabilities represents provider capabilities
type Capabilities struct {
    Streaming      bool
    FunctionCalling bool
    Vision          bool
    Embeddings      bool
}

// Usage represents token usage
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### OpenAI-Compatible Provider

**File:** `openai/provider.go`

```go
package openai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/yourusername/spin/internal/llm"
)

// Provider implements OpenAI-compatible API
type Provider struct {
    client      *http.Client
    baseURL     string
    apiKey      string
    model       string
    timeout     time.Duration
}

// Config configures OpenAI provider
type Config struct {
    BaseURL     string        // https://api.openai.com/v1 or compatible
    APIKey      string        // API key or empty for Ollama
    Model       string        // Model name
    Timeout     time.Duration // Request timeout
}

// NewProvider creates an OpenAI-compatible provider
func NewProvider(cfg Config) (*Provider, error) {
    if cfg.BaseURL == "" {
        return nil, fmt.Errorf("base URL required")
    }
    
    if cfg.Timeout == 0 {
        cfg.Timeout = 5 * time.Minute
    }
    
    return &Provider{
        client: &http.Client{
            Timeout: cfg.Timeout,
        },
        baseURL: cfg.BaseURL,
        apiKey:  cfg.APIKey,
        model:   cfg.Model,
        timeout: cfg.Timeout,
    }, nil
}

// Complete performs a completion request
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
    // Build request
    reqBody := p.buildRequest(req, false)
    
    // Make HTTP request
    httpReq, err := p.newRequest(ctx, "POST", "/chat/completions", reqBody)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, p.handleError(resp)
    }
    
    // Parse response
    var apiResp chatCompletionResponse
    if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }
    
    return p.convertResponse(&apiResp), nil
}

// Stream performs a streaming completion request
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
    // Build request with streaming enabled
    reqBody := p.buildRequest(req, true)
    
    // Make HTTP request
    httpReq, err := p.newRequest(ctx, "POST", "/chat/completions", reqBody)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    
    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, p.handleError(resp)
    }
    
    // Create channel and start streaming
    chunks := make(chan llm.StreamChunk, 10)
    
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

// Models returns available models
func (p *Provider) Models(ctx context.Context) ([]llm.Model, error) {
    req, err := p.newRequest(ctx, "GET", "/models", nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Data []struct {
            ID      string `json:"id"`
            Created int64  `json:"created"`
        } `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    models := make([]llm.Model, len(result.Data))
    for i, m := range result.Data {
        models[i] = llm.Model{
            ID:   m.ID,
            Name: m.ID,
        }
    }
    
    return models, nil
}

// Capabilities returns provider capabilities
func (p *Provider) Capabilities() llm.Capabilities {
    return llm.Capabilities{
        Streaming:       true,
        FunctionCalling: true,
        Vision:          false,
        Embeddings:      false,
    }
}

// Name returns provider name
func (p *Provider) Name() string {
    return "openai-compatible"
}

// Close closes the provider
func (p *Provider) Close() error {
    return nil
}

// buildRequest builds API request body
func (p *Provider) buildRequest(req llm.CompletionRequest, stream bool) interface{} {
    messages := make([]map[string]interface{}, len(req.Messages))
    for i, msg := range req.Messages {
        messages[i] = map[string]interface{}{
            "role":    msg.Role,
            "content": msg.Content,
        }
        
        if len(msg.ToolCalls) > 0 {
            messages[i]["tool_calls"] = msg.ToolCalls
        }
        
        if msg.ToolCallID != "" {
            messages[i]["tool_call_id"] = msg.ToolCallID
        }
    }
    
    body := map[string]interface{}{
        "model":    p.getModel(req.Model),
        "messages": messages,
        "stream":   stream,
    }
    
    if req.MaxTokens > 0 {
        body["max_tokens"] = req.MaxTokens
    }
    
    if req.Temperature > 0 {
        body["temperature"] = req.Temperature
    }
    
    if len(req.Tools) > 0 {
        body["tools"] = req.Tools
    }
    
    if len(req.StopTokens) > 0 {
        body["stop"] = req.StopTokens
    }
    
    return body
}

// streamResponse processes streaming response
func (p *Provider) streamResponse(ctx context.Context, r io.Reader, chunks chan<- llm.StreamChunk) error {
    scanner := newSSEScanner(r)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        line := scanner.Text()
        
        if line == "data: [DONE]" {
            chunks <- llm.StreamChunk{Type: llm.ChunkTypeDone}
            return nil
        }
        
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        
        data := strings.TrimPrefix(line, "data: ")
        
        var delta chatCompletionChunk
        if err := json.Unmarshal([]byte(data), &delta); err != nil {
            continue
        }
        
        // Convert delta to chunk
        chunk := p.convertChunk(&delta)
        if chunk != nil {
            chunks <- *chunk
        }
    }
    
    return scanner.Err()
}

// newRequest creates an HTTP request
func (p *Provider) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
    var bodyReader io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        bodyReader = bytes.NewReader(data)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
    if err != nil {
        return nil, err
    }
    
    if p.apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+p.apiKey)
    }
    req.Header.Set("Content-Type", "application/json")
    
    return req, nil
}

// getModel returns model name
func (p *Provider) getModel(model string) string {
    if model != "" {
        return model
    }
    return p.model
}

// API types
type chatCompletionResponse struct {
    ID      string `json:"id"`
    Model   string `json:"model"`
    Choices []struct {
        Message struct {
            Role      string `json:"role"`
            Content   string `json:"content"`
            ToolCalls []struct {
                ID       string `json:"id"`
                Type     string `json:"type"`
                Function struct {
                    Name      string `json:"name"`
                    Arguments string `json:"arguments"`
                } `json:"function"`
            } `json:"tool_calls"`
        } `json:"message"`
        FinishReason string `json:"finish_reason"`
    } `json:"choices"`
    Usage struct {
        PromptTokens     int `json:"prompt_tokens"`
        CompletionTokens int `json:"completion_tokens"`
        TotalTokens      int `json:"total_tokens"`
    } `json:"usage"`
}

type chatCompletionChunk struct {
    ID      string `json:"id"`
    Choices []struct {
        Delta struct {
            Content   string `json:"content"`
            ToolCalls []struct {
                Index    int    `json:"index"`
                ID       string `json:"id"`
                Type     string `json:"type"`
                Function struct {
                    Name      string `json:"name"`
                    Arguments string `json:"arguments"`
                } `json:"function"`
            } `json:"tool_calls"`
        } `json:"delta"`
        FinishReason *string `json:"finish_reason"`
    } `json:"choices"`
}
```

### Ollama Provider

**File:** `ollama/provider.go`

```go
package ollama

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    
    "github.com/yourusername/spin/internal/llm"
)

// Provider implements Ollama-specific optimizations
type Provider struct {
    client  *http.Client
    baseURL string // http://localhost:11434
    model   string
}

// Config configures Ollama provider
type Config struct {
    BaseURL string // http://localhost:11434
    Model   string // codellama:13b
}

// NewProvider creates an Ollama provider
func NewProvider(cfg Config) (*Provider, error) {
    if cfg.BaseURL == "" {
        cfg.BaseURL = "http://localhost:11434"
    }
    
    if cfg.Model == "" {
        return nil, fmt.Errorf("model required")
    }
    
    return &Provider{
        client:  &http.Client{},
        baseURL: cfg.BaseURL,
        model:   cfg.Model,
    }, nil
}

// Complete performs a completion request
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
    // Convert messages to Ollama format
    prompt := p.buildPrompt(req.Messages)
    
    reqBody := map[string]interface{}{
        "model":  p.getModel(req.Model),
        "prompt": prompt,
        "stream": false,
    }
    
    if req.Temperature > 0 {
        reqBody["temperature"] = req.Temperature
    }
    
    // Make request
    data, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Response string `json:"response"`
        Done     bool   `json:"done"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &llm.CompletionResponse{
        Content:      result.Response,
        FinishReason: "stop",
    }, nil
}

// Stream performs a streaming completion request
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
    prompt := p.buildPrompt(req.Messages)
    
    reqBody := map[string]interface{}{
        "model":  p.getModel(req.Model),
        "prompt": prompt,
        "stream": true,
    }
    
    data, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    
    chunks := make(chan llm.StreamChunk, 10)
    
    go func() {
        defer close(chunks)
        defer resp.Body.Close()
        
        decoder := json.NewDecoder(resp.Body)
        
        for {
            var chunk struct {
                Response string `json:"response"`
                Done     bool   `json:"done"`
            }
            
            if err := decoder.Decode(&chunk); err != nil {
                if err != io.EOF {
                    chunks <- llm.StreamChunk{Type: llm.ChunkTypeError, Error: err}
                }
                return
            }
            
            if chunk.Response != "" {
                chunks <- llm.StreamChunk{
                    Type:    llm.ChunkTypeContentDelta,
                    Content: chunk.Response,
                }
            }
            
            if chunk.Done {
                chunks <- llm.StreamChunk{Type: llm.ChunkTypeDone}
                return
            }
        }
    }()
    
    return chunks, nil
}

// Models returns available models
func (p *Provider) Models(ctx context.Context) ([]llm.Model, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Models []struct {
            Name       string `json:"name"`
            ModifiedAt string `json:"modified_at"`
            Size       int64  `json:"size"`
        } `json:"models"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    models := make([]llm.Model, len(result.Models))
    for i, m := range result.Models {
        models[i] = llm.Model{
            ID:   m.Name,
            Name: m.Name,
        }
    }
    
    return models, nil
}

// buildPrompt converts messages to Ollama prompt format
func (p *Provider) buildPrompt(messages []llm.Message) string {
    var parts []string
    
    for _, msg := range messages {
        switch msg.Role {
        case "system":
            parts = append(parts, "System: "+msg.Content)
        case "user":
            parts = append(parts, "User: "+msg.Content)
        case "assistant":
            parts = append(parts, "Assistant: "+msg.Content)
        }
    }
    
    parts = append(parts, "Assistant:")
    return strings.Join(parts, "\n\n")
}

// Capabilities returns provider capabilities
func (p *Provider) Capabilities() llm.Capabilities {
    return llm.Capabilities{
        Streaming:       true,
        FunctionCalling: false, // Ollama doesn't support function calling natively
        Vision:          false,
        Embeddings:      true,
    }
}

// Name returns provider name
func (p *Provider) Name() string {
    return "ollama"
}

// getModel returns model name
func (p *Provider) getModel(model string) string {
    if model != "" {
        return model
    }
    return p.model
}

// Close closes the provider
func (p *Provider) Close() error {
    return nil
}
```

### Provider Factory

**File:** `factory.go`

```go
package llm

import (
    "fmt"
    
    "github.com/yourusername/spin/internal/llm/ollama"
    "github.com/yourusername/spin/internal/llm/openai"
    "github.com/yourusername/spin/internal/llm/lmstudio"
)

// ProviderConfig configures a provider
type ProviderConfig struct {
    Type        string            // ollama, openai, lmstudio, etc.
    BaseURL     string            // API endpoint
    APIKey      string            // API key (if required)
    Model       string            // Model name
    Options     map[string]interface{} // Provider-specific options
}

// NewProvider creates a provider from config
func NewProvider(cfg ProviderConfig) (Provider, error) {
    switch cfg.Type {
    case "ollama":
        return ollama.NewProvider(ollama.Config{
            BaseURL: cfg.BaseURL,
            Model:   cfg.Model,
        })
    
    case "openai":
        return openai.NewProvider(openai.Config{
            BaseURL: cfg.BaseURL,
            APIKey:  cfg.APIKey,
            Model:   cfg.Model,
        })
    
    case "lmstudio":
        return lmstudio.NewProvider(lmstudio.Config{
            BaseURL: cfg.BaseURL,
            Model:   cfg.Model,
        })
    
    case "openai-compatible":
        return openai.NewProvider(openai.Config{
            BaseURL: cfg.BaseURL,
            APIKey:  cfg.APIKey,
            Model:   cfg.Model,
        })
    
    default:
        return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
    }
}
```

### HTTP Client with Retry

**File:** `client.go`

```go
package llm

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

// HTTPClient wraps http.Client with retry logic
type HTTPClient struct {
    client      *http.Client
    maxRetries  int
    retryDelay  time.Duration
}

// NewHTTPClient creates an HTTP client with retry
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
    c := &HTTPClient{
        client: &http.Client{
            Timeout: 5 * time.Minute,
        },
        maxRetries: 3,
        retryDelay: time.Second,
    }
    
    for _, opt := range opts {
        opt(c)
    }
    
    return c
}

// Do executes HTTP request with retry
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
    var lastErr error
    
    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff
            delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
            time.Sleep(delay)
        }
        
        resp, err := c.client.Do(req)
        if err != nil {
            lastErr = err
            continue
        }
        
        // Check if retryable
        if !c.isRetryable(resp.StatusCode) {
            return resp, nil
        }
        
        // Handle rate limiting
        if resp.StatusCode == http.StatusTooManyRequests {
            if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
                // Parse and wait
                if d, err := time.ParseDuration(retryAfter + "s"); err == nil {
                    time.Sleep(d)
                }
            }
        }
        
        resp.Body.Close()
        lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// isRetryable returns true for retryable status codes
func (c *HTTPClient) isRetryable(code int) bool {
    return code == http.StatusTooManyRequests ||
           code == http.StatusServiceUnavailable ||
           code == http.StatusGatewayTimeout
}

// ClientOption configures HTTPClient
type ClientOption func(*HTTPClient)

// WithTimeout sets request timeout
func WithTimeout(d time.Duration) ClientOption {
    return func(c *HTTPClient) {
        c.client.Timeout = d
    }
}

// WithMaxRetries sets max retries
func WithMaxRetries(n int) ClientOption {
    return func(c *HTTPClient) {
        c.maxRetries = n
    }
}
```

---

## Module 2: Authentication

**Package:** `internal/auth`  
**Purpose:** Credential management for LLM providers

### Architecture

```
internal/auth/
├── auth.go                  # Auth interface
├── keystore.go              # Secure credential storage
├── keystore_linux.go        # Secret Service
├── keystore_darwin.go       # Keychain
├── keystore_windows.go      # Credential Manager
├── apikey.go                # API key auth
└── oauth.go                 # OAuth flow (optional)
```

### Auth Interface

**File:** `auth.go`

```go
package auth

import (
    "context"
    "errors"
)

var (
    ErrNotAuthenticated = errors.New("not authenticated")
    ErrInvalidCredential = errors.New("invalid credential")
)

// Auth manages authentication credentials
type Auth interface {
    // GetCredential retrieves credential for provider
    GetCredential(ctx context.Context, provider string) (Credential, error)
    
    // SetCredential stores credential for provider
    SetCredential(ctx context.Context, provider string, cred Credential) error
    
    // DeleteCredential removes credential for provider
    DeleteCredential(ctx context.Context, provider string) error
    
    // ListProviders returns providers with stored credentials
    ListProviders(ctx context.Context) ([]string, error)
}

// Credential represents authentication credential
type Credential struct {
    Type  CredentialType
    Value string // API key, token, etc.
}

// CredentialType defines credential types
type CredentialType int

const (
    CredentialTypeAPIKey CredentialType = iota
    CredentialTypeToken
    CredentialTypeNone // For local providers like Ollama
)

// Manager implements Auth
type Manager struct {
    keystore Keystore
}

// NewManager creates an auth manager
func NewManager() (*Manager, error) {
    ks, err := NewKeystore()
    if err != nil {
        return nil, err
    }
    
    return &Manager{
        keystore: ks,
    }, nil
}

// GetCredential retrieves credential for provider
func (m *Manager) GetCredential(ctx context.Context, provider string) (Credential, error) {
    value, err := m.keystore.Get(provider)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return Credential{}, ErrNotAuthenticated
        }
        return Credential{}, err
    }
    
    return Credential{
        Type:  CredentialTypeAPIKey,
        Value: value,
    }, nil
}

// SetCredential stores credential for provider
func (m *Manager) SetCredential(ctx context.Context, provider string, cred Credential) error {
    return m.keystore.Set(provider, cred.Value)
}

// DeleteCredential removes credential for provider
func (m *Manager) DeleteCredential(ctx context.Context, provider string) error {
    return m.keystore.Delete(provider)
}

// ListProviders returns providers with stored credentials
func (m *Manager) ListProviders(ctx context.Context) ([]string, error) {
    return m.keystore.List()
}
```

### Keystore Interface

**File:** `keystore.go`

```go
package auth

import (
    "errors"
)

var (
    ErrNotFound = errors.New("credential not found")
    ErrNoKeystore = errors.New("no keystore available")
)

// Keystore provides secure credential storage
type Keystore interface {
    // Get retrieves credential
    Get(key string) (string, error)
    
    // Set stores credential
    Set(key, value string) error
    
    // Delete removes credential
    Delete(key string) error
    
    // List returns all keys
    List() ([]string, error)
}

// NewKeystore creates platform-specific keystore
func NewKeystore() (Keystore, error) {
    return newPlatformKeystore()
}
```

### macOS Keystore

**File:** `keystore_darwin.go`

```go
//go:build darwin

package auth

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

import (
    "fmt"
    "unsafe"
)

const keychainService = "spin"

// darwinKeystore implements Keystore using macOS Keychain
type darwinKeystore struct{}

func newPlatformKeystore() (Keystore, error) {
    return &darwinKeystore{}, nil
}

// Get retrieves credential from Keychain
func (k *darwinKeystore) Get(key string) (string, error) {
    serviceRef := C.CString(keychainService)
    defer C.free(unsafe.Pointer(serviceRef))
    
    accountRef := C.CString(key)
    defer C.free(unsafe.Pointer(accountRef))
    
    var passwordData C.CFDataRef
    var itemRef C.CFTypeRef
    
    status := C.SecKeychainFindGenericPassword(
        nil, // default keychain
        C.UInt32(len(keychainService)),
        serviceRef,
        C.UInt32(len(key)),
        accountRef,
        nil,
        &passwordData,
        &itemRef,
    )
    
    if status == C.errSecItemNotFound {
        return "", ErrNotFound
    }
    
    if status != C.errSecSuccess {
        return "", fmt.Errorf("keychain error: %d", status)
    }
    
    defer C.CFRelease(C.CFTypeRef(passwordData))
    defer C.CFRelease(itemRef)
    
    length := C.CFDataGetLength(passwordData)
    bytes := C.CFDataGetBytePtr(passwordData)
    password := C.GoStringN((*C.char)(unsafe.Pointer(bytes)), C.int(length))
    
    return password, nil
}

// Set stores credential in Keychain
func (k *darwinKeystore) Set(key, value string) error {
    // First try to delete existing
    k.Delete(key)
    
    serviceRef := C.CString(keychainService)
    defer C.free(unsafe.Pointer(serviceRef))
    
    accountRef := C.CString(key)
    defer C.free(unsafe.Pointer(accountRef))
    
    passwordRef := C.CString(value)
    defer C.free(unsafe.Pointer(passwordRef))
    
    status := C.SecKeychainAddGenericPassword(
        nil, // default keychain
        C.UInt32(len(keychainService)),
        serviceRef,
        C.UInt32(len(key)),
        accountRef,
        C.UInt32(len(value)),
        unsafe.Pointer(passwordRef),
        nil,
    )
    
    if status != C.errSecSuccess {
        return fmt.Errorf("keychain error: %d", status)
    }
    
    return nil
}

// Delete removes credential from Keychain
func (k *darwinKeystore) Delete(key string) error {
    serviceRef := C.CString(keychainService)
    defer C.free(unsafe.Pointer(serviceRef))
    
    accountRef := C.CString(key)
    defer C.free(unsafe.Pointer(accountRef))
    
    var itemRef C.CFTypeRef
    
    status := C.SecKeychainFindGenericPassword(
        nil,
        C.UInt32(len(keychainService)),
        serviceRef,
        C.UInt32(len(key)),
        accountRef,
        nil,
        nil,
        &itemRef,
    )
    
    if status == C.errSecItemNotFound {
        return nil // Already deleted
    }
    
    if status == C.errSecSuccess {
        C.SecKeychainItemDelete(C.SecKeychainItemRef(itemRef))
        C.CFRelease(itemRef)
    }
    
    return nil
}

// List returns all keys
func (k *darwinKeystore) List() ([]string, error) {
    // Not implemented for simplicity
    return nil, fmt.Errorf("list not implemented")
}
```

### Linux Keystore

**File:** `keystore_linux.go`

```go
//go:build linux

package auth

import (
    "github.com/zalando/go-keyring"
)

const keychainService = "spin"

// linuxKeystore implements Keystore using Secret Service
type linuxKeystore struct{}

func newPlatformKeystore() (Keystore, error) {
    return &linuxKeystore{}, nil
}

// Get retrieves credential from Secret Service
func (k *linuxKeystore) Get(key string) (string, error) {
    value, err := keyring.Get(keychainService, key)
    if err != nil {
        if err == keyring.ErrNotFound {
            return "", ErrNotFound
        }
        return "", err
    }
    return value, nil
}

// Set stores credential in Secret Service
func (k *linuxKeystore) Set(key, value string) error {
    return keyring.Set(keychainService, key, value)
}

// Delete removes credential from Secret Service
func (k *linuxKeystore) Delete(key string) error {
    return keyring.Delete(keychainService, key)
}

// List returns all keys
func (k *linuxKeystore) List() ([]string, error) {
    // Not easily supported by Secret Service API
    return nil, fmt.Errorf("list not implemented")
}
```

### API Key Auth

**File:** `apikey.go`

```go
package auth

import (
    "context"
    "fmt"
    "os"
    "strings"
)

// GetAPIKeyFromEnv retrieves API key from environment
func GetAPIKeyFromEnv(provider string) (string, error) {
    // Try provider-specific env var
    envVar := strings.ToUpper(provider) + "_API_KEY"
    if key := os.Getenv(envVar); key != "" {
        return key, nil
    }
    
    // Try common names
    commonNames := map[string]string{
        "openai":    "OPENAI_API_KEY",
        "anthropic": "ANTHROPIC_API_KEY",
        "together":  "TOGETHER_API_KEY",
    }
    
    if envVar, ok := commonNames[provider]; ok {
        if key := os.Getenv(envVar); key != "" {
            return key, nil
        }
    }
    
    return "", fmt.Errorf("API key not found in environment")
}

// SetAPIKey stores API key for provider
func SetAPIKey(ctx context.Context, provider, apiKey string) error {
    mgr, err := NewManager()
    if err != nil {
        return err
    }
    
    return mgr.SetCredential(ctx, provider, Credential{
        Type:  CredentialTypeAPIKey,
        Value: apiKey,
    })
}

// GetAPIKey retrieves API key for provider
func GetAPIKey(ctx context.Context, provider string) (string, error) {
    // First try environment
    if key, err := GetAPIKeyFromEnv(provider); err == nil {
        return key, nil
    }
    
    // Then try keystore
    mgr, err := NewManager()
    if err != nil {
        return "", err
    }
    
    cred, err := mgr.GetCredential(ctx, provider)
    if err != nil {
        return "", err
    }
    
    return cred.Value, nil
}
```

### CLI Commands

```go
// cmd/spin/auth.go

// AuthCmd implements auth commands
func AuthCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "auth",
        Short: "Manage authentication",
    }
    
    cmd.AddCommand(authLoginCmd())
    cmd.AddCommand(authLogoutCmd())
    cmd.AddCommand(authStatusCmd())
    cmd.AddCommand(authListCmd())
    
    return cmd
}

func authLoginCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "login <provider>",
        Short: "Store API key for provider",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            provider := args[0]
            
            // Prompt for API key
            fmt.Printf("Enter API key for %s: ", provider)
            var apiKey string
            fmt.Scanln(&apiKey)
            
            // Store in keystore
            if err := auth.SetAPIKey(cmd.Context(), provider, apiKey); err != nil {
                return fmt.Errorf("store credential: %w", err)
            }
            
            fmt.Printf("✓ Stored API key for %s\n", provider)
            return nil
        },
    }
}

func authLogoutCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "logout <provider>",
        Short: "Remove stored credential",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            provider := args[0]
            
            mgr, err := auth.NewManager()
            if err != nil {
                return err
            }
            
            if err := mgr.DeleteCredential(cmd.Context(), provider); err != nil {
                return err
            }
            
            fmt.Printf("✓ Removed credential for %s\n", provider)
            return nil
        },
    }
}

func authStatusCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show authentication status",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr, err := auth.NewManager()
            if err != nil {
                return err
            }
            
            providers, err := mgr.ListProviders(cmd.Context())
            if err != nil {
                return err
            }
            
            if len(providers) == 0 {
                fmt.Println("No stored credentials")
                return nil
            }
            
            fmt.Println("Stored credentials:")
            for _, p := range providers {
                fmt.Printf("  • %s\n", p)
            }
            
            return nil
        },
    }
}
```

---

## Module 3: Go SDK

**Package:** `pkg/sdk`  
**Purpose:** Programmatic Spin integration for Go applications

### Installation

```bash
go get github.com/yourusername/spin/pkg/sdk
```

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/yourusername/spin/pkg/sdk"
)

func main() {
    // Create client
    client := sdk.NewClient(sdk.Config{
        Provider: "ollama",
        Model:    "codellama:13b",
        WorkDir:  "/path/to/project",
    })
    defer client.Close()
    
    // Start thread
    thread := client.NewThread(context.Background())
    
    // Run task (blocking)
    result, err := thread.Run(context.Background(), "Refactor the auth module")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Result:", result.Output)
    fmt.Println("Files modified:", result.FilesModified)
}
```

### SDK Implementation

**File:** `pkg/sdk/client.go`

```go
package sdk

import (
    "context"
    "fmt"
    
    "github.com/yourusername/spin/internal/core"
    "github.com/yourusername/spin/internal/llm"
)

// Client is the main SDK client
type Client struct {
    manager *core.Manager
    config  Config
}

// Config configures the SDK client
type Config struct {
    Provider string // ollama, openai, lmstudio, etc.
    BaseURL  string // Provider endpoint
    APIKey   string // API key (if required)
    Model    string // Model name
    WorkDir  string // Working directory
}

// NewClient creates a new SDK client
func NewClient(cfg Config) *Client {
    // Create LLM provider
    providerCfg := llm.ProviderConfig{
        Type:    cfg.Provider,
        BaseURL: cfg.BaseURL,
        APIKey:  cfg.APIKey,
        Model:   cfg.Model,
    }
    
    provider, err := llm.NewProvider(providerCfg)
    if err != nil {
        panic(err) // Or return error
    }
    
    // Create core manager
    coreCfg := &core.Config{
        WorkDir: cfg.WorkDir,
    }
    
    manager, err := core.NewManager(coreCfg,
        core.WithLLMProvider(provider),
    )
    if err != nil {
        panic(err)
    }
    
    return &Client{
        manager: manager,
        config:  cfg,
    }
}

// NewThread starts a new conversation thread
func (c *Client) NewThread(ctx context.Context) *Thread {
    conv, err := c.manager.NewConversation(ctx, c.config.WorkDir)
    if err != nil {
        panic(err)
    }
    
    return &Thread{
        conversation: conv,
        client:       c,
    }
}

// ResumeThread resumes an existing thread
func (c *Client) ResumeThread(ctx context.Context, sessionID string) (*Thread, error) {
    conv, err := c.manager.ResumeConversation(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    return &Thread{
        conversation: conv,
        client:       c,
    }, nil
}

// ListThreads returns conversation history
func (c *Client) ListThreads(ctx context.Context) ([]ThreadInfo, error) {
    sessions, err := c.manager.ListConversations(ctx, core.Filter{})
    if err != nil {
        return nil, err
    }
    
    infos := make([]ThreadInfo, len(sessions))
    for i, s := range sessions {
        infos[i] = ThreadInfo{
            ID:        s.ID,
            CreatedAt: s.CreatedAt,
            Summary:   s.Summary,
        }
    }
    
    return infos, nil
}

// Close closes the client
func (c *Client) Close() error {
    return nil
}

// Thread represents a conversation thread
type Thread struct {
    conversation *core.Conversation
    client       *Client
}

// Run executes a task (blocking)
func (t *Thread) Run(ctx context.Context, prompt string) (*Result, error) {
    if err := t.conversation.RunTurn(ctx, prompt); err != nil {
        return nil, err
    }
    
    // Wait for completion
    state := t.conversation.State()
    
    return &Result{
        Output:        state.LastResponse,
        FilesModified: state.FilesModified,
        CommandsRun:   state.CommandsExecuted,
    }, nil
}

// RunStreaming executes a task with streaming
func (t *Thread) RunStreaming(ctx context.Context, prompt string) (<-chan Event, error) {
    events := make(chan Event, 10)
    
    // Start turn in goroutine
    go func() {
        defer close(events)
        
        // Subscribe to conversation events
        convEvents := t.conversation.Stream()
        
        // Start turn
        if err := t.conversation.RunTurn(ctx, prompt); err != nil {
            events <- Event{
                Type:  EventTypeError,
                Error: err,
            }
            return
        }
        
        // Forward events
        for e := range convEvents {
            events <- convertEvent(e)
        }
    }()
    
    return events, nil
}

// ID returns thread ID
func (t *Thread) ID() string {
    return t.conversation.SessionID()
}

// Result represents task execution result
type Result struct {
    Output        string
    FilesModified []string
    CommandsRun   []string
}

// ThreadInfo contains thread metadata
type ThreadInfo struct {
    ID        string
    CreatedAt time.Time
    Summary   string
}

// Event represents a streaming event
type Event struct {
    Type    EventType
    Content string
    Tool    *ToolCall
    Error   error
}

// EventType defines event types
type EventType int

const (
    EventTypeContentDelta EventType = iota
    EventTypeToolCall
    EventTypeComplete
    EventTypeError
)

// ToolCall represents a tool invocation
type ToolCall struct {
    Name   string
    Args   map[string]interface{}
    Result string
}
```

### Streaming Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/yourusername/spin/pkg/sdk"
)

func main() {
    client := sdk.NewClient(sdk.Config{
        Provider: "ollama",
        Model:    "codellama:13b",
        WorkDir:  "./project",
    })
    defer client.Close()
    
    thread := client.NewThread(context.Background())
    
    // Stream events
    events, err := thread.RunStreaming(context.Background(), "Fix all linter errors")
    if err != nil {
        log.Fatal(err)
    }
    
    for event := range events {
        switch event.Type {
        case sdk.EventTypeContentDelta:
            fmt.Print(event.Content)
        
        case sdk.EventTypeToolCall:
            fmt.Printf("\n🔧 Tool: %s\n", event.Tool.Name)
        
        case sdk.EventTypeComplete:
            fmt.Println("\n✓ Done!")
        
        case sdk.EventTypeError:
            fmt.Printf("\n✗ Error: %v\n", event.Error)
        }
    }
}
```

### Custom Approver

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
    
    "github.com/yourusername/spin/pkg/sdk"
)

func main() {
    client := sdk.NewClient(sdk.Config{
        Provider: "ollama",
        Model:    "codellama:13b",
        WorkDir:  "./project",
        Approver: &CustomApprover{},
    })
    defer client.Close()
    
    thread := client.NewThread(context.Background())
    result, _ := thread.Run(context.Background(), "Deploy to production")
    fmt.Println(result.Output)
}

// CustomApprover implements approval logic
type CustomApprover struct{}

func (a *CustomApprover) ApproveCommand(ctx context.Context, cmd string) (bool, error) {
    fmt.Printf("Allow command: %s? (y/n): ", cmd)
    
    reader := bufio.NewReader(os.Stdin)
    answer, _ := reader.ReadString('\n')
    
    return strings.TrimSpace(answer) == "y", nil
}

func (a *CustomApprover) ApproveFileWrite(ctx context.Context, path string) (bool, error) {
    fmt.Printf("Allow writing to %s? (y/n): ", path)
    
    reader := bufio.NewReader(os.Stdin)
    answer, _ := reader.ReadString('\n')
    
    return strings.TrimSpace(answer) == "y", nil
}
```

---

## Module 4: Distribution

### Installation Methods

#### 1. Go Install (Primary)

```bash
go install github.com/yourusername/spin/cmd/spin@latest
```

**Advantages:**
- Native Go installation
- Automatic binary building
- Version management via Go modules
- Works on all platforms

#### 2. GitHub Releases

```bash
# Download binary for your platform
curl -L https://github.com/yourusername/spin/releases/latest/download/spin-linux-amd64 -o spin
chmod +x spin
sudo mv spin /usr/local/bin/
```

#### 3. Homebrew (macOS/Linux)

```ruby
# Formula: homebrew-tap/spin.rb
class Spin < Formula
  desc "Vendor-agnostic AI coding agent"
  homepage "https://github.com/yourusername/spin"
  url "https://github.com/yourusername/spin/archive/v1.0.0.tar.gz"
  sha256 "..."
  
  depends_on "go" => :build
  
  def install
    system "go", "build", "-o", bin/"spin", "./cmd/spin"
  end
  
  test do
    system "#{bin}/spin", "--version"
  end
end
```

```bash
brew tap yourusername/tap
brew install spin
```

#### 4. Package Managers

**Arch (AUR):**
```bash
yay -S spin-bin
```

**Debian/Ubuntu:**
```bash
curl -L https://github.com/yourusername/spin/releases/latest/download/spin_amd64.deb -o spin.deb
sudo dpkg -i spin.deb
```

#### 5. Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o spin ./cmd/spin

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
COPY --from=builder /build/spin /usr/local/bin/
ENTRYPOINT ["spin"]
```

```bash
docker run -v $(pwd):/workspace spin "Refactor auth module"
```

### Build Scripts

**Makefile:**
```makefile
.PHONY: build release install clean

VERSION := $(shell git describe --tags --always)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o bin/spin ./cmd/spin

release:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/spin-linux-amd64 ./cmd/spin
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/spin-linux-arm64 ./cmd/spin
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/spin-darwin-amd64 ./cmd/spin
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/spin-darwin-arm64 ./cmd/spin
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/spin-windows-amd64.exe ./cmd/spin

install:
	go install -ldflags="$(LDFLAGS)" ./cmd/spin

clean:
	rm -rf bin/ dist/

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
```

### GitHub Actions Release

**.github/workflows/release.yml:**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Build binaries
        run: make release
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Configuration Example

**~/.spin/config.yaml:**
```yaml
# LLM Provider Configuration
providers:
  # Default provider
  default: ollama-local
  
  # Ollama (local)
  ollama-local:
    type: ollama
    base_url: http://localhost:11434
    model: codellama:13b
  
  # LMStudio (local)
  lmstudio:
    type: lmstudio
    base_url: http://localhost:1234/v1
    model: codellama-13b-instruct
  
  # vLLM (self-hosted)
  vllm-gpu:
    type: openai-compatible
    base_url: http://gpu-server:8000/v1
    model: deepseek-coder-33b-instruct
  
  # OpenAI (cloud fallback)
  openai:
    type: openai
    base_url: https://api.openai.com/v1
    api_key_env: OPENAI_API_KEY
    model: gpt-4

# Security
security:
  sandbox:
    mode: workspace-write
  policy_file: ~/.spin/policy.yaml

# Agent
agent:
  max_iterations: 50
  timeout: 300s
```

---

## Integration Example: Web Service

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    
    "github.com/yourusername/spin/pkg/sdk"
)

type Server struct {
    spin *sdk.Client
}

func main() {
    // Create Spin client
    spin := sdk.NewClient(sdk.Config{
        Provider: "ollama",
        Model:    "codellama:13b",
        WorkDir:  "/tmp/workspace",
    })
    defer spin.Close()
    
    server := &Server{spin: spin}
    
    http.HandleFunc("/analyze", server.handleAnalyze)
    http.HandleFunc("/refactor", server.handleRefactor)
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Code string `json:"code"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    thread := s.spin.NewThread(r.Context())
    result, err := thread.Run(r.Context(), "Analyze this code: "+req.Code)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]string{
        "analysis": result.Output,
    })
}

func (s *Server) handleRefactor(w http.ResponseWriter, r *http.Request) {
    var req struct {
        FilePath string `json:"file_path"`
        Task     string `json:"task"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    thread := s.spin.NewThread(r.Context())
    
    // Stream results
    events, err := thread.RunStreaming(r.Context(), req.Task)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Server-sent events
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, _ := w.(http.Flusher)
    
    for event := range events {
        data, _ := json.Marshal(event)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
}
```

---

## Testing

### SDK Tests

```go
package sdk_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/yourusername/spin/pkg/sdk"
)

func TestClient_NewThread(t *testing.T) {
    client := sdk.NewClient(sdk.Config{
        Provider: "mock",
        Model:    "test",
        WorkDir:  t.TempDir(),
    })
    defer client.Close()
    
    thread := client.NewThread(context.Background())
    assert.NotNil(t, thread)
}

func TestThread_Run(t *testing.T) {
    client := sdk.NewClient(sdk.Config{
        Provider: "mock",
        Model:    "test",
        WorkDir:  t.TempDir(),
    })
    defer client.Close()
    
    thread := client.NewThread(context.Background())
    
    result, err := thread.Run(context.Background(), "test task")
    require.NoError(t, err)
    assert.NotEmpty(t, result.Output)
}

func TestThread_RunStreaming(t *testing.T) {
    client := sdk.NewClient(sdk.Config{
        Provider: "mock",
        Model:    "test",
        WorkDir:  t.TempDir(),
    })
    defer client.Close()
    
    thread := client.NewThread(context.Background())
    
    events, err := thread.RunStreaming(context.Background(), "test task")
    require.NoError(t, err)
    
    var count int
    for event := range events {
        count++
        assert.NotEqual(t, sdk.EventTypeError, event.Type)
    }
    
    assert.Greater(t, count, 0)
}
```

---

## Conclusion

The Spin LLM, authentication, and SDK modules provide:
- **Vendor-agnostic LLM integration** - Works with Ollama, LMStudio, OpenAI, and any compatible API
- **Secure credential management** - Platform-native keystores
- **Clean Go SDK** - Programmatic access for Go applications
- **Multiple distribution channels** - `go install`, GitHub releases, Homebrew, Docker
- **Zero lock-in** - Easy switching between providers
- **Local-first** - Optimized for local LLMs

This architecture enables flexible, secure, and performant integration of Spin into diverse environments and workflows while maintaining complete vendor independence.


