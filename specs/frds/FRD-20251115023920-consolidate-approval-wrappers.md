# FRD-20251115023920: Consolidate Approval Request Wrappers

## Metadata
- **Status**: COMPLETE
- **Priority**: P1 (HIGH)
- **Effort**: S (1 day)
- **Dependencies**: Feature 2.1 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-22-consolidate-approval-request-wrappers)

## Problem Statement

Two wrapper methods exist for approval requests with overlapping functionality:

1. **`SecurityService.ValidateAndApprove()`** (security.go:92-127):
   - High-level method that validates and requests approval
   - Uses SecurityService's own validator and approvalService
   - Handles safe/forbidden commands explicitly
   - Returns `(bool, error)` - approved, error

2. **`ApprovalService.RequestApprovalWithValidator()`** (approval.go:231-262):
   - Lower-level method that takes validator as parameter
   - Checks if approval is needed, then requests approval
   - Does NOT handle forbidden commands explicitly
   - Returns `(bool, error)` - approved, error

3. **Current usage**:
   - Executor uses `RequestApprovalWithValidator()` and manually extracts validator from SecurityService
   - This is redundant - Executor has SecurityService and could use `ValidateAndApprove()` directly

**Impact:**
- Code duplication (overlapping functionality)
- Confusion about which method to use
- Maintenance burden (two methods to maintain)
- Inconsistent behavior (forbidden handling differs)

## Goals

1. **Analyze usage patterns** of both methods
2. **Determine canonical approval flow** - which method should be the standard?
3. **Consolidate usage** - update callers to use canonical pattern
4. **Document clear use cases** for remaining methods
5. **Remove redundant wrappers** if possible

## Non-Goals

1. **NOT removing core approval functionality** - `RequestApproval()` remains
2. **NOT changing approval behavior** - only consolidating wrappers
3. **NOT breaking backward compatibility unnecessarily** - deprecate rather than remove

## Design

### Analysis

#### SecurityService.ValidateAndApprove()
**Pros:**
- High-level, complete solution (validation + approval)
- Handles forbidden commands correctly (blocks without approval)
- Uses SecurityService's validator (no manual extraction)
- Clearer abstraction (SecurityService owns validation)

