# Spin Refactoring Roadmap

**Created:** 2026-01-19  
**Author:** Architecture Analysis  
**Status:** Draft

## Executive Summary

This roadmap identifies code duplications, architectural inconsistencies, and design issues discovered during comprehensive codebase analysis. The spin project has grown organically and accumulated technical debt that impacts maintainability, testability, and future extensibility.

---

## Table of Contents

1. [Critical Issues (P0)](#1-critical-issues-p0)
2. [High Priority Issues (P1)](#2-high-priority-issues-p1)
3. [Medium Priority Issues (P2)](#3-medium-priority-issues-p2)
4. [Low Priority Issues (P3)](#4-low-priority-issues-p3)
5. [Refactoring Phases](#5-refactoring-phases)
6. [Detailed Issue Catalog](#6-detailed-issue-catalog)

---

## 1. Critical Issues (P0)

### 1.1 Duplicate ToolCall Data Types ✅ COMPLETED

**Problem:** `ToolCallStartData` and `ToolCallCompleteData` defined in two locations with identical fields.

**Files:**
- `internal/agent/request.go:83-100` (REMOVED)
- `internal/events/event.go:206-230` (CANONICAL)

**Impact:** Changes in one location won't propagate to the other. Potential runtime type mismatches.

**Solution:** Consolidate to single definition in `internal/events/event.go`, remove duplicates from agent package.

**Resolution (2026-01-19):**
- Removed duplicate `ToolCallStartData`, `ToolCallCompleteData`, `EventType`, and `Event` types from `internal/agent/request.go`
- All usages already used `events` package types (the agent types were dead code)
- FRD: `specs/frds/FRD-20260119-001-toolcall-type-consolidation.md`

---

### 1.2 Triple ToolResult Definition ✅ COMPLETED

**Problem:** Three different `ToolResult` structs exist with overlapping but inconsistent fields.

**Files:**
- `internal/tools/tool.go:39` - Canonical tool result
- `internal/agent/tool_runtime.go:21` - Agent's version (REMOVED)
- `internal/agent/request.go` - Request processing version (REMOVED in 1.1)

**Impact:** Confusion about which type to use, conversion overhead, potential data loss.

**Solution:** Use single `tools.ToolResult` everywhere. Remove agent-specific definitions.

**Resolution (2026-01-19):**
- Extended `tools.ToolResult` with `ID`, `ExitCode`, and `Err` (error type) fields
- Removed duplicate `ToolResult` struct from `internal/agent/tool_runtime.go`
- Created type alias in agent package: `type ToolResult = tools.ToolResult`
- Added helper constructors: `NewToolResult`, `NewToolError`, `NewToolErrorWithID`
- Added fluent methods: `WithID`, `WithExitCode`, `WithMetadata`, `GetErr`, `String`
- Updated all usages in `agent.go` and `tool_runtime.go` to use unified type
- FRD: `specs/frds/FRD-20260119-002-toolresult-consolidation.md`


---

## 2. High Priority Issues (P1)

### 2.1 Duplicate MCP Manager Naming ✅ COMPLETED

**Problem:** Two `MCPManager` types serve completely different purposes.

**Files:**
- `internal/mcp/manager.go:38` - Runtime server management (RENAMED)
- `internal/config/mcp_manager.go:19` - Configuration management (RENAMED)

**Impact:** Confusing imports, naming collisions, unclear responsibilities.

**Solution:** 
- Rename config version to `MCPConfigStore`
- Rename runtime version to `MCPServerManager`

**Resolution (2026-01-19):**
- Renamed `MCPManager` in `internal/mcp/manager.go` to `MCPServerManager`
- Renamed `MCPManager` in `internal/config/mcp_manager.go` to `MCPConfigStore`
- Updated all usages across the codebase (agent, protocol/acp, cmd/spin, tests)
- Updated error constant from `ErrNilMCPManager` to `ErrNilMCPServerManager`
- FRD: `specs/frds/FRD-20260119-003-mcp-manager-naming.md`

---

### 2.2 Inconsistent Error Types ✅ COMPLETED

**Problem:** Three different error type patterns without unified approach.

**Files:**
- `internal/errors/errors.go:32` - Generic error with Code, Op, Err, Message
- `internal/git/errors.go:27` - PatchError with different fields
- `internal/patchapply/applier.go:78` - Another Error with different fields

**Impact:** No consistent error handling, difficult to wrap/unwrap errors uniformly.

**Solution:** Create base error interface in `internal/errors/`:
```go
type SpinError interface {
    error
    GetCode() ErrorCode
    Operation() string
    Unwrap() error
}
```

**Resolution (2026-01-20):**
- Added `SpinError` interface to `internal/errors/errors.go` with `GetCode()`, `Operation()`, `Unwrap()` methods
- Added `GetCode()` and `Operation()` methods to `Error` struct implementing `SpinError`
- Added new error codes: `CodePatch`, `CodeGit`, `CodeContextMismatch`
- 100% test coverage achieved
- FRD: `specs/frds/FRD-20260120-001-unified-error-interface.md`

---

### 2.3 Interface Pollution - Curator ✅ COMPLETED

**Problem:** Curator interface has 9 methods mixing different concerns.

**File:** `internal/ace/curator/curator.go`

**Methods:**
- Curate, CurateBatch, Refine (curation)
- FindDuplicates (deduplication)
- ApplyBulletFeedback, UpdateBulletContent, AddBulletTag, RemoveBulletTag, UpdateBulletEmbedding (updates)

**Solution:** Split into three interfaces:
```go
type BulletMerger interface {
    Curate(...) 
    CurateBatch(...)
    FindDuplicates(...)
}

type BulletRefiner interface {
    Refine(...)
}

type BulletUpdater interface {
    ApplyBulletFeedback(...)
    UpdateBulletContent(...)
    AddBulletTag(...)
    RemoveBulletTag(...)
    UpdateBulletEmbedding(...)
}
```

**Resolution (2026-01-20):**
- Added `BulletMerger` interface for curation/deduplication methods (Curate, CurateBatch, FindDuplicates)
- Added `BulletRefiner` interface for refinement/pruning methods (Refine)
- Added `BulletUpdater` interface for bullet modification methods (ApplyBulletFeedback, UpdateBulletContent, AddBulletTag, RemoveBulletTag, UpdateBulletEmbedding)
- Redefined `Curator` as composite interface embedding all three for backward compatibility
- All existing tests pass, no breaking changes to consumers
- FRD: `specs/frds/FRD-20260120-002-curator-interface-segregation.md`

---

### 2.4 Interface Pollution - Runtime

**Problem:** Runtime interface mixes 8 unrelated concerns.

**File:** `internal/agent/runtime/runtime.go`

**Solution:** Split into:
```go
type ToolRegistrar interface {
    RegisterTools(registry *tools.Registry)
}

type NotificationProvider interface {
    NotificationSender() NotificationSender
}

type SessionProvider interface {
    SessionStorage() session.Storage
    SessionID() string
}

type ApprovalProvider interface {
    ApprovalHandler() security.ApprovalHandler
}

type TerminalProvider interface {
    SupportsTerminals() bool
    TerminalClient() TerminalClient
}
```

---

### 2.5 Scattered Approval Logic

**Problem:** Approval decision-making spread across multiple locations.

**Files:**
- `internal/security/validator.go:145` - NeedsApproval()
- `internal/tools/write_file.go:77` - CheckApproval() 
- `internal/security/approval.go:58` - RequestApproval()
- `cmd/spin/approval_handlers.go` - Policy store building

**Impact:** Duplicated validation logic, inconsistent approval decisions.

**Solution:** Centralize in single `ApprovalDecisionService`:
```go
type ApprovalDecisionService interface {
    ShouldApprove(ctx context.Context, op Operation) (ApprovalDecision, error)
    RequestApproval(ctx context.Context, op Operation) (ApprovalResponse, error)
}
```

---

## 3. Medium Priority Issues (P2)

### 3.1 Duplicate Service Wrapper Pattern

**Problem:** Git, Shell, MCP services have identical boilerplate patterns.

**Files:**
- `internal/git/service.go:10`
- `internal/shell/service.go:11`
- `internal/mcp/service.go:12`

**Pattern:**
```go
type Service struct {
    impl *Implementation
}

func NewService(enabled bool, ...) (*Service, error) {
    if !enabled {
        return &Service{}, nil
    }
    // initialize implementation
}

func (s *Service) IsEnabled() bool { return s.impl != nil }
func (s *Service) Close() error { ... }
```

**Solution:** Extract generic `ServiceWrapper[T]` type:
```go
type ServiceWrapper[T any] struct {
    impl T
    enabled bool
}

func NewServiceWrapper[T any](enabled bool, factory func() (T, error)) (*ServiceWrapper[T], error)
```

---

### 3.2 Inconsistent Builder Patterns

**Problem:** Three builder implementations with different APIs.

**Files:**
- `internal/agent/builder.go:16` - Full fluent API with many With* methods
- `internal/conversation/builder.go:25` - Partial fluent API
- `internal/llm/builder/builder.go:14` - Minimal NewBuilder() + Build()

**Impact:** Learning curve, inconsistent usage patterns.

**Solution:** Standardize on fluent builder pattern with functional options:
```go
type Builder struct { ... }

func NewBuilder(opts ...Option) *Builder
func (b *Builder) Build() (*T, error)

type Option func(*Builder)
func WithConfig(c Config) Option
func WithProvider(p Provider) Option
```

---

### 3.3 Dual Error Representation in Tools

**Problem:** Tool interface returns `(ToolResult, error)` but ToolResult also has `Error` field.

**File:** `internal/tools/tool.go`

**Impact:** Callers must check both `err != nil` and `result.Error != ""`.

**Solution:** Use single error channel:
```go
// Option A: Only return error
func (t *Tool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error)
// where error indicates execution failure, ToolResult.Success indicates tool outcome

// Option B: ToolResult wraps all outcomes
func (t *Tool) Execute(ctx context.Context, params ToolParameters) ToolResult
// where ToolResult has Err() method
```

---

### 3.4 Type Aliases Instead of Proper Imports

**Problem:** Type aliases used to re-export types from other packages.

**File:** `internal/agent/tool_runtime.go:14-17`

```go
type ToolCall = message.ToolCall
type ToolCallFunction = message.ToolCallFunction
```

**Impact:** Indicates unclear type ownership, adds indirection.

**Solution:** Import and use types directly without aliasing. If aliasing is needed for API stability, document why.

---

### 3.5 Inconsistent Context Usage

**Problem:** context.Context used inconsistently across packages.

**Examples:**
- `internal/security/policy.go` - PolicyStore requires ctx, memoryPolicyStore ignores it
- `internal/ace/generator/generator.go` - ctx not propagated to helper methods
- `internal/shell/context.go` - Some methods accept ctx, some don't

**Solution:** Audit all interfaces. Every method that does I/O, network, or may block should accept context as first parameter. Propagate context through all call chains.

---

### 3.6 Inconsistent Naming Conventions

**Problem:** Multiple naming styles for same concepts.

**Examples:**
- Adapters: `ValidatorAdapter` vs `validatorAdapter` (PascalCase vs camelCase)
- Methods: `GetIntegration()` vs `Integration()` vs `ApprovalHandler()` 
- Interfaces: `CommandValidator` vs `Validator` vs `Tool`

**Solution:** Establish naming conventions:
- Exported types: PascalCase
- Unexported types: camelCase
- Getters: Omit "Get" prefix (`Handler()` not `GetHandler()`)
- Interfaces: Single-method interfaces use `-er` suffix (`Validator`, `Executor`)

---

## 4. Low Priority Issues (P3)

### 4.1 Tool Implementation Boilerplate

**Problem:** All tools follow identical pattern with only Execute() differing.

**Files:** `internal/tools/*.go` (ReadFileTool, WriteFileTool, ListDirectoryTool, etc.)

**Pattern:**
```go
type XTool struct { deps }
func (t *XTool) Name() string { return "x" }
func (t *XTool) Description() string { return "..." }
func (t *XTool) Schema() Schema { return ... }
func (t *XTool) Execute(...) (ToolResult, error) { ... }
```

**Solution:** Consider code generation or functional tool definition:
```go
func NewTool(name, desc string, schema Schema, execute ExecuteFunc) Tool
```

---

### 4.2 Thin Wrapper Interfaces

**Problem:** Some interfaces are thin wrappers that add no abstraction value.

**Example:** `internal/ace/retrieval/retriever.go`
```go
func (r *SemanticRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
    return r.playbook.Search(ctx, query, topK) // just delegates
}
```

**Solution:** Have `Playbook` implement `Retriever` interface directly instead of wrapping.

---

### 4.3 Multiple Notification Systems

**Problem:** Three overlapping event/notification systems.

**Files:**
- `internal/agent/runtime/notifications.go` - NotificationSender
- `internal/conversation/events.go` - EventTransformer
- `internal/events/event.go` - Event types

**Solution:** Consolidate into single event system in `internal/events/`. All components emit to single EventEmitter.

---

### 4.4 Policy Store Return Pattern

**Problem:** PolicyStore.Get returns `(Policy, bool, error)` instead of `(Policy, error)`.

**File:** `internal/security/policy.go`

**Solution:** Use `ErrNotFound` error instead of bool:
```go
type PolicyStore interface {
    Get(ctx context.Context, key PolicyKey, scope string) (Policy, error)
    // returns ErrPolicyNotFound if not found
}
```

---

### 4.5 ACE Package Concrete Dependencies

**Problem:** ACE components depend on concrete types instead of interfaces.

**Files:**
- `internal/ace/curator/curator.go` - depends on `*playbook.Playbook`
- `internal/ace/generator/generator.go` - depends on `prompt.Builder` concrete
- `internal/ace/reflector/reflector.go` - depends on `feedback.Parser` concrete

**Solution:** Define interfaces for all dependencies. Inject via constructor.

---

## 5. Refactoring Phases

### Phase 1: Type Consolidation (P0 issues) ✅ COMPLETED

**Goal:** Eliminate duplicate type definitions.

**Tasks:**
1. [x] Consolidate ToolCallStartData/ToolCallCompleteData to events package (2026-01-19)
2. [x] Consolidate ToolResult to tools package (2026-01-19)

**Estimated Files:** 8-10  
**Risk:** Medium (type changes affect many consumers)

---

### Phase 2: Interface Cleanup (P1 issues)

**Goal:** Split bloated interfaces, unify naming.

**Tasks:**
1. [x] Rename MCP managers for clarity (2026-01-19)
2. [x] Create unified error interface (2026-01-20)
3. [x] Split Curator interface (2026-01-20)
4. [ ] Split Runtime interface
5. [ ] Centralize approval logic

**Estimated Files:** 12-15  
**Risk:** Medium (interface changes require all implementations to update)

---

### Phase 3: Pattern Standardization (P2 issues)

**Goal:** Consistent patterns across codebase.

**Tasks:**
1. [ ] Extract generic ServiceWrapper
2. [ ] Standardize builder patterns
3. [ ] Fix dual error representation
4. [ ] Audit and fix context usage
5. [ ] Enforce naming conventions

**Estimated Files:** 15-20  
**Risk:** Low-Medium (mostly internal changes)

---

### Phase 4: Code Cleanup (P3 issues)

**Goal:** Remove redundancy, simplify architecture.

**Tasks:**
1. [ ] Consider tool code generation
2. [ ] Remove thin wrapper interfaces
3. [ ] Consolidate notification systems
4. [ ] Fix policy store return patterns
5. [ ] Add interfaces to ACE components

**Estimated Files:** 10-15  
**Risk:** Low (cleanup and optimization)

---

## 6. Detailed Issue Catalog

### Summary Statistics

| Category | Count | Priority |
|----------|-------|----------|
| Duplicate Types | 4 | P0 |
| Interface Pollution | 4 | P1 |
| Inconsistent Error Types | 3 | P1 |
| Scattered Logic | 2 | P1 |
| Service Pattern Duplication | 3 | P2 |
| Builder Inconsistency | 3 | P2 |
| Naming Inconsistency | 5+ | P2 |
| Thin Wrappers | 3 | P3 |
| **TOTAL** | **32+** | - |

### Files Most Affected

| File | Issue Count | Priority Impact |
|------|-------------|-----------------|
| `internal/agent/request.go` | 2 | P0 |
| `internal/events/event.go` | 1 | P0 |
| `internal/agent/tool_runtime.go` | 2 | P0, P2 |
| `internal/conversation/adapters.go` | 1 | P0 |
| `internal/agent/runtime/adapters.go` | 1 | P0 |
| `internal/agent/runtime/runtime.go` | 2 | P0, P1 |
| `internal/ace/curator/curator.go` | 2 | P1, P3 |
| `internal/security/approval.go` | 2 | P1 |
| `internal/tools/tool.go` | 2 | P0, P2 |

---

## Architecture Recommendations

### 1. Create Contracts Package

```
internal/contracts/
├── executor.go      # CommandExecutor, ToolExecutor
├── validator.go     # CommandValidator, ToolValidator  
├── provider.go      # LLMProvider, EmbeddingProvider
├── storage.go       # SessionStorage, PolicyStorage
└── events.go        # EventEmitter, EventHandler
```

This breaks circular dependencies and provides clear interface contracts.

### 2. Reorganize ACE Package

```
internal/ace/
├── bullet/           # Bullet domain type
├── playbook/         # Storage and retrieval
├── learning/         # Generator + Reflector combined
│   ├── generator.go
│   └── reflector.go
├── curation/         # Curator split interfaces
│   ├── merger.go
│   ├── refiner.go
│   └── updater.go
└── trajectory/       # Execution tracking
```

### 3. Standardize Service Layer

All services should follow:
```go
// Constructor
func NewService(cfg Config, deps Dependencies) (*Service, error)

// Lifecycle
func (s *Service) Start(ctx context.Context) error
func (s *Service) Stop(ctx context.Context) error

// Health
func (s *Service) IsReady() bool
func (s *Service) Health() HealthStatus
```

---

## Next Steps

1. **Review** this roadmap with team
2. **Prioritize** based on current pain points
3. **Create issues** for Phase 1 tasks
4. **Start** with type consolidation (lowest risk, highest impact)
5. **Add tests** before any refactoring work

---

## References

- Go Project Layout: https://github.com/golang-standards/project-layout
- Effective Go: https://go.dev/doc/effective_go
- SOLID Principles in Go: https://dave.cheney.net/2016/08/20/solid-go-design
- Interface Segregation: https://en.wikipedia.org/wiki/Interface_segregation_principle
