# FRD-20251115024549: Extract Operation Construction Helper

## Metadata
- **Status**: IN PROGRESS
- **Priority**: P2 (MEDIUM)
- **Effort**: S (1 day)
- **Dependencies**: Feature 2.1 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-23-extract-operation-construction-helper)

## Problem Statement

`security.Operation` struct is constructed manually in multiple places with similar patterns. Currently:

1. **Manual construction in security.go** (security.go:115-119):
   ```go
   operation := Operation{
       Command: cmd,
       Reason:  fmt.Sprintf("Command classified as %s: %s", result.Classification, result.Reason),
       WorkDir: workDir,
   }
   ```

2. **Manual construction in tool_runtime.go** (tool_runtime.go:114-118):
   ```go
   operation := security.Operation{
       Command: cmd,
       Reason:  needs.Reason,
       WorkDir: t.workDir,
   }
   ```

3. **Manual construction in protocol/acp/agent.go** (agent.go:1220-1224):
   ```go
   return security.Operation{
       Command: cmd,
       Reason:  reason,
       WorkDir: workDir,
   }, nil
   ```

4. **Manual construction in tests** (multiple test files):
   - Multiple test files construct Operation manually with similar patterns

**Impact:**
- Code duplication (same construction pattern repeated)
- Inconsistency risk (different formatting or field order)
- Maintenance burden (changes must be made in multiple places)
- No standardization (no single source of truth)

## Goals

1. **Create `security.NewOperation()` helper function** to standardize Operation construction
2. **Replace all manual Operation construction** with helper function
3. **Ensure consistent behavior** across all construction sites
4. **Simplify future changes** - single place to update Operation construction logic

## Non-Goals

1. **NOT changing Operation struct** - keep existing structure
2. **NOT changing Operation fields** - keep Command, Reason, WorkDir
3. **NOT removing existing functionality** - only standardizing construction

## Design

### Current Implementation

**Pattern 1 (security.go:115-119):**
```go
operation := Operation{
    Command: cmd,
    Reason:  fmt.Sprintf("Command classified as %s: %s", result.Classification, result.Reason),
    WorkDir: workDir,
}
```

**Pattern 2 (tool_runtime.go:114-118):**
```go
operation := security.Operation{
    Command: cmd,
    Reason:  needs.Reason,
    WorkDir: t.workDir,
}
```

**Pattern 3 (protocol/acp/agent.go:1220-1224):**
```go
return security.Operation{
    Command: cmd,
    Reason:  reason,
    WorkDir: workDir,
}, nil
```

### Target Implementation

**Helper function:**
```go
// NewOperation creates a new Operation with the given command, reason, and work directory.
func NewOperation(cmd *Command, reason string, workDir string) Operation {
    return Operation{
        Command: cmd,
        Reason:  reason,
        WorkDir: workDir,
    }
}
```

**Usage:**
```go
// Pattern 1
operation := NewOperation(cmd, fmt.Sprintf("Command classified as %s: %s", result.Classification, result.Reason), workDir)

// Pattern 2
operation := NewOperation(cmd, needs.Reason, t.workDir)

// Pattern 3
return NewOperation(cmd, reason, workDir), nil
```

### Changes Required

1. **Create `NewOperation()` helper** in `request.go` (same file as Operation struct)
2. **Replace manual construction** in:
   - `security.go:115-119` - ValidateAndApprove()
   - `tool_runtime.go:114-118` - ToolRuntime approval
   - `protocol/acp/agent.go:1220-1224` - ACP agent
3. **Update tests** to use helper (optional but recommended)

## API Changes

### New Function

```go
// NewOperation creates a new Operation with the given command, reason, and work directory.
//
// This helper function standardizes Operation construction across the codebase,
// ensuring consistent behavior and simplifying future changes.
func NewOperation(cmd *Command, reason string, workDir string) Operation {
    return Operation{
        Command: cmd,
        Reason:  reason,
        WorkDir: workDir,
    }
}
```

**Breaking Change**: No - additive change only.

## Implementation Plan

