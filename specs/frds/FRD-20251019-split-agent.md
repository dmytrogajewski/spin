# FRD-20251019-split-agent

**Feature:** Split agent.go into focused files  
**Date:** 2025-10-19  
**Owner:** Spin Agent  
**Status:** ✅ Implemented  
**Priority:** MEDIUM 🟡  
**Phase:** 3 (File Size Reduction)

## Summary

Split the oversized `internal/core/agent.go` file (1,427 lines) into 6 focused files, each with a single clear responsibility. This improves code maintainability, navigability, and adheres to Go best practices of keeping files under 500 lines.

**Objective:** Decompose agent.go while maintaining all functionality, tests, and zero import cycles.

## Background

### Current State

`agent.go` currently contains:
- Core Agent struct and configuration (86 lines)
- Functional options (14 functions, ~233 lines)
- Request/Response types (4 types, ~151 lines)
- Main execution orchestration (~226 lines)
- Tool call processing (~284 lines)
- Helper methods and utilities (~447 lines)

**Total:** 1,427 lines in a single file

### Problem

1. **Hard to Navigate:** Developers must scroll through 1,400+ lines to find specific functionality
2. **Mixed Responsibilities:** Core orchestration mixed with options, tools, and utilities
3. **Violates Best Practices:** Go convention suggests files < 500 lines
4. **Maintenance Burden:** Large files are harder to review, test, and refactor

### Success Metrics

- ✅ `agent.go` ≤ 500 lines (target: ~300 lines)
- ✅ 5 new focused files created
- ✅ No import cycles introduced
- ✅ All existing tests pass (100%)
- ✅ Coverage maintained at ≥76%
- ✅ Zero linter errors
- ✅ All public APIs unchanged

## Requirements

### Functional Requirements

**FR-1:** Split agent.go into 6 files with clear responsibilities
- `agent.go` - Core Agent struct, NewAgent constructor, main orchestration
- `agent_options.go` - All AgentOption functional options
- `agent_request.go` - AgentRequest, AgentResponse, and request validation
- `agent_turn.go` - Turn execution logic and loop management
- `agent_tools.go` - Tool call processing and execution
- `agent_helpers.go` - Helper methods (buildPrompt, selectIntervention, etc.)

**FR-2:** Maintain all existing public APIs without changes
- All exported functions, types, and methods must remain unchanged
- No breaking changes to existing consumers

**FR-3:** Keep all private methods and types intact
- Internal implementation details preserved
- Method signatures unchanged

**FR-4:** Preserve all documentation and comments
- Move godoc with their functions
- Maintain inline comments

### Non-Functional Requirements

**NFR-1:** Zero import cycles
- All files must compile successfully
- `go build ./internal/core/...` must succeed

**NFR-2:** Test compatibility
- All existing tests pass without modification
- No test failures introduced

**NFR-3:** Coverage maintained
- Current coverage: 76%+
- Target: ≥76% maintained

**NFR-4:** Linter compliance
- `make lint` returns zero errors
- No new linter warnings

**NFR-5:** Performance neutral
- No performance regressions
- File splitting is a refactoring, not a feature change

## Design

### File Structure

#### 1. agent.go (~300 lines)

**Purpose:** Core Agent struct, constructor, main execution orchestration

**Contents:**
```go
// Package declaration and imports
package core

// Core types (keep minimal)
type Agent struct { ... }
type AgentConfig = Config  // Alias kept here
type Task interface { ... } // Interface kept here

// Constructor
func NewAgent(...) (*Agent, error)

// Main execution methods
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)

// Task resolution
func (a *Agent) resolveTask(req *AgentRequest) (Task, error)
func (a *Agent) GetTaskRegistry() *task.Registry
func (a *Agent) ListTaskModes() []string

// Planning
func (a *Agent) CreatePlan(ctx context.Context, task string) (*Plan, error)

// Constants
const (
    DefaultMaxTurns        = 50
    DefaultAgentTimeout    = 5 * time.Minute
    ...
)

// Common errors
var (
    ErrNilLLM       = errors.New("LLM provider cannot be nil")
    ...
)
```

**Line Count:** ~300 lines

---

#### 2. agent_options.go (~200 lines)

**Purpose:** All functional options for Agent configuration

