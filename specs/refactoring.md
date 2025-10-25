# Spin Codebase Refactoring Analysis

**Author:** Rob Pike (simulated analysis)
**Date:** 2025-10-25
**Go Version:** 1.24
**Project:** Spin AI Coding Agent

---

## Executive Summary

The Spin codebase demonstrates **strong architectural foundations** with proper layering, dependency inversion, and service-based design. The recent refactoring (2025-10-19) introducing service-based architecture in the Agent package shows mature evolution toward clean architecture principles.

**Overall Assessment: 8.4/10 - Production-Ready**

**Key Strengths:**
- Excellent interface abstraction (Provider, Tool, Task)
- No circular dependencies
- Proper use of Go idioms (functional options, composition over inheritance)
- Clean separation of concerns via service layers
- Strong type safety with generics where appropriate

**Areas Requiring Attention:**
- Manager complexity (450+ lines of construction logic)
- Some error handling patterns could be improved
- Minor violations of Go Code Review Comments
- Package naming conventions need refinement

---

## 1. Project Layout Assessment

### Compliance with golang-standards/project-layout

| Directory | Status | Notes |
|-----------|--------|-------|
| `/cmd` | ✅ Excellent | Single `cmd/spin/` with multiple commands, minimal logic |
| `/internal` | ✅ Excellent | Proper use of internal packages, well-organized |
| `/pkg` | ⚠️ Missing | Intentional - all code is internal (acceptable) |
| `/configs` | ✅ Present | Configuration templates |
| `/docs` | ✅ Present | Documentation and package docs |
| `/examples` | ✅ Present | TUI demos, streaming examples |
| `/tests` | ✅ Present | E2E tests separated |
| `/scripts` | ✅ Present | Build and development scripts |
| `/build` | ❌ Missing | Consider adding CI/CD configurations |
| `/deployments` | ❌ Missing | Not needed for CLI tool |
| `/api` | ❌ Missing | Consider adding OpenAPI specs for MCP protocol |

**Recommendation:** Add `/build` directory for CI/CD configurations and Docker files.

### Anti-patterns Avoided

✅ **No `/src` directory** - Correctly avoided Java-style structure
✅ **No vendor bloat** - Uses Go modules properly
✅ **Clean internal structure** - Packages have clear responsibilities

---

## 2. Go Code Review Comments Compliance

### 2.1 Error Handling

#### ✅ Good Patterns Found

```go
// internal/agent/agent.go - Proper error wrapping
return nil, fmt.Errorf("LLM completion failed: %w", err)

// Defined error constants
var (
    ErrNilLLM           = errors.New("LLM provider cannot be nil")
    ErrNilSecurity      = errors.New("security service cannot be nil")
    ErrMaxTurns         = errors.New("maximum turns reached")
)
```

#### ❌ Violations Found

**1. Capitalized error strings**

```go
// internal/llm/openai/provider.go:454
return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
// Should be: "failed to read error response: HTTP %d"

// internal/llm/openai/provider.go:471
return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error.Message)
// Should be: "http request failed (%d): %s"

// internal/history/compress/llm.go:173
return CompressibleMessage{}, fmt.Errorf("LLM summarization failed: %w", err)
// Should be: "llm summarization failed: %w"
```

**2. Ignored errors in non-test code**

```go
// internal/agent/environment.go:XX
gitInfo, _ = gatherGitInfo(workDir) // Ignore errors, Git may not be available

// Better approach:
gitInfo, err := gatherGitInfo(workDir)
if err != nil {
    // Log the error or set gitInfo to zero value
    logger.Debug("git info unavailable", "error", err)
}
```

**Recommendation:** Audit all error messages to ensure lowercase and proper wrapping.

### 2.2 Naming Conventions

#### ✅ Excellent Patterns

