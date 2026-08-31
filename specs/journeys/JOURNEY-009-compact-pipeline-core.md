# JOURNEY-009-compact-pipeline-core: Compact pipeline core (fail-safe, exit codes, accounting)

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Compact pipeline core (fail-safe, exit codes, accounting)

## 1. Journey

When **the harness has a tool or shell result that will enter conversation history** I want **a compact pipeline that can accept command plus stdio plus exit code and return a Result** so I **keep the original bytes and exit status on unknown commands and on filter failure, and I can see a ceil(bytes/4) savings ledger even when reduction is 0%**.

## 2. CJM

Alex is wiring context-saving for spin. Steps 1–8 delivered skills, plugins, and lifecycle hooks. History compaction (`internal/contexteng/compactor`) and observation summaries already exist; they shrink conversation messages, not command stdout. RTK R12–R15 require a new `internal/contexteng/compact` package: fail-safe raw output, never rewrite exit codes, unknown-command passthrough, and a token *estimate* ledger. Command-specific filters (R1–R11) and shell-tool wiring are later steps. This journey ships the identity/passthrough path — raw output plus a 0% ledger — so later filters can plug in without changing the Apply contract.

### Phase 1: Call Apply with command and stdio

**User Intent:** Feed one command result into a single entry point.

**Actions:** Construct a `Pipeline` and call `Apply(cmd, stdout, stderr, exitCode)`. Receive a `Result` with stdout, stderr, exit code, strategy, and ledger.

**Pain / Risk:** A missing package forces callers to invent a parallel API; `Apply` swallows stderr; a nil pipeline panics; command string vs argv is ambiguous so later registry keys miss.

**Success Signal:** `Pipeline.Apply` exists and returns a `Result`. Unknown commands keep the same exit code. Production has no command-specific filters.

### Phase 2: Unknown command is identity (R14)

**User Intent:** Unrecognized commands inherit stdio and show 0% reduction.

**Actions:** Call `Apply` with a command that has no registered filter. Compare stdout/stderr bytes to the input. Read the ledger.

**Pain / Risk:** Unknown commands are dropped or truncated; reduction is reported as non-zero; a real tokenizer is called and disagrees with RTK; empty stdio divides by zero.

**Success Signal:** Bytes are unchanged. Ledger reduction is 0%. Strategy is `R14`.

### Phase 3: Exit code is never rewritten (R13)

**User Intent:** A failing command stays failing after compact.

**Actions:** Call `Apply` with exit code 0 and with a nonzero code (for example 2). Compare `Result.ExitCode` to the input.

**Pain / Risk:** Nonzero is coerced to 0 (the CI-agent failure named in the spec); fail-safe or panic recovery zeros the code; only the success path is tested.

**Success Signal:** `Result.ExitCode` equals the input exit code on passthrough, filter error, and panic.

### Phase 4: Filter error or panic is fail-safe (R12)

**User Intent:** A broken filter must not hide the raw output the agent needs.

**Actions:** Install a test-only filter hook that returns an error or panics. Call `Apply`. Inspect stdout, stderr, strategy, and exit code.

**Pain / Risk:** The error is returned to the caller instead of raw output; a panic escapes `Apply`; the filter mutates the slices then fails and the original bytes are gone; strategy is not recorded as `R12`.

**Success Signal:** Original stdout/stderr are returned. Strategy is `R12`. Exit code is unchanged. `Apply` does not panic.

### Phase 5: Ledger uses ceil(bytes/4) (R15)

**User Intent:** Savings accounting is visible without a tokenizer.

**Actions:** After `Apply`, read ledger bytes in/out and `ceil(in/4)-ceil(out/4)`. On the identity path, saved tokens are 0. A shrinking test filter proves the formula when in ≠ out.

**Pain / Risk:** Ledger calls `pkg/tokenizer` and drifts from RTK; integer division truncates instead of ceil; percent is computed on tokens not bytes; empty input panics.

**Success Signal:** Ledger records bytes in/out and `ceil(in/4)-ceil(out/4)`. No tokenizer is imported.

### Phase 6: Passthrough stays inside the compact budget

**User Intent:** Identity apply on a 64 KiB unknown-command fixture stays under the harness NFR.

**Actions:** Run `Apply` many times on a 64 KiB unknown-command fixture. Compute p99 of per-call durations.

**Pain / Risk:** Measuring setup instead of `Apply`; a flake on a loaded CI host; copying 64 KiB on every unknown call without need.

