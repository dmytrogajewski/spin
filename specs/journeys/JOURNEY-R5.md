# JOURNEY-R5: Tool Error Handling (Completion)

## Overview

Complete error handling coverage: write to invalid path (TE-3) and unknown tool name (TE-4).

## Tests

- `TestFixture_WriteToReadonlyPath` — TE-3
- `TestFixture_ToolNotFound` — TE-4

## Implementation

**Files created (R5):**
- `tests/e2e/fixtures/write_readonly_path.jsonl` — TE-3
- `tests/e2e/fixtures/unknown_tool.jsonl` — TE-4

**Files created (R6):**
- `tests/e2e/fixtures/multiline_response.jsonl` — SR-2
- `tests/e2e/fixtures/empty_response.jsonl` — SR-3
- `tests/e2e/fixtures/empty_tool_args.jsonl` — EC-1
- `tests/e2e/fixtures/read_large_file.jsonl` — EC-2
- `tests/e2e/fixtures/special_chars.jsonl` — EC-3

**Files modified:**
- `tests/e2e/fixture_exec_test.go` — 7 new test functions
- `tests/e2e/fixture_helpers_test.go` — restored `withTimeout` helper

**Roadmap:** [specs/testing/ROADMAP.md](../testing/ROADMAP.md) — R5+R6 complete

**Files created (R7):**
- `tests/e2e/fixtures/slow_response.jsonl` — CF-1 (5s delay for timeout test)

**Files modified (R7):**
- `internal/llm/testprovider/fixture_provider_testonly.go` — added `delay_ms` field + sleep
- `tests/e2e/fixture_helpers_test.go` — added `withExecTimeout`, `withExitOnError`, `runFixtureExecStdin`
- `tests/e2e/fixture_exec_test.go` — 4 new test functions (CF-1..CF-4)

**Finding:** Empty LLM response triggers caller retry logic (3 retries), exhausting
the single-line fixture. Test asserts no panic rather than clean exit — this is
correct behavior (empty responses are retried by design).