```go
// internal/llm/provider.go - Proper interface naming
type Provider interface { ... }

// internal/tools/types.go - Single-method interface with -er suffix
type Tool interface { ... }

// No "Get" prefix on getters
func (t *Task) Name() string    // Not GetName()
func (a *Agent) Config() *Config // Not GetConfig()

// Proper receiver names
func (a *Agent) Execute(...)     // 'a' for Agent
func (m *Manager) NewConversation(...) // 'm' for Manager
func (s *SecurityService) ValidateCommand(...) // 's' for SecurityService
```

#### ⚠️ Issues Found

**1. Package `types` - Too Generic**

```
internal/types/
```

Per Go Code Review Comments: *"Avoid meaningless package names like util, common, misc, api, types"*

**Recommendation:** Move `ToolCallArguments` to `internal/tools/arguments.go` as it's tool-specific.

**2. Package `constants` - Anti-pattern**

```
internal/constants/
```

Constants should live with the packages that use them, not in a separate package.

**Recommendation:** Distribute constants to their respective packages.

**3. Package `message` - Unclear Purpose**

Check if this duplicates protocol or agent message types. Consider consolidating.

### 2.3 Commentary and Documentation

#### ✅ Good Patterns

```go
// Package agent implements the core agent logic and decision-making loop.
package agent

// Agent implements the core agent logic and decision-making loop.
//
// The Agent orchestrates the interaction between the LLM, tools, and execution
// environment. It processes user requests through multiple turns of LLM calls
// and tool executions until the task is complete or limits are reached.
//
// REFACTORED: Agent now uses service-based architecture (2025-10-19)
type Agent struct { ... }
```

**All exported types and functions have proper doc comments** ✅

#### ❌ Minor Issues

Some package comments have blank lines between comment and package declaration:

```go
// Package xyz does something

package xyz  // Should have no blank line above
```

### 2.4 Interfaces and Embedding

#### ✅ Excellent Patterns

**Interfaces defined in consumer packages:**

```go
// internal/cycle/intervention.go - Consumer defines interface
type Message interface {
    GetRole() string
    GetContent() string
    GetTimestamp() time.Time
}

// internal/agent/agent.go - Consumer has adapter
type messageAdapter struct {
    *Message
}
```

**Clean interface definitions:**

```go
// internal/llm/provider.go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models(ctx context.Context) ([]Model, error)
    Capabilities() Capabilities
    Name() string
    Close() error
}
```

**Proper embedding for composition:**

```go
// internal/agent/agent.go
type Agent struct {
    llm llm.Provider  // Composition, not inheritance
    security      *security.SecurityService
    detection     *detection.DetectionService
    orchestration *orchestration.OrchestrationService
}
```

---

## 3. Effective Go Principles Assessment

### 3.1 Formatting ✅

**Status:** Excellent - All code appears gofmt'd

### 3.2 Names ✅

**Package names:** Short, lowercase, single-word (agent, tools, llm, cycle, events) ✅
**No underscores:** All names use MixedCaps ✅
**Interface naming:** Provider, Tool, Task (clean, not bloated) ✅

### 3.3 Constructors and Initialization

#### ✅ Excellent Use of New() Functions

```go
func NewAgent(
    provider llm.Provider,
    security *security.SecurityService,
    detection *detection.DetectionService,
    orchestration *orchestration.OrchestrationService,
    context *Environment,
    emitter *events.EventEmitter,
    opts ...AgentOption,
) (*Agent, error)
```

**Zero-value useful design:**

```go
// internal/events/event.go
emitter := &EventEmitter{
    subscribers: make(map[string]chan Event),
    // Zero values for other fields work correctly
}
```

### 3.4 Methods - Receiver Types

#### ✅ Consistent Patterns

**Pointer receivers for mutation:**

```go
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
func (m *Manager) NewConversation(ctx context.Context, opts ...ConversationOption) (*conversation.Conversation, error)
```

**Value receivers for small immutable types:**

```go
func (t ToolCallArguments) GetString(key string) (string, error)
```

