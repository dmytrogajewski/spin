# Reusability & Dedup Roadmap

Spec: [SPEC.md](SPEC.md) | Evidence: [LIST.md](LIST.md)

---

## Overview

Progressive decomposition of SPEC.md clusters into independently testable, shippable steps.
Each step is a single user journey delivering value on its own.

**Guiding principles:**
- Every step compiles and passes `go vet ./...` + existing tests
- No step introduces regressions — run `go test ./...` after each
- Steps are ordered by dependency: foundational utilities first, then consumers
- Each step is scoped to 1-3 files changed (micro-PRs)

---

## Phase 1 — Low-Hanging Fruit (Zero-Dependency Fixes)

### 1.1 Collapse Duplicate Sentinel Errors ✅ DONE

**Cluster:** 3 (SPEC.md) | **Priority:** P0 | **Effort:** Low

**Description:**
Eliminate ~55 numbered duplicate sentinel error variables (`ErrFoo2`, `ErrFoo3`, …) across 16 packages. Each group shares an identical error message — the numbered suffixes exist only to satisfy the `err113` linter. Fix by keeping one sentinel per message and reusing the canonical variable at each call-site.

**Journey:** [JOURNEY-dedup-sentinel-errors.md](../journeys/JOURNEY-dedup-sentinel-errors.md)

**DoR (Definition of Ready):**
- LIST.md findings 1, 16, 27, 31, 36, 46, 49, 53, 72 enumerated
- Codebase scan confirms 55 numbered error vars across 16 packages

**DoD (Definition of Done):**
- [x] All `Err.*[2-9]` variables removed from non-test `.go` files
- [x] Each call-site uses the single canonical sentinel, wrapped with context where needed
- [x] `go vet ./...` passes
- [x] `go test ./...` passes (no test references broken sentinels)
- [x] `grep -rn 'Err.*[2-9] =' --include='*.go' | grep -v _test.go` returns 0 matches

**Packages to touch (in order):**
1. `internal/storage` (4 vars)
2. `internal/patchapply` (8 vars)
3. `internal/conversation` (6 vars)
4. `internal/tools` (5 vars)
5. `internal/llm/factory` (2 vars)
6. `internal/ace/curator` (5 vars)
7. `internal/ace/playbook` (2 vars)
8. `internal/ace/adapter` (3 vars)
9. `internal/git` (10 vars)
10. `internal/ui/blocks` (6 vars)
11. `cmd/spin/config.go` (4 vars)
12. `internal/memory` (3 vars)
13. `internal/protocol/acp` (3 vars)
14. `internal/agent/executor` (2 vars)
15. `internal/auth` (1 var)
16. `internal/shell` (1 var)

**Test plan:**
- `go test ./internal/storage/... ./internal/patchapply/... ./internal/conversation/... ./internal/tools/... ./internal/llm/... ./internal/ace/... ./internal/git/... ./internal/ui/... ./cmd/spin/... ./internal/memory/... ./internal/protocol/... ./internal/agent/... ./internal/auth/... ./internal/shell/...`
- Verify no test file references a removed variable name

---

### 1.2 Unify ErrToolNotFound Triplicate ✅ DONE

**Cluster:** 6 (SPEC.md) | **Priority:** P0 | **Effort:** Low

**Description:**
Three packages define `ErrToolNotFound = errors.New("tool not found")`: `internal/tools`, `internal/mcp`, `internal/agent`. Keep `tools.ErrToolNotFound` as canonical, update the other two to import it.

**Journey:** [JOURNEY-unify-err-tool-not-found.md](../journeys/JOURNEY-unify-err-tool-not-found.md)

**DoR:**
- LIST.md findings 40, 42 confirm 3 definitions

**DoD:**
- [x] `mcp.ErrToolNotFound` removed; `mcp` imports `tools.ErrToolNotFound`
- [x] `agent.ErrToolNotFound` removed; `agent` imports `tools.ErrToolNotFound`
- [x] All `errors.Is(err, <old>)` checks updated
- [x] `go test ./internal/tools/... ./internal/mcp/... ./internal/agent/...` passes

**Test plan:**
- Grep for `mcp.ErrToolNotFound` and `agent.ErrToolNotFound` — expect 0 non-test matches
- Run tests for all three packages

**Key files:** `internal/mcp/errors.go`, `internal/agent/tool_runtime.go`, `internal/mcp/errors_test.go`

---

