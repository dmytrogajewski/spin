# JOURNEY-007-finish-the-hook-runner-contract: Finish the hook runner contract

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Finish the hook runner contract

## 1. Journey

When **an operator keeps global hook scripts under `~/.spin/hooks` and writes a `PRE_TOOL_USE` script that returns JSON `updated_input`** I want **spin to resolve that home path and replace the tool’s arguments with that payload** so I **can enforce org-wide hooks and sanitize tool input without the runner dropping the rewrite**.

## 2. CJM

Alex already has plugin hooks registered (Step 6) and a runner that parses `updated_input` on exit-code-2 JSON. Two gaps remain: builders pass `filepath.Join("~", ".spin", "hooks")`, so `os.Stat` never finds user-level scripts; and `HookResult.UpdatedInput` is never applied to `call.Function.Arguments`, so a rewrite is dropped even when parsed. This journey expands the global dir with `os.UserHomeDir` (via existing `pathx.ExpandHome` or equivalent), has ACP and TUI builders pass that expanded path, and applies a full JSON-object replace to tool arguments. Exit-code-2 still blocks. Script filenames stay `Event.ScriptName()`. Later lifecycle emitters (Step 8+) stay out of scope.

### Phase 1: Expand the global hooks directory

**User Intent:** Global scripts under the real home directory are discovered.

**Actions:** Stop constructing `filepath.Join("~", …)`. Resolve the user home with `os.UserHomeDir` (or `pathx.ExpandHome`). ACP `buildCoreAgent` and the TUI conversation builder pass the expanded `GlobalDir` into `hooks.NewRunner`. `NewRunner` expands a leading `~` if a caller still passes one.

**Pain / Risk:** `filepath.Join("~", ".spin", "hooks")` remains and `Stat` looks for a literal `~` directory; `UserHomeDir` fails and the builder panics; expansion happens only in tests; `~user` is treated as the current user; ProjectDir is rewritten accidentally.

**Success Signal:** `DefaultGlobalDir` (or equivalent) equals `filepath.Join(home, ".spin", "hooks")` and does not start with `~`. A runner configured with `~/.spin/hooks` executes a script placed under `$HOME/.spin/hooks`.

### Phase 2: Parse `updated_input` on a successful blocking hook

**User Intent:** A `PRE_TOOL_USE` script that exits 0 can rewrite input, not only veto.

**Actions:** Keep exit-code-2 JSON parse (`reason` + `updated_input`) as today. Also parse JSON stdout on exit 0 so `HookResult.UpdatedInput` is populated when `updated_input` is present. `executeBlocking` returns the last non-empty rewrite if no script blocked.

**Pain / Risk:** Success-path JSON is ignored and `UpdatedInput` stays empty; `executeBlocking` still returns a zero `HookResult` after a rewrite; requiring `reason` on success drops a rewrite-only payload; exit 2 no longer blocks.

**Success Signal:** Exit 0 with `{"updated_input":…}` yields `Blocked=false` and a non-empty `UpdatedInput`. Exit 2 still sets `Blocked=true`.

### Phase 3: Replace structured tool arguments

**User Intent:** The tool executes with the hook’s replacement object, not the model’s original JSON.

**Actions:** After `PRE_TOOL_USE`, if `UpdatedInput` is a JSON object, replace `call.Function.Arguments` and the parsed `ToolParameters` entirely (no field merge). If `UpdatedInput` is empty, leave original arguments unchanged. If `UpdatedInput` is a non-object (plain string such as `"sanitized"`), do not apply it as structured parameters.

**Pain / Risk:** A raw string is assigned onto object args and parse fails or corrupts keys; merge-by-key leaves attacker fields; rewrite happens only on `Blocked=true` so the tool never runs; approval sees rewritten args and the hook never does, or the reverse is undocumented as a later change.

**Success Signal:** A capturing structured tool’s `Execute` params match the replacement object. Empty `updated_input` keeps the original object.

### Phase 4: Replace shell-style arguments the same way

**User Intent:** `shell_command` (and any tool whose args are a JSON object with `command`) sees the same full-object replace.

