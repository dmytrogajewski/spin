# FRD-20251019-002: Core Package Decomposition

## Metadata
- **Status**: DRAFT
- **Priority**: P0 (CRITICAL)
- **Effort**: XL (1-2 weeks)
- **Dependencies**: FRD-20251019-001 (Agent Service Extraction - COMPLETED)
- **Related**: [Architectural Anti-Patterns](../../docs/architectural-anti-patterns.md)

## Problem Statement

The `internal/core/` package has become a monolithic "god package" containing 124 files with mixed responsibilities. This violates domain-driven design principles and makes the codebase difficult to navigate and maintain.

**Current Structure:**
```
internal/core/
├── agent.go (and 15+ agent_*.go files)
├── conversation.go (and 10+ conversation_*.go files)
├── manager.go
├── security_service.go
├── detection_service.go
├── orchestration_service.go
├── validator.go (and validator_*.go)
├── executor.go
├── approval.go
├── tool_executor.go
├── tool_protocol.go
├── event.go (and event_*.go)
├── environment.go
├── history.go
├── config.go
├── mcp_manager.go
├── git_integration.go
├── shell_integration.go
└── ... 100+ more files
```

**Issues:**
1. **Package has no clear domain** - "core" is a catch-all name
2. **Import cycles waiting to happen** - everything imports "core"
3. **Difficult to navigate** - 124 files in one directory
4. **Poor cohesion** - unrelated types in same package
5. **Testing complexity** - mixed test concerns
6. **Violates single responsibility** at package level

## Goals

1. **Decompose core into domain packages** with clear responsibilities
2. **Follow 1 package = 1 file + 1 test pattern** where feasible
3. **Eliminate import cycles** using interfaces
4. **Improve discoverability** - clear package names
5. **Maintain backward compatibility** for external users (cmd/spin, etc.)
6. **All tests must pass** throughout refactoring

## Non-Goals

1. **NOT changing public APIs** from cmd/spin perspective
2. **NOT adding new features** - pure refactoring
3. **NOT optimizing performance** - maintain current performance

## Design

### Proposed Package Structure

```
internal/
├── agent/           # Agent orchestration (agent.go + agent_test.go)
├── conversation/    # Conversation management
├── security/        # SecurityService, Validator, Command
│   ├── security.go
│   ├── security_test.go
│   ├── validator.go
│   ├── validator_test.go
│   └── types.go
├── detection/       # DetectionService, cycle detection
├── orchestration/   # OrchestrationService, tool execution
├── events/          # Event emitter and types
├── environment/     # Environment context
├── executor/        # Command executor
├── approval/        # Approval service and types
├── history/         # Already good - keep as is
├── manager/         # Manager (conversation factory)
├── config/          # Configuration (already exists - may merge)
├── integration/     # Git and Shell integrations
│   ├── git/
│   └── shell/
└── mcp/            # MCP manager (already exists - may merge)

pkg/
└── spin/           # Public API (if needed for external consumers)
```

### Package Responsibilities

#### internal/agent
- **Files**: agent.go, agent_test.go
- **Exports**: Agent, AgentRequest, AgentResponse, AgentOption
- **Dependencies**: security, detection, orchestration, environment, events

#### internal/security
- **Files**: security.go, security_test.go, validator.go, validator_test.go, types.go
- **Exports**: SecurityService, Validator, Command, ValidationResult, CommandClass
- **Dependencies**: approval, events

#### internal/detection
- **Files**: detection.go, detection_test.go
- **Exports**: DetectionService
- **Dependencies**: internal/core/cycle (keep existing)

#### internal/orchestration
- **Files**: orchestration.go, orchestration_test.go, tool_executor.go, tool_executor_test.go
- **Exports**: OrchestrationService, ToolExecutor
- **Dependencies**: tools, security, approval

#### internal/conversation
- **Files**: conversation.go, conversation_test.go
- **Exports**: Conversation
- **Dependencies**: agent, history, events

#### internal/manager
- **Files**: manager.go, manager_test.go
- **Exports**: Manager, ManagerOption
- **Dependencies**: agent, conversation, security, detection, orchestration

### Handling Import Cycles

**Principle**: Define interfaces at boundaries

```go
// internal/security/validator.go
type Validator interface {
    Validate(cmd Command) (*ValidationResult, error)
}

// internal/orchestration/executor.go  
type CommandValidator interface {
    Validate(cmd Command) (*ValidationResult, error)
}

// Break cycle: orchestration imports security.Validator interface, not concrete type
```

## Implementation Plan

### Phase A: Extract Services (Day 1-2)

1. Create new packages:
   - `internal/security/`
   - `internal/detection/`
   - `internal/orchestration/`

2. Move service files:
   - `core/security_service.go` → `security/security.go`
   - `core/detection_service.go` → `detection/detection.go`
   - `core/orchestration_service.go` → `orchestration/orchestration.go`

3. Move related types:
   - Validator → `security/validator.go`
   - ToolExecutor → `orchestration/tool_executor.go`

4. Update imports throughout codebase

5. Run tests after each move

### Phase B: Extract Infrastructure (Day 3-4)

1. Create packages:
   - `internal/events/`
   - `internal/environment/`
   - `internal/executor/`
   - `internal/approval/`

2. Move files accordingly

3. Update imports

4. Run tests

### Phase C: Extract Agent & Conversation (Day 5-6)

