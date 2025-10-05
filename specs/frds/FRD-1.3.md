# FRD-1.3: Standardize HTTP Client Usage

**Feature ID:** FRD-1.3
**Title:** Standardize HTTP Client Usage Across Providers
**Status:** ✅ Complete
**Priority:** HIGH
**Estimated Effort:** 2 hours
**Actual Effort:** 1 hour
**Related Roadmap:** [ROADMAP.md](../refactoring/ROADMAP.md#13-standardize-http-client-usage--high)

## Overview

This FRD addresses inconsistent HTTP client usage across LLM providers. Currently, the Ollama provider uses a raw `http.Client` without retry logic, while OpenAI uses `llm.HTTPClient` with built-in retry, backoff, and rate limiting. This inconsistency leads to reliability issues and poor user experience.

## Problem Statement

### Current State

**HTTP Client Usage by Provider:**

1. **OpenAI Provider** - ✅ Correct
   - Uses `llm.HTTPClient`
   - Has retry logic (max 3 retries)
   - Has exponential backoff
   - Has rate limit handling (429, 503, 504)

2. **Ollama Provider** - ❌ Problematic
   - Uses raw `http.Client`
   - **NO** retry logic
   - **NO** backoff
   - **NO** rate limit handling
   - Fails on transient network errors

3. **LMStudio Provider** - ✅ Correct
   - Delegates to OpenAI provider
   - Inherits `llm.HTTPClient` benefits

### Code Evidence

**Ollama (`internal/llm/ollama/provider.go:66`):**
```go
return &Provider{
    client: &http.Client{Timeout: timeout},  // Raw http.Client - NO RETRY!
    baseURL: baseURL,
    model: cfg.Model,
    timeout: timeout,
}, nil
```

**OpenAI (`internal/llm/openai/provider.go:60-63`):**
```go
client := llm.NewHTTPClient(
    llm.WithTimeout(timeout),
    llm.WithMaxRetries(3),  // Built-in retry
)
```

### Impact

**Reliability Issues:**
- ❌ Ollama fails on temporary network glitches
- ❌ Ollama fails on transient 503/504 errors
- ❌ No automatic recovery from rate limits
- ❌ Poor user experience with Ollama vs OpenAI

**Maintenance Issues:**
- Inconsistent behavior across providers
- Different error handling patterns
- Harder to test reliability scenarios

## Solution Design

### Approach

Replace Ollama's raw `http.Client` with `llm.HTTPClient` to provide:
1. Automatic retry on transient errors (429, 503, 504)
2. Exponential backoff between retries
3. Rate limit handling with Retry-After support
4. Consistent error handling across all providers

### API Changes

**Before:**
```go
type Provider struct {
    client  *http.Client  // Raw client
    baseURL string
    model   string
    timeout time.Duration
}

func NewProvider(cfg Config) (*Provider, error) {
    // ...
    return &Provider{
        client: &http.Client{Timeout: timeout},
        // ...
    }, nil
}
```

**After:**
```go
type Provider struct {
    client  *llm.HTTPClient  // Retry-capable client
    baseURL string
    model   string
    timeout time.Duration
}

func NewProvider(cfg Config) (*Provider, error) {
    // ...
    client := llm.NewHTTPClient(
        llm.WithTimeout(timeout),
        llm.WithMaxRetries(3),
        llm.WithRetryDelay(time.Second),
    )

    return &Provider{
        client: client,
        // ...
    }, nil
}
```

### File Changes

**`internal/llm/ollama/provider.go`:**
- Change `client *http.Client` → `client *llm.HTTPClient`
- Update `NewProvider` to use `llm.NewHTTPClient`
- No changes needed to HTTP request code (same `.Do()` API)

## Implementation Plan

### Step 1: Update Provider Struct (5 min)

```go
type Provider struct {
    client  *llm.HTTPClient  // Changed from *http.Client
    baseURL string
    model   string
    timeout time.Duration
}
```

### Step 2: Update NewProvider (10 min)

```go
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
```

### Step 3: Verify HTTP Request Code (5 min)

Check that all `.Do()` calls work with `llm.HTTPClient`:
- `Complete()` method
- `Stream()` method
- `Models()` method

**Good news:** `llm.HTTPClient` has the same `.Do(req)` signature as `http.Client`, so no changes needed!

### Step 4: Add Retry Tests (60 min)

Add tests to `provider_test.go`:

```go
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

    if err != nil {
        t.Fatalf("Stream() error = %v", err)
    }

    var content strings.Builder
    for chunk := range chunks {
        if chunk.Type == llm.ChunkTypeContentDelta {
            content.WriteString(chunk.Content)
        }
    }

    if content.String() != "Hello World" {
        t.Errorf("Content = %q, want %q", content.String(), "Hello World")
    }

    if attempts < 2 {
        t.Errorf("Expected at least 2 attempts (1 retry), got %d", attempts)
    }
}
```

### Step 5: Update Documentation (10 min)

Update godoc comments to mention retry behavior:

```go
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
```

## Definition of Ready (DoR)

- [x] HTTP client inconsistency identified
- [x] Impact analysis complete
- [x] Solution approach defined
- [x] Test strategy defined

## Definition of Done (DoD)

- [x] Ollama provider uses `llm.HTTPClient`
- [x] Raw `http.Client` removed from Ollama
- [x] All existing tests still pass
- [x] New retry tests added (5 test cases)
- [x] Documentation updated
- [x] No linter warnings
- [x] Code analyzed (no issues)
- [x] ROADMAP.md updated

## Success Metrics

| Metric | Before | After | Target Met |
|--------|--------|-------|------------|
| Ollama retry support | ❌ None | ✅ Yes | ✅ |
| Providers with retry | 1/2 (50%) | 2/2 (100%) | ✅ |
| Ollama reliability | Low | High | ✅ |
| Test coverage (retry scenarios) | 0 tests | 5+ tests | ✅ |

## Risks and Mitigation

### Risk 1: Breaking Existing Ollama Behavior
**Probability:** Low
**Impact:** Medium
**Mitigation:**
- `llm.HTTPClient` is API-compatible with `http.Client`
- All existing tests should pass without changes
- Retry logic only helps, doesn't change successful requests

### Risk 2: Timeout Changes
**Probability:** Low
**Impact:** Low
**Mitigation:**
- Keep same timeout values
- Retry delay is separate from request timeout
- Document timeout behavior

### Risk 3: Test Flakiness
**Probability:** Medium
**Impact:** Low
**Mitigation:**
- Use httptest.Server for controlled testing
- Mock retry scenarios deterministically
- Set reasonable timeout values in tests

## Dependencies

**Blocking:**
- None (llm.HTTPClient already exists)

**Blocked By:**
- None

**Related:**
- FRD-1.1 (SSE Scanner) - completed
- FRD-1.2 (Stream Processing) - completed

## References

- [Roadmap Task 1.3](../refactoring/ROADMAP.md#13-standardize-http-client-usage--high)
- [internal/llm/client.go](../../internal/llm/client.go) - HTTPClient implementation
- [internal/llm/ollama/provider.go](../../internal/llm/ollama/provider.go)

## Timeline

**Total Estimated Effort:** 2 hours

- Update struct & NewProvider: 15 min
- Verify HTTP calls: 5 min
- Write retry tests: 60 min
- Update docs: 10 min
- Run tests & verify: 30 min

**Start Date:** 2025-10-05
**Target Completion:** 2025-10-05

---

## Implementation Results

**Completed:** 2025-10-05

### Changes Made

1. **Updated Provider Struct** (`internal/llm/ollama/provider.go:38`)
   - Changed `client *http.Client` → `client *llm.HTTPClient`
   - Now supports retry logic

2. **Updated NewProvider Function** (`internal/llm/ollama/provider.go:44-88`)
   - Replaced raw `http.Client` instantiation
   - Created `llm.HTTPClient` with retry configuration:
     - MaxRetries: 3
     - RetryDelay: 1 second
     - Honors Retry-After header
   - Added comprehensive godoc explaining retry behavior

3. **Added 5 Retry Tests** (`internal/llm/ollama/provider_test.go`)
   - `TestProvider_Complete_RetryOn503` - retry on service unavailable
   - `TestProvider_Complete_RetryOn429` - retry with Retry-After header
   - `TestProvider_Complete_MaxRetriesExceeded` - fail after max retries
   - `TestProvider_Stream_RetryOn503` - streaming retry support
   - `TestProvider_Models_RetryOn504` - retry on gateway timeout

### Test Results

```bash
go test ./internal/llm/ollama -cover
ok      github.com/dmytrogajewski/spin/internal/llm/ollama    13.025s    coverage: 91.7% of statements (+0.1%)
```

**All 5 retry tests pass:**
- ✅ `TestProvider_Complete_RetryOn503` (3.00s)
- ✅ `TestProvider_Complete_RetryOn429` (1.00s)
- ✅ `TestProvider_Complete_MaxRetriesExceeded` (7.01s)
- ✅ `TestProvider_Stream_RetryOn503` (1.00s)
- ✅ `TestProvider_Models_RetryOn504` (1.00s)

### Benefits Achieved

- ✅ **Ollama now has automatic retry** on transient errors
- ✅ **Consistent behavior** across all providers (OpenAI, Ollama, LMStudio)
- ✅ **Better reliability** - handles 429, 503, 504 errors gracefully
- ✅ **Rate limit support** - respects Retry-After header
- ✅ **Exponential backoff** - prevents overwhelming servers
- ✅ **100% provider coverage** - all providers use HTTPClient

### API Compatibility

No breaking changes - `llm.HTTPClient.Do()` has same signature as `http.Client.Do()`, so all existing HTTP request code works unchanged.

---

**Status:** ✅ Complete
**Assigned To:** AI Agent
**Last Updated:** 2025-10-05
