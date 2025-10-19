# Architectural Anti-Patterns Analysis

**Project:** Spin AI Coding Agent  
**Analysis Date:** 2025-10-19  
**Scope:** Internal architecture review  

## Executive Summary

The Spin project follows clean architecture principles overall with good dependency inversion and clear layering. However, several anti-patterns have been identified that increase complexity, reduce maintainability, and violate stated architectural goals.

**Severity Levels:**
- 🔴 **CRITICAL**: Immediate refactoring needed
- 🟡 **MEDIUM**: Should be addressed in next major refactor
- 🟢 **LOW**: Nice-to-have improvement

---

## 1. 🔴 God Object: Agent Struct

**Location:** `internal/core/agent.go:38-58`

### Problem

The `Agent` struct has grown into a god object with 17+ dependencies:

```go
type Agent struct {
    llm             llm.Provider           
    executor        *Executor              // DEPRECATED but still present
    validator       *Validator             
    context         *Environment           
    emitter         *EventEmitter          
    config          *Config                
    toolRegistry    *tools.Registry        
    taskRegistry    *task.Registry         
    approvalHandler ApprovalHandler        // DEPRECATED but still present
    approvalService *ApprovalService       
    toolExecutor    *ToolExecutor          
    cycleDetector   *cycle.Detector        
    patternDetector *cycle.PatternDetector 
    planner         *Plan                  
}
```

### Issues

1. **Violates SRP (Single Responsibility Principle)**: Agent handles orchestration, tool execution, approval management, cycle detection, and planning
2. **High coupling**: Changes in any subsystem ripple through Agent
3. **Difficult to test**: Requires mocking 17+ dependencies
4. **Cognitive overload**: Developer must understand all subsystems to work with Agent
5. **Deprecated fields still present**: `executor` and `approvalHandler` marked deprecated but not removed

### Impact

- Testing requires complex setup with many mocks
- Hard to add new features without touching Agent
- Violates stated principle: "No god objects" (AGENTS.md:57)

### Recommended Fix

**Option A: Extract Services (Preferred)**
```go
type Agent struct {
    llm              llm.Provider
    orchestrator     *Orchestrator     // Tool execution + planning
    securityService  *SecurityService  // Validation + approval
    detectionService *DetectionService // Cycle + pattern detection
    emitter          *EventEmitter
    config           *Config
}
```

**Option B: Use Composition**
```go
type Agent struct {
    core      *AgentCore      // LLM + config + events
    execution *ExecutionLayer // Tools + executor + validator
    security  *SecurityLayer  // Approval + validation
    detection *DetectionLayer // Cycle + pattern detection
}
```

---

## 2. 🟡 Duplicate Service Instantiation: Tool Registry

**Location:** `internal/core/agent.go:113-122` and `internal/core/manager.go:238-267`

### Problem

Tool registries are created in TWO places with DUPLICATE code:

**In Agent (NewAgent):**
```go
registry := tools.NewRegistry()
_ = registry.Register(tools.NewReadFileTool())
_ = registry.Register(tools.NewWriteFileTool())
_ = registry.Register(tools.NewListDirectoryTool())
_ = registry.Register(tools.NewExecuteCommandTool(executor, validator))
_ = registry.Register(tools.NewGetContextTool(context))
_ = registry.Register(tools.NewApplyPatchTool(context.WorkDir))
_ = registry.Register(tools.NewFileSearchTool(context.WorkDir))
_ = registry.Register(tools.NewGitContextTool(context.WorkDir))
```

**In TUI (cmd/spin/tui.go:238-247):**
```go
registry := tools.NewRegistry()
registry.Register(tools.NewReadFileTool())
registry.Register(tools.NewWriteFileTool())
registry.Register(tools.NewListDirectoryTool())
// Note: ExecuteCommandTool and GetContextTool registered by Agent
```

### Issues