**Success Signal:** Timed test asserts compact p99 < 15 ms on that fixture.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| History compact and observation summaries are not RTK command filters | 1 | New `compact` package with a single `Apply` |
| Unknown commands must not be invented-filtered | 2 | Identity path + 0% ledger (R14) |
| Compact that swallows exit codes breaks CI agents | 3 | Exit code is copied, never derived |
| A filter bug would hide the one line that explained a failure | 4 | Recover + error → raw output, strategy `R12` |
| A real tokenizer disagrees with RTK’s chip | 5 | `ceil(bytes/4)` only |
| Silent performance cliff on large unknown blobs | 6 | p99 < 15 ms gate on 64 KiB |

### North Star Summary

Alex calls `Pipeline.Apply` with a command result and always gets the same exit code. Unknown commands come back byte-identical with a 0% ledger. A filter that errors or panics still returns the original stdio and records `R12`. The ledger is `ceil(bytes/4)` arithmetic, not a tokenizer. The identity path on a 64 KiB unknown fixture stays under 15 ms p99. Command-specific filters and shell wiring stay out of this journey.

### Stressors

1. Unknown command — stdout and stderr bytes are unchanged and reduction is 0%.
2. Exit code 0 is preserved on passthrough.
3. Nonzero exit code (for example 2) is preserved on passthrough.
4. Filter returns an error — original stdio, strategy `R12`, same exit code.
5. Filter panics — `Apply` recovers, original stdio, strategy `R12`, same exit code.
6. Filter mutates the stdout slice then returns an error — fail-safe still yields the pre-filter bytes.
7. Empty stdout and stderr — ledger bytes in/out are 0; reduction is 0%; no divide-by-zero.
8. Odd byte counts (for example 5 in, 1 out) — `TokensSaved` is `ceil(5/4)-ceil(1/4)`, not truncated division.
9. 64 KiB unknown-command fixture — p99 of `Apply` is < 15 ms.
10. Nil `*Pipeline` receiver — treated as no filters (passthrough), no panic.
11. Empty command string — treated as unknown (R14), not a special error.
12. Production `Apply` has no command-specific filters — every real command name is unknown until Step 10.
13. Ledger must not import or call `pkg/tokenizer`.
14. Binary stdout containing NUL bytes is passed through unchanged.
15. Fail-safe path does not rewrite a nonzero exit to 0.

## 3. UX Implementation and Assessment

The operator-facing surface is not a new flag. Value is that later shell and built-in tools can call one `Apply` and trust fail-safe, exit codes, and a 0% identity ledger.

### Time to First Value
- [ ] `Pipeline.Apply` on an unknown command returns usable raw stdout/stderr with no config
- [ ] Ledger is populated on the first call (0% on identity)

### Onboarding Clarity
- [ ] Strategy field names R12 vs R14 so a later status chip can explain passthrough vs fail-safe
- [ ] Package godoc states this step is identity-only (no R1–R11 filters)

### Production-Ready Defaults
- [ ] `New()` has an empty filter table (all commands unknown)
- [ ] Nil pipeline is passthrough, not a panic

### Golden Path Quality
- [ ] Unknown command → unchanged bytes, 0% reduction, same exit code
- [ ] Ledger `BytesIn`/`BytesOut`/`TokensSaved` match `ceil(bytes/4)`

### Decision Load
- [ ] Callers do not choose a tokenizer
- [ ] Callers do not opt into fail-safe; it is always on

### Progressive Complexity
- [ ] Identity path is the only production path
- [ ] Command filters are an opt-in hook (`SetFilter`) for tests and later registry

### Error Quality
- [ ] Filter error is not returned as `error`; it becomes raw output + `R12`
- [ ] Panic inside a filter does not escape `Apply`

### Failure Safety
- [ ] Fail-safe uses a copy of input bytes so in-place mutation cannot erase the original
- [ ] Exit code is never derived from filter success

### Runtime Transparency
- [ ] `Result.Strategy` records which RTK rule produced the bytes
- [ ] `Result.Ledger` exposes in/out bytes and estimated token savings

### Debuggability
- [ ] Original and compacted sizes are both on the ledger
- [ ] Tests name R12 / R13 / R14 / R15 in comments

### Cross-Surface Consistency
- [ ] Same `Apply` contract is what shell (Step 11) and built-ins (Step 12) will call
- [ ] Strategy IDs match the spec table (`R12`, `R14`)

### Workflow Consistency
- [ ] Package lives under `internal/contexteng/compact`, next to compress/observation, not a new top-level tree
- [ ] Journey and test files follow the JOURNEY-00N pattern

### Change Safety
- [ ] No shell tool or harness loop is wired in this step
- [ ] Existing `compactor` / `compress` / `observation` packages are not replaced

