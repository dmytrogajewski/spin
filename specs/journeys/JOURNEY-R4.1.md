# Journey R-4.1: Stale-Read File Tracker

**Roadmap Item**: R-4.1
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 4
**Status**: In Progress

## Context

When the agent reads a file and later edits it, the file may have been modified externally (by the user's editor, a build tool, or another process). Writing based on stale content silently overwrites those changes. A file tracker records when each file was last read and asserts freshness before any write.

## User Journey

### Persona
Developer using Spin while actively editing files in their editor.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Read | Agent reads `main.go` | File content returned | Content returned, read timestamp recorded |
| External edit | User edits `main.go` in VS Code | No tracking | Modification tracked via `os.Stat().ModTime()` |
| Write | Agent writes to `main.go` | Silent overwrite of user changes | Error: "file modified since last read; re-read first" |
| Re-read | Agent re-reads `main.go` | N/A | Fresh timestamp recorded, write succeeds |

### Success Criteria
- `FileTracker` records read timestamps per file path.
- `AssertFresh()` detects external modifications via `ModTime()` comparison.
- 50ms tolerance window for filesystem timestamp granularity.
- Writes to files never read return a descriptive error.
- Thread-safe via `sync.RWMutex`.
- `ReadFileTool` calls `RecordRead()` after successful read.
- `WriteFileTool` calls `AssertFresh()` before write.

## Technical Design

### Package Location
- `internal/tools/file_tracker.go` — `FileTracker` with `RecordRead()` and `AssertFresh()`.

### Types
```go
// FileTracker tracks file read timestamps to detect stale reads.
type FileTracker struct {
    mu    sync.RWMutex
    reads map[string]time.Time // path → last read time
}

func NewFileTracker() *FileTracker
func (ft *FileTracker) RecordRead(path string) error
func (ft *FileTracker) AssertFresh(path string) error
```

### Constants
- `ModTimeTolerance = 50 * time.Millisecond` — tolerance for filesystem timestamp granularity.

### Integration
- `ReadFileTool` gains an optional `*FileTracker` field; calls `RecordRead()` after successful read.
- `WriteFileTool` gains an optional `*FileTracker` field; calls `AssertFresh()` before write.
- Both tools accept tracker via constructor option.

### Error Messages
- No prior read: `"file not previously read: %s; read the file before editing"`.
- Stale read: `"file modified since last read: %s; re-read the file first"`.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestFileTracker_RecordAndAssertFresh` | "fresh check skipped" | Record read, assert fresh immediately passes |
| `TestFileTracker_FailsAfterModification` | "stale not detected" | Record read, modify file, assert fresh fails |
| `TestFileTracker_FailsWithoutPriorRead` | "missing read allowed" | Assert fresh without prior read fails |
| `TestFileTracker_ConcurrentAccess` | "race condition" | Concurrent RecordRead and AssertFresh don't panic |
| `TestFileTracker_ToleranceWindow` | "tolerance ignored" | Modification within tolerance passes |

## Implementation

**Status**: Complete

### Files Created
- `internal/tools/file_tracker.go` — `FileTracker` with `RecordRead()`, `AssertFresh()`, 50ms tolerance, `sync.RWMutex`.
- `internal/tools/file_tracker_test.go` — 8 tests covering all scenarios.

### Files Modified
- `internal/tools/read_file.go` — added `tracker *FileTracker` field, `SetTracker()`, calls `RecordRead()` after successful read.
- `internal/tools/write_file.go` — added `tracker *FileTracker` field, `SetTracker()`, calls `AssertFresh()` before write.
- `internal/agent/executor/builtin.go` — `RegisterTools()` creates shared `FileTracker` and injects into both tools.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-4.1 marked Done.
