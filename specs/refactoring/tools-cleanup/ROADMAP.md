# Roadmap: internal/tools Package Cleanup

**Spec**: [SPEC.md](./SPEC.md)
**Package**: `internal/tools`
**Created**: 2026-03-17

---

## Overview

Nine refactoring items, ordered by value and dependency. Each maps to a single
user journey document. Items are progressive — each delivers testable value
independently and the codebase compiles green after every step.

---

## R-T1: Eliminate Shell Detection Duplication
**Status**: Done
**Spec Section**: F-2
**Journey**: [JOURNEY-RT1.md](../../journeys/JOURNEY-RT1.md)

### Description
The `detectShell()` fallback in `shell_command.go` duplicates the
`isShellCmd()` method's shell metacharacter checks inline. Make the fallback
call `isShellCmd()` instead of repeating the same 8 condition checks.

### Definition of Ready
- [x] `shell_command.go` contains both `isShellCmd()` (line 224) and inline
  duplicate in `detectShell()` (line 430)
- [x] Existing tests in `shell_command_test.go` cover `detect_shell` operation

### Definition of Done
- [x] `detectShell()` fallback calls `t.isShellCmd(command)` instead of
  inline checks
- [x] Inline duplicate conditions removed (lines 430-438)
- [x] All existing `shell_command_test.go` tests pass unchanged
- [x] No new test needed (behavioral equivalence)

### Estimated Scope
~10 lines changed in 1 file.

---

## R-T2: Deduplicate URL Validation
**Status**: Done
**Spec Section**: F-3
**Journey**: [JOURNEY-RT2.md](../../journeys/JOURNEY-RT2.md)

### Description
`isValidURL()` in `web_fetch.go` and `isScreenshotURL()` in
`web_screenshot.go` are identical functions with different names. Delete
`isScreenshotURL` and have `web_screenshot.go` call `isValidURL`.

### Definition of Ready
- [x] Both functions exist and have identical implementations
- [x] `isScreenshotURL` is called only from `web_screenshot.go:71`

### Definition of Done
- [x] `isScreenshotURL` function deleted from `web_screenshot.go`
- [x] `web_screenshot.go:71` calls `isValidURL(rawURL)` instead
- [x] All `web_screenshot_test.go` and `web_fetch_test.go` tests pass
- [x] No new test needed (behavioral equivalence)

### Estimated Scope
~5 lines changed across 1 file.

---

## R-T3: Replace Reflection with fmt.Stringer in GetContextTool
**Status**: Done
**Spec Section**: F-4
**Journey**: [JOURNEY-RT3.md](../../journeys/JOURNEY-RT3.md)

### Description
`GetContextTool` stores its dependency as `any` and calls `String()` via
reflection. Change the field type to `fmt.Stringer`, which provides
compile-time safety and eliminates reflection.

### Definition of Ready
- [x] `get_context.go` uses `reflect.ValueOf` and `MethodByName("String")`
- [x] The concrete type (`agent.Environment`) implements `String() string`
- [x] Callers pass `agent.Environment` via `NewGetContextTool(env)`

### Definition of Done
- [x] `GetContextTool.context` field type changed from `any` to `fmt.Stringer`
- [x] `NewGetContextTool` parameter type changed from `any` to `fmt.Stringer`
- [x] `Execute()` calls `t.stringer.String()` directly — no reflection
- [x] `reflect` import removed from `get_context.go`
- [x] All callers compile (check `NewDefaultRegistry`, `executor/builtin.go`)
- [x] Existing `get_context_test.go` tests pass
- [x] Nil Stringer returns error result (existing TestGetContextTool_NilContext)

### Estimated Scope
~30 lines changed across 2-3 files.

---

## R-T4: Extract Path Resolution Helper
**Status**: Done
**Spec Section**: F-5
**Journey**: [JOURNEY-RT4.md](../../journeys/JOURNEY-RT4.md)

### Description
Four tools duplicate the same 3-line path resolution logic. Extract a
`resolvePath(path, workDir string) string` helper function and have all
four tools call it.

