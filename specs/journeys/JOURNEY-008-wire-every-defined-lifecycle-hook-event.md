# JOURNEY-008-wire-every-defined-lifecycle-hook-event: Wire every defined lifecycle hook event

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Wire every defined lifecycle hook event

## 1. Journey

When **an operator has installed hook scripts for the events declared in `internal/safety/hooks/event.go`** I want **spin to fire every parent-side lifecycle event at its mapped call site** so I **can audit tool failures, compact, loop end, and session teardown without waiting for child spawn (Step 19)**.

## 2. CJM

Alex already has a runner that discovers `Event.ScriptName()` files (Steps 6–7) and production emitters for `SESSION_START`, `USER_PROMPT_SUBMIT`, `PRE_TOOL_USE`, and `POST_TOOL_USE`. Four declared events have no production call site: `POST_TOOL_USE_FAILURE`, `PRE_COMPACT`, `STOP`, and `SESSION_END`. `SUBAGENT_START` and `SUBAGENT_STOP` stay unregistered on the parent spawn path until Step 19; this journey adds the eight parent-side emitters and an integration test that registers recording scripts for all ten names. Ctrl-C must still reach `SESSION_END` from the TUI teardown that already clears the screen.

### Phase 1: Tool error becomes POST_TOOL_USE_FAILURE

**User Intent:** A hook that watches failed tools runs only when the tool errors, not on success.

**Actions:** After `tool.Execute` returns an error, the tool runtime fires `POST_TOOL_USE_FAILURE` instead of `POST_TOOL_USE`. Success still fires `POST_TOOL_USE`. `PRE_TOOL_USE` still runs before execute.

**Pain / Risk:** The error path keeps calling `POST_TOOL_USE` so failure scripts never see the event; validation/approval denials are treated as tool errors and fire the wrong name; `nil` hook runner panics; a failed tool also fires success.

**Success Signal:** A recording `post-tool-use-failure` script creates its marker after `tool.Execute` returns an error. A successful execute creates the `post-tool-use` marker and does not create the failure marker.

### Phase 2: PRE_COMPACT before history rewrite

**User Intent:** Compact hooks run before the conversation history is replaced.

**Actions:** When a compactor is configured, the harness `phaseCompaction` path fires `PRE_COMPACT` and only then calls `Compact`. The builder passes the same hook runner already used for tool events (`WithHookRunner`). No compact pipeline (Step 9) is started.

**Pain / Risk:** The hook runs after `iterCtx.Messages` is assigned the compacted slice; the hook is skipped when `changed` is false but `Compact` already rewrote; a nil hook runner panics; a nil compactor still attempts the hook.

**Success Signal:** A recording `pre-compact` script’s marker exists at the moment `Compact` is entered. History rewrite happens only after that execute.

### Phase 3: STOP then SESSION_END on parent teardown

**User Intent:** Loop end and conversation close are observable, including Ctrl-C.

**Actions:** The parent harness `Execute` path fires `STOP` after `runLoop` returns. `Conversation.Close` fires `STOP` then `SESSION_END` using a context that is not canceled (`context.WithoutCancel`). `runTUI` / `stopTUILoop` invoke `Close` on the same teardown that already cancels the event loop and clears the TUI so Ctrl-C cannot skip `SESSION_END`.

**Pain / Risk:** `defer conv.Close(ctx)` uses the canceled TUI context and async hook scripts receive `ctx.Err()` immediately; `SESSION_END` is omitted from `stopTUILoop`; `STOP` and `SESSION_END` fire in reverse order; `Close` is skipped on `/exit` or input-channel close.

**Success Signal:** Close with a canceled context still creates `stop` then `session-end` markers, in that order. `stopTUILoop` still unblocks the event loop without a second SIGINT.

### Phase 4: Existing parent emitters stay wired

**User Intent:** Wiring the missing events does not drop the four that already work.

**Actions:** Keep `SESSION_START` on the builder, `USER_PROMPT_SUBMIT` on `RunTurn`, and `PRE_TOOL_USE` / `POST_TOOL_USE` on the tool runtime. Do not rename `Event.ScriptName()` files. Do not change exit-code-2 meaning.

