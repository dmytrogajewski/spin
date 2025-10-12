# Feature Requirements Document: Implement Tool Filtering

**FRD-20251012132817**

**Status**: Draft
**Created**: 2025-10-12
**Author**: Spin Agent
**Related**: [P1.3] Task Mode System Integration - Phase 1
**Priority**: HIGH - Core security feature
**Estimated Effort**: 4-5 hours
**Depends On**:
- P1.1 (Task Registry to Agent) - ✅ COMPLETE
- P1.2 (Task Resolution Logic) - ✅ COMPLETE

## Overview

This FRD describes the implementation of P1.3 from the task-modes roadmap: filtering the tool list based on task mode's allowed tools before sending to LLM.

## Context

### Current State

After P1.1 and P1.2 completion:
- ✅ Agent has `taskRegistry` field with 4 modes
- ✅ `resolveTask()` method determines which task to use
- ✅ Each task defines `AllowedTools()` method
- ❌ Tools are not filtered based on task mode
- ❌ `callLLM()` doesn't know about tasks
- ❌ All tools always sent to LLM regardless of mode

### Problem Statement

Currently, task modes cannot enforce tool restrictions:
- Review mode should only have read-only tools, but has all tools
- Compact mode should have minimal tools, but has all tools
- Planning mode should only have context tools, but has all tools
- Security risk: LLM could call dangerous tools in restricted modes

This defeats the purpose of having specialized task modes and creates security vulnerabilities.

### Goals

1. Implement `buildToolsForTask(Task) ([]llm.Tool, error)` method
2. Update `callLLM()` signature to accept Task parameter
3. Update all `callLLM()` invocations to pass resolved task
4. Filter tools using set-based O(1) lookup
5. Achieve < 100μs filtering performance for 50 tools
6. Maintain backward compatibility
7. Achieve ≥90% test coverage

### Non-Goals

- Token budget application (P1.4)
- Conversation-level integration (Phase 2)
- CLI/REPL integration (Phase 3)
- Custom tool definitions (future work)

## Requirements

### Functional Requirements

#### FR1: Implement buildToolsForTask() Method

**Priority**: CRITICAL

**Description**: Filter tool schemas based on task mode's allowed tools.

**Acceptance Criteria**:
```go
// buildToolsForTask constructs the filtered tool list for the LLM request,
// based on the task mode's allowed tools.
//
// Algorithm:
//  1. Get all tool schemas from tool registry
//  2. Get allowed tool names from task.AllowedTools()
//  3. Build allowed tool set for O(1) lookup
//  4. Filter tool schemas, keeping only allowed tools
//  5. Convert to LLM tool format
//
// Returns an error if tool registry is nil.
// Returns empty slice if no tools are allowed (not an error).
func (a *Agent) buildToolsForTask(task Task) ([]llm.Tool, error) {
	if a.toolRegistry == nil {
		return nil, nil
	}

	// Get all available tools
	allSchemas := a.toolRegistry.ListSchemas()
	if len(allSchemas) == 0 {
		return nil, nil
	}

	// Get allowed tools for this mode
	allowedTools := task.AllowedTools()
	if len(allowedTools) == 0 {
		slog.Debug("task allows no tools", "task", task.Name())
		return []llm.Tool{}, nil
	}

	// Build allowed tool set for O(1) lookup
	allowedSet := make(map[string]bool, len(allowedTools))
	for _, name := range allowedTools {
		allowedSet[name] = true
	}

	// Filter tools
	filtered := make([]llm.Tool, 0, len(allSchemas))
	for _, schema := range allSchemas {
		// Check if tool is allowed in this mode
		if !allowedSet[schema.Function.Name] {
			continue
		}

		// Convert ParameterSchema struct to map[string]interface{}
		params := convertParameterSchemaToMap(schema.Function.Parameters)

		filtered = append(filtered, llm.Tool{
			Type: schema.Type,
			Function: llm.Function{
				Name:        schema.Function.Name,
				Description: schema.Function.Description,
				Parameters:  params,
			},
		})
	}

	slog.Debug("filtered tools for task",
		"task", task.Name(),
		"total", len(allSchemas),
		"allowed", len(filtered))

	return filtered, nil
}
```

**Validation**:
- Must use set-based filtering for O(1) lookup
- Must handle nil tool registry gracefully
- Must handle empty allowed tools list (return empty slice, not error)
- Must preserve tool schema structure
- Must log filtering results for debugging

