# JOURNEY-replace-atomic-counter: Replace tui.atomicCounter with sync/atomic.Int64

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 1.4 — Replace `tui.atomicCounter` with `sync/atomic.Int64`
- Cluster: 19 (SPEC.md) | LIST.md finding 63

## 1. Journey

When **a developer maintaining the TUI mapper code** I want to **use stdlib `sync/atomic.Int64` instead of a custom mutex-based counter** so I can **reduce code complexity, improve performance with lock-free operations, and follow Go stdlib idioms**.

## 2. CJM

`internal/tui/mapper.go` defines a custom `atomicCounter` struct with `sync.Mutex`-based increment. The stdlib `sync/atomic.Int64` provides the same functionality with better performance (lock-free CAS) and zero custom code.

### Phase 1: Replace

**Actions:**
1. Add `sync/atomic` import.
2. Replace `var eventIDCounter = &atomicCounter{}` with `var eventIDCounter atomic.Int64`.
3. Remove `atomicCounter` type and its `Add` method.

**Success Signal:** `atomic.Int64.Add(1)` is a drop-in replacement — same signature, same semantics.

### Phase 2: Verification

**Actions:** `go build`, `go test`, `make lint` — all clean.

### North Star Summary

The custom `atomicCounter` type is replaced by stdlib `sync/atomic.Int64`, eliminating 14 lines of custom code with a lock-free, idiomatic alternative.

## 3. Tests

### TC-01: existing TUI mapper tests pass

**Given** `atomicCounter` is replaced with `atomic.Int64`.
**When** `go test ./internal/tui/...` is run.
**Then** all tests pass.

## Implementation

- Modified: `internal/tui/mapper.go` — replaced `atomicCounter` with `atomic.Int64`