**Pain / Risk:** A refactor of `runPostToolHook` drops success `POST_TOOL_USE`; builder no longer fires `SESSION_START`; `RunTurn` skips `USER_PROMPT_SUBMIT` when a harness executor is present.

**Success Signal:** Existing lifecycle and tool-hook tests still pass. The integration test records those four names along with the four new parent-side names.

### Phase 5: Integration coverage for all ten names

**User Intent:** One test proves the parent-side subset without requiring child spawn.

**Actions:** Register recording scripts for every name in `hooks.AllEvents()` (ten). Drive the parent emitters with a stub harness / stub subagent manager (no production spawn). Assert the eight parent-side names appear. Do not require `SUBAGENT_START` or `SUBAGENT_STOP` to fire.

**Pain / Risk:** The test requires `SUBAGENT_*` and fails until Step 19; only unit tests exist and the DoD integration bullet stays open; async non-blocking hooks are asserted before the marker exists.

**Success Signal:** The integration test waits for the eight parent-side markers and treats missing `SUBAGENT_*` markers as allowed.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Tool error still fires `POST_TOOL_USE` | 1 | Dedicated failure event on the `Execute` error return |
| Compact hook would see rewritten history | 2 | Fire `PRE_COMPACT` before `Compact` |
| Canceled TUI ctx drops async `SESSION_END` | 3 | `WithoutCancel` + teardown `Close` on `stopTUILoop` |
| New emitters could clobber existing four | 4 | Keep current call sites; assert them in the same recording test |
| `SUBAGENT_*` have no parent spawn yet | 5 | Register all ten names; assert only the parent-side eight |

### North Star Summary

Alex drops scripts named `post-tool-use-failure`, `pre-compact`, `stop`, and `session-end` next to the four that already run. A failed tool hits the failure script. Compact scripts run before history is replaced. Ending the parent loop and closing the conversation fire `STOP` then `SESSION_END`, including when the TUI tears down on Ctrl-C. Child start/stop scripts can sit on disk unused until Step 19.

### Stressors

1. `tool.Execute` returns an error — `POST_TOOL_USE_FAILURE` fires and `POST_TOOL_USE` does not.
2. `tool.Execute` succeeds — `POST_TOOL_USE` fires and `POST_TOOL_USE_FAILURE` does not.
3. Hook runner is nil on the tool runtime — execute still succeeds; no panic.
4. Compactor is nil — `PRE_COMPACT` does not run; the loop still completes.
5. Compactor is set and `Compact` rewrites messages — `PRE_COMPACT` marker exists before rewrite.
6. Parent `Execute` returns (stop, guard, max-turn, or cancel) — `STOP` fires.
7. `Conversation.Close` — `STOP` then `SESSION_END` in that order.
8. `Close` with an already-canceled context (Ctrl-C) — both teardown hooks still run.
9. `stopTUILoop` still unblocks the event loop without a second SIGINT after the closer is wired.
10. Existing `SESSION_START`, `USER_PROMPT_SUBMIT`, `PRE_TOOL_USE`, and `POST_TOOL_USE` call sites remain.
11. Integration test registers recording scripts for all ten `AllEvents()` names.
12. Integration test does not require `SUBAGENT_START` or `SUBAGENT_STOP` markers.
13. Double `Close` does not panic (existing conversation test).
14. `PRE_TOOL_USE` block (exit 2) still prevents `tool.Execute` and does not fire `POST_TOOL_USE_FAILURE` as a substitute for the block path.
15. No compact pipeline package is introduced (Step 9 stays closed).

## 3. UX Implementation and Assessment

The operator-facing surface is still workspace-trusted hook scripts. Value is that every parent-side name in the spec mapping table now has a production execute.

### Time to First Value
- [ ] A `post-tool-use-failure` script in `.spin/hooks` runs after a tool error with no new flag
- [ ] `session-end` runs when the TUI tears down, including Ctrl-C

### Onboarding Clarity
- [ ] Script filenames remain `Event.ScriptName()` (`post-tool-use-failure`, `pre-compact`, `stop`, `session-end`)
- [ ] Failure vs success tool hooks are distinct names

