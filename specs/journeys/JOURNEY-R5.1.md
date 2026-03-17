# Journey R-5.1: In-Memory Operation Log

**Roadmap Item**: R-5.1
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 5
**Status**: Done

## Context

When the agent edits a file incorrectly, the user has no built-in way to undo the change. An in-memory operation log records every file create/modify/delete with before-content, enabling single-step undo that restores the file to its previous state.

## User Journey

### Persona
Developer using Spin who needs to undo agent mistakes.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Edit | Agent edits `config.yaml` incorrectly | File changed, no rollback | Operation recorded with before-content |
| Undo | User triggers undo | No undo capability | Last operation reversed, file restored |
| Create | Agent creates `new.go` | No tracking | Create operation recorded; undo deletes file |
| Limit | 51st operation recorded | N/A | Oldest operation evicted (FIFO, 50-entry cap) |

### Success Criteria
- `OperationType` enum: Create, Modify, Delete.
- `Operation` struct with path, before-content, type, timestamp.
- `OperationLog` with `Record()`, `Undo()`, 50-entry FIFO cap.
- `Undo()` reverses: create→delete, modify→restore, delete→restore.
- Thread-safe via `sync.Mutex`.
- `WriteFileTool` records before mutation (create or modify).
- `EditFileTool` records before mutation (always modify).
- New event: `EventUndoRecorded`.

## Technical Design

### Package Location
- `internal/undo/` — operation log package.

### Types
```go
type OperationType int

const (
    OpCreate OperationType = iota
    OpModify
    OpDelete
)

type Operation struct {
    Type          OperationType
    Path          string
    BeforeContent []byte
    Timestamp     time.Time
}

type OperationLog struct {
    mu      sync.Mutex
    entries []Operation
}
```

### Constants
- `MaxEntries = 50` — FIFO eviction cap.

### Integration
- `WriteFileTool` and `EditFileTool` gain optional `*OperationLog` via `SetOperationLog()`.
- Before mutation, tool reads existing content (if file exists → Modify, else → Create).
- `OperationLog` created in `BuiltinRuntime.RegisterTools()` alongside `FileTracker`.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestOperationLog_RecordAndUndo_Create` | "create not reversed" | Undo deletes created file |
| `TestOperationLog_RecordAndUndo_Modify` | "modify not reversed" | Undo restores before-content |
| `TestOperationLog_EmptyLog_ReturnsError` | "empty undo succeeds" | Undo on empty log returns error |
| `TestOperationLog_FIFOEviction` | "cap not enforced" | 51st entry evicts oldest |
| `TestOperationLog_ConcurrentAccess` | "race condition" | Concurrent Record/Undo safe |
| `TestOperationLog_MultipleUndos` | "multi-undo fails" | Sequential undos work |
| `TestOperationLog_Len` | "count wrong" | Len reports correct count |

## Implementation

**Status**: Done

### Files Created
- `internal/undo/types.go` — `OperationType` enum, `Operation` struct with package doc.
- `internal/undo/log.go` — `OperationLog` with `Record()`, `Undo()`, `Len()`, `RecordFileChange()`, 50-entry FIFO cap.
- `internal/undo/log_test.go` — 12 tests covering create/modify/delete undo, FIFO eviction, concurrent access, multiple undos, convenience method.

### Files Modified
- `internal/tools/write_file.go` — Added `opLog *undo.OperationLog` field, `SetOperationLog()` method, `RecordFileChange()` call before write.
- `internal/tools/edit_file.go` — Added `opLog *undo.OperationLog` field, `SetOperationLog()` method, `RecordFileChange()` call before edit.
- `internal/agent/executor/builtin.go` — Creates `undo.NewOperationLog()` in `RegisterTools()`, injects into `WriteFileTool` and `EditFileTool`.
- `internal/events/event.go` — Added `EventUndoRecorded` to event type enum.
- `internal/events/event_test.go` — Added `EventUndoRecorded` test case.
