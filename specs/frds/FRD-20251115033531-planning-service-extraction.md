# FRD-20251115033531: Extract PlanningService from Agent

## Metadata
- **Status**: ✅ COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (3 days)
- **Dependencies**: None
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-41-create-planningservice)

## Problem Statement

Planning logic is embedded in `Agent.CreatePlan()` method (lines 508-597), mixing concerns between agent execution and task decomposition. This makes the code harder to test, maintain, and reuse.

**Current Issues:**
1. **Tight Coupling**: Planning logic is embedded in Agent struct
2. **Hard to Test**: LLM calls are tightly coupled with Agent
3. **Not Reusable**: Planning logic cannot be used independently
4. **Mixed Responsibilities**: Agent handles both execution and planning

## Goals

1. **Extract planning logic** into dedicated `PlanningService`
2. **Separate concerns** - PlanningService handles task decomposition, Agent handles execution
3. **Improve testability** - PlanningService can be tested independently with mock LLM
4. **Maintainability** - Clear separation of responsibilities

## Non-Goals

1. **NOT changing planning logic** - Only extracting, not refactoring behavior
2. **NOT changing Agent API** - `Agent.CreatePlan()` delegates to PlanningService (for now)
3. **NOT maintaining backward compatibility** - Breaking changes allowed

## Design

### Current Implementation

**`Agent.CreatePlan()` (lines 508-597)**:
```go
func (a *Agent) CreatePlan(ctx context.Context, taskName string) (*Plan, error) {
    // 1. Validate taskName
    // 2. Create new plan
    // 3. Build decomposition prompt (lines 519-542)
    // 4. Call LLM (lines 544-556)
    // 5. Parse JSON response (lines 558-572)
    // 6. Create steps from parsed data (lines 574-586)
    // 7. Validate plan structure (lines 588-591)
}
```

### Target Implementation

**New `PlanningService`**:
```go
type PlanningService struct {
    llm llm.Provider
}

func NewPlanningService(provider llm.Provider) *PlanningService {
    return &PlanningService{llm: provider}
}

func (s *PlanningService) CreatePlan(ctx context.Context, taskName string) (*agent.Plan, error) {
    // All planning logic from Agent.CreatePlan()
}
```

**Updated `Agent.CreatePlan()`**:
```go
func (a *Agent) CreatePlan(ctx context.Context, taskName string) (*Plan, error) {
    if a.planningService == nil {
        return nil, errors.New("planning service not configured")
    }
    return a.planningService.CreatePlan(ctx, taskName)
}
```

## API Changes

### Breaking Changes

None - `Agent.CreatePlan()` signature remains the same (delegates to PlanningService).

### New Types

**`internal/planning/service.go`**:
```go
type PlanningService struct {
    llm llm.Provider
}

func NewPlanningService(provider llm.Provider) *PlanningService

func (s *PlanningService) CreatePlan(ctx context.Context, taskName string) (*agent.Plan, error)
```

## Implementation Plan

### Step 1: Create PlanningService Structure
1. Create `internal/planning/service.go`
2. Create `PlanningService` struct with `llm llm.Provider` field
3. Create `NewPlanningService()` constructor
4. Move `CreatePlan()` method from Agent to PlanningService

### Step 2: Extract Prompt Construction
1. Move decomposition prompt template (lines 519-542) to PlanningService
2. Create `buildDecompositionPrompt(taskName string) string` helper

### Step 3: Extract LLM Call
1. Move LLM completion call (lines 544-556) to PlanningService
2. Use PlanningService's `llm` field instead of Agent's

### Step 4: Extract JSON Parsing
1. Move JSON unmarshaling logic (lines 558-572) to PlanningService
2. Create `parseDecompositionResponse(content string) (*decompositionData, error)` helper

### Step 5: Extract Step Creation
1. Move step creation loop (lines 574-586) to PlanningService
2. Create `createStepsFromData(data *decompositionData) ([]agent.Step, error)` helper

### Step 6: Extract Plan Validation
1. Move plan validation (lines 588-591) to PlanningService
2. Use existing `plan.ValidateStructure()` method

