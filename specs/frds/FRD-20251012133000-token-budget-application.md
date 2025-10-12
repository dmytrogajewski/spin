# FRD-20251012133000: Apply Token Budget from Task

**Status**: Implementation
**Created**: 2025-10-12
**Priority**: MEDIUM
**Complexity**: Low (1 hour)
**Related**:
- [FRD-20251012130940-task-registry-agent.md](FRD-20251012130940-task-registry-agent.md)
- [FRD-20251012132252-task-resolution-logic.md](FRD-20251012132252-task-resolution-logic.md)
- [FRD-20251012132817-tool-filtering.md](FRD-20251012132817-tool-filtering.md)
- [specs/task-modes/ROADMAP.md](../task-modes/ROADMAP.md) - P1.4

## Overview

Apply task-specific token budgets when calling the LLM, allowing different task modes to use appropriate context windows. This enables cost optimization (compact mode uses 4K tokens) and capability enhancement (regular mode uses 16K tokens).

## Problem Statement

Currently, the agent uses the global `agent.config.MaxTokens` for all LLM calls, regardless of task mode. This means:

1. **No Cost Optimization**: Compact mode queries use the same large token budget as regular mode
2. **No Mode Differentiation**: Task modes can't control their own token budgets
3. **Ignored Task Settings**: The `Task.MaxTokens()` method exists but is never used

## Goals

1. **Use Task-Specific Budgets**: Honor `task.MaxTokens()` when calling LLM
2. **Maintain Fallback**: Use `agent.config.MaxTokens` when task has no budget (0)
3. **Clear Precedence**: Document the token budget resolution logic
4. **Zero Breaking Changes**: Existing behavior unchanged when task budget is 0

## Non-Goals

- Implement dynamic token budget adjustment based on context
- Add token usage tracking or billing
- Validate token budgets against provider limits

## Current State Analysis

### Current Implementation

File: `internal/core/agent.go:731-737`

```go
// Build LLM request with filtered tools
req := llm.CompletionRequest{
    Messages:    llmMessages,
    Temperature: a.config.Temperature,
    MaxTokens:   a.config.MaxTokens,  // ❌ Always uses agent config
    Tools:       tools,
}
```

### Task Modes and Token Budgets

From `internal/core/task/`:

| Mode     | MaxTokens | Purpose                 |
|----------|-----------|-------------------------|
| regular  | 16384     | Full-featured coding    |
| review   | 12288     | Code analysis           |
| compact  | 4096      | Quick queries           |
| planning | 4096      | Task decomposition      |

All task modes **already implement** `MaxTokens()` method, but it's never called.

## Detailed Design

### Token Budget Resolution Logic

Precedence (highest to lowest):

1. **Task Budget**: If `task.MaxTokens() > 0`, use it
2. **Agent Config**: Otherwise, fall back to `a.config.MaxTokens`

### Implementation

File: `internal/core/agent.go:731-737`

```go
// Determine token budget: task overrides agent config
maxTokens := a.config.MaxTokens
if task != nil {
    taskMaxTokens := task.MaxTokens()
    if taskMaxTokens > 0 {
        maxTokens = taskMaxTokens
    }
}

// Build LLM request with filtered tools
req := llm.CompletionRequest{
    Messages:    llmMessages,
    Temperature: a.config.Temperature,
    MaxTokens:   maxTokens,  // ✅ Uses task-specific or agent default
    Tools:       tools,
}
```

### Edge Cases

| Case                          | Behavior                          |
|-------------------------------|-----------------------------------|
| `task == nil`                 | Use `a.config.MaxTokens`         |
| `task.MaxTokens() == 0`       | Use `a.config.MaxTokens`         |
| `task.MaxTokens() > 0`        | Use `task.MaxTokens()`           |
| No task mode specified        | Uses default task (regular: 16K) |

## Testing Strategy

### Unit Tests

File: `internal/core/agent_test.go`

#### Test 1: Task Budget Overrides Agent Config