### 1.3 Deduplicate Task Validate() and MaxAllowedTokens ✅ DONE

**Cluster:** 14 (SPEC.md) | **Priority:** P1 | **Effort:** Low

**Description:**
4 task types (`Compact`, `Planning`, `Regular`, `Review`) have identical `Validate()` bodies. Extract shared `validateMaxTokens(maxTokens int) error`. Remove duplicate private `maxAllowedTokens` constant in `regular.go`.

**Journey:** [JOURNEY-dedup-task-validate.md](../journeys/JOURNEY-dedup-task-validate.md)

**DoR:**
- LIST.md findings 61, 62

**DoD:**
- [x] New function `validateMaxTokens` in `internal/task/validate.go`
- [x] All 4 task `Validate()` methods call `validateMaxTokens`
- [x] `regular.go` private `maxAllowedTokens` removed; uses `MaxAllowedTokens` from `constants.go`
- [x] `go test ./internal/task/...` passes

**Test plan:**
- Unit test for `validateMaxTokens` with edge cases (0, negative, MaxAllowedTokens, MaxAllowedTokens+1)
- Existing task tests pass unchanged

**Key files:** `internal/task/validate.go`, `internal/task/validate_test.go`

---

### 1.4 Replace `tui.atomicCounter` with `sync/atomic.Int64` ✅ DONE

**Cluster:** 19 (SPEC.md) | **Priority:** P3 | **Effort:** Low

**Description:**
`internal/tui/mapper.go` defines a custom `atomicCounter` struct with mutex-based increment. Replace with stdlib `atomic.Int64`.

**Journey:** [JOURNEY-replace-atomic-counter.md](../journeys/JOURNEY-replace-atomic-counter.md)

**DoR:**
- LIST.md finding 63

**DoD:**
- [x] `atomicCounter` type removed from mapper.go
- [x] All usages replaced with `atomic.Int64`
- [x] `go test ./internal/tui/...` passes

**Test plan:**
- Run TUI mapper tests

**Key files:** `internal/tui/mapper.go`

---

### 1.5 Replace `boolString` with `strconv.FormatBool` ✅ DONE

**Cluster:** 19 (SPEC.md) | **Priority:** P3 | **Effort:** Low

**Description:**
`internal/conversation/builder.go` defines a custom `boolString` function. Replace with `strconv.FormatBool()`.

**Journey:** [JOURNEY-replace-boolstring.md](../journeys/JOURNEY-replace-boolstring.md)

**DoR:**
- LIST.md finding 29

**DoD:**
- [x] `boolString` function removed
- [x] Call-sites use `strconv.FormatBool()`
- [x] `go test ./internal/conversation/...` passes

**Test plan:**
- Run conversation tests

**Key files:** `internal/conversation/builder.go`

---

### 1.6 Fix Dead Code in `configureMaxTokens` ✅ DONE

**Cluster:** 19 (SPEC.md) | **Priority:** P3 | **Effort:** Low

**Description:**
`cmd/spin/tui.go:configureMaxTokens` iterates provider models but the loop body only breaks — never uses the found model. Either complete the implementation (set maxTokens from model capabilities) or remove the dead iteration. Prefer complete implementation.

**Journey:** [JOURNEY-fix-configure-max-tokens.md](../journeys/JOURNEY-fix-configure-max-tokens.md)

**DoR:**
- LIST.md finding 78

**DoD:**
- [x] Dead loop removed; function simplified to set default max tokens
- [x] `go build ./cmd/spin/...` succeeds
- [x] `go test ./cmd/spin/...` passes

**Test plan:**
- Manual verification of TUI startup with model flag

**Key files:** `cmd/spin/tui.go`

---

## Phase 2 — Shared Utility Extraction

### 2.1 Extract Cosine Similarity to `internal/mathutil` ✅ DONE

**Cluster:** 4 (SPEC.md) | **Priority:** P1 | **Effort:** Low

**Description:**
Extract the 5 identical `cosineSimilarity(a, b []float32) float64` implementations into a single `internal/mathutil` package. Remove custom `sqrt` from HNSW retriever. Fix the buggy test-only variant that omits square root.

**Journey:** [JOURNEY-extract-mathutil.md](../journeys/JOURNEY-extract-mathutil.md)

**DoR:**
- LIST.md finding 51, 58
- 5 implementations confirmed (4 production + 1 test)

