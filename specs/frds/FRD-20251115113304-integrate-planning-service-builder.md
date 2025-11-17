# FRD: Integrate PlanningService with Builder

**Feature ID**: FRD-20251115113304  
**Roadmap Feature**: 4.3  
**Priority**: P1 (HIGH)  
**Effort**: S (1 day)  
**Status**: ✅ **COMPLETE** (2025-11-15)

## Problem Statement

Currently, `PlanningService` is created directly inside `agent.NewAgent()` constructor (line 196), bypassing the `agent.Builder` pattern used for other services. This creates inconsistency:

- `SecurityService`, `DetectionService`, and `ACEService` are built via `agent.Builder` methods
- `PlanningService` is created directly in `NewAgent()`, breaking the service consolidation pattern
- This makes it harder to test, mock, and configure `PlanningService` independently

## Goals

1. **Integrate PlanningService with Builder pattern**: Add `BuildPlanningService()` to `agent.Builder`
2. **Remove PlanningService creation from NewAgent()**: Make `PlanningService` a required parameter to `NewAgent()`
3. **Update conversation.Builder**: Use `BuildPlanningService()` when constructing agents
4. **Maintain consistency**: Follow the same pattern as `BuildSecurityService()`, `BuildDetectionService()`, etc.

## Design

### Current State

**`internal/agent/agent.go` (lines 195-196)**:
```go
// Create planning service (uses same LLM provider)
planningService := planning.NewPlanningService(provider)
```

**`internal/agent/builder.go`**:
- Has `BuildSecurityService()`, `BuildDetectionService()`, `BuildACEService()`
- No `BuildPlanningService()` method

**`internal/conversation/agent.go` (line 65)**:
```go
ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, toolRuntime, env, b.emitter, opts...)
// PlanningService created inside NewAgent()
```

### Target State

**`internal/agent/builder.go`**:
- Add `BuildPlanningService() *planning.PlanningService` method
- Uses `b.provider` (LLM provider) to create `PlanningService`

**`internal/agent/agent.go`**:
- `NewAgent()` accepts `planningService *planning.PlanningService` as a required parameter
- Remove PlanningService creation from constructor
- Add validation: `ErrNilPlanning` error if `planningService` is nil

**`internal/conversation/agent.go`**:
- Use `agentBuilder.BuildPlanningService()` before calling `agent.NewAgent()`
- Pass `planningService` to `agent.NewAgent()`

### API Changes

**`internal/agent/builder.go`**:
- **NEW**: `BuildPlanningService() *planning.PlanningService` - Creates PlanningService using builder's LLM provider

**`internal/agent/agent.go`**:
- **CHANGED**: `NewAgent()` signature:
  ```go
  // Before:
  func NewAgent(
      provider llm.Provider,
      security *security.SecurityService,
      detection *detection.DetectionService,
      runtime *ToolRuntime,
      context *Environment,
      emitter *events.EventEmitter,
      opts ...AgentOption,
  ) (*Agent, error)
  
  // After:
  func NewAgent(
      provider llm.Provider,
      security *security.SecurityService,
      detection *detection.DetectionService,
      runtime *ToolRuntime,
      planning *planning.PlanningService,  // NEW: required parameter
      context *Environment,
      emitter *events.EventEmitter,
      opts ...AgentOption,
  ) (*Agent, error)
  ```
- **NEW**: `ErrNilPlanning` error constant

**`internal/conversation/agent.go`**:
- **CHANGED**: `buildAgent()` method:
  ```go
  // Build PlanningService using builder
  planningSvc := agentBuilder.BuildPlanningService()
  
  // Create agent with PlanningService
  ag, err := agent.NewAgent(b.llm, securitySvc, detectionSvc, toolRuntime, planningSvc, env, b.emitter, opts...)
  ```

**`internal/appserver/processor.go`**:
- **CHANGED**: `NewProcessor()` method - must build PlanningService before creating Agent

### Files to Modify

1. **`internal/agent/builder.go`**:
   - Add `BuildPlanningService() *planning.PlanningService` method (similar to `BuildSecurityService()`)
   - Uses `b.provider` to create `planning.NewPlanningService(b.provider)`

2. **`internal/agent/agent.go`**:
   - Update `NewAgent()` signature to accept `planning *planning.PlanningService` as required parameter
   - Remove PlanningService creation (line 195-196)
   - Add validation: `if planning == nil { return nil, ErrNilPlanning }`
   - Add `ErrNilPlanning` error constant

3. **`internal/conversation/agent.go`**:
   - Update `buildAgent()` to call `agentBuilder.BuildPlanningService()`
   - Pass `planningSvc` to `agent.NewAgent()` as 4th parameter

4. **`internal/appserver/processor.go`**:
   - Build PlanningService before creating Agent (use `agentBuilder.BuildPlanningService()`)
   - Pass PlanningService to `agent.NewAgent()` as 4th parameter

