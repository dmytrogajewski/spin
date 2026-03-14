# JOURNEY-migrate-manager-syncmap: Migrate conversation.Manager to syncmap.Map

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 3.2 — Migrate `conversation.Manager` to `syncmap.Map`
- Cluster: 1 (SPEC.md) | LIST.md finding 28

## 1. Journey

When **a developer maintaining the conversation Manager** I want to **use the generic `syncmap.Map` instead of hand-rolled map+mutex boilerplate** so I can **reduce code duplication and eliminate concurrency bugs**.

## 2. CJM

`conversation.Manager` maintained its own `map[string]*Conversation` protected by `sync.RWMutex`, duplicating the exact pattern that `syncmap.Map` was built to replace.

### Phase 1: Replace internal map with syncmap.Map

**Actions:**
1. Replace `map[string]*Conversation` + `sync.RWMutex` with `*syncmap.Map[string, *Conversation]`.
2. Keep `sync.Mutex` (`createMu`) for serializing conversation creation in `GetOrCreate`.
3. Add `Pop` method to `syncmap.Map` for atomic get-and-delete (needed by `Remove`).
4. Update all Manager methods to use syncmap API.
5. Fix test data races from shared `err` variables in parallel subtests.

**Success Signal:** All tests pass with `-race`, `make lint` zero issues.

### North Star Summary

`conversation.Manager` uses `syncmap.Map` as its first consumer migration, validating the generic API and eliminating hand-rolled concurrency boilerplate.

## 3. Tests

### TC-01: Pop returns value and removes key
### TC-02: Pop returns false for missing key
### TC-03: Existing Manager tests pass unchanged (public API preserved)
### TC-04: Race detector passes on all conversation tests
### TC-05: Race detector passes on concurrent Manager operations

## Implementation

- Modified: `internal/syncmap/map.go` — added `Pop` method for atomic get-and-delete
- Modified: `internal/syncmap/map_test.go` — added `TestMap_Pop_Existing`, `TestMap_Pop_Missing`
- Modified: `internal/conversation/manager.go` — replaced `map[string]*Conversation` + `sync.RWMutex` with `*syncmap.Map[string, *Conversation]`; `Remove` uses `Pop` for atomic removal
- Modified: `internal/conversation/manager_test.go` — fixed shared `err` variable races in parallel subtests
