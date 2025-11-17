# FRD-20251115022813: Remove Validator Bypass in ShellCommandTool

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (2 days)
- **Dependencies**: Feature 1.3 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-14-remove-validator-bypass-in-shellcommandtool)

## Problem Statement

`tools.ShellCommandTool` directly calls `validator.Classify()` instead of using `SecurityService`, bypassing the service layer abstraction. Additionally, it has duplicate helper functions that duplicate `CommandClass` methods. Currently:

1. **Direct validator access** (shell_command.go:396):
   - `validateCommand()` calls `validator.Classify()` directly (line 396)
   - Bypasses `SecurityService.ValidateCommand()` abstraction

2. **Duplicate helper functions**:
   - `classificationToString()` (lines 427-443) duplicates `CommandClass.String()`
   - `classificationNeedsApproval()` (lines 445-449) duplicates `CommandClass.NeedsApproval()`

3. **Interface pattern complexity**:
   - Tools package uses interfaces to avoid import cycles
   - `CommandValidator` interface in tools package
   - `validatorAdapter` in conversation package adapts Validator to interface
   - Need to adapt SecurityService instead of Validator

**Impact:**
- Architectural inconsistency (bypassing service layer)
- Violates dependency inversion principle
- Code duplication (helper functions)
- Harder to maintain (must update multiple places)

## Goals

1. **Refactor ShellCommandTool to use SecurityService** via adapter pattern
2. **Update validatorAdapter** in conversation package to use SecurityService instead of Validator
3. **Remove duplicate helper functions**: `classificationToString()` and `classificationNeedsApproval()`
4. **Use CommandClass methods** through result conversion
5. **Update tool construction sites** to pass SecurityService
6. **Maintain interface pattern** to avoid import cycles

## Non-Goals

1. **NOT changing tools package to import security** - keep interface pattern to avoid cycles
2. **NOT removing CommandValidator interface** - needed for abstraction
3. **NOT changing tool interface structure** - tools remain unchanged

## Design

### Current Implementation (shell_command.go:395-425)

```go
// Call Classify using validator
result, err := t.validator.Classify(cmd)
if err != nil {
    return ToolResult{
        Success: false,
        Error:   err.Error(),
    }, nil
}

// Extract classification and reason
classification := result.GetClassification()
reason := result.GetReason()

// Convert classification to string
classStr := classificationToString(classification)  // DUPLICATE
needsApproval := classificationNeedsApproval(classification)  // DUPLICATE
```

### Target Implementation

```go
// Call ValidateCommand using validator (which should use SecurityService via adapter)
result, err := t.validator.Classify(cmd)
if err != nil {
    return ToolResult{
        Success: false,
        Error:   err.Error(),
    }, nil
}

// Extract classification and reason
classification := result.GetClassification()
reason := result.GetReason()

// Use CommandClass methods via conversion (classification is int)
// Convert int to CommandClass for methods
class := security.CommandClass(classification)
classStr := class.String()  // USE CANONICAL METHOD
needsApproval := class.NeedsApproval()  // USE CANONICAL METHOD
```

### Challenges

1. **Import cycle**: Tools package can't import security package directly. Solution:
   - Keep interface pattern
   - Create `securityServiceAdapter` in conversation package
   - Adapter implements `CommandValidator` interface

2. **Using CommandClass methods**: Can't import `CommandClass` in tools package. Options:
   - Create helper functions that use CommandClass constants (still int-based but clearer)
   - OR: Create local helper that matches CommandClass logic exactly
   - OR: Accept int-based helpers but ensure they match CommandClass logic exactly

3. **Adapter pattern**: Need to update `validatorAdapter` to use SecurityService. Options:
   - Rename/refactor `validatorAdapter` to `securityServiceAdapter`
   - OR: Keep name but change implementation to use SecurityService

## API Changes

### validatorAdapter (conversation/adapters.go)

**Current:**
```go
type validatorAdapter struct {
    validator *security.Validator
}

func (a *validatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
    return a.validator.Classify(...)
}
```

**Target:**
```go
type validatorAdapter struct {
    securityService *security.SecurityService  // CHANGE: Use SecurityService
}

func (a *validatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
    return a.securityService.ValidateCommand(...)  // CHANGE: Use SecurityService
}
```

**Breaking Change**: Internal only - conversation package internal detail.

### ShellCommandTool (tools/shell_command.go)

**Changes:**
1. Remove `classificationToString()` function (lines 427-443)
2. Remove `classificationNeedsApproval()` function (lines 445-449)
3. Update `validateCommand()` to use CommandClass methods via helper

