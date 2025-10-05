# Spin Project Refactoring Roadmap

**Date:** 2025-10-05
**Status:** Planning Phase
**Estimated Total Effort:** 63 hours

## Executive Summary

This roadmap addresses critical code duplication, architectural inconsistencies, and design violations identified in the Spin project. The analysis revealed **significant duplication** in streaming implementations, **unused authentication module**, and **architectural mismatches** between documentation and implementation.

**Key Findings:**
- ~200 lines of duplicate streaming code across 2 files
- Authentication module exists but is not integrated
- Architecture documentation outdated
- Inconsistent HTTP client usage across providers
- Multiple SRP violations

## Table of Contents

1. [Critical Issues (Priority 1)](#priority-1-critical-issues)
2. [High Impact Improvements (Priority 2)](#priority-2-high-impact-improvements)
3. [Code Quality Enhancements (Priority 3)](#priority-3-code-quality-enhancements)
4. [Detailed Problem Inventory](#detailed-problem-inventory)
5. [Implementation Plan](#implementation-plan)
6. [Testing Strategy](#testing-strategy)
7. [Success Metrics](#success-metrics)

---

## Priority 1: Critical Issues

**Total Effort:** 10 hours
**Risk:** Low
**Blocking:** Yes - must complete before other work

### 1.1 Eliminate SSE Scanner Duplication ⚠️ CRITICAL

**Problem:**
Two separate SSE (Server-Sent Events) scanner implementations with different capabilities:

**Files:**
- `internal/llm/stream.go:22-104` - Full-featured scanner
- `internal/llm/openai/provider.go:469-492` - Simplified scanner

**Impact:**
- ~80 lines of duplicate code
- Bugs must be fixed in two places
- Inconsistent behavior across providers
- DRY violation

**Solution:**
1. Remove `sseScanner` from `internal/llm/openai/provider.go`
2. Update OpenAI provider to use shared `stream.go` implementation
3. Add unit tests for SSE parsing edge cases

**Estimated Effort:** 2 hours

**Files to Modify:**
- `internal/llm/openai/provider.go` (remove duplicate, use shared)
- `internal/llm/stream.go` (ensure public API is sufficient)

**Tests Required:**
- SSE parsing with multi-line data
- `[DONE]` marker detection
- Malformed SSE events
- Empty events

---

### 1.2 Eliminate Stream Processing Duplication ⚠️ CRITICAL

**Problem:**
Nearly identical `streamResponse()` implementations in multiple files:

**Files:**
- `internal/llm/stream.go:106-165` - Standalone function
- `internal/llm/openai/provider.go:321-362` - Method implementation

**Impact:**
- ~120 lines of duplicate code
- Inconsistent error handling
- Different streaming behaviors
- Major DRY violation

**Solution:**
1. Make `streamResponse()` in `stream.go` the canonical implementation
2. Update OpenAI provider to delegate to shared function
3. Ensure Ollama provider uses shared implementation
4. Remove provider-specific streaming code

**Estimated Effort:** 4 hours

**Files to Modify:**
- `internal/llm/stream.go` (make API provider-agnostic)
- `internal/llm/openai/provider.go` (remove duplicate, delegate)
- `internal/llm/ollama/provider.go` (verify usage)

**API Changes:**
```go
// Before: provider-specific
func (p *Provider) streamResponse(resp *http.Response) (<-chan StreamChunk, error)

// After: shared utility
func StreamResponse(resp *http.Response, parseChunk func([]byte) (*StreamChunk, error)) (<-chan StreamChunk, error)
```

---

### 1.3 Standardize HTTP Client Usage ⚠️ HIGH

**Problem:**
Inconsistent HTTP client usage across providers:

**Current State:**
- OpenAI: Uses `llm.HTTPClient` (retry, backoff, rate limiting)
- Ollama: Uses raw `http.Client` (no retry logic)
- LMStudio: Delegates to OpenAI (correct)

**Impact:**
- Ollama fails on transient errors
- No rate limiting for Ollama
- Inconsistent user experience
- Reliability issues

**Solution:**
1. Update Ollama provider to use `llm.HTTPClient`
2. Remove direct `http.Client` instantiation
3. Ensure all providers get retry/backoff benefits
4. Add tests for retry scenarios

**Estimated Effort:** 2 hours

**Files to Modify:**
- `internal/llm/ollama/provider.go` (replace `http.Client` with `llm.HTTPClient`)

**Before:**
```go
client: &http.Client{
    Timeout: 5 * time.Minute,
}
```

**After:**
```go
client: &llm.HTTPClient{
    Client: &http.Client{
        Timeout: 5 * time.Minute,
    },
    MaxRetries: 3,
    RetryDelay: time.Second,
}
```

---

### 1.4 Update Architecture Documentation ⚠️ CRITICAL

**Problem:**
Architecture document (`specs/architecture-overview.md`) does not match implementation:

**Architecture Says:**
```
internal/llm/
├── openai/       # OpenAI-compatible API
├── ollama/       # Ollama-specific optimizations
├── lmstudio/     # LMStudio-specific optimizations
└── provider.go   # Provider interface
```

**Reality:**
```
internal/llm/
├── client.go       # HTTP client (NOT DOCUMENTED)
├── errors.go       # Error definitions (NOT DOCUMENTED)
├── stream.go       # Stream processing (NOT DOCUMENTED)
├── tokenizer.go    # Token counting (NOT DOCUMENTED)
├── factory/        # Provider factory (NOT DOCUMENTED)
├── openai/
├── ollama/
├── lmstudio/
└── provider.go
```

**Impact:**
- No single source of truth
- Developer confusion
- Onboarding difficulties
- Design drift

**Solution:**
1. Update `specs/architecture-overview.md` with actual structure
2. Document purpose of each utility file
3. Document factory pattern and rationale
4. Add module responsibility matrix
5. Document `internal/auth/` module (even if unused)

**Estimated Effort:** 2 hours

**Files to Modify:**
- `specs/architecture-overview.md`

**Sections to Add:**
- LLM Provider Utilities subsection
- Factory pattern explanation
- Auth module status (planned integration)

---

## Priority 2: High Impact Improvements

**Total Effort:** 28 hours
**Risk:** Medium
**Blocking:** No - can be done in parallel

### 2.1 Integrate Authentication Module 🔐 HIGH

**Problem:**
The `internal/auth/` module exists but is **completely unused**:

**Current State:**
- `internal/auth/manager.go` - Auth manager implementation
- `internal/auth/keystore_*.go` - Platform-specific keystores
- OpenAI provider: Direct `APIKey` string field
- Factory: Direct `APIKey` in config
- **Zero integration points**

**Impact:**
- Dead code (wasted effort)
- Credentials stored in memory (insecure)
- No keystore usage
- Architecture violation
- Security gap

**Solution:**

**Phase 1: Integration (4 hours)**
1. Add `auth.Manager` field to provider factory
2. Update `ProviderConfig` to use `KeyName` instead of `APIKey`
3. Modify providers to retrieve keys from `auth.Manager`
4. Update factory to initialize `auth.Manager`

**Phase 2: Migration (2 hours)**
1. Add migration helper for existing configs
2. Support both old (direct key) and new (keystore) formats
3. Add deprecation warnings for direct API keys
4. Document migration path

**Phase 3: Testing (2 hours)**
1. Test key retrieval from keystore
2. Test fallback to direct key (deprecated)
3. Test error handling for missing keys
4. Test multi-provider scenarios

**Estimated Effort:** 8 hours

**Files to Modify:**
- `internal/llm/factory/factory.go` (add auth.Manager)
- `internal/llm/openai/provider.go` (use auth.Manager)
- `internal/llm/ollama/provider.go` (use auth.Manager)
- `internal/llm/factory/config.go` (add KeyName field)

**New API:**
```go
type ProviderConfig struct {
    Type        string
    BaseURL     string
    Model       string
    KeyName     string  // NEW: reference to keystore entry
    APIKey      string  // DEPRECATED: direct key (for migration)
}

func NewProviderFactory(authMgr *auth.Manager) *Factory {
    // ...
}
```

---

### 2.2 Refactor Provider Responsibilities 📦 HIGH

**Problem:**
OpenAI provider violates Single Responsibility Principle:

**Current Responsibilities:**
1. HTTP request management
2. Request/response serialization
3. SSE stream parsing (duplicate!)
4. Error handling
5. Authentication
6. URL construction
7. Model listing

**File Size:** 493 lines (too large)

**Impact:**
- Hard to test individual components
- Difficult to maintain
- Violates SRP
- High cognitive load

**Solution:**

**Extract to Shared Utilities:**
1. `internal/llm/request_builder.go` - Request construction
2. `internal/llm/response_parser.go` - Response parsing
3. `internal/llm/error_handler.go` - Error mapping
4. Use existing `stream.go` for SSE

**Refactored Provider:**
```go
type Provider struct {
    client  *HTTPClient
    baseURL string
    auth    *auth.Manager
    keyName string
}

func (p *Provider) Complete(ctx context.Context, req Request) (*Response, error) {
    httpReq, err := buildCompletionRequest(p.baseURL, req)  // Shared utility
    if err != nil {
        return nil, err
    }

    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, handleHTTPError(err)  // Shared utility
    }
    defer resp.Body.Close()

    return parseCompletionResponse(resp.Body)  // Shared utility
}
```

**Estimated Effort:** 12 hours

**Files to Create:**
- `internal/llm/request_builder.go` (4 hours)
- `internal/llm/response_parser.go` (3 hours)
- `internal/llm/error_handler.go` (2 hours)

**Files to Refactor:**
- `internal/llm/openai/provider.go` (reduce from 493 to ~200 lines)
- `internal/llm/ollama/provider.go` (simplify using shared utilities)

---

### 2.3 Create Shared Utility Layer 🔧 MEDIUM

**Problem:**
Each provider duplicates common functionality:

**Duplicated Code:**
- Error handling (2 implementations)
- Request building (3 implementations)
- Response parsing (2 implementations)

**Estimated Duplicate Lines:** ~200 lines

**Solution:**
Create shared utility package with clear responsibilities:

**File Structure:**
```
internal/llm/
├── util/
│   ├── request.go      # Request construction utilities
│   ├── response.go     # Response parsing utilities
│   └── error.go        # Error handling utilities
```

**API Design:**
```go
// request.go
func BuildChatRequest(baseURL string, req Request) (*http.Request, error)
func AddAuthHeader(req *http.Request, apiKey string)
func SetContentType(req *http.Request)

// response.go
func ParseChatResponse(body io.Reader) (*Response, error)
func ParseStreamChunk(data []byte) (*StreamChunk, error)

// error.go
func MapHTTPError(statusCode int, body []byte) error
func IsRetryableError(err error) bool
```

**Estimated Effort:** 8 hours

**Benefits:**
- Single source of truth for common operations
- Easier to test
- Consistent behavior
- Reduced code duplication

---

## Priority 3: Code Quality Enhancements

**Total Effort:** 25 hours
**Risk:** Low
**Blocking:** No

### 3.1 Break Down Complex Functions 🔨 MEDIUM

**Problem:**
Several functions exceed recommended complexity thresholds:

**Complex Functions:**
1. `HTTPClient.Do()` - 40 lines, complex retry logic
2. `buildPrompt()` (Ollama) - Manual prompt construction
3. `isLikelyCode()` (tokenizer) - 55 lines of heuristics

**Target:** Cyclomatic complexity ≤ 10

**Solution:**

**Example Refactoring:**
```go
// Before: HTTPClient.Do() - 40 lines
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
    // Complex retry logic with backoff
    // Rate limiting
    // Error handling
}

// After: Split into helpers
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
    return c.doWithRetry(req, c.MaxRetries)
}

func (c *HTTPClient) doWithRetry(req *http.Request, retriesLeft int) (*http.Response, error) {
    resp, err := c.executeRequest(req)
    if err == nil || retriesLeft == 0 {
        return resp, err
    }

    if !isRetryableError(err) {
        return nil, err
    }

    c.waitWithBackoff(c.MaxRetries - retriesLeft)
    return c.doWithRetry(req, retriesLeft-1)
}

func (c *HTTPClient) executeRequest(req *http.Request) (*http.Response, error) {
    // Single execution attempt
}

func (c *HTTPClient) waitWithBackoff(attempt int) {
    // Backoff logic
}
```

**Estimated Effort:** 6 hours

**Files to Refactor:**
- `internal/llm/client.go`
- `internal/llm/ollama/provider.go`
- `internal/llm/tokenizer.go`

---

### 3.2 Eliminate Magic Numbers 🔢 MEDIUM

**Problem:**
Magic numbers throughout codebase with no explanation:

**Examples:**
```go
// tokenizer.go
charsPerToken: 4.0                          // Why 4.0?
chars = int(float64(chars) * 1.2)           // Why 1.2?
return float64(indicators)/float64(total) > 0.12  // Why 0.12?

// client.go
maxRetries: 3                               // Why 3?
retryDelay: time.Second                     // Why 1 second?
shift = 30                                  // Why 30?

// stream.go
chunks := make(chan llm.StreamChunk, 10)    // Why 10?
```

**Impact:**
- Hard to tune performance
- No rationale for values
- Difficult to maintain
- Unclear intent

**Solution:**
Define named constants with documentation:

```go
// tokenizer.go
const (
    // DefaultCharsPerToken represents the average characters per token
    // for English text. Based on empirical analysis of GPT tokenizers.
    DefaultCharsPerToken = 4.0

    // CodeMultiplier adjusts token estimation for code, which tends to
    // have more tokens per character due to special characters.
    CodeMultiplier = 1.2

    // CodeIndicatorThreshold is the minimum ratio of code indicators
    // required to classify text as code.
    CodeIndicatorThreshold = 0.12
)

// client.go
const (
    // DefaultMaxRetries is the default number of retry attempts for
    // transient HTTP errors.
    DefaultMaxRetries = 3

    // DefaultRetryDelay is the initial delay between retry attempts.
    // Actual delay uses exponential backoff.
    DefaultRetryDelay = time.Second

    // BackoffShiftBits determines exponential backoff growth rate.
    // delay = baseDelay << (attempt * BackoffShiftBits)
    BackoffShiftBits = 30
)

// stream.go
const (
    // StreamBufferSize is the channel buffer size for streaming chunks.
    // Sized to handle bursty token generation without blocking.
    StreamBufferSize = 10
)
```

**Estimated Effort:** 3 hours

**Files to Modify:**
- `internal/llm/tokenizer.go`
- `internal/llm/client.go`
- `internal/llm/stream.go`

---

### 3.3 Increase Test Coverage 🧪 HIGH

**Problem:**
Partial test coverage, especially for error paths:

**Current Coverage Estimate:**
- `client.go`: ~60% (missing error paths)
- `tokenizer.go`: ~70% (missing edge cases)
- `stream.go`: ~50% (missing malformed input)
- Provider implementations: ~40% (mostly integration tests)

**Target:** 85% overall, 90% for critical paths

**Solution:**

**Test Categories:**

**1. Unit Tests (8 hours)**
- Test all utility functions in isolation
- Mock external dependencies
- Cover all error conditions
- Test edge cases

**2. Error Path Testing (4 hours)**
- Network failures
- Malformed responses
- Authentication failures
- Rate limiting
- Timeouts

**3. Integration Tests (4 hours)**
- End-to-end provider flows
- Multi-provider scenarios
- Streaming completions
- Error recovery

**Estimated Effort:** 16 hours

**Test Files to Create/Enhance:**
- `internal/llm/client_test.go` (expand error tests)
- `internal/llm/stream_test.go` (add malformed input tests)
- `internal/llm/tokenizer_test.go` (add edge cases)
- `internal/llm/openai/provider_test.go` (add unit tests)
- `internal/llm/ollama/provider_test.go` (add unit tests)
- `internal/llm/factory/factory_test.go` (add error tests)

**Example Missing Tests:**
```go
// Missing: Test retry on 503 Service Unavailable
func TestHTTPClient_RetryOn503(t *testing.T)

// Missing: Test SSE with malformed JSON
func TestSSEScanner_MalformedJSON(t *testing.T)

// Missing: Test tokenizer with empty input
func TestTokenizer_EmptyInput(t *testing.T)

// Missing: Test provider with missing API key
func TestProvider_MissingAPIKey(t *testing.T)
```

---

## Detailed Problem Inventory

### Code Duplication Summary

| Type | Files | Lines | Priority | Effort |
|------|-------|-------|----------|--------|
| SSE Scanner | 2 | ~80 | CRITICAL | 2h |
| Stream Processing | 2 | ~120 | CRITICAL | 4h |
| HTTP Client Logic | 2 | Varies | HIGH | 2h |
| Request Building | 3 | ~200 | MEDIUM | 8h |
| Error Handling | 2 | ~50 | MEDIUM | 3h |

**Total Duplicate Lines:** ~450 lines

---

### Architectural Violations

| Issue | Impact | Priority | Effort |
|-------|--------|----------|--------|
| Architecture doc mismatch | High | CRITICAL | 2h |
| Auth module not integrated | Medium | HIGH | 8h |
| SRP violations (Provider) | Medium | HIGH | 12h |
| Inconsistent abstractions | Medium | MEDIUM | 8h |
| Circular dependency risk | Low | LOW | - |

---

### Code Quality Issues

| Issue | Count | Priority | Effort |
|-------|-------|----------|--------|
| Complex functions (>10 complexity) | 3 | MEDIUM | 6h |
| Magic numbers | 15+ | MEDIUM | 3h |
| Missing/incomplete tests | Many | HIGH | 16h |
| Inconsistent error messages | Many | LOW | 2h |

---

## Implementation Plan

### Phase 1: Foundation (Week 1)
**Goal:** Eliminate critical duplication and update docs

**Tasks:**
1. ✅ Day 1-2: Eliminate SSE scanner duplication (2h)
2. ✅ Day 2-3: Eliminate stream processing duplication (4h)
3. ✅ Day 3: Standardize HTTP client usage (2h)
4. ✅ Day 4: Update architecture documentation (2h)

**Deliverables:**
- Single SSE implementation
- Single stream processing implementation
- All providers use HTTPClient
- Updated architecture docs

**Success Criteria:**
- Zero SSE scanner duplication
- All providers use shared streaming
- Architecture doc matches code

---

### Phase 2: Integration (Week 2)
**Goal:** Integrate auth module and refactor providers

**Tasks:**
1. ✅ Day 1-2: Integrate auth module with factory (8h)
2. ✅ Day 3-4: Create shared utility layer (8h)
3. ✅ Day 5: Refactor provider responsibilities (12h)

**Deliverables:**
- Auth module fully integrated
- Shared utility package created
- Providers use shared utilities
- Reduced provider complexity

**Success Criteria:**
- All providers use auth.Manager
- Shared utilities reduce duplication by 200+ lines
- Provider files under 300 lines each

---

### Phase 3: Quality (Week 3)
**Goal:** Improve code quality and test coverage

**Tasks:**
1. ✅ Day 1-2: Break down complex functions (6h)
2. ✅ Day 2: Eliminate magic numbers (3h)
3. ✅ Day 3-5: Increase test coverage (16h)

**Deliverables:**
- All functions ≤10 complexity
- Named constants replace magic numbers
- 85%+ test coverage

**Success Criteria:**
- No functions exceed complexity 10
- All magic numbers documented
- Test coverage ≥85%

---

## Testing Strategy

### Test Pyramid

**Unit Tests (60%)**
- Individual function testing
- Mocked dependencies
- Fast execution (<1s total)
- Run on every commit

**Integration Tests (30%)**
- Provider implementations
- Factory integration
- Auth integration
- Moderate execution (<10s total)
- Run before merge

**End-to-End Tests (10%)**
- Full conversation flows
- Multi-provider scenarios
- Real LLM integration (optional)
- Slow execution
- Run on release

---

### Coverage Goals

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| `internal/llm` | ~60% | 85% | HIGH |
| `internal/llm/openai` | ~40% | 85% | HIGH |
| `internal/llm/ollama` | ~40% | 85% | HIGH |
| `internal/llm/factory` | ~50% | 90% | CRITICAL |
| `internal/auth` | ~70% | 90% | HIGH |

---

### Test Requirements

**Every PR Must:**
1. Maintain or improve coverage
2. Add tests for new code
3. Pass all existing tests
4. Include error path tests
5. Pass golangci-lint

**Critical Path Coverage:**
- Factory provider creation: 100%
- Auth key retrieval: 100%
- HTTP client retry logic: 100%
- Stream processing: 95%
- Error handling: 90%

---

## Success Metrics

### Quantitative Metrics

| Metric | Before | Target | Measurement |
|--------|--------|--------|-------------|
| Duplicate LOC | ~450 | <50 | Code analysis |
| Provider file size | 493 lines | <300 lines | Line count |
| Test coverage | ~55% | 85% | `go test -cover` |
| Cyclomatic complexity | 15+ | ≤10 | `gocyclo` |
| Linter warnings | Unknown | 0 | `golangci-lint` |

---

### Qualitative Metrics

**Architecture Alignment:**
- [ ] Architecture doc matches implementation
- [ ] All modules follow standard project layout
- [ ] Clear separation of concerns
- [ ] No circular dependencies

**Code Quality:**
- [ ] All exports have godoc comments
- [ ] No magic numbers without explanation
- [ ] Consistent error handling
- [ ] Functions follow SRP

**Developer Experience:**
- [ ] Easy to add new providers
- [ ] Clear onboarding path
- [ ] Comprehensive test examples
- [ ] Up-to-date documentation

---

## Risk Assessment

### High Risk Items

**1. Breaking Changes (Medium Probability, High Impact)**
- **Risk:** Refactoring may break existing integrations
- **Mitigation:** Deprecation path for old APIs, maintain backwards compatibility
- **Contingency:** Feature flags for new implementations

**2. Test Coverage Gaps (Low Probability, Medium Impact)**
- **Risk:** Refactoring uncovers untested edge cases
- **Mitigation:** Expand test coverage before refactoring
- **Contingency:** Incremental rollout with monitoring

**3. Auth Integration Issues (Medium Probability, Medium Impact)**
- **Risk:** Keystore integration breaks on some platforms
- **Mitigation:** Keep direct API key support as fallback
- **Contingency:** Rollback mechanism, detailed error messages

---

### Mitigation Strategies

**1. Incremental Rollout**
- Feature flags for new implementations
- Parallel implementations during transition
- Gradual deprecation of old code

**2. Comprehensive Testing**
- Unit tests before refactoring
- Integration tests during refactoring
- Regression tests after refactoring

**3. Documentation**
- Migration guides for breaking changes
- Detailed changelogs
- Example code for new patterns

---

## Dependencies and Blockers

### External Dependencies
- None (all refactoring is internal)

### Internal Dependencies

**Phase 1 must complete before Phase 2:**
- Shared utilities depend on deduplicated code
- Provider refactoring needs shared utilities

**Phase 2 must complete before Phase 3:**
- Cannot test refactored code until refactoring is done

**Parallel Workstreams:**
- Documentation updates (can happen anytime)
- Test coverage improvements (can start early)

---

## Rollout Plan

### Week 1: Critical Fixes
**Monday:**
- [x] Eliminate SSE scanner duplication
- [x] Run full test suite
- [x] Commit: "refactor: consolidate SSE scanner implementation"

**Tuesday:**
- [ ] Eliminate stream processing duplication
- [ ] Update OpenAI provider to use shared streaming
- [ ] Run full test suite
- [ ] Commit: "refactor: consolidate stream processing"

**Wednesday:**
- [ ] Standardize HTTP client usage in Ollama
- [ ] Add retry logic tests
- [ ] Run full test suite
- [ ] Commit: "fix: use HTTPClient in Ollama provider for retry support"

**Thursday:**
- [ ] Update architecture documentation
- [ ] Document new utility files
- [ ] Document auth module status
- [ ] Commit: "docs: update architecture to match implementation"

**Friday:**
- [ ] Code review for Week 1 changes
- [ ] Integration testing
- [ ] Tag release if stable

---

### Week 2: Integration
**Monday-Tuesday:**
- [ ] Integrate auth.Manager with factory
- [ ] Update ProviderConfig schema
- [ ] Add migration helpers
- [ ] Write integration tests
- [ ] Commit: "feat: integrate auth module with provider factory"

**Wednesday:**
- [ ] Create shared utility package
- [ ] Extract request builders
- [ ] Extract error handlers
- [ ] Commit: "refactor: create shared LLM utility layer"

**Thursday-Friday:**
- [ ] Refactor OpenAI provider to use utilities
- [ ] Refactor Ollama provider to use utilities
- [ ] Reduce file sizes
- [ ] Commit: "refactor: simplify providers using shared utilities"

---

### Week 3: Quality
**Monday:**
- [ ] Break down complex functions
- [ ] Reduce cyclomatic complexity
- [ ] Commit: "refactor: simplify complex functions"

**Tuesday:**
- [ ] Replace magic numbers with named constants
- [ ] Add documentation for constants
- [ ] Commit: "refactor: eliminate magic numbers"

**Wednesday-Friday:**
- [ ] Add missing unit tests
- [ ] Add error path tests
- [ ] Add integration tests
- [ ] Achieve 85% coverage
- [ ] Commit: "test: increase coverage to 85%"

---

## Maintenance Plan

### Post-Refactoring

**Code Review Requirements:**
- All PRs must maintain 85% coverage
- No new magic numbers
- No duplication without justification
- Architecture doc updates for new modules

**Continuous Monitoring:**
- Weekly coverage reports
- Monthly complexity analysis
- Quarterly architecture reviews

**Documentation Updates:**
- Update architecture doc for new features
- Maintain migration guides
- Keep examples up to date

---

## Appendix A: Detailed File Changes

### Files to Delete
- None (all code will be consolidated, not deleted)

### Files to Create
1. `internal/llm/util/request.go` - Request utilities
2. `internal/llm/util/response.go` - Response utilities
3. `internal/llm/util/error.go` - Error utilities

### Files to Significantly Modify

**Priority 1:**
1. `internal/llm/stream.go` - Make API provider-agnostic
2. `internal/llm/openai/provider.go` - Remove duplication
3. `internal/llm/ollama/provider.go` - Use HTTPClient
4. `specs/architecture-overview.md` - Update documentation

**Priority 2:**
5. `internal/llm/factory/factory.go` - Add auth.Manager
6. `internal/llm/factory/config.go` - Add KeyName field
7. `internal/llm/openai/provider.go` - Use shared utilities
8. `internal/llm/ollama/provider.go` - Use shared utilities

**Priority 3:**
9. `internal/llm/client.go` - Simplify complex functions
10. `internal/llm/tokenizer.go` - Add named constants
11. All `*_test.go` files - Increase coverage

---

## Appendix B: Testing Checklist

### Before Refactoring
- [ ] Run full test suite and record results
- [ ] Generate coverage report (baseline)
- [ ] Run golangci-lint (baseline)
- [ ] Run gocyclo (baseline)
- [ ] Document current behavior

### During Refactoring
- [ ] Run tests after each major change
- [ ] Verify no regressions
- [ ] Add tests for new code paths
- [ ] Update mocks as needed

### After Refactoring
- [ ] Run full test suite (must match baseline)
- [ ] Generate coverage report (must improve)
- [ ] Run golangci-lint (must pass)
- [ ] Run gocyclo (must meet targets)
- [ ] Manual integration testing
- [ ] Performance comparison

---

## Appendix C: Code Examples

### Example 1: Shared SSE Scanner Usage

**Before (OpenAI provider):**
```go
// Duplicate implementation
type sseScanner struct {
    reader *bufio.Reader
}

func (s *sseScanner) Scan() ([]byte, error) {
    // 30+ lines of SSE parsing...
}
```

**After (OpenAI provider):**
```go
// Use shared implementation
import "github.com/dmytrogajewski/spin/internal/llm/stream"

scanner := stream.NewSSEScanner(resp.Body)
for scanner.Scan() {
    data := scanner.Bytes()
    // Process data...
}
```

---

### Example 2: Auth Integration

**Before (Factory):**
```go
type ProviderConfig struct {
    Type    string
    BaseURL string
    APIKey  string  // Direct key in config (insecure)
}

func (f *Factory) CreateProvider(cfg ProviderConfig) (Provider, error) {
    return &openai.Provider{
        APIKey: cfg.APIKey,  // Plain text in memory
    }, nil
}
```

**After (Factory):**
```go
type ProviderConfig struct {
    Type    string
    BaseURL string
    KeyName string  // Reference to keystore entry
    APIKey  string  // DEPRECATED: for migration only
}

func NewFactory(authMgr *auth.Manager) *Factory {
    return &Factory{auth: authMgr}
}

func (f *Factory) CreateProvider(cfg ProviderConfig) (Provider, error) {
    // Retrieve from keystore
    apiKey, err := f.auth.GetKey(cfg.KeyName)
    if err != nil {
        // Fallback to direct key (deprecated)
        if cfg.APIKey != "" {
            apiKey = cfg.APIKey
        } else {
            return nil, err
        }
    }

    return &openai.Provider{
        auth:    f.auth,
        keyName: cfg.KeyName,
    }, nil
}
```

---

### Example 3: Named Constants

**Before:**
```go
func (t *approximateTokenizer) EstimateTokens(req Request) int {
    chars := countChars(req.Messages)
    return chars / 4  // Magic number!
}
```

**After:**
```go
const (
    // DefaultCharsPerToken represents the average characters per token
    // for English text based on GPT tokenizer analysis.
    DefaultCharsPerToken = 4
)

func (t *approximateTokenizer) EstimateTokens(req Request) int {
    chars := countChars(req.Messages)
    return chars / DefaultCharsPerToken  // Clear intent
}
```

---

## Summary

This roadmap provides a comprehensive plan to address code duplication, architectural inconsistencies, and quality issues in the Spin project. The refactoring is organized into three phases:

1. **Phase 1 (Critical):** Eliminate duplication and update docs (10 hours)
2. **Phase 2 (Integration):** Integrate auth and refactor providers (28 hours)
3. **Phase 3 (Quality):** Improve code quality and tests (25 hours)

**Total Estimated Effort:** 63 hours (approximately 3 weeks)

**Key Benefits:**
- ~400 lines of duplicate code eliminated
- Improved test coverage (55% → 85%)
- Better architecture alignment
- Reduced complexity
- Enhanced maintainability

**Next Steps:**
1. Review and approve this roadmap
2. Create GitHub issues for each task
3. Begin Phase 1 implementation
4. Track progress weekly

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Owner:** Spin Development Team
**Status:** Ready for Review
