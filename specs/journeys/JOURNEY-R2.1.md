# Journey R-2.1: Pipeline Framework & Safety Stage

**Roadmap Item**: R-2.1
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 1
**Status**: In Progress

## Context

The `agent.Executor.Execute()` method is monolithic — validation, approval, caching, execution, and output handling are interwoven. This makes it difficult to add new stages (command preparation, server detection, output truncation) without growing the method further. A pipeline architecture enables composable, testable stages.

## User Journey

### Persona
Internal developer extending Spin's command execution capabilities.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Add stage | Add new execution concern | Edit monolithic Execute method | Add a Stage function, register in pipeline |
| Test stage | Test a single concern | Must test through full Execute flow | Unit test single Stage in isolation |
| Halt flow | Stop execution early | Return error deep in call chain | Set `PipelineContext.Halted = true` |
| Compose | Combine multiple concerns | Nested if/else in Execute | Sequential stages with clear ordering |

### Friction Points (Current)
1. **Monolithic method**: Adding validation, approval, execution all in one function.
2. **Hard to test stages**: Must set up full Executor to test one concern.
3. **No short-circuit**: Early exit requires nested error handling.

### Success Criteria
- `Pipeline` struct runs stages sequentially, stopping on halt or error.
- `PipelineContext` carries command, options, result, and halt state.
- Safety stage wraps existing validation + approval logic.
- All existing `executor_test.go` tests continue to pass.
- `agent.Executor.Execute()` delegates to pipeline internally.

## Technical Design

### Package Location
`internal/agent/executor/pipeline.go` — pipeline framework.
`internal/agent/executor/stage_safety.go` — safety validation stage.

### Types
```go
// PipelineContext carries data through pipeline stages.
type PipelineContext struct {
    Ctx     context.Context
    Command *safety.Command
    WorkDir string
    Env     map[string]string
    Timeout time.Duration
    Halted  bool
    HaltErr error
    Result  *CommandResult
    Values  map[string]any
}

// Stage processes a PipelineContext.
type Stage func(pc *PipelineContext) error

// Pipeline runs stages in sequence.
type Pipeline struct {
    stages []Stage
}
```

### Integration
`agent.Executor.Execute()` constructs a `PipelineContext`, builds stages from its fields, creates a `Pipeline`, and calls `Run()`. The result is converted back to `agent.Result`.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestPipeline_RunsAllStages` | "stages skipped" | All stages execute in order |
| `TestPipeline_ShortCircuitsOnHalt` | "halt ignored" | Halted context skips remaining stages |
| `TestPipeline_StopsOnError` | "error swallowed" | Stage error stops pipeline |
| `TestPipeline_EmptyPipeline` | "nil panic on empty" | No stages returns nil |
| `TestSafetyStage_ForbiddenHalts` | "forbidden passes" | Forbidden command halts pipeline |
| `TestSafetyStage_SafePasses` | "safe blocked" | Safe command continues |
| `TestSafetyStage_NilValidator` | "nil panic" | No validator skips validation |

## Implementation

**Status**: Complete

### Files Created
- `internal/agent/executor/pipeline.go` — `PipelineContext`, `Stage`, `Pipeline` with `Run()`.
- `internal/agent/executor/stage_safety.go` — `NewSafetyStage()` wrapping `Validator.Classify()`.
- `internal/agent/executor/pipeline_test.go` — 6 pipeline framework tests.
- `internal/agent/executor/stage_safety_test.go` — 3 safety stage tests.

### Files Modified
- `internal/agent/executor.go` — `Execute()` delegates to pipeline via `buildPipeline()`, `validationStage()`, `approvalStage()`.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-2.1 marked Done.