---

#### FR2: Update callLLM() Signature

**Priority**: CRITICAL

**Description**: Modify `callLLM()` to accept Task parameter and use filtered tools.

**Current Signature**:
```go
func (a *Agent) callLLM(ctx context.Context, messages []Message) (*llm.CompletionResponse, error)
```

**New Signature**:
```go
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error)
```

**Acceptance Criteria**:
- Add `task Task` parameter to signature
- Replace existing tool list building with `buildToolsForTask(task)`
- Update godoc to explain task parameter
- Maintain all existing functionality

**Changes Required**:
```go
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
	// ... existing message conversion code ...

	// Build filtered tool list for this task mode
	tools, err := a.buildToolsForTask(task)
	if err != nil {
		return nil, fmt.Errorf("failed to build tools: %w", err)
	}

	// Build LLM request with filtered tools
	req := llm.CompletionRequest{
		Messages:    llmMessages,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
		Tools:       tools,
	}

	// ... rest of method (unchanged) ...
}
```

---

#### FR3: Update All callLLM() Call Sites

**Priority**: CRITICAL

**Description**: Find and update all places where `callLLM()` is invoked to pass the resolved task.

**Call Sites to Update** (from grep):
```bash
$ grep -n "callLLM" internal/core/agent.go
588:func (a *Agent) callLLM(ctx context.Context, messages []Message) (*llm.CompletionResponse, error) {
XXX:    llmResp, err := a.callLLM(ctx, messages)  // In Execute()
```

**Required Changes**:
1. In `Execute()` method: Resolve task early and pass to callLLM
2. Store resolved task in local variable for repeated use
3. Pass task to callLLM on every invocation

**Implementation Pattern**:
```go
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
	// ... validation ...

	// Resolve task once at start of execution
	task, err := a.resolveTask(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve task mode: %w", err)
	}

	// ... build messages ...

	// Pass task to callLLM
	llmResp, err := a.callLLM(ctx, messages, task)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// ... rest of execution ...
}
```

---

#### FR4: Tool Filtering Verification

**Priority**: HIGH

**Description**: Verify correct tools are exposed for each mode.

**Expected Tool Lists**:

**Regular Mode** (all tools):
- bash
- read_file
- write_file
- list_directory
- get_context
- apply_patch
- file_search
- git_context

**Review Mode** (read-only):
- read_file
- list_directory
- get_context
- file_search
- git_context

**Compact Mode** (minimal):
- read_file
- get_context
- file_search

**Planning Mode** (context only):
- get_context
- file_search
- git_context

**Validation**:
- Each mode filters correctly
- No extra tools exposed
- No missing expected tools
- Tool schemas preserved correctly

---

### Non-Functional Requirements

#### NFR1: Performance

**Description**: Tool filtering must be extremely fast.

**Acceptance Criteria**:
- Filtering time: < 100μs for 50 tools
- Set-based lookup ensures O(1) per tool
- No unnecessary allocations
- Pre-sized slices for efficiency

**Validation**:
```bash
go test -bench=BenchmarkBuildToolsForTask ./internal/core/
```

**Expected Results**:
```
BenchmarkBuildToolsForTask/regular-8     50000    80 µs/op    2048 B/op    10 allocs/op
BenchmarkBuildToolsForTask/review-8      60000    75 µs/op    1536 B/op     8 allocs/op
BenchmarkBuildToolsForTask/compact-8     80000    65 µs/op     512 B/op     5 allocs/op
```

---

#### NFR2: Security

**Description**: Tool filtering must be enforced and not bypassable.

**Acceptance Criteria**:
- Filtering happens before LLM call (not after)
- No way to bypass filtering
- Invalid tool calls fail validation
- Audit logging for security analysis

**Validation**:
- Review mode cannot execute bash
- Compact mode cannot write files
- Planning mode cannot execute commands
- Attempts logged for security review

---

#### NFR3: Backward Compatibility

**Description**: Existing code must continue to work.

**Acceptance Criteria**:
- All existing tests pass without modification
- Default behavior unchanged (regular mode has all tools)
- No breaking changes to public API
- Optional task parameter in requests

**Validation**:
```bash
go test ./internal/core/
```

---

#### NFR4: Code Quality

**Description**: Code must meet project standards.

**Acceptance Criteria**:
- `make lint` passes (zero errors)
- Cyclomatic complexity ≤ 10 for buildToolsForTask
- Test coverage ≥ 90%
- Godoc complete
- No dead code
- Race detector clean