**Actions:** A hook that returns `{"updated_input":"{\"command\":\"echo safe\"}"}` or an object-valued `updated_input` replaces the whole argument object. Tests cover a shell-shaped payload (`command`) and a structured payload (`path` or equivalent).

**Pain / Risk:** Shell argv is treated as a raw string and the command JSON is invalid; only structured tools are tested; array argv is silently concatenated in a way that changes meaning — out of scope unless defined as a JSON object replace.

**Success Signal:** A capturing tool invoked as `shell_command`-shaped args receives `command` from `updated_input`, not the original.

### Phase 5: Stay inside Step 7

**User Intent:** Finish the runner contract without adding emitters or new filenames.

**Actions:** Do not add `POST_TOOL_USE_FAILURE`, `PRE_COMPACT`, `STOP`, or `SESSION_END` production call sites. Do not rename `Event.ScriptName()` files. Do not change exit-code-2 meaning.

**Pain / Risk:** Step 8 emitters land here; a new script suffix (`.sh`) is required; exit 2 with `updated_input` is reinterpreted as allow-and-rewrite.

**Success Signal:** Existing lifecycle and plugin hook tests still pass. Roadmap Step 7 DoD is the only newly closed hook work.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Literal `~` in GlobalDir never matches `$HOME` | 1 | `UserHomeDir` / `pathx.ExpandHome` |
| `UpdatedInput` parsed on block and unused | 2–3 | Apply on the non-block path as a full JSON object |
| Raw string rewrite corrupts tool JSON | 3 | Replace only when `updated_input` is a JSON object |
| Shell and structured tools could diverge | 4 | Same object-replace contract for both |
| Exit-2 semantics could be overloaded | 5 | Exit 2 still blocks; rewrite is exit 0 |

### North Star Summary

Alex drops `~/.spin/hooks/pre-tool-use`. Spin finds it because ACP and TUI pass an expanded home path. A blocking-event script that exits 0 with JSON `updated_input` as a full argument object runs the tool with those arguments; an empty `updated_input` leaves the original arguments; exit 2 still vetoes. Script names stay `pre-tool-use` and friends.

### Stressors

1. `UserHomeDir` fails — global dir is empty (no panic); project hooks still run.
2. `GlobalDir` is already absolute — expansion is a no-op.
3. `GlobalDir` is `~/.spin/hooks` — expands to `$HOME/.spin/hooks`.
4. `filepath.Join("~", …)` is reintroduced at a call site — tests or grep-equivalent assert builders do not pass a tilde path.
5. Hook exits 0 with `{"updated_input":"{\"path\":\"safe\"}"}` — tool sees `path=safe`.
6. Hook exits 0 with empty or omitted `updated_input` — original arguments unchanged.
7. Hook exits 0 with `{"updated_input":"sanitized"}` (non-object) — original arguments unchanged (no JSON corruption).
8. Hook exits 2 with reason and `updated_input` — still blocked; tool does not execute (existing lifecycle assertion).
9. Global then project both rewrite — last successful rewrite wins; first exit 2 still stops the chain.
10. Structured tool (`path`) and shell-shaped tool (`command`) both receive object replace.
11. `parseBlockResult` still requires a non-empty `reason` for the JSON block branch (plain-text exit 2 unchanged).
12. Invalid replacement JSON is not assigned onto `Function.Arguments`.
13. `~user/foo` is not treated as the current user’s home (`pathx.ExpandHome` contract).
14. Plugin hook scripts and `ScriptName()` filenames are unchanged.
15. No new lifecycle event emitters are added.

## 3. UX Implementation and Assessment

The operator-facing surface is still workspace-trusted hook scripts. Value is that `~/.spin/hooks` actually runs and a rewrite JSON changes the tool call the model issued.

### Time to First Value
- [ ] A script in the real `~/.spin/hooks/pre-tool-use` is discovered after home expansion
- [ ] No new CLI flag is required to expand home or apply `updated_input`

### Onboarding Clarity
- [ ] Global dir is documented as `~/.spin/hooks` meaning the expanded home path
- [ ] `updated_input` is a full JSON object replace, not a merge and not a raw string

