# FRD-20260120-002: Runtime Interface Segregation

**Created:** 2026-01-20  
**Author:** Architecture Refactoring  
**Status:** Draft  
**Priority:** P1 (High)  
**Roadmap Item:** 2.4 Interface Pollution - Runtime

## Problem Statement

The `Runtime` interface in `internal/agent/runtime/runtime.go` mixes 8 unrelated concerns into a single interface, violating the Interface Segregation Principle (ISP). This creates several problems:

1. **Forced Implementation**: All runtimes must implement methods they don't need
2. **Tight Coupling**: Consumers depend on the entire interface even when using only a subset
3. **Testing Complexity**: Mock implementations must implement all 8 methods
4. **Reduced Clarity**: The interface's purpose is unclear from its signature

### Current Interface

```go
type Runtime interface {
    RegisterTools(registry *tools.Registry)           // Tool registration
    NotificationSender() NotificationSender           // Notifications
    ApprovalHandler() security.ApprovalHandler        // Approvals
    SessionStorage() session.Storage                  // Session persistence
    SessionID() string                                // Session identification
    SupportsTerminals() bool                          // Terminal capability check
    TerminalClient() TerminalClient                   // Terminal access
}
```

### Affected Files

- `internal/agent/runtime/runtime.go:33-64` - Interface definition
- `internal/agent/runtime/builtin.go` - BuiltinRuntime implementation
- `internal/agent/runtime/acp.go` - ACPRuntime implementation
- `internal/conversation/builder.go:35` - Uses Runtime for multiple concerns
- `internal/conversation/agent.go` - Uses Runtime.RegisterTools()
- `internal/agent/builder.go:24,165` - Stores and uses Runtime
- `cmd/spin/acp.go:239,261` - Creates runtime and uses RegisterTools

## Solution

Split the `Runtime` interface into five focused interfaces following ISP:

### 1. ToolRegistrar

```go
// ToolRegistrar provides tool registration capability.
type ToolRegistrar interface {
    // RegisterTools registers runtime-specific tools to the registry.
    RegisterTools(registry *tools.Registry)
}
```

### 2. NotificationProvider

```go
// NotificationProvider provides notification sending capability.
type NotificationProvider interface {
    // NotificationSender returns the notification sender for this runtime.
    NotificationSender() NotificationSender
}
```

### 3. ApprovalProvider

```go
// ApprovalProvider provides approval handling capability.
type ApprovalProvider interface {
    // ApprovalHandler returns the approval handler for this runtime.
    ApprovalHandler() security.ApprovalHandler
}
```

### 4. SessionProvider

```go
// SessionProvider provides session management capability.
type SessionProvider interface {
    // SessionStorage returns the session storage for persistence.
    SessionStorage() session.Storage
    // SessionID returns the current session ID.
    SessionID() string
}
```

### 5. TerminalProvider

```go
// TerminalProvider provides terminal capability.
type TerminalProvider interface {
    // SupportsTerminals returns whether this runtime supports terminal protocol.
    SupportsTerminals() bool
    // TerminalClient returns the terminal client if supported, nil otherwise.
    TerminalClient() TerminalClient
}
```

### Composite Interface (Backward Compatibility)

```go
// Runtime combines all runtime capabilities.
// Consumers should prefer using specific interfaces when possible.
type Runtime interface {
    ToolRegistrar
    NotificationProvider
    ApprovalProvider
    SessionProvider
    TerminalProvider
}
```

## Impact Analysis

### Consumers and Required Interfaces

| Consumer | Required Interfaces |
|----------|---------------------|
| `conversation/builder.go` | `Runtime` (full) |
| `conversation/agent.go` | `ToolRegistrar` |
| `agent/builder.go` | `ApprovalProvider`, `ToolRegistrar` |
| `cmd/spin/acp.go` | `ToolRegistrar` |

### Implementation Updates

Both `BuiltinRuntime` and `ACPRuntime` already implement all methods, so they will automatically satisfy the composite `Runtime` interface. No changes needed to implementations.

## Acceptance Criteria

1. [ ] Five focused interfaces defined in `internal/agent/runtime/runtime.go`
2. [ ] Composite `Runtime` interface maintained for backward compatibility
3. [ ] All existing tests pass without modification
4. [ ] Test coverage >= 90% for new interface code
5. [ ] `make lint` passes with zero errors
6. [ ] Documentation updated in `docs/investigations/ARCHITECTURE.md`
7. [ ] No dead code introduced

## Test Plan

### Unit Tests

1. **Interface Satisfaction Tests**
   - Verify `BuiltinRuntime` implements all five interfaces
   - Verify `ACPRuntime` implements all five interfaces
   - Verify both implement composite `Runtime`

2. **Interface Segregation Tests**
   - Test that consumers can depend on specific interfaces
   - Test type assertion from `Runtime` to specific interfaces

### Integration Tests

1. Existing `conversation/builder_test.go` tests pass
2. Existing `agent/agent_test.go` tests pass
3. Existing `protocol/acp/*_test.go` tests pass

## Implementation Steps

1. Add five focused interface definitions above existing `Runtime` interface
2. Redefine `Runtime` as composite interface embedding all five
3. Add interface satisfaction tests
4. Run `make lint` and fix any issues
5. Run all tests to verify backward compatibility
6. Update ARCHITECTURE.md documentation

## Rollback Plan

If issues arise, revert to single `Runtime` interface by removing the embedded interfaces and restoring the inline method signatures. The composite interface ensures this is a non-breaking change.

## References

- ROADMAP.md Section 2.4
- Interface Segregation Principle: https://en.wikipedia.org/wiki/Interface_segregation_principle
- Go Interface Best Practices: https://go.dev/doc/effective_go#interfaces