**Validation**:
```bash
make lint
go test -cover ./internal/core/
go test -race ./internal/core/
gocyclo -over 10 internal/core/agent.go
```

---

## Technical Design

### Algorithm: buildToolsForTask

```
INPUT: task Task
OUTPUT: []llm.Tool, error

FUNCTION buildToolsForTask(task):
    // Guard: nil registry
    IF toolRegistry == nil:
        RETURN [], nil

    // Get all available tools
    allSchemas = toolRegistry.ListSchemas()
    IF len(allSchemas) == 0:
        RETURN [], nil

    // Get allowed tools for this mode
    allowedTools = task.AllowedTools()
    IF len(allowedTools) == 0:
        LOG "task allows no tools"
        RETURN [], nil

    // Build allowed set for O(1) lookup
    allowedSet = map[string]bool{}
    FOR EACH name IN allowedTools:
        allowedSet[name] = true

    // Filter tools
    filtered = []llm.Tool{}  // Pre-allocate with cap=len(allSchemas)
    FOR EACH schema IN allSchemas:
        IF schema.Function.Name NOT IN allowedSet:
            CONTINUE  // Skip disallowed tool

        // Convert parameters
        params = convertParameterSchemaToMap(schema.Function.Parameters)

        // Add to filtered list
        filtered.append(llm.Tool{
            Type: schema.Type,
            Function: llm.Function{
                Name:        schema.Function.Name,
                Description: schema.Function.Description,
                Parameters:  params,
            }
        })

    LOG "filtered tools" task=task.Name() total=len(allSchemas) allowed=len(filtered)
    RETURN filtered, nil
```

**Time Complexity**: O(n + m) where n = allowed tools, m = total tools
**Space Complexity**: O(n + k) where n = allowed tools set, k = filtered tools

---

### Call Chain Update

**Before** (P1.2):
```
Execute()
  → resolveTask() → Task
  → callLLM(ctx, messages)
      → toolRegistry.ListSchemas() → all tools
      → send all tools to LLM
```

**After** (P1.3):
```
Execute()
  → resolveTask() → Task
  → callLLM(ctx, messages, task)
      → buildToolsForTask(task)
          → toolRegistry.ListSchemas() → all tools
          → task.AllowedTools() → allowed list
          → filter → subset of tools
      → send filtered tools to LLM
```

---

### Data Flow

```
┌─────────────────────────────────────────────┐
│         Execute(AgentRequest)               │
│  1. Resolve task (P1.2)                     │
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│         task = resolveTask(req)             │
│         (returns Task object)               │
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│       callLLM(ctx, messages, task)          │
│  1. buildToolsForTask(task)                 │
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│       buildToolsForTask(task)               │
│  1. Get all tools from registry             │
│  2. Get allowed tools from task             │
│  3. Build allowedSet map                    │
│  4. Filter: keep only allowed               │
│  5. Return filtered tools                   │
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│         LLM Request                         │
│  Tools: [filtered subset]                   │
│  - Regular: all 8 tools                     │
│  - Review: 5 read-only tools                │
│  - Compact: 3 minimal tools                 │
│  - Planning: 3 context tools                │
└─────────────────────────────────────────────┘
```

---

## Testing Strategy

### Unit Tests

#### Test 1: Regular Mode Has All Tools
```go
func TestAgent_BuildToolsForTask_RegularMode(t *testing.T) {
	agent := newTestAgent(t)

	regularTask, _ := agent.taskRegistry.Get("regular")
	tools, err := agent.buildToolsForTask(regularTask)

	require.NoError(t, err)
	require.NotNil(t, tools)

	// Regular mode should have all 8 tools
	require.Equal(t, 8, len(tools))

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	expected := []string{"bash", "read_file", "write_file", "list_directory",
		"get_context", "apply_patch", "file_search", "git_context"}
	for _, name := range expected {
		require.Contains(t, toolNames, name)
	}
}
```

#### Test 2: Review Mode Read-Only Tools
```go
func TestAgent_BuildToolsForTask_ReviewMode(t *testing.T) {
	agent := newTestAgent(t)

	reviewTask, _ := agent.taskRegistry.Get("review")
	tools, err := agent.buildToolsForTask(reviewTask)

	require.NoError(t, err)
	require.NotNil(t, tools)

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	// Review mode should only have read-only tools
	expectedAllowed := []string{"read_file", "list_directory", "get_context",
		"file_search", "git_context"}
	for _, name := range expectedAllowed {
		require.Contains(t, toolNames, name)
	}

	// Should NOT have write tools
	forbidden := []string{"bash", "write_file", "apply_patch"}
	for _, name := range forbidden {
		require.NotContains(t, toolNames, name)
	}
}
```

