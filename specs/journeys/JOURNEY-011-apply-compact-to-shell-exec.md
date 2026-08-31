# JOURNEY-011-apply-compact-to-shell-exec: Apply compact to shell exec and PreToolUse rewrite

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Apply compact to shell exec and PreToolUse rewrite

## 1. Journey

When **the harness runs a Bash-equivalent `shell_command` (for example `git status`)** I want **the result compacted before it becomes conversation history, and argv rewritten at PreToolUse when `compact.backend: rtk` can exec PATH `rtk`** so I **do not spend model tokens on a wrapper, keep nonzero exits nonzero, and can turn the whole path off with `SPIN_COMPACT=0` or `compact.enabled: false`**.

## 2. CJM

Alex already has `Default()` filters and goldens (Step 10). Production `shell_command` still dumps raw stdio into the transcript. Typing `rtk git status` is not adoption — the model forgets. This journey applies the Go pipeline once on the tool result (history path) and, only when the optional `rtk` backend is selected and the binary exists, prefixes argv so the process itself is compact-aware. Built-in read/grep/ls stay raw (Step 12). No TUI chip (Step 13).

### Phase 1: Compact shell results into history

**User Intent:** After `shell_command` execute, the observation the model sees is the compact form, not the raw blob.

**Actions:** Run `git status` (or a fake blob through a mock executor). Inspect the tool result / harness observation. Confirm unknown commands still passthrough (R14).

**Pain / Risk:** Compact never runs and history stays huge; compact runs on `get_environment` / `validate` and hides introspection; apply happens twice (rewrite + post-filter) and garbles already-compact text; TruncateOutput runs on raw bytes and drops the compact shape.

**Success Signal:** A fake porcelain `git status` blob becomes the registry compact text in the harness observation. Unknown commands stay raw. One apply site per result.

### Phase 2: Preserve exit codes (R13)

**User Intent:** Compaction never lies about process success.

**Actions:** Execute a command whose mock/real exit is nonzero. Read `ToolResult.Success`, `Error`, and `ExitCode`.

**Pain / Risk:** Exit 1 becomes 0 after filter; `Success` flips true because compact stdout looks “clean”; executor errors drop the original exit; CI agents treat a failed `go test` as green.

**Success Signal:** Nonzero shell exit stays nonzero after compact. `ExitCode` on the tool result matches the process.

### Phase 3: Argv-level R11 rewrite without model tokens

**User Intent:** The model types `git status`. The runtime prefixes compacting when the rtk backend is live. The model never types `rtk`.

**Actions:** Configure `compact.backend: rtk` with a fake PATH `rtk`. Issue `shell_command` with `command: git status`. Inspect argv that reached the executor.

**Pain / Risk:** Rewrite is a prompt hint (costs tokens); rewrite is a shell-string wrap the model must emit; PreToolUse `UpdatedInput` from plugins is overwritten; rewrite runs for `read_file` (Step 12 scope); missing `rtk` still prefixes and exec fails.

**Success Signal:** When PATH has `rtk` and backend is `rtk`, argv is compact-aware (`rtk git status`) with zero extra model tokens. Plugin `UpdatedInput` still replaces structured args first. When `rtk` is absent, argv is unchanged and the Go pipeline is the apply site.

### Phase 4: Escape hatch and rtk fallback

**User Intent:** Operators can restore raw output and never depend on a real `rtk` binary in tests or default installs.

**Actions:** Set `SPIN_COMPACT=0` or `compact.enabled: false`. Repeat `git status`. Set backend `rtk` with LookPath miss. Confirm Go pipeline still runs when compact is on.

**Pain / Risk:** Escape hatch skips rewrite but not the filter (or the reverse); default is off and nobody gets savings; tests require a real `rtk` on the machine; network is used to fetch rtk.

**Success Signal:** Either escape skips both rewrite and filters. Default compact is on. `rtk` backend uses PATH when present and falls back to Go `Apply` when it does not. Tests use a fake PATH entry.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Model must remember to type `rtk` | 3 | Argv rewrite after the model already chose `git status` |
| Raw `git status` / test blobs fill the window | 1 | Post-filter `Default().Apply` on exec results only |
| Double-compact if rewrite and filter both apply | 1 / 3 | One apply site: skip Go filter when argv is already `rtk`-prefixed |
| Compact that swallows exit codes breaks CI agents | 2 | Copy exit code; never rewrite it |
| No way to see raw output while debugging | 4 | `SPIN_COMPACT=0` and `compact.enabled: false` |

### North Star Summary

Alex runs the agent as today. Compact is on. `shell_command` observations are compact text (grouped `git status`, failures-only tests) with the same exit code. If they set `compact.backend: rtk` and have `rtk` on PATH, PreToolUse-equivalent rewrite prefixes the binary — the model still typed the original command. If `rtk` is missing, the Go pipeline is the only apply site. `SPIN_COMPACT=0` restores raw exec output. Built-in read/grep/ls and the status chip are untouched.