1. **Code duplication**: Same registration logic in multiple places
2. **Inconsistent behavior**: Different initialization paths may register different tools
3. **Maintenance burden**: Adding a new built-in tool requires changes in 2+ places
4. **Hidden dependencies**: Agent re-registers tools even if registry is passed in
5. **Violates DRY**: "SOLID, DRY, KISS" (AGENTS.md:9)

### Impact

- Risk of tool registration divergence between modes
- Difficult to understand which tools are available in which context
- Increased test surface area

### Recommended Fix

```go
// internal/tools/builtin.go
func NewDefaultRegistry(executor *Executor, validator *Validator, env *Environment) *Registry {
    registry := NewRegistry()
    
    // File operations (no dependencies)
    _ = registry.Register(NewReadFileTool())
    _ = registry.Register(NewWriteFileTool())
    _ = registry.Register(NewListDirectoryTool())
    
    // Context-aware tools
    if executor != nil && validator != nil {
        _ = registry.Register(NewExecuteCommandTool(executor, validator))
    }
    if env != nil {
        _ = registry.Register(NewGetContextTool(env))
        _ = registry.Register(NewApplyPatchTool(env.WorkDir))
        _ = registry.Register(NewFileSearchTool(env.WorkDir))
        _ = registry.Register(NewGitContextTool(env.WorkDir))
    }
    
    return registry
}
```

---

## 3. 🟡 Shared Mutable State: EventEmitter

**Location:** `internal/core/event.go:207-217`, `internal/core/manager.go:24`

### Problem

The `EventEmitter` is created once in `Manager` and shared across `Manager`, `Agent`, `Conversation`, and `History`:

```go
// Manager creates emitter
type Manager struct {
    emitter *EventEmitter  // Shared across all conversations
    ...
}

// Passed to Agent
agent := NewAgent(llm, executor, validator, env, m.emitter, opts...)

// Passed to Conversation
conv := NewConversation(agent, hist, m.emitter)

// Passed to History
hist.SetEventEmitter(m.emitter)
```

### Issues

1. **Global state**: Single emitter shared across all conversations
2. **Event collision**: Events from different conversations intermixed
3. **Testing difficulty**: Cannot isolate conversation events in tests
4. **Race conditions**: While thread-safe, event ordering between conversations is non-deterministic
5. **Violates isolation**: Conversations should be independent

### Impact

- Events from Conversation A can appear in Conversation B's stream
- Cannot test conversations in parallel
- Debugging multi-conversation scenarios is difficult
- Memory leak risk if subscriptions aren't cleaned up

### Recommended Fix

**Option A: Per-Conversation Emitters (Preferred)**
```go
// Each conversation gets its own emitter
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
    // Create isolated emitter for this conversation
    emitter := NewEventEmitter(DefaultEventBufferSize)
    
    agent, err := m.buildAgent(executor, ctxEnv, emitter, logger)
    hist := m.createHistory(emitter)
    conv := NewConversation(agent, hist, emitter)
    
    return conv, nil
}
```

**Option B: Event Routing**
```go
// Add conversation ID to events and route appropriately
type Event struct {
    Type           EventType
    ConversationID string  // NEW: Isolate by conversation
    Timestamp      time.Time
    Data           interface{}
}
```

---

## 4. 🟡 Feature Envy: Manager Building Everything

**Location:** `internal/core/manager.go:70-347`

### Problem

`Manager` has intimate knowledge of how to build `Executor`, `Agent`, `Environment`, `History`, and all their dependencies through numerous private builder methods:

```go
func (m *Manager) buildExecutor(workDir string, logger *slog.Logger) (*Executor, error)
func (m *Manager) buildExecutorOptions(...) []ExecutorOption
func (m *Manager) gatherEnvironmentContext(...) (*Environment, error)
func (m *Manager) buildEnvironmentOptions() []EnvironmentOption
func (m *Manager) enrichEnvironmentWithIntegrations(...)
func (m *Manager) addGitContext(...)
func (m *Manager) addShellContext(...)
func (m *Manager) registerIntegrationTools(...) error
func (m *Manager) registerMCPTools(...) error
func (m *Manager) registerGitTools(...) error
func (m *Manager) registerShellTools(...) error
func (m *Manager) buildAgent(...) (*Agent, error)
func (m *Manager) buildAgentOptions(...) []AgentOption
func (m *Manager) createHistory() *History
```

