# JOURNEY-create-syncmap: Create internal/syncmap.Map[K, V]

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 3.1 — Create `internal/syncmap.Map[K, V]`
- Cluster: 1 (SPEC.md) | LIST.md findings 7, 28, 37, 39, 47, 52, 54, 65

## 1. Journey

When **a developer implementing a thread-safe registry or store** I want to **use a generic `syncmap.Map[K, V]`** so I can **avoid re-implementing the same map+mutex boilerplate in every package**.

## 2. CJM

Seven packages independently implement `map[K]V` + `sync.RWMutex`:
- `internal/conversation/manager.go` — `map[string]*Conversation`
- `internal/tools/registry.go` — `map[string]Tool`
- `internal/mcp/registry_manager.go` — `map[string]Registry`
- `internal/ace/playbook/playbook.go` — `map[string]*bullet.Bullet`
- `internal/ace/refine/archive.go` — `map[string]*ArchivedBullet`
- `internal/ace/adapter/adapter.go` — `map[string]*Session`
- `internal/commands/commands.go` — `map[string]Command`

All share: Get, Set, Delete, Range, Len patterns with RWMutex protection.

### Phase 1: Create syncmap package

**Actions:**
1. Create `internal/syncmap/map.go` with generic `Map[K comparable, V any]`.
2. API: `New`, `Set`, `Get`, `GetOrCreate`, `Delete`, `Range`, `Len`, `Keys`, `Clear`, `Close`.
3. `Close` accepts optional cleanup function for resource disposal.
4. Write comprehensive unit tests including concurrent access.
5. Write benchmarks.

**Success Signal:** All tests pass with `-race`, benchmarks show comparable performance to raw map+mutex.

### North Star Summary

A single generic `syncmap.Map[K, V]` replaces 7+ hand-rolled concurrent map implementations.

## 3. Tests

### TC-01: Set and Get basic operations
### TC-02: Get returns zero value and false for missing key
### TC-03: Delete removes key
### TC-04: Len returns correct count
### TC-05: Keys returns all keys
### TC-06: Range iterates all entries
### TC-07: Range stops early when callback returns false
### TC-08: GetOrCreate creates on miss
### TC-09: GetOrCreate returns existing on hit
### TC-10: Clear removes all entries
### TC-11: Close is idempotent
### TC-12: Close calls cleanup function
### TC-13: Operations after Close return zero values
### TC-14: Concurrent Set/Get/Delete with race detector
### TC-15: Concurrent GetOrCreate with race detector

## Implementation

- Created: `internal/syncmap/map.go` — generic `Map[K, V]` with `New`, `Set`, `SetIfAbsent`, `SetIfPresent`, `Get`, `GetOrCreate`, `Delete`, `Pop`, `Values`, `Range`, `Len`, `Keys`, `Clear`, `Close`
- Created: `internal/syncmap/map_test.go` — 27 unit tests including concurrent access with race detector
- Created: `internal/syncmap/bench_test.go` — benchmarks for Set, Get, GetOrCreate, Range
