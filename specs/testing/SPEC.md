# Fixture E2E Test Specification

## Overview

This document specifies all fixture-based E2E test cases for `spin exec`.
Tests run at the **cobra command level** (subprocess) using JSONL fixture files
to drive deterministic LLM responses.

**Test runner:** `go test -tags e2e_llm_test ./tests/e2e/ -run TestFixture -v`

---

## How to Add a Test

1. Create a fixture file in `tests/e2e/fixtures/<name>.jsonl`
   — one JSONL line per LLM response (consumed sequentially)
2. Add a test function in `tests/e2e/fixture_exec_test.go`
3. Use helpers: `runFixtureExec`, `assertOutputContains`, `setupFixtureWorkDir`

**Fixture format** — each line is one `Stream()` call:
```json
{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"text","tool_calls":[{"index":0,"id":"tc-1","type":"function","function":{"name":"tool_name","arguments":"{...}"}}]},"finish_reason":"tool_calls|stop"}]}]}
```

**Test template:**
```go
func TestFixture_MyCase(t *testing.T) {
    t.Parallel()
    if testing.Short() { t.Skip("Skipping E2E test in short mode") }

    workDir := setupFixtureWorkDir(t, map[string]string{"file.txt": "content"})
    r := runFixtureExec(t, "my_fixture.jsonl", "prompt", withWorkDir(workDir), withAutoApprove())
    assertNoError(t, r)
    assertOutputContains(t, r, "expected text")
}
```

---

## Test Categories

### 1. Tool Visibility (Regression)

These tests verify that tool call blocks appear in terminal output.
Catches: missing `EventToolCallStart`/`EventToolCallComplete` emission.

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| TV-1 | `TestFixture_ToolCallBlockVisible` | `read_file_tool_visible.jsonl` | `READ` block appears for read_file |
| TV-2 | `TestFixture_ShellCommandOutput` | `shell_command_output.jsonl` | shell_command output visible |
| TV-3 | `TestFixture_WriteFileBlockVisible` | `write_file_block.jsonl` | write_file shows APPLY_PATCH block |
| TV-4 | `TestFixture_EditFileBlockVisible` | `edit_file_block.jsonl` | edit_file shows APPLY_PATCH block |
| TV-5 | `TestFixture_ListDirectoryBlockVisible` | `list_directory_block.jsonl` | list_directory shows EXECUTE block |
| TV-6 | `TestFixture_FileSearchBlockVisible` | `file_search_block.jsonl` | file_search shows GREP block |
| TV-7 | `TestFixture_GitContextBlockVisible` | `git_context_block.jsonl` | git_context shows NOTICE block |

### 2. Observation Summarizer (Regression)

These tests verify that the observation summarizer does not destroy tool
results before the LLM has seen them. Catches: `phaseObservation`
summarizing the current turn's results instead of only older ones.

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| OB-1 | `TestFixture_MultiToolObservation` | `multi_tool_observation.jsonl` | 3-turn: list→read→response. Final response references file content |
| OB-2 | `TestFixture_ReadThenShell` | `read_then_shell.jsonl` | 3-turn: read_file→shell_command→response. Both results consumed |
| OB-3 | `TestFixture_FiveToolTurns` | `five_tool_turns.jsonl` | 6 turns: 5 tool calls + final response. Validates observation only summarizes seen results |

### 3. Simple Responses

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| SR-1 | `TestFixture_SimpleResponse` | `simple_response.jsonl` | Plain text response without tools |
| SR-2 | `TestFixture_MultiLineResponse` | `multiline_response.jsonl` | Multi-line text preserved |
| SR-3 | `TestFixture_EmptyResponse` | `empty_response.jsonl` | Empty assistant message handled gracefully |

### 4. Security & Approval

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| SA-1 | `TestFixture_WriteDeniedWithoutAutoApprove` | `write_denied_no_approve.jsonl` | write_file blocked, file not created on disk |
| SA-2 | `TestFixture_ShellDeniedWithoutAutoApprove` | `shell_denied_no_approve.jsonl` | shell_command blocked without --auto-approve |
| SA-3 | `TestFixture_ReadAllowedWithoutAutoApprove` | `read_allowed_no_approve.jsonl` | read_file works without --auto-approve (read is safe) |
| SA-4 | `TestFixture_ListDirAllowedWithoutAutoApprove` | `list_dir_no_approve.jsonl` | list_directory works without --auto-approve |
| SA-5 | `TestFixture_WriteAllowedWithAutoApprove` | `write_with_approve.jsonl` | write_file succeeds with --auto-approve, file created |
| SA-6 | `TestFixture_EditDeniedWithoutAutoApprove` | `edit_denied_no_approve.jsonl` | edit_file blocked without --auto-approve |

