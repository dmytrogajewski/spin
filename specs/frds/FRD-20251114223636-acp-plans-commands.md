# Feature Requirements Document: ACP Plans and Commands Implementation

**Feature ID**: FRD-20251114223636  
**Feature**: 9.2 - Plans and Commands Implementation  
**Date**: 2025-11-14  
**Status**: In Progress

## Overview

Integrate Spin's planning system (`orchestration.Plan`) with ACP protocol to send structured plan notifications. Enhance the existing text-based plan detection (Feature 4.2) with full integration to the orchestration planning system, providing dependency tracking, status updates, and proper plan structure.

## Background

Spin has two planning systems:
1. **`internal/orchestration/plan.go`** - Full-featured planning system with:
   - `Plan` struct with steps, dependencies, status tracking
   - `Step` struct with execution state, dependencies, results
   - Plan execution logic, topological sorting, cycle detection
2. **`internal/agent/request.go`** - Simpler `agent.Plan` type used by `Agent.CreatePlan()`

Feature 4.2 implemented basic text-based plan detection that sends `UpdatePlan` notifications. Feature 9.2 should integrate with the full planning system when available.

ACP protocol supports:
- `PlanEntry` with `content`, `priority`, `status`
- `UpdatePlan` notification for plan updates
- `AvailableCommandsUpdate` notification for slash commands (if supported)

## Requirements

### Functional Requirements

1. **Plan Integration**
   - Detect when `orchestration.Plan` is available in agent execution
   - Convert `orchestration.Plan` → `acp.PlanEntry[]`
   - Convert `orchestration.Step` → `acp.PlanEntry`
   - Map step status to ACP plan entry status
   - Map step priority/importance to ACP plan entry priority
   - Send plan notifications when plan is created/updated

2. **Status Updates**
   - Track plan execution status
   - Send plan updates when step status changes
   - Include dependency information in plan entries

3. **Fallback to Text Detection**
   - Keep existing text-based plan detection as fallback
   - Use when `orchestration.Plan` is not available
   - Ensure backward compatibility

4. **Commands Support (Optional)**
   - Review Spin's slash command system
   - Implement `available_commands_update` notification if commands are supported
   - Advertise available commands on session creation

### Technical Requirements

1. **Plan Conversion**
   - Create converter: `orchestration.Plan` → `acp.PlanEntry[]`
   - Map `orchestration.StepStatus` → `acp.PlanEntryStatus`
   - Map step description/action → `acp.PlanEntry.Content`
   - Determine priority from step dependencies or metadata

2. **Status Mapping**
   - `StepStatusPending` → `PlanEntryStatusPending`
   - `StepStatusRunning` → `PlanEntryStatusInProgress` (or appropriate status)
   - `StepStatusCompleted` → `PlanEntryStatusCompleted`
   - `StepStatusFailed` → `PlanEntryStatusFailed`
   - `StepStatusSkipped` → `PlanEntryStatusCancelled` (or appropriate status)

3. **Plan Detection**
   - Check if agent response includes `orchestration.Plan`
   - Check if `OrchestrationService` has active planner
   - Fall back to text-based detection if no structured plan

4. **Testing**
   - Unit tests for plan conversion
   - Unit tests for status mapping
   - Integration tests for plan notifications
   - Tests for fallback to text detection

## Design

### Architecture

```
Agent Execution
    │
    ├─ Creates orchestration.Plan (if planning enabled)
    │   │
    │   └─ OrchestrationService.SetPlanner(plan)
    │
    └─ AgentResponse
        │
        ├─ Check for orchestration.Plan
        │   └─ Convert to acp.PlanEntry[]
        │
        └─ Fallback: Text-based detection (Feature 4.2)
            └─ detectPlanFromOutput()
```

### Plan Conversion

```go
func convertOrchestrationPlanToACP(plan *orchestration.Plan) []acp.PlanEntry {
    entries := make([]acp.PlanEntry, 0, len(plan.Steps))
    for _, step := range plan.Steps {
        entry := acp.PlanEntry{
            Content:  buildStepContent(step),
            Priority: mapStepPriority(step),
            Status:   mapStepStatus(step.Status),
        }
        entries = append(entries, entry)
    }
    return entries
}
```

### Status Mapping

| orchestration.StepStatus | acp.PlanEntryStatus |
|--------------------------|---------------------|
| StepStatusPending        | PlanEntryStatusPending |
| StepStatusReady          | PlanEntryStatusPending |
| StepStatusRunning        | PlanEntryStatusInProgress |
| StepStatusCompleted      | PlanEntryStatusCompleted |
| StepStatusFailed         | PlanEntryStatusFailed |
| StepStatusSkipped        | PlanEntryStatusCancelled |

### Priority Mapping

- Steps with no dependencies → `PlanEntryPriorityHigh` (can start immediately)
- Steps with dependencies → `PlanEntryPriorityMedium` (normal priority)
- Steps that are prerequisites for many others → `PlanEntryPriorityHigh` (critical path)

### Implementation Details

1. **Plan Detection**
   - Check `AgentResponse` for plan information
   - Check `OrchestrationService.GetPlanner()` for active plan
   - Use text-based detection as fallback

2. **Plan Conversion**
   - Convert each step to `acp.PlanEntry`
   - Include step description and action in content
   - Map status and priority appropriately

3. **Notification Sending**
   - Send plan notification when plan is created
   - Send plan updates when step status changes
   - Use existing `sendPlanNotifications()` method

4. **Commands (Future)**
   - Review Spin's command system
   - Implement if commands are available
   - Defer if not available

## Acceptance Criteria

- [ ] `orchestration.Plan` is detected when available
- [ ] Plan is converted to `acp.PlanEntry[]` correctly
- [ ] Step status is mapped correctly
- [ ] Step priority is determined appropriately
- [ ] Plan notifications are sent when plan is created
- [ ] Plan updates are sent when step status changes
- [ ] Text-based detection still works as fallback
- [ ] Unit tests cover all conversion scenarios
- [ ] Integration tests verify notifications
- [ ] Documentation updated

## Testing Strategy

### Unit Tests

1. **Plan Conversion**
   - Test `convertOrchestrationPlanToACP()` with various plans
   - Test status mapping for all step statuses
   - Test priority determination
   - Test with plans containing dependencies

2. **Status Mapping**
   - Test each `StepStatus` → `PlanEntryStatus` mapping
   - Test edge cases (empty plan, single step, etc.)

3. **Fallback**
   - Test that text-based detection still works
   - Test when no plan is available

### Integration Tests

1. **Plan Notification Flow**
   - Create plan via agent
   - Verify plan notification is sent
   - Update step status
   - Verify plan update notification is sent

## Dependencies

- `github.com/coder/acp-go-sdk` v0.6.3
- `internal/orchestration/plan.go` - Planning system
- `internal/agent/agent.go` - Agent execution
- `internal/protocol/acp/notifications.go` - Existing plan detection

## Risks

1. **Plan Availability**: May not always have structured plan
   - Mitigation: Keep text-based detection as fallback

2. **Status Tracking**: Need to track plan execution
   - Mitigation: Use event system or check plan status periodically

3. **Priority Determination**: No explicit priority in `orchestration.Step`
   - Mitigation: Infer from dependencies or use default priority

## Notes

- Feature 4.2 already implements text-based plan detection
- This feature enhances it with structured plan integration
- Commands support is optional and can be deferred if not available
- Plan status updates may require event system integration or polling

