# JOURNEY-CTX-7.1 — File Tools Check ctx.Err()

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 7.1
**Spec**: specs/ctx/SPEC.md -> CTX-033, CTX-034, CTX-044

## User Journey

File tools (read, write, edit, list, patch) discard their context parameter. After this change, they check `ctx.Err()` before I/O as a cancellation gate, and `patchapply.Apply` accepts ctx.

## Implementation

### Files Modified
- `internal/tools/read_file.go` — `Execute` renamed `_` to `ctx`, added `ctx.Err()` check.
- `internal/tools/write_file.go` — same pattern.
- `internal/tools/edit_file.go` — same pattern.
- `internal/tools/list_directory.go` — same pattern.
- `internal/tools/apply_patch.go` — `Execute` and `applyPatch` accept ctx, pass to `applier.Apply(ctx, ...)`.
- `internal/patchapply/applier.go` — `Apply(ctx, patch)` accepts ctx, checks `ctx.Err()` at entry.
- `internal/patchapply/applier_test.go` — all `Apply` calls pass `t.Context()`.
- `cmd/spin/apply_patch.go` — `Apply` call passes `cmd.Context()`.