**DoD:**
- [x] New package `internal/mathutil` with `CosineSimilarity`, `DotProduct`, `Magnitude`
- [x] Unit tests in `internal/mathutil/vector_test.go` (zero vectors, identical vectors, orthogonal, unit, large)
- [x] All 4 production call-sites updated to call `mathutil.CosineSimilarity`
- [x] Custom `sqrt` removed from `hnsw_retriever.go`
- [x] Test helper in `ollama_embedder_test.go` updated
- [x] `go test ./internal/mathutil/... ./internal/ace/...` passes

**Test plan:**
- `mathutil` unit tests with known cosine similarity values
- ACE integration tests pass

**Key files:** `internal/mathutil/vector.go`, `internal/mathutil/vector_test.go`

---

### 2.2 Extract `cleanJSONResponse` to `internal/llmutil` ✅ DONE

**Cluster:** 16 (SPEC.md) | **Priority:** P2 | **Effort:** Low

**Description:**
Extract identical `cleanJSONResponse` from `ace/curator` and `ace/reflector` into `internal/llmutil.CleanJSONResponse`.

**Journey:** [JOURNEY-extract-llmutil.md](../journeys/JOURNEY-extract-llmutil.md)

**DoR:**
- LIST.md finding 50

**DoD:**
- [x] New package `internal/llmutil` with `CleanJSONResponse(raw string) string`
- [x] Unit tests covering: bare JSON, ` ```json` wrapper, ` ``` ` wrapper, nested backticks, empty
- [x] Both call-sites updated
- [x] `go test ./internal/llmutil/... ./internal/ace/curator/... ./internal/ace/reflector/...` passes

**Test plan:**
- `llmutil` unit tests
- Curator and reflector tests pass

**Key files:** `internal/llmutil/json.go`, `internal/llmutil/json_test.go`

---

### 2.3 Extract Home Directory Expansion to `internal/pathutil` ✅ DONE

**Cluster:** 8 (SPEC.md) | **Priority:** P3 | **Effort:** Low

**Description:**
Extract `~` prefix expansion into `internal/pathutil.ExpandHome(path string) (string, error)`. Update `conversation/events.go:resolveSessionDir`, `memory/persistent.go`, and `storage/store.go` to use it.

**Journey:** [JOURNEY-extract-pathutil.md](../journeys/JOURNEY-extract-pathutil.md)

**DoR:**
- LIST.md finding 30, related to finding 22

**DoD:**
- [x] New package `internal/pathutil` with `ExpandHome`
- [x] Unit tests: `~`, `~/foo`, `/absolute`, `relative`, empty
- [x] 3 call-sites updated (conversation, memory, storage)
- [x] `go test ./internal/pathutil/... ./internal/conversation/... ./internal/memory/... ./internal/storage/...` passes

**Test plan:**
- `pathutil` unit tests
- Conversation, memory, and storage tests pass

**Key files:** `internal/pathutil/expand.go`, `internal/pathutil/expand_test.go`

---

### 2.4 Extract Atomic File Write to `internal/storage.AtomicWriteFile` ✅ DONE

**Cluster:** 2 (SPEC.md) | **Priority:** P2 | **Effort:** Low

**Description:**
Extract the temp-file + `os.Rename` pattern as a standalone function in `internal/storage`. Update `config.MCPConfigStore.writeConfig`, `memory.PersistentStore`, and `ace/playbook.Save` to use it. `storage.FileStore` already has this internally — just expose it.

**Journey:** [JOURNEY-extract-atomic-write.md](../journeys/JOURNEY-extract-atomic-write.md)

**DoR:**
- LIST.md findings 1, 10, 22, 48

**DoD:**
- [x] New exported function `storage.AtomicWriteFile(path string, data []byte, perm os.FileMode) error`
- [x] Unit test in `storage/atomic_test.go` (success, dir not exist, rename failure)
- [x] `FileStore.Save` refactored to use `AtomicWriteFile` internally
- [x] 3 external call-sites updated
- [x] `go test ./internal/storage/... ./internal/config/... ./internal/memory/... ./internal/ace/playbook/...` passes

**Test plan:**
- `storage` unit tests for `AtomicWriteFile`
- All consuming package tests pass

**Key files:** `internal/storage/atomic.go`, `internal/storage/atomic_test.go`

---

## Phase 3 — Generic Concurrent Map

### 3.1 Create `internal/syncmap.Map[K, V]` ✅ DONE

**Cluster:** 1 (SPEC.md) | **Priority:** P1 | **Effort:** Medium

