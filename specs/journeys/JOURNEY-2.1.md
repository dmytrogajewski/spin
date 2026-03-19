# JOURNEY-2.1: Undo & Snapshot System

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 2.1 |
| Title | Wire Undo Service and Snapshot Middleware |
| User Story | As a developer, each agent turn creates a git-based snapshot so destructive changes can be rolled back, and file operations go through a unified undo Service. |
| Paper Section | Persistence layer — operation log + git-based snapshots |
| Roadmap Item | JOURNEY-2.1: Undo & Snapshot System (22 functions) |
| Depends on | JOURNEY-1.3 (hooks for lifecycle triggers) |

## Phases

### Phase 1: Discovery
- `undo.Service` wraps `OperationLog` + `SnapshotManager` — fully implemented
- `SnapshotManager` uses shadow bare git repo for full working-tree snapshots
- `snapshot.Middleware` implements `harness.Middleware` — calls `TakeSnapshot()` after execution
- All have unit tests. Never wired into production.
- `builtin.go` creates bare `OperationLog` directly — should go through `Service`

### Phase 2: Integration
- Create `SnapshotManager` + init in `builder.go::Build()`
- Create `Service(opLog, snapshotMgr)` and store on Builder
- Add `snapshot.NewMiddleware(service)` to `buildHarnessMiddlewares()`
- Pass `Service` through to builtin runtime for `OperationLog()` access

### Phase 3: Verification
- Existing unit tests pass
- `make lint` confirms 22 functions reachable

## Implementation

### Files Modified
- `internal/conversation/builder.go` — Added `createUndoService()`, passed `undo.Service` to `buildHarnessMiddlewares()`, added `snapshotmw` and `undo` imports, snapshot middleware appended to middleware chain

### Files Not Modified (existing tests sufficient)
- `internal/undo/service.go`, `internal/undo/snapshot.go` — already fully implemented
- `internal/agent/middleware/snapshot/snapshot.go` — already fully implemented
