# Fixture E2E Test Roadmap

**Spec:** [SPEC.md](SPEC.md)
**Progress:** 31 / 31 test cases implemented -- COMPLETE

---

## R1: P0 Regression Tests (Core Safety Net)

> **Journey:** Developer pushes code → CI catches regressions in tool visibility,
> observation summarizer, security enforcement, and error propagation.
>
> **Why first:** These 8 tests would have caught all three bugs we already fixed.
> Five are done; three remain to close the P0 safety net.

**DoR:**
- [x] Fixture provider compiles (`fixture_provider_testonly.go`)
- [x] Test helpers compile (`fixture_helpers_test.go`)
- [x] Existing 5 tests pass

**Checklist:**
- [x] SR-1 `TestFixture_SimpleResponse` — plain text, no tools
  - fixture: `simple_response.jsonl`
  - assert: "The answer is 42." in output
- [x] TV-1 `TestFixture_ToolCallBlockVisible` — read_file renders READ block
  - fixture: `read_file_tool_visible.jsonl`
  - assert: "READ" block in output, final response visible
- [x] TV-2 `TestFixture_ShellCommandOutput` — shell_command output visible
  - fixture: `shell_command_output.jsonl`
  - assert: "hello world" in output
- [x] OB-1 `TestFixture_MultiToolObservation` — observation summarizer preserves current-turn results
  - fixture: `multi_tool_observation.jsonl` (3-turn: list→read→response)
  - assert: EXECUTE block, READ block, "value 12345" in final response
- [x] SA-1 `TestFixture_WriteDeniedWithoutAutoApprove` — write_file blocked without flag
  - fixture: `write_denied_no_approve.jsonl`
  - assert: `secret.txt` does NOT exist on disk
- [x] SA-5 `TestFixture_WriteAllowedWithAutoApprove` — shell_command creates file with --auto-approve
  - fixture: `write_with_approve.jsonl` (shell_command "echo > output.txt" → response)
  - assert: file created on disk, final response visible
  - journey: [JOURNEY-R1](../journeys/JOURNEY-R1.md)
- [x] TE-1 `TestFixture_ReadNonexistentFile` — tool error visible, LLM recovers
  - fixture: `read_nonexistent.jsonl` (read_file "missing.txt" → response)
  - assert: READ block visible, LLM error-recovery response visible
  - journey: [JOURNEY-R1](../journeys/JOURNEY-R1.md)
- [x] TE-2 `TestFixture_ShellCommandFailure` — non-zero exit code visible
  - fixture: `shell_command_fail.jsonl` (shell "exit 1" → response)
  - assert: "exit" text in output, LLM response visible
  - journey: [JOURNEY-R1](../journeys/JOURNEY-R1.md)

**DoD:** COMPLETE
- [x] All 8 tests green: `go test -tags e2e_llm_test ./tests/e2e/ -run TestFixture -v`
- [x] Each test has its own `.jsonl` fixture file
- [x] SA-5 asserts file on disk; TE-1/TE-2 assert error text in output

---

## R2: Tool Block Visibility for All Tool Types

> **Journey:** Developer changes TUI mapper or adds a tool → CI verifies every
> tool type renders its expected block header (READ, EXECUTE, APPLY_PATCH, GREP).
>
> **Why second:** TV-1/TV-2 only cover read_file and shell_command. The mapper
> has distinct code paths per tool type — a broken path needs per-type coverage.

**DoR:**
- [x] R1 checklist complete

**Checklist:**
- [x] TV-3 `TestFixture_WriteFileBlockVisible` — write_file renders WRITE block
  - fixture: `write_file_block.jsonl` (read_file → write_file → response)
  - assert: WRITE block visible, file updated on disk
  - journey: [JOURNEY-R2](../journeys/JOURNEY-R2.md)
- [x] TV-4 `TestFixture_EditFileBlockVisible` — edit_file renders block
  - fixture: `edit_file_block.jsonl` (read_file → edit_file → response)
  - assert: block visible, file content changed from "foo" to "bar"
  - journey: [JOURNEY-R2](../journeys/JOURNEY-R2.md)