### Production-Ready Defaults
- [ ] Nil hook runner is a no-op on every new call site
- [ ] Nil compactor skips `PRE_COMPACT`

### Golden Path Quality
- [ ] Tool error → `POST_TOOL_USE_FAILURE`
- [ ] Compact → `PRE_COMPACT` then history rewrite
- [ ] Close / TUI teardown → `STOP` then `SESSION_END`

### Decision Load
- [ ] Operators do not choose which parent events exist
- [ ] `SUBAGENT_*` remain unused until Step 19 without extra config

### Progressive Complexity
- [ ] Existing four events need no new scripts to keep working
- [ ] Child spawn hooks stay opt-in later

### Error Quality
- [ ] Tool errors still return a `ToolResult` to the model
- [ ] Hook timeout / cancel on teardown does not hang `Close`

### Failure Safety
- [ ] Canceled TUI context does not drop `SESSION_END`
- [ ] Compact hook failure does not prevent the fail-safe compact path (hook is non-blocking)

### Runtime Transparency
- [ ] Recording scripts can append each event name as it fires
- [ ] Parent-side subset is eight names; ten names may be installed

### Debuggability
- [ ] Tests wait for async markers with `Eventually`
- [ ] Compact-order test observes the marker from inside `Compact`

### Cross-Surface Consistency
- [ ] TUI teardown and `Conversation.Close` share the same `STOP`/`SESSION_END` sequence
- [ ] ACP/exec `Close` uses the same conversation method

### Workflow Consistency
- [ ] Emitters call `hooks.Runner.Execute` with the existing `Event` constants
- [ ] Builder reuses the runner already passed to the tool runtime

### Change Safety
- [ ] Exit-code-2 `PRE_TOOL_USE` block semantics are unchanged
- [ ] Step 9 compact pipeline is not started

### Experimentation Safety
- [ ] Tests use `t.TempDir` hook dirs, not the operator home tree
- [ ] Subagent manager stays a stub (no production spawn)

### Interaction Latency
- [ ] New events are non-blocking (`IsBlocking` remains false)
- [ ] Runner timeout budget is unchanged

### Developer Feedback Speed
- [ ] Package tests isolate tool failure vs compact vs close
- [ ] One integration test lists the parent-side subset

### Team Scale
- [ ] The same `ScriptName()` files work for global, project, and plugin dirs
- [ ] Org-wide `~/.spin/hooks` scripts receive the new events once builders pass the runner

### System Scale
- [ ] Extra parent events do not require a new runner type
- [ ] Child events can be added in Step 19 without renaming these emitters

### Right Behavior by Default
- [ ] Missing scripts are a no-op (`discoverScripts` empty)
- [ ] Success and failure tool hooks do not share one name

### Anti-Bypass Design
- [ ] TUI Ctrl-C cannot skip `SESSION_END` by canceling the conversation context first
- [ ] Compact cannot rewrite history before `PRE_COMPACT` scripts are started

## 4. Tests

### TC-01: tool_error_fires_post_tool_use_failure

**Given** a tool whose `Execute` returns an error and a `post-tool-use-failure` recording script.
**When** `Runtime.Execute` runs that call.
**Then** the failure marker exists and the `post-tool-use` marker does not.

### TC-02: tool_success_still_fires_post_tool_use

**Given** a successful tool and both post-tool scripts.
**When** `Runtime.Execute` runs.
**Then** `post-tool-use` exists and `post-tool-use-failure` does not.

### TC-03: nil_hook_runner_tool_error_no_panic

**Given** a nil hook runner and a failing tool.
**When** `Runtime.Execute` runs.
**Then** the call returns a tool-error result and does not panic.

### TC-04: pre_compact_before_history_rewrite

**Given** a harness executor with a hook runner and a compactor that records whether the `pre-compact` marker exists when `Compact` is entered.
**When** `Execute` runs a turn that invokes compaction.
**Then** `Compact` observed the marker and then rewrote history.

### TC-05: nil_compactor_skips_pre_compact

**Given** an executor with a hook runner and no compactor.
**When** `Execute` completes.
**Then** the `pre-compact` marker is absent and the loop still finishes.