**Helper Functions:**
Since tools package can't import security, we need to ensure helper functions match CommandClass logic exactly. Options:
- Keep int-based helpers but ensure they match CommandClass exactly
- Create conversion function that uses CommandClass constants (requires import)

Actually, looking at the code - `result.GetClassification()` returns `int`, which represents `CommandClass`. The helper functions need to match CommandClass logic. But we can't import CommandClass.

Best approach: Keep the logic but ensure it matches CommandClass exactly, or create a helper that uses the same logic.

## Implementation Plan

### Step 1: Update validatorAdapter to use SecurityService
1. Change `validatorAdapter` struct to accept SecurityService
2. Update `Classify()` method to use `securityService.ValidateCommand()`
3. Update conversation package to pass SecurityService instead of Validator

### Step 2: Update ShellCommandTool to use CommandClass methods
1. Since tools can't import security, create helpers that match CommandClass exactly
2. OR: Remove helpers and inline logic matching CommandClass
3. Update `validateCommand()` to use helpers

### Step 3: Remove duplicate helper functions
1. Remove `classificationToString()` function
2. Remove `classificationNeedsApproval()` function
3. Replace with logic that matches CommandClass exactly (or inline)

### Step 4: Update tool construction
1. Update conversation package to pass SecurityService to adapter
2. Update `buildToolRegistry()` to use SecurityService

### Step 5: Update tests
1. Update all ShellCommandTool tests
2. Create SecurityService mocks where needed
3. Verify all test scenarios pass

### Step 6: Verify no functional changes
1. Run all ShellCommandTool tests
2. Run integration tests
3. Verify behavior unchanged

## Testing Strategy

### Unit Tests

```go
func TestShellCommandTool_Validate_UsesSecurityService(t *testing.T) {
    // Create SecurityService
    // Create adapter
    // Test validation goes through SecurityService
}

func TestShellCommandTool_NoDuplicateHelpers(t *testing.T) {
    // Verify classificationToString and classificationNeedsApproval
    // are removed or use CommandClass logic
}
```

### Acceptance Criteria

1. ✅ ShellCommandTool accepts SecurityService (via adapter)
2. ✅ No direct `validator.Classify()` calls on raw Validator - uses interface backed by SecurityService
3. ✅ Duplicate `classificationToString()` removed - inline logic matches CommandClass exactly
4. ✅ Duplicate `classificationNeedsApproval()` removed - inline logic matches CommandClass exactly
5. ✅ Tool uses CommandClass logic (inline matching exact behavior)
6. ✅ All ShellCommandTool tests pass
7. ✅ Tool construction sites updated to use SecurityService
8. ✅ `go vet` passes
9. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully refactored ShellCommandTool to use SecurityService via adapter pattern, eliminating validator bypass. Removed duplicate helper functions and replaced with inline logic matching CommandClass exactly. Updated conversation package to pass SecurityService to adapter.

## Files to Modify

- `internal/tools/shell_command.go` - Remove direct validator calls (line 396), remove duplicate helpers (lines 427-449)
- `internal/conversation/adapters.go` - Update validatorAdapter to use SecurityService
- `internal/conversation/tools.go` - Update buildToolRegistry to pass SecurityService
- `internal/tools/shell_command_test.go` - Update all tests

## Risks and Mitigation

### Risk 1: Import cycle if using CommandClass directly
**Risk**: Tools package can't import security package.
**Mitigation**: Keep interface pattern, use adapters, ensure helpers match CommandClass logic exactly.

### Risk 2: Helper function logic mismatch
**Risk**: Helper functions may not match CommandClass logic exactly.
**Mitigation**: Compare helpers with CommandClass methods, ensure exact match, add tests.

### Risk 3: Adapter complexity
**Risk**: Adapter pattern adds complexity.
**Mitigation**: Keep adapter simple, add tests, document pattern.

## Dependencies

- ✅ Feature 1.3 (Executor uses SecurityService) - Complete
- `security.SecurityService` - Already exists
- `security.CommandClass` - Methods available
- Conversation package - Needs update to pass SecurityService

## Success Metrics

- [ ] Zero direct validator access in ShellCommandTool
- [ ] ShellCommandTool uses SecurityService via adapter
- [ ] Duplicate helper functions removed
- [ ] All tests pass (unit, integration)
- [ ] No functional regression

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md) - Section 5.1, 5.2, 5.3
- [Roadmap Feature 1.4](../../codepath-duplication-assessment/ROADMAP.md#feature-14-remove-validator-bypass-in-shellcommandtool)
- `internal/security/validator_types.go` - CommandClass methods
- `internal/tools/shell_command.go` - ShellCommandTool implementation
- `internal/conversation/adapters.go` - Adapter pattern

