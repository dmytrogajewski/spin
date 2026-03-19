# JOURNEY-2.2: Prompt Composer

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 2.2 |
| Title | Wire Prompt Composer into Harness Executor |
| User Story | As a developer, the system prompt is assembled from modular, priority-ordered sections (identity, tool guidance, response style) instead of an empty string. |
| Paper Section | 2.3.1 — conditional prompt composition pipeline |
| Roadmap Item | JOURNEY-2.2: Prompt Composer (10 functions) |

## Phases

### Phase 1: Discovery
- `prompt.Composer` fully implemented with section registration, variable injection, priority sorting
- `prompt.DefaultRegularSections()` returns 4 standard sections
- `prompt.ProjectInstructionsSection()` creates conditional project instructions
- Full unit tests exist
- **Friction**: Never called — `scaffold.Spec.SystemPrompt` is empty string

### Phase 2: Integration
- Pass `env *agent.Environment` to `buildHarnessExecutor()`
- Create `NewComposer()`, load `DefaultRegularSections()`, call `Compose(env)`
- Set result as `scaffold.Spec.SystemPrompt`

## Implementation

### Files Modified
- `internal/conversation/builder.go` — Create `NewComposer()`, load `DefaultRegularSections()`, add `ProjectInstructionsSection` from AGENTS.md, `SetVar()` for env vars, `Compose()` to produce `SystemPrompt`. Added `resolveAgentsMDContent()` helper.
- `internal/agent/prompt/composer.go` — Refactored `Compose()` to delegate to `ComposeTwoPart()` (DRY, both methods now reachable)
- `internal/agent/prompt/sections_test.go` — Updated test to verify cacheable-first ordering instead of legacy ordering
- `internal/git/patch.go` — Use `PatchError` in `ApplyPatch()` error path (was using generic `fmt.Errorf`)
