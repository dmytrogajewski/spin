# FRD-20251115021647: Unify Processor to Use Builder Pattern

## Metadata
- **Status**: COMPLETE
- **Priority**: P0 (CRITICAL)
- **Effort**: M (2 days)
- **Dependencies**: Feature 1.1 (complete)
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-12-unify-processor-to-use-builder-pattern)

## Problem Statement

`appserver/processor.go` manually constructs services (ApprovalService, SecurityService, DetectionService) instead of using the `agent.Builder` pattern. Currently:

1. **Manual service construction** (lines 111-116):
   - Creates `ApprovalService` directly with `security.NewApprovalService()`
   - Creates `SecurityService` directly with `security.NewSecurityService()`
   - Creates `DetectionService` manually with hardcoded disabled cycle detection config

2. **Inconsistent pattern**: 
   - `conversation/agent.go` already uses `agent.Builder` (lines 13-50)
   - `appserver/processor.go` duplicates service construction logic
   - Builder provides `BuildSecurityService()` and `BuildDetectionService()` helpers that aren't used

3. **Hardcoded configuration**:
   - Cycle detection is always disabled: `cycle.Config{Enabled: false}`
   - No way to use unified config settings from config package
   - Cannot benefit from Builder's config-driven construction

**Impact:**
- Code duplication (service construction in 2+ places)
- Maintenance burden (must update multiple sites for service changes)
- Configuration inconsistencies (hardcoded vs config-driven)
- Violates DRY and single responsibility principles

## Goals

1. **Refactor Processor to use `agent.Builder`** for service construction
2. **Eliminate manual service creation** from Processor constructor
3. **Use Builder helper methods**: `BuildSecurityService()`, `BuildDetectionService()`, `BuildAgentOptions()`
4. **Maintain ToolRuntime construction** using factory from Feature 1.1
5. **Preserve existing functionality** - no behavioral changes
6. **Support optional unified config** if provided in ProcessorConfig

## Non-Goals

1. **NOT changing ProcessorConfig interface** - keep backward compatible
2. **NOT requiring unified config** - Builder should work without it (returns defaults)
3. **NOT refactoring ToolRuntime construction** - already uses factory from Feature 1.1
4. **NOT changing conversation package** - it already uses Builder correctly

## Design

### Current Implementation (processor.go:111-137)

```go
if config.Provider != nil {
    approvalService := security.NewApprovalService(nil, emitter, validator)
    securityService := security.NewSecurityService(validator, approvalService)

    cycleConfig := cycle.Config{Enabled: false}
    cycleDetector := cycle.NewDetector(cycleConfig)
    detectionService := detection.NewDetectionService(cycleDetector, nil)

    toolRegistry := tools.NewDefaultRegistry(environment.WorkDir, environment)

    toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
        Registry:        toolRegistry,
        Validator:       validator,
        ApprovalService: approvalService,
        Emitter:         emitter,
        WorkDir:         environment.WorkDir,
    })

    agentInstance, err = agent.NewAgent(
        config.Provider,
        securityService,
        detectionService,
        toolRuntime,
        environment,
        emitter,
    )
}
```

### Target Implementation

```go
if config.Provider != nil {
    // Create Builder instance
    agentBuilder := agent.NewBuilder().
        WithProvider(config.Provider).
        WithWorkingDir(config.WorkspacePath).
        WithEmitter(emitter).
        WithApprovalHandler(nil) // Processor doesn't have approval handler
    
    // Use Builder helpers for service construction
    securityService := agentBuilder.BuildSecurityService()
    detectionService := agentBuilder.BuildDetectionService()
    
    // Tool registry already uses factory (from Feature 1.1)
    toolRegistry := tools.NewDefaultRegistry(environment.WorkDir, environment)
    
    // Extract ApprovalService from SecurityService for ToolRuntime
    approvalService := securityService.ApprovalService() // If method exists, or extract differently
    
    toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
        Registry:        toolRegistry,
        Validator:       validator, // Still need validator for ToolRuntime
        ApprovalService: approvalService,
        Emitter:         emitter,
        WorkDir:         environment.WorkDir,
    })
    
    // Use Builder for agent options
    opts := agentBuilder.BuildAgentOptions()
    
    agentInstance, err = agent.NewAgent(
        config.Provider,
        securityService,
        detectionService,
        toolRuntime,
        environment,
        emitter,
        opts...,
    )
}
```

### Challenges

