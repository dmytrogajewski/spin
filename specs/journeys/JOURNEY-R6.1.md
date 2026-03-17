# JOURNEY-R6.1 — JSONL Transcript Writer

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-6.1

## User Journey

Agent is mid-session when a crash occurs. Today the entire session is lost because the single JSON wasn't flushed. After this change, all messages up to the crash are preserved in the JSONL transcript. Each message is appended as a single JSON line under advisory file lock, so recovery reads can skip corrupted trailing lines gracefully.

## Phases

### Phase 1: TranscriptWriter Core
- `NewTranscriptWriter(path)` creates or opens the JSONL file
- `Append(msg)` serializes one `message.Message` as JSON, appends under `LOCK_EX`
- `ReadAll()` reads all lines, skipping corrupted ones, returns `[]message.Message`
- Thread-safe via `sync.Mutex`

### Phase 2: Graceful Degradation
- Corrupted lines (invalid JSON) are silently skipped on `ReadAll()`
- Scanner uses 10 MB buffer for large tool-call payloads
- Empty file returns empty slice, no error

### Phase 3: Concurrent Safety
- Advisory file locking via `syscall.Flock` on append (exclusive lock)
- `ReadAll()` uses shared lock (`LOCK_SH`)
- Multiple goroutines can append safely

### Phase 4: Close and Cleanup
- `Close()` flushes and closes the underlying file handle
- Safe to call multiple times (idempotent)

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Large tool-call JSON | Line exceeds default scanner buffer | 10 MB scanner buffer |
| Crash mid-write | Trailing partial JSON line | Skip corrupted lines on read |
| Concurrent writers | Data interleaving | Advisory file lock per append |
| Disk full | Append fails | Return error, caller decides |

## Design Decisions

1. **Append-only JSONL**: One message per line, no rewriting. Crash-safe by design.
2. **Advisory locking**: Uses `syscall.Flock` matching existing pattern in `safety/policy_file_store.go`.
3. **Scanner buffer 10 MB**: Tool calls can produce large JSON; default 64 KB is insufficient.
4. **Skip corrupted lines**: `ReadAll()` logs nothing, just skips — caller gets best-effort recovery.
5. **File handle management**: Writer keeps file open for duration, reducing open/close overhead on hot path.
6. **safeFlockFd helper**: Reuse pattern from `safety/policy_file_store.go` (local copy to avoid cross-package dependency).

## DoD

- [x] `internal/session/transcript.go` — `TranscriptWriter` with `Append(msg)`, `ReadAll()`, `Close()`
- [x] File format: one JSON object per line (JSONL)
- [x] Advisory file locking via `syscall.Flock` on append
- [x] Corrupted lines skipped on read
- [x] Scanner buffer: 10 MB max line
- [x] Thread-safe via `sync.Mutex`
- [x] Unit tests (≥90% coverage)
- [x] `go vet` and `make lint` clean

## Implementation

### Files Created
- `internal/session/transcript.go` — `TranscriptWriter` core implementation
- `internal/session/transcript_test.go` — Unit tests

### Files Modified
- `specs/refactoring/opendev-gaps/ROADMAP.md` — R-6.1 marked Done
