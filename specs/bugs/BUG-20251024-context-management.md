# Context Management Issues

**Date:** 2025-10-24  
**Severity:** Medium  
**Status:** Fixed

## Summary

Investigation of context management patterns throughout the Spin codebase reveals several issues where contexts are not properly propagated, checked, or cancelled.

## Issues Found

### 1. Goroutine Without Context Cancellation in Manager (CRITICAL)

**Location:** `internal/manager/manager.go:558-575`

**Problem:**
A goroutine is launched to write debug events to a JSONL file, but it doesn't check for context cancellation. This goroutine will continue running even after the conversation context is cancelled.

```go
go func() {
    defer func() {
        m.emitter.Unsubscribe(subID)
        _ = f.Close()
    }()
    enc := json.NewEncoder(f)
    for ev := range ch {  // <-- No context check
        // Wrap event with session metadata
        record := map[string]any{
            "session_id": sess.ID,
            "timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
            "type":       ev.Type.String(),
            "data":       ev.Data,
        }
        // Best-effort write; ignore individual write errors
        _ = enc.Encode(record)
    }
}()
```

**Impact:**
- Goroutine leak when conversation ends
- File handle may not be closed promptly
- Wasted resources
- Potential deadlock if emitter channel blocks

**Fix:** ✅ FIXED
Added context checking using a select statement to monitor context cancellation:

```go
go func() {
    defer func() {
        m.emitter.Unsubscribe(subID)
        _ = f.Close()
    }()
    enc := json.NewEncoder(f)
    for {
        select {
        case ev, ok := <-ch:
            if !ok {
                return
            }
            record := map[string]any{
                "session_id": sess.ID,
                "timestamp":  ev.Timestamp.Format(time.RFC3339Nano),
                "type":       ev.Type.String(),
                "data":       ev.Data,
            }
            _ = enc.Encode(record)
        case <-ctx.Done():
            return
        }
    }
}()
```

**File:** `internal/manager/manager.go:564-583`

### 2. Stream Output Goroutines Without Context Cancellation

**Location:** `internal/agent/executor.go:601-611`

**Problem:**
Goroutines streaming stdout/stderr don't check for context cancellation:

```go
// Stream stdout
wg.Add(1)
go func() {
    defer wg.Done()
    e.streamOutput(stdout, "stdout", chunks)  // <-- No context check
}()

// Stream stderr
wg.Add(1)
go func() {
    defer wg.Done()
    e.streamOutput(stderr, "stderr", chunks)  // <-- No context check
}()
```

**Impact:**
- Output streams may continue reading after context cancellation
- Deadlock if reader blocks indefinitely
- Resource leak

**Fix:** ✅ FIXED
Added context parameter to `streamOutput` and check cancellation in read loop:

```go
func (e *Executor) streamOutput(ctx context.Context, r io.Reader, stream string, chunks chan<- OutputChunk) {
    buf := make([]byte, 4096)
    for {
        // Check context cancellation before reading
        select {
        case <-ctx.Done():
            return
        default:
        }
        
        n, err := r.Read(buf)
        if n > 0 {
            // ... send chunk
        }
        if err != nil {
            // ... handle error
            break
        }
    }
}
```

**Files:** 
- `internal/agent/executor.go:750-785` (method signature and implementation)
- `internal/agent/executor.go:603,610` (callers updated to pass context)

### 3. Context.Background() Used Instead of Propagating Context

**Location:** `internal/manager/manager.go:494`

**Problem:**
`context.Background()` is used when building the manager, which doesn't propagate cancellation from the caller:

```go
// Build and return manager
ctx := context.Background()  // <-- Lost context
return builder.Build(ctx)
```

**Impact:**
- Cannot cancel manager initialization
- Long-running initialization operations cannot be interrupted
- No timeout protection

**Fix:** ✅ FIXED
Added documentation explaining that manager initialization is immediate setup and doesn't need context propagation. The context from `NewConversation()` is properly used for actual operations.

**File:** `internal/manager/manager.go:493-496`

### 4. Missing Context Parameter in streamOutput

**Location:** `internal/agent/executor.go:749`

**Problem:**
The `streamOutput` method doesn't accept a context parameter:

```go
func (e *Executor) streamOutput(r io.Reader, stream string, chunks chan<- OutputChunk) {
    // ... reads from r without checking context
}
```

**Impact:**
- Cannot cancel reading operations
- No way to propagate cancellation to stream readers

**Fix:** ✅ FIXED
Added context parameter to `streamOutput` method and checks for cancellation during read operations.

**File:** `internal/agent/executor.go:750-785`

## Root Causes

1. **Pattern Missing:** No standard pattern for goroutines to check context cancellation ✅ FIXED
2. **Missing Context Propagation:** Some functions don't accept context parameter when they should ✅ FIXED
3. **Background Context:** Use of `context.Background()` loses cancellation propagation ✅ FIXED (documented)
4. **Long-Running Operations:** Goroutines in loops don't monitor context state ✅ FIXED

## Additional Fixes

### Context Checks in Compression Loops

**Location:** `internal/history/compress/hybrid.go:104-109`

Added context cancellation checks in the greedy selection loop of HybridCompressor:

```go
for _, cm := range classified {
    // Check context cancellation periodically during selection
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ... rest of loop
}
```

**Location:** `internal/history/compress/llm.go:128-133`

Added context cancellation checks in the summarization loop of LLMSummarizer:

```go
for i := 0; i < len(summarizableMsgs); i += s.chunkSize {
    // Check context cancellation during summarization loop
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ... rest of loop
}
```

## Recommendations

### Immediate Fixes (Priority 1) ✅ COMPLETED

1. ✅ Fix manager.go goroutine to check context cancellation
2. ✅ Add context parameter to `streamOutput` method
3. ✅ Update goroutines in executor to pass and check context
4. ✅ Add context checks in compression loops

### Pattern Enforcement (Priority 2)

1. Document context management patterns in CONTRIBUTING.md
2. Add linter rules to catch missing context cancellation checks
3. Create helper functions for common patterns

### Testing (Priority 3)

1. ✅ Add tests that verify goroutines terminate on context cancellation (existing test: `TestHybridCompressor_Compress_ContextCancellation`)
2. Test for goroutine leaks using runtime tracking
3. Add integration tests for cancellation propagation

## Related Issues

- Event emitter may block goroutines if channels are full
- File I/O operations don't respect context cancellation
- Long-running LLM calls need timeout protection

## References

- [Go Context Best Practices](https://go.dev/blog/context)
- [Effective Go - Context](https://go.dev/doc/effective_go#context)
- Internal PRD: Context Management Guidelines (to be created)