5. **`internal/agent/agent_test.go`**:
   - Update all `NewAgent()` calls to include PlanningService parameter
   - Create PlanningService in test setup (using builder or direct construction)

6. **`internal/conversation/agent_test.go`** (if exists):
   - Verify PlanningService is built correctly

7. **`internal/appserver/processor_test.go`**:
   - Update tests to verify PlanningService is built correctly

### Files to Create

None

## Testing Strategy

### Unit Tests

**`internal/agent/builder_test.go`**:
```go
func TestBuilder_BuildPlanningService(t *testing.T) {
    provider := llm.NewMockProvider("test")
    builder := agent.NewBuilder().WithProvider(provider)
    
    planningService := builder.BuildPlanningService()
    
    require.NotNil(t, planningService)
    // Verify PlanningService has correct LLM provider
}

func TestBuilder_BuildPlanningService_NilProvider(t *testing.T) {
    builder := agent.NewBuilder()
    // Should handle nil provider gracefully or panic
    // Check actual behavior and test accordingly
}
```

**`internal/agent/agent_test.go`**:
```go
func TestNewAgent_WithPlanningService(t *testing.T) {
    // Setup
    provider := llm.NewMockProvider("test")
    planningService := planning.NewPlanningService(provider)
    
    // Create agent
    agent, err := agent.NewAgent(
        provider,
        securityService,
        detectionService,
        toolRuntime,
        planningService,  // NEW: required parameter
        env,
        emitter,
    )
    
    require.NoError(t, err)
    assert.NotNil(t, agent)
    assert.Equal(t, planningService, agent.planningService) // If exposed
}

func TestNewAgent_NilPlanningService(t *testing.T) {
    // Should return ErrNilPlanning error
    agent, err := agent.NewAgent(
        provider,
        securityService,
        detectionService,
        toolRuntime,
        nil,  // nil PlanningService
        env,
        emitter,
    )
    
    assert.Nil(t, agent)
    assert.Equal(t, agent.ErrNilPlanning, err)
}
```

**`internal/conversation/agent_test.go`** (or similar):
```go
func TestBuilder_BuildAgent_IncludesPlanningService(t *testing.T) {
    // Verify conversation.Builder builds PlanningService correctly
    // Verify PlanningService is passed to agent.NewAgent()
}
```

### Integration Tests

Verify end-to-end flow:
1. Builder creates PlanningService
2. PlanningService is passed to Agent
3. Agent can use PlanningService for plan creation

## Acceptance Criteria

1. ✅ `BuildPlanningService()` method added to `agent.Builder`
2. ✅ `NewAgent()` accepts `PlanningService` as required parameter
3. ✅ PlanningService creation removed from `NewAgent()` constructor
4. ✅ `conversation.Builder` uses `BuildPlanningService()` when building agents
5. ✅ `appserver.Processor` builds PlanningService before creating Agent
6. ✅ All tests updated to pass PlanningService to `NewAgent()`
7. ✅ All tests pass (unit + integration)
8. ✅ `go vet` passes
9. ✅ No dead code introduced
10. ✅ Code coverage ≥90% for new/modified code

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully integrated PlanningService with Builder pattern. Added `BuildPlanningService()` method to `agent.Builder` following the same pattern as `BuildSecurityService()` and `BuildDetectionService()`. Updated `agent.NewAgent()` to accept `PlanningService` as a required parameter (4th parameter) instead of creating it internally. Removed PlanningService creation from `NewAgent()` constructor (line 195-196). Updated `conversation.Builder.buildAgent()` and `appserver.Processor.NewProcessor()` to use `BuildPlanningService()` when constructing agents. Updated all `NewAgent()` call sites including:
- `internal/agent/agent_test.go` (all test cases)
- `internal/agent/planner_test.go`
- `internal/protocol/acp/plan_integration_test.go`
- `cmd/spin/acp.go`

Added tests for `BuildPlanningService()` and nil PlanningService validation in `NewAgent()`. Added `ErrNilPlanning` error constant. All tests pass. This completes the service consolidation pattern for PlanningService.

## Implementation Notes

- Follow the same pattern as `BuildSecurityService()` and `BuildDetectionService()`
- PlanningService creation is straightforward: `planning.NewPlanningService(provider)`
- Ensure all call sites of `agent.NewAgent()` are updated (grep for "NewAgent(")
- No backward compatibility needed (per user instruction)

## Risks

- **Breaking change**: All `agent.NewAgent()` call sites must be updated
- **Mitigation**: Grep for all `NewAgent(` usages and update them systematically

- **Missing call sites**: Some test files or internal packages might call `NewAgent()` directly
- **Mitigation**: Run `go build ./...` and fix compilation errors

## Dependencies

- Feature 4.1 complete (PlanningService created)
- Feature 4.2 complete (Agent uses PlanningService)

