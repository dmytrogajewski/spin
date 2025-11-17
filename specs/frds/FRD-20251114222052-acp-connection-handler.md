# Feature Requirements Document: ACP Connection Handler

**Feature ID**: FRD-20251114222052  
**Feature**: 8.1 - Create ACP Connection Handler  
**Date**: 2025-11-14  
**Status**: In Progress

## Overview

Complete the ACP connection handler implementation in `cmd/spin/acp.go` to provide a fully functional ACP server entry point. This includes wiring all necessary components, ensuring ApprovalService is properly configured, and adding comprehensive tests.

## Background

The ACP connection handler serves as the main entry point for Spin when running as an ACP agent. It sets up all Spin components, creates the ACP agent adapter, establishes the connection, and handles the connection lifecycle.

## Requirements

### Functional Requirements

1. **Component Initialization**
   - Initialize all Spin components (agent, MCP manager, event emitter, approval service)
   - Create `SpinACPAgent` instance with all components
   - Wire `ApprovalService` to `SpinACPAgent` for `RequestPermission` support

2. **Connection Setup**
   - Create ACP connection using `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)`
   - Set connection on agent for sending notifications
   - Handle connection lifecycle (start, run, shutdown)

3. **Command-Line Interface**
   - Provide `spin acp` command
   - Support command-line flags for configuration
   - Handle graceful shutdown on signals

4. **Error Handling**
   - Handle component creation errors
   - Handle connection errors
   - Provide clear error messages

### Technical Requirements

1. **ApprovalService Integration**
   - Create `ApprovalService` during component initialization
   - Set `ApprovalService` on `SpinACPAgent` via `SetApprovalService()`
   - Ensure `RequestPermission` method works correctly

2. **Component Wiring**
   - All components properly initialized
   - Dependencies correctly passed
   - No missing connections

3. **Testing**
   - Unit tests for component creation
   - Integration tests for connection setup
   - Error handling tests

## Design

### Architecture

```
spin acp command
    ↓
Initialize components:
  - LLM provider
  - Agent
  - MCP manager
  - Event emitter
  - Approval service
    ↓
Create SpinACPAgent
    ↓
Set ApprovalService
    ↓
Create ACP connection
    ↓
Set connection on agent
    ↓
Wait for connection lifecycle
```

### Implementation Details

1. **Component Creation**
   - `createProviderForACP()`: Creates LLM provider
   - `createACPComponents()`: Creates all Spin components
   - `createAgentComponents()`: Creates agent, emitter, approval service

2. **Connection Lifecycle**
   - Connection starts automatically when created
   - Waits on `conn.Done()` or context cancellation
   - Handles graceful shutdown

3. **Signal Handling**
   - SIGINT/SIGTERM trigger graceful shutdown
   - Context cancellation propagates to all components

## Acceptance Criteria

- [ ] All components initialized correctly
- [ ] ApprovalService wired to SpinACPAgent
- [ ] Connection established successfully
- [ ] Notifications can be sent via connection
- [ ] RequestPermission works with ApprovalService
- [ ] Graceful shutdown on signals
- [ ] Unit tests cover component creation
- [ ] Integration tests cover connection setup
- [ ] Error handling tested
- [ ] Documentation updated

## Testing Strategy

### Unit Tests

1. **Component Creation**
   - Test provider creation (ollama, openai)
   - Test component creation functions
   - Test error handling

2. **Connection Setup**
   - Test connection creation
   - Test approval service wiring
   - Test connection setting on agent

### Integration Tests

1. **End-to-End Flow**
   - Test full server startup
   - Test connection lifecycle
   - Test signal handling

## Dependencies

- `github.com/coder/acp-go-sdk` - ACP SDK
- `internal/protocol/acp` - ACP agent implementation
- `internal/agent` - Core agent
- `internal/security` - Approval service
- `internal/events` - Event emitter
- `internal/mcp` - MCP manager

## Risks

1. **Component Dependencies**: Complex dependency chain
   - Mitigation: Follow existing patterns from other commands

2. **Connection Lifecycle**: May need to handle edge cases
   - Mitigation: Use SDK's built-in lifecycle management

3. **Testing**: Integration tests may be complex
   - Mitigation: Start with unit tests, add integration tests incrementally

## Notes

- ApprovalService handler is nil initially (no interactive approval in ACP mode)
- MCP is disabled by default (can be enabled via configuration)
- Connection uses stdio for communication
- All existing components are reused from other command implementations

