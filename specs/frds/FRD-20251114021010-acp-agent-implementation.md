# FRD: Create ACP Agent Implementation

**ID**: FRD-20251114021010  
**Feature**: 2.1 - Create ACP Agent Implementation  
**Status**: In Progress  
**Created**: 2025-11-14  
**Roadmap**: [specs/acp/ROADMAP.md](../../specs/acp/ROADMAP.md)

## Overview

Create the `SpinACPAgent` struct that implements the `acp.Agent` interface. This is the core adapter that connects Spin's existing components to the ACP protocol.

## Goals

1. Create `SpinACPAgent` struct implementing `acp.Agent`
2. Hold references to all required Spin components
3. Implement constructor with proper initialization
4. Write comprehensive tests
5. Document the implementation

## Requirements

### Functional Requirements

1. **Struct Definition**
   - Define `SpinACPAgent` struct in `internal/protocol/acp/agent.go`
   - Implement `acp.Agent` interface (6 methods)
   - Hold references to Spin components:
     - `*agent.Agent` - Core agent execution
     - Session management (need to determine structure)
     - `*mcp.MCPManager` - MCP integration
     - `*events.EventEmitter` - Event emission
     - Connection for sending notifications (to be added later)

2. **Constructor**
   - `NewSpinACPAgent()` function
   - Accept all required Spin components
   - Validate inputs (no nil checks)
   - Initialize struct properly
   - Return error on validation failure

3. **Interface Stubs**
   - All 6 `acp.Agent` methods as stubs (return errors for now)
   - Methods: `Initialize`, `NewSession`, `Prompt`, `Cancel`, `SetSessionMode`, `Authenticate`
   - Proper method signatures matching SDK interface

### Non-Functional Requirements

1. **Testing**
   - Unit tests for struct creation
   - Constructor validation tests
   - Interface implementation verification
   - Minimum 90% coverage

2. **Documentation**
   - Godoc comments on struct and constructor
   - Package documentation
   - Update `docs/packages/protocol-acp.md`

## Design

### Struct Definition

```go
type SpinACPAgent struct {
    agent      *agent.Agent
    sessionMgr SessionManager // TBD: need to check session manager structure
    mcpManager *mcp.MCPManager
    emitter    *events.EventEmitter
    // Connection will be added later when we implement connection setup
}
```

### Constructor

```go
func NewSpinACPAgent(
    agent *agent.Agent,
    sessionMgr SessionManager,
    mcpManager *mcp.MCPManager,
    emitter *events.EventEmitter,
) (*SpinACPAgent, error) {
    // Validation
    // Initialization
}
```

### Interface Methods (Stubs)

All methods will be stubs that return "not implemented" errors initially:

```go
func (a *SpinACPAgent) Initialize(ctx context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error) {
    return acp.InitializeResponse{}, errors.New("not implemented")
}
// ... other methods
```

## Implementation Plan

### Phase 1: Struct and Constructor (TDD)
1. Write test for struct creation
2. Write test for constructor validation
3. Implement struct definition
4. Implement constructor
5. Verify tests pass

### Phase 2: Interface Stubs (TDD)
1. Write test verifying interface implementation
2. Implement all 6 method stubs
3. Verify interface compliance

### Phase 3: Documentation
1. Add Godoc comments
2. Create/update package documentation
3. Update roadmap

## Testing Strategy

### Unit Tests

- `TestNewSpinACPAgent` - Constructor success
- `TestNewSpinACPAgent_Validation` - Nil checks
- `TestSpinACPAgent_ImplementsInterface` - Interface verification
- `TestSpinACPAgent_MethodStubs` - All methods return errors

### Test Coverage

- Constructor: 100%
- Validation: 100%
- Interface compliance: 100%

## Acceptance Criteria

- [ ] `SpinACPAgent` struct defined
- [ ] Implements `acp.Agent` interface (compiles)
- [ ] Constructor validates inputs
- [ ] All 6 methods exist as stubs
- [ ] Unit tests written and passing
- [ ] Coverage ≥90%
- [ ] Documentation complete

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Session manager structure unclear | Medium | Medium | Check session package for manager interface |
| Component dependencies complex | Low | Low | Use dependency injection pattern |
| Interface changes in SDK | Low | Low | SDK is stable, pin version |

## Dependencies

### External Dependencies
- `github.com/coder/acp-go-sdk` v0.6.3 (already added)

### Internal Dependencies
- `internal/agent` - Agent struct
- `internal/session` - Session management
- `internal/mcp` - MCP manager
- `internal/events` - Event emitter

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)

## Notes

- This is the foundation for all ACP protocol methods
- Methods will be implemented in subsequent features
- Focus on structure and validation, not functionality yet

