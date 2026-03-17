# JOURNEY-R5.2 — Shadow Git Snapshots

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-5.2

## User Journey

Agent runs `npm install` (modifies `package-lock.json` as a side-effect), then edits `src/index.ts`. User decides to undo everything to the pre-install state. `/undo --step 3` restores the entire working tree to step 3's snapshot.

## Phases

### Phase 1: Initialization
- SnapshotManager detects the project work directory
- Creates a shadow bare git repo at `~/.spin/snapshots/<project-hash>/`
- Syncs `.gitignore` to shadow repo's `info/exclude`

### Phase 2: Snapshot Capture
- After each harness dispatch phase, middleware calls `TakeSnapshot()`
- SnapshotManager runs `git add -A && git write-tree` with `GIT_DIR` pointing to shadow repo
- Returns the tree hash as an opaque snapshot identifier

### Phase 3: Restore
- User invokes undo-to-step
- SnapshotManager runs `git diff-tree` to find changed files between current tree and target snapshot
- Restores each changed file from the snapshot tree

### Phase 4: Cleanup
- On session end or periodically, `Cleanup()` runs `git gc --prune=7.days.ago`
- Keeps snapshot history manageable

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| git not on PATH | All snapshot ops fail | Return descriptive error, degrade gracefully |
| Large repos slow snapshot | Snapshot takes >1s | Best-effort async, don't block agent loop |
| Shadow repo corruption | Restore fails | Re-initialize shadow repo on error |
| Project hash collision | Wrong snapshot dir | Use crypto/sha256 of absolute workdir path |

## Design Decisions

1. **Shell out to git binary**: No Go git libraries needed. `exec.CommandContext` with timeouts.
2. **Shadow bare repo**: Avoids interfering with user's actual `.git/`. Uses `GIT_DIR` + `GIT_WORK_TREE` env vars.
3. **Project hash**: SHA256 of absolute workdir path, truncated to 16 hex chars.
4. **UndoService**: Combines `OperationLog` (fine-grained file undo) + `SnapshotManager` (full-tree undo).
5. **SnapshotMiddleware**: Moved to `internal/agent/middleware/snapshot/` to avoid import cycle (undo → harness → tools → undo). Uses `Snapshotter` interface for decoupling.
6. **Graceful degradation**: If git is not available or snapshot fails, log warning and continue.

## DoD

- [x] `internal/undo/snapshot.go` — `SnapshotManager` with `Init()`, `Snapshot()`, `Restore()`, `Cleanup()`
- [x] Shadow bare repo at `~/.spin/snapshots/<project-hash>/`
- [x] `Snapshot()` runs `git add -A && git write-tree` with `GIT_DIR` pointing to shadow repo
- [x] `Restore()` runs `git diff-tree` to find changed files, restores via `git cat-file`
- [x] `.gitignore` synced from real repo to shadow repo's `info/exclude`
- [x] `Cleanup()` runs `git gc --prune=7.days.ago`
- [x] `internal/undo/service.go` — `Service` combining `OperationLog` + `SnapshotManager`
- [x] `internal/agent/middleware/snapshot/` — `Middleware` implementing `harness.Middleware`
- [x] New event: `EventSnapshotTaken` (34)
- [x] Unit tests with temp git repos (24 tests total)
- [x] `go vet` and `make lint` clean (0 issues)

## Implementation

### Files Created
- `internal/undo/snapshot.go` — `SnapshotManager`, `ProjectHash()`, shadow bare repo management
- `internal/undo/service.go` — `Service` combining `OperationLog` + `SnapshotManager`
- `internal/undo/snapshot_test.go` — 15 tests: init, snapshot, restore (files/new/deleted), gitignore sync, cleanup, hash
- `internal/undo/service_test.go` — 7 tests: undo last, take snapshot, undo to step, nil handling
- `internal/agent/middleware/snapshot/snapshot.go` — `Middleware` with `Snapshotter` interface
- `internal/agent/middleware/snapshot/snapshot_test.go` — 3 tests: after execution, error handling, no-op

### Files Modified
- `internal/events/event.go` — added `EventSnapshotTaken` (34)
- `internal/events/event_test.go` — added test case for `snapshot_taken`
