# FRD-20251019-001: Agent Service Extraction

## Metadata
- **Status**: DRAFT
- **Priority**: P0 (CRITICAL)
- **Effort**: L (5 days)
- **Dependencies**: None
- **Related**: [Architectural Anti-Patterns](../../docs/architectural-anti-patterns.md), [Refactoring Roadmap](../refactoring/ROADMAP.md)

## Problem Statement

The `Agent` struct has evolved into a god object with 17+ direct dependencies, violating the Single Responsibility Principle and core architectural guidelines stated in AGENTS.md ("No god objects"). 

Current Agent structure:
```go
type Agent struct {
    llm             llm.Provider           // Core LLM interaction
    executor        *Executor              // DEPRECATED
    validator       *Validator             // Security
    context         *Environment           // Environment
    emitter         *EventEmitter          // Events
    config          *Config                // Configuration
    toolRegistry    *tools.Registry        // Tools
    taskRegistry    *task.Registry         // Task modes
    approvalHandler ApprovalHandler        // DEPRECATED
    approvalService *ApprovalService       // Security
    toolExecutor    *ToolExecutor          // Orchestration
    cycleDetector   *cycle.Detector        // Detection
    patternDetector *cycle.PatternDetector // Detection
    planner         *Plan                  // Planning
}
```

**Issues:**
1. **High coupling**: Changes in any subsystem ripple through Agent
2. **Testing complexity**: Requires mocking 17+ dependencies
3. **Cognitive overload**: Developers must understand all subsystems
4. **Scattered responsibilities**: Security, orchestration, detection, and planning mixed
5. **Deprecated fields**: `executor` and `approvalHandler` still present but unused

**Impact:**
- Difficult to test (complex setup)
- Hard to extend (must touch Agent for any feature)
- Violates SOLID principles
- Makes parallel development difficult

## Goals

1. **Decompose Agent** into focused services with clear boundaries
2. **Reduce dependencies** from 17+ to ≤7 direct dependencies
3. **Improve testability** through interface-based dependency injection
4. **Eliminate deprecated fields** (`executor`, `approvalHandler`)
5. **Maintain all existing functionality** without breaking changes to public API
6. **Enable independent testing** of each service

## Non-Goals

1. **NOT changing public Agent API** - external usage remains the same
2. **NOT refactoring Manager** - that's a separate task (Phase 2)
3. **NOT adding new features** - pure refactoring only
4. **NOT maintaining backward compatibility** for internal structures - user has explicitly requested no backward compatibility

## Design

### Service Decomposition

Extract Agent responsibilities into three focused services:

#### 1. SecurityService
**Responsibility**: Command validation and approval management

```go
// SecurityService handles all security-related operations
type SecurityService struct {
    validator       *Validator
    approvalService *ApprovalService
}

// NewSecurityService creates a security service
func NewSecurityService(validator *Validator, approvalService *ApprovalService) *SecurityService {
    return &SecurityService{
        validator:       validator,
        approvalService: approvalService,
    }
}

// ValidateCommand classifies a command's safety level
func (s *SecurityService) ValidateCommand(cmd *Command) (*ValidationResult, error)

// RequestApproval requests approval for a potentially dangerous operation
func (s *SecurityService) RequestApproval(ctx context.Context, operation Operation) (bool, error)

// NeedsApproval checks if a command requires approval
func (s *SecurityService) NeedsApproval(cmd *Command) bool
```

#### 2. DetectionService
**Responsibility**: Cycle and pattern detection

```go
// DetectionService handles cycle and pattern detection
type DetectionService struct {
    cycleDetector   *cycle.Detector
    patternDetector *cycle.PatternDetector
}

// NewDetectionService creates a detection service
func NewDetectionService(cycleDetector *cycle.Detector, patternDetector *cycle.PatternDetector) *DetectionService {
    return &DetectionService{
        cycleDetector:   cycleDetector,
        patternDetector: patternDetector,
    }
}

// RecordSnapshot records agent state for cycle detection
func (s *DetectionService) RecordSnapshot(snapshot cycle.Snapshot)

// CheckCycle checks for cycle patterns in agent behavior
func (s *DetectionService) CheckCycle() (cycle.Result, error)

// DetectPattern detects advanced patterns in agent behavior
func (s *DetectionService) DetectPattern() (*PatternResult, error)

// Reset clears detection history (useful for new conversations)
func (s *DetectionService) Reset()
```

#### 3. OrchestrationService
**Responsibility**: Tool execution, planning, and task management