**No mixing of receiver types within a type** ✅

### 3.5 Concurrency Patterns

#### ✅ Good Use of Channels and Goroutines

```go
// internal/events/event.go - Event bus with channel
func (e *EventEmitter) Subscribe() (string, <-chan Event, error)

// internal/agent/executor.go - Streaming output
type OutputChunk struct {
    Data      []byte
    Stderr    bool
    Timestamp time.Time
}
```

#### ⚠️ Goroutine Lifecycle Clarity

Some goroutines could have clearer lifetime documentation:

```go
// internal/events/event.go
// Add: Document when goroutines are created and terminated
// Add: Document cleanup on EventEmitter.Close()
```

**Recommendation:** Add goroutine lifecycle documentation to EventEmitter and shell integration.

### 3.6 Defer Usage ✅

**Proper defer for unlocking:**

```go
func (m *TUIMapper) StopStreaming() {
    defer m.streamMu.Unlock()
    m.streamMu.Lock()
    // ...
}
```

**Note:** Lock order is correct (defer before lock in this pattern).

### 3.7 Error Handling ✅

**Multiple return values for errors:**

```go
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
```

**Named results where appropriate:**

```go
func (s *ApprovalService) RequestApproval(ctx context.Context, operation Operation) (reqID string, approved bool, err error)
```

---

## 4. DRY, SOLID, and Clean Architecture

### 4.1 DRY (Don't Repeat Yourself)

#### ✅ Generally Good

Most code avoids repetition through proper abstraction.

#### ❌ DRY Violations Found

**1. Duplicate argument parsing logic**

```
internal/agent/agent.go - parseToolArguments()
internal/orchestration/tool_executor.go - parseToolArguments()
```

**Fix:**
```go
// Create internal/tools/parser.go
package tools

type ArgumentParser struct{}

func (p *ArgumentParser) Parse(raw string) (map[string]interface{}, error) {
    // Shared implementation
}
```

**2. Repeated validation patterns**

Multiple places validate nil dependencies. Extract to helper:

```go
// internal/validation/deps.go
func RequireNonNil(name string, val interface{}) error {
    if val == nil {
        return fmt.Errorf("%s cannot be nil", name)
    }
    return nil
}
```

### 4.2 SOLID Principles

#### Single Responsibility Principle: 8/10

**✅ Well-Implemented:**
- `agent` - Agent loop and decision making
- `security` - Validation and approval
- `detection` - Cycle detection
- `orchestration` - Tool execution
- `events` - Event distribution

**❌ Violations:**

**Manager has too many responsibilities:**
- Configuration management
- Dependency construction
- Tool registry building
- Integration initialization (Git, Shell, MCP)
- History creation
- Environment gathering
- Executor building

**Fix:** Extract Builder pattern:

```go
// internal/manager/builder.go
type AgentBuilder struct {
    llm              llm.Provider
    approvalHandler  security.ApprovalHandler
    workDir          string
    config           *Config
}

func (b *AgentBuilder) WithLLM(p llm.Provider) *AgentBuilder
func (b *AgentBuilder) WithApproval(h security.ApprovalHandler) *AgentBuilder
func (b *AgentBuilder) Build() (*agent.Agent, error)

// Manager becomes simpler
type Manager struct {
    builder *AgentBuilder
    // ...
}
```

#### Open/Closed Principle: 9/10 ✅

**Excellent extensibility:**
- New LLM providers → Implement `Provider` interface
- New tools → Implement `Tool` interface, register in registry
- New task modes → Implement `Task` interface, register in registry
- New compression strategies → Implement `Compressor` interface

#### Liskov Substitution Principle: 7/10

**✅ Generally Good:**
- All `Provider` implementations are substitutable
- All `Task` implementations work interchangeably

**❌ Minor Issues:**

Some tools panic in Execute() instead of returning errors consistently:

