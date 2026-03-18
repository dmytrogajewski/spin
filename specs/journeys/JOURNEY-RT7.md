# JOURNEY-RT7 — Unexport Internal-Only Sentinel Errors

**Status**: Done
**Roadmap**: specs/refactoring/tools-cleanup/ROADMAP.md -> R-T7

## User Journey

All sentinel errors in the tools package are exported but never referenced from outside `internal/tools/`. Unexporting them reduces API surface and signals they are internal implementation details.

## DoD

- [x] 17 errors in `errors.go` unexported.
- [x] 8 errors in `git_operation_tool.go` unexported.
- [x] 1 unused error removed (`errContentParameterRequired`).
- [x] 12 files updated with new names.
- [x] `make lint` 0 issues.

## Implementation

### Files Modified
- `internal/tools/errors.go` — all vars renamed `Err` -> `err`; removed unused `errContentParameterRequired`.
- `internal/tools/git_operation_tool.go` — all 8 sentinel vars renamed `Err` -> `err`.
- `internal/tools/shell_command.go` — references updated.
- `internal/tools/scratchpad_tool.go` — references updated.
- `internal/tools/memory_tool.go` — references updated.
- `internal/tools/read_file.go` — references updated.
- `internal/tools/write_file.go` — references updated.
- `internal/tools/edit_file.go` — references updated.
- `internal/tools/list_directory.go` — references updated.
- `internal/tools/file_search.go` — references updated.
- `internal/tools/get_process_output.go` — references updated.
- `internal/tools/kill_process.go` — references updated.
- `internal/tools/store_helpers.go` — references updated.
