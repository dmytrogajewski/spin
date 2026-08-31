# JOURNEY-019-subagent-hooks-and-no-silent-drop: Subagent hooks and no silent drop on spawn

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Subagent hooks and no silent drop on spawn

## 1. Journey

When **an operator has installed `subagent-start` / `subagent-stop` scripts (project, global, skill, plugin, or agent-frontmatter) and the parent is about to start a process child** I want **`SUBAGENT_START` to run as a blocking gate before any pid exists, `SUBAGENT_STOP` to run after every exit, and every hook that fired in the parent to be registered on the child** so I **can veto a spawn with exit 2, audit child lifetime on success/failure/crash, and never lose a plugin hook to a silent drop**.

## 2. CJM

Alex already has a production process executor (Step 18) that starts `spin a2a`, speaks A2A, and emits `EventSubagentSpawn` / `EventSubagentComplete`. Plugin hooks are discovered (Step 6) and the parent-side emitters exist (Step 8). `SUBAGENT_START` and `SUBAGENT_STOP` are declared and marked blocking/non-blocking in `event.go`, but the production spawn path never fires them. Child `Harness` / `Server` do not inherit parent, skill, plugin, or agent-frontmatter scripts. A missing child hook today would be a log line, not a failed test. This journey wires the two spawn events on the process start path, copies extra scripts into the child without dropping them, and keeps spawn-tool recursion deny-by-default. It does **not** add a `/tasks` registry (Step 20).

Assumption: reuse `hooks.Runner` and `PluginScripts`; do not invent a second runner. Assumption: `SUBAGENT_START` exit 2 means `StartSpec` is never called (pid 0 / nil process). Assumption: `SUBAGENT_STOP` is non-blocking and must still run when the spawn context is canceled or the child crashes. Assumption: skill and agent-frontmatter extras ride the same `PluginScript` channel as plugin hooks. Assumption: children do not receive a spawn tool unless the Spec allowlists it.

### Phase 1: Veto spawn before a process exists

**User Intent:** Block a child with a policy hook before the OS creates a pid.

**Actions:** Parent executor runs `SUBAGENT_START` (blocking) with session/workdir/spec context. Exit 2 returns a typed block error. `StartSpec` is not called.

**Pain / Risk:** Hook runs after `cmd.Start`, so a veto still leaves a pid to reap. Exit 2 is treated as a soft warning. Nil runner panics. Spawn event fires even when the child never started.

**Success Signal:** `SUBAGENT_START` exit 2 prevents process start (no pid). Nil runner is a no-op admit. Spawn events emit only after admit succeeds.

### Phase 2: Stop hook on every child lifetime

**User Intent:** See a `subagent-stop` script run whether the child succeeded, failed, or crashed.

**Actions:** After admit, the executor defers `SUBAGENT_STOP`. Success path (artifact returned), failure path (send/card error), and crash path (`/bin/sh` helper exit) all reach the defer. Context cancel uses `WithoutCancel` so the async hook still starts.

**Pain / Risk:** STOP is skipped on crash because `return` happens before the emit. STOP is skipped when `Send` fails. STOP is skipped when the parent context is already canceled. STOP fires on a vetoed spawn that never started.

**Success Signal:** `SUBAGENT_STOP` runs on success, failure, and crash. Marker files exist for each path. Vetoed admit does not fire STOP.

### Phase 3: Copy hooks into the child — no silent drop

**User Intent:** A plugin or skill hook that already fired in the parent is still registered after spawn.

**Actions:** Snapshot parent extras (`PluginScripts` plus any skill/agent-frontmatter scripts on that channel). Copy them onto the child `Harness`. A child-side runner built from the copy can write a marker file. A test asserts the copied list is non-empty and equal to the parent extras.

**Pain / Risk:** Child `NewHarness` starts with an empty extra list and never copies. Missing child hooks are only `slog.Warn`. Plugin cwd is lost so the marker is written in the wrong place. Project/global dirs are dropped so only extras survive — or extras are dropped so only dirs survive.

**Success Signal:** A skill/plugin hook that fired in the parent is registered in the child (test with a marker file). Missing child hooks is a test failure, not a log line.

### Phase 4: Deny spawn-tool recursion by default

**User Intent:** A child cannot spawn another child unless its Spec explicitly allowlists the spawn tool.

**Actions:** Builtin specs keep their existing allowlists (no spawn tool). `HasTool(ToolSpawn)` is false for nil and for every builtin. A Spec that lists the spawn tool is the only grant.

**Pain / Risk:** `HasTool` treats nil `AllowedTools` as unrestricted and grants spawn. Card skills or TaskFrame tools include spawn by accident. A later config extra inherits spawn from the parent tool registry.