### Issues

1. **Feature envy**: Manager knows too much about internal construction of other types
2. **Violates encapsulation**: Each type should know how to build itself
3. **High coupling**: Manager changes when Executor/Agent/Environment change
4. **Difficult to extend**: Adding new integrations requires modifying Manager
5. **Testing burden**: Must test all construction paths through Manager

### Impact

- Manager has 513 lines, mostly construction logic
- Cannot create Agent/Executor independently for testing
- Adding new features (e.g., new integration) requires Manager changes

### Recommended Fix

**Use Builder Pattern:**
```go
// internal/core/executor_builder.go
type ExecutorBuilder struct {
    workDir         string
    validator       *Validator
    approvalService *ApprovalService
    timeout         time.Duration
    cache           *CommandCache
}

func NewExecutorBuilder(workDir string) *ExecutorBuilder { ... }
func (b *ExecutorBuilder) WithValidator(v *Validator) *ExecutorBuilder { ... }
func (b *ExecutorBuilder) WithApproval(a *ApprovalService) *ExecutorBuilder { ... }
func (b *ExecutorBuilder) Build() (*Executor, error) { ... }

// Similar for AgentBuilder, EnvironmentBuilder, etc.

// Manager delegates to builders
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error) {
    executor := NewExecutorBuilder(workDir).
        WithValidator(NewValidator()).
        WithApproval(m.buildApprovalService()).
        WithTimeout(m.cfg.Timeout).
        Build()
    
    agent := NewAgentBuilder(m.llm).
        WithExecutor(executor).
        WithEnvironment(env).
        WithTools(m.toolRegistry).
        Build()
    
    // ...
}
```

---

## 5. 🟢 Deprecated Fields Not Removed

**Location:** `internal/core/agent.go:45,52`

### Problem

Agent carries deprecated fields that are no longer used:

```go
type Agent struct {
    executor        *Executor       // deprecated - use toolExecutor
    approvalHandler ApprovalHandler // deprecated - use approvalService
    ...
    approvalService *ApprovalService // NEW replacement
    toolExecutor    *ToolExecutor    // NEW replacement
}
```

### Issues

1. **Confusion**: Developers don't know which to use
2. **Memory waste**: Storing unused references
3. **Maintenance burden**: Must maintain compatibility
4. **Violates "zero dead code"**: (AGENTS.md:9)

### Recommended Fix

Remove deprecated fields and update all callers to use new services. If backward compatibility is needed, provide adapter functions:

```go
// Deprecated: Use approvalService.RequestApproval instead
func (a *Agent) RequestApproval(ctx context.Context, cmd string) (bool, error) {
    return a.approvalService.RequestApproval(ctx, Operation{Command: cmd})
}
```

Then remove in next major version.

---

## 6. 🟢 Lack of Interface Segregation: Agent Dependencies

**Location:** `internal/core/agent.go:44-58`

### Problem

Agent depends on concrete types rather than interfaces for most dependencies:

```go
type Agent struct {
    executor        *Executor              // Concrete type
    validator       *Validator             // Concrete type
    context         *Environment           // Concrete type
    emitter         *EventEmitter          // Concrete type
    toolRegistry    *tools.Registry        // Concrete type
    taskRegistry    *task.Registry         // Concrete type
    approvalService *ApprovalService       // Concrete type
    toolExecutor    *ToolExecutor          // Concrete type
    cycleDetector   *cycle.Detector        // Concrete type
    patternDetector *cycle.PatternDetector // Concrete type
}
```

### Issues

