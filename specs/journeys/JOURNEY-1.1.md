# JOURNEY-1.1: Execution Pipeline

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 1.1 |
| Title | Wire Execution Pipeline into Command Adapter |
| User Story | As a developer running shell commands through spin, every command passes through staged safety/detection/preparation stages before execution, so dangerous commands are caught by defense-in-depth even if the tool-level classifier misses them. |
| Paper Section | 2.2.6 — staged executor pipeline |
| Roadmap Item | JOURNEY-1.1: Execution Pipeline (4 functions) |

## Phases

### Phase 1: Discovery (current state)
- `Pipeline`, `PipelineContext`, `Stage`, `NewSafetyStage` exist in `internal/agent/executor/`
- Full unit tests exist in `pipeline_test.go` and `stage_safety_test.go`
- The `Adapter` in `adapters.go` wraps `CommandExecutor` directly — no pipeline
- **Friction**: Pipeline is fully implemented and tested but never called from production code

### Phase 2: Integration
- Modify `Adapter` to accept an optional `Pipeline` and run it before executing commands
- `SafetyStage` uses the existing `safety.Validator` already available in `BuiltinRuntime`
- Wire in `RegisterTools()` where the `Adapter` is created

### Phase 3: Verification
- Existing pipeline unit tests continue to pass
- New integration test verifies `Adapter` runs pipeline before execution
- `make lint` confirms 4 deadcode functions are now reachable
- Forbidden commands are halted by SafetyStage before reaching the executor

## Friction Points

| Friction | Severity | Mitigation |
|----------|----------|------------|
| Adapter is a simple struct with no config | Low | Add pipeline field, nil-check for backward compat |
| SafetyStage needs `*safety.Validator` | Low | Already available as `r.validator` in BuiltinRuntime |

## Test Plan

- `TestAdapter_RunsPipelineBeforeExecution` — verify pipeline stages execute before command
- `TestAdapter_HaltedPipelineSkipsExecution` — verify halted pipeline prevents command execution
- `TestAdapter_NilPipelineExecutesDirectly` — backward compat: nil pipeline = direct execution
- Existing `pipeline_test.go` and `stage_safety_test.go` continue to pass

## Implementation

### Files Modified
- `internal/agent/executor/adapters.go` — Added `pipeline *Pipeline` field to `Adapter`, `NewAdapterWithPipeline()` constructor, pipeline execution in `Execute()`
- `internal/agent/executor/builtin.go` — Changed `RegisterTools()` to create `Pipeline` with `SafetyStage` and pass to `NewAdapterWithPipeline()`

### Files Created
- `internal/agent/executor/adapter_pipeline_test.go` — 3 tests: runs pipeline, halted skips exec, nil pipeline compat

### Legacy Removed
- Direct `&Adapter{executor: r.executor}` construction bypassing pipeline stages