**Success Signal:** Children do not get the spawn tool unless the Spec allowlists it; test deny-by-default (recursion risk).

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| `SUBAGENT_*` declared but unused on spawn | 1 | Fire start before `StartSpec` |
| Exit 2 still leaves a pid | 1 | Typed block error; no `cmd.Start` |
| STOP skipped on crash/fail | 2 | Defer stop after admit |
| Plugin extras dropped on child | 3 | `CopyScripts` onto `Harness` |
| Missing child hooks only logged | 3 | Test `require`s registered extras |
| Nil allowlist grants spawn | 4 | Spawn requires explicit allowlist |
| Child stdout / TUI mix-up | 1 | Unchanged: stdout stays the RPC pipe |
| `/tasks` registry temptation | 1–4 | Out of scope (Step 20) |

### North Star Summary

A parent spawn is a gated process: `SUBAGENT_START` can veto before a pid exists, `SUBAGENT_STOP` always records the end of a started child, and every parent/skill/plugin/agent-frontmatter extra script is copied onto the child so a marker-file test fails if any are missing. Recursion stays closed unless a Spec names the spawn tool.

### Stressors

1. `SUBAGENT_START` is fired after `cmd.Start`, so a veto still produces a pid.
2. Exit 2 is logged and ignored; spawn continues.
3. `SUBAGENT_STOP` is omitted on the crash return path.
4. `SUBAGENT_STOP` is omitted when `Send` or card handshake fails.
5. `SUBAGENT_STOP` is omitted on the success return (only failure is hooked).
6. Parent plugin extras are not copied; child `HookScripts()` is empty.
7. A missing child hook is a `slog.Warn` and the suite stays green.
8. Skill or agent-frontmatter extras are a different type and are dropped by a plugin-only copy.
9. `HasTool("spawn")` is true when `AllowedTools` is nil (unrestricted).
10. Builtin explorer/planner/reviewer/ask_user accidentally receive a spawn tool.
11. A canceled parent context prevents the async STOP hook from starting.
12. Nil hook runner panics on admit or stop.
13. STOP runs after a veto even though no child existed.
14. Builder still injects `NewExecutor` without the parent runner, so production never sees START/STOP.
15. Copy loses plugin `Cwd`, so the child marker is written outside the plugin root.

## 3. UX Implementation and Assessment

### Time to First Value
- [ ] A project `subagent-start` script that exits 2 blocks the next spawn with no extra flags
- [ ] A project `subagent-stop` script records a marker on the next successful spawn

### Onboarding Clarity
- [ ] Script names stay `subagent-start` and `subagent-stop` (no new filenames)
- [ ] Block reason from hook stdout is on the returned error

### Production-Ready Defaults
- [ ] Nil runner admits and does not fire scripts
- [ ] Children do not get the spawn tool unless the Spec allowlists it

### Golden Path Quality
- [ ] Admit → start → send → STOP → artifact on the echo child
- [ ] Plugin extra that fired in the parent fires from the child's copied runner

### Decision Load
- [ ] Operators keep using existing hook dirs and plugin packages
- [ ] No new CLI flags required for START/STOP

### Progressive Complexity
- [ ] Project-dir scripts work without plugins
- [ ] Plugin/skill extras are additive on the same runner

### Error Quality
- [ ] Veto error wraps a typed `ErrStartBlocked` plus hook reason
- [ ] Crash still returns `ErrChildCrashed` after STOP is started

### Failure Safety
- [ ] Veto leaves no pid to reap
- [ ] STOP still starts if the parent context is canceled

### Runtime Transparency
- [ ] Spawn/complete events still emit after a successful admit
- [ ] Hook scripts receive JSON `EventContext` on stdin

### Debuggability
- [ ] Marker-file tests prove registration (not a log line)
- [ ] Child `HookScripts()` is inspectable

### Cross-Surface Consistency
- [ ] TUI/exec/ACP builder injects the same parent runner into the process executor
- [ ] Event names match `event.go` (`SUBAGENT_START` / `SUBAGENT_STOP`)

### Workflow Consistency
- [ ] Reuse `hooks.Runner.Execute` and exit-2 block semantics
- [ ] Plugin extras stay `PluginScript` (Step 6)

### Change Safety
- [ ] Existing Step 18 pid/artifact/crash tests remain green
- [ ] Parent-side eight lifecycle events from Step 8 still fire

### Experimentation Safety
- [ ] Tests use temp dirs and marker files; no user hook dir is rewritten
- [ ] Veto tests never start `/bin/spin` when blocked

### Interaction Latency
- [ ] Non-blocking STOP returns immediately (async)
- [ ] Blocking START waits only for the existing runner timeout

### Developer Feedback Speed
- [ ] Missing child extras fail the test at `require`, not in CI logs
- [ ] Veto is visible as an error on `Manager.Spawn`

### Team Scale
- [ ] Project `.spin/hooks` and plugin packages remain VCS-friendly
- [ ] Copy is deterministic (order-preserving append)

### System Scale
- [ ] Extra script sources (skill, agent-frontmatter) use the same copy helper
- [ ] No second hook runner type

### Right Behavior by Default
- [ ] Spawn tool denied unless allowlisted
- [ ] Empty extras copy to empty extras (valid); expected extras must be asserted

### Anti-Bypass Design
- [ ] Exit 2 cannot be skipped by starting the process first
- [ ] A log line cannot stand in for a missing-child-hooks assertion