### Experimentation Safety
- [ ] Test-only filter hook does not register production command filters
- [ ] p99 gate is a measured pass/fail, not a forecast

### Interaction Latency
- [ ] Unknown-command apply on 64 KiB stays under 15 ms p99
- [ ] Identity path does not invoke a tokenizer

### Developer Feedback Speed
- [ ] Unit tests cover panic, unknown command, and nonzero exit
- [ ] Fail-safe strategy is asserted as `R12`

### Team Scale
- [ ] Journey and tests live in the repo next to the package
- [ ] Ledger formula is a named `TokenDivisor` constant, not a scattered literal

### System Scale
- [ ] Filter map can grow in Step 10 without changing `Apply`’s signature
- [ ] Accounting stays O(1) on the identity path (lengths only)

### Right Behavior by Default
- [ ] Unrecognized commands inherit stdio
- [ ] Fail-safe is the default for filter errors

### Anti-Bypass Design
- [ ] Exit code cannot be omitted from `Result` (it is a field, always set)
- [ ] There is no API to disable fail-safe recover

## 4. Tests

### TC-01: unknown_command_preserves_zero_exit

**Given** a new pipeline and an unrecognized command with exit code 0.
**When** `Apply` runs.
**Then** `Result.ExitCode` is 0.

### TC-02: unknown_command_preserves_nonzero_exit

**Given** a new pipeline and an unrecognized command with a nonzero exit code.
**When** `Apply` runs.
**Then** `Result.ExitCode` equals that nonzero code.

### TC-03: unknown_command_passthrough_bytes

**Given** a new pipeline, an unrecognized command, and nonempty stdout/stderr.
**When** `Apply` runs.
**Then** stdout and stderr bytes equal the inputs and strategy is `R14`.

### TC-04: unknown_command_zero_reduction

**Given** the same unknown-command apply.
**When** the ledger is read.
**Then** reduction is 0% and `BytesIn` equals `BytesOut`.

### TC-05: ledger_ceil_bytes_over_four

**Given** known input and output byte counts (identity, and a shrinking test filter).
**When** `Apply` returns.
**Then** `TokensSaved` equals `ceil(BytesIn/4)-ceil(BytesOut/4)` and no tokenizer is used.

### TC-06: filter_error_fail_safe

**Given** a test filter that returns an error.
**When** `Apply` runs for that command.
**Then** original stdout/stderr are returned and strategy is `R12`.

### TC-07: filter_panic_fail_safe

**Given** a test filter that panics.
**When** `Apply` runs for that command.
**Then** `Apply` returns (does not panic), original stdout/stderr are returned, and strategy is `R12`.

### TC-08: fail_safe_preserves_nonzero_exit

**Given** a panicking or erroring filter and a nonzero exit code.
**When** `Apply` runs.
**Then** `Result.ExitCode` is still that nonzero code.

### TC-09: empty_stdio_ledger

**Given** nil or empty stdout and stderr.
**When** `Apply` runs on an unknown command.
**Then** ledger bytes are 0, reduction is 0%, and `Apply` does not panic.

### TC-10: compact_p99_64kib

**Given** a 64 KiB unknown-command fixture.
**When** `Apply` is timed over many samples.
**Then** the p99 duration is < 15 ms.

### TC-11: nil_pipeline_passthrough

**Given** a nil `*Pipeline`.
**When** `Apply` is called.
**Then** the call is identity passthrough (R14) and does not panic.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 9:

- `Pipeline.Apply(cmd, stdout, stderr, exitCode) Result` returns same exit code always
- Panic or filter error yields original stdout/stderr and records strategy `R12`
- Unknown command yields unchanged bytes and 0% reduction (`R14`)
- Ledger records bytes in/out and `ceil(in/4)-ceil(out/4)` (`R15`)
- Unit tests for panic, unknown command, nonzero exit preserved
- Compact p99 < 15 ms on a 64 KiB unknown-command fixture
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 9
- Implementation files: `internal/contexteng/compact/pipeline.go`, `internal/contexteng/compact/ledger.go`
- Test files: `internal/contexteng/compact/pipeline_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-009-compact-pipeline-core.md` — this journey
- `internal/contexteng/compact/pipeline.go` — `Pipeline.Apply`, fail-safe recover, test-only `SetFilter`
- `internal/contexteng/compact/ledger.go` — R15 `ceil(bytes/4)` accounting
- `internal/contexteng/compact/pipeline_test.go` — panic, unknown command, nonzero exit, ledger, 64 KiB p99

Files modified:
- `specs/agent-harness/ROADMAP.md` — Step 9 DoD ticks and traceability
- `docs/testing.md` — journey 009 test row