1. **Hard to mock**: Testing requires creating real instances
2. **Tight coupling**: Agent depends on implementation details
3. **Violates DIP**: Dependency Inversion Principle not followed
4. **Contradicts documentation**: "Interfaces at boundaries only" but Agent is at boundary (AGENTS.md:54)

### Impact

- Test setup is complex
- Cannot easily swap implementations
- Hard to add alternative implementations (e.g., different validation strategies)

### Recommended Fix

```go
// Define minimal interfaces
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

// Agent depends on interfaces
type Agent struct {
    llm             llm.Provider
    executor        CommandExecutor        // Interface
    validator       CommandValidator       // Interface
    emitter         EventBus               // Interface
    ...
}
```

---

## 7. 🟡 Incomplete Abstraction: Task Registry Duplication

**Location:** `internal/core/agent.go:124-139` and `cmd/spin/tui.go:234-266`

### Problem

Task registry initialization is duplicated with identical code:

**In Agent:**
```go
taskRegistry := task.NewRegistry()
if err := taskRegistry.Register("regular", task.NewRegular()); err != nil { ... }
if err := taskRegistry.Register("review", task.NewReview()); err != nil { ... }
if err := taskRegistry.Register("compact", task.NewCompact()); err != nil { ... }
if err := taskRegistry.Register("planning", task.NewPlanning()); err != nil { ... }
if err := taskRegistry.SetDefault("regular"); err != nil { ... }
```

This same code appears in multiple places, violating DRY.

### Recommended Fix

```go
// internal/core/task/registry.go
func NewDefaultRegistry() (*Registry, error) {
    registry := NewRegistry()
    
    // Register all built-in task modes
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

---

## 8. 🟢 Anemic Domain Model: Config Struct

**Location:** `internal/core/config.go`

### Problem

The `Config` struct is a data bag with no behavior, requiring external validation and manipulation:

```go
type Config struct {
    Provider    string
    Model       string
    MaxTurns    int
    Timeout     time.Duration
    // ... 20+ fields
}

func (c *Config) Validate() error { ... } // Only method
```

All config merging, defaulting, and manipulation happens externally in Manager and command handlers.

### Issues

1. **Anemic model**: Config has data but no behavior
2. **Scattered logic**: Config manipulation spread across codebase
3. **Hard to maintain**: Must find all places that touch config

### Recommended Fix

```go
type Config struct {
    // ... fields
}

// Add methods that encapsulate config behavior
func (c *Config) WithDefaults() *Config { ... }
func (c *Config) Merge(other *Config) *Config { ... }
func (c *Config) ForProvider(provider string) *ProviderConfig { ... }
func (c *Config) ExecutorOptions() []ExecutorOption { ... }
func (c *Config) AgentOptions() []AgentOption { ... }
```

---

## 9. 🔴 Hidden Coupling: TUI → Core → TUI Mapper

**Location:** `cmd/spin/tui.go:97-98` and `internal/core/tui_mapper.go`

### Problem

The TUI creates a `TUIMapper` that depends on core events, but core has a `TUIMapper` type, creating circular dependency via type coupling:

```
cmd/spin/tui.go
    ↓ imports
internal/ui/adapters
    ↓ imports
internal/core (for Event types)
    ↓ contains
internal/core/tui_mapper.go (knows about TUI specifics)
```

### Issues

1. **Circular dependency** (type-level): Core knows about TUI concepts
2. **Violates layering**: Core (domain) depends on UI concerns
3. **Hard to test core independently**: TUI mapper in core package
4. **Violates clean architecture**: "domain first, adapters second, frameworks last" (AGENTS.md:53)

### Recommended Fix

Move `tui_mapper.go` OUT of core package:

```
internal/ui/adapters/core_event_mapper.go  // UI layer translates core events
internal/core/                              // Pure domain, no UI knowledge
```

Or use pure adapter pattern:

```go
// internal/ui/adapters/event_adapter.go
type CoreEventAdapter struct {
    ui        UI
    coreEvents <-chan core.Event
}