## 4. Tests

### TC-01: start_exit_2_no_pid

**Given** a `subagent-start` script that exits 2.
**When** `StartIfAllowed` / the process executor runs.
**Then** the error wraps the start-blocked sentinel, the process is nil or `PID() == 0`, and `StartSpec` is not reached.

### TC-02: start_allow_starts_process

**Given** a `subagent-start` script that exits 0 (or no script).
**When** `StartIfAllowed` runs against `build/bin/spin`.
**Then** `pid > 0` and the child is not the parent pid.

### TC-03: nil_runner_admits

**Given** a nil hook runner.
**When** admit / `StartIfAllowed` runs.
**Then** there is no panic and spawn may proceed.

### TC-04: stop_on_success

**Given** a `subagent-stop` recording script and a successful echo spawn.
**When** the executor returns artifact text.
**Then** the STOP marker exists.

### TC-05: stop_on_failure

**Given** a `subagent-stop` recording script and a child that fails the card/send path.
**When** the executor returns an error.
**Then** the STOP marker exists.

### TC-06: stop_on_crash

**Given** a `subagent-stop` recording script and a `/bin/sh` helper that exits non-zero.
**When** the executor / start path records a crash.
**Then** the STOP marker exists and the parent does not panic.

### TC-07: stop_not_on_veto

**Given** a blocking `subagent-start` and a recording `subagent-stop`.
**When** admit vetoes.
**Then** the STOP marker is absent.

### TC-08: plugin_hook_copied_to_child_marker

**Given** a plugin `PluginScript` that fired in the parent (parent marker exists).
**When** extras are copied onto the child `Harness` and the child's runner executes the same script name.
**Then** the child marker exists and `HookScripts()` equals the parent extras.

### TC-09: missing_child_hooks_fail_the_test

**Given** a parent extra list that is non-empty.
**When** the child registration is inspected.
**Then** an empty child list fails `require` (not a log assertion).

### TC-10: copy_preserves_cwd_and_skill_extras

**Given** parent extras that include a plugin script and a skill/frontmatter-shaped `PluginScript` with distinct `Cwd`.
**When** `CopyScripts` / inherit runs.
**Then** both entries survive with Path, Name, and Cwd intact.

### TC-11: spawn_tool_deny_by_default

**Given** every builtin Spec and a Spec with `AllowedTools == nil`.
**When** `HasTool(ToolSpawn)` and card/frame tool lists are inspected.
**Then** spawn is absent unless a Spec lists it explicitly.

### TC-12: builder_wires_parent_runner

**Given** a project `subagent-start` that exits 2 on the builder work dir.
**When** `Manager.Spawn` runs through the production executor.
**Then** spawn fails with the start-blocked error and no child pid is observed.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 19:

- `SUBAGENT_START` exit 2 prevents process start (no pid)
- `SUBAGENT_STOP` runs on success, failure, and crash
- A skill/plugin hook that fired in the parent is registered in the child (test with a marker file)
- Missing child hooks is a test failure, not a log line
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 19
- Implementation files: `internal/agent/child/spawn_hooks.go`, `internal/agent/child/spawn_exec.go`, `internal/agent/child/harness.go`, `internal/safety/hooks/inherit.go`, `internal/conversation/builder.go`, `internal/agent/subagent/spec.go`
- Test files: `internal/agent/child/spawn_hooks_test.go`, `internal/agent/child/hooks_inherit_test.go`, `internal/agent/child/card_test.go`, `internal/safety/hooks/inherit_test.go`, `internal/agent/subagent/spec_test.go`, `internal/conversation/subagents_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md` — this journey
- `internal/agent/child/spawn_hooks.go` — `StartIfAllowed`, `ErrStartBlocked`, admit + STOP
- `internal/agent/child/spawn_hooks_test.go` — veto no pid; STOP on success/failure/crash; no STOP on veto
- `internal/agent/child/hooks_inherit_test.go` — parent plugin extra copied; child marker; missing extras fail `require`
- `internal/safety/hooks/inherit.go` — `CopyScripts`, `Runner.PluginScripts`
- `internal/safety/hooks/inherit_test.go` — extras preserved including skill-shaped `Cwd`
- `internal/agent/subagent/spec_test.go` — spawn tool deny-by-default and explicit allowlist

Files modified:
- `internal/agent/child/spawn_exec.go` — admit before start; defer STOP; hook runner on `NewExecutor`
- `internal/agent/child/harness.go` — `InheritHookScripts` / `HookScripts` (no silent drop)
- `internal/agent/child/card_test.go` — card/frame tools omit spawn for builtins
- `internal/agent/subagent/spec.go` — `ToolSpawn` requires explicit allowlist
- `internal/conversation/builder.go` — process executor receives the parent hook runner
- `internal/conversation/subagents_test.go` — builder spawn veto
- `internal/agent/child/spawn_exec_test.go` — `NewExecutor` runner argument
- `docs/testing.md` — journey 019 row
- `specs/agent-harness/ROADMAP.md` — Step 19 DoD and traceability
