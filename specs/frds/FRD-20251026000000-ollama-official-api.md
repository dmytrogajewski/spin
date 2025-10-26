# FRD: Migrate Ollama Provider to Official API Package

**Feature ID:** FRD-20251026000000  
**Status:** In Progress  
**Created:** 2025-10-26  
**Author:** System  
**Related Roadmap:** Phase 2.3 - Ollama Provider (Priority: P1)

---

## 1. Executive Summary

Migrate the Ollama provider from custom HTTP/JSON handling to the official `github.com/ollama/ollama/api` package. This eliminates maintenance burden, reduces code complexity, and ensures compatibility with official Ollama API changes.

### Key Benefits
- **Zero Maintenance**: Official API handles protocol changes
- **Type Safety**: SDK provides proper Go types vs manual JSON handling
- **Reduced LOC**: ~500 lines → ~200 lines (60% reduction)
- **Better Testing**: SDK is battle-tested by Ollama team
- **Feature Parity**: Immediate access to new Ollama features

### No Backward Compatibility Required
Per instructions, we're making a clean break - no compatibility shims needed.

---

## 2. Current State Analysis

### Current Implementation (`internal/llm/ollama/`)
```
provider.go       ~400 lines   - Custom HTTP client, retry logic
api.go           ~120 lines   - Manual type definitions
provider_test.go ~200 lines   - Tests with mock HTTP servers
Total: ~720 lines
```

### Current Architecture
```
Provider → HTTPClient → Manual JSON → Ollama REST API
         ↓
    Custom error mapping
    Custom retry logic
    Custom streaming (SSE)
    Custom type conversions
```

### Issues with Current Approach
1. **Duplication**: Reimplements what SDK already provides
2. **Maintenance**: Must track Ollama API changes manually
3. **Type Safety**: `map[string]interface{}` for options/parameters
4. **Error Prone**: Manual JSON marshaling/unmarshaling
5. **Testing**: Mock HTTP servers instead of SDK mocks

---

## 3. Target Architecture

### New Implementation Using Official API
```
Provider → api.Client → Ollama REST API
         ↓
    Thin adapter layer
    Type converters (spin types ↔ SDK types)
    Simplified error mapping
```

### Design Principles
1. **Thin Adapter**: Provider is just a thin wrapper around `api.Client`
2. **Type Conversion**: Clean converters between spin and SDK types
3. **Delegate Everything**: Let SDK handle HTTP, retry, streaming
4. **Remove Custom Code**: Delete all manual HTTP handling

---

## 4. Functional Requirements

### FR-1: Client Initialization
**Requirement**: Create Ollama provider using official SDK client

```go
import "github.com/ollama/ollama/api"

provider, err := ollama.NewProvider(ollama.Config{
    BaseURL: "http://localhost:11434",
    Model:   "llama3.1",
})
```

**Implementation**:
- Use `api.ClientFromEnvironment()` or `api.NewClient(url, httpClient)`
- Store `*api.Client` in Provider struct
- Remove custom HTTP client code

**Test Coverage**:
- Valid configuration
- Invalid/empty model name
- Custom base URL
- Default base URL

---

### FR-2: Chat Completion (Non-Streaming)
**Requirement**: Complete chat requests using SDK's `Chat()` method

**Spin Interface**:
```go
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error)
```

**SDK Mapping**:
```go
sdkReq := api.ChatRequest{
    Model:    p.model,
    Messages: convertMessages(req.Messages),
    Tools:    convertTools(req.Tools),
    Options:  convertOptions(req),
}

err := p.client.Chat(ctx, &sdkReq, func(resp api.ChatResponse) error {
    // Accumulate response
    return nil
})
```

**Type Conversions Required**:
- `[]llm.Message` → `[]api.Message`
- `[]llm.Tool` → `[]api.Tool`
- `llm.CompletionRequest` options → `map[string]interface{}`
- `api.ChatResponse` → `llm.CompletionResponse`

**Test Coverage**:
- Simple message completion
- Multi-turn conversation
- Tool calls in request
- Tool calls in response
- Temperature/options handling
- Context cancellation
- Error scenarios

---

### FR-3: Streaming Completion
**Requirement**: Stream chat responses using SDK's streaming callback

**Spin Interface**:
```go
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error)
```