**Description:**
Create a generic thread-safe map with lifecycle support. This is the foundation for deduplicating 8+ concurrent map implementations.

**Journey:** [JOURNEY-create-syncmap.md](../journeys/JOURNEY-create-syncmap.md)

**DoR:**
- LIST.md findings 7, 28, 37, 39, 47, 52, 54, 65

**DoD:**
- [x] New package `internal/syncmap`
- [x] `Map[K comparable, V any]` with: `Set`, `Get`, `GetOrCreate`, `Delete`, `Range`, `Len`, `Keys`, `Close`, `Clear`
- [x] `Close` accepts optional `func(V)` for resource cleanup
- [x] Comprehensive unit tests (concurrent read/write, GetOrCreate race, Close idempotency)
- [x] Benchmark tests comparing to raw `map` + `sync.RWMutex`
- [x] `go test -race ./internal/syncmap/...` passes

**Test plan:**
- Unit tests with `-race` flag
- Benchmark: `BenchmarkMapSet`, `BenchmarkMapGet`, `BenchmarkMapGetOrCreate`

**Key files:** `internal/syncmap/map.go`, `internal/syncmap/map_test.go`, `internal/syncmap/bench_test.go`

---

### 3.2 Migrate `conversation.Manager` to `syncmap.Map` ✅ DONE

**Cluster:** 1 (SPEC.md) | **Priority:** P1 | **Effort:** Medium

**Description:**
Replace the hand-rolled `map[string]*Conversation` + `sync.RWMutex` in `conversation.Manager` with `syncmap.Map[string, *Conversation]`. This is the first consumer migration — validates the API.

**Journey:** [JOURNEY-migrate-manager-syncmap.md](../journeys/JOURNEY-migrate-manager-syncmap.md)

**DoR:**
- Step 3.1 completed
- LIST.md finding 28

**DoD:**
- [x] `Manager` uses `syncmap.Map` internally
- [x] `Manager` public API unchanged (no breaking changes)
- [x] All `conversation` tests pass
- [x] `go test -race ./internal/conversation/...` passes

**Test plan:**
- Existing `conversation` test suite
- Race detector enabled

**Key files:** `internal/conversation/manager.go`, `internal/syncmap/map.go` (added `Pop` method)

---

### 3.3 Migrate Remaining Concurrent Maps ✅ DONE

**Cluster:** 1 (SPEC.md) | **Priority:** P2 | **Effort:** Medium

**Description:**
Migrate remaining 6 concurrent map instances to `syncmap.Map`:
1. `tools.Registry`
2. `mcp.DefaultRegistryManager`
3. `ace/playbook.Playbook`
4. `ace/refine.Archive`
5. `ace/adapter.sessions`
6. `commands` registry

**Journey:** [JOURNEY-migrate-remaining-syncmaps.md](../journeys/JOURNEY-migrate-remaining-syncmaps.md)

**DoR:**
- Steps 3.1, 3.2 completed

**DoD:**
- [x] All 6 sites migrated
- [x] No public API changes
- [x] `go test -race ./internal/tools/... ./internal/mcp/... ./internal/ace/... ./internal/commands/...` passes

**Test plan:**
- Run tests for each package with race detector

**Key files:** `internal/syncmap/map.go` (added `SetIfAbsent`, `SetIfPresent`, `Values`), `internal/tools/registry.go`, `internal/mcp/registry_manager.go`, `internal/ace/playbook/playbook.go`, `internal/ace/refine/archive.go`, `internal/ace/adapter/adapter.go`, `internal/commands/commands.go`

---

## Phase 4 — TTL Cache and Worker Pool

### 4.1 Extract `internal/cache.TTLCache[K, V]`

**Cluster:** 5 (SPEC.md) | **Priority:** P2 | **Effort:** Medium

**Description:**
Create a generic TTL cache unifying `security.MemoryPolicyStore` and `agent.CommandCache`. Support TTL eviction, optional max-size, and optional background janitor.

**DoR:**
- LIST.md findings 19, 43
- Step 3.1 completed (can use `syncmap.Map` internally or standalone)

**DoD:**
- [ ] New package `internal/cache`
- [ ] `TTLCache[K comparable, V any]` with: `Get`, `Set`, `Delete`, `Len`, `Clear`, `Close`
- [ ] Options: `WithMaxSize(n)`, `WithJanitorInterval(d)`
- [ ] Unit tests: TTL expiration, max-size eviction, janitor cleanup, concurrent access
- [ ] `go test -race ./internal/cache/...` passes