### Production-Ready Defaults
- [ ] Home expansion failure does not crash the process
- [ ] Empty `updated_input` is a no-op

### Golden Path Quality
- [ ] Expanded `GlobalDir` is passed from ACP and TUI builders
- [ ] Exit 0 + JSON object `updated_input` is what the tool executes

### Decision Load
- [ ] Operators do not choose an expander or a rewrite backend
- [ ] Filenames remain the existing `ScriptName()` set

### Progressive Complexity
- [ ] Project hooks work with no global dir
- [ ] Object rewrite is opt-in via hook JSON

### Error Quality
- [ ] Non-object `updated_input` does not corrupt arguments
- [ ] Exit 2 still reports the hook reason as a block

### Failure Safety
- [ ] Invalid replacement JSON keeps original arguments
- [ ] Hook timeout behavior is unchanged

### Runtime Transparency
- [ ] `HookResult.UpdatedInput` is populated from JSON on exit 0 and exit 2
- [ ] `call.Function.Arguments` after a rewrite equals the replacement object

### Debuggability
- [ ] Tests place scripts under a fake `$HOME/.spin/hooks`
- [ ] Capturing tools record the arguments they received

### Cross-Surface Consistency
- [ ] ACP and TUI builders both pass an expanded global dir
- [ ] Shell-shaped and structured tools share the object-replace contract

### Workflow Consistency
- [ ] Expansion uses `pathx.ExpandHome` or `os.UserHomeDir`, not a new dependency
- [ ] Execution stays in `internal/safety/hooks` and `internal/agent/tool`

### Change Safety
- [ ] Exit-code-2 block semantics are unchanged
- [ ] Step 8+ emitters are not added

### Experimentation Safety
- [ ] Tests use `t.TempDir` / `t.Setenv("HOME", …)`, not the operator home tree
- [ ] Scripts still run via `/bin/sh` without the executable bit

### Interaction Latency
- [ ] Expansion is a single `UserHomeDir` + join
- [ ] Runner timeout budget is unchanged

### Developer Feedback Speed
- [ ] `go test ./internal/safety/hooks ./internal/agent/tool` isolates expansion vs replace
- [ ] Lifecycle test still asserts parsed `updated_input` `"sanitized"`

### Team Scale
- [ ] Org-wide hooks in `~/.spin/hooks` apply to every workdir
- [ ] The same `ScriptName()` files work for global, project, and plugin dirs

### System Scale
- [ ] Extra global scripts do not require a new runner type
- [ ] Rewrite is per tool call, not a session-wide mutation

### Right Behavior by Default
- [ ] Tilde in `GlobalDir` expands
- [ ] Missing rewrite field leaves arguments alone

### Anti-Bypass Design
- [ ] Exit 2 still prevents `tool.Execute`
- [ ] A non-object rewrite cannot smuggle a string into structured args

## 4. Tests

### TC-01: default_global_dir_expands_home

**Given** `os.UserHomeDir` succeeds.
**When** `DefaultGlobalDir` (or the builder helper) is called.
**Then** the result is `filepath.Join(home, ".spin", "hooks")` and does not contain a `~` path segment.

### TC-02: new_runner_expands_tilde_global_dir

**Given** `HOME` is a temp directory containing `.spin/hooks/pre-tool-use` that writes a marker.
**When** `NewRunner` is constructed with `GlobalDir: "~/.spin/hooks"` and `Execute` runs `PRE_TOOL_USE`.
**Then** the marker exists (the tilde path was expanded).

### TC-03: builders_pass_expanded_global_dir

**Given** ACP and TUI builder hook configuration helpers.
**When** they compute `GlobalDir`.
**Then** the value equals `DefaultGlobalDir()` and is not `filepath.Join("~", …)`.

### TC-04: exit_0_json_updated_input_parsed

**Given** a `pre-tool-use` script that prints `{"updated_input":"{\"path\":\"safe\"}"}` and exits 0.
**When** `Execute` runs `PRE_TOOL_USE`.
**Then** `Blocked` is false and `UpdatedInput` is the replacement JSON object string.