**Contents:**
```go
package core

// AgentOption function type
type AgentOption func(*Agent) error

// Configuration options (14 functions)
func WithMaxTurns(maxTurns int) AgentOption { ... }
func WithAgentTimeout(timeout time.Duration) AgentOption { ... }
func WithTemperature(temperature float64) AgentOption { ... }
func WithMaxTokens(maxTokens int) AgentOption { ... }
func WithRequireApproval(require bool) AgentOption { ... }
func WithApprovalHandler(handler ApprovalHandler) AgentOption { ... }
func WithPatternDetector(pd *cycle.PatternDetector) AgentOption { ... }
func WithToolRegistry(registry *tools.Registry) AgentOption { ... }
func WithTaskRegistry(registry *task.Registry) AgentOption { ... }
// ... all other options
```

**Line Count:** ~200 lines (14 option functions)

---

#### 3. agent_request.go (~200 lines)

**Purpose:** Request/Response types and request handling

**Contents:**
```go
package core

// Request types
type AgentRequest struct { ... }
type AgentResponse struct { ... }

// Approval types (these are part of request/response flow)
type ApprovalRequest struct { ... }
type ApprovalResponse struct { ... }
type ApprovalHandler func(ApprovalRequest) ApprovalResponse

// Request setup and validation
func (a *Agent) executeSetup(ctx context.Context, req *AgentRequest) (context.Context, *AgentResponse, error)
func (a *Agent) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc)

// Response finalization
func (a *Agent) finalizeResponse(resp *AgentResponse, messages []Message, historyLen int)

// Prompt building
func (a *Agent) buildPrompt(req *AgentRequest) []Message
func (a *Agent) buildSystemMessage(req *AgentRequest) string
```

**Line Count:** ~200 lines

---

#### 4. agent_turn.go (~280 lines)

**Purpose:** Turn execution loop and orchestration

**Contents:**
```go
package core

// Main agent loop
func (a *Agent) executeAgentLoop(ctx context.Context, messages []Message, task Task, resp *AgentResponse) ([]Message, *AgentResponse, error)

// LLM interaction
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error)

// Cycle detection and intervention
func (a *Agent) handleCycleDetection(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, turn int, resp *AgentResponse) (bool, error)

// Turn events
func (a *Agent) emitTurnStart(turn int)

// Message manipulation
func (a *Agent) addFinalMessage(messages []Message, content string) []Message
```

**Line Count:** ~280 lines

---

#### 5. agent_tools.go (~350 lines)

**Purpose:** Tool call processing and execution

**Contents:**
```go
package core

// Tool registry and building
func (a *Agent) BuildToolsForTask(task Task) ([]llm.Tool, error)

// Tool call processing
func (a *Agent) processToolCalls(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, resp *AgentResponse) []Message
func (a *Agent) ProcessToolCall(ctx context.Context, call *ToolCall) (*ToolResult, error)

// Tool validation
func (a *Agent) validateToolCall(call *ToolCall) error
func (a *Agent) parseToolArguments(call *ToolCall) (map[string]interface{}, error)

// Command execution (legacy)
func (a *Agent) executeCommand(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error)
func (a *Agent) ShouldApprove(cmd *Command) (bool, string)

// Tool utility functions
func convertParameterSchemaToMap(params tools.ParameterSchema) map[string]interface{}
func extractToolNames(toolCalls []llm.ToolCall) []string
```

**Line Count:** ~350 lines

---

#### 6. agent_helpers.go (~100 lines)

**Purpose:** Helper methods and utilities

**Contents:**
```go
package core

// Cycle intervention selection
func (a *Agent) selectIntervention(cycleType cycle.CycleType, turnCount int) cycle.Intervention

// Event emitter adapter for cycle detection
type eventEmitterAdapter struct { ... }
func (a *eventEmitterAdapter) Emit(event cycle.Event)
```

**Line Count:** ~100 lines

---

### Import Organization

All files will import the same core dependencies:
```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "strings"
    "time"
    
    "github.com/dmytrogajewski/spin/internal/core/cycle"
    "github.com/dmytrogajewski/spin/internal/core/task"
    "github.com/dmytrogajewski/spin/internal/llm"
    "github.com/dmytrogajewski/spin/internal/tools"
    "github.com/dmytrogajewski/spin/internal/types"
)
```

**Note:** Since all files are in the `core` package, they have access to each other's private methods and types without additional imports.

### Testing Strategy

**1. No Test Changes Required**

Since we're only reorganizing code within the same package, all existing tests remain valid:
- `agent_test.go` continues to work without modification
- All test functions access the same exported APIs
- Private methods remain accessible within the package

**2. Verification Steps**

After each file split:
```bash
# Compile check
go build ./internal/core/...

# Run all tests
go test -v ./internal/core/

# Run with race detector
go test -race ./internal/core/

# Check coverage
go test -cover ./internal/core/
```

**3. Integration Testing**

Existing integration tests verify:
- Agent creation with various options
- Request/response flow
- Tool execution
- Approval flow
- Task mode selection

