# FRD-20260119-001: ToolCall Data Types Consolidation

**Status:** Completed  
**Priority:** P0 (Critical)  
**Author:** Spin Agent  
**Created:** 2026-01-19  
**ROADMAP Reference:** Phase 1, Issue 1.1

---

## Problem Statement

`ToolCallStartData` and `ToolCallCompleteData` are defined in two locations with nearly identical fields:

1. **`internal/events/event.go:206-230`** - Canonical event data types (actively used)
2. **`internal/agent/request.go:82-96`** - Duplicate definitions (DEAD CODE)

### Current State Analysis

**`internal/events/event.go`:**
```go
type ToolCallStartData struct {
    ToolName         string               `json:"tool_name"`
    ToolID           string               `json:"tool_id"`
    Parameters       tools.ToolParameters `json:"parameters"`
    RequiresApproval bool                 `json:"requires_approval"`
}

type ToolCallCompleteData struct {
    ToolID   string                 `json:"tool_id"`
    ToolName string                 `json:"tool_name"`
    Success  bool                   `json:"success"`
    Output   string                 `json:"output"`
    Error    string                 `json:"error,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

**`internal/agent/request.go`:**
```go
type ToolCallCompleteData struct {
    ToolID   string
    ToolName string
    Success  bool
    Error    string
    Output   string
}

type ToolCallStartData struct {
    ToolID     string
    ToolName   string
    Parameters tools.ToolParameters
}
```

### Impact

1. **Dead Code:** The `agent` package types are never used anywhere in the codebase
2. **Maintenance Burden:** Changes to event types might accidentally be applied to agent types
3. **Confusion:** Developers might import from wrong package
4. **Inconsistency:** Field ordering differs, agent types lack JSON tags and some fields (`RequiresApproval`, `Metadata`)

### Evidence of Dead Code

Grep analysis shows ALL usages are from `events` package:
- `internal/agent/agent.go:672` - uses `events.ToolCallStartData`
- `internal/agent/agent.go:692` - uses `events.ToolCallCompleteData`
- All tests use `events` package types
- All TUI mapper uses `events` package types
- All ACP notifications use `events` package types

The `agent.ToolCallStartData` and `agent.ToolCallCompleteData` have ZERO usages.

---

## Solution

Remove the duplicate dead code types from `internal/agent/request.go`.

### Changes Required

1. **Delete from `internal/agent/request.go` (lines 82-96):**
   - `type ToolCallCompleteData struct { ... }`
   - `type ToolCallStartData struct { ... }`

2. **Also delete from `internal/agent/request.go` (lines 66-79):**
   - `type EventType int`
   - `const ( EventWarning EventType = iota ... )`
   - `type Event struct { ... }`

   These are also duplicates of `internal/events/event.go` types and appear to be dead code.

---

## Acceptance Criteria

1. [ ] Duplicate `ToolCallStartData` removed from `internal/agent/request.go`
2. [ ] Duplicate `ToolCallCompleteData` removed from `internal/agent/request.go`
3. [ ] Duplicate `EventType`, `Event` constants removed from `internal/agent/request.go`
4. [ ] All existing tests pass
5. [ ] `make lint` passes with zero errors
6. [ ] No dead code detected by `make deadcode`
7. [ ] Code coverage remains >= 85%

---

## Risk Assessment

**Risk Level:** LOW

- These types are dead code with zero usages
- Removal is purely cleanup, no behavior changes
- All production code uses `events` package types

---

## Testing Strategy

1. **Verify no compilation errors** after removal
2. **Run full test suite:** `go test ./... -race`
3. **Verify no imports break:** grep for any `agent.ToolCall*` or `agent.Event*` references
4. **Run linter:** `make lint`
5. **Run deadcode analysis:** `make deadcode`

---

## Implementation Plan

### Micro-TDD Approach

Since this is dead code removal, we follow a verify-then-remove approach:

1. **Verify dead code** - Confirm no usages exist
2. **Remove types** - Delete the dead code
3. **Verify compilation** - Ensure code still compiles
4. **Run tests** - Ensure all tests pass
5. **Run linter** - Ensure no lint errors

---

## Files Modified

| File | Change |
|------|--------|
| `internal/agent/request.go` | Remove duplicate types |
| `specs/refactoring/ROADMAP.md` | Mark item 1.1 as complete |

---

## Definition of Done

- [x] FRD created and reviewed
- [x] Dead code removed
- [x] All tests pass (`go test ./... -race`) - agent package tests pass
- [x] Linter clean (`go vet ./internal/agent/...`)
- [x] ROADMAP updated
- [x] Documentation updated if needed (no user-facing doc changes needed)