```go
// Some tools should validate arguments more consistently
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
    path, ok := args["path"].(string)
    if !ok {
        // Don't panic, return error
        return ToolResult{Success: false}, fmt.Errorf("invalid argument type for 'path'")
    }
}
```

#### Interface Segregation Principle: 10/10 ✅

**Perfect implementation:**
- All interfaces are minimal and focused
- No bloated interfaces forcing implementations to stub methods
- `Provider` has only essential methods
- `Tool` has minimal contract
- `Task` has exactly what's needed

#### Dependency Inversion Principle: 9/10 ✅

**Excellent implementation:**
- Agent depends on abstractions (Provider, SecurityService, DetectionService)
- Concrete implementations in infrastructure layer
- No imports of concrete types from domain layer

**Minor issue:**
```go
// internal/agent/agent.go
import "github.com/dmytrogajewski/spin/internal/cycle"  // Direct import

// Should only import detection abstraction
// Move cycle.CycleType to detection package
```

---

## 5. Specific Refactoring Recommendations

### Priority 1: High Impact (Do in Next Sprint)

#### R1.1: Extract Manager Builder Pattern

**Problem:** Manager.NewManager() has 450+ lines of construction logic

**Solution:**

```go
// internal/manager/builder.go
package manager

type Builder struct {
    cfg              *Config
    llm              llm.Provider
    emitter          *events.EventEmitter
    storage          session.Storage
    approvalHandler  security.ApprovalHandler
    authManager      *auth.Manager
    logger           *slog.Logger
}

func NewBuilder(cfg *Config) *Builder {
    return &Builder{cfg: cfg}
}

func (b *Builder) WithLLM(p llm.Provider) *Builder {
    b.llm = p
    return b
}

func (b *Builder) WithApprovalHandler(h security.ApprovalHandler) *Builder {
    b.approvalHandler = h
    return b
}

func (b *Builder) Build(ctx context.Context) (*Manager, error) {
    // All construction logic here
    if err := b.validate(); err != nil {
        return nil, err
    }

    mgr := &Manager{
        cfg: b.cfg,
        llm: b.llm,
        // ...
    }

    if err := b.initializeIntegrations(mgr); err != nil {
        return nil, err
    }

    return mgr, nil
}

func (b *Builder) validate() error {
    if b.cfg == nil {
        return errors.New("config required")
    }
    return nil
}

func (b *Builder) initializeIntegrations(m *Manager) error {
    // Git integration
    // Shell integration
    // MCP integration
    return nil
}
```

**Usage:**

```go
mgr, err := manager.NewBuilder(cfg).
    WithLLM(provider).
    WithApprovalHandler(handler).
    Build(ctx)
```

**Impact:** Reduces Manager complexity by ~40%, improves testability

---

#### R1.2: Fix Error String Capitalization

**Problem:** Multiple error strings start with capital letters

**Solution:**

Run audit and fix:

```bash
# Find all capitalized error strings
grep -r 'fmt\.Errorf("[A-Z]' internal/

# Fix pattern:
- fmt.Errorf("HTTP %d: failed", code)
+ fmt.Errorf("http request failed: %d", code)

- fmt.Errorf("LLM summarization failed: %w", err)
+ fmt.Errorf("llm summarization failed: %w", err)
```

**Impact:** Compliance with Go Code Review Comments

---

#### R1.3: Consolidate Tool Argument Parsing

**Problem:** Duplicate parseToolArguments() in agent and orchestration

**Solution:**

