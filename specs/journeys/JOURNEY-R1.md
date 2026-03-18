# JOURNEY-R1: P0 Regression Safety Net (Completion)

## Overview

Complete the P0 fixture E2E test safety net by adding the three remaining test cases:
SA-5 (write allowed with --auto-approve), TE-1 (read nonexistent file error),
TE-2 (shell command failure).

## Phases

### Phase 1: Write Approval (SA-5)
- **Actor:** Developer runs `spin exec "write file" --auto-approve`
- **Fixture:** LLM calls write_file → final response
- **Expected:** File created on disk, block visible in output
- **Friction:** None — follows existing SA-1 pattern inverted

### Phase 2: Read Error Propagation (TE-1)
- **Actor:** LLM calls read_file on nonexistent path
- **Fixture:** read_file "missing.txt" → LLM responds about the error
- **Expected:** Error message visible in output, LLM still gives final response
- **Friction:** Error text format depends on OS — use broad match

### Phase 3: Shell Failure Propagation (TE-2)
- **Actor:** LLM calls shell_command that exits non-zero
- **Fixture:** shell_command "exit 1" → LLM responds about the failure
- **Expected:** Error/exit-code text visible, final response visible
- **Friction:** None — follows existing shell_command_output pattern

## Tests

- `TestFixture_WriteAllowedWithAutoApprove` — SA-5
- `TestFixture_ReadNonexistentFile` — TE-1
- `TestFixture_ShellCommandFailure` — TE-2

## Implementation

**Files created:**
- `tests/e2e/fixtures/write_with_approve.jsonl` — SA-5 fixture (shell_command → file creation)
- `tests/e2e/fixtures/read_nonexistent.jsonl` — TE-1 fixture (read_file on missing path)
- `tests/e2e/fixtures/shell_command_fail.jsonl` — TE-2 fixture (shell_command "exit 1")

**Files modified:**
- `tests/e2e/fixture_exec_test.go` — 3 new test functions: `TestFixture_WriteAllowedWithAutoApprove`, `TestFixture_ReadNonexistentFile`, `TestFixture_ShellCommandFailure`

**Roadmap:** [specs/testing/ROADMAP.md](../testing/ROADMAP.md) — R1 checklist complete

**Finding:** `write_file` tool has `FileTracker.AssertFresh()` check that blocks writes
to files never read. SA-5 uses `shell_command` instead to create files, which tests
the approval gate without hitting the tracker constraint.
