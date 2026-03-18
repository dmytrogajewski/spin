# JOURNEY-CTX-4.4 — ACE Background Operations Gain Timeout Boundaries

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 4.4
**Spec**: specs/ctx/SPEC.md -> CTX-021, CTX-022, CTX-029

## User Journey

ACE service spawns background goroutines for async playbook save and refinement with no timeout. These can hang on LLM calls or filesystem issues. After this change, background save gets a 30s timeout, refinement gets a 60s timeout, and batch delta workers check ctx.Done() for early exit.

## Design Decisions

1. **Timeout constants**: `asyncSaveTimeout = 30s`, `refinementTimeout = 60s` — generous for their operations.
2. **context.WithoutCancel + WithTimeout**: Detach from parent cancellation but add own timeout boundary.
3. **Batch workers select on ctx.Done()**: Prevents processing remaining jobs after cancellation.
4. **Playbook.Save gains ctx**: Passes ctx to `AtomicWriteFile` instead of `context.Background()`.

## DoD

- [x] `savePlaybookAfterUpdate(ctx)` async path uses `context.WithTimeout(context.WithoutCancel(ctx), 30s)`.
- [x] `checkGrowthAndRefine` background goroutine uses `context.WithTimeout(context.WithoutCancel(ctx), 60s)`.
- [x] `runBatchWorkers` workers select on `ctx.Done()` for early exit.
- [x] `Playbook.Save(ctx, path)` accepts and passes ctx to `AtomicWriteFile`.
- [x] `SavePlaybook(ctx)` accepts and passes ctx.
- [x] All callers updated (UpdateBullets, GenerateBullets, GenerateBulletsWithReflection, initial save).
- [x] All test callers updated.
- [x] `go vet ./...` clean. `make lint` clean. `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/ace/service.go` — added `asyncSaveTimeout`/`refinementTimeout` constants; `savePlaybookAfterUpdate(ctx)` adds timeout on async path; `SavePlaybook(ctx)` passes ctx; `checkGrowthAndRefine` adds timeout to bgCtx; all callers updated.
- `internal/ace/delta/batch.go` — `runBatchWorkers` worker goroutines select on `ctx.Done()` for early exit.
- `internal/ace/playbook/storage.go` — `Save(ctx, path)` passes ctx to `AtomicWriteFile`.
- `internal/ace/playbook/playbook_test.go` — `pb.Save` calls pass `t.Context()`.
- `internal/ace/playbook/version_test.go` — `pb.Save` call passes `t.Context()`.
- `internal/ace/service_test.go` — `SavePlaybook` call passes `t.Context()`.