### Definition of Ready
- [x] Pattern identified in `read_file.go:63`, `write_file.go:73`,
  `edit_file.go:144`, `list_directory.go:64`
- [x] All four implementations are semantically identical

### Definition of Done
- [x] `resolvePath(path, workDir string) string` function added in `path.go`
- [x] All four tools use `resolvePath` instead of inline logic
- [x] Unit test for `resolvePath`: absolute, relative, empty workDir, empty path, both empty
- [x] All existing tool tests pass unchanged

### Estimated Scope
~25 lines changed across 5 files (4 tools + 1 new helper).

---

## R-T5: Consolidate ToolResult Construction Patterns
**Status**: Pending
**Spec Section**: F-6
**Journey**: [JOURNEY-RT5.md](../../journeys/JOURNEY-RT5.md)

### Description
Eliminate the private `gitSuccessResult()`/`gitErrorResult()` helpers and
inline `ToolResult{Success: false, Error: "msg"}` literals. Replace all with
the standard `NewToolResult()`/`NewToolError()`/`ErrToResultf()` constructors.

### Definition of Ready
- [x] `gitSuccessResult` and `gitErrorResult` in `git_operation_tool.go:26-39`
- [x] Inline `ToolResult{...}` literals scattered across `shell_command.go`,
  `file_search.go`, `apply_patch.go`, `kill_process.go`, `get_process_output.go`

### Definition of Done
- [ ] `gitSuccessResult` and `gitErrorResult` deleted
- [ ] All git operation handlers use `NewToolResult()` / `NewToolError()`
- [ ] Inline `ToolResult{Success: false, Error: "msg"}` replaced with
  `NewToolError(errors.New("msg"))` or `ErrToResultf`
- [ ] All tests pass unchanged (behavioral equivalence)

### Estimated Scope
~60 lines changed across 4-5 files.

---

## R-T6: Move Registry Validation to Package Functions
**Status**: Pending
**Spec Section**: F-7
**Journey**: [JOURNEY-RT6.md](../../journeys/JOURNEY-RT6.md)

### Description
Seven validation methods on `*Registry` never access `r.tools`. Convert them
to package-level functions. This clarifies that validation is a pure operation
with no Registry side effects.

### Definition of Ready
- [x] Methods identified: `validateParams`, `validateRequiredParams`,
  `validateParameterTypes`, `validateParameter`, `createUnknownParameterError`,
  `validateTypeFromJSON`, `validateEnumFromJSON`
- [x] None access `r.tools` or any Registry field

### Definition of Done
- [ ] All 7 methods converted to package-level functions (remove receiver)
- [ ] `Registry.Execute` and `Registry.validateParams` call the functions
- [ ] All `registry_test.go` tests pass unchanged
- [ ] No exported API changes (all 7 were unexported)

### Estimated Scope
~40 lines changed in 1 file.

---

## R-T7: Unexport Internal-Only Sentinel Errors
**Status**: Pending
**Spec Section**: F-8
**Journey**: [JOURNEY-RT7.md](../../journeys/JOURNEY-RT7.md)

### Description
11 sentinel errors in `errors.go` are exported but never used outside the
`tools` package. Unexport them to reduce API surface.

### Definition of Ready
- [x] Verified via grep: none of these errors are referenced outside
  `internal/tools/`
- [x] List: `ErrPathParameterRequired`, `ErrContentParameterRequired`,
  `ErrOperationParameterRequired`, `ErrUnknownOperation`,
  `ErrKeyParameterRequiredForPut`, `ErrValueParameterRequiredForPut`,
  `ErrKeyParameterRequiredForGet`, `ErrKeyParameterRequiredForDelete`,
  `ErrQueryParameterRequiredForSearch`, `ErrKeyParameterRequiredForPin`,
  `ErrKeyParameterRequiredForUnpin`

### Definition of Done
- [ ] All 11 errors renamed from `ErrX` to `errX` (unexported)
- [ ] All internal references updated
- [ ] Package compiles, all tests pass
- [ ] `go vet ./...` clean for the package

### Estimated Scope
~30 lines changed across 5-6 files.

---

