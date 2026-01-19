# FRD-20260119-002: ToolResult Type Consolidation

**Created:** 2026-01-19  
**Author:** Architecture Refactoring  
**Status:** Completed  
**Completed:** 2026-01-19  
**Priority:** P0 - Critical  
**Roadmap Item:** 1.2 Triple ToolResult Definition

## Executive Summary

This FRD addresses the consolidation of duplicate `ToolResult` types across the codebase. Currently, two different `ToolResult` structs exist with overlapping but inconsistent fields, creating confusion, conversion overhead, and potential data loss.

## Problem Statement

### Current State

Two `ToolResult` types exist:

**1. `tools.ToolResult`** (`internal/tools/tool.go:39`)
```go
type ToolResult struct {
    Success  bool                   `json:"success"`
    Output   string                 `json:"output"`
    Error    string                 `json:"error,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

**2. `agent.ToolResult`** (`internal/agent/tool_runtime.go:21`)
```go
type ToolResult struct {
    ID       string
    Success  bool
    Output   string
    Error    error
    ExitCode int
    Metadata map[string]interface{}
}
```

### Key Differences

| Field | `tools.ToolResult` | `agent.ToolResult` |
|-------|-------------------|-------------------|
| ID | Not present | Present (tool call ID) |
| Success | `bool` | `bool` |
| Output | `string` | `string` |
| Error | `string` | `error` |
| ExitCode | Not present | Present |
| Metadata | `map[string]interface{}` | `map[string]interface{}` |

### Impact

1. **Type Confusion:** Developers must understand which `ToolResult` to use in each context
2. **Conversion Overhead:** `tool_runtime.go` converts between types (lines 135-143)
3. **Inconsistent Error Handling:** `string` vs `error` type creates different error patterns
4. **Missing Context:** Tools cannot access their call ID through `tools.ToolResult`
5. **Lost Information:** ExitCode is only available in agent layer, not in tool implementations

## Proposed Solution

### Design Decision

Consolidate to a single `tools.ToolResult` type that includes all fields from both definitions. The `tools` package is the canonical location since:
1. Tools are the producers of ToolResult
2. The `tools` package defines the `Tool` interface
3. Having the result type in the same package as the interface follows Go conventions

### New Unified Type

**Location:** `internal/tools/tool.go`

```go
// ToolResult represents the result of executing a tool.
type ToolResult struct {
    // ID is the unique identifier for this tool call.
    // This links the result back to the original ToolCall.
    ID string `json:"id,omitempty"`

    // Success indicates whether the tool execution succeeded.
    Success bool `json:"success"`

    // Output contains the tool's output message for the LLM.
    Output string `json:"output"`

    // Error contains an error if the tool failed.
    // When serializing to JSON, this is converted to a string.
    Error error `json:"-"`

    // ErrorMessage is the string representation of Error for JSON serialization.
    ErrorMessage string `json:"error,omitempty"`

    // ExitCode contains the exit code for command-based tools.
    // Zero indicates success, non-zero indicates failure.
    ExitCode int `json:"exit_code,omitempty"`

    // Metadata contains additional tool-specific data.
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

### Helper Functions

```go
// NewToolResult creates a successful tool result.
func NewToolResult(output string) ToolResult {
    return ToolResult{
        Success: true,
        Output:  output,
    }
}

// NewToolResultWithID creates a successful tool result with ID.
func NewToolResultWithID(id, output string) ToolResult {
    return ToolResult{
        ID:      id,
        Success: true,
        Output:  output,
    }
}

// NewToolError creates a failed tool result from an error.
func NewToolError(err error) ToolResult {
    return ToolResult{
        Success:      false,
        Error:        err,
        ErrorMessage: err.Error(),
    }
}

// NewToolErrorWithID creates a failed tool result with ID from an error.
func NewToolErrorWithID(id string, err error) ToolResult {
    return ToolResult{
        ID:           id,
        Success:      false,
        Error:        err,
        ErrorMessage: err.Error(),
    }
}

