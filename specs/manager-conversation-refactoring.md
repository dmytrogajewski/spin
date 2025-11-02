# Manager/Conversation/Agent Refactoring Plan

## Executive Summary

This document outlines a comprehensive refactoring plan to eliminate the `manager` package and restructure the conversation/integration architecture following established patterns in the codebase.

**Current Problems:**
- `Manager` package exists solely as a factory with no independent value
- Integrations (Git, Shell, MCP) are tightly coupled to Manager
- Three-layer indirection: Manager → Conversation → Agent
- Inconsistent with the tools pattern used elsewhere

**Proposed Solution:**
- Eliminate `manager` package entirely
- Extract integrations into independent services (following tools pattern)
- `Conversation` becomes the primary entry point
- Clean dependency injection via builder pattern

**Benefits:**
- ✅ Simpler API (2 steps instead of 3)
- ✅ Consistent with existing tools/service pattern
- ✅ Better testability (mock individual services)
- ✅ Clear separation of concerns
- ✅ No loss of functionality

---

## Table of Contents

1. [Current Architecture Analysis](#1-current-architecture-analysis)
2. [Problems Identified](#2-problems-identified)
3. [Proposed Architecture](#3-proposed-architecture)
4. [Implementation Plan](#4-implementation-plan)
5. [Migration Strategy](#5-migration-strategy)
6. [Testing Strategy](#6-testing-strategy)
7. [Rollback Plan](#7-rollback-plan)

---

## 1. Current Architecture Analysis

### 1.1 Package Structure

```
internal/
├── manager/                    # PROBLEM: Factory with no independent value
│   ├── manager.go             # Public API, NewConversation, Close
│   ├── executor.go            # Builds executor
│   ├── environment.go         # Gathers Git/Shell context
│   ├── tools.go               # Registers tools
│   ├── agent.go               # Builds agent with services
│   ├── history.go             # Creates history
│   ├── adapters.go            # Validator/shell/executor adapters
│   ├── events.go              # JSONL event logger
│   ├── builder.go             # Builder pattern
│   └── config.go              # Configuration
│
├── conversation/              # THIN: Just coordinates agent/history
│   └── conversation.go        # 160 lines, mostly delegation
│
├── agent/                     # CORE: LLM interaction & execution
│   ├── agent.go               # Main agent logic
│   ├── loop.go                # Turn execution
│   └── executor.go            # Command execution
│
├── git/                       # Integration service
│   └── integration.go         # Git repository context
│
├── shell/                     # Integration service
│   └── context.go             # Shell environment
│
└── mcp/                       # Integration service
    └── manager.go             # MCP server connections
```

### 1.2 Current Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    APPLICATION (TUI/CLI)                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │     Manager     │ ← PROBLEM: Unnecessary middleman
                  │  (Factory Only) │
                  └────────┬────────┘
                           │
                   ┌───────┴────────┐
                   │                │
        Builds Integrations    Builds Conversation
                   │                │
                   ▼                ▼
        ┌──────────────────┐  ┌──────────────────┐
        │  Git/Shell/MCP   │  │  Conversation    │
        │   Integrations   │  │ (Thin Wrapper)   │
        └──────────────────┘  └────────┬─────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │      Agent      │
                              │  (Core Logic)   │
                              └─────────────────┘
```

### 1.3 Manager Responsibilities

**Current Manager owns:**
```go
type Manager struct {
    cfg              *Config
    llm              llm.Provider
    emitter          *events.EventEmitter
    storage          session.Storage
    toolRegistry     *tools.Registry
    taskRegistry     *orchestration.Registry
    approvalHandler  security.ApprovalHandler
    authManager      *auth.Manager
    mcpManager       *mcp.MCPManager        // Integration
    gitIntegration   *git.GitIntegration    // Integration
    shellIntegration *shell.Context         // Integration
    logger           *slog.Logger
}
```

**Manager does:**
1. ✅ Initialize integrations (Git, Shell, MCP)
2. ✅ Build executor
3. ✅ Gather environment context
4. ✅ Build agent with services
5. ✅ Create history
6. ✅ Wire everything together
7. ❌ **But has no independent lifecycle or value**

### 1.4 Conversation Responsibilities

**Current Conversation owns:**
```go
type Conversation struct {
    agent     *agent.Agent
    history   *history.History
    emitter   *events.EventEmitter
    taskMode  string
    sessionID string
}
```

**Conversation does:**
1. ✅ Manages turn-by-turn execution
2. ✅ Coordinates history and agent
3. ✅ Handles task mode switching
4. ❌ **Too thin - only 160 lines**

### 1.5 Tool Connection Flow

```
Manager.buildToolRegistry()
    ├─► Register stateless tools
    │   ├─► tools.NewReadFileTool()
    │   ├─► tools.NewWriteFileTool()
    │   └─► tools.NewListDirectoryTool()
    │
    ├─► Register tools with dependencies
    │   └─► tools.NewShellCommandTool(validator, shellCtx, executor)
    │
    └─► registerIntegrationTools()
        ├─► For each MCP tool: registry.Register(tool)
        └─► registry.Register(tools.NewGitOperationTool(gitIntegration))
                                                         ^^^^^^^^^^^^^^
                                                         Injected at construction
```

**Key Pattern:** Tools receive dependencies via constructor injection

---

## 2. Problems Identified

### 2.1 Manager is a Middleman Anti-Pattern

**Problem:**
```go
// Current: 3 steps to use
manager := manager.NewManager(cfg)           // Step 1: Create manager
conv := manager.NewConversation(ctx, workDir) // Step 2: Create conversation
conv.RunTurn(ctx, input)                     // Step 3: Use conversation
```

**Evidence:**
- Manager's **only** public method is `NewConversation()`
- Manager is created solely to create Conversation
- Manager has no independent lifecycle after Conversation creation
- No tests create Manager without immediately creating Conversation

### 2.2 Integration Coupling

**Problem:** Integrations are tightly coupled to Manager

```go
// Integrations created inside Manager
func (b *Builder) Build() (*Manager, error) {
    // Initialize MCP
    if cfg.EnableMCP {
        mcpMgr := mcp.NewMCPManager(...)
        mcpMgr.Initialize(ctx)
    }

    // Initialize Git
    if cfg.EnableGit {
        gitInt := git.NewGitIntegration(...)
        gitInt.Initialize(ctx)
    }

    // Return Manager with integrations
    return &Manager{
        mcpManager: mcpMgr,
        gitIntegration: gitInt,
        // ...
    }
}
```

**Issues:**
- Integrations cannot be reused across conversations
- Difficult to test Manager without initializing all integrations
- Integrations closed when Manager closes (even if still needed)

### 2.3 Inconsistent with Tools Pattern

**Existing tools pattern:**
```go
// Tools receive services via constructor injection
gitTool := tools.NewGitOperationTool(gitIntegration)
shellTool := tools.NewShellCommandTool(validator, shellCtx, executor)
```

**Current integration pattern:**
```go
// Manager owns integrations and passes them internally
// NOT consistent with dependency injection pattern
```

### 2.4 Poor Testability

**Problems:**
- Cannot mock Manager easily (too many responsibilities)
- Cannot test Conversation with mock integrations
- Integration initialization tied to Manager construction
- Difficult to test partial configurations

### 2.5 Unclear Ownership

**Questions the current architecture raises:**
- Who owns integrations lifetime? Manager or Application?
- Can integrations be shared between conversations? (Should but can't)
- Should Conversation close integrations? (Currently doesn't)
- Should Manager exist after creating Conversation? (Currently does nothing)

---

## 3. Proposed Architecture

### 3.1 New Package Structure

```
internal/
├── conversation/              # PRIMARY: Entry point + lifecycle
│   ├── conversation.go        # Main conversation logic
│   ├── builder.go             # Builder pattern for construction
│   ├── executor.go            # Builds executor (from manager)
│   ├── environment.go         # Gathers environment (from manager)
│   ├── tools.go               # Registers tools (from manager)
│   ├── agent.go               # Builds agent (from manager)
│   ├── history.go             # Creates history (from manager)
│   ├── adapters.go            # Adapters (from manager)
│   └── events.go              # Event logging (from manager)
│
├── agent/                     # UNCHANGED
│   └── agent.go               # Core agent logic
│
├── git/                       # Add service wrapper
│   ├── service.go             # NEW: GitService
│   └── integration.go         # EXISTING: Implementation
│
├── shell/                     # Add service wrapper
│   ├── service.go             # NEW: ShellService
│   └── context.go             # EXISTING: Implementation
│
└── mcp/                       # Add service wrapper
    ├── service.go             # NEW: MCPService
    └── manager.go             # EXISTING: Implementation
```

**Key Changes:**
- ❌ Delete `internal/manager` package entirely
- ✅ Move Manager logic into Conversation
- ✅ Add thin Service wrappers for integrations
- ✅ Use builder pattern for Conversation construction

### 3.2 Integration Services Pattern

Following the **tools pattern**, create thin service wrappers:

```go
// internal/git/service.go
package git

type Service struct {
    integration *GitIntegration
}

func NewService(enabled bool, workDir string, logger *slog.Logger) (*Service, error) {
    integration := NewGitIntegration(enabled, workDir, logger)
    if err := integration.Initialize(context.Background()); err != nil {
        return nil, err
    }
    return &Service{integration: integration}, nil
}

func (s *Service) GetContextInfo() GitContextInfo {
    return s.integration.GetContextInfo()
}

func (s *Service) IsRepository() bool {
    return s.integration.IsRepository()
}

func (s *Service) Close() error {
    return s.integration.Close()
}

// internal/shell/service.go
package shell

type Service struct {
    context *Context
}

func NewService(enabled bool, workDir string, logger *slog.Logger, timeout time.Duration) (*Service, error) {
    ctx := NewContext(enabled, workDir, logger, timeout)
    if err := ctx.Initialize(context.Background()); err != nil {
        return nil, err
    }
    return &Service{context: ctx}, nil
}

// internal/mcp/service.go
package mcp

type Service struct {
    manager *MCPManager
}

func NewService(cfg *Config, logger *slog.Logger) (*Service, error) {
    mgr := NewMCPManager(cfg, logger)
    if err := mgr.Initialize(context.Background()); err != nil {
        return nil, err
    }
    return &Service{manager: mgr}, nil
}
```

**Why services?**
- ✅ Consistent with tools pattern (dependency injection)
- ✅ Independent lifecycle (can outlive conversations)
- ✅ Easy to mock for testing
- ✅ Application controls initialization
- ✅ Can be shared across conversations

### 3.3 Conversation Builder Pattern

```go
// internal/conversation/builder.go
package conversation

type Builder struct {
    cfg          *Config
    workDir      string
    gitService   *git.Service
    shellService *shell.Service
    mcpService   *mcp.Service
}

func NewBuilder(cfg *Config, workDir string) *Builder {
    return &Builder{
        cfg:     cfg,
        workDir: workDir,
    }
}

func (b *Builder) WithGit(service *git.Service) *Builder {
    b.gitService = service
    return b
}

func (b *Builder) WithShell(service *shell.Service) *Builder {
    b.shellService = service
    return b
}

func (b *Builder) WithMCP(service *mcp.Service) *Builder {
    b.mcpService = service
    return b
}

func (b *Builder) Build(ctx context.Context) (*Conversation, error) {
    // Build executor (from manager/executor.go)
    exec, err := b.buildExecutor(b.workDir)
    if err != nil {
        return nil, err
    }

    // Gather environment (from manager/environment.go)
    env, err := b.gatherEnvironment(b.workDir)
    if err != nil {
        return nil, err
    }

    // Enrich with services
    if b.gitService != nil {
        b.enrichWithGit(env)
    }
    if b.shellService != nil {
        b.enrichWithShell(env)
    }

    // Build tool registry (from manager/tools.go)
    toolRegistry := b.buildToolRegistry(exec, env)

    // Build agent (from manager/agent.go)
    agent, err := b.buildAgent(exec, env, toolRegistry)
    if err != nil {
        return nil, err
    }

    // Create history (from manager/history.go)
    history := b.createHistory()

    return &Conversation{
        gitService:   b.gitService,
        shellService: b.shellService,
        mcpService:   b.mcpService,
        agent:        agent,
        history:      history,
        emitter:      b.cfg.EventEmitter,
        taskMode:     "regular",
        sessionID:    generateSessionID(),
        workDir:      b.workDir,
    }, nil
}
```

### 3.4 Simplified Conversation

```go
// internal/conversation/conversation.go
package conversation

type Conversation struct {
    // Services (optional, can be nil)
    gitService   *git.Service
    shellService *shell.Service
    mcpService   *mcp.Service

    // Core components
    agent     *agent.Agent
    history   *history.History
    emitter   *events.EventEmitter

    // State
    taskMode  string
    sessionID string
    workDir   string
}

func (c *Conversation) RunTurn(ctx context.Context, input string) error {
    // Same implementation as current
}

func (c *Conversation) Close() error {
    // Close only conversation-specific resources
    // Services are owned by application, not closed here
    return nil
}
```

### 3.5 New Application Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    APPLICATION (TUI/CLI)                     │
└──────────────┬───────────────────────────────┬──────────────┘
               │                               │
        ┌──────▼──────┐                ┌──────▼──────┐
        │  Services   │                │    Config   │
        │  (Optional) │                │             │
        └──────┬──────┘                └──────┬──────┘
               │                               │
        ┌──────▼──────┐                       │
        │ GitService  │                       │
        │ ShellService│                       │
        │ MCPService  │                       │
        └──────┬──────┘                       │
               │                               │
               └───────────┬───────────────────┘
                           │
                  ┌────────▼────────┐
                  │  Conversation   │ ← PRIMARY ENTRY POINT
                  │    .Builder()   │
                  └────────┬────────┘
                           │
                ┌──────────┴──────────┐
                │                     │
         ┌──────▼──────┐      ┌──────▼──────┐
         │   History   │      │    Agent    │
         │  (Storage)  │      │  (Executor) │
         └─────────────┘      └─────────────┘
```

### 3.6 Usage Examples

**Simple usage (no integrations):**
```go
// Create conversation without integrations
conv, err := conversation.NewBuilder(cfg, workDir).Build(ctx)
if err != nil {
    log.Fatal(err)
}
defer conv.Close()

conv.RunTurn(ctx, "Hello")
```

**Full usage (with integrations):**
```go
// Application layer: Create services
var gitSvc *git.Service
var shellSvc *shell.Service
var mcpSvc *mcp.Service

if cfg.EnableGit {
    gitSvc, _ = git.NewService(true, workDir, logger)
    defer gitSvc.Close()
}

if cfg.EnableShell {
    shellSvc, _ = shell.NewService(true, workDir, logger, cfg.ShellTimeout)
    defer shellSvc.Close()
}

if cfg.EnableMCP {
    mcpSvc, _ = mcp.NewService(mcpCfg, logger)
    defer mcpSvc.Close()
}

// Build conversation with services
conv, err := conversation.NewBuilder(cfg, workDir).
    WithGit(gitSvc).
    WithShell(shellSvc).
    WithMCP(mcpSvc).
    Build(ctx)

if err != nil {
    log.Fatal(err)
}
defer conv.Close()

conv.RunTurn(ctx, "Create a file")
```

**Reusing services across conversations:**
```go
// Services are long-lived, shareable
gitSvc, _ := git.NewService(true, workDir, logger)
defer gitSvc.Close()

// Create multiple conversations with same services
conv1, _ := conversation.NewBuilder(cfg, "/project1").
    WithGit(gitSvc).
    Build(ctx)

conv2, _ := conversation.NewBuilder(cfg, "/project2").
    WithGit(gitSvc).
    Build(ctx)

// Both conversations share same Git service
```

---

## 4. Implementation Roadmap

### Phase 1: Add Service Wrappers (Week 1)

[] **Task 1.1: Create git.Service**
  - File: `internal/git/service.go`
  - Wrap existing `GitIntegration`
  - Add tests: `internal/git/service_test.go`

[] **Task 1.2: Create shell.Service**
  - File: `internal/shell/service.go`
  - Wrap existing `Context`
  - Add tests: `internal/shell/service_test.go`

[] **Task 1.3: Create mcp.Service**
  - File: `internal/mcp/service.go`
  - Wrap existing `MCPManager`
  - Add tests: `internal/mcp/service_test.go`

**Deliverables:**
- ✅ 3 new service files
- ✅ 3 new test files with 90%+ coverage
- ✅ All existing tests still pass

### Phase 2: Create Conversation Builder (Week 2)

**Task 2.1: Create conversation package structure**
```
internal/conversation/
├── builder.go          # NEW: Builder pattern
├── builder_test.go     # NEW: Builder tests
├── conversation.go     # EXISTING: Update to use builder
├── executor.go         # FROM: manager/executor.go
├── environment.go      # FROM: manager/environment.go
├── tools.go            # FROM: manager/tools.go
├── agent.go            # FROM: manager/agent.go
├── history.go          # FROM: manager/history.go
├── adapters.go         # FROM: manager/adapters.go
└── events.go           # FROM: manager/events.go
```

**Task 2.2: Move Manager code to Conversation**
- Move all `internal/manager/*.go` → `internal/conversation/*.go`
- Update package declarations
- Make methods private (buildExecutor, buildAgent, etc.)
- Update imports

**Task 2.3: Implement Builder pattern**
- Create `Builder` struct with service fields
- Implement `WithGit()`, `WithShell()`, `WithMCP()` fluent API
- Implement `Build()` method (uses moved Manager code)
- Add comprehensive tests

**Deliverables:**
- ✅ Conversation package with all Manager logic
- ✅ Builder pattern implemented
- ✅ All tests passing

### Phase 3: Update Application Layer & Remove Manager (Week 3)

**Task 3.1: Update cmd/spin/tui.go**
- Create services in application layer
- Use new builder pattern
- Maintain service lifecycle

**Task 3.2: Update cmd/spin/exec.go**
- Same updates as TUI

**Task 3.3: Update all imports**
- Replace `internal/manager` imports with `internal/conversation`
- Add service imports (`internal/git`, `internal/shell`, `internal/mcp`)

**Task 3.4: Update all tests**
- Migrate manager tests to conversation tests
- Add new builder pattern tests
- Ensure 90%+ coverage maintained

**Task 3.5: Delete Manager package**
- Delete `internal/manager/` directory
- Final test sweep

**Deliverables:**
- ✅ Application uses new pattern
- ✅ Manager package removed
- ✅ All tests passing
- ✅ Documentation updated

---

## 5. Migration Strategy

### 5.1 Direct Migration Approach

**Old API (to be removed):**
```go
manager := manager.NewManager(cfg)
conv := manager.NewConversation(ctx, workDir)
conv.RunTurn(ctx, input)
```

**New API (direct replacement):**
```go
// Create services (optional)
gitSvc, _ := git.NewService(cfg.EnableGit, workDir, logger)
defer gitSvc.Close()

// Create conversation
conv, _ := conversation.NewBuilder(cfg, workDir).
    WithGit(gitSvc).
    Build(ctx)
defer conv.Close()

// Use (unchanged)
conv.RunTurn(ctx, input)
```

### 5.2 Update Locations

Files that need to be updated:

1. **Application entry points:**
   - `cmd/spin/tui.go` - TUI initialization
   - `cmd/spin/exec.go` - Exec mode initialization

2. **Tests:**
   - All tests in `internal/manager/*_test.go` → migrate to `internal/conversation/*_test.go`
   - Integration tests that create Manager

3. **Imports:**
   - Replace `"github.com/dmytrogajewski/spin/internal/manager"`
   - With `"github.com/dmytrogajewski/spin/internal/conversation"`
   - Add service imports as needed

---

## 6. Testing Strategy

### 6.1 Service Tests

```go
// internal/git/service_test.go
func TestGitService_New(t *testing.T) {
    tests := []struct {
        name    string
        enabled bool
        workDir string
        wantErr bool
    }{
        {"enabled_valid_repo", true, "/valid/repo", false},
        {"enabled_invalid_repo", true, "/invalid", false}, // Not error - just not a repo
        {"disabled", false, "/any", false},
    }
    // ...
}

func TestGitService_Integration(t *testing.T) {
    // Test that service wraps integration correctly
}
```

### 6.2 Builder Tests

```go
// internal/conversation/builder_test.go
func TestBuilder_WithServices(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(b *Builder)
        validate func(t *testing.T, conv *Conversation)
    }{
        {
            name: "with_git_only",
            setup: func(b *Builder) {
                gitSvc, _ := git.NewService(true, "/tmp", logger)
                b.WithGit(gitSvc)
            },
            validate: func(t *testing.T, conv *Conversation) {
                assert.NotNil(t, conv.gitService)
                assert.Nil(t, conv.shellService)
                assert.Nil(t, conv.mcpService)
            },
        },
        // More test cases...
    }
}

func TestBuilder_Build_Integration(t *testing.T) {
    // Test full build with all services
}
```

### 6.3 Coverage Targets

| Package | Target | Notes |
|---------|--------|-------|
| `git/service.go` | 90% | Service wrapper |
| `shell/service.go` | 90% | Service wrapper |
| `mcp/service.go` | 90% | Service wrapper |
| `conversation/builder.go` | 95% | Critical path |
| `conversation/*` (moved from manager) | 85% | Maintain existing coverage |

---

## 7. Rollback Plan

### 7.1 Rollback Triggers

Rollback if:
- ❌ Test coverage drops below 80%
- ❌ Critical bug found in production
- ❌ Performance regression > 20%
- ❌ Migration causes data loss

### 7.2 Rollback Procedure

**Phase 1 rollback (Services added):**
- Simply don't use new services
- Old Manager code unchanged
- Zero risk

**Phase 2 rollback (Builder created):**
- Don't migrate application code yet
- Manager package still exists
- Low risk

**Phase 3 rollback (Manager deleted):**
1. Restore `internal/manager/` from git
2. Revert application changes (tui.go, exec.go)
3. Revert import changes
4. Remove service instantiation code

### 7.3 Risk Mitigation

- Commit after each phase completes successfully
- Run full test suite after each phase
- Keep git history clean for easy rollback
- Test both TUI and exec modes before final commit

---

## 8. Benefits Analysis

### 8.1 Code Quality Improvements

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| **API Complexity** | 3 steps | 2 steps | -33% |
| **Package Count** | 4 (manager, conversation, agent, integrations) | 3 (conversation, agent, integrations) | -25% |
| **Lines of Code** | manager(~500) + conversation(160) | conversation(~600) | -10% |
| **Cyclomatic Complexity** | Manager.buildAgent: 15 | Builder.buildAgent: 15 | 0% (same logic) |
| **Test Coverage** | Manager: 82.4% | Conversation: 90% (target) | +9% |

### 8.2 Developer Experience

**Before:**
```go
// What is Manager for?
// Why do I need it?
// Can I reuse it?
manager := manager.NewManager(cfg)
conv := manager.NewConversation(ctx, workDir)
conv.RunTurn(ctx, input)
```

**After:**
```go
// Clear: Conversation is what I need
// Services are optional, explicit
conv := conversation.NewBuilder(cfg, workDir).Build(ctx)
conv.RunTurn(ctx, input)
```

### 8.3 Consistency with Codebase

**Tools Pattern (existing):**
```go
tool := tools.NewGitOperationTool(gitIntegration)
                                  ^^^^^^^^^^^^^^
                                  Dependency injected
```

**New Service Pattern:**
```go
conv := conversation.NewBuilder(cfg, workDir).
           WithGit(gitService)
                   ^^^^^^^^^^
                   Dependency injected (same pattern!)
```

✅ **Consistent dependency injection throughout codebase**

---

## 9. Open Questions & Decisions

### Q1: Should services auto-initialize?

**Option A:** Services require manual initialization
```go
svc := git.NewService(enabled, workDir, logger)
svc.Initialize(ctx) // Explicit
```

**Option B:** Services initialize in constructor
```go
svc := git.NewService(enabled, workDir, logger) // Auto-initializes
```

**Decision:** **Option B** - Initialize in constructor
- Simpler API
- Consistent with current Manager behavior
- Error handling at construction time

### Q2: Should Conversation.Close() close services?

**Option A:** Conversation owns services, closes them
```go
conv.Close() // Closes services
```

**Option B:** Application owns services, closes separately
```go
defer gitSvc.Close()
defer conv.Close() // Doesn't close services
```

**Decision:** **Option B** - Application owns services
- Services can be shared across conversations
- Clear ownership model
- More flexible

### Q3: Should we keep Config in conversation package?

**Option A:** Keep Config in manager package (legacy)
```go
import "github.com/dmytrogajewski/spin/internal/manager"
cfg := manager.DefaultConfig()
```

**Option B:** Move Config to conversation package
```go
import "github.com/dmytrogajewski/spin/internal/conversation"
cfg := conversation.DefaultConfig()
```

**Option C:** Move Config to root config package
```go
import "github.com/dmytrogajewski/spin/internal/config"
cfg := config.Default()
```

**Decision:** **Option C** - Move to `internal/config`
- Config is application-wide, not manager-specific
- More discoverable
- Follows Go project layout standards

---

## 10. Success Criteria

### 10.1 Functional Requirements

- ✅ All existing functionality preserved
- ✅ Services can be shared across conversations
- ✅ All tests passing
- ✅ No performance regression

### 10.2 Non-Functional Requirements

- ✅ Test coverage ≥ 90% for new code
- ✅ Test coverage ≥ 85% for moved code
- ✅ API surface reduced by 25%
- ✅ Build time unchanged
- ✅ Memory footprint unchanged

### 10.3 Documentation Requirements

- ✅ API documentation updated
- ✅ Architecture diagrams updated
- ✅ Examples updated
- ✅ CHANGELOG entry

---

## 11. Timeline

| Phase | Duration | Deliverables | Risk |
|-------|----------|--------------|------|
| **Phase 1: Service Wrappers** | Week 1 | 3 service files, tests | Low |
| **Phase 2: Conversation Builder** | Week 2 | Builder + moved code | Medium |
| **Phase 3: Update & Cleanup** | Week 3 | Application updated, Manager deleted | Medium |
| **Total** | **3 weeks** | | |

**Milestones:**
- Week 1 end: Services ready, tests passing
- Week 2 end: Builder working with all Manager functionality
- Week 3 end: Manager removed, application updated, all tests passing

---

## 12. Conclusion

This refactoring eliminates an unnecessary layer (Manager) while improving code organization, testability, and consistency with existing patterns. The service-based approach aligns with the tools pattern already established in the codebase, making the architecture more coherent and easier to understand.

**Key Takeaways:**
1. Manager package is a factory with no independent value → Remove it
2. Integrations should be services, not owned by Manager → Extract them
3. Conversation should be the primary entry point → Make it so
4. Follow existing tools pattern for consistency → Service injection

**Next Steps:**
1. Review and approve this plan
2. Create tracking issue with subtasks
3. Begin Phase 1 implementation
4. Weekly progress reviews

---

## Appendix A: File Movement Map

| Current File | New Location | Notes |
|-------------|--------------|-------|
| `manager/manager.go` | `conversation/conversation.go` | Merge with existing |
| `manager/executor.go` | `conversation/executor.go` | Move as-is |
| `manager/environment.go` | `conversation/environment.go` | Move as-is |
| `manager/tools.go` | `conversation/tools.go` | Move as-is |
| `manager/agent.go` | `conversation/agent.go` | Move as-is |
| `manager/history.go` | `conversation/history.go` | Move as-is |
| `manager/adapters.go` | `conversation/adapters.go` | Move as-is |
| `manager/events.go` | `conversation/events.go` | Move as-is |
| `manager/builder.go` | DELETE | Replaced by conversation builder |
| `manager/config.go` | `config/config.go` | Move to root config |
| NEW | `git/service.go` | New wrapper |
| NEW | `shell/service.go` | New wrapper |
| NEW | `mcp/service.go` | New wrapper |
| NEW | `conversation/builder.go` | New builder |

---

## Appendix B: API Comparison

### Before (Current)

```go
// Import
import "github.com/dmytrogajewski/spin/internal/manager"

// Create
cfg := manager.DefaultConfig()
mgr, err := manager.NewManager(cfg)
conv, err := mgr.NewConversation(ctx, workDir)

// Use
err = conv.RunTurn(ctx, input)

// Cleanup
mgr.Close()
```

### After (Proposed)

```go
// Imports
import "github.com/dmytrogajewski/spin/internal/conversation"
import "github.com/dmytrogajewski/spin/internal/git"
import "github.com/dmytrogajewski/spin/internal/shell"

// Create services (optional)
gitSvc, _ := git.NewService(true, workDir, logger)
defer gitSvc.Close()

shellSvc, _ := shell.NewService(true, workDir, logger, timeout)
defer shellSvc.Close()

// Create conversation
cfg := conversation.DefaultConfig()
conv, err := conversation.NewBuilder(cfg, workDir).
    WithGit(gitSvc).
    WithShell(shellSvc).
    Build(ctx)
defer conv.Close()

// Use (unchanged)
err = conv.RunTurn(ctx, input)
```

---

**Document Version:** 1.0
**Last Updated:** 2025-11-02
**Author:** Architecture Team
**Status:** Proposed
