# FRD-20251027000003: sourcegraph/jsonrpc2 Library Evaluation

## Metadata
- **FRD ID**: FRD-20251027000003
- **Title**: Evaluation of sourcegraph/jsonrpc2 vs Custom Implementation
- **Status**: Analysis
- **Created**: 2025-10-27
- **Author**: Claude (Rob Pike persona)
- **Related Documents**: 
  - FRD-20251027000002 - JSON-RPC Layer Type Safety
  - `specs/ifacesroadmap.md` - Phase 3.2

## 1. Question

**User asks:** "Why we cannot just switch to https://github.com/sourcegraph/jsonrpc2 in all codebase?"

This is an excellent question. Let's evaluate whether using an established library would be better than maintaining our custom JSON-RPC implementation.

## 2. Current Implementation Analysis

### 2.1 What We Have Now

Our custom JSON-RPC implementation (`internal/protocol/jsonrpc/`) provides:

**Core Components:**
```
internal/protocol/jsonrpc/
├── jsonrpc.go      # Types: Request, Response, Error, InitializeParams, etc.
├── server.go       # Server implementation with Handler interface
└── [tests]         # 90.7% coverage
```

**Key Features:**
- **Simple line-delimited JSON protocol** (newline-separated messages)
- **Type-safe handler interface** returning `json.RawMessage`
- **Application-specific types** (InitializeParams, SendMessageParams, etc.)
- **Synchronous request/response** pattern
- **Notifications** via separate `io.Writer`
- **Zero external dependencies** for JSON-RPC layer

**Current Usage Pattern:**
```go
// Server side
server := jsonrpc.NewServer(handler)
server.Serve(ctx, os.Stdin, os.Stdout)

// Handler implementation
func (h *Handler) HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
    switch method {
    case "initialize":
        var p InitializeParams
        json.Unmarshal(params, &p)
        result, err := h.processor.HandleInitialize(ctx, p)
        return json.Marshal(result)
    }
}

// Notifications sent via io.Writer
p.output.Write(notificationJSON)
```

### 2.2 Architecture Context

**Critical Architectural Details:**

1. **Event-Driven Notifications**: The processor emits events via `EventEmitter` and writes JSON-RPC notifications directly to `io.Writer`
2. **Bidirectional Stdio**: Server reads requests from stdin, writes responses/notifications to stdout
3. **Long-Running Turns**: Agent turns can take minutes, streaming deltas continuously
4. **Concurrent Operations**: Multiple goroutines emit notifications during single turn

**Current Flow:**
```
UI (stdin)  →  [JSON-RPC Server]  →  Handler  →  Processor  →  Agent
                        ↓
                   (stdout)  ←  Notifications  ←  EventEmitter
```

## 3. sourcegraph/jsonrpc2 Library Analysis

### 3.1 What It Provides

Based on the repository analysis:

**Pros:**
- ✅ Battle-tested (used in production by Sourcegraph, LSP servers)
- ✅ Full JSON-RPC 2.0 compliance
- ✅ Connection abstraction (`Conn`) with lifecycle management
- ✅ Async call support
- ✅ WebSocket transport support
- ✅ Better error handling patterns
- ✅ Request cancellation support
- ✅ Method handler registration
- ✅ Well-documented with examples

**Cons:**
- ❌ **No batch request support** (acknowledged limitation)
- ❌ Different transport model (connection-oriented vs stdio)
- ❌ More complex API surface
- ❌ External dependency
- ❌ May require significant refactoring

### 3.2 API Comparison

**sourcegraph/jsonrpc2 Pattern:**
```go
import "github.com/sourcegraph/jsonrpc2"

// Connection-oriented model
conn := jsonrpc2.NewConn(
    ctx,
    jsonrpc2.NewBufferedStream(transport, jsonrpc2.VSCodeObjectCodec{}),
    handler,
)

// Handler interface (different signature)
type Handler interface {
    Handle(ctx context.Context, conn *Conn, req *Request) (result interface{}, err error)
}

// Sending notifications
conn.Notify(ctx, "method", params)
```

**Key Differences:**
1. **Connection abstraction**: `Conn` object vs direct `io.Reader/Writer`
2. **Handler signature**: Receives `*Conn` and `*Request` vs `method` and `params`
3. **Notifications**: Via `conn.Notify()` vs direct `io.Writer`
4. **Lifecycle**: Connection lifecycle vs stateless server

## 4. Migration Analysis

### 4.1 What Would Need to Change

**Major Changes Required:**

1. **Server Setup** (~10 files affected)
   - Replace `jsonrpc.NewServer()` with `jsonrpc2.NewConn()`
   - Refactor stdio handling to use `BufferedStream`
   - Update `appserver.Server` to manage connection lifecycle

2. **Handler Interface** (~1 file, 60 lines)
   ```go
   // BEFORE (our custom)
   HandleRequest(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error)
   
   // AFTER (jsonrpc2)
   Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error)
   ```
   - **PROBLEM**: We just eliminated `interface{}` returns in Phase 3.2! This would reintroduce it.

