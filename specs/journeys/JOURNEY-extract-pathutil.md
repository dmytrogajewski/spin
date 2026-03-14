# JOURNEY-extract-pathutil: Extract Home Directory Expansion to internal/pathutil

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 2.3 — Extract Home Directory Expansion to `internal/pathutil`
- Cluster: 8 (SPEC.md) | LIST.md findings 30, 22

## 1. Journey

When **a developer handling user-configured paths** I want to **call a single, tested `pathutil.ExpandHome` function** so I can **avoid maintaining three identical `~` expansion patterns across conversation, memory, and storage packages**.

## 2. CJM

Three production packages independently implement `~` prefix expansion:
- `internal/conversation/events.go:resolveSessionDir` — handles `~` and `~/`
- `internal/memory/persistent.go:NewPersistentStore` — handles `~/`
- `internal/storage/store.go:NewFileStore` — handles `~/`

### Phase 1: Create pathutil package

**Actions:**
1. Create `internal/pathutil/expand.go` with `ExpandHome(path string) (string, error)`.
2. Write comprehensive unit tests.

**Success Signal:** All edge cases covered: `~`, `~/foo`, `/absolute`, `relative`, empty.

### Phase 2: Migrate call-sites

**Actions:**
1. Replace 3 inline expansion patterns with `pathutil.ExpandHome`.

**Success Signal:** All tests pass, no duplicate patterns remain.

### Phase 3: Verification

**Actions:** `go vet`, `make lint`, `go test -race ./internal/pathutil/... ./internal/conversation/... ./internal/memory/... ./internal/storage/...`

**Success Signal:** Zero issues, all tests green.

### North Star Summary

A single `pathutil.ExpandHome` function serves all tilde expansion needs. No duplicate patterns remain.

## 3. Tests

### TC-01: `~` expands to home directory
### TC-02: `~/foo` expands to home/foo
### TC-03: `/absolute` passes through unchanged
### TC-04: `relative` passes through unchanged
### TC-05: empty string passes through unchanged
### TC-06: `~user` passes through unchanged (not supported)

## Implementation

- Created: `internal/pathutil/expand.go` — `ExpandHome`
- Created: `internal/pathutil/expand_test.go` — comprehensive unit tests
- Modified: `internal/conversation/events.go` — uses `pathutil.ExpandHome`
- Modified: `internal/memory/persistent.go` — uses `pathutil.ExpandHome`
- Modified: `internal/storage/store.go` — uses `pathutil.ExpandHome`
- Fixed: `internal/conversation/conversation.go` — added `taskMu` to protect `taskMode` from race conditions
