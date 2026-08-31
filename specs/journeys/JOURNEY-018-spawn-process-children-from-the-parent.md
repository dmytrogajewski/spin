# JOURNEY-018-spawn-process-children-from-the-parent: Spawn process children from the parent (replace the stub)

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Spawn process children from the parent (replace the stub)

## 1. Journey

When **the parent harness needs a specialized worker (explorer, planner, reviewer, or a config-registered spec)** I want **the production executor to start `spin a2a` as an OS process, speak A2A over the child's stdio, wait for a completed or failed Task, and return the artifact text** so I **get crash-isolated children, a real pid, and a summary I can treat as a tool result — without the old `ErrSubagentSpawnNotSupported` stub and without mixing child logs into the TUI stdout**.

## 2. CJM

Alex already has a child-side `spin a2a` server (Step 17) that announces an Agent Card and answers `message/send` with the in-memory echo Task. The parent still injects a stub executor that always returns `ErrSubagentSpawnNotSupported`. Admission control (`Manager` + `DefaultMaxConcurrent`) and the injectable `Executor` test double already exist. This journey replaces only the production executor: spawn an OS process, blocking `message/send` until Task completed/failed, artifact summary back to the parent, spawn/complete events, and config `Subagents` extras on the Manager. It does **not** add `SUBAGENT_START` veto (Step 19) and does **not** add a `/tasks` registry (Step 20).

Assumption: Step 17 `MemoryHandler` echo Task is a valid blocking-send peer for this step — a full child ReAct loop is out of scope. Assumption: `subagent.Executor` stays the test double; production builder injects the process executor. Assumption: `DefaultMaxConcurrent` remains the admission semaphore. Assumption: integration tests exec `build/bin/spin` (`make test` already builds it) or a helper binary. Assumption: child stdout is the A2A RPC stream only; logs go to a captured stderr, never the parent's TUI stdout.

### Phase 1: Admit and start a child process

**User Intent:** Ask the parent to spawn a named spec and get a real isolated process, not a goroutine stub.

**Actions:** Parent `Manager.Spawn` acquires a semaphore slot, then the production executor starts `spin a2a --spec <name> --stdio`. The process is an extra OS pid.

**Pain / Risk:** Stub still returns `ErrSubagentSpawnNotSupported`. Spawn is an in-process goroutine with pid 0. Child logs write to stdout and corrupt the TUI or the RPC stream. Tests leave orphans because nothing kills on cleanup.

**Success Signal:** Spawn is an OS process (`pid > 0`, not the parent pid). Builder no longer returns `ErrSubagentSpawnNotSupported` on a real spawn. `Executor` remains injectable for unit tests. Semaphore still caps concurrency.

### Phase 2: Blocking send and artifact summary

**User Intent:** Wait for the child's work to finish and read a single summary, not a raw transcript.

**Actions:** Parent A2A client sends `message/send` (blocking). If the Task is working, poll `tasks/get` until completed or failed. Return the artifact text to the caller.

**Pain / Risk:** Send returns immediately with a working Task and no wait. Artifact text is empty or the whole child transcript. Client hangs forever if the child never reaches a terminal state. Stdio mix-up makes `NewClient` fail with `ErrUnexpectedCard`.

**Success Signal:** Blocking send waits for Task completed/failed and returns artifact text. Echo child (Step 17) yields the sent query as the summary. Tests use a timeout and `t.Cleanup` kill.

### Phase 3: Survive a child crash

**User Intent:** A dead child must not take down the parent harness.

**Actions:** Child exits or is killed before or during send. Parent maps that to Task `failed` plus a stderr artifact, emits completion, and stays alive for another spawn.

**Pain / Risk:** Parent panics or the conversation becomes unusable. Crash is reported as a generic I/O error with no stderr. Failed Task is never recorded. Unreaped zombies accumulate.

**Success Signal:** Child crash → Task `failed` + stderr artifact; parent harness survives (second spawn or a later call does not panic). Process is reaped on close.

### Phase 4: Observe spawn and register extras

**User Intent:** See that a child started and finished, and attach extra specs from config without code changes.

**Actions:** Subscribe to the parent emitter. Spawn. Inspect `EventSubagentSpawn` / `EventSubagentComplete`. Put an extra name in config `Subagents` and Build.

**Pain / Risk:** Events exist as types but never emit on the production path. Config map is parsed but never `Register`ed. Extra spec overwrites a builtin and drops its allowlist/prompt. Hook veto is added here (Step 19 leak).

**Success Signal:** Both events emit with agent type and query/summary. Config `Subagents` map registers extra specs on the Manager. Builtin overlays keep identity and only apply model / max iterations.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Production executor is a stub error | 1 | Process executor starts `spin a2a` |
| In-process goroutine has no pid | 1 | OS process; test asserts `pid > 0` |
| No wait on Task state | 2 | Blocking send + `tasks/get` until terminal |
| Child crash kills or bricks the parent | 3 | Failed Task + stderr artifact; parent lives |
| Events defined, never fired on spawn | 4 | Emit spawn/complete around the process |
| Config `Subagents` unused at Build | 4 | Register extras (and overlay builtins) |
| Child logs on TUI stdout | 1 | Stdout is the RPC pipe; stderr is captured |
| Tests hang on a live child | 2 | Low MaxIterations on extras; timeout; kill on cleanup |