### Stressors

1. Fake porcelain `git status` blob — observation is compact grouped text, not the raw `M ` / `??` lines.
2. Unknown command (`echo hello`) — bytes unchanged (R14), still a successful observation.
3. Nonzero exit (for example 127 or fixture exit 1) stays nonzero after compact; `Success` stays false.
4. Applying compact twice (idempotence) — exit code unchanged; no panic; second apply is not a second product path (one apply site).
5. `compact.backend: rtk` with LookPath hit — argv becomes `rtk git status`; model arguments are not re-prompted.
6. `compact.backend: rtk` with LookPath miss — argv unchanged; Go pipeline still filters the result.
7. `SPIN_COMPACT=0` — no argv prefix and no Go filter; raw blob in the observation.
8. `compact.enabled: false` — same skip as the env hatch (rewrite and filters).
9. Command already prefixed `rtk git status` — Go pipeline does not apply again (no double-compact).
10. Plugin PreToolUse `UpdatedInput` replaces `command` — rewrite must not drop that replacement; empty/non-object UpdatedInput still keeps original args.
11. Introspection ops (`get_environment`, `validate`) — not passed through exec compact.
12. Default config — `compact.enabled` is on without any YAML key; no network; no real `rtk` binary required.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Compact applies to `shell_command` execute with no extra prompt from the user
- [x] Default-on: first exec observation is already compacted

### Onboarding Clarity
- [x] Escape hatch names (`SPIN_COMPACT=0`, `compact.enabled`) match the spec
- [x] Missing `rtk` is silent fallback, not a hard error

### Production-Ready Defaults
- [x] `compact.enabled` defaults true
- [x] Empty `compact.backend` uses the Go pipeline (no PATH `rtk` required)

### Golden Path Quality
- [x] `git status` fixture → compact observation
- [x] Exit code and success flag match the process

### Decision Load
- [x] No new tool the model must choose
- [x] Backend `rtk` is opt-in config, not a per-call flag

### Progressive Complexity
- [x] Simple case: default Go pipeline on exec results
- [x] Advanced: `compact.backend: rtk` only when the operator wants the binary

### Error Quality
- [x] Nonzero exit still surfaces as command failed with that code
- [x] Filter errors stay fail-safe raw (R12) via existing pipeline

### Failure Safety
- [x] Escape hatch restores raw output
- [x] LookPath miss does not fail the tool call

### Runtime Transparency
- [x] Rewritten argv is what the executor runs (inspectable in tests)
- [x] Ledger/strategy stay on the pipeline result (chip is Step 13)

### Debuggability
- [x] `SPIN_COMPACT=0` is the raw-output probe
- [x] Goldens remain the compact contract for `git status`

### Cross-Surface Consistency
- [x] Same `Default()` filters as Step 10
- [x] Terminology: compact, R11, rtk backend, escape hatch

### Workflow Consistency
- [x] PreToolUse `UpdatedInput` path unchanged
- [x] Plugin hook scripts still run before rewrite

### Change Safety
- [x] Config key is additive; existing YAML without `compact:` stays valid
- [x] `NewShellCommandTool` signature unchanged for existing callers

### Experimentation Safety
- [x] Tests inject LookPath / fake PATH — no real `rtk`
- [x] Escape hatch is revertible per process / config

### Interaction Latency
- [x] Rewrite is in-process argv mutation (no extra model round-trip)
- [x] Go `Apply` is the existing in-memory pipeline

### Developer Feedback Speed
- [x] Failed exec still returns output + error immediately
- [x] Operator can disable compact without restarting a rewrite protocol

### Team Scale
- [x] `compact.enabled` / `compact.backend` live in versionable YAML
- [x] `SPIN_COMPACT=0` works uniformly for local debug

### System Scale
- [x] One apply site per exec result (no second history filter)
- [x] Built-in tools left for Step 12 — no structural fork

### Right Behavior by Default
- [x] Compact on; backend Go; exit codes preserved
- [x] No network; no required extra binary

### Anti-Bypass Design
- [x] Model cannot skip compact by omitting a wrapper (rewrite or post-filter)
- [x] Escape hatch is explicit env/config, not a tool argument the model invents

## 4. Tests

### TC-01: default compact enabled

**Given** `DefaultV2()`.
**When** the compact section is read.
**Then** `compact.enabled` is true and backend is empty (Go pipeline).

### TC-02: shell git status compacted

**Given** a `shell_command` tool with a mock executor returning the porcelain fixture.
**When** execute `git status`.
**Then** `ToolResult.Output` equals the compact grouped text.

### TC-03: nonzero exit preserved

**Given** the same fixture with exit code 1.
**When** execute completes.
**Then** `ExitCode` is 1, `Success` is false, output is still compact.

