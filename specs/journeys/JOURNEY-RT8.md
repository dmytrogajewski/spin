# JOURNEY-RT8 — Unify ScratchpadTool / MemoryTool CRUD Operations

**Status**: Done
**Roadmap**: specs/refactoring/tools-cleanup/ROADMAP.md -> R-T8

## User Journey

ScratchpadTool and MemoryTool duplicate nearly identical `executePut`, `executeGet`, `executeDelete` logic. After this refactoring, shared `storePut`, `storeGet`, `storeDelete` functions in `store_helpers.go` handle the common logic, parameterized by label and an optional entry formatter.

## Design Decisions

1. **EntryFormatter callback**: `storeGet` accepts an optional `entryFormatter` to add tool-specific fields (CreatedAt/UpdatedAt for MemoryTool).
2. **Label suffix**: `storePut` and `storeDelete` use `labelSuffix` for messages.
3. **ScratchpadTool retains Pin/Unpin/Clear**: These are unique and not shared.

## DoD

- [x] `storePut`, `storeGet`, `storeDelete` added to `store_helpers.go`.
- [x] Both tools delegate to shared functions.
- [x] All tests pass unchanged.
- [x] `make lint` 0 issues.

## Implementation

### Files Modified
- `internal/tools/store_helpers.go` — added `storePut`, `storeGet`, `storeDelete`, `labelSuffix`, `entryFormatter` type.
- `internal/tools/scratchpad_tool.go` — `executePut/Get/Delete` now delegate to shared helpers; removed unused `strings` import.
- `internal/tools/memory_tool.go` — `executePut/Get/Delete` now delegate to shared helpers; added `memoryEntryFormatter`; removed unused `errors` import.
