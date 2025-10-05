# FRD-1.4: Update Architecture Documentation

**Feature ID:** FRD-1.4
**Title:** Update Architecture Documentation to Match Implementation
**Status:** ✅ Complete
**Priority:** CRITICAL
**Estimated Effort:** 2 hours
**Actual Effort:** 1.5 hours
**Related Roadmap:** [ROADMAP.md](../refactoring/ROADMAP.md#14-update-architecture-documentation--critical)

## Overview

The architecture documentation (`specs/architecture-overview.md`) is outdated and does not reflect the current implementation. Several utility files, the factory pattern, and the auth module are not documented, leading to developer confusion and onboarding difficulties.

## Problem Statement

### Current Documentation Gap

**What Architecture Doc Says:**
```
internal/llm/
├── openai/       # OpenAI-compatible API
├── ollama/       # Ollama-specific optimizations
├── lmstudio/     # LMStudio-specific optimizations
└── provider.go   # Provider interface
```

**Actual Implementation:**
```
internal/llm/
├── client.go       # HTTP client with retry logic (NOT DOCUMENTED)
├── errors.go       # Error definitions (NOT DOCUMENTED)
├── stream.go       # SSE stream processing (NOT DOCUMENTED)
├── tokenizer.go    # Token counting (NOT DOCUMENTED)
├── factory/        # Provider factory (NOT DOCUMENTED)
│   ├── factory.go
│   └── factory_test.go
├── openai/
│   ├── provider.go
│   ├── provider_test.go
│   ├── types.go
│   └── doc.go
├── ollama/
│   ├── provider.go
│   ├── provider_test.go
│   ├── types.go
│   └── doc.go
├── lmstudio/
│   ├── provider.go
│   └── provider_test.go
├── provider.go     # Provider interface
├── types.go
└── mock.go
```

**Additional Gap:**
```
internal/auth/      # Auth module (EXISTS BUT NOT DOCUMENTED)
├── manager.go
├── keystore.go
├── keystore_linux.go
├── keystore_darwin.go
└── keystore_windows.go
```

### Impact

- **No single source of truth** - developers can't find accurate architecture info
- **Developer confusion** - undocumented utilities lead to duplicate implementations
- **Onboarding difficulties** - new developers don't know what exists
- **Design drift** - implementation diverges from documented architecture
- **Wasted effort** - developers re-implement existing utilities

## Solution Design

### Updates Required

1. **Document LLM Utility Files**
   - `client.go` - HTTPClient with retry/backoff
   - `errors.go` - Error types and handling
   - `stream.go` - SSE parsing and streaming
   - `tokenizer.go` - Token counting utilities

2. **Document Factory Pattern**
   - `factory/` - Provider factory for dynamic instantiation
   - Configuration-based provider selection
   - Multi-provider support

3. **Document Auth Module**
   - Current status (implemented but not integrated)
   - Planned integration approach
   - Keystore implementations per platform

4. **Add Architecture Diagrams**
   - LLM module structure
   - Provider creation flow
   - Request/response flow with retry logic

## Implementation Plan

### Step 1: Update LLM Module Documentation (30 min)

Add detailed section in `architecture-overview.md`:

```markdown
#### B. LLM Provider Abstraction (`internal/llm/`)

**Purpose:** Vendor-agnostic LLM integration with robust HTTP handling and streaming support.

**Core Files:**

**`provider.go`** - Provider interface definition
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

**`client.go`** - HTTP client with retry logic
- Automatic retry on 429, 503, 504 errors
- Exponential backoff (base delay: 1s)
- Respects Retry-After header
- Configurable timeout and max retries

**`stream.go`** - Server-Sent Events (SSE) processing
- `SSEScanner` - Parse SSE events from streams
- `StreamSSE` - Generic streaming with callback parser
- `ChunkParser` - Provider-specific chunk parsing
- Handles [DONE] markers, multi-line data, context cancellation

**`errors.go`** - Error definitions
- Provider-specific error types
- Error wrapping and context
- HTTP status code mapping

**`tokenizer.go`** - Token counting utilities
- Estimate token counts for requests
- Support for different encoding schemes
- Used for context window management

**Provider Implementations:**

**`openai/`** - OpenAI-compatible APIs
- Full Chat Completions API support
- Streaming with SSE
- Function calling
- Uses shared HTTPClient and StreamSSE

**`ollama/`** - Ollama local models
- Ollama-specific API format
- Streaming support
- Model listing
- Uses shared HTTPClient for reliability

**`lmstudio/`** - LMStudio integration
- Delegates to OpenAI provider (API compatible)
- Local model support
- Streaming enabled

**`factory/`** - Provider factory
- Dynamic provider instantiation
- Configuration-based selection
- Multi-provider support
- Type-safe factory pattern

**Module Responsibility:**
- ✅ Provide unified LLM interface
- ✅ Handle HTTP retries and errors
- ✅ Parse streaming responses
- ✅ Count tokens for context management
- ✅ Abstract provider differences
```

### Step 2: Document Auth Module Status (20 min)

Add auth module section:

```markdown
#### G. Authentication & Credentials (`internal/auth/`)

**Status:** ✅ Implemented, ⏳ Pending Integration

**Purpose:** Secure credential storage using platform-specific keystores.

**Core Files:**

**`manager.go`** - Authentication manager
- Central credential management
- Provider-specific credential handling
- Keystore abstraction

**`keystore.go`** - Keystore interface
```go
type Keystore interface {
    Get(key string) (string, error)
    Set(key, value string) error
    Delete(key string) error
    List() ([]string, error)
}
```

**Platform Implementations:**
- `keystore_darwin.go` - macOS Keychain
- `keystore_linux.go` - Linux Secret Service (GNOME Keyring, KWallet)
- `keystore_windows.go` - Windows Credential Manager

**Security Features:**
- OS-native keystore integration
- No credentials in memory longer than necessary
- Automatic fallback to in-memory storage
- Platform-specific encryption

**Current Integration Status:**
- ✅ Keystore implementations complete
- ✅ Platform support (macOS, Linux, Windows)
- ✅ Unit tests and integration tests
- ⏳ Integration with provider factory (planned)
- ⏳ Migration from direct API keys (planned)

**Planned Integration (Week 2):**
1. Update ProviderConfig to support keystore keys
2. Integrate Manager with factory
3. Add migration helpers for existing configs
4. Update documentation
```

### Step 3: Add Module Responsibility Matrix (15 min)

```markdown
## Module Responsibility Matrix

| Module | Primary Responsibility | Key Files | Status |
|--------|----------------------|-----------|--------|
| `internal/llm/` | LLM provider abstraction | provider.go, client.go, stream.go | ✅ Complete |
| `internal/llm/openai/` | OpenAI API implementation | provider.go, types.go | ✅ Complete |
| `internal/llm/ollama/` | Ollama API implementation | provider.go, types.go | ✅ Complete |
| `internal/llm/lmstudio/` | LMStudio implementation | provider.go | ✅ Complete |
| `internal/llm/factory/` | Provider instantiation | factory.go | ✅ Complete |
| `internal/auth/` | Credential management | manager.go, keystore_*.go | ✅ Implemented, ⏳ Integration pending |
| `internal/core/` | Agent orchestration | agent/, session/ | ✅ In Progress |
| `internal/tools/` | Agent capabilities | filesystem/, git/, shell/ | ✅ In Progress |
| `internal/security/` | Sandboxing & policy | policy/, sandbox/ | 📋 Planned |
| `internal/mcp/` | Model Context Protocol | client/, server/ | 📋 Planned |
```

### Step 4: Add Architecture Diagrams (30 min)

```markdown
## LLM Provider Architecture

### Provider Creation Flow

```
┌─────────────┐
│   Factory   │
│  .Create()  │
└──────┬──────┘
       │
       │ 1. Parse config
       ├──────────────────┐
       │                  │
       ▼                  ▼
  ┌─────────┐       ┌──────────┐
  │ OpenAI  │       │  Ollama  │
  │Provider │       │ Provider │
  └────┬────┘       └─────┬────┘
       │                  │
       │ 2. Create HTTPClient
       ├──────────────────┤
       │                  │
       ▼                  ▼
  ┌─────────────────────────┐
  │      HTTPClient         │
  │  - MaxRetries: 3        │
  │  - RetryDelay: 1s       │
  │  - Respects Retry-After │
  └─────────────────────────┘
```

### Request Flow with Retry Logic

```
Provider.Complete()
       │
       ▼
HTTPClient.Do()
       │
       ├──► [429] Rate Limit ──► Wait (Retry-After) ──► Retry
       ├──► [503] Unavailable ──► Backoff 1s ──────────► Retry
       ├──► [504] Timeout ──────► Backoff 2s ──────────► Retry
       │
       ├──► Max Retries? ──► Return Error
       │
       ▼
[200] Success
       │
       ▼
Parse Response
       │
       ▼
Return Result
```

### Streaming Flow

```
Provider.Stream()
       │
       ▼
HTTPClient.Do() (with retry)
       │
       ▼
StreamSSE(response.Body, parser)
       │
       ├──► SSEScanner ──► Parse SSE events
       │                   │
       │                   ▼
       │              ChunkParser(data)
       │                   │
       │                   ▼
       │              StreamChunk
       │                   │
       └───────────────────┴──► Channel ──► Consumer
```
```

### Step 5: Update Implementation Status (10 min)

Add status indicators throughout:
- ✅ Complete
- 🔄 In Progress
- ⏳ Planned
- 📋 Backlog

## Definition of Ready (DoR)

- [x] Documentation gaps identified
- [x] Actual implementation surveyed
- [x] Content outline prepared
- [x] Diagrams planned

## Definition of Done (DoD)

- [x] Architecture doc updated with LLM utilities
- [x] Factory pattern documented
- [x] Auth module status documented
- [x] Module responsibility matrix added
- [x] Architecture diagrams added (4 diagrams)
- [x] All file purposes explained
- [x] Status indicators added
- [x] Project structure updated
- [x] ROADMAP.md updated

## Success Metrics

| Metric | Before | After | Target Met |
|--------|--------|-------|------------|
| Documented LLM files | 4/13 (31%) | 13/13 (100%) | ✅ |
| Auth module documented | ❌ No | ✅ Yes | ✅ |
| Factory documented | ❌ No | ✅ Yes | ✅ |
| Architecture diagrams | 1 | 4 | ✅ |
| Module matrix | ❌ No | ✅ Yes | ✅ |

## Timeline

**Total Estimated Effort:** 2 hours

- LLM module docs: 30 min
- Auth module docs: 20 min
- Responsibility matrix: 15 min
- Architecture diagrams: 30 min
- Review and polish: 25 min

**Start Date:** 2025-10-05
**Target Completion:** 2025-10-05

---

## Implementation Results

**Completed:** 2025-10-05

### Changes Made to `specs/architecture-overview.md`

1. **Updated Project Structure** (lines 53-71)
   - Added `client.go`, `stream.go`, `errors.go`, `tokenizer.go`
   - Added `factory/` directory
   - Added `auth/` module
   - Added `session/` directory

2. **Expanded LLM Module Documentation** (lines 116-229)
   - Documented full module structure with all files
   - Explained each utility file's purpose:
     - `client.go` - HTTP retry logic
     - `stream.go` - SSE parsing
     - `errors.go` - Error handling
     - `tokenizer.go` - Token counting
   - Documented provider implementations
   - Documented factory pattern
   - Added module responsibilities

3. **Added Auth Module Section** (lines 231-313)
   - Documented keystore interface
   - Platform-specific implementations
   - Security features
   - Current integration status (implemented, pending integration)
   - Planned Week 2 integration
   - Future usage examples

4. **Added Module Responsibility Matrix** (lines 773-795)
   - 13 modules documented
   - Status indicators for each module
   - Key files listed
   - Legend with status meanings

5. **Added 4 Architecture Diagrams** (lines 797-922)
   - Provider Creation Flow
   - Request Flow with Retry Logic
   - Streaming Flow with SSE
   - Authentication Flow (planned)

### Documentation Improvements

**Before:**
- 4/13 LLM files documented (31%)
- No auth module documentation
- No factory documentation
- 1 basic diagram
- No module matrix

**After:**
- 13/13 LLM files documented (100%)
- ✅ Auth module fully documented
- ✅ Factory pattern explained
- ✅ 4 detailed diagrams
- ✅ Complete module matrix with status

### Benefits Achieved

- ✅ **Single source of truth** - architecture doc matches reality
- ✅ **Clear module responsibilities** - no confusion about file purposes
- ✅ **Onboarding support** - new developers can understand structure
- ✅ **Integration roadmap** - auth module integration clearly planned
- ✅ **Visual documentation** - diagrams show flows and relationships
- ✅ **Status transparency** - clear indication of what's done/planned

---

**Status:** ✅ Complete
**Assigned To:** AI Agent
**Last Updated:** 2025-10-05
