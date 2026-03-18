# Journey R-T1: Eliminate Shell Detection Duplication

**Roadmap Item**: R-T1
**Spec**: [SPEC.md](../refactoring/tools-cleanup/SPEC.md) Section F-2
**Status**: Done

## Context

`shell_command.go` contains an `isShellCmd()` method that detects shell
metacharacters (pipes, redirects, `$`, `&&`, `||`, builtins). The
`detectShell()` operation had an identical fallback check when `shellCtx`
is nil. The same 8 conditions appeared twice in the same file.

## User Journey

### Persona
Developer maintaining the tools package who needs to add a new shell
metacharacter (e.g., backtick expansion).

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Discover | Find shell detection logic | Two locations, 60 lines apart | One canonical location |
| Modify | Add new metacharacter | Must update both places | Update once in `isShellCmd()` |
| Verify | Run tests | Tests pass even if one copy missed | Tests exercise the single function |

### Friction Points (Resolved)
1. **Silent divergence risk**: eliminated — single source of truth.
2. **Code review burden**: eliminated — one location to verify.

## Implementation

### Change Summary
Replaced 9-line inline shell metacharacter check in `detectShell()` fallback
with a single call to the existing `t.isShellCmd(command)` method.

### Files Modified
| File | Change |
|------|--------|
| `internal/tools/shell_command.go` | Lines 429-438: replaced 9-line inline check with `t.isShellCmd(command)` |

## Tests

- All existing `shell_command_test.go` tests pass unchanged
- No new tests required (behavioral equivalence — same logic, same method)
- `go vet ./internal/tools/...` clean
- No new lint issues introduced