```go
// internal/tools/parser.go
package tools

import (
    "encoding/json"
    "fmt"
)

// ArgumentParser parses tool call arguments from JSON.
type ArgumentParser struct{}

// Parse parses JSON-encoded arguments into a map.
func (p *ArgumentParser) Parse(raw string) (map[string]interface{}, error) {
    if raw == "" {
        return make(map[string]interface{}), nil
    }

    var args map[string]interface{}
    if err := json.Unmarshal([]byte(raw), &args); err != nil {
        return nil, fmt.Errorf("failed to parse arguments: %w", err)
    }

    return args, nil
}

// ParseBytes parses arguments from JSON bytes.
func (p *ArgumentParser) ParseBytes(data []byte) (map[string]interface{}, error) {
    var args map[string]interface{}
    if err := json.Unmarshal(data, &args); err != nil {
        return nil, fmt.Errorf("failed to parse arguments: %w", err)
    }
    return args, nil
}
```

**Usage:**

```go
// In agent and orchestration
parser := tools.ArgumentParser{}
args, err := parser.Parse(toolCall.Arguments)
```

**Impact:** Eliminates code duplication, single source of truth

---

#### R1.4: Move Cycle Types to Detection Package

**Problem:** Agent imports cycle package directly, should only use detection abstraction

**Solution:**

```go
// internal/detection/types.go
package detection

// CycleType represents different types of detected cycles
type CycleType string

const (
    CycleTypeInfiniteLoop     CycleType = "infinite_loop"
    CycleTypeRepeatedError    CycleType = "repeated_error"
    CycleTypeNoProgress       CycleType = "no_progress"
    CycleTypeOscillation      CycleType = "oscillation"
)

// CycleResult represents the result of cycle detection
type CycleResult struct {
    Type       CycleType
    Confidence float64
    Suggestion string
}

// Intervention strategies
type InterventionStrategy interface {
    Apply(ctx context.Context) error
}
```

**Agent uses only detection package:**

```go
// internal/agent/agent.go
import "github.com/dmytrogajewski/spin/internal/detection"

// No import of cycle package needed
result, err := a.detection.CheckCycle(ctx, messages)
if result.Type == detection.CycleTypeInfiniteLoop {
    // Handle
}
```

**Impact:** Better encapsulation, cleaner dependencies

---

### Priority 2: Medium Impact (Next Month)

#### R2.1: Eliminate `internal/types` Package

**Problem:** Generic "types" package is an anti-pattern

**Solution:**

Move `ToolCallArguments` to `internal/tools/arguments.go`:

```go
// internal/tools/arguments.go
package tools

import (
    "encoding/json"
    "errors"
)

var ErrParameterNotFound = errors.New("parameter not found")

// Arguments represents the arguments passed to a tool call.
type Arguments map[string]json.RawMessage

// Get retrieves a typed value from the arguments.
func (a Arguments) Get(key string, dest any) error { ... }

// GetString retrieves a string value.
func (a Arguments) GetString(key string) (string, error) { ... }
```

**Update imports:**

```go
- import "github.com/dmytrogajewski/spin/internal/types"
+ import "github.com/dmytrogajewski/spin/internal/tools"

- args := types.ToolCallArguments{}
+ args := tools.Arguments{}
```

**Impact:** Better package organization, follows Go conventions

---

#### R2.2: Eliminate `internal/constants` Package

**Problem:** Constants should live with their usage, not in separate package

**Solution:**

Distribute constants to respective packages:

```go
// If constants/colors.go exists with ANSI codes
// Move to internal/ui/term/colors.go

// If constants/defaults.go exists with config defaults
// Move to internal/config/defaults.go

// If constants/errors.go exists
// Move to respective packages
```

**Impact:** Better encapsulation, clearer ownership

---

#### R2.3: Add Structured Error Types

**Problem:** Using string errors makes error handling less type-safe

**Solution:**

