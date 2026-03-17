# JOURNEY-R6.2 — Self-Healing Session Index

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-6.2

## User Journey

Developer opens Spin in a project with 200 sessions. Today, listing requires loading all 200 JSON files (~2s). After this change, listing reads one 40KB index file (~10ms). If the index is corrupted or missing, it transparently rebuilds from session metadata files.

## Phases

### Phase 1: Index Entry and Persistence
- `IndexEntry` struct: `{ID, Title, MessageCount, LastModified, WorkDir}`
- `SessionIndex` loads/saves `sessions-index.json` via `storage.AtomicWriteFile()`
- Thread-safe via `sync.RWMutex`

### Phase 2: CRUD Operations
- `Update(entry)` upserts an entry by ID
- `Remove(id)` deletes an entry by ID
- `List(workDir)` returns entries filtered by workDir, sorted by `LastModified` desc

### Phase 3: Rebuild and Self-Healing
- `Rebuild(scanner)` takes a `MetadataScanner` interface to scan session metadata files
- On `NewSessionIndex()`, if index file is missing or corrupted, auto-rebuild
- Emits `EventSessionIndexRebuilt` event via optional callback

### Phase 4: Persistence
- `Save()` persists index to disk atomically
- Called after every mutation (`Update`, `Remove`, `Rebuild`)

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Index file corruption | List returns stale data | Auto-rebuild from metadata files |
| Concurrent writes | Data race | sync.RWMutex |
| Large number of sessions | Index file grows | ~200 bytes per entry, 200 sessions = 40KB |
| Rebuild performance | Slow on many sessions | One-time cost, subsequent reads are O(1) |

## Design Decisions

1. **Atomic writes**: Uses `storage.AtomicWriteFile()` for crash-safe persistence.
2. **MetadataScanner interface**: Decouples index from storage implementation for testability.
3. **Sort on read**: `List()` sorts by LastModified desc. Small N makes this negligible.
4. **Optional rebuild callback**: Avoids hard dependency on event system.
5. **Self-healing on load**: If JSON parse fails, rebuilds transparently.

## DoD

- [x] `internal/session/index.go` — `Index` with `Update()`, `List()`, `Remove()`, `Rebuild()`
- [x] Index entries: `{ID, Title, MessageCount, LastModified, WorkDir}`
- [x] `List()` filters by workDir, sorts by LastModified desc
- [x] `Rebuild()` scans metadata files via `MetadataScanner` interface
- [x] Auto-rebuild on corruption or missing index
- [x] Atomic writes via `storage.AtomicWriteFile()`
- [x] New event: `EventSessionIndexRebuilt` (35)
- [x] Unit tests (≥90% coverage)
- [x] `go vet` and `make lint` clean

## Implementation

### Files Created
- `internal/session/index.go` — `SessionIndex` core implementation
- `internal/session/index_test.go` — Unit tests

### Files Modified
- `internal/events/event.go` — Added `EventSessionIndexRebuilt` (35)
- `internal/events/event_test.go` — Added test case
- `specs/refactoring/opendev-gaps/ROADMAP.md` — R-6.2 marked Done
