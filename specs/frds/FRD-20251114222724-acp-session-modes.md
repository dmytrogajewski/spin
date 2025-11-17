# Feature Requirements Document: ACP Session Modes Implementation

**Feature ID**: FRD-20251114222724  
**Feature**: 9.1 - Session Modes Implementation  
**Date**: 2025-11-14  
**Status**: In Progress

## Overview

Implement session mode support in the ACP adapter, allowing clients to query available modes, set the current mode, and receive mode update notifications. This maps Spin's task modes (regular, review, compact, planning) to ACP session modes.

## Background

Spin supports four task modes that control token budgets, tool access, and system prompts:
- **regular**: Full-featured mode with all tools (16K tokens)
- **review**: Read-only code analysis mode (12K tokens)
- **compact**: Minimal context mode for quick tasks (4K tokens)
- **planning**: Task decomposition mode with context-only tools (4K tokens)

ACP protocol supports session modes through:
- `SessionModeState` in `NewSessionResponse` (available modes and current mode)
- `SetSessionMode` method for changing modes
- `CurrentModeUpdate` notification for mode changes

## Requirements

### Functional Requirements

1. **Mode Mapping**
   - Map Spin task modes to ACP `SessionMode` objects
   - Each mode must have unique `SessionModeId`, `Name`, and `Description`
   - Default mode: "regular"

2. **Session Mode Storage**
   - Store current mode per ACP session
   - Initialize with default mode ("regular") on session creation
   - Track mode changes

3. **NewSessionResponse Enhancement**
   - Include `SessionModeState` in `NewSessionResponse`
   - List all available modes in `AvailableModes`
   - Set `CurrentModeId` to default mode

4. **SetSessionMode Implementation**
   - Validate session exists
   - Validate mode ID is in available modes
   - Update stored mode for session
   - Send `CurrentModeUpdate` notification
   - Return success response

5. **Mode Update Notifications**
   - Send `CurrentModeUpdate` notification when mode changes
   - Include `CurrentModeId` in notification

### Technical Requirements

1. **Data Structures**
   - Add `sessionModes map[acp.SessionId]acp.SessionModeId` to `SpinACPAgent`
   - Protect with mutex (use existing `mu`)

2. **Mode Definitions**
   - Define ACP `SessionMode` objects for each Spin task mode
   - Use consistent naming: "regular", "review", "compact", "planning"

3. **Error Handling**
   - Return error if session not found
   - Return error if mode ID is invalid
   - Return error if mode is not in available modes

4. **Testing**
   - Unit tests for mode mapping
   - Unit tests for `SetSessionMode` method
   - Unit tests for `NewSessionResponse` mode state
   - Integration tests for mode change notifications

## Design

### Architecture

```
ACP Client
    │
    ├─ NewSession → NewSessionResponse (includes SessionModeState)
    │
    ├─ SetSessionMode → SetSessionModeResponse
    │   │
    │   └─ CurrentModeUpdate notification
    │
    └─ SpinACPAgent
        ├─ sessionModes map[SessionId]SessionModeId
        └─ getAvailableModes() → []SessionMode
```

### Mode Mapping

| Spin Mode | ACP ModeId | ACP Name | Description |
|-----------|------------|----------|-------------|
| regular | "regular" | "Regular" | Full-featured interactive coding mode with access to all tools |
| review | "review" | "Review" | Read-only code analysis and review mode |
| compact | "compact" | "Compact" | Quick queries with minimal context and tool access |
| planning | "planning" | "Planning" | Task decomposition and planning mode with context-only tools |

### Implementation Details

1. **Mode Storage**
   ```go
   type SpinACPAgent struct {
       // ... existing fields
       sessionModes map[acp.SessionId]acp.SessionModeId
   }
   ```

2. **Mode Definitions**
   ```go
   func getAvailableModes() []acp.SessionMode {
       return []acp.SessionMode{
           {Id: "regular", Name: "Regular", Description: "Full-featured mode..."},
           {Id: "review", Name: "Review", Description: "Read-only mode..."},
           {Id: "compact", Name: "Compact", Description: "Minimal context mode..."},
           {Id: "planning", Name: "Planning", Description: "Task decomposition mode..."},
       }
   }
   ```

3. **SetSessionMode Implementation**
   - Validate session exists
   - Validate mode ID
   - Update `sessionModes` map
   - Send notification via `connection.SessionUpdate()`
   - Return response

4. **NewSessionResponse Enhancement**
   - Build `SessionModeState` with available modes
   - Set `CurrentModeId` to "regular" (default)
   - Include in response

## Acceptance Criteria

- [ ] `NewSessionResponse` includes `SessionModeState` with all available modes
- [ ] Default mode is "regular"
- [ ] `SetSessionMode` validates session and mode
- [ ] `SetSessionMode` updates stored mode
- [ ] `SetSessionMode` sends `CurrentModeUpdate` notification
- [ ] Mode changes are persisted per session
- [ ] Unit tests cover all scenarios
- [ ] Integration tests verify notifications
- [ ] Documentation updated

## Testing Strategy

### Unit Tests

1. **Mode Definitions**
   - Test `getAvailableModes()` returns all 4 modes
   - Test mode IDs, names, descriptions

2. **NewSessionResponse**
   - Test `NewSessionResponse` includes `SessionModeState`
   - Test default mode is "regular"
   - Test all modes are listed

3. **SetSessionMode**
   - Test success case (valid session, valid mode)
   - Test error: session not found
   - Test error: invalid mode ID
   - Test error: mode not in available modes
   - Test mode is stored correctly

### Integration Tests

1. **Mode Change Flow**
   - Create session
   - Call `SetSessionMode`
   - Verify notification is sent
   - Verify mode is updated

## Dependencies

- `github.com/coder/acp-go-sdk` v0.6.3
- `internal/task` - Task mode definitions
- `internal/session` - Session management

## Risks

1. **Mode Persistence**: Modes are stored in memory only
   - Mitigation: Document that modes are session-scoped and not persisted

2. **Mode Validation**: Need to ensure mode IDs match between Spin and ACP
   - Mitigation: Use constants and validation functions

3. **Notification Timing**: Mode update notification must be sent before response
   - Mitigation: Send notification synchronously before returning response

## Notes

- Modes are session-scoped (not global)
- Default mode is "regular" (matches Spin's default)
- Mode changes don't affect in-progress prompt executions
- Future: Could integrate with Spin's `Conversation.SetTaskMode()` for actual mode enforcement