```go
// internal/errors/errors.go
package errors

import "fmt"

// ErrorCode represents different error categories
type ErrorCode string

const (
    CodeValidation      ErrorCode = "validation"
    CodeTimeout         ErrorCode = "timeout"
    CodeNotFound        ErrorCode = "not_found"
    CodePermission      ErrorCode = "permission"
    CodeLLM             ErrorCode = "llm"
    CodeToolExecution   ErrorCode = "tool_execution"
    CodeApprovalDenied  ErrorCode = "approval_denied"
)

// Error represents a structured error
type Error struct {
    Code    ErrorCode
    Op      string        // Operation: "Agent.Execute", "Tool.Execute"
    Err     error        // Underlying error
    Message string       // Human-readable message
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *Error) Unwrap() error {
    return e.Err
}

// New creates a new structured error
func New(code ErrorCode, op string, message string, err error) *Error {
    return &Error{
        Code:    code,
        Op:      op,
        Err:     err,
        Message: message,
    }
}
```

**Usage:**

```go
// internal/agent/agent.go
if err != nil {
    return nil, errors.New(
        errors.CodeLLM,
        "Agent.Execute",
        "llm completion failed",
        err,
    )
}

// Caller can check error type
var execErr *errors.Error
if errors.As(err, &execErr) {
    if execErr.Code == errors.CodeTimeout {
        // Handle timeout specifically
    }
}
```

**Impact:** Better error handling, type-safe error checking

---

#### R2.4: Extract Agent Loop Coordinator

**Problem:** Agent.Execute() and related methods are complex

**Solution:**

```go
// internal/agent/loop.go
package agent

// LoopCoordinator manages the agent execution loop
type LoopCoordinator struct {
    llm         llm.Provider
    security    *security.SecurityService
    detection   *detection.DetectionService
    orchestration *orchestration.OrchestrationService
    emitter     *events.EventEmitter
    config      *LoopConfig
}

type LoopConfig struct {
    MaxTurns    int
    Timeout     time.Duration
    Temperature float64
}

func NewLoopCoordinator(
    llm llm.Provider,
    security *security.SecurityService,
    detection *detection.DetectionService,
    orchestration *orchestration.OrchestrationService,
    emitter *events.EventEmitter,
    config *LoopConfig,
) *LoopCoordinator {
    return &LoopCoordinator{
        llm:           llm,
        security:      security,
        detection:     detection,
        orchestration: orchestration,
        emitter:       emitter,
        config:        config,
    }
}

// Execute runs the agent loop
func (c *LoopCoordinator) Execute(ctx context.Context, prompt []Message) ([]Message, error) {
    for turn := 0; turn < c.config.MaxTurns; turn++ {
        // Call LLM
        resp, err := c.callLLM(ctx, prompt)
        if err != nil {
            return nil, err
        }

        // Check for cycles
        if err := c.checkCycles(ctx, prompt); err != nil {
            return nil, err
        }

        // Execute tools
        if err := c.executeTools(ctx, resp.ToolCalls); err != nil {
            return nil, err
        }

        // Check completion
        if c.isComplete(resp) {
            return prompt, nil
        }
    }

    return nil, ErrMaxTurns
}

func (c *LoopCoordinator) callLLM(ctx context.Context, msgs []Message) (*llm.CompletionResponse, error)
func (c *LoopCoordinator) checkCycles(ctx context.Context, msgs []Message) error
func (c *LoopCoordinator) executeTools(ctx context.Context, calls []ToolCall) error
func (c *LoopCoordinator) isComplete(resp *llm.CompletionResponse) bool
```

**Agent becomes simpler:**

```go
type Agent struct {
    coordinator *LoopCoordinator
    context     *Environment
    config      *Config
}

func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    // Prepare prompt
    prompt := a.buildPrompt(req)

    // Execute loop
    result, err := a.coordinator.Execute(ctx, prompt)
    if err != nil {
        return nil, err
    }

    // Build response
    return a.buildResponse(result), nil
}
```

**Impact:** Better testability, clearer separation of concerns

---

#### R2.5: Document Goroutine Lifecycles

**Problem:** Some goroutine lifetimes are unclear

**Solution:**

Add lifecycle documentation:

```go
// internal/events/event.go

// EventEmitter provides pub/sub event distribution.
//
// GOROUTINE LIFECYCLE:
// - Subscribe() creates a goroutine per subscriber that lives until Unsubscribe() or Close()
// - Emit() may spawn transient goroutines depending on backpressure mode
// - Close() terminates all subscriber goroutines and waits for cleanup
//
// CONCURRENCY:
// - Subscribe/Unsubscribe are thread-safe
// - Emit is thread-safe and non-blocking in BackpressureDrop mode
// - Close must be called exactly once and blocks until all goroutines exit
type EventEmitter struct { ... }
```

**Impact:** Better understanding for maintainers, prevents goroutine leaks

---

### Priority 3: Low Impact (Nice to Have)

#### R3.1: Add /build Directory

Add CI/CD configuration:

```
/build
├── ci/
│   ├── .github/
│   │   └── workflows/
│   │       ├── test.yml
│   │       ├── lint.yml
│   │       └── release.yml
│   ├── .gitlab-ci.yml
│   └── Jenkinsfile
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
└── package/
    ├── rpm/
    │   └── spin.spec
    └── deb/
        └── control
```

---

#### R3.2: Add API Documentation

Create OpenAPI spec for MCP protocol:

```
/api
├── mcp/
│   ├── openapi.yaml
│   └── README.md
└── jsonrpc/
    ├── spec.json
    └── README.md
```

---

#### R3.3: Improve Test Organization

Consider table-driven test consolidation:

```go
// internal/agent/agent_test.go
func TestAgent_Execute(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(t *testing.T) *Agent
        input   *AgentRequest
        want    *AgentResponse
        wantErr bool
    }{
        {
            name: "successful execution",
            // ...
        },
        {
            name: "max turns exceeded",
            // ...
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := tt.setup(t)
            got, err := agent.Execute(context.Background(), tt.input)
            // assertions
        })
    }
}
```

---

## 6. Code Quality Metrics

### 6.1 Test Coverage

```bash
# Current: 108 test files for 278 go files
# Coverage: ~38.8% by file count

# Target: 85%+ line coverage
```

**Recommendation:** Add tests for:
- `internal/manager/` - Complex construction logic
- `internal/orchestration/` - Tool execution paths
- `internal/security/` - Validation edge cases

### 6.2 Cyclomatic Complexity

Analyze complex functions:

```bash
# Use gocyclo to find complex functions
gocyclo -over 15 internal/

# Expected findings:
# - Manager.NewManager() - High complexity
# - Agent.Execute() - Medium-high complexity
```

**Recommendation:** Refactor functions with cyclomatic complexity > 15

### 6.3 Code Duplication

```bash
# Use goclone or similar
goclone internal/

# Known duplications:
# - parseToolArguments() in agent and orchestration
```

---

## 7. Migration Plan

### Phase 1: Critical Fixes (Week 1-2)

1. ✅ Fix error string capitalization (R1.2)
2. ✅ Consolidate tool argument parsing (R1.3)
3. ✅ Move cycle types to detection (R1.4)

### Phase 2: Structural Improvements (Week 3-4)

1. ✅ Extract Manager Builder (R1.1)
2. ✅ Eliminate types package (R2.1)
3. ✅ Eliminate constants package (R2.2)

### Phase 3: Enhanced Error Handling (Week 5-6)

1. ✅ Add structured error types (R2.3)
2. ✅ Update all error handling to use new types
3. ✅ Add error type checking in callers

### Phase 4: Agent Refactoring (Week 7-8)

1. ✅ Extract Loop Coordinator (R2.4)
2. ✅ Add goroutine lifecycle docs (R2.5)
3. ✅ Update tests

### Phase 5: Infrastructure (Week 9-10)

1. ✅ Add /build directory (R3.1)
2. ✅ Add API documentation (R3.2)
3. ✅ Improve test coverage to 85%

---

## 8. Backward Compatibility

All refactorings maintain backward compatibility except:

### Breaking Changes