**Test plan:**
- Unit tests with time manipulation (short TTLs)
- Race detector

---

### 4.2 Migrate `security.MemoryPolicyStore` and `agent.CommandCache`

**Cluster:** 5 (SPEC.md) | **Priority:** P2 | **Effort:** Medium

**Description:**
Replace both cache implementations with `cache.TTLCache`.

**DoR:**
- Step 4.1 completed

**DoD:**
- [ ] `MemoryPolicyStore` uses `cache.TTLCache` internally
- [ ] `CommandCache` uses `cache.TTLCache` internally
- [ ] Public APIs unchanged
- [ ] `go test ./internal/security/... ./internal/agent/...` passes

**Test plan:**
- Existing security and agent tests pass

---

### 4.3 Extract `internal/workerpool.Run[In, Out]`

**Cluster:** 7 (SPEC.md) | **Priority:** P2 | **Effort:** Low-Medium

**Description:**
Extract the fan-out/fan-in worker pool from `ace/delta/batch.go` and `ace/curator/parallel.go`.

**DoR:**
- LIST.md finding 57

**DoD:**
- [ ] New package `internal/workerpool`
- [ ] `func Run[In, Out any](ctx context.Context, workers int, inputs []In, fn func(context.Context, In) (Out, error)) ([]Out, error)`
- [ ] Unit tests: single worker, multi-worker, context cancellation, error propagation
- [ ] Both call-sites migrated
- [ ] `go test -race ./internal/workerpool/... ./internal/ace/delta/... ./internal/ace/curator/...` passes

**Test plan:**
- `workerpool` unit tests with race detector
- ACE delta and curator tests pass

---

## Phase 5 — cmd/spin Wiring Dedup

### 5.1 Unify Flag Helpers and JSON Output

**Cluster:** 11, 9 (SPEC.md) | **Priority:** P1 | **Effort:** Low

**Description:**
1. Replace 5 `flagX` helpers with single `flagString(cmd, name)`.
2. Make `outputJSON` call `printJSON(os.Stdout, data)`.
3. Extract `loadMCPConfigStore(cmd)` helper for 8 repeated sites.

**DoR:**
- LIST.md findings 66, 73, 75

**DoD:**
- [ ] Single `flagString` function replaces 5 helpers
- [ ] `outputJSON` calls `printJSON`
- [ ] `loadMCPConfigStore` extracts repeated 4-line pattern
- [ ] `go build ./cmd/spin/...` succeeds
- [ ] `go test ./cmd/spin/...` passes (if tests exist)

**Test plan:**
- Build verification
- Existing cmd tests

---

### 5.2 Unify Event Processing and Signal Handling

**Cluster:** 11 (SPEC.md) | **Priority:** P1 | **Effort:** Medium

**Description:**
1. Unify `processEvent` / `processExecEvent` into single function.
2. Unify `startEventLoop` / `startExecEventLoop`.
3. Unify `setupSignalHandling` / `setupACPServerSignalHandling`.

**DoR:**
- LIST.md findings 67, 68, 71

**DoD:**
- [ ] Single `processEvent` used by both TUI and exec
- [ ] Single `startEventLoop` returning `chan struct{}` used by both
- [ ] Single `setupSignalHandling(cancel, onSignal func())` with optional callback
- [ ] `go build ./cmd/spin/...` succeeds

**Test plan:**
- Build verification
- Manual TUI + exec mode smoke test

---

### 5.3 Unify Conversation Creation (TUI calls `buildConversation`)

**Cluster:** 11 (SPEC.md) | **Priority:** P1 | **Effort:** Medium

**Description:**
1. Make `createConversationForTUI` call `buildConversation` instead of inlining the same logic.
2. Replace inline session resolution with `resolveSessionID(storage, workDir, "tui")`.
3. Unify `buildProviderForACP` and `buildProvider`.

**DoR:**
- LIST.md findings 69, 70, 77

**DoD:**
- [ ] `createConversationForTUI` calls `buildConversation`
- [ ] Inline session logic replaced with `resolveSessionID`
- [ ] Single `buildProviderWithExtra` function for both exec and ACP
- [ ] `go build ./cmd/spin/...` succeeds
- [ ] TUI, exec, and ACP modes all start correctly

**Test plan:**
- Build verification
- Manual smoke test: `spin tui`, `spin exec "hello"`, `spin acp`

