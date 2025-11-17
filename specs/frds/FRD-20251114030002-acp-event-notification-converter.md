# FRD: ACP Event to Notification Converter

**ID**: FRD-20251114030002
**Status**: Implementation
**Created**: 2025-11-14
**Roadmap**: [P4.2] Event to ACP Notification Converter
**Phase**: 4 - Prompt Processing Implementation
**Priority**: HIGH
**Estimated Effort**: 4-5 hours

## Overview

Create a converter that transforms Spin events to ACP `session/update` notifications and sends them via the ACP connection in real-time during agent execution.

## Problem Statement

The ACP protocol requires real-time notifications during prompt execution to keep clients informed about agent progress, tool calls, and content generation. Currently, the `Prompt()` method executes the agent but doesn't stream updates to the client.

## Goals

1. Convert Spin events to ACP SessionUpdate notifications
2. Subscribe to EventEmitter during Prompt execution
3. Send notifications in real-time via connection.SessionUpdate()
4. Handle all relevant event types (content, tool calls, turns)
5. Thread-safe notification sending
6. Support conversation history replay for LoadSession

## Non-Goals

- Enhanced tool call notifications with diffs (Feature 6.1)
- Plan notifications (Feature 9.2)
- Advanced content types (Feature 9.3)

## Design

### 1. Connection Integration

The `AgentSideConnection` is created in `cmd/spin/acp.go` after the agent. We need to:
- Add optional `connection *acp.AgentSideConnection` field to `SpinACPAgent`
- Add `SetConnection(conn *acp.AgentSideConnection)` method
- Use connection in `Prompt()` to send notifications

### 2. Event Subscription

In `Prompt()` method:
1. Subscribe to EventEmitter before agent execution
2. Start goroutine to process events
3. Convert each event to ACP notification
4. Send via `connection.SessionUpdate()`
5. Unsubscribe when prompt completes

### 3. Event to Notification Mapping

| Spin Event | ACP Notification | Helper Function |
|------------|------------------|-----------------|
| `EventContentDelta` | `agent_message_chunk` | `acp.UpdateAgentMessageText()` |
| `EventToolCallStart` | `tool_call` | `acp.StartToolCall()` |
| `EventToolCallProgress` | `tool_call_update` (in_progress) | `acp.UpdateToolCall()` |
| `EventToolCallComplete` | `tool_call_update` (completed/failed) | `acp.UpdateToolCall()` |
| `EventTurnStart` | (optional) `user_message_chunk` | `acp.UpdateUserMessageText()` |

### 4. Notification Converter

Create `internal/protocol/acp/notifications.go` with:
- `convertEventToSessionUpdate(event events.Event) (acp.SessionUpdate, bool)` - converts event, returns false if not applicable
- Helper functions for each event type conversion
- Tool call ID mapping (Spin tool ID → ACP ToolCallId)

### 5. Thread Safety

- Use mutex to protect connection access
- Use buffered channel for event processing
- Handle connection nil case gracefully

### 6. Error Handling

- Log notification send errors but don't fail prompt execution
- Handle connection unavailability gracefully
- Continue processing even if some notifications fail

## Implementation Steps

1. **Add connection field to SpinACPAgent**
   - Add optional `connection *acp.AgentSideConnection` field
   - Add `SetConnection()` method
   - Protect with mutex

2. **Create notifications.go**
   - Implement `convertEventToSessionUpdate()`
   - Implement event-specific converters
   - Handle tool call ID mapping

3. **Integrate with Prompt()**
   - Subscribe to EventEmitter
   - Start event processing goroutine
   - Send notifications during execution
   - Unsubscribe on completion

4. **Update cmd/spin/acp.go**
   - Call `SetConnection()` after creating connection

5. **Write comprehensive tests**

## Testing Strategy

### Unit Tests

1. **Event Conversion**:
   - ContentDelta → agent_message_chunk
   - ToolCallStart → tool_call
   - ToolCallProgress → tool_call_update (in_progress)
   - ToolCallComplete → tool_call_update (completed/failed)
   - Unknown events → returns false

2. **Connection Integration**:
   - SetConnection stores connection
   - SessionUpdate called with correct params
   - Errors logged but don't fail execution

3. **Thread Safety**:
   - Concurrent notification sending
   - Connection nil handling

### Integration Tests

1. **Full Flow**:
   - Execute Prompt with event subscription
   - Verify notifications sent
   - Verify all event types converted

2. **Error Handling**:
   - Connection unavailable
   - Notification send failures
   - Event processing errors

## Acceptance Criteria

- ✅ Converts all relevant events to ACP notifications
- ✅ Sends notifications in real-time during execution
- ✅ Handles event stream correctly
- ✅ Thread-safe notification sending
- ✅ Streams updates during execution
- ✅ Unit tests (≥90% coverage)
- ✅ Integration tests
- ✅ No lint errors
- ✅ No deadcode

## Dependencies

- Feature 4.1 (Prompt) - ✅ Completed
- SDK SessionUpdate types and helpers
- EventEmitter subscription mechanism
- AgentSideConnection.SessionUpdate() method

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md#feature-42-event-to-acp-notification-converter)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [Event System](../../docs/packages/events.md)

## Notes

- Connection is optional - if not set, notifications are skipped (graceful degradation)
- Notification failures don't fail prompt execution
- Tool call ID mapping uses Spin's tool ID as ACP ToolCallId
- Content deltas are accumulated per message (can be enhanced later)

