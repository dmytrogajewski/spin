# JOURNEY-3.1: Scaffold Factory + SubAgent System

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 3.1 |
| Title | Wire Scaffold Factory and SubAgent Manager |
| User Story | As a developer, agent specs are compiled by a Factory and subagents are registered for future spawning. |
| Paper Section | 2.2.1 (factory), 2.2.7 (subagents) |
| Roadmap Item | JOURNEY-3.1 (12 functions) |
| Depends on | JOURNEY-2.2 (Prompt Composer) |

## Phases

### Phase 1: Discovery
- `scaffold.Factory` compiles Specs from config+registry+providers
- `subagent.Manager` orchestrates subagent lifecycle with concurrency control
- `subagent.Builtins()` defines 4 builtin subagent specs (explorer, planner, reviewer, ask_user)
- All have full unit tests. Never wired into production.

### Phase 2: Integration
- Create Factory in `buildHarnessExecutor()`, use `Compile("main")` for base Spec
- Override SystemPrompt with Composer output (richer than Factory default)
- Create SubAgent Manager in builder, store on Conversation for future use
- Manager auto-registers Builtins in constructor

## Implementation

### Files Modified
- `internal/conversation/builder.go` — Replace manual `scaffold.Spec{}` with `scaffold.NewFactory()` + `Factory.Compile("main")`; create `subagent.NewManager()` with Builtins auto-registered; override SystemPrompt with Composer output
- `internal/conversation/conversation.go` — Added `subagentManager` field, `GetSubagentManager()` getter