**Cons:**
- Requires SecurityService instance
- Less flexible (can't use external validator)

#### ApprovalService.RequestApprovalWithValidator()
**Pros:**
- More flexible (accepts external validator)
- Lower-level (allows customization)
- Can be used without SecurityService

**Cons:**
- Doesn't handle forbidden commands explicitly
- Requires manual validator extraction
- Duplicates logic that exists in SecurityService
- Less clear abstraction (ApprovalService doesn't own validation)

### Recommendation

**Canonical Pattern:** Use `SecurityService.ValidateAndApprove()` as the standard approval flow.

**Reasoning:**
1. SecurityService is the appropriate abstraction for validation + approval
2. It handles all cases correctly (safe, forbidden, dangerous)
3. Executor already has SecurityService - no reason to extract validator
4. Clearer separation of concerns

**Action:** 
- Update Executor to use `ValidateAndApprove()` directly
- Deprecate `RequestApprovalWithValidator()` (mark as deprecated but keep for backward compatibility)
- Document that `ValidateAndApprove()` is the preferred method

### Changes Required

1. **Update Executor** (executor.go:332):
   - Replace `RequestApprovalWithValidator()` with `ValidateAndApprove()`
   - Remove validator extraction (lines 326-330)
   - Simplify approval logic

2. **Deprecate `RequestApprovalWithValidator()`** (approval.go:229-262):
   - Add deprecation notice to godoc
   - Recommend using `SecurityService.ValidateAndApprove()` instead
   - Keep method for backward compatibility

3. **Update documentation**:
   - Document canonical approval pattern
   - Explain when to use each method (if both remain)

## API Changes

### Executor.requestApprovalIfNeeded()

**Current:**
```go
func (e *Executor) requestApprovalIfNeeded(ctx context.Context, cmd *security.Command, opts *ExecuteOptions) error {
    securityService := e.securityService
    approvalService := e.approvalService

    if approvalService == nil {
        return nil
    }

    workDir := opts.WorkDir
    if workDir == "" {
        workDir = e.workDir
    }

    // Extract validator from SecurityService for RequestApprovalWithValidator
    var validator *security.Validator
    if securityService != nil {
        validator = securityService.Validator()
    }

    approved, err := approvalService.RequestApprovalWithValidator(ctx, cmd, validator, workDir)
    // ...
}
```

**Target:**
```go
func (e *Executor) requestApprovalIfNeeded(ctx context.Context, cmd *security.Command, opts *ExecuteOptions) error {
    securityService := e.securityService

    if securityService == nil {
        return nil // No security service, skip approval
    }

    workDir := opts.WorkDir
    if workDir == "" {
        workDir = e.workDir
    }

    // Use SecurityService's high-level approval method
    approved, err := securityService.ValidateAndApprove(ctx, cmd, workDir)
    if err != nil {
        return fmt.Errorf("approval request failed: %w", err)
    }

    if !approved {
        return fmt.Errorf("command execution denied by user")
    }

    return nil
}
```

**Breaking Change**: No - internal method only.

### ApprovalService.RequestApprovalWithValidator()

**Change:** Add deprecation notice:
```go
// RequestApprovalWithValidator requests approval for a command using a validator.
// This is a convenience method that checks if approval is needed before requesting it.
//
// Deprecated: Use SecurityService.ValidateAndApprove() instead. This method
// is kept for backward compatibility but is less complete than ValidateAndApprove()
// (e.g., doesn't handle forbidden commands explicitly).
func (s *ApprovalService) RequestApprovalWithValidator(...) (bool, error) {
    // ... existing implementation
}
```

**Breaking Change**: No - deprecated but still functional.

## Implementation Plan

### Step 1: Update Executor
1. Replace `RequestApprovalWithValidator()` with `ValidateAndApprove()`
2. Remove validator extraction logic
3. Simplify approval service check (check SecurityService instead)
4. Handle nil SecurityService gracefully

### Step 2: Update Executor tests
1. Update tests to verify `ValidateAndApprove()` is called
2. Verify behavior unchanged (safe, forbidden, dangerous commands)
3. Test nil SecurityService handling

### Step 3: Deprecate RequestApprovalWithValidator
1. Add deprecation notice to godoc
2. Add recommendation to use ValidateAndApprove()
3. Keep implementation for backward compatibility

### Step 4: Update documentation
1. Document canonical approval pattern
2. Explain use cases for each method
3. Provide migration guide (if needed)

### Step 5: Verify no other callers
1. Search codebase for `RequestApprovalWithValidator()` calls
2. Update or document any remaining usage

### Step 6: Verify behavior unchanged
1. Run all Executor tests
2. Run integration tests
3. Verify approval flow works correctly

## Testing Strategy

### Unit Tests

```go
func TestExecutor_RequestApproval_UsesValidateAndApprove(t *testing.T) {
    securityService := createMockSecurityService(t)
    executor := NewExecutor("/tmp", WithSecurityService(securityService))
    
    // Verify ValidateAndApprove is called
    // Test safe, forbidden, dangerous commands
}

func TestExecutor_RequestApproval_NilSecurityService(t *testing.T) {
    executor := NewExecutor("/tmp") // No SecurityService
    
    // Should skip approval gracefully
}
```

### Acceptance Criteria

1. ✅ Executor uses `ValidateAndApprove()` instead of `RequestApprovalWithValidator()`
2. ✅ No validator extraction in Executor
3. ✅ `RequestApprovalWithValidator()` method removed entirely
4. ✅ All tests for removed method removed
5. ✅ All Executor tests pass
6. ✅ Integration tests verify approval flow works correctly
7. ✅ `go vet` passes
8. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully consolidated approval wrappers. Executor now uses `SecurityService.ValidateAndApprove()` as the canonical approval pattern, eliminating manual validator extraction. `RequestApprovalWithValidator()` method removed entirely along with all its tests. All tests pass with no functional changes.

## Files to Modify

- `internal/agent/executor.go` - Update `requestApprovalIfNeeded()` to use `ValidateAndApprove()`
- `internal/security/approval.go` - Remove `RequestApprovalWithValidator()` method entirely
- `internal/security/approval_service_test.go` - Remove tests for `RequestApprovalWithValidator()`
- `internal/security/approval_coverage_test.go` - Remove tests for `RequestApprovalWithValidator()`
- `internal/agent/executor_test.go` - Update tests for new implementation

## Risks and Mitigation

### Risk 1: Behavior change
**Risk**: `ValidateAndApprove()` might behave differently than `RequestApprovalWithValidator()`.
**Mitigation**: Compare behavior carefully, add tests for all cases, verify forbidden handling matches.

### Risk 2: Nil SecurityService handling
**Risk**: Executor might have SecurityService set but ApprovalService not set (legacy config).
**Mitigation**: Check SecurityService first, handle nil gracefully.

### Risk 3: Backward compatibility
**Risk**: Deprecating `RequestApprovalWithValidator()` might break existing code.
**Mitigation**: Keep method functional, only mark as deprecated, don't remove.

## Dependencies

- ✅ Feature 2.1 (ShouldApprove removed) - Complete
- `security.SecurityService` - ValidateAndApprove() method exists
- `security.ApprovalService` - RequestApprovalWithValidator() exists

## Success Metrics

- [ ] Executor uses canonical approval pattern
- [ ] No manual validator extraction in Executor
- [ ] `RequestApprovalWithValidator()` deprecated
- [ ] All tests pass (unit, integration)
- [ ] Documentation updated

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 2.2](../../codepath-duplication-assessment/ROADMAP.md#feature-22-consolidate-approval-request-wrappers)
- `internal/security/security.go` - ValidateAndApprove() method
- `internal/security/approval.go` - RequestApprovalWithValidator() method
- `internal/agent/executor.go` - Executor approval logic

