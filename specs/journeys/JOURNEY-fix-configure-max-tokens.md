# JOURNEY-fix-configure-max-tokens: Fix Dead Code in configureMaxTokens

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 1.6 — Fix Dead Code in `configureMaxTokens`
- Cluster: 19 (SPEC.md) | LIST.md finding 78

## 1. Journey

When **a developer reading the TUI startup code** I want to **see only meaningful logic in `configureMaxTokens`** so I can **understand the intent without being confused by a loop that finds a model but never uses it**.

## 2. CJM

`cmd/spin/tui.go:configureMaxTokens` iterated over `provider.Models()` looking for the current model, but the loop body only called `break` — it never used the found model. The function always set `maxTokens` to `defaultMaxTokens` regardless.

The fix removes the dead iteration and simplifies the function to its actual behavior: setting max tokens to the default value. The function signature was simplified to remove unused parameters (`ctx`, `provider`, `currentModel`).

### North Star Summary

`configureMaxTokens` is reduced to a single-line function that sets `ui.SetMaxTokens(defaultMaxTokens)`. Dead iteration and unused parameters removed.

## 3. Tests

### TC-01: existing cmd tests pass

**Given** the dead loop is removed.
**When** `go test ./cmd/spin/...` is run.
**Then** all tests pass.

## Implementation

- Modified: `cmd/spin/tui.go` — removed dead iteration, simplified function signature