1. Create packages:
   - `internal/agent/`
   - `internal/conversation/`
   - `internal/manager/`

2. Move agent.go and related files

3. Move conversation.go

4. Move manager.go

5. Update imports

6. Run tests

### Phase D: Extract Integrations (Day 7-8)

1. Create `internal/integration/git/`
2. Create `internal/integration/shell/`
3. Move integration code
4. Update imports
5. Run tests

### Phase E: Cleanup & Verification (Day 9-10)

1. Remove empty `internal/core/` directory
2. Create compatibility shim in `internal/core/` if needed (deprecated)
3. Update all documentation
4. Run full test suite
5. Run benchmarks
6. Verify no performance regression

## Migration Strategy

### For External Consumers (cmd/spin)

**Option 1: Compatibility Package (Recommended)**
```go
// internal/core/core.go - deprecated compatibility shim
package core

import (
    "github.com/dmytrogajewski/spin/internal/agent"
    "github.com/dmytrogajewski/spin/internal/security"
    // ... etc
)

// Deprecated: Use internal/agent.Agent
type Agent = agent.Agent

// Deprecated: Use internal/security.SecurityService
type SecurityService = security.Service

// ... re-export all types
```

**Option 2: Direct Migration**
```go
// cmd/spin/tui.go
-import "github.com/dmytrogajewski/spin/internal/core"
+import (
+    "github.com/dmytrogajewski/spin/internal/agent"
+    "github.com/dmytrogajewski/spin/internal/manager"
+)

-mgr, _ := core.NewManager(cfg)
+mgr, _ := manager.New(cfg)
```

Since user said "no backward compatibility", we'll use **Option 2**.

## Testing Strategy

### Per-Package Tests

Each package has ONE test file:
```
internal/security/
├── security.go
├── security_test.go       # All SecurityService tests
├── validator.go
├── validator_test.go      # All Validator tests  
└── types.go              # Shared types, no separate test
```

### Test Organization

```go
// security_test.go
func TestNewSecurityService(t *testing.T) { ... }
func TestSecurityService_ValidateCommand(t *testing.T) { ... }
func TestSecurityService_RequestApproval(t *testing.T) { ... }
// ... all SecurityService tests in one file
```

### Migration Plan for Tests

1. Move tests with their source files
2. Consolidate multiple test files into one per package
3. Remove duplicate test helpers
4. Update test imports

## API Changes

### Breaking Changes (Internal Only)

**Import paths change:**
```go
// Before
import "github.com/dmytrogajewski/spin/internal/core"

// After
import (
    "github.com/dmytrogajewski/spin/internal/agent"
    "github.com/dmytrogajewski/spin/internal/security"
    "github.com/dmytrogajewski/spin/internal/orchestration"
)
```

**Type references change:**
```go
// Before
agent, _ := core.NewAgent(...)
sec := &core.SecurityService{}

// After
agent, _ := agent.New(...)
sec := &security.Service{}
```

### Non-Breaking (External API Preserved)

- `cmd/spin/` imports updated but functionality unchanged
- Public behavior identical
- Test assertions remain the same

## Acceptance Criteria

- ✅ No `internal/core/` package exists (or only deprecated shim)
- ✅ Each domain has its own package
- ✅ Maximum 5-10 files per package (ideally 2-4)
- ✅ No import cycles
- ✅ All tests pass
- ✅ Coverage ≥85% maintained
- ✅ `make lint` passes
- ✅ `make deadcode` passes
- ✅ Documentation updated for new package structure

## Risks

### High Risk
**Risk**: Breaking existing functionality during large-scale move
- **Mitigation**: Move and test incrementally, one package at a time
- **Mitigation**: Automated tests run after each move

**Risk**: Import cycle creation
- **Mitigation**: Use interfaces at package boundaries
- **Mitigation**: Dependency graph analysis before each move

### Medium Risk
**Risk**: Test suite organization becomes complex
- **Mitigation**: Consolidate tests during move
- **Mitigation**: Clear test file naming convention

## Alternatives Considered

### Alternative 1: Keep core, just organize subdirectories
```
internal/core/
├── agent/
├── security/
└── ...
```
**Rejected**: Still promotes "core" as catch-all, doesn't solve root issue

### Alternative 2: Feature-based packages
```
internal/
├── chat/
├── execution/
├── safety/
```
**Rejected**: Less clear than domain-based approach

### Alternative 3: Gradual migration with both structures
**Rejected**: User explicitly wants no backward compatibility

## Definition of Done

- [ ] FRD created and reviewed ← **WE ARE HERE**
- [ ] All packages created with clear responsibilities
- [ ] All code moved from `internal/core/`
- [ ] Import cycles resolved via interfaces
- [ ] One .go file + one _test.go per package (where feasible)
- [ ] All tests pass
- [ ] `go test -race ./...` passes
- [ ] `make lint` passes
- [ ] `make deadcode` passes
- [ ] Coverage ≥85% maintained
- [ ] Documentation updated for new structure
- [ ] `cmd/spin/` updated with new imports
- [ ] `internal/core/` directory removed or contains only deprecated shim

## Related Work

**Builds on:**
- FRD-20251019-001: Agent Service Extraction (completed)

**Enables:**
- Cleaner package structure
- Better separation of concerns
- Easier navigation and maintenance

---

**Created**: 2025-10-19  
**Author**: Spin Agent  
**Version**: 1.0  
**Status**: DRAFT → Ready for implementation

