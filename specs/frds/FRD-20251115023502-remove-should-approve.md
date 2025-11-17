# FRD-20251115023502: Remove Agent.ShouldApprove() Method

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (2 days)
- **Dependencies**: Feature 1.3 (complete), Feature 1.4 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-21-remove-agentshouldapprove-method)

## Problem Statement

`Agent.ShouldApprove()` method duplicates classification-to-approval logic that already exists in SecurityService. Currently:

1. **Duplication of logic** (agent.go:599-638):
   - `ShouldApprove()` reimplements classification-to-approval mapping
   - Core logic duplicates `CommandClass.NeedsApproval()` behavior
   - Only adds `requireApproval` flag check and custom reason messages

2. **Service layer violation**:
   - Agent directly implements approval logic that should be in SecurityService
   - Violates separation of concerns - Agent shouldn't know approval details

3. **Maintenance burden**:
   - Approval logic maintained in two places
   - Changes must be synchronized between Agent and SecurityService

**Impact:**
- Code duplication (classification-to-approval mapping)
- Service layer violation (Agent implements SecurityService logic)
- Maintenance burden (changes in two places)
- Inconsistent behavior risk

## Goals

1. **Remove `Agent.ShouldApprove()` method** entirely
2. **Update `determineRequiresApproval()`** to use SecurityService directly
3. **Preserve `requireApproval` flag behavior** - check flag before SecurityService
4. **Maintain same functionality** - no behavioral changes
5. **Update tests** to use SecurityService directly

## Non-Goals

1. **NOT removing `requireApproval` flag** - this is Agent-level configuration
2. **NOT changing approval flow** - only removing duplicate logic
3. **NOT changing SecurityService API** - use existing methods

## Design

### Current Implementation (agent.go:599-638)

```go
func (a *Agent) ShouldApprove(cmd *security.Command) (bool, string) {
    // If approval is disabled, never require approval
    if !a.requireApproval {
        return false, ""
    }

    // Classify the command via security service
    result, err := a.security.ValidateCommand(cmd)
    if err != nil {
        // On error, require approval for safety
        return true, fmt.Sprintf("Classification error: %v", err)
    }

    switch result.Classification {
    case security.CommandSafe:
        return false, ""
    case security.CommandInteractive:
        return true, "This command may modify files or system state"
    case security.CommandDangerous:
        return true, fmt.Sprintf("WARNING: Dangerous operation - %s", result.Reason)
    case security.CommandForbidden:
        return false, fmt.Sprintf("BLOCKED: %s", result.Reason)
    case security.CommandUnverified:
        return true, "Unknown command, approval required for safety"
    default:
        return true, "Unknown command classification, approval required"
    }
}
```

### Target Implementation

Replace `ShouldApprove()` usage in `determineRequiresApproval()`:

**Current (agent.go:658):**
```go
needsApproval, _ := a.ShouldApprove(cmdStruct)
return needsApproval
```

**Target:**
```go
// Check Agent-level approval flag first
if !a.requireApproval {
    return false
}

// Use SecurityService to check if approval is needed
return a.security.NeedsApproval(cmdStruct)
```

### Changes Required

1. **Remove `ShouldApprove()` method** (lines 599-638)
2. **Update `determineRequiresApproval()`** (line 658):
   - Replace `a.ShouldApprove(cmdStruct)` with direct SecurityService call
   - Check `a.requireApproval` flag first
   - Use `a.security.NeedsApproval(cmdStruct)`
3. **Update tests**:
   - Remove tests for `ShouldApprove()` method
   - Update `determineRequiresApproval()` tests to verify SecurityService usage
   - Remove benchmark for `ShouldApprove()`

### Challenges

1. **Return signature difference**:
   - `ShouldApprove()` returns `(bool, string)` - needsApproval, reason
   - `SecurityService.NeedsApproval()` returns `bool` - needsApproval only
   - Solution: `determineRequiresApproval()` only uses the bool, so this is fine

2. **Error handling**:
   - `ShouldApprove()` returns true on validation error (safe default)
   - `SecurityService.NeedsApproval()` returns false on error (fail-open)
   - Solution: Both are safe defaults, but we should verify behavior matches

3. **Forbidden commands**:
   - `ShouldApprove()` returns false for forbidden commands (they're blocked, not approved)
   - `SecurityService.NeedsApproval()` returns true for forbidden (they need approval)
   - Actually wait - let me check `CommandClass.NeedsApproval()` behavior...

Actually, looking at the code:
- `ShouldApprove()` returns false for forbidden (blocked, not approved)
- `SecurityService.NeedsApproval()` delegates to `validator.NeedsApproval()` which checks `CommandClass.NeedsApproval()`
- `CommandClass.NeedsApproval()` returns true for forbidden (they need approval)

So there's a discrepancy! Forbidden commands:
- `ShouldApprove()` returns false (blocked)
- `CommandClass.NeedsApproval()` returns true (needs approval)

But actually, forbidden commands should be blocked, not approved. So `determineRequiresApproval()` should return false for forbidden commands.

Let me check what `determineRequiresApproval()` is used for - it's used to populate `RequiresApproval` field in ToolCallStartData. For forbidden commands, we should return false (don't require approval, just block it).

So the solution is:
- Check if command is forbidden first (return false - blocked)
- Then check if approval is needed (return true/false)

Actually, `determineRequiresApproval()` is just for the event - it doesn't actually block execution. The blocking happens elsewhere (in executor). So for forbidden commands, we could return false (don't require approval in the event, just block it).

But to be safe, let's keep the same behavior:
- Check if command is forbidden - return false (blocked, not approved)
- Otherwise, use `SecurityService.NeedsApproval()`

We need to validate first, then check classification, then use NeedsApproval.