### North Star Summary

The parent starts a real `spin a2a` child, talks A2A on that child's stdio, waits for a completed or failed Task, and hands back artifact text. A crash becomes a failed Task with stderr, not a parent panic. Spawn and complete events fire. Config can register extra specs. Tests still inject `Executor`. Admission stays on the semaphore. Hooks and `/tasks` wait for later steps.

### Stressors

1. Builder still returns `ErrSubagentSpawnNotSupported` so the TUI/exec path never starts a child.
2. "Spawn" is a goroutine with `pid == 0` or the parent pid, violating process isolation.
3. Child slog/help lands on stdout; `a2a.NewClient` fails with `ErrUnexpectedCard` or the TUI prints RPC frames.
4. Blocking send returns a working Task and never polls `tasks/get`.
5. Child crash panics the parent or leaves the Manager unusable for the next spawn.
6. Crash path returns an I/O error with empty artifacts and no stderr.
7. `EventSubagentSpawn` / `EventSubagentComplete` stay unused on the production executor.
8. Config `Subagents` extras are validated but never registered, so `Manager.Spec` is nil.
9. Overlaying a builtin from config drops `AllowedTools` / `SystemPrompt`.
10. Integration test mocks the process and never execs `build/bin/spin` or a helper binary.
11. Tests spawn forever: no timeout, no `t.Cleanup` kill, child `MaxIterations` unbounded.
12. `Executor` is removed or hardcoded so existing Manager unit tests cannot inject a double.
13. Semaphore is bypassed so more than `DefaultMaxConcurrent` children start.
14. `SUBAGENT_START` veto or `/tasks` registry is implemented here (scope leak).
15. Child inherits the parent's stdout/stderr file descriptors instead of pipes/buffers.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] First successful spawn returns artifact text from a completed Task
- [x] No extra CLI beyond the parent already knowing the spec name

### Onboarding Clarity
- [x] Stub error is gone on a real spawn; failures name the child or the binary
- [x] Unknown spec still fails at Manager lookup before exec

### Production-Ready Defaults
- [x] Production builder injects the process executor without operator config
- [x] `DefaultMaxConcurrent` remains admission control

### Golden Path Quality
- [x] `Spawn(explorer, query)` starts `spin a2a`, sends, returns echo artifact text
- [x] Integration uses `build/bin/spin` or a helper binary

### Decision Load
- [x] Operator does not choose a transport; stdio is the child binding
- [x] Extra specs come from config `Subagents` when present

### Progressive Complexity
- [x] Builtins work with zero `Subagents` config
- [x] Extra names and model/iteration overlays are opt-in

### Error Quality
- [x] Child crash surfaces Task `failed` and stderr text
- [x] Missing binary is a start error, not the old stub sentinel

### Failure Safety
- [x] Parent harness survives a child crash
- [x] Cleanup kills the child so tests do not leak processes

### Runtime Transparency
- [x] `EventSubagentSpawn` carries agent type and query
- [x] `EventSubagentComplete` carries agent type and summary

### Debuggability
- [x] Child stderr is captured as an artifact on crash
- [x] Child stdout is the A2A stream, inspectable as NDJSON-RPC

### Cross-Surface Consistency
- [x] Methods stay Step 16/17 (`message/send`, `tasks/get`)
- [x] Spec names match builtins and config map keys

### Workflow Consistency
- [x] `Executor` remains the injectable test double
- [x] Manager semaphore is unchanged

### Change Safety
- [x] No `SUBAGENT_*` hook veto
- [x] No `/tasks` registry or non-blocking wait API

### Experimentation Safety
- [x] Unit tests keep using an in-process `Executor`
- [x] Process tests kill on `t.Cleanup`

### Interaction Latency
- [x] Echo child returns on the completed Task without a parent ReAct loop
- [x] Wait loop honors context cancel

### Developer Feedback Speed
- [x] Failed start/card/send errors wrap the underlying cause
- [x] Events arrive around the process lifetime, not after the parent exits

### Team Scale
- [x] Config `Subagents` is the shared way to register extras
- [x] Journey + roadmap name the implementation files

### System Scale
- [x] New extras in the map appear on the next Build
- [x] Process spawn is one executor; Manager stays admission control

### Right Behavior by Default
- [x] Real spawn is a process; tests can still inject a double
- [x] Child logs do not hit TUI stdout

### Anti-Bypass Design
- [x] Tests assert `pid > 0` / extra process
- [x] Tests fail if the stub sentinel is returned on a real spawn

## 4. Tests

### TC-01: Start is an OS process

**Given** `build/bin/spin` (or a helper) and spec `explorer`.
**When** `StartSpec`.
**Then** `pid > 0` and the pid is not the parent pid.