#### Test 3: Compact Mode Minimal Tools
```go
func TestAgent_BuildToolsForTask_CompactMode(t *testing.T) {
	agent := newTestAgent(t)

	compactTask, _ := agent.taskRegistry.Get("compact")
	tools, err := agent.buildToolsForTask(compactTask)

	require.NoError(t, err)
	require.Equal(t, 3, len(tools))

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	expected := []string{"read_file", "get_context", "file_search"}
	require.ElementsMatch(t, expected, toolNames)
}
```

#### Test 4: Planning Mode Context Tools
```go
func TestAgent_BuildToolsForTask_PlanningMode(t *testing.T) {
	agent := newTestAgent(t)

	planningTask, _ := agent.taskRegistry.Get("planning")
	tools, err := agent.buildToolsForTask(planningTask)

	require.NoError(t, err)
	require.Equal(t, 3, len(tools))

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	expected := []string{"get_context", "file_search", "git_context"}
	require.ElementsMatch(t, expected, toolNames)
}
```

#### Test 5: Nil Tool Registry Handling
```go
func TestAgent_BuildToolsForTask_NilRegistry(t *testing.T) {
	agent := newTestAgent(t)
	agent.toolRegistry = nil  // Simulate nil registry

	task, _ := agent.taskRegistry.Get("regular")
	tools, err := agent.buildToolsForTask(task)

	require.NoError(t, err)
	require.Nil(t, tools)
}
```

#### Test 6: Empty Allowed Tools
```go
func TestAgent_BuildToolsForTask_NoTools(t *testing.T) {
	agent := newTestAgent(t)

	// Create mock task with no allowed tools
	mockTask := &mockTask{
		name:         "notools",
		allowedTools: []string{},
	}

	tools, err := agent.buildToolsForTask(mockTask)

	require.NoError(t, err)
	require.NotNil(t, tools)
	require.Equal(t, 0, len(tools))
}
```

#### Test 7: Tool Schemas Preserved
```go
func TestAgent_BuildToolsForTask_SchemaPreserved(t *testing.T) {
	agent := newTestAgent(t)

	task, _ := agent.taskRegistry.Get("regular")
	tools, err := agent.buildToolsForTask(task)

	require.NoError(t, err)

	// Find bash tool
	var bashTool *llm.Tool
	for i := range tools {
		if tools[i].Function.Name == "bash" {
			bashTool = &tools[i]
			break
		}
	}

	require.NotNil(t, bashTool, "bash tool should be present")
	require.Equal(t, "function", bashTool.Type)
	require.NotEmpty(t, bashTool.Function.Description)
	require.NotNil(t, bashTool.Function.Parameters)
}
```

### Integration Tests

#### Test 8: End-to-End Tool Filtering
```go
func TestAgent_Execute_ToolFiltering(t *testing.T) {
	// This test will be part of P1.5 integration tests
	// Verifies that Execute() properly filters tools through entire chain
}
```

### Benchmarks

#### Benchmark 1: Regular Mode (All Tools)
```go
func BenchmarkBuildToolsForTask_Regular(b *testing.B) {
	agent := newBenchAgent(b)
	task, _ := agent.taskRegistry.Get("regular")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.buildToolsForTask(task)
		if err != nil {
			b.Fatal(err)
		}
		_ = tools
	}
}
```

#### Benchmark 2: Review Mode (Subset)
```go
func BenchmarkBuildToolsForTask_Review(b *testing.B) {
	agent := newBenchAgent(b)
	task, _ := agent.taskRegistry.Get("review")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tools, err := agent.buildToolsForTask(task)
		if err != nil {
			b.Fatal(err)
		}
		_ = tools
	}
}
```

---

## Implementation Plan

### Step 1: Implement buildToolsForTask() Method (1 hour)
- Add method to agent.go
- Implement set-based filtering
- Add debug logging
- Handle edge cases (nil registry, empty tools)

### Step 2: Update callLLM() Signature (30 min)
- Add task parameter to signature
- Replace tool building with buildToolsForTask()
- Update godoc