```go
// OrchestrationService handles tool execution and task orchestration
type OrchestrationService struct {
    toolExecutor *ToolExecutor
    planner      *Plan
    toolRegistry *tools.Registry
    taskRegistry *task.Registry
}

// NewOrchestrationService creates an orchestration service
func NewOrchestrationService(
    toolExecutor *ToolExecutor,
    toolRegistry *tools.Registry,
    taskRegistry *task.Registry,
) *OrchestrationService {
    return &OrchestrationService{
        toolExecutor: toolExecutor,
        toolRegistry: toolRegistry,
        taskRegistry: taskRegistry,
    }
}

// ExecuteTool executes a single tool call
func (s *OrchestrationService) ExecuteTool(ctx context.Context, call *ToolCall) (*ToolResult, error)

// ExecuteBatch executes multiple tool calls concurrently
func (s *OrchestrationService) ExecuteBatch(ctx context.Context, calls []*ToolCall) ([]*ToolResult, error)

// GetTask retrieves a task by name from the registry
func (s *OrchestrationService) GetTask(name string) (Task, error)

// GetDefaultTask returns the default task mode
func (s *OrchestrationService) GetDefaultTask() (Task, error)

// ListTasks returns all registered task names
func (s *OrchestrationService) ListTasks() []string

// SetPlanner sets the execution planner
func (s *OrchestrationService) SetPlanner(planner *Plan)

// GetPlanner returns the current planner
func (s *OrchestrationService) GetPlanner() *Plan
```

### Refactored Agent Structure

```go
// Agent orchestrates LLM interactions with cleaner dependencies
type Agent struct {
    // Core LLM interaction
    llm              llm.Provider
    
    // Service layers
    security         *SecurityService
    detection        *DetectionService
    orchestration    *OrchestrationService
    
    // Infrastructure
    context          *Environment
    emitter          *EventEmitter
    config           *Config
}
```

**Dependency count**: 7 (down from 17)

### Constructor Changes

```go
// NewAgent creates a new agent with service-based architecture
func NewAgent(
    provider llm.Provider,
    security *SecurityService,
    detection *DetectionService,
    orchestration *OrchestrationService,
    context *Environment,
    emitter *EventEmitter,
    opts ...AgentOption,
) (*Agent, error) {
    // Validate required dependencies
    if provider == nil {
        return nil, ErrNilLLM
    }
    if security == nil {
        return nil, ErrNilSecurity
    }
    if detection == nil {
        return nil, ErrNilDetection
    }
    if orchestration == nil {
        return nil, ErrNilOrchestration
    }
    if context == nil {
        return nil, ErrNilContext
    }
    if emitter == nil {
        return nil, ErrNilEmitter
    }

    // Create agent
    agent := &Agent{
        llm:           provider,
        security:      security,
        detection:     detection,
        orchestration: orchestration,
        context:       context,
        emitter:       emitter,
        config:        DefaultConfig(),
    }

    // Apply options
    for _, opt := range opts {
        if err := opt(agent); err != nil {
            return nil, fmt.Errorf("applying option: %w", err)
        }
    }

    return agent, nil
}
```

### Backward Compatibility Approach

**User has explicitly requested NO backward compatibility**, so:
1. Remove deprecated fields immediately (`executor`, `approvalHandler`)
2. Update all internal callers to use new services
3. No deprecation warnings or adapters

### Migration Path

**For Agent methods that delegate to services:**

```go
// Before (direct field access)
func (a *Agent) validateCommand(cmd *Command) error {
    return a.validator.Validate(cmd)
}

// After (delegate to service)
func (a *Agent) validateCommand(cmd *Command) error {
    _, err := a.security.ValidateCommand(cmd)
    return err
}
```

**For tool execution:**

```go
// Before
result, err := a.toolExecutor.Execute(ctx, call)

// After
result, err := a.orchestration.ExecuteTool(ctx, call)
```

**For cycle detection:**

```go
// Before
a.cycleDetector.Record(snapshot)
result := a.cycleDetector.Check()

// After
a.detection.RecordSnapshot(snapshot)
result, _ := a.detection.CheckCycle()
```

## API Changes

### Breaking Changes (Internal Only)

**Agent struct fields:**
- ❌ **REMOVED**: `executor *Executor` (deprecated field)
- ❌ **REMOVED**: `approvalHandler ApprovalHandler` (deprecated field)
- ❌ **REMOVED**: `validator *Validator` (moved to SecurityService)
- ❌ **REMOVED**: `approvalService *ApprovalService` (moved to SecurityService)
- ❌ **REMOVED**: `toolExecutor *ToolExecutor` (moved to OrchestrationService)
- ❌ **REMOVED**: `cycleDetector *cycle.Detector` (moved to DetectionService)
- ❌ **REMOVED**: `patternDetector *cycle.PatternDetector` (moved to DetectionService)
- ❌ **REMOVED**: `planner *Plan` (moved to OrchestrationService)
- ❌ **REMOVED**: `toolRegistry *tools.Registry` (moved to OrchestrationService)
- ❌ **REMOVED**: `taskRegistry *task.Registry` (moved to OrchestrationService)