### TC-04: unknown command passthrough

**Given** mock stdout `hello world` for `echo hello`.
**When** execute completes.
**Then** output is unchanged (R14).

### TC-05: SPIN_COMPACT=0 skips filter

**Given** `SPIN_COMPACT=0` and a git status fixture.
**When** execute `git status`.
**Then** output is the raw blob.

### TC-06: compact.enabled false skips filter

**Given** `SetCompactEnabled(false)` (or config `enabled: false`).
**When** execute `git status`.
**Then** output is the raw blob and argv is not rewritten.

### TC-07: rtk rewrite when binary exists

**Given** `compact.backend: rtk` and a LookPath that finds `rtk`.
**When** PreToolUse-equivalent rewrite runs on `git status`.
**Then** argv/command is `rtk git status` (zero extra model tokens).

### TC-08: rtk missing falls back to Go pipeline

**Given** backend `rtk` and LookPath miss.
**When** execute `git status` with the fixture.
**Then** command stays `git status` and output is Go-pipeline compact text.

### TC-09: rtk-prefixed command is not double-compacted

**Given** command already `rtk git status` and compact on.
**When** the post-filter runs.
**Then** stdout is not passed through `Default().Apply` again.

### TC-10: UpdatedInput still applied

**Given** a `pre-tool-use` script that sets `updated_input` command.
**When** Runtime executes `shell_command`.
**Then** the hook replacement is the command the tool sees; rewrite does not drop it.

### TC-11: harness observation integration

**Given** Runtime + `shell_command` + fake git status blob.
**When** `Runtime.Execute` returns.
**Then** the observation content is compact text (the history path).

### TC-12: apply idempotence

**Given** compact stdout from `Default().Apply("git status", raw, …)`.
**When** `Apply` is called again on that stdout.
**Then** exit code is unchanged and the call does not panic.

### TC-13: loader default and env hatch

**Given** config YAML without a `compact` key, then `SPIN_COMPACT=0`.
**When** config is loaded / `Active()` is evaluated.
**Then** enabled defaults on; env `0` makes compact inactive.

## 5. Acceptance Criteria

- Shell tool results are compacted before they enter conversation history
- Nonzero shell exit stays nonzero after compact
- Rewrite is argv-level (e.g. `git status` → compact-aware execution) with zero extra model tokens
- `compact.backend: rtk` uses PATH `rtk` when the binary exists; falls back to the Go pipeline when it does not
- `SPIN_COMPACT=0` / `compact.enabled: false` skips rewrite and filters
- Integration test: fake `git status` blob becomes compact text in a harness observation
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 11
- Implementation files: `internal/contexteng/compact/rewrite.go`, `internal/tools/shell_command.go`, `internal/agent/tool/runtime.go`, `internal/config/config_v2.go`, `internal/config/loader_v2.go`, `internal/conversation/agent.go`, `internal/conversation/tools.go`, `internal/agent/executor/builtin.go`, `cmd/spin/services.go`, `cmd/spin/acp.go`
- Test files: `internal/contexteng/compact/rewrite_test.go`, `internal/tools/shell_command_compact_test.go`, `internal/agent/tool/runtime_compact_test.go`, `internal/config/config_v2_test.go`, `internal/config/loader_v2_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md` — this journey
- `internal/contexteng/compact/rewrite.go` — `RewriteArgv`, `ShouldApply`, `EnvDisabled`, rtk prefix helpers
- `internal/contexteng/compact/rewrite_test.go` — R11 rewrite, fake PATH `rtk`, escape hatch, Apply idempotence
- `internal/tools/shell_command_compact_test.go` — exec post-filter, exit preserve, skip apply
- `internal/agent/tool/runtime_compact_test.go` — argv rewrite, rtk fallback, git status harness observation

Files modified:
- `internal/tools/shell_command.go` — one `Default().Apply` site on execute results; `SetCompactEnabled`; preserve `ExitCode`
- `internal/agent/tool/runtime.go` — R11 argv rewrite after PreToolUse `UpdatedInput`
- `internal/config/config_v2.go` — `CompactV2`, default on, `Active()`, `CompactBackendRTK`
- `internal/config/loader_v2.go` — compact defaults and env bind
- `internal/conversation/agent.go` — wire compact settings + `LookPath` into tool runtime
- `internal/conversation/tools.go` — `SetCompactEnabled` from `cfg.Compact.Active()`
- `internal/agent/executor/builtin.go` — optional `CompactEnabled` override
- `cmd/spin/services.go` / `cmd/spin/acp.go` — production compact wiring
- `internal/contexteng/compact/pipeline.go` — R11 comment (argv, not a filter)
- `specs/agent-harness/ROADMAP.md` — Step 11 DoD ticks and traceability
- `docs/testing.md` — journey 011 test row