// WithID returns a copy of the result with the given ID.
func (r ToolResult) WithID(id string) ToolResult {
    r.ID = id
    return r
}

// WithExitCode returns a copy of the result with the given exit code.
func (r ToolResult) WithExitCode(code int) ToolResult {
    r.ExitCode = code
    return r
}

// WithMetadata returns a copy of the result with the given metadata.
func (r ToolResult) WithMetadata(metadata map[string]interface{}) ToolResult {
    r.Metadata = metadata
    return r
}

// Err returns the error if present, or nil.
func (r ToolResult) Err() error {
    return r.Error
}

// String returns a string representation of the result.
func (r ToolResult) String() string {
    if r.Success {
        return r.Output
    }
    if r.Error != nil {
        return r.Error.Error()
    }
    return r.ErrorMessage
}
```

## Migration Strategy

### Phase 1: Extend `tools.ToolResult`

1. Add `ID`, `ExitCode` fields to `tools.ToolResult`
2. Change `Error` from `string` to `error` type
3. Add `ErrorMessage` field for JSON serialization backward compatibility
4. Add helper constructors and methods

### Phase 2: Update Tool Implementations

All tool implementations already return `tools.ToolResult`. No changes needed for:
- `internal/tools/*.go` (all built-in tools)
- `internal/agent/runtime/acp_tools.go`
- `internal/mcp/manager.go`

### Phase 3: Update Agent Package

1. Remove `agent.ToolResult` type from `tool_runtime.go`
2. Update `ToolRuntime.Execute()` to use `tools.ToolResult` directly
3. Update `ToolRuntime.ExecuteBatch()` signature
4. Update `Agent.ProcessToolCall()` to use `tools.ToolResult`

### Phase 4: Update Tests

Update all test files that reference either ToolResult type.

## Acceptance Criteria

1. [x] Single `ToolResult` type in `internal/tools/tool.go`
2. [x] `agent.ToolResult` removed from `internal/agent/tool_runtime.go` (now type alias)
3. [x] All tool implementations use unified type
4. [x] All tests pass
5. [x] No conversion code between result types
6. [x] Code coverage >= 90%
7. [x] `go vet` passes (golangci-lint not installed)
8. [x] `uast/herr` analysis - documentation coverage 100%

## Test Plan

### Unit Tests

1. **ToolResult Construction**
   - `NewToolResult` creates success result
   - `NewToolError` creates error result
   - `WithID` sets ID correctly
   - `WithExitCode` sets exit code correctly
   - `WithMetadata` sets metadata correctly

2. **ToolResult Methods**
   - `Err()` returns error or nil
   - `String()` returns appropriate content

3. **JSON Serialization**
   - Success result serializes correctly
   - Error result includes error message
   - ID and ExitCode included when non-zero

### Integration Tests

1. **ToolRuntime.Execute**
   - Returns proper result type
   - ID is propagated from tool call
   - Error handling works correctly

2. **Agent.ProcessToolCall**
   - Result ID matches call ID
   - Events contain correct data

## Files Affected

| File | Change Type |
|------|-------------|
| `internal/tools/tool.go` | Modify - extend ToolResult |
| `internal/tools/tool_test.go` | Add - new tests |
| `internal/agent/tool_runtime.go` | Modify - remove duplicate type |
| `internal/agent/tool_runtime_test.go` | Modify - update tests |
| `internal/agent/agent.go` | Modify - use tools.ToolResult |
| `internal/agent/agent_test.go` | Modify - update tests |

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing tool implementations | High | All tools already use tools.ToolResult |
| JSON serialization changes | Medium | Add ErrorMessage field for compatibility |
| Test failures | Medium | Update tests incrementally |

## Timeline

This is a focused refactoring with minimal risk. Implementation follows micro-TDD cycles.

## References

- Roadmap: `specs/refactoring/ROADMAP.md` section 1.2
- Previous consolidation: `specs/frds/FRD-20260119-001-toolcall-type-consolidation.md`
- AGENTS.md quality gates
