# Architectural Refactoring Roadmap

**Project:** Spin AI Coding Agent  
**Document Type:** Refactoring Roadmap  
**Created:** 2025-10-19  
**Status:** ACTIVE  
**Related:** [Architectural Anti-Patterns Analysis](../../docs/architectural-anti-patterns.md)

## Overview

This roadmap addresses 10 identified architectural anti-patterns through systematic refactoring. Each phase is designed to be completed independently WITHOUT maintaining backward compatibility and 100% test coverage.

**Guiding Principles:**
- ✅ All tests must pass before and after each refactor
- ✅ Coverage must remain ≥85% (≥90% for refactored code)
- ✅ Each task must have an FRD in `specs/frds/`
- ✅ No breaking changes to public APIs without deprecation period
- ✅ Follow 14-step workflow from AGENTS.md
- ✅ Refactor, but never simplify implementation
- ✅ Zero dead code

---

## Phase 1: Foundation (Critical Issues)

**Goal:** Fix critical architectural violations that block other improvements  
**Duration:** 3-4 weeks  
**Risk:** High (core changes)  
**Effort:** High

### 1.1 Extract Agent Services [P0 - CRITICAL]

**Problem:** Agent is a god object with 17+ dependencies (Anti-Pattern #1)  
**FRD:** `FRD-202510-001-agent-service-extraction.md` (TO CREATE)  
**Dependencies:** None  
**Effort:** 5 days  

#### Tasks

- [ ] **1.1.1** Create FRD for agent service extraction
  - [ ] Document current Agent structure
  - [ ] Design service boundaries
  - [ ] Define interfaces for each service

- [ ] **1.1.2** Create `SecurityService` 
  - [ ] Extract validation logic from Agent
  - [ ] Extract approval logic from Agent
  - [ ] Create `SecurityService` struct with interfaces
  - [ ] Write unit tests (≥90% coverage)
  - [ ] Write integration tests
  ```go
  type SecurityService struct {
      validator       CommandValidator
      approvalService ApprovalService
  }
  ```

- [ ] **1.1.3** Create `DetectionService`
  - [ ] Extract cycle detection from Agent
  - [ ] Extract pattern detection from Agent
  - [ ] Create `DetectionService` struct
  - [ ] Write unit tests (≥90% coverage)
  - [ ] Write integration tests
  ```go
  type DetectionService struct {
      cycleDetector   *cycle.Detector
      patternDetector *cycle.PatternDetector
  }
  ```

- [ ] **1.1.4** Create `OrchestrationService`
  - [ ] Extract tool execution logic
  - [ ] Extract planning logic
  - [ ] Create `OrchestrationService` struct
  - [ ] Write unit tests (≥90% coverage)
  - [ ] Write integration tests
  ```go
  type OrchestrationService struct {
      toolExecutor *ToolExecutor
      planner      *Plan
      toolRegistry *tools.Registry
      taskRegistry *task.Registry
  }
  ```

- [ ] **1.1.5** Refactor Agent to use services
  - [ ] Update Agent struct to use new services
  - [ ] Update Agent constructor
  - [ ] Update all Agent methods to delegate to services
  - [ ] Ensure all existing tests pass
  - [ ] Run `go test -race ./internal/core/...`
  - [ ] Run `make deadcode`

- [ ] **1.1.6** Remove deprecated fields
  - [ ] Remove `executor` field (use OrchestrationService)
  - [ ] Remove `approvalHandler` field (use SecurityService)
  - [ ] Remove `toolExecutor` field (use OrchestrationService)
  - [ ] Remove `cycleDetector` field (use DetectionService)
  - [ ] Remove `patternDetector` field (use DetectionService)
  - [ ] Remove `planner` field (use OrchestrationService)
  - [ ] Update all call sites

- [ ] **1.1.7** Update documentation
  - [ ] Update `docs/packages/core.md`
  - [ ] Update `AGENTS.md` if contracts changed
  - [ ] Add architecture diagrams
  - [ ] Document service responsibilities

**Acceptance Criteria:**
- ✅ Agent has ≤7 direct dependencies
- ✅ All tests pass with `-race` flag
- ✅ Coverage ≥90% for new service code
- ✅ `make lint` passes
- ✅ `make deadcode` shows zero dead functions
- ✅ No breaking changes to public Agent API
- ✅ Documentation updated

**Rollback Plan:**
- Keep deprecated fields for one release cycle
- Tag release as `v0.x.0-refactor-preview`

---

### 1.2 Isolate Event Emitter [P0 - CRITICAL]

**Problem:** Single EventEmitter shared across all conversations (Anti-Pattern #3)  
**FRD:** `FRD-202510-002-event-emitter-isolation.md` (TO CREATE)  
**Dependencies:** None  
**Effort:** 3 days

#### Tasks

- [ ] **1.2.1** Create FRD for event emitter isolation
  - [ ] Document current event flow
  - [ ] Design per-conversation isolation strategy
  - [ ] Define event routing if shared emitter needed

- [ ] **1.2.2** Add conversation ID to events
  - [ ] Add `ConversationID` field to Event struct
  - [ ] Update all event emission sites to include ID
  - [ ] Write tests for event routing
  ```go
  type Event struct {
      Type           EventType
      ConversationID string      // NEW
      Timestamp      time.Time
      Data           interface{}
  }
  ```

- [ ] **1.2.3** Create per-conversation emitters
  - [ ] Update Manager.NewConversation to create isolated emitter
  - [ ] Pass conversation-specific emitter to Agent, History
  - [ ] Update Conversation to use its own emitter
  - [ ] Ensure emitter cleanup on conversation close

- [ ] **1.2.4** Add event routing support (optional)
  - [ ] Create EventRouter for cross-conversation events
  - [ ] Implement subscription filtering by conversation ID
  - [ ] Add tests for multi-conversation scenarios

- [ ] **1.2.5** Update all tests
  - [ ] Fix conversation tests to expect isolated events
  - [ ] Add parallel conversation tests
  - [ ] Verify no event leakage between conversations
  - [ ] Run `go test -race ./internal/core/...`

- [ ] **1.2.6** Update documentation
  - [ ] Update `docs/packages/core.md` event section
  - [ ] Document event isolation guarantees
  - [ ] Add examples for multi-conversation scenarios

**Acceptance Criteria:**
- ✅ Each conversation has isolated event stream
- ✅ No event leakage between conversations
- ✅ Tests can run conversations in parallel
- ✅ All existing tests pass
- ✅ Coverage ≥90% for event isolation code
- ✅ `make lint` passes

---

### 1.3 Move TUI Mapper to UI Layer [P0 - CRITICAL]

**Problem:** TUI mapper in core package violates clean architecture (Anti-Pattern #9)  
**FRD:** `FRD-202510-003-tui-mapper-relocation.md` (TO CREATE)  
**Dependencies:** None  
**Effort:** 2 days

#### Tasks

- [ ] **1.3.1** Create FRD for TUI mapper relocation
  - [ ] Document current coupling
  - [ ] Design clean adapter pattern
  - [ ] Plan file relocation
  - [ ] Document interface boundaries

- [ ] **1.3.2** Create adapter interface in core
  - [ ] Define minimal EventAdapter interface
  - [ ] Core emits events through interface
  - [ ] No TUI-specific types in core
  ```go
  // internal/core/event_adapter.go
  type EventAdapter interface {
      HandleEvent(event Event) error
  }
  ```

- [ ] **1.3.3** Move TUI mapper to UI layer
  - [ ] Move `internal/core/tui_mapper.go` → `internal/ui/adapters/core_event_mapper.go`
  - [ ] Move `internal/core/tui_mapper_test.go` → `internal/ui/adapters/core_event_mapper_test.go`
  - [ ] Move `internal/core/tui_mapper_e2e_test.go` → `internal/ui/adapters/core_event_mapper_e2e_test.go`
  - [ ] Update imports in all files

- [ ] **1.3.4** Implement adapter in UI layer
  - [ ] UI adapter implements EventAdapter interface
  - [ ] Converts core events to UI representations
  - [ ] No reverse dependencies (UI→Core only)

- [ ] **1.3.5** Update TUI command
  - [ ] Update `cmd/spin/tui.go` to use new adapter
  - [ ] Remove direct dependency on core mapper
  - [ ] Verify TUI functionality unchanged

- [ ] **1.3.6** Verify clean architecture
  - [ ] Run `go mod graph` to check dependencies
  - [ ] Ensure core doesn't import ui packages
  - [ ] Run all tests
  - [ ] Run `make deadcode`

- [ ] **1.3.7** Update documentation
  - [ ] Update `docs/packages/core.md`
  - [ ] Update `docs/packages/ui-blocks.md`
  - [ ] Update architecture diagrams
  - [ ] Document adapter pattern

**Acceptance Criteria:**
- ✅ No TUI-specific code in `internal/core/`
- ✅ Core package has no UI dependencies
- ✅ TUI functionality unchanged
- ✅ All tests pass
- ✅ Coverage maintained
- ✅ `make lint` passes

---

## Phase 2: Consolidation (Medium Priority)

**Goal:** Eliminate code duplication and improve maintainability  
**Duration:** 2-3 weeks  
**Risk:** Medium  
**Effort:** Medium

### 2.1 Consolidate Tool Registry [P1 - HIGH]

**Problem:** Duplicate tool registration in multiple places (Anti-Pattern #2)  
**FRD:** `FRD-202510-004-tool-registry-consolidation.md` (TO CREATE)  
**Dependencies:** 1.1 (Agent refactor)  
**Effort:** 2 days

#### Tasks

- [ ] **2.1.1** Create FRD for tool registry consolidation

- [ ] **2.1.2** Create default registry factory
  - [ ] Create `internal/tools/default.go`
  - [ ] Implement `NewDefaultRegistry()` function
  - [ ] Support optional dependencies (executor, validator, env)
  - [ ] Write comprehensive tests
  ```go
  func NewDefaultRegistry(opts ...RegistryOption) *Registry {
      registry := NewRegistry()
      
      // Always register basic tools
      _ = registry.Register(NewReadFileTool())
      _ = registry.Register(NewWriteFileTool())
      _ = registry.Register(NewListDirectoryTool())
      
      // Apply options for context-dependent tools
      for _, opt := range opts {
          opt(registry)
      }
      
      return registry
  }
  
  func WithExecutor(exec *Executor, val *Validator) RegistryOption { ... }
  func WithEnvironment(env *Environment) RegistryOption { ... }
  ```

- [ ] **2.1.3** Update Agent to use factory
  - [ ] Replace manual registration with factory call
  - [ ] Remove duplicate registration code
  - [ ] Ensure all tests pass

- [ ] **2.1.4** Update TUI to use factory
  - [ ] Replace manual registration in `cmd/spin/tui.go`
  - [ ] Remove duplicate registration code
  - [ ] Verify TUI functionality

- [ ] **2.1.5** Update Manager to use factory
  - [ ] Use factory in Manager.NewConversation
  - [ ] Remove manual registration code
  - [ ] Run all manager tests

- [ ] **2.1.6** Add extensibility tests
  - [ ] Test custom tool addition
  - [ ] Test tool override
  - [ ] Test conditional registration

- [ ] **2.1.7** Update documentation
  - [ ] Update `docs/packages/tools.md`
  - [ ] Add usage examples
  - [ ] Document extension points

**Acceptance Criteria:**
- ✅ Single source of truth for default tools
- ✅ All registration code removed from Agent/TUI/Manager
- ✅ All tests pass
- ✅ Easy to add new default tools
- ✅ Coverage ≥90%

---

### 2.2 Consolidate Task Registry [P1 - HIGH]

**Problem:** Duplicate task registry initialization (Anti-Pattern #7)  
**FRD:** `FRD-202510-005-task-registry-consolidation.md` (TO CREATE)  
**Dependencies:** 1.1 (Agent refactor)  
**Effort:** 1 day

#### Tasks

- [ ] **2.2.1** Create FRD for task registry consolidation

- [ ] **2.2.2** Create default task registry factory
  - [ ] Create `internal/core/task/default.go`
  - [ ] Implement `NewDefaultRegistry()` function
  - [ ] Register all built-in modes
  - [ ] Write tests
  ```go
  func NewDefaultRegistry() (*Registry, error) {
      registry := NewRegistry()
      
      modes := map[string]Task{
          "regular":  NewRegular(),
          "review":   NewReview(),
          "compact":  NewCompact(),
          "planning": NewPlanning(),
      }
      
      for name, mode := range modes {
          if err := registry.Register(name, mode); err != nil {
              return nil, fmt.Errorf("register %s: %w", name, err)
          }
      }
      
      if err := registry.SetDefault("regular"); err != nil {
          return nil, err
      }
      
      return registry, nil
  }
  ```

- [ ] **2.2.3** Update Agent to use factory
  - [ ] Replace manual registration with factory
  - [ ] Remove duplicate code
  - [ ] Ensure tests pass

- [ ] **2.2.4** Update all call sites
  - [ ] Update TUI if needed
  - [ ] Update Manager if needed
  - [ ] Update tests

- [ ] **2.2.5** Update documentation
  - [ ] Update task mode documentation
  - [ ] Add extension examples

**Acceptance Criteria:**
- ✅ Single initialization function
- ✅ All duplicate code removed
- ✅ All tests pass
- ✅ Easy to add new modes

---

### 2.3 Extract Builder Pattern [P2 - MEDIUM]

**Problem:** Manager has feature envy building all components (Anti-Pattern #4)  
**FRD:** `FRD-202510-006-builder-pattern-extraction.md` (TO CREATE)  
**Dependencies:** 1.1 (Agent refactor), 2.1 (Tool registry), 2.2 (Task registry)  
**Effort:** 4 days

#### Tasks

- [ ] **2.3.1** Create FRD for builder pattern

- [ ] **2.3.2** Create ExecutorBuilder
  - [ ] Create `internal/core/executor_builder.go`
  - [ ] Implement fluent builder API
  - [ ] Move construction logic from Manager
  - [ ] Write tests
  ```go
  type ExecutorBuilder struct {
      workDir         string
      validator       *Validator
      approvalService *ApprovalService
      timeout         time.Duration
      cache           *CommandCache
      logger          *slog.Logger
  }
  
  func NewExecutorBuilder(workDir string) *ExecutorBuilder { ... }
  func (b *ExecutorBuilder) WithValidator(v *Validator) *ExecutorBuilder { ... }
  func (b *ExecutorBuilder) WithApproval(a *ApprovalService) *ExecutorBuilder { ... }
  func (b *ExecutorBuilder) WithTimeout(d time.Duration) *ExecutorBuilder { ... }
  func (b *ExecutorBuilder) WithCache(c *CommandCache) *ExecutorBuilder { ... }
  func (b *ExecutorBuilder) Build() (*Executor, error) { ... }
  ```

- [ ] **2.3.3** Create EnvironmentBuilder
  - [ ] Create `internal/core/environment_builder.go`
  - [ ] Implement builder for Environment
  - [ ] Move gathering logic from Manager
  - [ ] Write tests

- [ ] **2.3.4** Create AgentBuilder
  - [ ] Create `internal/core/agent_builder.go`
  - [ ] Implement builder for Agent
  - [ ] Support service injection
  - [ ] Write tests

- [ ] **2.3.5** Refactor Manager to use builders
  - [ ] Update Manager.NewConversation
  - [ ] Remove all builder methods from Manager
  - [ ] Manager delegates to builders
  - [ ] Simplify Manager to <200 lines

- [ ] **2.3.6** Update all tests
  - [ ] Test builders in isolation
  - [ ] Test Manager with builders
  - [ ] Ensure all integration tests pass

- [ ] **2.3.7** Update documentation
  - [ ] Document builder pattern
  - [ ] Add usage examples
  - [ ] Update architecture docs

**Acceptance Criteria:**
- ✅ Manager <200 lines (down from 513)
- ✅ Builders are independent and testable
- ✅ All construction logic encapsulated in builders
- ✅ All tests pass
- ✅ Coverage ≥90% for builder code

---

## Phase 3: Polish (Low Priority)

**Goal:** Clean up technical debt and improve code quality  
**Duration:** 1-2 weeks  
**Risk:** Low  
**Effort:** Low to Medium

### 3.1 Add Interface Segregation [P3 - LOW]

**Problem:** Agent depends on concrete types (Anti-Pattern #6)  
**FRD:** `FRD-202510-007-interface-segregation.md` (TO CREATE)  
**Dependencies:** 1.1 (Agent refactor), 2.3 (Builders)  
**Effort:** 3 days

#### Tasks

- [ ] **3.1.1** Create FRD for interface segregation

- [ ] **3.1.2** Define minimal interfaces
  - [ ] Create `internal/core/interfaces.go`
  - [ ] Define CommandExecutor interface
  - [ ] Define CommandValidator interface
  - [ ] Define EventBus interface
  - [ ] Define ToolRegistry interface
  ```go
  type CommandExecutor interface {
      Execute(ctx context.Context, cmd string, opts ...ExecOption) (*ExecResult, error)
  }
  
  type CommandValidator interface {
      Classify(cmd string) Classification
      RequiresApproval(cmd string) bool
  }
  
  type EventBus interface {
      Emit(event Event)
      Subscribe() (string, <-chan Event, error)
  }
  ```

- [ ] **3.1.3** Update Agent to use interfaces
  - [ ] Change Agent struct fields to interfaces
  - [ ] Update all tests

- [ ] **3.1.4** Update builders to return interfaces
  - [ ] Builders return interface types
  - [ ] Concrete types implement interfaces
  - [ ] Verify all call sites

- [ ] **3.1.5** Create mock implementations
  - [ ] Generate mocks for testing
  - [ ] Update test helpers
  - [ ] Simplify test setup

- [ ] **3.1.6** Update documentation
  - [ ] Document interface contracts
  - [ ] Add testing guide with mocks

**Acceptance Criteria:**
- ✅ Agent depends on interfaces, not concrete types
- ✅ Easy to mock in tests
- ✅ All tests use mocks where appropriate
- ✅ Test setup simplified
- ✅ Coverage maintained

---

### 3.2 Enrich Config Model [P3 - LOW]

**Problem:** Config is anemic with no behavior (Anti-Pattern #8)  
**FRD:** `FRD-202510-008-config-enrichment.md` (TO CREATE)  
**Dependencies:** 2.3 (Builders)  
**Effort:** 2 days

#### Tasks

- [ ] **3.2.1** Create FRD for config enrichment

- [ ] **3.2.2** Add config methods
  - [ ] Add `WithDefaults()` method
  - [ ] Add `Merge(other *Config)` method
  - [ ] Add `ExecutorOptions()` method
  - [ ] Add `AgentOptions()` method
  - [ ] Add `ProviderConfig()` method
  - [ ] Write tests for all methods

- [ ] **3.2.3** Move config manipulation to methods
  - [ ] Extract from Manager into Config methods
  - [ ] Extract from builders into Config methods
  - [ ] Remove scattered config logic

- [ ] **3.2.4** Update all call sites
  - [ ] Use new Config methods
  - [ ] Simplify Manager
  - [ ] Simplify builders

- [ ] **3.2.5** Update documentation
  - [ ] Document Config methods
  - [ ] Add usage examples

**Acceptance Criteria:**
- ✅ Config has behavior, not just data
- ✅ All config manipulation centralized
- ✅ Tests cover all config methods
- ✅ Manager/builders simplified

---

### 3.3 Type-safe Event Types [P3 - LOW]

**Problem:** Int-based EventType is error-prone (Anti-Pattern #10)  
**FRD:** `FRD-202510-009-typesafe-events.md` (TO CREATE)  
**Dependencies:** 1.2 (Event isolation)  
**Effort:** 1 day

#### Tasks

- [ ] **3.3.1** Create FRD for type-safe events

- [ ] **3.3.2** Convert EventType to string-based enum
  - [ ] Change `type EventType int` to `type EventType string`
  - [ ] Update all constants to string values
  - [ ] Add `Valid()` method
  - [ ] Update tests

- [ ] **3.3.3** Update all event emission sites
  - [ ] Use string constants instead of ints
  - [ ] Ensure compile-time safety
  - [ ] Run all tests

- [ ] **3.3.4** Update documentation
  - [ ] Document event type constants
  - [ ] Update event handling examples

**Acceptance Criteria:**
- ✅ String-based event types
- ✅ Compile-time type safety
- ✅ All tests pass
- ✅ No runtime panics in String()

---

### 3.4 Remove Deprecated Fields [P3 - LOW]

**Problem:** Agent has deprecated fields (Anti-Pattern #5)  
**FRD:** `FRD-202510-010-deprecated-cleanup.md` (TO CREATE)  
**Dependencies:** 1.1 (Agent refactor - services extracted)  
**Effort:** 1 day

#### Tasks

- [ ] **3.4.1** Create FRD for deprecated field removal

- [ ] **3.4.2** Audit deprecated field usage
  - [ ] Search for all references to `agent.executor`
  - [ ] Search for all references to `agent.approvalHandler`
  - [ ] Document remaining usage

- [ ] **3.4.3** Remove deprecated fields
  - [ ] Remove from Agent struct
  - [ ] Remove from Agent constructor
  - [ ] Update all call sites to use new services

- [ ] **3.4.4** Run full test suite
  - [ ] Fix any breakage
  - [ ] Ensure coverage maintained

- [ ] **3.4.5** Update documentation
  - [ ] Remove deprecated field references

**Acceptance Criteria:**
- ✅ No deprecated fields in Agent
- ✅ All tests pass
- ✅ Zero references to removed fields
- ✅ Clean `make deadcode`

---

## Cross-Cutting Concerns

### Documentation Strategy

For each phase:
- [ ] Update package documentation in `docs/packages/`
- [ ] Update architecture overview if applicable
- [ ] Update examples in `examples/`
- [ ] Update AGENTS.md if contracts change

### Testing Strategy

For each task:
- [ ] Unit tests for new code (≥90% coverage)
- [ ] Integration tests for component interactions
- [ ] E2E tests for user-facing changes
- [ ] Run `go test -race ./...`
- [ ] Run `make lint`
- [ ] Run `make deadcode`

---

## Success Metrics

### Code Quality Metrics

| Metric | Current | Target | Phase |
|--------|---------|--------|-------|
| Agent dependencies | 17 | ≤7 | Phase 1 |
| Manager LOC | 513 | <200 | Phase 2 |
| Tool registry duplication | 3 places | 1 place | Phase 2 |
| Event isolation | Shared | Per-conversation | Phase 1 |
| Core→UI coupling | TUI mapper in core | Zero | Phase 1 |
| Deprecated fields | 2 | 0 | Phase 3 |
| Interface usage in Agent | 20% | 80% | Phase 3 |
| Overall test coverage | Current | ≥85% | All phases |
| Dead code functions | 0 | 0 | All phases |

### Velocity Tracking

| Phase | Tasks | Estimated Days | Start Date | End Date | Status |
|-------|-------|----------------|------------|----------|--------|
| Phase 1 | 3 major tasks | 10 days | TBD | TBD | NOT STARTED |
| Phase 2 | 3 major tasks | 7 days | TBD | TBD | NOT STARTED |
| Phase 3 | 4 major tasks | 7 days | TBD | TBD | NOT STARTED |

---

## Risk Management

### High Risk Items

| Risk | Impact | Mitigation | Owner |
|------|--------|------------|-------|
| Agent refactor breaks existing functionality | HIGH | Comprehensive test suite, feature flags | TBD |
| Event isolation changes break UI | MEDIUM | Extensive integration tests, gradual rollout | TBD |
| Builder pattern adds complexity | LOW | Clear documentation, examples | TBD |

### Rollback Procedures

**Phase 1 Tasks:**
- Keep deprecated fields for 1 release
- Feature flags for new services
- Tag releases as preview versions

**Phase 2 Tasks:**
- Do not maintain old factory functions, remove them

**Phase 3 Tasks:**
- Low risk, direct refactoring acceptable

---

## Definition of Done (Per Task)

Before marking any task complete:

- [ ] FRD created and reviewed
- [ ] Code implements FRD specification
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] All E2E tests pass
- [ ] Test coverage ≥90% for new code
- [ ] `go test -race ./...` passes
- [ ] `make lint` passes (zero errors)
- [ ] `make deadcode` passes (zero dead functions)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc complete for all exports
- [ ] `docs/packages/` updated
- [ ] AGENTS.md updated if contracts changed
- [ ] Examples updated if API changed
- [ ] ROADMAP.md updated with completion date

---

## Notes

### Deviations from Original Plan

Document any deviations here as they occur:
- _None yet_

### Lessons Learned

Document insights as work progresses:
- _To be filled during execution_

### Follow-up Items

Track items that need attention post-refactoring:
- _To be determined_

---

## Appendix A: FRD Template

Each refactoring task must have an FRD following this structure:

```markdown
# FRD-YYYYMMDD-NNN: [Task Name]

## Metadata
- **Status**: DRAFT | REVIEW | APPROVED | IMPLEMENTED
- **Priority**: P0 | P1 | P2 | P3
- **Effort**: S | M | L | XL
- **Dependencies**: [List FRDs]

## Problem Statement
What anti-pattern are we fixing?

## Goals
What outcomes do we want?

## Non-Goals
What are we explicitly not doing?

## Design
How will we implement this?

## API Changes
What public APIs change?

## Testing Strategy
How do we ensure correctness?

## Acceptance Criteria
When is this done?

## Risks
What could go wrong?

## Alternatives Considered
What else did we consider?
```

---

## Appendix B: Checklist Automation

Consider automating checklist tracking:

```bash
# scripts/roadmap-status.sh
#!/bin/bash
# Parse ROADMAP.md and show completion percentage
# Can be integrated into CI/CD
```

---

**Status Legend:**
- ✅ Complete
- 🔄 In Progress
- ⏸️ Blocked
- ⏭️ Skipped
- ❌ Failed

**Last Updated:** 2025-10-19  
**Next Review:** After Phase 1 completion

