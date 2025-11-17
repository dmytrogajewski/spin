# FRD-20251115022244: Remove Validator Bypass in Executor

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (2 days)
- **Dependencies**: None
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-13-remove-validator-bypass-in-executor)

## Problem Statement

`agent.Executor` directly calls `validator.Classify()` instead of using `SecurityService`, bypassing the service layer abstraction. Currently:

1. **Direct validator access** (executor.go:361-370):
   - `Validate()` method calls `validator.Classify()` directly (line 362)
   - Bypasses `SecurityService.ValidateCommand()` abstraction

2. **Inconsistent pattern**:
   - Other components use `SecurityService` for validation
   - Executor directly accesses Validator, violating service layer pattern
   - Same validation logic accessed via different paths

3. **Service layer violation**:
   - Violates SOLID principles (dependency inversion)
   - Makes it harder to mock/test (must mock low-level Validator)
   - Breaks service abstraction boundary

**Impact:**
- Architectural inconsistency (bypassing service layer)
- Violates dependency inversion principle
- Harder to test (must mock Validator instead of SecurityService)
- Makes it harder to add cross-cutting concerns to validation

## Goals

1. **Refactor Executor to use `SecurityService`** instead of raw `Validator`
2. **Replace `WithValidator()` option** with `WithSecurityService()`
3. **Update `Validate()` method** to use `securityService.ValidateCommand()`
4. **Update `requestApprovalIfNeeded()`** to work with SecurityService
5. **Update Builder** to pass SecurityService instead of Validator
6. **Update all Executor callers** to use SecurityService
7. **Maintain backward compatibility** where possible

## Non-Goals

1. **NOT changing SecurityService API** - use existing `ValidateCommand()` method
2. **NOT changing ApprovalService** - approval flow remains unchanged
3. **NOT changing Executor public API** (Execute, ExecuteStreaming) - internal changes only

## Design

### Current Implementation (executor.go:347-373)

```go
func (e *Executor) Validate(cmd *security.Command) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	// If validator is present, use it
	e.mu.RLock()
	validator := e.validator
	e.mu.RUnlock()

	if validator != nil {
		result, err := validator.Classify(cmd)  // DIRECT ACCESS - BYPASS
		if err != nil {
			return fmt.Errorf("%w: %v", ErrValidationFailed, err)
		}

		if result.Classification == security.CommandForbidden {
			return fmt.Errorf("%w: %s", ErrValidationFailed, result.Reason)
		}
	}

	return nil
}
```

### Target Implementation

```go
func (e *Executor) Validate(cmd *security.Command) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	// If security service is present, use it
	e.mu.RLock()
	securityService := e.securityService
	e.mu.RUnlock()

	if securityService != nil {
		result, err := securityService.ValidateCommand(cmd)  // VIA SERVICE LAYER
		if err != nil {
			return fmt.Errorf("%w: %v", ErrValidationFailed, err)
		}

		if result.Classification == security.CommandForbidden {
			return fmt.Errorf("%w: %s", ErrValidationFailed, result.Reason)
		}
	}

	return nil
}
```

### Changes Required

1. **Executor struct**: Replace `validator *security.Validator` with `securityService *security.SecurityService`
2. **WithValidator()**: Replace with `WithSecurityService()`
3. **Validate()**: Use `securityService.ValidateCommand()` instead of `validator.Classify()`
4. **requestApprovalIfNeeded()**: Extract validator from SecurityService or use SecurityService directly
5. **Builder.buildExecutor()**: Pass SecurityService instead of Validator
6. **All callers**: Update to use SecurityService

### Challenges

1. **ApprovalService access**: `requestApprovalIfNeeded()` uses `RequestApprovalWithValidator()` which needs a validator. Options:
   - Extract validator from SecurityService: `securityService.Validator()`
   - Or update to use `SecurityService.ValidateAndApprove()` instead

2. **Backward compatibility**: If `WithValidator()` is called from external code, we need to handle it. Options:
   - Remove `WithValidator()` entirely (breaking change)
   - Deprecate `WithValidator()` and create SecurityService from it internally
   - Keep both for transition period

3. **Builder integration**: Builder creates SecurityService, so we can use it directly

## API Changes

### Executor struct

```go
type Executor struct {
	securityService  *security.SecurityService  // NEW: replaces validator
	approvalService  *security.ApprovalService  // Keep as-is
	sandbox          any
	cache            *CommandCache
	workDir          string
	timeout          time.Duration
	maxOutput        int64
	env              map[string]string
	mu               sync.RWMutex
}
```

### ExecutorOption (replace)

```go
// OLD: WithValidator sets the command validator.
func WithValidator(v *security.Validator) ExecutorOption { ... }

// NEW: WithSecurityService sets the security service.
func WithSecurityService(s *security.SecurityService) ExecutorOption {
	return func(e *Executor) error {
		if s == nil {
			return fmt.Errorf("security service cannot be nil")
		}
		e.securityService = s
		return nil
	}
}
```

**Breaking Change**: Yes - `WithValidator()` removed. Callers must use `WithSecurityService()`.

### requestApprovalIfNeeded() update

```go
func (e *Executor) requestApprovalIfNeeded(ctx context.Context, cmd *security.Command, opts *ExecuteOptions) error {
	e.mu.RLock()
	securityService := e.securityService
	approvalService := e.approvalService
	e.mu.RUnlock()

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
	if err != nil {
		return fmt.Errorf("approval request failed: %w", err)
	}

	if !approved {
		return fmt.Errorf("command execution denied by user")
	}

	return nil
}
```