**SDK Mapping**:
```go
chunks := make(chan llm.StreamChunk, 10)

go func() {
    defer close(chunks)
    
    err := p.client.Chat(ctx, &sdkReq, func(resp api.ChatResponse) error {
        chunk := convertStreamChunk(resp)
        select {
        case chunks <- chunk:
        case <-ctx.Done():
            return ctx.Err()
        }
        return nil
    })
    
    if err != nil {
        chunks <- llm.StreamChunk{Type: llm.ChunkTypeError, Error: err}
    }
}()

return chunks, nil
```

**Test Coverage**:
- Stream full response
- Stream with tool calls
- Stream cancellation
- Stream errors
- Channel cleanup

---

### FR-4: Model Listing
**Requirement**: List available models using SDK's `List()` method

**Spin Interface**:
```go
func (p *Provider) Models(ctx context.Context) ([]llm.Model, error)
```

**SDK Mapping**:
```go
resp, err := p.client.List(ctx)
if err != nil {
    return nil, mapError(err)
}

models := make([]llm.Model, len(resp.Models))
for i, m := range resp.Models {
    models[i] = convertModel(m)
}
return models, nil
```

**Test Coverage**:
- List models successfully
- Empty model list
- Error handling

---

### FR-5: Error Mapping
**Requirement**: Map SDK errors to spin's error types

**SDK Error Types**:
- `api.StatusError` - HTTP status errors (404, 500, etc.)
- Standard Go errors - network, timeout, etc.

**Spin Error Mapping**:
```go
func mapError(err error) error {
    if err == nil {
        return nil
    }
    
    // Context errors (pass-through)
    if errors.Is(err, context.Canceled) {
        return context.Canceled
    }
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("timeout: %w", err)
    }
    
    // SDK StatusError
    var statusErr api.StatusError
    if errors.As(err, &statusErr) {
        switch statusErr.StatusCode {
        case 404:
            return fmt.Errorf("model not found: %w", err)
        case 500, 502, 503:
            return fmt.Errorf("server error: %w", err)
        }
    }
    
    return err
}
```

**Test Coverage**:
- 404 model not found
- 500 server error
- Context cancellation
- Network errors

---

### FR-6: VRAM Auto-Tuning Support
**Requirement**: Preserve existing auto-tune functionality

**Current Behavior**:
- Detects available VRAM
- Calculates optimal `num_ctx` and `num_gpu`
- Sets via `options` in chat request

**Implementation with SDK**:
```go
options := map[string]interface{}{
    "temperature": req.Temperature,
}

if p.autoTuneCtxLen > 0 {
    options["num_ctx"] = p.autoTuneCtxLen
}
if p.autoTuneGPULayers > 0 {
    options["num_gpu"] = p.autoTuneGPULayers
}

sdkReq := api.ChatRequest{
    Model:   p.model,
    Options: options,
}
```

**No Changes Required**: Auto-tune logic stays the same, just pass options to SDK.

---

## 5. Non-Functional Requirements

### NFR-1: Performance
- **Latency**: No performance degradation vs current implementation
- **Memory**: Streaming must not buffer entire response
- **Concurrency**: Thread-safe for concurrent requests

### NFR-2: Code Quality
- **Coverage**: Minimum 90% test coverage
- **Linting**: Zero `make lint` errors
- **Deadcode**: Zero unreachable functions
- **Documentation**: All public APIs documented

### NFR-3: Maintainability
- **LOC Reduction**: Target 60% reduction
- **Complexity**: Cyclomatic complexity < 10
- **Dependencies**: Single new dependency (`github.com/ollama/ollama`)

---

## 6. Implementation Plan

### Step 1: Add Dependency
```bash
go get github.com/ollama/ollama/api@latest
```

### Step 2: Create Type Converters (`convert.go`)
- `convertMessages([]llm.Message) []api.Message`
- `convertTools([]llm.Tool) []api.Tool`
- `convertResponse(api.ChatResponse) *llm.CompletionResponse`
- `convertStreamChunk(api.ChatResponse) llm.StreamChunk`
- `convertModel(api.ListModelResponse) llm.Model`

### Step 3: Rewrite Provider (`provider.go`)
- Replace HTTPClient with `*api.Client`
- Implement `Complete()` using SDK's `Chat()`
- Implement `Stream()` using SDK's streaming callback
- Implement `Models()` using SDK's `List()`
- Preserve auto-tune logic