- [x] TV-5 `TestFixture_ListDirectoryBlockVisible` — list_directory renders EXECUTE block
  - fixture: `list_directory_block.jsonl`
  - assert: EXECUTE block, file name in listing
  - journey: [JOURNEY-R2](../journeys/JOURNEY-R2.md)
- [x] TV-6 `TestFixture_FileSearchBlockVisible` — file_search tool executes
  - fixture: `file_search_block.jsonl`
  - assert: tool executes, LLM response visible
  - journey: [JOURNEY-R2](../journeys/JOURNEY-R2.md)

**DoD:** COMPLETE
- [x] All 4 new tests green
- [x] Each test asserts the specific block keyword in ANSI-stripped output

---

## R3: Security & Approval Matrix

> **Journey:** Security auditor reviews exec mode → every dangerous tool is blocked
> without `--auto-approve`, every safe tool is allowed without it.
>
> **Why third:** SA-1/SA-5 cover write only. The full matrix covers shell, edit,
> read, and list — ensuring the approval boundary is correct in both directions.

**DoR:**
- [x] R1 checklist complete

**Checklist:**
- [x] SA-2 `TestFixture_ShellDeniedWithoutAutoApprove` — shell_command blocked
  - fixture: `shell_denied_no_approve.jsonl`
  - assert: "secret" NOT in output (command never ran)
  - journey: [JOURNEY-R3](../journeys/JOURNEY-R3.md)
- [x] SA-3 `TestFixture_ReadAllowedWithoutAutoApprove` — read_file allowed (safe)
  - fixture: `read_allowed_no_approve.jsonl`
  - assert: LLM response visible (file was read)
  - journey: [JOURNEY-R3](../journeys/JOURNEY-R3.md)
- [x] SA-4 `TestFixture_ListDirAllowedWithoutAutoApprove` — list_directory allowed (safe)
  - fixture: `list_dir_no_approve.jsonl`
  - assert: file name visible in output
  - journey: [JOURNEY-R3](../journeys/JOURNEY-R3.md)
- [x] SA-6 `TestFixture_EditDeniedWithoutAutoApprove` — edit_file blocked
  - fixture: `edit_denied_no_approve.jsonl`
  - assert: file unchanged on disk ("original")
  - journey: [JOURNEY-R3](../journeys/JOURNEY-R3.md)

**DoD:** COMPLETE
- [x] All 4 tests green
- [x] Denial tests: file system unchanged (no side effects)
- [x] Allow tests: tool output visible (block rendered, content present)

---

## R4: Multi-Turn & Observation Stress

> **Journey:** Agent runs a complex multi-step task → all tool results visible,
> context not corrupted by the observation summarizer across many turns.
>
> **Why fourth:** OB-1 covers 3 turns. These push to 5+ turns and mixed tool
> types to stress the `preDispatchLen` boundary logic in `phaseObservation`.

**DoR:**
- [x] R1, R2 complete

**Checklist:**
- [x] OB-2 `TestFixture_ReadThenShell` — mixed tool types across turns
  - fixture: `read_then_shell.jsonl` — journey: [JOURNEY-R4](../journeys/JOURNEY-R4.md)
- [x] OB-3 `TestFixture_FiveToolTurns` — 6-turn observation stress
  - fixture: `five_tool_turns.jsonl` — journey: [JOURNEY-R4](../journeys/JOURNEY-R4.md)
- [x] MT-1 `TestFixture_ReadThenWrite` — read-before-write pattern
  - fixture: `read_then_write.jsonl` — journey: [JOURNEY-R4](../journeys/JOURNEY-R4.md)
- [x] MT-2 `TestFixture_ThreeToolCalls` — 3 sequential calls, all visible
  - fixture: `three_tool_calls.jsonl` — journey: [JOURNEY-R4](../journeys/JOURNEY-R4.md)

**DoD:** COMPLETE
- [x] All 4 tests green
- [x] Each test asserts ALL tool blocks appear in output order
- [x] OB-3: final response references "marker99" from 5th tool call

