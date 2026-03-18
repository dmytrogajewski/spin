# JOURNEY-R4: Multi-Turn & Observation Stress

## Overview

Stress-test the observation summarizer boundary logic with 3-6 turn conversations
using mixed tool types. Validates that `preDispatchLen` correctly preserves current
turn results while summarizing older ones.

## Tests

- `TestFixture_ReadThenShell` — OB-2: read_file → shell_command → response
- `TestFixture_FiveToolTurns` — OB-3: 5 sequential list_directory calls + final response
- `TestFixture_ReadThenWrite` — MT-1: read_file → write_file → response
- `TestFixture_ThreeToolCalls` — MT-2: list_directory → read_file → shell_command → response

## Implementation

**Files created:**
- `tests/e2e/fixtures/read_then_shell.jsonl` — OB-2 (3-turn mixed tools)
- `tests/e2e/fixtures/five_tool_turns.jsonl` — OB-3 (6-turn stress test)
- `tests/e2e/fixtures/read_then_write.jsonl` — MT-1 (read→write)
- `tests/e2e/fixtures/three_tool_calls.jsonl` — MT-2 (list→read→shell)

**Files modified:**
- `tests/e2e/fixture_exec_test.go` — 4 new test functions

**Roadmap:** [specs/testing/ROADMAP.md](../testing/ROADMAP.md) — R4 checklist complete
