# JOURNEY-1.2: Blocklist Checker

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 1.2 |
| Title | Wire Blocklist Checker as Pipeline Stage |
| User Story | As a developer, dangerous command patterns (rm -rf /, fork bombs, dd to device) are blocked by a dedicated blocklist layer independent of the command classifier, providing defense-in-depth at Layer 4. |
| Paper Section | 2.1, Layer 4 — DANGEROUS_PATTERNS blocklist |
| Roadmap Item | JOURNEY-1.2: Blocklist Checker (4 functions) |

## Phases

### Phase 1: Discovery
- `blocklist.Checker` exists with `NewChecker()`, `Check()`, `Enabled()`, `defaultRules()` — fully implemented
- Full unit tests exist in `blocklist_test.go` (249 lines, 15+ test cases)
- **Friction**: Never called from production code

### Phase 2: Integration
- Create `stage_blocklist.go` — a Pipeline `Stage` that delegates to `blocklist.Checker`
- Add the blocklist stage to the Pipeline in `builtin.go::RegisterTools()` alongside `SafetyStage`
- The stage runs on the raw command string (`pc.Command.Raw`)

### Phase 3: Verification
- New stage test in `stage_blocklist_test.go`
- Existing blocklist unit tests continue to pass
- `make lint` confirms 4 deadcode functions reachable

## Test Plan

- `TestBlocklistStage_BlocksForbiddenCommand` — verify stage halts pipeline on dangerous command
- `TestBlocklistStage_AllowsSafeCommand` — verify stage passes through safe commands
- `TestBlocklistStage_NilCheckerNoOp` — nil checker = no-op stage
- Existing `blocklist_test.go` continues to pass

## Implementation

### Files Created
- `internal/agent/executor/stage_blocklist.go` — `NewBlocklistStage()` wrapping `blocklist.Checker` as Pipeline `Stage`
- `internal/agent/executor/stage_blocklist_test.go` — 3 tests: blocks forbidden, allows safe, nil no-op

### Files Modified
- `internal/agent/executor/builtin.go` — Added `NewBlocklistStage(blocklist.NewChecker(true))` as first pipeline stage before `SafetyStage`