### Step 3: Update Execute() to Pass Task (30 min)
- Call resolveTask() early in Execute()
- Store resolved task in local variable
- Pass task to callLLM()

### Step 4: Write Unit Tests (1.5 hours)
- Write all 7 unit tests
- Ensure ≥90% coverage

### Step 5: Write Benchmarks (30 min)
- Write 2 benchmarks
- Verify < 100μs performance

### Step 6: Quality Checks (30 min)
- Run all tests
- Run lint
- Run race detector
- Check coverage

### Step 7: Integration Testing (30 min)
- Manual testing with different modes
- Verify tool filtering works end-to-end

**Total Estimate**: ~5 hours

---

## Risks and Mitigations

### Risk 1: Breaking Existing Tests
**Likelihood**: High (signature change affects many places)
**Impact**: High
**Mitigation**: Update all call sites systematically, run tests frequently
**Contingency**: Revert changes, fix one by one

### Risk 2: Missing Call Sites
**Likelihood**: Medium
**Impact**: High (compilation failure)
**Mitigation**: Use grep to find all callLLM invocations
**Contingency**: Compiler will catch missing updates

### Risk 3: Performance Regression
**Likelihood**: Low (O(1) lookup is fast)
**Impact**: Medium
**Mitigation**: Benchmark before/after, use set-based filtering
**Contingency**: Optimize with caching if needed

### Risk 4: Tool Name Mismatches
**Likelihood**: Medium
**Impact**: Medium (tools silently filtered)
**Mitigation**: Add debug logging, write comprehensive tests
**Contingency**: Add validation test comparing registry names with task names

---

## Success Criteria

### Definition of Done

- [ ] `buildToolsForTask()` method implemented
- [ ] `callLLM()` signature updated with task parameter
- [ ] All `callLLM()` call sites updated
- [ ] All 7 unit tests written and passing
- [ ] All 2 benchmarks written and meeting targets (< 100μs)
- [ ] Test coverage ≥ 90% for new code
- [ ] `make lint` passes (zero errors)
- [ ] Race detector clean (`go test -race`)
- [ ] All existing tests still pass
- [ ] Godoc complete
- [ ] Performance targets met

### Acceptance Testing

1. **Regular Mode**: Has all 8 tools
2. **Review Mode**: Has only 5 read-only tools
3. **Compact Mode**: Has only 3 minimal tools
4. **Planning Mode**: Has only 3 context tools
5. **Performance**: < 100μs for 50 tools
6. **Security**: Forbidden tools are filtered out
7. **Compatibility**: All existing tests pass

---

## Dependencies

### Upstream Dependencies (Must Be Complete)
- ✅ P1.1: Task registry in Agent (COMPLETE)
- ✅ P1.2: Task resolution logic (COMPLETE)
- ✅ Tool registry with ListSchemas() (EXISTS)
- ✅ Task interface with AllowedTools() (EXISTS)

### Downstream Dependencies (Blocked By This)
- P1.4: Token budget application
- P1.5: Integration tests for core agent
- Phase 2-5: All dependent on filtered tools

---

## Open Questions

1. **Q**: Should we cache filtered tool lists per mode?
   **A**: NO - Filtering is already < 100μs, caching adds complexity

2. **Q**: What if a task references a tool that doesn't exist in registry?
   **A**: Silently skip (with debug log). Not an error - allows forward compatibility

3. **Q**: Should we validate tool names at task registration time?
   **A**: FUTURE - Good idea but not in scope for P1.3. Add to backlog

4. **Q**: How to handle dynamic tool registration after agent creation?
   **A**: Works fine - filtering happens at call time, always gets current registry

---

## Revision History

| Date       | Version | Author     | Changes              |
|------------|---------|------------|----------------------|
| 2025-10-12 | 1.0     | Spin Agent | Initial draft        |

---

## Approval

**Author**: Spin Agent
**Reviewers**: [To be assigned]
**Status**: Draft → Ready for Implementation

---

## References

1. [Task Modes Specification](../task-modes/specification.md)
2. [Task Modes Roadmap](../task-modes/ROADMAP.md) - P1.3
3. [AGENTS.md](../../AGENTS.md)
4. [Core Package Docs](../../docs/packages/core.md)
5. [P1.1 FRD](./FRD-20251012130940-task-registry-agent.md)
6. [P1.2 FRD](./FRD-20251012132252-task-resolution-logic.md)