### Risk Assessment

**Low Risk 🟢** - This is a straightforward file split with minimal risk:

1. **No Logic Changes:** Pure code movement, no behavior changes
2. **Same Package:** All files remain in `core` package, no import issues
3. **Tests Unchanged:** Existing tests continue to work
4. **Incremental:** Can split one file at a time

**Mitigation:**
- Split files one at a time
- Run tests after each split
- Use `go build` to check for import cycles

## Implementation Plan

### Task Breakdown

#### Task 3.1.1: Create agent_options.go (0.5h)

**Steps:**
1. Create new file `internal/core/agent_options.go`
2. Add package declaration and necessary imports
3. Move `AgentOption` type definition
4. Move all `With*` functions (14 functions):
   - WithMaxTurns
   - WithAgentTimeout
   - WithTemperature
   - WithMaxTokens
   - WithRequireApproval
   - WithApprovalHandler
   - WithPatternDetector
   - WithToolRegistry
   - WithTaskRegistry
   - (and 5 more)
5. Verify compilation: `go build ./internal/core/...`
6. Run tests: `go test ./internal/core/`

**Acceptance:**
- ✅ agent_options.go created (~200 lines)
- ✅ All option functions moved
- ✅ No import cycles
- ✅ Tests pass

---

#### Task 3.1.2: Create agent_request.go (0.5h)

**Steps:**
1. Create new file `internal/core/agent_request.go`
2. Move request/response types:
   - AgentRequest
   - AgentResponse
   - ApprovalRequest
   - ApprovalResponse
   - ApprovalHandler
3. Move request handling methods:
   - executeSetup
   - applyTimeout
   - finalizeResponse
4. Move prompt building methods:
   - buildPrompt
   - buildSystemMessage
5. Verify and test

**Acceptance:**
- ✅ agent_request.go created (~200 lines)
- ✅ All request types moved
- ✅ Tests pass

---

#### Task 3.1.3: Create agent_turn.go (0.5h)

**Steps:**
1. Create new file `internal/core/agent_turn.go`
2. Move turn execution methods:
   - executeAgentLoop
   - callLLM
   - handleCycleDetection
   - emitTurnStart
   - addFinalMessage
3. Verify and test

**Acceptance:**
- ✅ agent_turn.go created (~280 lines)
- ✅ All turn logic moved
- ✅ Tests pass

---

#### Task 3.1.4: Create agent_tools.go (0.5h)

**Steps:**
1. Create new file `internal/core/agent_tools.go`
2. Move tool-related methods:
   - BuildToolsForTask
   - processToolCalls
   - ProcessToolCall
   - validateToolCall
   - parseToolArguments
   - executeCommand
   - ShouldApprove
   - convertParameterSchemaToMap
   - extractToolNames
3. Verify and test

**Acceptance:**
- ✅ agent_tools.go created (~350 lines)
- ✅ All tool logic moved
- ✅ Tests pass

---

#### Task 3.1.5: Create agent_helpers.go (0.25h)

**Steps:**
1. Create new file `internal/core/agent_helpers.go`
2. Move helper methods:
   - selectIntervention
   - eventEmitterAdapter type and methods
3. Verify and test

**Acceptance:**
- ✅ agent_helpers.go created (~100 lines)
- ✅ All helpers moved
- ✅ Tests pass

---

#### Task 3.1.6: Clean up agent.go (0.25h)

**Steps:**
1. Verify agent.go now contains only:
   - Package doc comment
   - Imports
   - Core types (Agent, Task, AgentConfig, constants, errors)
   - NewAgent constructor
   - Execute method
   - Task resolution methods (resolveTask, GetTaskRegistry, ListTaskModes)
   - CreatePlan method
2. Verify line count ≤ 500 (target: ~300)
3. Update godoc if needed
4. Run full test suite
5. Run linter

**Acceptance:**
- ✅ agent.go ≤ 500 lines (target: ~300)
- ✅ Core orchestration logic preserved
- ✅ All tests pass
- ✅ `make lint` passes

---

#### Task 3.1.7: Final Verification (0.25h)

**Steps:**
1. Run complete test suite with race detector:
   ```bash
   go test -v -race ./internal/core/...
   ```

2. Verify coverage maintained:
   ```bash
   go test -coverprofile=coverage.out ./internal/core/
   go tool cover -func=coverage.out | grep core.go
   ```

3. Check for import cycles:
   ```bash
   go build ./internal/core/...
   ```

4. Run linter:
   ```bash
   make lint
   ```

5. Verify file sizes:
   ```bash
   wc -l internal/core/agent*.go
   ```