### Step 4: Create Error Mapper (`errors.go`)
- Map `api.StatusError` to spin errors
- Handle context errors
- Handle network errors

### Step 5: Update Tests (`provider_test.go`)
- Rewrite tests to use SDK types
- Keep integration test structure
- Add converter unit tests
- Add error mapping tests

### Step 6: Delete Old Code
- Delete `api.go` (replaced by SDK types)
- Remove custom HTTP client code
- Remove manual JSON parsing

### Step 7: Verification
- Run `make test` (ensure 90%+ coverage)
- Run `make lint` (zero errors)
- Test with real Ollama instance
- Benchmark performance vs old implementation

---

## 7. File Structure

### Before
```
internal/llm/ollama/
├── doc.go          (~50 lines)
├── provider.go     (~400 lines, custom HTTP)
├── api.go          (~120 lines, manual types)
└── provider_test.go (~200 lines)
Total: ~770 lines
```

### After
```
internal/llm/ollama/
├── doc.go          (~50 lines, updated)
├── provider.go     (~150 lines, SDK wrapper)
├── convert.go      (~200 lines, type converters)
├── errors.go       (~50 lines, error mapping)
└── provider_test.go (~250 lines, SDK-based tests)
Total: ~700 lines (but simpler, SDK-maintained logic)
```

**Net Result**: Similar LOC but much simpler - complex HTTP/streaming logic now in SDK.

---

## 8. Testing Strategy

### Unit Tests
- Type conversion functions (message, tool, response)
- Error mapping logic
- Auto-tune option passing

### Integration Tests
- Complete() with real request/response structure
- Stream() with channel behavior
- Models() list functionality
- Context cancellation
- Timeout handling

### Test Data
- Reuse existing test fixtures
- Add SDK-specific edge cases
- Tool call conversions (JSON arguments)

### Coverage Target
- Minimum: 90%
- Goal: 95%
- Exclude: Test helpers only

---

## 9. Migration Checklist

- [x] Read roadmap Phase 2.3
- [x] Read documentation
- [x] Write FRD
- [ ] Add SDK dependency
- [ ] Write converter functions (TDD)
- [ ] Rewrite Provider.Complete() (TDD)
- [ ] Rewrite Provider.Stream() (TDD)
- [ ] Rewrite Provider.Models() (TDD)
- [ ] Implement error mapping (TDD)
- [ ] Update tests for SDK
- [ ] Delete api.go and old HTTP code
- [ ] Run make lint (zero errors)
- [ ] Run make test (90%+ coverage)
- [ ] Verify with real Ollama
- [ ] Update documentation
- [ ] Mark Phase 2.3 complete in roadmap

---

## 10. Risk Analysis

### Risk: SDK API Changes
**Likelihood**: Low  
**Impact**: Medium  
**Mitigation**: Pin SDK version, monitor releases

### Risk: Feature Gaps
**Likelihood**: Low  
**Impact**: Low  
**Mitigation**: SDK is feature-complete for our needs

### Risk: Performance Regression
**Likelihood**: Very Low  
**Impact**: Medium  
**Mitigation**: Benchmark before/after

### Risk: Breaking Changes
**Likelihood**: None (no backward compatibility required)  
**Impact**: N/A  
**Mitigation**: N/A

---

## 11. Success Criteria

1. ✅ All tests pass with 90%+ coverage
2. ✅ Zero `make lint` errors
3. ✅ Zero deadcode warnings
4. ✅ Provider works with real Ollama instance
5. ✅ No performance regression
6. ✅ Code reduced by ~30% (simpler, not just fewer lines)
7. ✅ Documentation updated
8. ✅ Phase 2.3 marked complete in roadmap

---

## 12. References

- Ollama API Package: https://pkg.go.dev/github.com/ollama/ollama/api
- Roadmap Phase 2.3: `specs/ifacesroadmap.md:146-169`
- Current Ollama Provider: `internal/llm/ollama/`
- OpenAI Provider (reference): `internal/llm/openai/`
- Testing Guidelines: `docs/testing-patterns.md`

---

## Document Metadata
**Version:** 1.0  
**Last Updated:** 2025-10-26  
**Review Status:** Draft  
**Approved By:** N/A