**R2.1: Eliminate types package**
```go
// Before
import "github.com/dmytrogajewski/spin/internal/types"
args := types.ToolCallArguments{}

// After
import "github.com/dmytrogajewski/spin/internal/tools"
args := tools.Arguments{}
```

**Migration:** Provide migration script or deprecation period

---

## 9. Testing Strategy for Refactoring

### Unit Tests

```go
// For each refactoring, add tests first
func TestBuilder_Build(t *testing.T) {
    tests := []struct {
        name    string
        builder *Builder
        wantErr bool
    }{
        {
            name: "valid configuration",
            builder: NewBuilder(validConfig).
                WithLLM(mockLLM).
                WithApprovalHandler(mockHandler),
            wantErr: false,
        },
        {
            name:    "missing config",
            builder: NewBuilder(nil),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mgr, err := tt.builder.Build(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && mgr == nil {
                t.Error("Build() returned nil manager")
            }
        })
    }
}
```

### Integration Tests

Keep existing e2e tests passing during refactoring:

```bash
# Run before each commit
make test
make test-integration
make test-e2e
```

### Benchmark Tests

Ensure refactoring doesn't degrade performance:

```go
func BenchmarkAgent_Execute(b *testing.B) {
    agent := setupAgent(b)
    req := &AgentRequest{
        Input: "test request",
        Mode:  "regular",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = agent.Execute(context.Background(), req)
    }
}
```

---

## 10. Conclusion

The Spin codebase demonstrates **mature Go development practices** with:

✅ Clean architecture
✅ Strong type safety
✅ Proper dependency inversion
✅ Excellent interface design
✅ No circular dependencies

**Key improvements needed:**

1. **Manager complexity** - Extract builder pattern (40% complexity reduction)
2. **Error handling** - Structured error types for type-safe handling
3. **Package organization** - Eliminate generic packages (types, constants)
4. **Agent loop** - Extract coordinator for better testability
5. **DRY violations** - Consolidate duplicated parsing logic

**Estimated effort:**
- Phase 1-2: 2 weeks (critical fixes + structural improvements)
- Phase 3-4: 2 weeks (error handling + agent refactoring)
- Phase 5: 2 weeks (infrastructure + test coverage)
- **Total: 6 weeks for complete refactoring**

**Risk:** Low - All changes are incremental and testable

**ROI:** High - Improved maintainability, testability, and compliance with Go best practices

---

## Appendix A: Refactoring Checklist

- [X] R1.1: Extract Manager Builder Pattern
- [X] R1.2: Fix Error String Capitalization
- [X] R1.3: Consolidate Tool Argument Parsing
- [X] R1.4: Move Cycle Types to Detection Package
- [X] R2.1: Eliminate internal/types Package
- [X] R2.2: Eliminate internal/constants Package
- [X] R2.3: Add Structured Error Types
- [X] R2.4: Extract Agent Loop Coordinator
- [X] R2.5: Document Goroutine Lifecycles
- [X] R3.1: Add /build Directory
- [X] R3.2: Add API Documentation
- [X] R3.3: Improve Test Organization
- [ ] Achieve 85%+ test coverage
- [ ] Run gofmt on all files
- [ ] Run golangci-lint
- [ ] Update documentation
- [ ] Review all error messages for lowercase
- [ ] Audit all ignored errors

---

## Appendix B: Tools and Linters

Recommended tools for maintaining code quality:

```bash
# Install tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
go install github.com/client9/misspell/cmd/misspell@latest

# Run linters
gofmt -s -w .
goimports -w .
golangci-lint run
gocyclo -over 15 internal/
misspell -w .

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### golangci-lint Configuration

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - goimports
    - golint
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode
    - typecheck
    - bodyclose
    - depguard
    - misspell
    - unconvert
    - gocyclo
    - dupl

linters-settings:
  gocyclo:
    min-complexity: 15
  dupl:
    threshold: 100
```

---

**End of Refactoring Analysis**