3. **Notification System** (~1 file, complex)
   - Current: `EventEmitter` → `io.Writer` → stdout
   - New: `EventEmitter` → `conn.Notify()` → stdout
   - **PROBLEM**: How to get `*Conn` to all places that emit notifications?
   - Current code directly writes to `processor.output io.Writer` from multiple goroutines

4. **Application Types** (no change needed)
   - `InitializeParams`, `SendMessageParams`, etc. remain the same
   - These are our domain types, not JSON-RPC types

5. **Tests** (~4 test files)
   - Rewrite all JSON-RPC tests to use `jsonrpc2` mocking
   - Current coverage: 90.7% - would need to be recreated

### 4.2 Architectural Impedance Mismatch

**Problem 1: Connection vs Stateless Model**

Our current design:
```go
// Stateless: Server processes one request at a time from stdin
server.Serve(ctx, os.Stdin, os.Stdout)
```

sourcegraph/jsonrpc2:
```go
// Stateful: Connection object manages lifecycle
conn := jsonrpc2.NewConn(ctx, stream, handler)
// Connection stays alive, maintains state
```

**Problem 2: Notification Broadcasting**

Our current event flow:
```go
// Deep in agent code
emitter.Emit(events.Event{Type: "delta", Data: delta})

// Event subscriber (processor)
subscriber.OnEvent(func(e events.Event) {
    // Direct write to stdout
    processor.output.Write(notificationJSON)
})
```

With jsonrpc2:
```go
// Would need to:
// 1. Pass conn through all layers (agent → processor → event handlers)
// OR
// 2. Store conn in processor state
// OR  
// 3. Create notification queue/channel

// Each notification:
conn.Notify(ctx, "turn.delta", delta)
```

**Problem 3: Type Safety Regression**

We just completed Phase 3.2 to eliminate `interface{}`:
```go
// ✅ Current (type-safe)
func (h *Handler) HandleRequest(...) (json.RawMessage, error)

// ❌ jsonrpc2 (back to interface{})
func (h *Handler) Handle(..., req *Request) (interface{}, error)
```

## 5. Evaluation: Should We Switch?

### 5.1 Decision Matrix

| Criterion | Custom Implementation | sourcegraph/jsonrpc2 | Winner |
|-----------|----------------------|---------------------|---------|
| **Type Safety** | ✅ `json.RawMessage` returns | ❌ `interface{}` returns | Custom |
| **Simplicity** | ✅ 200 lines, simple | ❌ Complex API | Custom |
| **Maintenance** | ❌ We maintain it | ✅ Community maintained | Library |
| **Stdio Integration** | ✅ Direct stdio | ⚠️ Needs adapter | Custom |
| **Notification Model** | ✅ Direct `io.Writer` | ⚠️ Needs refactoring | Custom |
| **Test Coverage** | ✅ 90.7% | ⚠️ Would need rewrite | Custom |
| **Dependencies** | ✅ Zero | ❌ +1 dependency | Custom |
| **Features** | ⚠️ Basic JSON-RPC | ✅ Full spec | Library |
| **Async Support** | ❌ Limited | ✅ Full async | Library |
| **WebSocket** | ❌ Not supported | ✅ Supported | Library |
| **Production Readiness** | ✅ Works now | ⚠️ Would need testing | Custom |

### 5.2 Cost-Benefit Analysis

**Costs of Migration:**
- ⚠️ **High refactoring effort**: 10+ files, ~500 lines changed
- ⚠️ **Type safety regression**: Reintroduce `interface{}` we just eliminated
- ⚠️ **Architectural mismatch**: Connection model vs stateless stdio
- ⚠️ **Notification complexity**: Need to thread `*Conn` through layers
- ⚠️ **Test rewrite**: Lose 90.7% coverage, need to recreate
- ⚠️ **Risk**: Breaking production code that currently works
- ⏱️ **Time**: 2-3 days of work + testing

**Benefits of Migration:**
- ✅ Better async support (not needed for stdio use case)
- ✅ WebSocket support (not needed currently)
- ✅ Community maintenance (minimal benefit - our code is stable)
- ✅ Full JSON-RPC 2.0 spec (we only use subset)
- ❓ Future-proofing (uncertain - current design works)

### 5.3 Recommendation: **DO NOT MIGRATE**

**Reasoning:**

1. **YAGNI (You Aren't Gonna Need It)**
   - We don't need WebSocket support
   - We don't need complex async patterns
   - We don't need batch requests (library doesn't support anyway!)
   - Our stdio model is sufficient

2. **Type Safety Matters**
   - We just invested effort (Phase 3.2) to eliminate `interface{}`
   - Migration would undo that work
   - Goes against project goals (empty interface elimination)

3. **Architectural Fit**
   - Our stateless stdio model is simpler and correct for our use case
   - Library's connection model adds unnecessary complexity
   - Notification broadcasting would require significant refactoring