**New fields:**
- ✅ **ADDED**: `security *SecurityService`
- ✅ **ADDED**: `detection *DetectionService`
- ✅ **ADDED**: `orchestration *OrchestrationService`

**NewAgent signature:**
```go
// Before
func NewAgent(
    provider llm.Provider,
    executor *Executor,
    validator *Validator,
    context *Environment,
    emitter *EventEmitter,
    opts ...AgentOption,
) (*Agent, error)

// After
func NewAgent(
    provider llm.Provider,
    security *SecurityService,
    detection *DetectionService,
    orchestration *OrchestrationService,
    context *Environment,
    emitter *EventEmitter,
    opts ...AgentOption,
) (*Agent, error)
```

### Non-Breaking Changes (Public API Preserved)

**Public methods remain unchanged:**
- ✅ `Execute(ctx, req) (*AgentResponse, error)` - same signature
- ✅ `BuildToolsForTask(task) ([]llm.Tool, error)` - same signature
- ✅ `GetTaskRegistry() *task.Registry` - delegates to orchestration service
- ✅ `ListTaskModes() []string` - delegates to orchestration service
- ✅ `CreatePlan(ctx, task) (*Plan, error)` - same signature

### New Public APIs

**Service constructors:**
```go
func NewSecurityService(validator *Validator, approvalService *ApprovalService) *SecurityService
func NewDetectionService(cycleDetector *cycle.Detector, patternDetector *cycle.PatternDetector) *DetectionService
func NewOrchestrationService(toolExecutor *ToolExecutor, toolRegistry *tools.Registry, taskRegistry *task.Registry) *OrchestrationService
```

### Files to Create

1. `internal/core/security_service.go` - SecurityService implementation
2. `internal/core/security_service_test.go` - SecurityService tests
3. `internal/core/detection_service.go` - DetectionService implementation
4. `internal/core/detection_service_test.go` - DetectionService tests
5. `internal/core/orchestration_service.go` - OrchestrationService implementation
6. `internal/core/orchestration_service_test.go` - OrchestrationService tests

### Files to Modify

1. `internal/core/agent.go` - Update Agent struct and constructor
2. `internal/core/agent_test.go` - Update tests with new services
3. `internal/core/agent_execute.go` - Update execution logic to use services
4. `internal/core/agent_tools.go` - Update tool methods to use orchestration
5. `internal/core/manager.go` - Update to build services and pass to Agent
6. `cmd/spin/tui.go` - Update agent construction if needed

### Files to Remove

None - we're refactoring existing code, not removing features.

## Testing Strategy

### Unit Tests (≥90% coverage)

**SecurityService:**
- Command validation with different safety levels
- Approval request/response flow
- Timeout handling
- Modified command re-validation
- Edge cases (nil inputs, empty commands)

**DetectionService:**
- Snapshot recording
- Cycle detection (repeated tools, same error, oscillation)
- Pattern detection
- History management (window size, reset)
- Thread safety (concurrent access)

**OrchestrationService:**
- Tool execution (single and batch)
- Task retrieval (by name, default)
- Task listing
- Planner management
- Registry operations

**Agent (integration with services):**
- Agent constructor with services
- Execute method delegation
- Tool execution flow
- Cycle detection integration
- Approval flow integration

### Integration Tests

**Agent + Services:**
```go
func TestAgent_WithServices_ExecutesToolWithApproval(t *testing.T) {
    // Setup
    validator := NewValidator()
    approvalService := NewApprovalService(mockApprovalHandler)
    security := NewSecurityService(validator, approvalService)
    
    detector := cycle.NewDetector(cycle.DefaultConfig())
    detection := NewDetectionService(detector, nil)
    
    toolExecutor := NewToolExecutor(ToolExecutorConfig{...})
    orchestration := NewOrchestrationService(toolExecutor, registry, taskRegistry)
    
    agent, err := NewAgent(llm, security, detection, orchestration, env, emitter)
    require.NoError(t, err)
    
    // Execute dangerous command requiring approval
    resp, err := agent.Execute(ctx, &AgentRequest{
        Input: "delete all files",
    })
    
    // Verify approval was requested and command executed
    assert.NoError(t, err)
    assert.True(t, approvalWasRequested)
}
```