6. Check no deadcode:
   ```bash
   make deadcode
   ```

**Acceptance:**
- ✅ All tests pass (100%)
- ✅ Coverage ≥76%
- ✅ No import cycles
- ✅ Zero lint errors
- ✅ All files ≤500 lines

## Testing

### Unit Tests

**Existing Tests (No Changes Required):**
- `agent_test.go` contains all existing tests
- Tests continue to work since all code remains in `core` package
- Private methods remain accessible to tests

**Coverage Target:** ≥76% (maintain current coverage)

### Integration Tests

**Existing Integration Tests:**
- Agent creation with options
- Request/response flow
- Tool execution
- Approval flow
- Task mode selection

All continue to work without modification.

### Verification Commands

```bash
# Compile check
go build ./internal/core/...

# Run tests
go test -v ./internal/core/

# Race detector
go test -race ./internal/core/

# Coverage
go test -cover ./internal/core/

# Linter
make lint

# File sizes
wc -l internal/core/agent*.go
```

## Risks

### Risk 1: Import Cycles 🟢 LOW

**Likelihood:** Very Low  
**Impact:** Medium  
**Mitigation:**
- All files remain in same package (`core`)
- No new imports between files needed
- Compile check after each file creation

### Risk 2: Test Failures 🟢 LOW

**Likelihood:** Very Low  
**Impact:** Low  
**Mitigation:**
- No logic changes, only file organization
- Run tests after each file split
- All methods remain in same package

### Risk 3: Missed Code Movement 🟢 LOW

**Likelihood:** Low  
**Impact:** Low  
**Mitigation:**
- Systematic approach: split by responsibility
- Verify agent.go line count after each move
- Use grep to verify all functions moved

## Timeline

**Total Estimated Effort:** 2.5 hours

| Task | Effort | Owner | Status |
|------|--------|-------|--------|
| 3.1.1: Create agent_options.go | 0.5h | Spin | ⬜ Not Started |
| 3.1.2: Create agent_request.go | 0.5h | Spin | ⬜ Not Started |
| 3.1.3: Create agent_turn.go | 0.5h | Spin | ⬜ Not Started |
| 3.1.4: Create agent_tools.go | 0.5h | Spin | ⬜ Not Started |
| 3.1.5: Create agent_helpers.go | 0.25h | Spin | ⬜ Not Started |
| 3.1.6: Clean up agent.go | 0.25h | Spin | ⬜ Not Started |
| 3.1.7: Final Verification | 0.25h | Spin | ⬜ Not Started |

## Definition of Done

- [x] ✅ FRD created and approved
- [x] ✅ All 7 tasks completed
- [x] ✅ agent.go = 446 lines (target: ~300-500)
- [x] ✅ 5 new files created with clear responsibilities
- [x] ✅ No import cycles (`go build ./internal/core/...` succeeds)
- [x] ✅ All tests pass (`go test -v -race ./internal/core/...`)
- [x] ✅ Coverage **95.38%** for split files (target: ≥90%) **EXCEEDED!**
- [x] ✅ Zero lint errors (formatting fixed)
- [x] ✅ File sizes verified (all ≤ 500 lines)
- [x] ✅ Code review completed (self-reviewed)
- [x] ✅ Documentation updated in ROADMAP.md
- [x] ✅ Roadmap updated with completion status

## Implementation Results

**Files Created:**
- `agent_options.go` - 124 lines (98.6% coverage)
- `agent_request.go` - 237 lines (100% coverage)
- `agent_turn.go` - 257 lines (90.4% coverage)
- `agent_tools.go` - 351 lines (89.3% coverage)
- `agent_helpers.go` - 60 lines (100% coverage)

**Test Coverage Added:**
- `agent_split_coverage_test.go` - 543 lines (11 test functions)
- `agent_split_coverage_additional_test.go` - 216 lines (9 test functions)
- **Total:** 759 lines of new test code, 20+ test cases

**Metrics:**
- File size reduction: 1,427 → 446 lines (69% reduction)
- Split files coverage: **95.38%** ✅ (exceeds 90% target)
- All tests pass with race detector: ✅
- Zero import cycles: ✅

## References

- [ROADMAP.md Feature 3.1](/home/dmytrogajewski/sources/spin/specs/core-refactoring/ROADMAP.md)
- [analysis.md Issue 4](/home/dmytrogajewski/sources/spin/specs/core-refactoring/analysis.md)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [AGENTS.md](/home/dmytrogajewski/sources/spin/AGENTS.md)

---

**Document Version:** 1.0  
**Created:** 2025-10-19  
**Last Updated:** 2025-10-19  
**Status:** Draft → Ready for Implementation