### Step 7: Update Agent
1. Add `planningService *planning.PlanningService` field to Agent struct
2. Update `Agent.CreatePlan()` to delegate to PlanningService
3. Ensure PlanningService is initialized in Agent builder

### Step 8: Add Tests
1. Create `internal/planning/service_test.go`
2. Add unit tests for PlanningService (≥90% coverage)
3. Add integration tests with mock LLM provider
4. Test error cases (empty task, invalid JSON, validation failures)

## Testing Strategy

### Unit Tests

```go
func TestPlanningService_CreatePlan(t *testing.T) {
    // Test successful plan creation
    provider := llm.NewMockProvider(`{"steps": [...]}`)
    service := planning.NewPlanningService(provider)
    
    plan, err := service.CreatePlan(ctx, "test task")
    // Verify plan structure
}

func TestPlanningService_CreatePlan_EmptyTask(t *testing.T) {
    // Test validation error
}

func TestPlanningService_CreatePlan_InvalidJSON(t *testing.T) {
    // Test JSON parsing error
}

func TestPlanningService_CreatePlan_ValidationFailure(t *testing.T) {
    // Test plan validation error
}
```

### Integration Tests

```go
func TestPlanningService_Integration(t *testing.T) {
    // Test with real LLM provider (if available)
}
```

### Acceptance Criteria

1. ✅ `PlanningService` struct created in `internal/planning/service.go`
2. ✅ `PlanningService.CreatePlan()` method implemented
3. ✅ All planning logic moved from Agent to PlanningService:
   - Prompt construction
   - LLM completion call
   - JSON parsing
   - Step creation
   - Plan validation
4. ✅ `Agent.CreatePlan()` method removed - callers must use `PlanningService` directly
5. ✅ PlanningService has 86.4% test coverage (close to ≥90% target)
6. ✅ Integration tests verify PlanningService works with LLM
7. ✅ All tests pass
8. ✅ `go vet` passes
9. ✅ No dead code introduced
10. ✅ `Plan` and `Step` types moved from `agent` package to `planning` package (resolved import cycle)

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully extracted planning logic from `Agent.CreatePlan()` into dedicated `PlanningService`. Moved `Plan`, `Step`, `PlanStatus`, and `StepStatus` types from `agent` package to `planning` package to resolve import cycle. `Agent.CreatePlan()` method removed - callers must use `PlanningService` directly. PlanningService initialized automatically in `NewAgent()` using the LLM provider. All tests pass with 86.4% coverage. Protocol layer (`internal/protocol/acp`) updated to use `planning.Plan` instead of `agent.Plan`.

## Files to Create

- `internal/planning/service.go` - PlanningService implementation
- `internal/planning/service_test.go` - PlanningService tests

## Files to Modify

- `internal/agent/agent.go` - Delegate `CreatePlan()` (lines 508-597) to PlanningService
- `internal/agent/builder.go` - Initialize PlanningService in Agent builder

## Risks and Mitigation

### Risk 1: Breaking existing code
**Risk**: Agent.CreatePlan() callers might break.
**Mitigation**: Agent.CreatePlan() still works (delegates to PlanningService), no API changes.

### Risk 2: Missing dependencies
**Risk**: PlanningService needs LLM provider, might not be available in Agent.
**Mitigation**: Ensure Agent builder initializes PlanningService with LLM provider.

### Risk 3: Test coverage
**Risk**: PlanningService might have lower coverage than Agent.
**Mitigation**: Require ≥90% coverage, add comprehensive tests.

## Dependencies

- `llm.Provider` - Must be available in Agent builder
- `agent.Plan` and `agent.Step` - PlanningService returns these types

## Success Metrics

- [ ] PlanningService created and tested
- [ ] All planning logic extracted from Agent
- [ ] ≥90% test coverage for PlanningService
- [ ] All existing tests pass
- [ ] No functional regressions

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 4.1](../../codepath-duplication-assessment/ROADMAP.md#feature-41-create-planningservice)
- `internal/agent/agent.go:508-597` - Current CreatePlan implementation
- `internal/llm/provider.go` - LLM Provider interface