```go
func TestAgent_TaskBudgetOverridesConfig(t *testing.T) {
    agent := newTestAgentWithConfig(t, &AgentConfig{
        MaxTokens: 4096,
    })

    // Regular mode has 16K tokens
    task := task.NewRegular()
    assert.Equal(t, 16384, task.MaxTokens())

    // Create request with regular task
    req := &AgentRequest{
        Input:   "test",
        Task:    task,
        WorkDir: t.TempDir(),
    }

    // Capture LLM request
    mockLLM := &mockLLMProvider{
        captureRequest: true,
    }
    agent.llm = mockLLM

    // Execute
    _, _ = agent.Execute(context.Background(), req)

    // Verify task budget was used (16K, not 4K)
    assert.Equal(t, 16384, mockLLM.lastRequest.MaxTokens)
}
```

#### Test 2: Agent Config Fallback When Task Budget is Zero

```go
func TestAgent_AgentConfigFallbackWhenTaskBudgetZero(t *testing.T) {
    agent := newTestAgentWithConfig(t, &AgentConfig{
        MaxTokens: 8192,
    })

    // Custom task with zero budget
    customTask := &mockTask{
        name:      "custom",
        maxTokens: 0,  // Zero means no override
    }

    req := &AgentRequest{
        Input:   "test",
        Task:    customTask,
        WorkDir: t.TempDir(),
    }

    mockLLM := &mockLLMProvider{
        captureRequest: true,
    }
    agent.llm = mockLLM

    _, _ = agent.Execute(context.Background(), req)

    // Verify agent config was used
    assert.Equal(t, 8192, mockLLM.lastRequest.MaxTokens)
}
```

#### Test 3: Compact Mode Uses 4K Budget

```go
func TestAgent_CompactModeUses4KBudget(t *testing.T) {
    agent := newTestAgentWithConfig(t, &AgentConfig{
        MaxTokens: 16384,
    })

    // Compact mode has 4K tokens
    task := task.NewCompact()
    assert.Equal(t, 4096, task.MaxTokens())

    req := &AgentRequest{
        Input:   "Quick question",
        Task:    task,
        WorkDir: t.TempDir(),
    }

    mockLLM := &mockLLMProvider{
        captureRequest: true,
    }
    agent.llm = mockLLM

    _, _ = agent.Execute(context.Background(), req)

    // Verify compact budget was used (4K, not 16K)
    assert.Equal(t, 4096, mockLLM.lastRequest.MaxTokens)
}
```

#### Test 4: All Task Modes Apply Correct Budgets

```go
func TestAgent_AllTaskModesApplyCorrectBudgets(t *testing.T) {
    tests := []struct {
        name           string
        task           Task
        agentMaxTokens int
        wantMaxTokens  int
    }{
        {
            name:           "regular mode",
            task:           task.NewRegular(),
            agentMaxTokens: 4096,
            wantMaxTokens:  16384,  // task overrides
        },
        {
            name:           "review mode",
            task:           task.NewReview(),
            agentMaxTokens: 4096,
            wantMaxTokens:  12288,  // task overrides
        },
        {
            name:           "compact mode",
            task:           task.NewCompact(),
            agentMaxTokens: 16384,
            wantMaxTokens:  4096,   // task restricts
        },
        {
            name:           "planning mode",
            task:           task.NewPlanning(),
            agentMaxTokens: 16384,
            wantMaxTokens:  4096,   // task restricts
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := newTestAgentWithConfig(t, &AgentConfig{
                MaxTokens: tt.agentMaxTokens,
            })

            req := &AgentRequest{
                Input:   "test",
                Task:    tt.task,
                WorkDir: t.TempDir(),
            }

            mockLLM := &mockLLMProvider{
                captureRequest: true,
            }
            agent.llm = mockLLM

            _, _ = agent.Execute(context.Background(), req)

            assert.Equal(t, tt.wantMaxTokens, mockLLM.lastRequest.MaxTokens,
                "task=%s agent=%d want=%d",
                tt.task.Name(), tt.agentMaxTokens, tt.wantMaxTokens)
        })
    }
}
```