**Service interaction tests:**
- Security service validates before orchestration executes
- Detection service records after orchestration completes
- Event emitter receives events from all services

### E2E Tests

**Full agent loop with services:**
```go
func TestE2E_AgentServices_CompleteWorkflow(t *testing.T) {
    // Real LLM provider (or mock)
    // Real file system operations
    // Real approval flow
    // Real cycle detection
    
    // Verify:
    // 1. Security validates all commands
    // 2. Orchestration executes tools correctly
    // 3. Detection records snapshots
    // 4. Events are emitted properly
    // 5. Cycles are detected and handled
}
```

### Test Helpers

**Service builders for tests:**
```go
func NewTestSecurityService(t *testing.T) *SecurityService {
    validator := NewValidator()
    approvalService := NewApprovalService(nil) // no approval by default
    return NewSecurityService(validator, approvalService)
}

func NewTestDetectionService(t *testing.T) *DetectionService {
    config := cycle.Config{
        WindowSize:       3,
        SimilarityThresh: 0.8,
        ToolRepeatLimit:  3,
        ErrorRepeatLimit: 2,
        Enabled:          true,
    }
    detector := cycle.NewDetector(config)
    return NewDetectionService(detector, nil)
}

func NewTestOrchestrationService(t *testing.T, registry *tools.Registry) *OrchestrationService {
    taskRegistry := task.NewRegistry()
    taskRegistry.Register("regular", task.NewRegular())
    taskRegistry.SetDefault("regular")
    
    toolExecutor := NewToolExecutor(ToolExecutorConfig{
        Registry: registry,
    })
    
    return NewOrchestrationService(toolExecutor, registry, taskRegistry)
}
```

### Mutation Testing

**Key mutants to kill:**
1. Service is nil but no error returned → must fail
2. Approval denied but command executes → must not execute
3. Cycle detected but agent continues → must stop
4. Tool execution fails but result shows success → must show failure

## Acceptance Criteria

### Code Quality
- ✅ Agent has ≤7 direct dependencies (target achieved)
- ✅ All tests pass with `-race` flag
- ✅ Coverage ≥90% for new service code
- ✅ Coverage ≥85% overall maintained
- ✅ `make lint` passes (zero errors)
- ✅ `make deadcode` shows zero dead functions
- ✅ Complexity ≤15 for all new functions

### Functional Requirements
- ✅ All existing Agent functionality works unchanged
- ✅ SecurityService handles all validation and approval
- ✅ DetectionService handles all cycle and pattern detection
- ✅ OrchestrationService handles all tool execution and task management
- ✅ Services are independently testable
- ✅ Services can be mocked easily in tests
- ✅ No deprecated fields remain in Agent struct

### Documentation
- ✅ Godoc for all exported types and methods
- ✅ `docs/packages/core.md` updated with service architecture
- ✅ Architecture diagrams showing service decomposition
- ✅ Migration guide for internal developers (if needed)
- ✅ Examples showing service usage

### Performance
- ✅ No performance degradation (benchmark comparison)
- ✅ Memory usage unchanged or reduced
- ✅ No goroutine leaks in services

## Risks

### High Risk

**Risk**: Breaking existing functionality during refactor
- **Mitigation**: Comprehensive test suite runs after each step
- **Fallback**: Incremental refactoring allows easy rollback

**Risk**: Performance regression from additional indirection
- **Mitigation**: Benchmark tests before/after refactoring
- **Impact**: Likely negligible (method call overhead is nanoseconds)

### Medium Risk

**Risk**: Increased complexity from service management
- **Mitigation**: Clear documentation and examples
- **Mitigation**: Helper functions for service construction

**Risk**: Test setup becomes more complex
- **Mitigation**: Provide test helper functions
- **Mitigation**: Document common test patterns

### Low Risk

**Risk**: Manager refactoring complexity increases
- **Mitigation**: Manager already has builder pattern partially
- **Impact**: Phase 2 task, not blocking

## Alternatives Considered

### Alternative 1: Composition with Embedded Structs

```go
type Agent struct {
    *AgentCore       // LLM + config + events
    *ExecutionLayer  // Tools + executor + validator
    *SecurityLayer   // Approval + validation
    *DetectionLayer  // Cycle + pattern detection
}
```

**Pros:**
- Method delegation is automatic via embedding
- Cleaner field access (no a.security.validator)

**Cons:**
- Name collisions possible
- Less explicit about dependencies
- Harder to mock individual layers
- **REJECTED**: Less clear boundaries

### Alternative 2: Single Service Object

