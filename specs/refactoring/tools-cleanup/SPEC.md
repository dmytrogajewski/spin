# SPEC: internal/tools Package Cleanup

**Package**: `internal/tools`
**Size**: 35 source files, ~5100 LOC production, ~7700 LOC test
**Tools**: 22 tool implementations

## Problem Statement

The `internal/tools` package is the largest in the codebase and has accumulated
structural debt from rapid feature development. While each tool individually
works, the package suffers from copy-paste duplication, inconsistent patterns,
unnecessary reflection, and dead exports.

## Findings

### F-1: ScratchpadTool / MemoryTool Copy-Paste Duplication

Both tools implement the `memory.Store` interface wrapper pattern. They share:

| Method | Duplication Level | Only Difference |
|--------|------------------|-----------------|
| `executePut` | 95% identical | success message text |
| `executeGet` | 90% identical | MemoryTool adds CreatedAt/UpdatedAt |
| `executeDelete` | 85% identical | error message label |
| `Schema()` | 90% identical | ScratchpadTool adds pin/unpin/clear enum |
| `Execute()` switch | 85% identical | ScratchpadTool adds 3 extra cases |

The `store_helpers.go` file already extracted `storeList()` and `storeSearch()`
using the `memory.Store` interface. The remaining `put/get/delete` methods
were not extracted.

**Impact**: Bug fixes must be applied in two places. New Store features
require parallel changes. Test duplication (~80% structural overlap).

### F-2: Shell Detection Logic Duplicated in Same File

`shell_command.go` contains the same shell metacharacter check in two places:
- `isShellCmd()` method (line 229-238) — used by `buildCommand()`
- `detectShell()` fallback (line 430-438) — identical inline copy

**Impact**: Low but gratuitous. The fallback should call `isShellCmd()`.

### F-3: URL Validation Functions Duplicated

- `isValidURL()` in `web_fetch.go:110-112`
- `isScreenshotURL()` in `web_screenshot.go:119-121`

These are identical functions with different names.

**Impact**: Low. Confusing for readers. Easy fix.

### F-4: GetContextTool Uses Reflection

`get_context.go:57-84` uses `reflect.ValueOf` and `MethodByName("String")`
to call `String()` on an `any`-typed field. This is done to avoid an import
cycle with the `agent` package.

The standard library provides `fmt.Stringer` for exactly this purpose.

**Impact**: Reflection is fragile (panics on nil, no compile-time safety).
`fmt.Stringer` gives compile-time guarantees with zero reflection overhead.

### F-5: Path Resolution Duplication

```go
if !filepath.IsAbs(path) && t.workDir != "" {
    path = filepath.Join(t.workDir, path)
}
```

Appears in: `read_file.go:63`, `write_file.go:73`, `edit_file.go:144`,
`list_directory.go:64`.

**Impact**: Medium. Risk of divergence if security checks are added later.

### F-6: Inconsistent ToolResult Construction

Four patterns for constructing error results:
1. `NewToolError(err)` — standard
2. `ErrToResultf(format, err)` — standard with formatting
3. `ToolResult{Success: false, Error: "msg"}` — inline literal
4. `gitSuccessResult()`/`gitErrorResult()` — private git-only helpers

**Impact**: Cognitive noise. Readers must decode which pattern is in use.

### F-7: Registry Validation Methods on Wrong Receiver

Seven methods on `*Registry` that never access `r.tools`:
`validateParams`, `validateRequiredParams`, `validateParameterTypes`,
`validateParameter`, `createUnknownParameterError`, `validateTypeFromJSON`,
`validateEnumFromJSON`.

**Impact**: False coupling. Suggests these modify Registry state when they don't.

### F-8: Exported Sentinel Errors Used Only Internally

11 sentinel errors in `errors.go` are exported (`Err...`) but never referenced
outside the `tools` package:
`ErrPathParameterRequired`, `ErrContentParameterRequired`,
`ErrOperationParameterRequired`, `ErrUnknownOperation`,
`ErrKeyParameterRequiredForPut`, `ErrValueParameterRequiredForPut`,
`ErrKeyParameterRequiredForGet`, `ErrKeyParameterRequiredForDelete`,
`ErrQueryParameterRequiredForSearch`, `ErrKeyParameterRequiredForPin`,
`ErrKeyParameterRequiredForUnpin`.

**Impact**: API surface pollution. Exported symbols promise stability.
Internal-only errors should be unexported.

### F-9: Process Tools Could Be Unified

`list_processes.go`, `kill_process.go`, `get_process_output.go` are three
separate tools that each wrap a single method on the `TaskManager` interface.
The `git_operation_tool.go` demonstrates the multi-operation pattern already.

**Impact**: Low. 3 tiny tools vs 1 multi-op tool is a style preference.
Keeping them separate is acceptable but increases tool count for the LLM.

## Non-Issues (Assessed and Dismissed)

### Tool Interface Boilerplate
Each tool implements 4 methods (Name, Description, Schema, Execute).
This is idiomatic Go interface satisfaction. A `BaseTool` embedding would
save lines but obscure intent. **No action needed.**

### Package Size (35 files)
The package has a clear single responsibility: define tools. Shared types
(`Tool`, `ToolParameters`, `ToolResult`) bind the files. Sub-packages would
create awkward cross-imports. **No action needed yet. Monitor at 40+ files.**

### SetTracker/SetOperationLog Pattern
The mutable setter pattern (`readTool.SetTracker(tracker)`) is used because
tools are created with `BuiltinTools` and later configured. This is a
known trade-off for the factory/registration design. **No action needed.**
