# JOURNEY-R3: Security & Approval Matrix

## Overview

Verify that every dangerous tool is blocked without `--auto-approve` and every
safe tool is allowed without it. Completes the approval boundary coverage started
by SA-1 (write denied) and SA-5 (write allowed).

## Phases

### Phase 1: shell_command denied (SA-2)
- Fixture: shell "echo secret" → response
- No `--auto-approve`
- Assert: "secret" NOT in output (command never ran)

### Phase 2: read_file allowed (SA-3)
- Fixture: read_file → response
- No `--auto-approve` (read is safe)
- Assert: file content visible in output

### Phase 3: list_directory allowed (SA-4)
- Fixture: list_directory → response
- No `--auto-approve` (list is safe)
- Assert: directory entries visible

### Phase 4: edit_file denied (SA-6)
- Fixture: read_file → edit_file → response
- No `--auto-approve`
- Assert: file unchanged on disk

## Tests

- `TestFixture_ShellDeniedWithoutAutoApprove` — SA-2
- `TestFixture_ReadAllowedWithoutAutoApprove` — SA-3
- `TestFixture_ListDirAllowedWithoutAutoApprove` — SA-4
- `TestFixture_EditDeniedWithoutAutoApprove` — SA-6

## Implementation

**Files created:**
- `tests/e2e/fixtures/shell_denied_no_approve.jsonl` — SA-2
- `tests/e2e/fixtures/read_allowed_no_approve.jsonl` — SA-3
- `tests/e2e/fixtures/list_dir_no_approve.jsonl` — SA-4
- `tests/e2e/fixtures/edit_denied_no_approve.jsonl` — SA-6

**Files modified:**
- `tests/e2e/fixture_exec_test.go` — 4 new test functions

**Roadmap:** [specs/testing/ROADMAP.md](../testing/ROADMAP.md) — R3 checklist complete
