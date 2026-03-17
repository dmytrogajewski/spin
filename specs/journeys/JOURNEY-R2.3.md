# Journey R-2.3: Server Detection Stage

**Roadmap Item**: R-2.3
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 1
**Status**: In Progress

## Context

Long-running server commands (dev servers, watchers) block agent execution until timeout. Detecting these commands early enables extending timeouts or promoting them to background execution (once R-3.1 is available).

## User Journey

### Persona
Developer using Spin to start dev servers or watchers.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Start | `npm run dev` | Blocks for 5 min, times out | Detected as server, flagged for background |
| Start | `flask run` | Same — hangs | Detected, timeout extended |
| Normal | `go test ./...` | Runs normally | Not detected, runs normally |

### Success Criteria
- 16+ server/watcher patterns compiled as regexps.
- Matching commands set `IsServer=true` on PipelineContext.
- Non-matching commands leave flags unchanged.
- Stage registered in pipeline after Prepare.
- All patterns compile without panic.

## Technical Design

### Package Location
- `internal/agent/executor/patterns.go` — compiled regexp patterns.
- `internal/agent/executor/stage_detect.go` — detection stage.

### PipelineContext Extension
Add `IsServer bool` field to `PipelineContext`.

### Patterns (16)
npm/yarn/pnpm dev|start|serve, next dev, vite, webpack serve, flask run,
uvicorn, gunicorn, rails server, php artisan serve, go run, cargo run,
air, nodemon, docker compose up, hugo server, jekyll serve.

### Integration
Added to `buildPipeline()` after Prepare, before Approval.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestDetectStage_NpmRunDev` | "npm dev missed" | npm run dev detected |
| `TestDetectStage_YarnStart` | "yarn missed" | yarn start detected |
| `TestDetectStage_FlaskRun` | "flask missed" | flask run detected |
| `TestDetectStage_UvicornMain` | "uvicorn missed" | uvicorn main:app detected |
| `TestDetectStage_GoRun` | "go run missed" | go run . detected |
| `TestDetectStage_DockerComposeUp` | "docker missed" | docker compose up detected |
| `TestDetectStage_GoTestNotDetected` | "false positive" | go test not a server |
| `TestDetectStage_LsNotDetected` | "false positive" | ls not a server |
| `TestDetectStage_AllPatternsCompile` | "regex panic" | All patterns compile |

## Implementation

**Status**: Complete

### Files Created
- `internal/agent/executor/patterns.go` — 16 compiled server/watcher regexp patterns.
- `internal/agent/executor/stage_detect.go` — `NewDetectStage()` sets `IsServer=true` on match.
- `internal/agent/executor/stage_detect_test.go` — 10 tests covering matches and non-matches.

### Files Modified
- `internal/agent/executor/pipeline.go` — added `IsServer` field to `PipelineContext`.
- `internal/agent/executor.go` — `buildPipeline()` includes `NewDetectStage()` after Prepare.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-2.3 marked Done.