### TC-06: parent_loop_end_fires_stop

**Given** a recording `stop` script and a harness executor.
**When** `Execute` returns (implicit completion).
**Then** the `stop` marker exists.

### TC-07: close_fires_stop_then_session_end

**Given** recording `stop` and `session-end` scripts on a conversation.
**When** `Conversation.Close` is called.
**Then** both markers exist and `stop` is recorded before `session-end`.

### TC-08: close_canceled_context_still_fires_session_end

**Given** a canceled context (Ctrl-C shape) and recording teardown scripts.
**When** `Conversation.Close` is called with that context.
**Then** `STOP` and `SESSION_END` still run.

### TC-09: stop_tui_loop_still_unblocks_without_second_signal

**Given** the existing TUI event-loop fixture.
**When** `stopTUILoop` runs with a closer that calls `Close`.
**Then** the event loop unblocks without a second SIGINT.

### TC-10: existing_session_start_and_prompt_still_fire

**Given** recording `session-start` and `user-prompt-submit` scripts.
**When** the builder fires session start and `RunTurn` runs a non-empty prompt.
**Then** both markers exist.

### TC-11: integration_registers_all_ten_asserts_parent_subset

**Given** recording scripts for every name in `hooks.AllEvents()`.
**When** the parent-side emitters run (session start, prompt, pre/post tool, tool failure, pre-compact, stop, session end) with a stub harness / stub manager.
**Then** the eight parent-side names are present. `SUBAGENT_START` and `SUBAGENT_STOP` are not required.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 8:

- Tool error path executes `POST_TOOL_USE_FAILURE`
- Compactor path executes `PRE_COMPACT` before history rewrite
- Parent loop end / `Conversation.Close` execute `STOP` then `SESSION_END`
- Existing wired events (`SESSION_START`, `USER_PROMPT_SUBMIT`, `PRE/POST_TOOL_USE`) still fire
- Integration test registers recording scripts for all ten names and asserts the parent-side subset
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 8
- Implementation files: `internal/agent/tool/runtime.go`, `internal/agent/harness/executor.go`, `internal/agent/harness/loop.go`, `internal/conversation/conversation.go`, `internal/conversation/builder.go`, `cmd/spin/tui.go`
- Test files: `internal/agent/tool/runtime_lifecycle_hooks_test.go`, `internal/agent/harness/lifecycle_hooks_test.go`, `internal/conversation/lifecycle_hooks_test.go`, `internal/conversation/parent_lifecycle_integration_test.go`, `cmd/spin/tui_quit_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md` — this journey
- `internal/agent/tool/runtime_lifecycle_hooks_test.go` — tool error fires `POST_TOOL_USE_FAILURE`; success still fires `POST_TOOL_USE`
- `internal/agent/harness/lifecycle_hooks_test.go` — `PRE_COMPACT` before rewrite; `STOP` on parent loop end
- `internal/conversation/lifecycle_hooks_test.go` — `Close` fires `STOP` then `SESSION_END`, including canceled context and idempotent Close
- `internal/conversation/parent_lifecycle_integration_test.go` — recording scripts for all ten names; asserts the parent-side eight

Files modified:
- `internal/agent/tool/runtime.go` — error path fires `POST_TOOL_USE_FAILURE`; success path still fires `POST_TOOL_USE`
- `internal/agent/harness/executor.go` — `HookRunner` interface and `WithHookRunner`
- `internal/agent/harness/loop.go` — `PRE_COMPACT` before `Compact`; `STOP` after `runLoop`
- `internal/conversation/conversation.go` — `Close` fires `STOP` then `SESSION_END` via `WithoutCancel`; Close is once-only
- `internal/conversation/builder.go` — harness executor receives the same hook runner as the tool runtime
- `cmd/spin/tui.go` — `stopTUILoop` closes the conversation on the same teardown as TUI clear
- `cmd/spin/tui_quit_test.go` — closer is invoked so `SESSION_END` can run on Ctrl-C
- `specs/agent-harness/ROADMAP.md` — Step 8 DoD ticks and traceability
- `docs/testing.md` — journey 008 integration test row