### TC-05: exit_2_still_blocks

**Given** a `pre-tool-use` script that prints `{"reason":"policy violation","updated_input":"sanitized input"}` and exits 2.
**When** `Execute` runs `PRE_TOOL_USE`.
**Then** `Blocked` is true, `Reason` is the policy text, and `UpdatedInput` is still parsed (`sanitized input`). The tool does not execute.

### TC-06: empty_updated_input_keeps_original_args

**Given** a `PRE_TOOL_USE` hook that exits 0 with no `updated_input` (or empty string).
**When** the tool runtime executes a call with original JSON arguments.
**Then** the tool’s `Execute` params equal the original object.

### TC-07: object_updated_input_replaces_structured_args

**Given** a capturing structured tool and a hook that returns a JSON object `updated_input` (`path` replaced).
**When** `Runtime.Execute` runs.
**Then** the tool sees the replacement object and not the original `path`.

### TC-08: object_updated_input_replaces_shell_shaped_args

**Given** a capturing tool and original args `{"command":"rm -rf /"}`.
**When** the hook returns `updated_input` `{"command":"echo safe"}`.
**Then** the tool sees `command=echo safe`.

### TC-09: non_object_updated_input_does_not_corrupt

**Given** a hook that returns `updated_input` `"sanitized"` (JSON string, not object).
**When** `Runtime.Execute` runs a structured call.
**Then** original arguments are unchanged.

### TC-10: last_rewrite_wins_until_block

**Given** a global hook that rewrites `updated_input` and a project hook that exits 0 without rewrite.
**When** `Execute` runs.
**Then** the global rewrite is preserved. If the project hook exits 2, the operation is blocked.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 7:

- `filepath.Join("~", …)` is not used; `os.UserHomeDir` (or equivalent) expands the global hooks dir
- ACP and TUI builders pass an expanded global dir
- When a blocking hook returns JSON `updated_input`, the tool sees the replaced arguments
- When `updated_input` is empty, original arguments are unchanged
- Tests cover home expansion and argument replacement
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 7
- Implementation files: `internal/safety/hooks/dir.go`, `internal/safety/hooks/runner.go`, `internal/agent/tool/runtime.go`, `cmd/spin/acp.go`, `internal/conversation/builder.go`, `internal/conversation/agent.go`
- Test files: `internal/safety/hooks/dir_test.go`, `internal/agent/tool/runtime_updated_input_test.go`, `internal/conversation/hooks_dir_test.go`, `cmd/spin/acp_hooks_dir_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-007-finish-the-hook-runner-contract.md` — this journey
- `internal/safety/hooks/dir.go` — `DefaultGlobalDir` expands `~/.spin/hooks` via `pathx.ExpandHome` / `os.UserHomeDir`
- `internal/safety/hooks/dir_test.go` — home expansion, exit-0 `updated_input` parse, last rewrite wins, object-valued `updated_input`
- `internal/agent/tool/runtime_updated_input_test.go` — structured, shell-shaped, empty, and non-object replacement
- `internal/conversation/hooks_dir_test.go` — TUI builder passes expanded global dir
- `cmd/spin/acp_hooks_dir_test.go` — ACP builder passes expanded global dir

Files modified:
- `internal/safety/hooks/runner.go` — expand `GlobalDir` on `NewRunner`; parse `updated_input` on exit 0; last rewrite wins; `updated_input` accepts JSON object or string
- `internal/safety/hooks/context.go` — `UpdatedInput` is a full argument-JSON replace
- `internal/agent/tool/runtime.go` — apply JSON-object `UpdatedInput` to tool arguments; leave original args when empty or non-object
- `cmd/spin/acp.go` — `acpHooksGlobalDir()` instead of `filepath.Join("~", …)`
- `internal/conversation/builder.go` — `hooksGlobalDir()` uses `DefaultGlobalDir`
- `internal/conversation/agent.go` — TUI `NewRunner` receives the expanded global dir
- `specs/agent-harness/ROADMAP.md` — Step 7 DoD ticks and traceability
- `docs/testing.md` — journey 007 integration test row

