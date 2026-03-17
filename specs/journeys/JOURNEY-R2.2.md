# Journey R-2.2: Command Preparation Stage

**Roadmap Item**: R-2.2
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 1
**Status**: In Progress

## Context

Package managers and Python commands often block on interactive prompts or buffer output, causing the agent to hang indefinitely. A preparation stage rewrites known commands to add auto-confirm flags and sets environment variables to unbuffer output.

## User Journey

### Persona
Developer using Spin to install dependencies or run Python scripts.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Install | `npm install` | Hangs on prompt | Auto-appends `--yes` |
| Pip | `pip install flask` | Hangs on prompt | Auto-appends `--no-input` |
| Apt | `apt install curl` | Hangs on prompt | Auto-appends `-y` |
| Python | `python script.py` | Output buffered, appears late | `PYTHONUNBUFFERED=1` set |
| Pytest | `pytest tests/` | Output buffered | `PYTHONUNBUFFERED=1` set |
| Passthrough | `ls -la` | Unchanged | Unchanged |

### Success Criteria
- `npm install` gets `--yes` appended.
- `pip install` gets `--no-input` appended.
- `apt install` / `apt-get install` get `-y` appended.
- `python` / `python3` / `pytest` commands get `PYTHONUNBUFFERED=1` in env.
- Commands that already have the flag are not double-flagged.
- Unknown commands pass through unchanged.
- Non-install subcommands (e.g., `npm run dev`) are not rewritten.

## Technical Design

### Package Location
`internal/agent/executor/stage_prepare.go`

### Function
```go
// NewPrepareStage creates a stage that rewrites commands for non-interactive execution.
func NewPrepareStage() Stage
```

### Rewrite Rules
| Program | Condition | Action |
|---------|-----------|--------|
| `npm` | args[0] == "install" | Append `--yes` |
| `pip` / `pip3` | args[0] == "install" | Append `--no-input` |
| `apt` / `apt-get` | args[0] == "install" | Append `-y` |
| `python` / `python3` / `pytest` | always | Set env `PYTHONUNBUFFERED=1` |

### Integration
Added to `buildPipeline()` between validation and approval stages.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestPrepareStage_NpmInstall` | "npm not rewritten" | Appends --yes |
| `TestPrepareStage_PipInstall` | "pip not rewritten" | Appends --no-input |
| `TestPrepareStage_AptInstall` | "apt not rewritten" | Appends -y |
| `TestPrepareStage_PythonUnbuffered` | "env not set" | Sets PYTHONUNBUFFERED=1 |
| `TestPrepareStage_PytestUnbuffered` | "pytest missed" | Sets PYTHONUNBUFFERED=1 |
| `TestPrepareStage_UnknownUnchanged` | "rewrites everything" | ls passes through |
| `TestPrepareStage_NpmRunNotRewritten` | "non-install rewritten" | npm run dev unchanged |
| `TestPrepareStage_AlreadyHasFlag` | "double flag" | --yes not added twice |

## Implementation

**Status**: Complete

### Files Created
- `internal/agent/executor/stage_prepare.go` — `NewPrepareStage()` with auto-confirm and Python unbuffer rules.
- `internal/agent/executor/stage_prepare_test.go` — 11 tests covering all rewrite patterns and edge cases.

### Files Modified
- `internal/agent/executor.go` — `buildPipeline()` includes `NewPrepareStage()` between validation and approval.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-2.2 marked Done.