### 5. Tool Error Handling

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| TE-1 | `TestFixture_ReadNonexistentFile` | `read_nonexistent.jsonl` | read_file error visible, LLM sees error and responds |
| TE-2 | `TestFixture_ShellCommandFailure` | `shell_command_fail.jsonl` | Non-zero exit code visible in output |
| TE-3 | `TestFixture_WriteToReadonlyPath` | `write_readonly_path.jsonl` | write_file permission error visible |
| TE-4 | `TestFixture_ToolNotFound` | `unknown_tool.jsonl` | Unknown tool name returns error, LLM recovers |

### 6. Multi-Turn Conversations

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| MT-1 | `TestFixture_ReadThenWrite` | `read_then_write.jsonl` | read_file then write_file in sequence |
| MT-2 | `TestFixture_ThreeToolCalls` | `three_tool_calls.jsonl` | 3 sequential tool calls, all blocks visible |
| MT-3 | `TestFixture_ToolThenTextThenTool` | `tool_text_tool.jsonl` | Tool→text response→tool→final (verifies loop continues correctly) |

### 7. Command-Line Flags

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| CF-1 | `TestFixture_TimeoutExpired` | `slow_response.jsonl` | `--timeout 1s` causes timeout (fixture has delay) |
| CF-2 | `TestFixture_ExitOnErrorTrue` | `tool_error_exit.jsonl` | `--exit-on-error` returns non-zero exit code |
| CF-3 | `TestFixture_ExitOnErrorFalse` | `tool_error_continue.jsonl` | Without --exit-on-error=false, error is printed but exit 0 |
| CF-4 | `TestFixture_StdinPrompt` | `simple_response.jsonl` | Prompt from stdin pipe works |

### 8. Edge Cases

| ID | Test | Fixture | What it verifies |
|----|------|---------|------------------|
| EC-1 | `TestFixture_EmptyToolArguments` | `empty_tool_args.jsonl` | Tool call with `{}` arguments handled |
| EC-2 | `TestFixture_LargeFileRead` | `read_large_file.jsonl` | Large file content (>100 chars) not truncated on current turn |
| EC-3 | `TestFixture_SpecialCharsInOutput` | `special_chars.jsonl` | Unicode, quotes, newlines in tool output preserved |
| EC-4 | `TestFixture_ConcurrentToolCalls` | N/A | If LLM returns multiple tool_calls in one response (parallel execution) |

---

## Priority

**P0 — Must have (regression prevention):**
- TV-1, TV-2 (tool visibility)
- OB-1 (observation summarizer)
- SA-1, SA-5 (write denied/allowed)
- SR-1 (basic response)
- TE-1 (tool error handling)

**P1 — Should have (comprehensive coverage):**
- TV-3 through TV-6 (all tool block types)
- OB-2, OB-3 (more observation scenarios)
- SA-2 through SA-4, SA-6 (all approval permutations)
- TE-2 through TE-4 (error scenarios)
- MT-1 through MT-3 (multi-turn)

**P2 — Nice to have (edge cases):**
- SR-2, SR-3 (response variants)
- CF-1 through CF-4 (flag testing)
- EC-1 through EC-4 (edge cases)
- TV-7 (git context)

---

## Existing Coverage (Already Implemented)

| Test | Status |
|------|--------|
| `TestFixture_SimpleResponse` (SR-1) | Done |
| `TestFixture_ToolCallBlockVisible` (TV-1) | Done |
| `TestFixture_MultiToolObservation` (OB-1) | Done |
| `TestFixture_ShellCommandOutput` (TV-2) | Done |
| `TestFixture_WriteDeniedWithoutAutoApprove` (SA-1) | Done |

---

## Test Infrastructure

| File | Purpose |
|------|---------|
| `internal/llm/testprovider/fixture_provider_testonly.go` | JSONL fixture-based LLM provider |
| `cmd/spin/exec_extra_provider_testonly.go` | Routes `SPIN_TEST_FIXTURE` to fixture provider |
| `tests/e2e/fixture_helpers_test.go` | `runFixtureExec`, `stripANSI`, assertions |
| `tests/e2e/fixture_exec_test.go` | All fixture test functions |
| `tests/e2e/fixtures/*.jsonl` | Fixture files |

**Build tag:** `e2e_llm_test` (test code excluded from production binary)

**Environment variable:** `SPIN_TEST_FIXTURE` — absolute path to fixture file,
set automatically by `runFixtureExec` helper.