## R-T8: Unify ScratchpadTool / MemoryTool CRUD Operations
**Status**: Pending
**Spec Section**: F-1
**Journey**: [JOURNEY-RT8.md](../../journeys/JOURNEY-RT8.md)

### Description
Extract the duplicated `executePut`, `executeGet`, `executeDelete` logic into
`store_helpers.go`, extending the existing pattern of `storeList`/`storeSearch`.
Both tools delegate to shared functions parameterized by label string and
an optional entry formatter.

This is the highest-value refactoring but touches the most code. It is
sequenced last so that prior items (R-T5 result consistency, R-T7 error
unexporting) are done first, reducing merge surface.

### Definition of Ready
- [x] `store_helpers.go` already contains `storeList` and `storeSearch`
- [x] Both tools implement `memory.Store` wrapper pattern
- [x] Duplication analysis complete (see SPEC F-1 table)

### Definition of Done
- [ ] `storePut(ctx, store, params, label) (ToolResult, error)` added to
  `store_helpers.go`
- [ ] `storeGet(ctx, store, params, label, formatEntry) (ToolResult, error)`
  added with optional entry formatter callback for timestamps
- [ ] `storeDelete(ctx, store, params, label) (ToolResult, error)` added
- [ ] `ScratchpadTool.executePut/Get/Delete` delegate to shared functions
- [ ] `MemoryTool.executePut/Get/Delete` delegate to shared functions
- [ ] MemoryTool passes a formatter that includes CreatedAt/UpdatedAt
- [ ] ScratchpadTool retains its own `executePin/Unpin/Clear` (unique)
- [ ] All scratchpad_tool_test.go and memory_tool_test.go tests pass
- [ ] Net LOC reduction: ~80-100 lines

### Estimated Scope
~150 lines changed across 3 files.

---

## R-T9: Consolidate Process Tools (Optional)
**Status**: Pending (Optional)
**Spec Section**: F-9
**Journey**: [JOURNEY-RT9.md](../../journeys/JOURNEY-RT9.md)

### Description
Merge `list_processes`, `kill_process`, `get_process_output` into a single
`ProcessTool` with an `operation` parameter, following the pattern established
by `GitOperationTool` and `ShellCommandTool`.

This is marked optional because the current 3-tool design is functional and
the consolidation provides modest value (reduces LLM tool count by 2).

### Definition of Ready
- [x] Three tools identified, each wrapping one TaskManager method
- [x] GitOperationTool provides the multi-operation pattern to follow

### Definition of Done
- [ ] `ProcessTool` struct with `operation` enum: `list`, `kill`, `get_output`
- [ ] Old 3 tool files deleted or consolidated
- [ ] `task_manager.go` types remain unchanged
- [ ] All `process_tools_test.go` tests adapted and pass
- [ ] Registration updated in `BuiltinTools` or factory functions

### Estimated Scope
~120 lines changed across 4 files.

---

## Execution Order

```
R-T1 (shell dedup)        ─┐
R-T2 (URL dedup)           │── Trivial fixes, independent
R-T3 (reflection removal)  │
                           ─┘
R-T4 (path helper)         ── Introduces shared helper pattern
R-T5 (result consistency)  ── Normalizes patterns before big refactor
R-T6 (registry validation) ── Pure mechanical move
R-T7 (unexport errors)     ── Must verify no external usage first
R-T8 (store CRUD unify)    ── Largest change, benefits from prior cleanup
R-T9 (process consolidate) ── Optional, independent
```

Items R-T1 through R-T3 are independent and can be done in parallel.
R-T4 through R-T7 are independent of each other but benefit from R-T1-T3
being done first (cleaner diff).
R-T8 depends on R-T5 and R-T7 conceptually.
R-T9 is independent and optional.

---

## Success Criteria

After all items (R-T1 through R-T8):
- `go build ./internal/tools/...` succeeds
- `go test ./internal/tools/...` passes
- `go vet ./internal/tools/...` clean
- Net LOC reduction: ~150-200 lines
- Zero behavioral changes (all refactoring is mechanical)
- No new exported symbols
- Reflection removed from production code