1. **ApprovalService extraction**: ToolRuntime needs ApprovalService separately. Options:
   - Add `SecurityService.ApprovalService()` getter method (if doesn't exist)
   - Keep validator creation for ToolRuntime (still needed)
   - Builder creates ApprovalService internally - need access for ToolRuntime

2. **Validator access**: ToolRuntime still needs validator. Builder creates it internally but doesn't expose it. Options:
   - Extract validator from SecurityService (if method exists)
   - Create validator separately for ToolRuntime (acceptable duplication)
   - Add `Builder.BuildValidator()` method

3. **Unified config**: Processor doesn't currently use unified config. Can add optional support:
   - Add `UnifiedConfig *config.ConfigV2` to ProcessorConfig (optional)
   - Pass to Builder if provided
   - If not provided, Builder uses defaults

## API Changes

### ProcessorConfig (optional extension)

```go
type ProcessorConfig struct {
    WorkspacePath string
    Version       string
    Provider      llm.Provider
    Executor      *agent.Executor
    Validator     *security.Validator
    Environment   *agent.Environment
    UnifiedConfig *config.ConfigV2  // NEW: Optional unified config
}
```

**Breaking Change**: No - field is optional, existing code continues to work.

### SecurityService (may need getter)

```go
// If not already exists, add getter for ApprovalService
func (s *SecurityService) ApprovalService() *ApprovalService {
    return s.approvalService
}
```

## Implementation Plan

### Step 1: Add ApprovalService getter (if needed)
1. Check if `SecurityService` has getter for ApprovalService
2. If not, add `ApprovalService() *ApprovalService` method
3. Add unit tests

### Step 2: Create Builder in Processor
1. Create `agent.NewBuilder()` instance
2. Set required parameters: `WithProvider()`, `WithWorkingDir()`, `WithEmitter()`
3. Add optional unified config support if provided

### Step 3: Replace SecurityService construction
1. Replace manual `ApprovalService` + `SecurityService` creation
2. Use `agentBuilder.BuildSecurityService()`
3. Extract ApprovalService for ToolRuntime

### Step 4: Replace DetectionService construction
1. Replace manual cycle detector + DetectionService creation
2. Use `agentBuilder.BuildDetectionService()`
3. Builder will use config for cycle detection (or defaults to disabled)

### Step 5: Use Builder for Agent options
1. Replace direct Agent creation
2. Use `agentBuilder.BuildAgentOptions()` for agent configuration
3. Apply options to `NewAgent()` call

### Step 6: Update tests
1. Verify all Processor tests pass
2. Add integration test verifying Builder is used
3. Test with and without unified config

### Step 7: Verify no functional changes
1. Run all existing tests
2. Verify E2E tests pass
3. Check for any behavioral differences

## Testing Strategy

### Unit Tests

```go
func TestProcessor_UsesBuilder(t *testing.T) {
    // Verify Processor uses Builder internally
    // Indirect verification via integration test
}

func TestProcessor_AgentCreation_Equivalent(t *testing.T) {
    // Create Agent via old manual method
    // Create Agent via new Builder method
    // Compare capabilities, services, configuration
    // Verify they produce equivalent agents
}

func TestProcessor_WithUnifiedConfig(t *testing.T) {
    // Test Processor with unified config provided
    // Verify Builder uses config settings
}

func TestProcessor_WithoutUnifiedConfig(t *testing.T) {
    // Test Processor without unified config
    // Verify Builder uses defaults
}
```

### Integration Tests

```go
func TestProcessor_AgentExecution_BehaviorUnchanged(t *testing.T) {
    // Run agent execution via Processor
    // Verify behavior matches previous implementation
    // Test approval flow, tool execution, etc.
}
```

### Acceptance Criteria

1. ✅ Processor uses `agent.Builder` for service construction
2. ✅ No manual `ApprovalService` creation in Processor
3. ✅ No manual `SecurityService` creation in Processor  
4. ✅ No manual `DetectionService` creation in Processor
5. ✅ Tool registry uses factory from Feature 1.1
6. ✅ Agent creation uses `BuildAgentOptions()` if available
7. ✅ All Processor tests pass
8. ✅ Integration tests verify Processor creates Agent correctly
9. ✅ No functional regression (E2E tests pass)
10. ✅ `go vet` passes
11. ✅ Code review confirms Builder pattern correctly applied

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully refactored Processor to use Builder pattern for service construction, eliminating manual ApprovalService, SecurityService, and DetectionService creation. Added getter methods to SecurityService to support ToolRuntime requirements.

## Files to Modify

- `internal/appserver/processor.go` - Refactor `NewProcessor()` to use Builder (lines 108-141)
- `internal/security/security.go` - Add `ApprovalService()` getter if needed (optional)
- `internal/appserver/processor_test.go` - Update/add tests
- `internal/appserver/processor_integration_test.go` - Verify behavior unchanged

## Risks and Mitigation

### Risk 1: ApprovalService access pattern
**Risk**: ToolRuntime needs ApprovalService, but Builder creates it internally in SecurityService.
**Mitigation**: Add getter method or extract via SecurityService.Validator() if available.

### Risk 2: Behavioral changes
**Risk**: Builder may create services with different defaults than manual construction.
**Mitigation**: Comprehensive integration tests, compare before/after behavior.

### Risk 3: Config compatibility
**Risk**: Processor doesn't have unified config, Builder may expect it.
**Mitigation**: Builder handles nil config gracefully (returns defaults), already tested.

## Dependencies

- ✅ Feature 1.1 (Tool Registry Factory) - Complete
- `agent.Builder` - Already exists and tested
- `SecurityService` - May need getter method
- `config.ConfigV2` - Optional, for future enhancement

## Success Metrics

- [ ] Zero manual service construction in Processor
- [ ] Processor uses Builder helper methods
- [ ] All tests pass (unit, integration, E2E)
- [ ] No functional regression
- [ ] Code duplication reduced (service construction unified)

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 1.2](../../codepath-duplication-assessment/ROADMAP.md#feature-12-unify-processor-to-use-builder-pattern)
- `internal/agent/builder.go` - Builder implementation
- `internal/conversation/agent.go` - Example of Builder usage