---

## R5: Tool Error Handling

> **Journey:** LLM calls a tool that fails → error is visible in output,
> LLM sees the error message and responds gracefully.
>
> **Why fifth:** Error paths are common (missing files, failed commands,
> bad tool names). The LLM must see errors to recover.

**DoR:**
- [ ] R1 complete (TE-1, TE-2 done)

**Checklist:**
- [x] TE-3 `TestFixture_WriteToReadonlyPath` — write to invalid path, clean exit
  - fixture: `write_readonly_path.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] TE-4 `TestFixture_ToolNotFound` — unknown tool name, LLM recovers
  - fixture: `unknown_tool.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)

**DoD:** COMPLETE
- [x] All 4 error tests green (TE-1..TE-4)
- [x] Error messages visible in ANSI-stripped output
- [x] No panics, no hangs — clean process exit

---

## R6: Response Variants & Edge Cases

> **Journey:** LLM returns unusual responses → exec handles all gracefully
> without crashes, truncation, or data loss.

**DoR:**
- [ ] R1 complete

**Checklist:**
- [x] SR-2 `TestFixture_MultiLineResponse` — multi-line text preserved
  - fixture: `multiline_response.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] SR-3 `TestFixture_EmptyResponse` — empty content, no panic
  - fixture: `empty_response.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] EC-1 `TestFixture_EmptyToolArguments` — empty `{}` args, LLM recovers
  - fixture: `empty_tool_args.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] EC-2 `TestFixture_LargeFileRead` — 500-char file not truncated
  - fixture: `read_large_file.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] EC-3 `TestFixture_SpecialCharsInOutput` — unicode preserved
  - fixture: `special_chars.jsonl` — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)

**DoD:** COMPLETE
- [x] All 5 tests green
- [x] EC-2: LLM response proves content not truncated
- [x] SR-3: no panic (caller retries exhaust fixture, acceptable error)

---

## R7: Command-Line Flags

> **Journey:** CI pipeline uses `spin exec` with various flags → each flag
> behaves as documented.

**DoR:**
- [x] R1 complete
- [x] Fixture provider enhanced with `delay_ms` field

**Checklist:**
- [x] CF-1 `TestFixture_TimeoutExpired` — `--timeout 1s` causes timeout
  - fixture: `slow_response.jsonl` (5s delay) — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] CF-2 `TestFixture_ExitOnErrorTrue` — error returns non-zero exit
  - fixture: `empty_response.jsonl` (reuse) — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] CF-3 `TestFixture_ExitOnErrorFalse` — error printed but exit 0
  - fixture: `empty_response.jsonl` (reuse) — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)
- [x] CF-4 `TestFixture_StdinPrompt` — prompt piped via stdin
  - fixture: `simple_response.jsonl` (reuse) — journey: [JOURNEY-R5](../journeys/JOURNEY-R5.md)

**DoD:** COMPLETE
- [x] All 4 tests green
- [x] CF-1: `delay_ms` field added to fixture format, provider sleeps accordingly
- [x] CF-4: `runFixtureExecStdin` helper added to `fixture_helpers_test.go`

---

## Summary

| Round | New | Done | Total | Focus |
|-------|-----|------|-------|-------|
| R1 | 0 | 8 | 8 | P0 regression safety net -- DONE |
| R2 | 0 | 4 | 12 | Tool block type coverage -- DONE |
| R3 | 0 | 4 | 16 | Security approval boundary -- DONE |
| R4 | 0 | 4 | 20 | Multi-turn observation stress -- DONE |
| R5 | 0 | 2 | 22 | Tool error propagation -- DONE |
| R6 | 0 | 5 | 27 | Edge cases & response variants -- DONE |
| R7 | 0 | 4 | 31 | CLI flag behavior -- DONE |

**Deferred:** MT-3 (tool→text→tool loop), EC-4 (parallel tool calls), TV-7 (git_context) — require infrastructure changes or overlap with existing non-fixture tests.