### Test Coverage Target

- **Overall**: ≥90% for new code
- **Critical Path**: 100% for token budget resolution logic

## Implementation Plan

### Step 1: Update callLLM() Method

**File**: `internal/core/agent.go:731-737`

1. Add token budget resolution logic before building LLM request
2. Use resolved budget in `llm.CompletionRequest`
3. Add comment documenting precedence

### Step 2: Add Unit Tests

**File**: `internal/core/agent_test.go`

1. Implement mock LLM that captures requests
2. Write 4 test cases covering all scenarios
3. Verify correct token budgets applied

### Step 3: Update Documentation

**File**: `internal/core/agent.go`

Add godoc comment on `callLLM()`:

```go
// callLLM sends messages to the LLM and returns the response.
// The task parameter controls both tool filtering and token budget:
//   - Tools: Only tools in task.AllowedTools() are included
//   - Tokens: Uses task.MaxTokens() if > 0, otherwise agent.config.MaxTokens
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
```

## Rollout Plan

1. **Implement**: Add token budget logic to `callLLM()`
2. **Test**: Run all unit tests, verify coverage ≥90%
3. **Lint**: Run `make lint`, ensure clean
4. **Race Check**: Run with `-race` flag
5. **Integration**: Verify existing tests still pass
6. **Document**: Update godoc and inline comments

## Acceptance Criteria

### Functional

- [x] Regular mode uses 16K tokens (overrides agent config)
- [x] Review mode uses 12K tokens (overrides agent config)
- [x] Compact mode uses 4K tokens (restricts agent config)
- [x] Planning mode uses 4K tokens (restricts agent config)
- [x] Custom tasks with zero budget fall back to agent config
- [x] Nil task falls back to agent config

### Quality

- [x] Unit tests cover all cases
- [x] Test coverage ≥90% for new code
- [x] `make lint` passes
- [x] Race detector clean
- [x] Godoc updated
- [x] Inline comments clear

### Performance

- [x] No measurable overhead (simple integer assignment)
- [x] No additional memory allocation

## Risks and Mitigations

### Risk: Token Budget Exceeds Provider Limit

**Probability**: Low
**Impact**: Medium
**Mitigation**: Provider will enforce its own limits

### Risk: Breaking Change

**Probability**: Very Low
**Impact**: High
**Mitigation**: All changes are additive; default behavior unchanged

### Risk: Test Mocking Complexity

**Probability**: Medium
**Impact**: Low
**Mitigation**: Use simple mock that captures request

## Future Enhancements

1. **Token Budget Validation**: Warn if task budget exceeds provider limit
2. **Dynamic Budgets**: Adjust budget based on context size
3. **Token Usage Tracking**: Track actual vs budgeted tokens per mode
4. **Budget Recommendations**: Suggest optimal budget for task type

## Dependencies

### Depends On

- [x] P1.1: Task Registry to Agent (Complete)
- [x] P1.2: Task Resolution Logic (Complete)
- [x] P1.3: Tool Filtering (Complete)

### Blocks

- [ ] P1.5: Integration Tests for Core Agent
- [ ] P2.1: Add Task Mode to Conversation

## References

- [ROADMAP.md](../task-modes/ROADMAP.md) - P1.4
- [specification.md](../task-modes/specification.md) - Section 1.4
- [task/regular.go](../../internal/core/task/regular.go) - Regular mode (16K)
- [task/compact.go](../../internal/core/task/compact.go) - Compact mode (4K)
- [agent.go:731-737](../../internal/core/agent.go#L731-L737) - Current implementation

## Approval

- **Stakeholders**: Core Team
- **Estimated Effort**: 1 hour
- **Risk Level**: Low
- **Implementation Ready**: Yes

---

**Last Updated**: 2025-10-12
**Status**: Ready for Implementation