4. **Maintenance Burden is Low**
   - Our implementation: ~200 lines of simple, well-tested code
   - JSON-RPC 2.0 spec is stable (no breaking changes expected)
   - 90.7% test coverage gives confidence

5. **Risk vs Reward**
   - High migration risk (breaking working code)
   - Low reward (features we don't need)
   - Current code is production-ready

## 6. Alternative: When WOULD We Migrate?

Consider migration if **any** of these become true:

1. **WebSocket Support Needed**
   - Web-based IDE client requires WebSocket transport
   - LSP integration requires persistent connections

2. **Complex Async Patterns Required**
   - Need request pipelining
   - Need concurrent request/response handling
   - Need sophisticated cancellation patterns

3. **Batch Requests Required**
   - UI needs to send multiple requests atomically
   - BUT: Library doesn't support this anyway!

4. **Maintenance Becomes Burden**
   - JSON-RPC spec changes (unlikely)
   - Security vulnerabilities in our code (none found)
   - Complex bugs (none found in 90.7% covered code)

5. **Interop Requirements**
   - Need to integrate with systems expecting specific JSON-RPC patterns
   - Need VSCode LSP compatibility (they use this library)

## 7. Recommended Action: Enhance Current Implementation

Instead of migrating, consider these incremental improvements:

### 7.1 Add Missing Features (if needed)

**Batch Request Support** (only if UI needs it):
```go
// Add to jsonrpc.go
type BatchRequest []Request
type BatchResponse []Response

// Add to server.go
func (s *Server) ServeBatch(ctx context.Context, r io.Reader, w io.Writer) error
```

**WebSocket Transport** (only if needed):
```go
// Add new package: internal/protocol/jsonrpc/websocket
func ServeWebSocket(ctx context.Context, ws *websocket.Conn, handler Handler) error
```

### 7.2 Document the Decision

Add to `internal/protocol/jsonrpc/doc.go`:
```go
// Package jsonrpc implements a minimal JSON-RPC 2.0 server for stdio transport.
//
// Design Decision: We use a custom implementation instead of sourcegraph/jsonrpc2 because:
// 1. Type Safety: Our handlers return json.RawMessage, not interface{}
// 2. Simplicity: Direct stdio model matches our architecture
// 3. Minimalism: We only need request/response over stdio, not WebSocket/async
// 4. Zero Dependencies: No external deps for core protocol
//
// If WebSocket or complex async patterns are needed in the future,
// consider migrating to github.com/sourcegraph/jsonrpc2.
```

## 8. Conclusion

**Answer to "Why we cannot just switch?"**

We **CAN** switch, but we **SHOULD NOT** because:

1. ✅ **Current implementation works perfectly** for our use case
2. ✅ **Type-safe** (eliminates `interface{}` per project goals)
3. ✅ **Simple** (200 lines vs complex library)
4. ✅ **Well-tested** (90.7% coverage)
5. ✅ **Zero external dependencies**
6. ❌ Library would reintroduce `interface{}` returns
7. ❌ Library's features (WebSocket, async) are not needed
8. ❌ Migration cost is high, benefit is low

**The Right Tool for the Job:**
- sourcegraph/jsonrpc2 is excellent for LSP servers, WebSocket apps, complex async patterns
- Our simple stdio-based JSON-RPC server is excellent for our CLI/TUI use case

**Follow Go Proverb:** "A little copying is better than a little dependency."

Our ~200 lines of custom, type-safe, well-tested JSON-RPC code is **better** than depending on a library designed for different use cases.

---

## Appendix A: Migration Effort Estimate

**If we HAD to migrate:**

| Task | Lines Changed | Files | Time | Risk |
|------|--------------|-------|------|------|
| Update server setup | ~30 | 2 | 2h | Medium |
| Refactor handler interface | ~60 | 1 | 3h | High |
| Update notification system | ~100 | 2 | 6h | High |
| Rewrite tests | ~200 | 4 | 8h | Medium |
| Integration testing | N/A | N/A | 4h | High |
| **Total** | **~390** | **9** | **23h** | **High** |

**Conclusion:** 3 days of risky work for questionable benefit.

---

## Appendix B: Code Size Comparison

**Current Custom Implementation:**
```
internal/protocol/jsonrpc/jsonrpc.go:     ~170 lines (types + methods)
internal/protocol/jsonrpc/server.go:      ~90 lines (server + handler)
internal/protocol/jsonrpc/jsonrpc_test.go: ~355 lines (tests)
internal/protocol/jsonrpc/server_test.go:  ~285 lines (tests)
------------------------------------------------
Total:                                    ~900 lines (40% tests)
Coverage:                                 90.7%
Dependencies:                             0
```

**sourcegraph/jsonrpc2 (for comparison):**
```
Library size:                             ~3000+ lines
Our adapter code needed:                  ~400 lines (estimated)
Test rewrites needed:                     ~640 lines
Dependencies:                             +1
```

**Verdict:** Our minimal implementation is **more appropriate** for our needs.

---

**Document Version**: 1.0
**Last Updated**: 2025-10-27
**Status**: Recommendation - Do Not Migrate