## Implementation Plan

### Step 1: Update Executor struct
1. Replace `validator` field with `securityService`
2. Update struct documentation

### Step 2: Replace WithValidator with WithSecurityService
1. Remove `WithValidator()` function
2. Add `WithSecurityService()` function
3. Add validation (nil check)

### Step 3: Update Validate() method
1. Replace `validator.Classify()` with `securityService.ValidateCommand()`
2. Update error handling if needed
3. Verify same behavior

### Step 4: Update requestApprovalIfNeeded()
1. Extract validator from SecurityService using getter
2. Pass to `RequestApprovalWithValidator()` as before
3. Handle nil SecurityService gracefully

### Step 5: Update Builder
1. Update `buildExecutor()` to pass SecurityService instead of Validator
2. Use `BuildSecurityService()` if available
3. Extract validator from SecurityService for ApprovalService if needed

### Step 6: Update Executor tests
1. Update all tests to use SecurityService
2. Create SecurityService mocks where needed
3. Verify all test scenarios pass

### Step 7: Update callers
1. Find all `NewExecutor()` calls with `WithValidator()`
2. Update to use `WithSecurityService()`
3. Update Builder usage

### Step 8: Verify no functional changes
1. Run all Executor tests
2. Run integration tests
3. Verify behavior unchanged

## Testing Strategy

### Unit Tests

```go
func TestExecutor_WithSecurityService(t *testing.T) {
	securityService := createTestSecurityService(t)
	executor, err := NewExecutor("/tmp", WithSecurityService(securityService))
	require.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestExecutor_Validate_UsesSecurityService(t *testing.T) {
	securityService := createMockSecurityService(t)
	executor, _ := NewExecutor("/tmp", WithSecurityService(securityService))
	
	cmd := &security.Command{Program: "rm", Args: []string{"-rf", "/"}}
	err := executor.Validate(cmd)
	
	// Verify SecurityService.ValidateCommand was called
	assert.Error(t, err)
	// Verify correct error for forbidden command
}

func TestExecutor_Validate_NilSecurityService(t *testing.T) {
	executor, _ := NewExecutor("/tmp") // No SecurityService
	
	cmd := &security.Command{Program: "ls", Args: []string{"-la"}}
	err := executor.Validate(cmd)
	
	// Should not error if SecurityService is nil (backward compatible)
	assert.NoError(t, err)
}
```

### Integration Tests

```go
func TestExecutor_Validation_EquivalentBehavior(t *testing.T) {
	// Create Executor with SecurityService
	// Run validation tests
	// Compare results with previous implementation
	// Verify same validation behavior
}
```

### Acceptance Criteria

1. ✅ Executor accepts `SecurityService` instead of `Validator`
2. ✅ `WithValidator()` replaced with `WithSecurityService()`
3. ✅ No direct `validator.Classify()` calls in Executor
4. ✅ All `validator.Classify()` calls replaced with `securityService.ValidateCommand()`
5. ✅ Error handling updated for SecurityService API
6. ✅ `requestApprovalIfNeeded()` works with SecurityService
7. ✅ All Executor unit tests pass
8. ✅ All Executor integration tests pass
9. ✅ Builder updated to pass SecurityService
10. ✅ All callers updated
11. ✅ `go vet` passes
12. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully refactored Executor to use SecurityService instead of direct Validator access, eliminating validator bypass. Updated Builder and all tests to use SecurityService pattern.

## Files to Modify

- `internal/agent/executor.go` - Replace validator with SecurityService (lines 171, 186-194, 347-373, 311-336)
- `internal/agent/builder.go` - Update BuildExecutor to use SecurityService (lines 202-229)
- `internal/agent/executor_test.go` - Update all tests to use SecurityService
- `internal/appserver/processor.go` - May need updates if creates Executor directly

## Risks and Mitigation

### Risk 1: Breaking change for external callers
**Risk**: `WithValidator()` removal breaks existing code.
**Mitigation**: 
- Search codebase for all `WithValidator()` usages
- Update all callers before removing
- Verify no external callers in search

### Risk 2: requestApprovalIfNeeded complexity
**Risk**: `RequestApprovalWithValidator()` still needs validator, requiring extraction.
**Mitigation**: Use `SecurityService.Validator()` getter added in Feature 1.2.

### Risk 3: Behavioral changes
**Risk**: SecurityService may behave differently than direct validator access.
**Mitigation**: Comprehensive tests, compare before/after behavior, ensure SecurityService.ValidateCommand() wraps validator correctly.

## Dependencies

- ✅ SecurityService has `Validator()` getter (added in Feature 1.2)
- `agent.Builder` - May need update to pass SecurityService

## Success Metrics

- [ ] Zero direct validator access in Executor
- [ ] Executor uses SecurityService for all validation
- [ ] All tests pass (unit, integration)
- [ ] No functional regression
- [ ] Service layer abstraction enforced

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md) - Section 5.1
- [Roadmap Feature 1.3](../../codepath-duplication-assessment/ROADMAP.md#feature-13-remove-validator-bypass-in-executor)
- `internal/security/security.go` - SecurityService implementation
- `internal/agent/executor.go` - Executor implementation

