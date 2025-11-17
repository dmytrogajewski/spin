# FRD: ACP Prompt Method Implementation

**ID**: FRD-20251114030000
**Status**: Implementation
**Created**: 2025-11-14
**Roadmap**: [P4.1] Implement Prompt Method
**Phase**: 4 - Prompt Processing Implementation
**Priority**: HIGH
**Estimated Effort**: 4-6 hours

## Overview

Implement the `acp.Agent.Prompt()` method to process user prompts and execute the agent loop, converting ACP content blocks to Spin messages and streaming updates via ACP notifications.

## Problem Statement

The ACP agent currently has a stub `Prompt()` method that returns "not implemented". We need to:

1. Parse ACP `PromptRequest` with `ContentBlock[]`
2. Convert ACP content blocks to Spin messages
3. Validate session ID exists
4. Execute agent with proper context cancellation
5. Stream updates via `connection.SendUpdate()` during execution
6. Return appropriate stop reason on completion

## Goals

1. Implement `Prompt()` method that processes ACP prompt requests
2. Convert ACP `ContentBlock[]` to Spin `message.Message[]`
3. Integrate with existing `agent.Execute()` flow
4. Support session ID validation
5. Stream real-time updates via ACP notifications (basic implementation, full converter in Feature 4.2)
6. Return appropriate stop reasons

## Non-Goals

- Full event-to-notification conversion (Feature 4.2)
- Stop reason detection and conversion (Feature 4.3)
- Permission request handling (Feature 7.1)
- Cancellation handling (Feature 5.1)

## Design

### 1. Method Signature

```go
func (a *SpinACPAgent) Prompt(ctx context.Context, req acp.PromptRequest) (acp.PromptResponse, error)
```

### 2. Content Block Conversion

Convert ACP `ContentBlock[]` to Spin messages:

- **Text blocks** → `message.Message` with text content
- **Resource links** → `message.Message` with file reference (extract URI)
- **Embedded resources** → `message.Message` with embedded content
- **Images** → Not yet supported (will return error or skip)

**Conversion Pattern**:
```go
func convertACPContentBlocksToMessages(blocks []acp.ContentBlock) ([]message.Message, error) {
    var messages []message.Message
    for _, block := range blocks {
        if block.Text != nil {
            messages = append(messages, message.Message{
                Role:    message.RoleUser,
                Content: block.Text.Text,
            })
        } else if block.ResourceLink != nil {
            // Extract file path from URI
            path := extractPathFromURI(block.ResourceLink.Uri)
            messages = append(messages, message.Message{
                Role:    message.RoleUser,
                Content: fmt.Sprintf("File: %s", path),
            })
        }
        // Handle other block types
    }
    return messages, nil
}
```

### 3. Session Validation

Check if session exists in `a.sessions` map:

```go
a.mu.RLock()
sess, exists := a.sessions[req.SessionId]
a.mu.RUnlock()
if !exists {
    return acp.PromptResponse{}, fmt.Errorf("session not found: %s", req.SessionId)
}
```

### 4. Agent Execution

Use existing `agent.Execute()` method:

```go
agentReq := &agent.AgentRequest{
    Input:   extractTextFromMessages(messages),
    Task:    task.DefaultTask(), // Use default task mode for now
    History: []message.Message{}, // No history for now (will be added in LoadSession)
}

resp, err := a.agent.Execute(ctx, agentReq)
```

### 5. Basic Event Streaming

Subscribe to event emitter and send basic notifications:

```go
subID, eventChan, err := a.emitter.Subscribe()
defer a.emitter.Unsubscribe(subID)

go func() {
    for event := range eventChan {
        // Basic conversion (full converter in Feature 4.2)
        if event.Type == events.EventContentDelta {
            if data, ok := event.ContentDeltaData(); ok {
                update := acp.SessionNotification{
                    SessionId: req.SessionId,
                    Update:    acp.UpdateAgentMessageText(data.Content),
                }
                // Send via connection (will be wired in Feature 8.1)
                // For now, we'll prepare the notification structure
            }
        }
    }
}()
```

### 6. Stop Reason

Map `AgentResponse.FinishReason` to ACP `StopReason`:

- Normal completion → `acp.StopReasonEndTurn`
- Token limit → `acp.StopReasonMaxTokens`
- Turn limit → `acp.StopReasonMaxTurnRequests`
- Error → `acp.StopReasonEndTurn` (with error)
- Cancellation → `acp.StopReasonCancelled` (handled in Feature 5.1)

**Basic mapping** (full conversion in Feature 4.3):
```go
func mapStopReason(finishReason string) acp.StopReason {
    switch finishReason {
    case "max_tokens":
        return acp.StopReasonMaxTokens
    case "max_turns":
        return acp.StopReasonMaxTurnRequests
    case "cancelled":
        return acp.StopReasonCancelled
    default:
        return acp.StopReasonEndTurn
    }
}
```

## Implementation Steps

1. **Create content block converter** (`convertACPContentBlocksToMessages`)
2. **Add session validation** in `Prompt()` method
3. **Implement basic Prompt() method**:
   - Parse request
   - Validate session
   - Convert content blocks
   - Execute agent
   - Return response
4. **Add basic event subscription** (prepare for Feature 4.2)
5. **Map stop reason** (basic implementation)
6. **Write comprehensive tests**

## Testing Strategy

### Unit Tests

1. **Content Block Conversion**:
   - Text blocks → messages
   - Resource links → messages
   - Empty blocks → error
   - Unsupported blocks → error or skip

2. **Session Validation**:
   - Valid session ID → success
   - Invalid session ID → error
   - Missing session ID → error

3. **Agent Execution**:
   - Successful execution → correct stop reason
   - Agent error → appropriate error handling
   - Context cancellation → cancelled stop reason

4. **Stop Reason Mapping**:
   - All stop reason types mapped correctly

### Integration Tests

1. **Full Prompt Flow**:
   - Create session
   - Send prompt with text blocks
   - Verify agent execution
   - Verify stop reason

2. **Event Streaming**:
   - Verify events are emitted
   - Verify notifications are prepared (full sending in Feature 4.2)

## Acceptance Criteria

- ✅ `Prompt()` method implemented
- ✅ Content blocks converted correctly
- ✅ Session validation works
- ✅ Agent execution integrated
- ✅ Basic event streaming prepared
- ✅ Stop reasons returned correctly
- ✅ Unit tests (≥90% coverage)
- ✅ Integration tests
- ✅ No lint errors
- ✅ No deadcode

## Dependencies

- Feature 3.1 (NewSession) - ✅ Completed
- SDK types (`acp.PromptRequest`, `acp.PromptResponse`, `acp.ContentBlock`)
- `internal/agent.Agent.Execute()`
- `internal/events.EventEmitter`
- `internal/message.Message`

## References

- [ACP Roadmap](../../specs/acp/ROADMAP.md#feature-41-implement-prompt-method)
- [ACP Type Mapping](../../docs/packages/acp-type-mapping.md)
- [ACP SDK Integration](../../docs/packages/acp-sdk-integration.md)
- [ACP Specification](../../specs/acp/specification.md)

## Notes

- Event-to-notification conversion will be enhanced in Feature 4.2
- Stop reason detection will be enhanced in Feature 4.3
- Connection for sending notifications will be wired in Feature 8.1
- For now, we'll prepare notification structures but not send them (connection not available yet)

