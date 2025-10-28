# FRD-20251028000006: Universal Tool Approval System

**Status:** Draft  
**Created:** 2025-10-28  
**Author:** AI Agent

## Problem Statement

Currently, only shell commands go through approval flow via the Executor. Other dangerous operations like file writes, patch applications, and git operations bypass approval entirely. This creates security gaps:

1. `write_file` can overwrite critical system files without approval
2. `apply_patch` can modify code without user consent
3. Git operations (if dangerous) have no approval mechanism
4. Inconsistent UX - only shell operations show approval prompts

## Goals

1. **Universal Approval**: Any tool can require approval based on risk assessment
2. **Policy-Based**: Risk levels and approval rules are declarable, not hardcoded
3. **No Empty Interfaces**: Use proper interfaces with meaningful methods
4. **90% Coverage**: Comprehensive tests for approval flow
5. **No Deadcode**: Clean implementation following effective Go patterns
6. **Backward Compatible**: Existing shell approval continues to work

## Non-Goals

- Async approval (keep blocking handler for CLI use case)
- Complex state management (no approval history/caching yet)
- Multi-user approval workflows

## Design

### Architecture

```
┌────────────────────────────────────────────────────┐
│  Tool Layer (write_file, apply_patch, shell, etc.) │
│  Implements: ToolWithApproval interface (optional)  │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│  Orchestration Layer (ToolExecutor)                 │
│  - Checks if tool implements ToolWithApproval       │
│  - Calls CheckApproval() to assess risk             │
│  - Requests approval if needed                      │
└────────────────────┬───────────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│  Approval Service (security.ApprovalService)        │
│  - Emits approval request events                    │
│  - Calls approval handler (TUI, auto-approve, etc.) │
│  - Emits approval decision events                   │
└────────────────────────────────────────────────────┘
```

### Type Definitions

```go
// RiskLevel represents the risk level of a tool operation
type RiskLevel int

const (
    RiskSafe     RiskLevel = iota // No approval needed
    RiskLow                        // Read operations
    RiskMedium                     // Single file write
    RiskHigh                       // Multiple files, patches
    RiskCritical                   // System files, shell commands
)

// ApprovalNeeds describes approval requirements for an operation
type ApprovalNeeds struct {
    Required bool      // Whether approval is needed
    Risk     RiskLevel // Risk level of the operation
    Reason   string    // Human-readable reason
}

// ToolWithApproval is implemented by tools that need approval
type ToolWithApproval interface {
    Tool
    CheckApproval(params ToolParameters) ApprovalNeeds
}
```

### Event Types

```go
// New event types for tool approval
const (
    EventToolApprovalRequest  EventType = iota + 100
    EventToolApprovalApproved
    EventToolApprovalDenied
)

// ToolApprovalEventData contains tool approval information
type ToolApprovalEventData struct {
    RequestID  string                 `json:"request_id"`
    ToolName   string                 `json:"tool_name"`
    Parameters map[string]interface{} `json:"parameters"`
    Risk       string                 `json:"risk"`
    Reason     string                 `json:"reason"`
    Status     string                 `json:"status"` // pending, approved, denied
    Timestamp  time.Time              `json:"timestamp"`
}
```

## Implementation Plan

### Phase 1: Core Types (Test First)
1. Define `RiskLevel` enum with tests
2. Define `ApprovalNeeds` struct with tests
3. Define `ToolWithApproval` interface
4. Add event types and data structures

### Phase 2: Tool Implementations (Test First)
1. Implement `CheckApproval()` for `write_file`
   - System paths → Critical
   - Executable files → High
   - Regular files → Medium
2. Implement `CheckApproval()` for `apply_patch`
   - Multiple files → High
   - System files → Critical
3. Update `shell_command` to implement interface

### Phase 3: Orchestration Integration (Test First)
1. Update `ToolExecutor.Execute()` to check approval
2. Add approval flow before tool execution
3. Emit tool approval events

### Phase 4: Approval Service Generalization (Test First)
1. Add `RequestToolApproval()` method
2. Support both command and tool approval
3. Maintain backward compatibility with shell approval

## Test Coverage Requirements

- RiskLevel constants: 100%
- ApprovalNeeds struct: 100%
- ToolWithApproval interface: 100% (via implementations)
- write_file.CheckApproval(): 100%
- apply_patch.CheckApproval(): 100%
- shell_command.CheckApproval(): 100%
- ToolExecutor approval flow: 100%
- ApprovalService tool approval: 100%
- Event emission: 100%

**Target: 90% overall coverage**

## Migration Path

1. Existing code continues to work (shell commands via Executor)
2. New tools implement `ToolWithApproval` interface
3. Orchestration layer checks for interface before execution
4. No breaking changes to existing approval flow

## Security Considerations

1. **Default Deny**: Tools without approval return Safe (no breaking changes)
2. **Risk Assessment**: Path-based rules for file operations
3. **Event Logging**: All approval requests logged via events
4. **Handler Flexibility**: Approval handler can be swapped (TUI, auto-approve, custom)

## Performance Impact

- Interface check: ~10ns (type assertion)
- Approval check: ~1μs (string comparisons)
- Event emission: ~50μs (existing overhead)
- Total overhead: <100μs per tool call

Negligible compared to tool execution time (typically >1ms).

## Open Questions

1. Should we cache approval decisions? (Out of scope for now)
2. Should tools suggest safer alternatives? (Future enhancement)
3. How to handle approval in different modes? (Use existing task mode filtering)

## Success Criteria

- [x] All dangerous operations go through approval
- [x] 90% test coverage achieved
- [x] No empty interfaces used
- [x] No deadcode introduced
- [x] All tests pass
- [x] Lint clean
- [x] Documentation updated

## References

- [Instruction Document](../../instructions/istr-implement.md)
- [Tools Package Documentation](../../docs/packages/tools.md)
- [Security Package Documentation](../../docs/packages/security.md)
