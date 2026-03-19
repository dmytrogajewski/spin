# JOURNEY-1.3: Lifecycle Hooks

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 1.3 |
| Title | Wire Lifecycle Hooks into Tool Runtime and Conversation |
| User Story | As a developer with custom `.spin/hooks/` scripts, hooks fire at PRE_TOOL_USE, POST_TOOL_USE, SESSION_START, and USER_PROMPT_SUBMIT — and can block tool execution with exit code 2. |
| Paper Section | 2.1, Layer 5 — lifecycle hooks, JSON stdin protocol |
| Roadmap Item | JOURNEY-1.3: Lifecycle Hooks (10 functions) |

## Phases

### Phase 1: Discovery
- `hooks.Runner` fully implemented with discovery, blocking/async execution, JSON stdin protocol
- `hooks.Event` defines 10 lifecycle events with blocking semantics
- Full unit tests exist in `hooks/` package
- **Friction**: Runner is never created or called from production code

### Phase 2: Integration — Tool Runtime
- Add optional `HookRunner *hooks.Runner` field to `tool.RuntimeConfig` and `tool.Runtime`
- In `Execute()`, call PRE_TOOL_USE before tool execution; if blocked, return error
- In `Execute()`, call POST_TOOL_USE after tool execution (non-blocking)

### Phase 3: Integration — Conversation Layer
- Create `hooks.NewRunner()` in `conversation/builder.go::Build()`
- Pass to tool runtime via `RuntimeConfig`
- Add `hookRunner` field to `Conversation`
- Call SESSION_START in `Build()` after runner creation
- Call USER_PROMPT_SUBMIT in `RunTurn()` before harness execution

### Phase 4: Verification
- New tests for hook integration in tool runtime
- Existing hooks unit tests continue to pass
- `make lint` confirms 10 deadcode functions reachable

## Test Plan

- `TestRuntime_CallsPreToolHook` — verify PRE_TOOL_USE fires before tool execution
- `TestRuntime_BlockedHookPreventsExecution` — verify blocked hook returns error
- `TestRuntime_NilHookRunnerNoOp` — nil runner = no-op (backward compat)
- Existing `hooks/` package tests continue to pass

## Implementation

### Files Modified
- `internal/agent/tool/runtime.go` — Added `HookRunner` to `RuntimeConfig`/`Runtime`, `runPreToolHook()` (PRE_TOOL_USE with blocking), `runPostToolHook()` (POST_TOOL_USE non-blocking)
- `internal/conversation/agent.go` — Create `hooks.NewRunner()`, pass to `RuntimeConfig.HookRunner`, add `hookRunner` to `agentBuildResult`
- `internal/conversation/builder.go` — Fire `SESSION_START` hook in `Build()`, pass `hookRunner` to `Conversation`
- `internal/conversation/conversation.go` — Added `hookRunner` field, fire `USER_PROMPT_SUBMIT` hook in `RunTurn()` with blocking support
- `internal/safety/hooks/runner.go` — Use `AllEvents()` in constructor to pre-build valid script name set

### Files Created
- `internal/agent/tool/runtime_hooks_test.go` — 3 tests: nil runner no-op, pre-tool hook fires, blocked hook prevents execution