func (a *CoreEventAdapter) Start() {
    for event := range a.coreEvents {
        a.adaptToUI(event)
    }
}
```

---

## 10. 🟢 Primitive Obsession: String-based Event Types

**Location:** `internal/core/event.go:36-65`

### Problem

Event types use primitive int with string conversion instead of type-safe enums:

```go
type EventType int

const (
    EventContentDelta EventType = iota
    ...
)

func (e EventType) String() string {
    names := []string{"content_delta", ...}
    return names[e]
}
```

### Issues

1. **Type safety**: Can accidentally pass wrong int
2. **Magic numbers**: Int values have no meaning
3. **Runtime errors**: String() can panic if int out of range
4. **Hard to extend**: Adding events requires updating array

### Recommended Fix

Use type-safe enum with explicit values:

```go
type EventType string

const (
    EventContentDelta    EventType = "content_delta"
    EventContentComplete EventType = "content_complete"
    EventToolCallStart   EventType = "tool_call_start"
    // ...
)

func (e EventType) Valid() bool {
    switch e {
    case EventContentDelta, EventContentComplete, EventToolCallStart, ...:
        return true
    default:
        return false
    }
}
```

---

## Summary Table

| # | Anti-Pattern | Severity | Location | Impact | Effort |
|---|-------------|----------|----------|--------|--------|
| 1 | God Object: Agent | 🔴 Critical | `core/agent.go` | High coupling, hard to test | High |
| 2 | Duplicate Tool Registry | 🟡 Medium | `core/agent.go`, `cmd/spin/tui.go` | Code duplication, maintenance burden | Medium |
| 3 | Shared Mutable State: EventEmitter | 🟡 Medium | `core/manager.go`, `core/event.go` | Event collision, testing issues | Medium |
| 4 | Feature Envy: Manager | 🟡 Medium | `core/manager.go` | High coupling, poor encapsulation | High |
| 5 | Deprecated Fields | 🟢 Low | `core/agent.go` | Confusion, memory waste | Low |
| 6 | Lack of Interface Segregation | 🟢 Low | `core/agent.go` | Hard to test, tight coupling | Medium |
| 7 | Task Registry Duplication | 🟡 Medium | `core/agent.go`, `cmd/spin/tui.go` | Code duplication | Low |
| 8 | Anemic Config | 🟢 Low | `core/config.go` | Scattered logic | Medium |
| 9 | Hidden Coupling: TUI Mapper | 🔴 Critical | `core/tui_mapper.go` | Violates clean architecture | Medium |
| 10 | Primitive Obsession: Events | 🟢 Low | `core/event.go` | Type safety issues | Low |

---

## Recommended Refactoring Priority

### Phase 1: Critical Issues (Immediate)
1. **Extract Agent services** - Break god object into smaller services
2. **Move TUI Mapper** - Remove UI concerns from core package
3. **Isolate EventEmitter** - Per-conversation emitters or event routing

### Phase 2: Medium Issues (Next Sprint)
4. **Consolidate Tool Registry** - Single source of truth for default tools
5. **Consolidate Task Registry** - Single initialization function
6. **Extract Builder Pattern** - Move construction logic out of Manager

### Phase 3: Low Priority (Technical Debt)
7. **Remove deprecated fields** - Clean up Agent struct
8. **Add interfaces** - Interface segregation for Agent dependencies
9. **Enrich Config** - Add behavior to Config struct
10. **Type-safe Events** - Use string-based enum for EventType

---

## Conclusion

The Spin project has solid fundamentals but has accumulated technical debt through incremental feature addition. The primary issues are:

1. **Agent as God Object** - Needs decomposition
2. **Manager as Factory** - Needs builder pattern
3. **Shared State** - EventEmitter needs isolation
4. **Layer Violation** - TUI mapper in core package

Addressing these will significantly improve maintainability, testability, and adherence to the project's stated architectural principles.