### Step 1: Create NewOperation() helper
1. Add helper function to `request.go` (same file as Operation struct)
2. Add godoc documentation
3. Add unit tests for helper

### Step 2: Replace manual construction in security.go
1. Replace `Operation{...}` with `NewOperation(...)`
2. Verify behavior unchanged

### Step 3: Replace manual construction in tool_runtime.go
1. Replace `security.Operation{...}` with `security.NewOperation(...)`
2. Verify behavior unchanged

### Step 4: Replace manual construction in protocol/acp/agent.go
1. Replace `security.Operation{...}` with `security.NewOperation(...)`
2. Verify behavior unchanged

### Step 5: Update tests (optional)
1. Replace manual Operation construction in tests with helper
2. Verify all tests pass

### Step 6: Verify no functional changes
1. Run all security tests
2. Run all agent tests
3. Run all protocol tests
4. Verify behavior unchanged

## Testing Strategy

### Unit Tests

```go
func TestNewOperation(t *testing.T) {
    cmd := &Command{Program: "rm", Args: []string{"-rf", "/tmp"}}
    reason := "Dangerous operation"
    workDir := "/tmp"
    
    op := NewOperation(cmd, reason, workDir)
    
    assert.Equal(t, cmd, op.Command)
    assert.Equal(t, reason, op.Reason)
    assert.Equal(t, workDir, op.WorkDir)
}

func TestNewOperation_NilCommand(t *testing.T) {
    op := NewOperation(nil, "test", "/tmp")
    assert.Nil(t, op.Command)
}

func TestNewOperation_EmptyReason(t *testing.T) {
    cmd := &Command{Program: "ls"}
    op := NewOperation(cmd, "", "/tmp")
    assert.Equal(t, "", op.Reason)
}

func TestNewOperation_EmptyWorkDir(t *testing.T) {
    cmd := &Command{Program: "ls"}
    op := NewOperation(cmd, "test", "")
    assert.Equal(t, "", op.WorkDir)
}
```

### Acceptance Criteria

1. ✅ `security.NewOperation()` helper created
2. ✅ All manual Operation construction replaced with helper
3. ✅ Helper has unit tests with 100% coverage
4. ✅ All tests pass
5. ✅ `go vet` passes
6. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully extracted Operation construction helper. `security.NewOperation()` standardizes Operation construction across the codebase. All manual Operation constructions in production code and tests replaced with helper function. All tests pass with no functional changes.

## Files to Modify

- `internal/security/request.go` - Create `NewOperation()` helper
- `internal/security/security.go` - Replace manual construction (line 115-119)
- `internal/agent/tool_runtime.go` - Replace manual construction (line 114-118)
- `internal/protocol/acp/agent.go` - Replace manual construction (line 1220-1224)
- `internal/security/request_test.go` - Add tests for `NewOperation()` (if exists, or create)

## Risks and Mitigation

### Risk 1: Behavior change
**Risk**: Helper function might behave differently than manual construction.
**Mitigation**: Keep implementation simple, identical to manual construction, add tests.

### Risk 2: Performance overhead
**Risk**: Function call overhead vs direct struct literal.
**Mitigation**: Minimal overhead (simple struct construction), compiler should inline.

### Risk 3: Missing construction sites
**Risk**: Some manual construction sites might be missed.
**Mitigation**: Use grep to find all `Operation{` constructions, verify all replaced.

## Dependencies

- ✅ Feature 2.1 (ShouldApprove removed) - Complete
- `security.Operation` - Struct already exists

## Success Metrics

- [ ] Zero manual `Operation{` constructions in production code
- [ ] All Operation construction uses helper
- [ ] All tests pass (unit, integration)
- [ ] No functional regression

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 2.3](../../codepath-duplication-assessment/ROADMAP.md#feature-23-extract-operation-construction-helper)
- `internal/security/request.go` - Operation struct definition
- `internal/security/security.go` - ValidateAndApprove() method
- `internal/agent/tool_runtime.go` - ToolRuntime approval logic
- `internal/protocol/acp/agent.go` - ACP agent Operation construction

