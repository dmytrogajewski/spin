# Feature Requirements Document: ACP RequestPermission Method

**Feature ID**: FRD-20251114100600  
**Feature**: 7.1 - Implement RequestPermission Method  
**Date**: 2025-11-14  
**Status**: In Progress

## Overview

Implement the `acp.Agent.RequestPermission()` method to handle permission requests from the ACP client. This method integrates with Spin's existing `ApprovalService` to provide a unified approval system.

## Background

The ACP protocol allows agents to request user permission before executing potentially dangerous operations (e.g., file writes, shell commands). The `RequestPermission` method bridges ACP permission requests with Spin's internal approval system.

## Requirements

### Functional Requirements

1. **RequestPermission Method Implementation**
   - Implement `RequestPermission(ctx context.Context, req acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error)`
   - Parse ACP `RequestPermissionRequest` from SDK
   - Extract tool call information from `RequestPermissionToolCall`
   - Convert to Spin `security.Operation`
   - Call `ApprovalService.RequestApproval()`
   - Convert `security.ApprovalResponse` to ACP `RequestPermissionResponse`

2. **Tool Call Information Extraction**
   - Extract tool name from `RequestPermissionToolCall.Title`
   - Extract tool ID from `RequestPermissionToolCall.ToolCallId`
   - Extract tool parameters from `RequestPermissionToolCall.RawInput`
   - Extract work directory from session (if available)

3. **Permission Options Mapping**
   - Map ACP permission options to approval decisions
   - Support `allow_once`, `allow_always`, `deny` option kinds
   - Handle option selection in response

4. **Response Conversion**
   - Convert `ApprovalResponse.Approved` → `RequestPermissionOutcome.Selected` with `optionId`
   - Convert `ApprovalResponse` denial → `RequestPermissionOutcome.Selected` with deny option
   - Handle context cancellation → `RequestPermissionOutcome.Cancelled`

### Technical Requirements

1. **Integration with ApprovalService**
   - Use existing `ApprovalService` if available in `SpinACPAgent`
   - Handle case where `ApprovalService` is not configured (return error)
   - Support context cancellation propagation

2. **Session Management**
   - Validate session exists
   - Extract work directory from session
   - Handle session not found errors

3. **Error Handling**
   - Handle missing approval service
   - Handle invalid tool call information
   - Handle session not found
   - Handle context cancellation
   - Return appropriate ACP errors

## Design

### Architecture

```
ACP RequestPermissionRequest
    ↓
Extract tool call info (title, toolCallId, rawInput)
    ↓
Get session work directory
    ↓
Convert to security.Operation
    ↓
Call ApprovalService.RequestApproval()
    ↓
Convert ApprovalResponse to ACP RequestPermissionResponse
    ↓
Return response
```

### Implementation Details

1. **Tool Call to Operation Conversion**
   - Function: `convertToolCallToOperation(toolCall acp.RequestPermissionToolCall, sessionID acp.SessionId, sessions map[acp.SessionId]*session.Session) (security.Operation, error)`
   - Extract tool name from `toolCall.Title`
   - Extract work directory from session
   - Create `security.Command` from tool call info
   - Create `security.Operation` with command, reason, and work directory

2. **Permission Options Handling**
   - Parse options from `RequestPermissionRequest.Options`
   - Map option kinds to approval decisions:
     - `allow_once` → approve for this operation only
     - `allow_always` → approve and remember (future enhancement)
     - `deny` → deny operation
   - Store selected option ID in response

3. **Response Conversion**
   - Function: `convertApprovalResponseToACP(resp security.ApprovalResponse, options []acp.PermissionOption) (acp.RequestPermissionResponse, error)`
   - If approved: Find matching option and return `Selected` outcome
   - If denied: Find deny option and return `Selected` outcome
   - If cancelled: Return `Cancelled` outcome

4. **ApprovalService Integration**
   - Add `approvalService *security.ApprovalService` field to `SpinACPAgent`
   - Update constructor to accept optional `ApprovalService`
   - Use `ApprovalService` in `RequestPermission` method

## Acceptance Criteria

- [ ] `RequestPermission()` method implemented
- [ ] Parses ACP `RequestPermissionRequest` correctly
- [ ] Extracts tool call information
- [ ] Converts to Spin `security.Operation`
- [ ] Calls `ApprovalService.RequestApproval()`
- [ ] Converts response to ACP format
- [ ] Handles permission options correctly
- [ ] Handles context cancellation
- [ ] Handles missing approval service
- [ ] Handles session not found
- [ ] Unit tests cover all scenarios
- [ ] Integration tests cover end-to-end flow

## Testing Strategy

### Unit Tests

1. **RequestPermission Method**
   - Test successful approval flow
   - Test denial flow
   - Test context cancellation
   - Test missing approval service
   - Test session not found
   - Test invalid tool call information

2. **Tool Call Conversion**
   - Test conversion from ACP tool call to Spin operation
   - Test work directory extraction from session
   - Test tool name extraction
   - Test parameter extraction

3. **Response Conversion**
   - Test approved response conversion
   - Test denied response conversion
   - Test cancelled response conversion
   - Test option ID mapping

### Integration Tests

1. **End-to-End Flow**
   - Test full approval flow with real ApprovalService
   - Test denial flow
   - Test cancellation flow
   - Test with different permission options

## Dependencies

- `github.com/coder/acp-go-sdk` - ACP SDK types
- `internal/security` - ApprovalService, Operation, Command
- `internal/session` - Session management
- `internal/events` - Event emission (via ApprovalService)

## Risks

1. **ApprovalService Integration**: May need to adapt ApprovalService interface
   - Mitigation: Use existing interface, add optional field to SpinACPAgent

2. **Option Mapping**: ACP options may not map cleanly to Spin approval decisions
   - Mitigation: Map common options (allow_once, deny), defer complex mappings

3. **Session Work Directory**: May not always be available
   - Mitigation: Use default or return error if required

## Notes

- ApprovalService is optional - if not configured, return error
- Permission options are provided by client - agent presents them to user
- `allow_always` option may require future enhancement for persistent approval storage
- Tool call information extraction is basic - may need enhancement for complex tool calls