Actually, I think the cleanest approach is:
1. Check `requireApproval` flag first
2. Validate command
3. If forbidden, return false (blocked)
4. Otherwise, use `SecurityService.NeedsApproval()`

## API Changes

### Agent struct

**No changes** - `requireApproval` field remains.

### Agent methods

**Removed:**
```go
func (a *Agent) ShouldApprove(cmd *security.Command) (bool, string)
```

**Modified:**
```go
func (a *Agent) determineRequiresApproval(toolName string, args map[string]interface{}) bool {
    // Tools that always require approval
    requiresApprovalTools := map[string]bool{
        "execute_command": true,
        "write_file":      true,
        "apply_patch":     true,
    }

    if requiresApprovalTools[toolName] {
        return true
    }

    // For execute_command, also check if the command itself requires approval
    if toolName == "execute_command" {
        if cmd, ok := args["command"].(string); ok && cmd != "" {
            cmdStruct := &security.Command{Program: cmd}
            
            // Check Agent-level approval flag first
            if !a.requireApproval {
                return false
            }

            // Validate command to check if forbidden
            result, err := a.security.ValidateCommand(cmdStruct)
            if err != nil {
                // On validation error, require approval for safety (matching ShouldApprove behavior)
                return true
            }

            // Forbidden commands are blocked, not approved
            if result.Classification == security.CommandForbidden {
                return false
            }

            // Use SecurityService to check if approval is needed
            return a.security.NeedsApproval(cmdStruct)
        }
    }

    return false
}
```

**Breaking Change**: Yes - `ShouldApprove()` method removed. Callers must use SecurityService directly.

## Implementation Plan

### Step 1: Update determineRequiresApproval()
1. Replace `a.ShouldApprove(cmdStruct)` call with SecurityService logic
2. Check `requireApproval` flag first
3. Validate command to check if forbidden
4. Use `SecurityService.NeedsApproval()` for approval check
5. Handle errors appropriately

### Step 2: Remove ShouldApprove() method
1. Remove method implementation (lines 599-638)
2. Update godoc comments if needed

### Step 3: Update tests
1. Remove tests for `ShouldApprove()` method
2. Update `determineRequiresApproval()` tests to verify SecurityService usage
3. Remove benchmark for `ShouldApprove()`
4. Add tests for `determineRequiresApproval()` with SecurityService

### Step 4: Verify no other callers
1. Search codebase for all `ShouldApprove()` calls
2. Update any remaining callers

### Step 5: Verify behavior unchanged
1. Run all Agent tests
2. Run integration tests
3. Verify approval flow works correctly

## Testing Strategy

### Unit Tests

```go
func TestAgent_DetermineRequiresApproval_UsesSecurityService(t *testing.T) {
    securityService := createMockSecurityService(t)
    agent := NewAgent(..., securityService, ...)
    
    // Test that determineRequiresApproval uses SecurityService
    // Verify same behavior as ShouldApprove
}

func TestAgent_DetermineRequiresApproval_RequireApprovalFlag(t *testing.T) {
    // Test that requireApproval flag is checked first
    // If false, should return false without calling SecurityService
}

func TestAgent_DetermineRequiresApproval_ForbiddenCommands(t *testing.T) {
    // Test that forbidden commands return false (blocked, not approved)
}
```

### Acceptance Criteria

1. ✅ All callers of `Agent.ShouldApprove()` replaced
2. ✅ All callers use `SecurityService` directly
3. ✅ `Agent.ShouldApprove()` method removed
4. ✅ `determineRequiresApproval()` updated to use SecurityService
5. ✅ `requireApproval` flag behavior preserved
6. ✅ Forbidden commands return false (blocked)
7. ✅ All Agent tests pass
8. ✅ Integration tests verify approval flow works correctly
9. ✅ `go vet` passes
10. ✅ No dead code introduced

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully removed `Agent.ShouldApprove()` method and replaced it with direct SecurityService usage in `determineRequiresApproval()`. Preserved `requireApproval` flag behavior and forbidden command handling. All tests pass with no functional changes.

## Files to Modify

- `internal/agent/agent.go` - Remove `ShouldApprove()` method (lines 599-638), update `determineRequiresApproval()` (line 658)
- `internal/agent/agent_test.go` - Remove tests for `ShouldApprove()`, update `determineRequiresApproval()` tests

## Risks and Mitigation

### Risk 1: Behavior change for forbidden commands
**Risk**: `ShouldApprove()` returns false for forbidden, but `NeedsApproval()` might return true.
**Mitigation**: Explicitly check for forbidden commands before calling `NeedsApproval()`.

### Risk 2: Error handling differences
**Risk**: `ShouldApprove()` returns true on error, `NeedsApproval()` returns false on error.
**Mitigation**: Explicitly handle validation errors, match `ShouldApprove()` behavior (return true on error for safety).

### Risk 3: requireApproval flag not checked
**Risk**: SecurityService doesn't know about Agent's `requireApproval` flag.
**Mitigation**: Check `requireApproval` flag first before calling SecurityService.

## Dependencies

- ✅ Feature 1.3 (Executor uses SecurityService) - Complete
- ✅ Feature 1.4 (ShellCommandTool uses SecurityService) - Complete
- `security.SecurityService` - NeedsApproval() method exists
- `security.Validator` - NeedsApproval() method exists

## Success Metrics

- [ ] Zero calls to `Agent.ShouldApprove()`
- [ ] All approval checks use SecurityService
- [ ] All tests pass (unit, integration)
- [ ] No functional regression

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md) - Section 5.4
- [Roadmap Feature 2.1](../../codepath-duplication-assessment/ROADMAP.md#feature-21-remove-agentshouldapprove-method)
- `internal/agent/agent.go` - ShouldApprove() method
- `internal/security/security.go` - NeedsApproval() method