---

### 5.4 Dedup MCP Registry Filtering

**Cluster:** 11 (SPEC.md) | **Priority:** P2 | **Effort:** Low

**Description:**
Replace inline filter loop in `runMCPListTools` with call to existing `filterServersByRegistry`.

**DoR:**
- LIST.md finding 74

**DoD:**
- [ ] Inline loop removed, `filterServersByRegistry` called instead
- [ ] `go build ./cmd/spin/...` succeeds

**Test plan:**
- Build verification

---

## Phase 6 — Detection / Events Cleanup

### 6.1 Remove Duplicate Detection Interfaces

**Cluster:** 10 (SPEC.md) | **Priority:** P2 | **Effort:** Medium

**Description:**
Have `internal/detection` use `events.EventEmitter` and `message.Message` directly instead of defining its own interfaces. Remove `eventEmitterAdapter` from `agent/loop.go`.

**DoR:**
- LIST.md findings 4, 45

**DoD:**
- [ ] `detection.EventEmitter` interface removed; uses `events.EventEmitter`
- [ ] `detection.Message` interface removed; uses `message.Message` or direct types
- [ ] `agent.eventEmitterAdapter` removed
- [ ] `go test ./internal/detection/... ./internal/agent/...` passes

**Test plan:**
- Detection and agent tests pass

---

### 6.2 Generic Event Data Accessor

**Cluster:** 9 (SPEC.md) | **Priority:** P2 | **Effort:** Low

**Description:**
Add generic `GetEventData[T any](e Event) (T, bool)` function to `internal/events`. Keep existing 12 accessor methods as deprecated aliases initially.

**DoR:**
- LIST.md finding 5

**DoD:**
- [ ] New `GetEventData[T]` function in `events` package
- [ ] Unit test verifying type assertion behavior
- [ ] At least 3 call-sites migrated as proof of concept
- [ ] `go test ./internal/events/...` passes

**Test plan:**
- Unit tests for `GetEventData` with valid/invalid types

---

## Phase 7 — MCP Config and Approval Handlers

### 7.1 Unify MCP Server Config Types

**Cluster:** 18 (SPEC.md) | **Priority:** P3 | **Effort:** Medium

**Description:**
Merge `config.MCPServer` and `config.MCPServerConfigV2` into a single struct with all necessary serialization tags. Consolidate duplicate validation methods.

**DoR:**
- LIST.md findings 9, 11

**DoD:**
- [ ] Single `MCPServerConfig` struct with json/yaml/mapstructure tags
- [ ] Validation methods consolidated (no duplication)
- [ ] `go test ./internal/config/...` passes

**Test plan:**
- Config loading and validation tests

---

### 7.2 Move Approval Handler Factories to `security` Package

**Cluster:** 19 (SPEC.md) | **Priority:** P3 | **Effort:** Low

**Description:**
Move `createAutoApproveHandler` and `createDenyHandler` from `cmd/spin/approval_handlers.go` to `internal/security` as `security.AutoApproveHandler()` and `security.DenyHandler(reason)`.

**DoR:**
- LIST.md finding 76

**DoD:**
- [ ] Functions moved to `internal/security/handlers.go`
- [ ] `cmd/spin` imports from `security` package
- [ ] `go test ./internal/security/... ./cmd/spin/...` passes

**Test plan:**
- Security and cmd tests pass

---

## Summary

| Phase | Steps | Effort | Value |
|-------|-------|--------|-------|
| 1 — Low-Hanging Fruit | 1.1–1.6 | Low | Remove ~53 dup errors, 3 dup sentinels, dead code |
| 2 — Utility Extraction | 2.1–2.4 | Low | 4 new shared packages, eliminate 10+ duplicates |
| 3 — Concurrent Map | 3.1–3.3 | Medium | Generic `syncmap.Map`, 8 sites deduped |
| 4 — Cache & Pool | 4.1–4.3 | Medium | Generic cache + worker pool, 4 sites deduped |
| 5 — cmd/spin Wiring | 5.1–5.4 | Medium | Major cmd dedup, ~9 findings resolved |
| 6 — Detection Cleanup | 6.1–6.2 | Medium | Remove adapter layer, generic accessors |
| 7 — Config & Handlers | 7.1–7.2 | Medium | Unify MCP config, relocate handlers |

**Total: 19 steps across 7 phases. Each step is independently testable and shippable.**
