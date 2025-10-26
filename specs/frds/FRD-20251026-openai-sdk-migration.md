# FRD-20251026: OpenAI Provider SDK Migration

**Date**: 2025-10-26  
**Status**: Draft  
**Priority**: P1  
**Author**: Spin Agent  
**Related**: Phase 2.2 - OpenAI Provider

## Executive Summary

Migrate the OpenAI provider implementation from a custom HTTP client to the official [openai-go SDK](https://github.com/openai/openai-go) to reduce maintenance burden, improve reliability, and leverage vendor-maintained features. This migration will replace ~495 lines of custom HTTP/SSE handling code with SDK calls while maintaining 100% backward compatibility with the existing `llm.Provider` interface.

**No backward compatibility required** - This is a clean cutover migration.

## Problem Statement

### Current State

The OpenAI provider (`internal/llm/openai/`) currently implements:
- Custom HTTP client with retry logic
- Manual SSE (Server-Sent Events) parsing for streaming
- Custom type marshaling/unmarshaling
- ~495 lines in `provider.go` + 94 lines in `api.go` + 68 lines in `doc.go`
- 91.4% test coverage (1294 lines of tests)

**Pain Points**:
1. **Maintenance burden**: Must track OpenAI API changes manually
2. **Missing features**: No built-in support for newer OpenAI features (structured outputs, moderation, etc.)
3. **Error handling**: Custom error parsing prone to breakage
4. **Testing**: Manual mocking of HTTP responses

### Desired State

- Use official `openai-go` SDK v0.1.0-alpha.37 (latest stable)
- Reduce custom code from ~657 lines to ~200-300 lines
- Delegate HTTP/SSE/retry logic to SDK
- Maintain `llm.Provider` interface contract
- Keep test coverage ≥90%
- Zero goroutine leaks
- Race detector clean

## Requirements

### Functional Requirements

1. **FR1**: Provider must implement `llm.Provider` interface without breaking existing consumers
2. **FR2**: Support synchronous completion via `Complete(ctx, req) (*CompletionResponse, error)`
3. **FR3**: Support streaming via `Stream(ctx, req) (<-chan StreamChunk, error)`
4. **FR4**: Support function/tool calling with exact argument preservation
5. **FR5**: Support model listing via `Models(ctx) ([]Model, error)`
6. **FR6**: Preserve existing configuration structure (`Config` struct)
7. **FR7**: Map SDK errors to `llm.Err*` error codes for consistent error handling
8. **FR8**: Support context cancellation for all operations
9. **FR9**: Clean up resources (goroutines, connections) on context cancel or completion

### Non-Functional Requirements

1. **NFR1**: Test coverage ≥90% (currently 91.4%)
2. **NFR2**: Zero goroutine leaks (verified with leak detector)
3. **NFR3**: Race detector clean (`go test -race`)
4. **NFR4**: No performance regression (within 5% of current implementation)
5. **NFR5**: Complexity ≤15 per function (measured by gocyclo)
6. **NFR6**: Zero dead code (verified with `make deadcode`)
7. **NFR7**: Clean lint (`make lint` must pass)
8. **NFR8**: Update all documentation

### Out of Scope

- ❌ Azure OpenAI Service specific features
- ❌ Assistants API, Fine-tuning API, Embeddings API
- ❌ Image generation, Audio transcription
- ❌ Batch API
- ❌ Legacy completions endpoint (only chat completions)
- ❌ Changing `llm.Provider` interface

## Technical Design

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent / Orchestration                     │
└───────────────────────────────┬─────────────────────────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │   llm.Provider        │ (interface, unchanged)
                    │   - Complete()        │
                    │   - Stream()          │
                    │   - Models()          │
                    └───────────┬───────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │  openai.Provider      │ (refactored)
                    │  - Wraps openai.Client│
                    │  - Type conversion    │
                    │  - Error mapping      │
                    └───────────┬───────────┘
                                │
                                ▼
                    ┌───────────────────────┐
                    │ github.com/openai/    │ (new dependency)
                    │ openai-go SDK         │
                    │ - HTTP client         │
                    │ - SSE parsing         │
                    │ - Type definitions    │
                    └───────────────────────┘
```

### SDK Integration Points

**SDK Client Creation**:
```go
import "github.com/openai/openai-go"

client := openai.NewClient(
    option.WithAPIKey(config.APIKey),
    option.WithBaseURL(config.BaseURL),
    option.WithTimeout(config.Timeout),
)
```

**Type Mapping**:
| Spin Type | SDK Type | Conversion |
|-----------|----------|------------|
| `llm.CompletionRequest` | `openai.ChatCompletionNewParams` | Convert messages, tools, parameters |
| `llm.CompletionResponse` | `openai.ChatCompletion` | Extract content, tool calls, usage |
| `llm.StreamChunk` | `openai.ChatCompletionChunk` | Map chunk types and deltas |
| `llm.Message` | `openai.ChatCompletionMessageParamUnion` | Map roles and content |
| `llm.ToolCall` | `openai.ChatCompletionMessageToolCall` | Direct mapping |
| `llm.Tool` | `openai.ChatCompletionToolParam` | Convert function schema |

**Error Mapping**:
| SDK Error | Spin Error | HTTP Code |
|-----------|------------|-----------|
| `openai.APIError` with 401 | `llm.ErrAuthentication` | 401 |
| `openai.APIError` with 429 | `llm.ErrRateLimit` | 429 |
| `openai.APIError` with 500+ | `llm.ErrProviderError` | 5xx |
| `context.DeadlineExceeded` | `llm.ErrTimeout` | - |
| `context.Canceled` | `context.Canceled` (pass-through) | - |
| Network errors | `llm.ErrConnection` | - |

### Implementation Plan

#### Phase 1: Setup and Dependency
1. Add `github.com/openai/openai-go v0.1.0-alpha.37` to `go.mod`
2. Verify no conflicts with existing dependencies
3. Review SDK documentation and examples

#### Phase 2: Type Converters (TDD)
1. Create `internal/llm/openai/convert.go` with conversion functions:
   - `convertRequest(llm.CompletionRequest) openai.ChatCompletionNewParams`
   - `convertMessages([]llm.Message) []openai.ChatCompletionMessageParamUnion`
   - `convertTools([]llm.Tool) []openai.ChatCompletionToolParam`
   - `convertResponse(openai.ChatCompletion) *llm.CompletionResponse`
   - `convertChunk(openai.ChatCompletionChunk) llm.StreamChunk`
2. Write comprehensive tests for each converter (table-driven)
3. Handle edge cases: empty arrays, nil pointers, invalid JSON

#### Phase 3: Provider Core (TDD)
1. Refactor `provider.go`:
   - Replace `http.Client` with `openai.Client`
   - Update `Complete()` to use `client.Chat.Completions.New()`
   - Update `Stream()` to use `client.Chat.Completions.NewStreaming()`
   - Update `Models()` to use `client.Models.List()`
2. Implement error mapping in `mapError(error) error`
3. Preserve goroutine lifecycle for streaming
4. Write unit tests for each method

#### Phase 4: Error Handling
1. Create `internal/llm/openai/errors.go` with error mapper
2. Test all error paths (401, 429, 500, timeout, cancel, network)
3. Ensure errors are actionable and include context

#### Phase 5: Integration Testing
1. Run existing test suite: `go test ./internal/llm/openai/...`
2. Add new tests for SDK-specific behavior
3. Test with real OpenAI API (optional, requires API key)
4. Verify coverage ≥90%

#### Phase 6: Cleanup
1. Delete `api.go` (custom HTTP types)
2. Remove SSE parsing code
3. Update `doc.go` to reference SDK
4. Run `make deadcode` and remove unused functions

### Data Flow

**Synchronous Completion**:
```
User Request (llm.CompletionRequest)
    ↓
convertRequest() → openai.ChatCompletionNewParams
    ↓
client.Chat.Completions.New(ctx, params)
    ↓
SDK handles: HTTP, auth, retry, parse
    ↓
convertResponse() → llm.CompletionResponse
    ↓
Return to caller
```

**Streaming**:
```
User Request (llm.CompletionRequest)
    ↓
convertRequest() → openai.ChatCompletionNewParams
    ↓
client.Chat.Completions.NewStreaming(ctx, params)
    ↓
Goroutine spawned:
  - Read from SDK stream
  - Convert each chunk with convertChunk()
  - Send to channel
  - Handle context cancellation
  - Close channel on EOF/error
    ↓
Return channel to caller
```

## Testing Strategy

### Unit Tests (90%+ coverage)

**Test Files**:
- `convert_test.go` - Type conversion tests
- `provider_test.go` - Provider method tests
- `errors_test.go` - Error mapping tests

**Test Scenarios**:
1. **Basic Completion**:
   - Simple prompt → response
   - Multi-turn conversation
   - System message handling
   
2. **Streaming**:
   - Content delta chunks
   - Tool call chunks
   - Finish reason handling
   - Context cancellation mid-stream
   - Error during stream

3. **Tool Calling**:
   - Single tool call
   - Multiple parallel tool calls
   - Tool call with complex arguments (nested JSON)
   - Round-trip: request → tool call → tool result → response

4. **Error Handling**:
   - 401 Unauthorized → `llm.ErrAuthentication`
   - 429 Rate Limit → `llm.ErrRateLimit`
   - 500 Server Error → `llm.ErrProviderError`
   - Timeout → `llm.ErrTimeout`
   - Context cancel → `context.Canceled`
   - Network error → `llm.ErrConnection`

5. **Edge Cases**:
   - Empty message list
   - Nil tool calls
   - Very long responses (>10k tokens)
   - Rapid successive requests (connection reuse)
   - Invalid JSON in function arguments

### Integration Tests

1. **Existing Test Suite**: All 1294 lines of tests must pass
2. **Race Detector**: `go test -race ./internal/llm/openai/...` must be clean
3. **Leak Detector**: Verify no goroutine leaks with `goleak`
4. **Benchmark**: Compare performance with current implementation

### Manual Testing (Optional)

1. Run against real OpenAI API with valid key
2. Test with `gpt-4`, `gpt-3.5-turbo`, `gpt-4-turbo-preview`
3. Verify tool calling with complex nested arguments
4. Test streaming with long responses

## Error Handling

### Error Taxonomy

1. **Authentication Errors** (`llm.ErrAuthentication`):
   - Invalid API key
   - Expired API key
   - Organization not authorized

2. **Rate Limit Errors** (`llm.ErrRateLimit`):
   - Too many requests
   - Token limit exceeded
   - Include `Retry-After` header if available

3. **Provider Errors** (`llm.ErrProviderError`):
   - 500 Internal Server Error
   - 502 Bad Gateway
   - 503 Service Unavailable
   - Model overloaded

4. **Client Errors** (`llm.ErrInvalidRequest`):
   - Invalid model name
   - Invalid parameters
   - Malformed request

5. **Timeout Errors** (`llm.ErrTimeout`):
   - Context deadline exceeded
   - Request timeout

6. **Connection Errors** (`llm.ErrConnection`):
   - DNS resolution failed
   - Connection refused
   - TLS handshake failed

### Error Handling Strategy

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
        return llm.ErrTimeout
    }

    // SDK API errors
    var apiErr *openai.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 401, 403:
            return llm.ErrAuthentication
        case 429:
            return llm.ErrRateLimit
        case 400, 404, 422:
            return llm.ErrInvalidRequest
        case 500, 502, 503:
            return llm.ErrProviderError
        default:
            return llm.ErrProviderError
        }
    }

    // Network errors
    if isNetworkError(err) {
        return llm.ErrConnection
    }

    // Unknown error
    return fmt.Errorf("openai: %w", err)
}
```

## Risks and Mitigation

### High Risk

1. **Breaking Changes in SDK**
   - **Risk**: SDK is alpha, API may change
   - **Mitigation**: Pin exact version `v0.1.0-alpha.37`, test thoroughly before upgrading

2. **Goroutine Leaks**
   - **Risk**: Improper stream cleanup
   - **Mitigation**: Use `goleak` in tests, verify with `runtime.NumGoroutine()`

3. **Type Conversion Errors**
   - **Risk**: SDK types differ from current types
   - **Mitigation**: Comprehensive converter tests, round-trip testing

### Medium Risk

1. **Performance Regression**
   - **Risk**: SDK adds overhead
   - **Mitigation**: Benchmark before/after, accept 5% overhead for maintainability

2. **Tool Calling Format**
   - **Risk**: SDK formats tool calls differently
   - **Mitigation**: Test with real OpenAI API, verify exact format

3. **Error Message Changes**
   - **Risk**: Different error messages break parsing
   - **Mitigation**: Map to semantic error codes, not strings

### Low Risk

1. **Configuration Differences**
   - **Risk**: SDK expects different config format
   - **Mitigation**: Wrap SDK client, maintain current `Config` struct

2. **Dependency Conflicts**
   - **Risk**: SDK brings conflicting dependencies
   - **Mitigation**: Check `go mod why` and resolve before merging

## Success Metrics

### Code Quality

- ✅ Test coverage ≥90% (current: 91.4%)
- ✅ Zero lint errors (`make lint`)
- ✅ Zero dead code (`make deadcode`)
- ✅ Complexity ≤15 per function
- ✅ Race detector clean
- ✅ No goroutine leaks

### Functional

- ✅ All existing tests pass
- ✅ Streaming works with context cancellation
- ✅ Tool calling round-trip successful
- ✅ Error mapping correct for all error types
- ✅ Models() returns valid model list

### Performance

- ✅ Completion latency within 5% of current
- ✅ Streaming throughput within 5% of current
- ✅ Memory usage within 10% of current
- ✅ Connection reuse working (verify with `netstat`)

### Documentation

- ✅ FRD complete and approved
- ✅ Code comments updated
- ✅ `doc.go` references SDK
- ✅ `docs/` updated with migration notes
- ✅ AGENTS.md updated if needed

## Implementation Checklist

### Pre-Implementation
- [ ] Read all docs in `docs/`
- [x] Create FRD
- [ ] Review FRD with team
- [ ] Get FRD approval

### Phase 1: Setup (Day 1)
- [ ] Add `openai-go` to `go.mod`
- [ ] Review SDK documentation
- [ ] Create feature branch `feature/openai-sdk-migration`

### Phase 2: Type Converters (Day 2)
- [ ] Create `convert.go`
- [ ] Write tests for `convertRequest()`
- [ ] Implement `convertRequest()`
- [ ] Write tests for `convertResponse()`
- [ ] Implement `convertResponse()`
- [ ] Write tests for `convertChunk()`
- [ ] Implement `convertChunk()`
- [ ] Write tests for `convertMessages()`
- [ ] Implement `convertMessages()`
- [ ] Write tests for `convertTools()`
- [ ] Implement `convertTools()`
- [ ] Verify coverage ≥90%

### Phase 3: Provider Core (Day 3)
- [ ] Refactor `provider.go` - `Complete()`
- [ ] Write tests for `Complete()`
- [ ] Refactor `provider.go` - `Stream()`
- [ ] Write tests for `Stream()`
- [ ] Refactor `provider.go` - `Models()`
- [ ] Write tests for `Models()`
- [ ] Implement error mapper
- [ ] Write tests for error mapper

### Phase 4: Integration (Day 4)
- [ ] Run full test suite: `go test ./internal/llm/openai/...`
- [ ] Run race detector: `go test -race ./internal/llm/openai/...`
- [ ] Run goleak tests
- [ ] Benchmark vs current implementation

### Phase 5: Cleanup (Day 5)
- [ ] Delete `api.go`
- [ ] Remove SSE parsing code
- [ ] Update `doc.go`
- [ ] Run `make deadcode`
- [ ] Run `make lint`
- [ ] Fix all issues

### Phase 6: Documentation (Day 6)
- [ ] Update `docs/` with migration notes
- [ ] Update AGENTS.md if needed
- [ ] Update roadmap (mark Phase 2.2 complete)
- [ ] Create completion document

### Phase 7: Review & Merge (Day 7)
- [ ] Final test run
- [ ] Code review
- [ ] Merge to main branch

## Appendix

### SDK Resources

- **Repository**: https://github.com/openai/openai-go
- **Documentation**: https://pkg.go.dev/github.com/openai/openai-go
- **Examples**: https://github.com/openai/openai-go/tree/main/examples
- **Version**: v0.1.0-alpha.37 (pin this version)

### Current vs SDK Comparison

| Aspect | Current (Custom) | SDK (openai-go) |
|--------|------------------|-----------------|
| Lines of code | ~657 | ~200-300 (estimated) |
| HTTP handling | Manual | SDK |
| SSE parsing | Manual (~100 lines) | SDK |
| Retry logic | Custom | SDK (configurable) |
| Type safety | Custom structs | SDK types |
| Maintenance | Manual API tracking | Vendor-maintained |
| Features | Chat only | Chat + future features |
| Error handling | Custom parsing | Structured errors |
| Testing | Mock HTTP | Mock SDK (easier) |

### References

- Phase 2.1 Completion: `docs/llm-base-types-phase-2.1-completion.md`
- Architecture Overview: `specs/architecture-overview.md`
- Implementation Instructions: `instructions/istr-implement.md`
- AGENTS.md: Project development standards

---

**Document Version**: 1.0  
**Last Updated**: 2025-10-26  
**Next Review**: After implementation completion
