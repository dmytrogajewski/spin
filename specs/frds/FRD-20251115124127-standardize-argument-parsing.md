# FRD: Standardize Argument Parsing

**Feature ID**: FRD-20251115124127  
**Roadmap Feature**: 5.2  
**Priority**: P2 (MEDIUM)  
**Effort**: S (1 day)  
**Status**: ✅ **COMPLETE** (2025-11-15)

## Problem Statement

Argument parsing is inconsistent between `Agent.parseToolArguments()` (line 710-713) and `ToolRuntime.parseToolArguments()` (line 186-189):

- **Agent** uses `tools.NewStrictArgumentParser()` - requires non-empty arguments (`AllowEmpty: false`)
- **ToolRuntime** uses `tools.NewArgumentParser()` - allows empty arguments (`AllowEmpty: true`)

This inconsistency creates confusion and potential bugs:
- Empty arguments pass validation in ToolRuntime but fail in Agent
- Different error handling paths for the same input
- Unclear which behavior is "correct"

## Goals

1. **Standardize on single parser**: Choose canonical parser (`StrictArgumentParser` or `ArgumentParser`)
2. **Update both implementations**: Make Agent and ToolRuntime use the same parser
3. **Maintain consistency**: Ensure uniform argument parsing behavior across codebase
4. **No backward compatibility**: Remove inconsistent behavior (per user instruction)

## Design

### Current State

**`internal/agent/agent.go` (lines 710-713)**:
```go
func (a *Agent) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()  // AllowEmpty: false
	return parser.Parse(call.Function.Arguments)
}
```

**`internal/agent/tool_runtime.go` (lines 186-189)**:
```go
func (t *ToolRuntime) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewArgumentParser()  // AllowEmpty: true
	return parser.Parse(call.Function.Arguments)
}
```

### Parser Differences

**`tools.NewArgumentParser()`**:
- `AllowEmpty: true`
- Empty arguments return empty `ToolParameters`, no error
- More lenient, allows tools to be called without arguments

**`tools.NewStrictArgumentParser()`**:
- `AllowEmpty: false`
- Empty arguments return error: "tool arguments cannot be empty"
- Stricter, requires all tool calls to have arguments

### Decision: Standardize on `StrictArgumentParser`

**Rationale:**
1. **Agent already uses strict parser** - Consistency with higher-level orchestration
2. **Stricter validation is safer** - Catches errors earlier, prevents ambiguous tool calls
3. **Tool calls should have arguments** - Most tools require parameters, empty calls are edge cases
4. **Fail-fast principle** - Better to reject invalid input early rather than silently accept it

### Target State

**`internal/agent/tool_runtime.go`**:
```go
func (t *ToolRuntime) parseToolArguments(call *ToolCall) (tools.ToolParameters, error) {
	parser := tools.NewStrictArgumentParser()  // Changed from NewArgumentParser()
	return parser.Parse(call.Function.Arguments)
}
```

**`internal/agent/agent.go`**:
- No change needed (already uses `StrictArgumentParser`)

### API Changes

**CHANGED**: `ToolRuntime.parseToolArguments()` - Now uses `StrictArgumentParser` instead of `ArgumentParser`

**UNCHANGED**: `Agent.parseToolArguments()` - Already uses `StrictArgumentParser`

### Files to Modify

1. **`internal/agent/tool_runtime.go`**:
   - Change `tools.NewArgumentParser()` to `tools.NewStrictArgumentParser()` (line 187)

2. **`internal/agent/tool_runtime_test.go`** (if exists):
   - Update tests that verify empty argument handling to expect errors instead of empty map
   - Add tests for strict parsing behavior

3. **`internal/agent/agent_test.go`** (if any tests depend on ToolRuntime parsing):
   - Verify tests still pass with stricter parsing

### Files to Create

None

## Testing Strategy

### Unit Tests

**`internal/agent/tool_runtime_test.go`** (if exists):
```go
func TestToolRuntime_ParseToolArguments_EmptyArguments(t *testing.T) {
	// Should return error with strict parser
	call := &agent.ToolCall{
		ID: "test-id",
		Function: agent.ToolCallFunction{
			Name:      "test_tool",
			Arguments: "",  // Empty arguments
		},
	}
	
	toolRuntime := newTestToolRuntime(...)
	_, err := toolRuntime.parseToolArguments(call)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}
```

### Integration Tests

Verify that tool calls with empty arguments are consistently rejected:
- Agent rejects empty arguments
- ToolRuntime rejects empty arguments (changed behavior)
- Both return the same error message

### Test Updates Needed

Any tests that rely on ToolRuntime accepting empty arguments will need to be updated:
- Change test expectations from "empty map, no error" to "error"
- Update integration tests that use empty arguments

## Acceptance Criteria

1. ✅ Canonical parser chosen (`StrictArgumentParser`)
2. ✅ `ToolRuntime.parseToolArguments()` uses `StrictArgumentParser()`
3. ✅ `Agent.parseToolArguments()` still uses `StrictArgumentParser()` (no change)
4. ✅ Both Agent and ToolRuntime have consistent argument parsing behavior
5. ✅ All tests updated to reflect stricter parsing
6. ✅ All tests pass
7. ✅ `go vet` passes
8. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully standardized argument parsing across Agent and ToolRuntime. Chose `StrictArgumentParser` as canonical parser (Agent already used it, ToolRuntime was updated from `ArgumentParser`). Updated `ToolRuntime.parseToolArguments()` (line 187) to use `NewStrictArgumentParser()` instead of `NewArgumentParser()`. Both Agent and ToolRuntime now use consistent strict parsing that rejects empty arguments with error "tool arguments cannot be empty". Created comprehensive tests for `ToolRuntime.parseToolArguments()` in `internal/agent/tool_runtime_test.go` verifying strict parsing behavior (valid arguments, empty arguments, empty JSON object, invalid JSON). All tests pass, including integration tests. Empty arguments now consistently return errors in both Agent and ToolRuntime, ensuring uniform behavior across the codebase.

## Implementation Notes

- Standardizing on `StrictArgumentParser` means empty arguments will now return errors in ToolRuntime
- This is a breaking change, but per user instruction: "Do not maintain backward compatibility"
- Tools that legitimately need empty arguments should be updated to pass `{}` instead of empty string
- The error message is consistent: "tool arguments cannot be empty"

## Risks

- **Breaking change**: ToolRuntime will now reject empty arguments that were previously accepted
  - **Mitigation**: Update tests and verify no production code depends on lenient parsing

- **Tool compatibility**: Some tools might expect empty arguments to work
  - **Mitigation**: Check tool implementations, update them to pass `{}` instead of empty string

## Dependencies

- Feature 5.1 complete (tool call validation unified)