```go
type AgentServices struct {
    Security      *SecurityService
    Detection     *DetectionService
    Orchestration *OrchestrationService
}

type Agent struct {
    llm      llm.Provider
    services *AgentServices
    context  *Environment
    emitter  *EventEmitter
    config   *Config
}
```

**Pros:**
- Single dependency injection point
- Easier to pass services around

**Cons:**
- Less explicit about which services are used where
- Harder to test individual service usage
- **REJECTED**: Doesn't improve clarity

### Alternative 3: Interface-Based Services (Phase 2)

Move to interfaces for all services in Phase 2:
```go
type SecurityService interface {
    ValidateCommand(cmd *Command) (*ValidationResult, error)
    RequestApproval(ctx context.Context, op Operation) (bool, error)
}

type Agent struct {
    security SecurityService // Interface instead of concrete type
}
```

**Decision**: Deferred to Phase 3 (Interface Segregation task)

## Implementation Plan

### Step 1: Create SecurityService (Day 1)
1. Create `security_service.go` with struct and constructor
2. Move validation logic from Agent
3. Move approval logic from Agent
4. Write unit tests (≥90% coverage)
5. Write integration tests

### Step 2: Create DetectionService (Day 1-2)
1. Create `detection_service.go` with struct and constructor
2. Move cycle detection from Agent
3. Move pattern detection from Agent
4. Write unit tests (≥90% coverage)
5. Write integration tests

### Step 3: Create OrchestrationService (Day 2)
1. Create `orchestration_service.go` with struct and constructor
2. Move tool execution logic from Agent
3. Move task registry from Agent
4. Move planner from Agent
5. Write unit tests (≥90% coverage)
6. Write integration tests

### Step 4: Refactor Agent (Day 3-4)
1. Update Agent struct to use services
2. Update Agent constructor
3. Update all Agent methods to delegate to services
4. Remove deprecated fields (`executor`, `approvalHandler`)
5. Ensure all existing tests pass
6. Run `go test -race ./internal/core/...`

### Step 5: Update Manager (Day 4)
1. Update Manager to build services
2. Update Manager.NewConversation to pass services to Agent
3. Ensure all manager tests pass

### Step 6: Update TUI and CLI (Day 4-5)
1. Update `cmd/spin/tui.go` if needed
2. Update any other command files
3. Run full integration tests

### Step 7: Documentation and Cleanup (Day 5)
1. Update `docs/packages/core.md`
2. Add architecture diagrams
3. Document service responsibilities
4. Update `AGENTS.md` if contracts changed
5. Run `make lint`
6. Run `make deadcode`
7. Generate coverage report

### Step 8: Final Verification (Day 5)
1. Run full test suite with `-race`
2. Run benchmarks to verify no regression
3. Verify all acceptance criteria
4. Update roadmap with completion

## Definition of Done

- [x] FRD created and reviewed ← **WE ARE HERE**
- [ ] All 3 services implemented with ≥90% test coverage
- [ ] Agent refactored to use services
- [ ] Manager updated to build and inject services
- [ ] All deprecated fields removed
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] `go test -race ./...` passes
- [ ] `make lint` passes (zero errors)
- [ ] `make deadcode` passes (zero dead functions)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc complete for all exports
- [ ] `docs/packages/core.md` updated
- [ ] `AGENTS.md` updated (if needed)
- [ ] Benchmarks show no regression
- [ ] Roadmap updated with completion date and status

## Related Work

**Depends on:**
- None (first task in Phase 1)

**Blocks:**
- Phase 1.2: Event Emitter Isolation (can work in parallel)
- Phase 1.3: TUI Mapper Relocation (can work in parallel)
- Phase 2.1: Tool Registry Consolidation (needs cleaner Agent)
- Phase 2.3: Builder Pattern (needs cleaner Agent)

**Related FRDs:**
- FRD-20251019-002: Event Emitter Isolation (Phase 1.2)
- FRD-20251019-003: TUI Mapper Relocation (Phase 1.3)

## References

- [Architectural Anti-Patterns Analysis](../../docs/architectural-anti-patterns.md) - Problem identification
- [Refactoring Roadmap](../refactoring/ROADMAP.md) - Overall refactoring plan
- [Core Package Documentation](../../docs/packages/core.md) - Current architecture
- [AGENTS.md](../../AGENTS.md) - Development guidelines
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
- [SOLID Principles](https://en.wikipedia.org/wiki/SOLID) - Design principles

---

**Created**: 2025-10-19  
**Author**: Spin Agent  
**Version**: 1.0  
**Status**: DRAFT → Ready for implementation