### TC-02: Builder does not return the stub

**Given** a built conversation with the production executor and a resolvable spin binary.
**When** `GetSubagentManager().Spawn(explorer, query)`.
**Then** the error is not `ErrSubagentSpawnNotSupported`.

### TC-03: Blocking send returns artifact text

**Given** a started `spin a2a --spec explorer --stdio` child.
**When** the parent sends a user text part and waits for a terminal Task.
**Then** the Task is completed and the returned summary is the artifact text (echo of the query).

### TC-04: Wait for working then terminal

**Given** a Task that is not yet terminal.
**When** Send polls `tasks/get`.
**Then** it returns only after completed or failed (or ctx cancel), never a working Task as success.

### TC-05: Child crash failed + stderr

**Given** a child that writes stderr and exits non-zero before serving a card (helper/`sh`).
**When** Start/Send runs.
**Then** Task state is `failed`, an artifact contains the stderr text, and the parent does not panic.

### TC-06: Parent survives a crash

**Given** a crash result from TC-05.
**When** the parent starts or sends again (or Close then a new Start).
**Then** the second call does not panic and the Manager can be used again.

### TC-07: Events emit

**Given** a subscribed emitter and the production process executor.
**When** Spawn runs to completion (or crash).
**Then** `EventSubagentSpawn` and `EventSubagentComplete` are received.

### TC-08: Config extra specs register

**Given** `cfg.Subagents["research"]` with model and max iterations.
**When** the builder creates the Manager.
**Then** `Spec("research")` is non-nil with those overlays.

### TC-09: Config overlay keeps builtin residue

**Given** `cfg.Subagents["explorer"]` with only model / max iterations.
**When** the builder registers config specs.
**Then** explorer still has its allowlist and system prompt; model and max iterations come from config.

### TC-10: Integration exec of spin

**Given** `make test` has built `build/bin/spin`.
**When** the integration test starts that binary (or a helper) as the child.
**Then** card + blocking send succeed; the test does not replace the process with a pure mock.

### TC-11: Executor stays a test double

**Given** `NewManager(echoExecutor, …)`.
**When** existing Manager unit tests Spawn.
**Then** they still pass without starting an OS process.

### TC-12: Child stdio is not TUI stdout

**Given** a started Process.
**When** the child's stdout file is inspected.
**Then** it is the RPC pipe, not `os.Stdout`.

### TC-13: Cleanup kills the child

**Given** a started process.
**When** `t.Cleanup` / `Close` runs.
**Then** the process is reaped (Wait returns) and no orphan is left for the test.

## Acceptance Criteria

- [x] Builder no longer returns `ErrSubagentSpawnNotSupported` on a real spawn
- [x] Spawn is an OS process (test asserts `pid > 0` / extra process)
- [x] Blocking send waits for Task completed/failed and returns artifact text
- [x] Child crash → Task `failed` + stderr artifact, parent harness survives
- [x] Events `EventSubagentSpawn` / `EventSubagentComplete` emit
- [x] Config `Subagents` map can register extra specs
- [x] Integration test uses the built `build/bin/spin` or a test helper binary
- [x] `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 18
- Implementation files: `internal/agent/child/spawn.go`, `internal/agent/child/spawn_send.go`, `internal/agent/child/spawn_exec.go`, `internal/agent/child/spawn_bin.go`, `internal/conversation/builder.go`, `internal/agent/subagent/manager.go`
- Test files: `internal/agent/child/spawn_test.go`, `internal/agent/child/spawn_exec_test.go`, `internal/conversation/subagents_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-018-spawn-process-children-from-the-parent.md` — this journey
- `internal/agent/child/spawn.go` — OS process start, pid, pipe bind, Close
- `internal/agent/child/spawn_send.go` — blocking `message/send` + `tasks/get` wait, crash Task
- `internal/agent/child/spawn_exec.go` — production `Executor` + spawn/complete events
- `internal/agent/child/spawn_bin.go` — `SPIN_BIN` / `build/bin/spin` resolve
- `internal/agent/child/spawn_test.go` — pid, artifact, crash+survive, stdout isolation
- `internal/agent/child/spawn_exec_test.go` — `EventSubagentSpawn` / `EventSubagentComplete`
- `internal/conversation/subagents_test.go` — config extras, overlay residue, builder spawn

Files modified:
- `internal/conversation/builder.go` — process executor; register `cfg.Subagents`
- `internal/agent/subagent/manager.go` — admission comment (`Executor` stays the test double)
- `docs/testing.md` — journey 018 row
- `specs/agent-harness/ROADMAP.md` — Step 18 DoD and traceability

Deviation: the child still answers `message/send` via Step 17 `MemoryHandler` (echo Task). A full child ReAct loop is out of scope. Crash coverage uses `/bin/sh` as the helper binary; the happy path execs `build/bin/spin`. `ErrSubagentSpawnNotSupported` remains exported so callers can assert it is no longer returned. No `SUBAGENT_*` veto and no `/tasks` registry.
