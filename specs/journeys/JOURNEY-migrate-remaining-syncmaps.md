# JOURNEY-migrate-remaining-syncmaps: Migrate Remaining Concurrent Maps to syncmap.Map

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 3.3 — Migrate Remaining Concurrent Maps
- Cluster: 1 (SPEC.md) | LIST.md findings 7, 37, 39, 47, 52, 54, 65

## 1. Journey

When **a developer maintaining concurrent map implementations** I want to **use the shared `syncmap.Map` across all remaining packages** so I can **eliminate 6 hand-rolled map+mutex implementations and reduce concurrency bug surface area**.

## 2. CJM

Six packages independently implement `map[K]V` + `sync.RWMutex`:
- `internal/tools/registry.go` — `map[string]Tool`
- `internal/mcp/registry_manager.go` — `map[string]Registry`
- `internal/ace/playbook/playbook.go` — `map[string]*bullet.Bullet`
- `internal/ace/refine/archive.go` — `map[string]*ArchivedBullet`
- `internal/ace/adapter/adapter.go` — `map[string]*Session`
- `internal/commands/commands.go` — `map[string]Command` (package-level globals)

### Phase 1: Add helper methods to syncmap

**Actions:**
1. Add `SetIfAbsent(key, value) bool` for atomic check-and-insert patterns.
2. Add `SetIfPresent(key, value) bool` for atomic check-and-update patterns.
3. Add `Values() []V` for collecting all values.
4. Write tests for new methods.

### Phase 2: Migrate all 6 sites

**Actions:**
1. Migrate each site replacing `map + sync.RWMutex` with `*syncmap.Map`.
2. Use `SetIfAbsent` for register-if-absent patterns (tools.Register, mcp.Register, playbook.Add).
3. Use `SetIfPresent` for update-if-exists patterns (playbook.Update).
4. Use `Pop` for atomic get-and-delete patterns (mcp.Unregister, adapter.EndSession).
5. Use `Range` for iteration patterns.
6. Preserve all public APIs unchanged.

**Success Signal:** All tests pass with `-race`, `make lint` zero issues.

### North Star Summary

All 6 remaining concurrent map implementations replaced by `syncmap.Map`, completing the Phase 3 dedup effort.

## 3. Tests

### TC-01: SetIfAbsent returns true on miss, false on hit
### TC-02: SetIfPresent returns true on hit, false on miss
### TC-03: Values returns all values
### TC-04: Existing tests pass for all 6 migrated packages
### TC-05: Race detector passes on all migrated packages

## Implementation

- Modified: `internal/syncmap/map.go` — added `SetIfAbsent`, `SetIfPresent`, `Values` methods
- Modified: `internal/syncmap/map_test.go` — added 8 tests for new methods
- Modified: `internal/tools/registry.go` — replaced `map[string]Tool` + `sync.RWMutex` with `*syncmap.Map[string, Tool]`
- Modified: `internal/mcp/registry_manager.go` — replaced `map[string]Registry` + `sync.RWMutex` with `*syncmap.Map[string, Registry]`
- Modified: `internal/ace/playbook/playbook.go` — replaced `map[string]*bullet.Bullet` + `sync.RWMutex` with `*syncmap.Map`
- Modified: `internal/ace/playbook/storage.go` — updated to use syncmap API
- Modified: `internal/ace/playbook/search.go` — updated to use syncmap Range
- Modified: `internal/ace/playbook/snapshot.go` — updated to use syncmap API, removed `statsLocked`
- Modified: `internal/ace/refine/archive.go` — replaced `map[string]*ArchivedBullet` + `sync.RWMutex` with `*syncmap.Map`
- Modified: `internal/ace/adapter/adapter.go` — replaced `map[string]*Session` + `sync.RWMutex` with `*syncmap.Map`
- Modified: `internal/commands/commands.go` — replaced package-level `map[string]Command` + `sync.RWMutex` with `*syncmap.Map`
