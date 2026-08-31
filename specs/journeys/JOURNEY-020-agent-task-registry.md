# JOURNEY-020-agent-task-registry: Agent task registry — wait, list, cancel, persist

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Agent task registry — wait, list, cancel, persist

## 1. Journey

When **the parent harness starts an A2A child in the background** I want to **keep chatting, list running agent tasks, wait on a task id later in the same session, and cancel a stuck child** so I **can fan out work without hanging the ReAct loop and without losing the registry across a new parent turn**.

## 2. CJM

Alex already has blocking process spawn (Step 18): `Manager.Spawn` acquires the admission semaphore, the production executor starts `spin a2a`, and `Process.Send` waits for a completed or failed Task. Shell background (`start_process` / `list_processes`) is a separate family and stays that way until Step 21. There is no parent-side A2A registry, no non-blocking `message/send`, and no `/tasks` commands. This journey adds `internal/agent/tasks` so a non-blocking spawn returns a task id immediately, persists records on the session, and exposes wait / list / cancel through tools and slash commands. It does **not** merge shell+A2A views (Step 21) and does **not** add TUI mapper blocks (Step 23).

Assumption: `a2a.Client.SendMessageImmediate` (`returnImmediately`) is the non-blocking send; MemoryHandler already returns `TASK_STATE_WORKING` for that path. Assumption: the parent registry is the `/tasks` source of truth (id, spec, state) — child's `tasks/list` is not the operator view. Assumption: wait polls or joins the live handle and must never call `semaphore.Acquire`. Assumption: persist writes records onto `session.Metadata` so a later parent turn in the same session can `Wait` the same id. Assumption: cancel is `tasks/cancel` then SIGTERM, in that order.

### Phase 1: Non-blocking spawn returns a task id

**User Intent:** Start a child and keep the parent turn moving.

**Actions:** Parent admits the spec, starts `spin a2a`, calls `message/send` with `returnImmediately`, registers id/spec/state, returns the id. ReAct continues without waiting for a terminal Task.

**Pain / Risk:** Spawn still uses blocking `Process.Send`, so the parent loop hangs. Immediate send is implemented but never registered. Task id is empty or reused. Semaphore is held on the ReAct goroutine until the child finishes, so the parent cannot continue. Tests treat a working MemoryHandler Task as completed.

**Success Signal:** Non-blocking spawn returns a non-empty task id while the Task is still working. The caller’s next statement runs without waiting for completed/failed/canceled.

### Phase 2: List, wait, and cancel

**User Intent:** See what is running, join a task later, or stop it.

**Actions:** `/tasks` (and the list tool) prints id, spec, state. `/task wait <id>` blocks until completed, failed, or canceled, or until the context is canceled. `/task cancel <id>` calls `tasks/cancel` then SIGTERM.

**Pain / Risk:** List includes shell `start_process` rows (Step 21 leak). Wait acquires the spawn semaphore and deadlocks when slots are full. Cancel only kills, or only RPCs, or uses SIGKILL first. Unknown id is a silent no-op. Context cancel is ignored and wait hangs.

**Success Signal:** List shows only A2A registry rows with id, spec, and state. Wait returns the terminal record or `ctx.Err()`. Cancel invokes `tasks/cancel` then SIGTERM. A dedicated test holds every semaphore slot and still completes `Wait`.

### Phase 3: Persist across a new parent turn

**User Intent:** Start work in one turn; wait in the next turn of the same session.

**Actions:** Registry mutations write a snapshot onto the session. A later `IncrementTurnCount` (or a registry restored from that session) can still `Wait` the same id.

**Pain / Risk:** Registry is a process-local map dropped when the conversation rebuilds from session metadata. Snapshot omits spec or state. Restore creates rows that `Wait` cannot find. Bind races with `Session` mutex. JSON load of old sessions without `agent_tasks` panics.

**Success Signal:** After a new parent turn in the same session, `Wait` finds the id. Restoring from session metadata lists the same id, spec, and state. Missing `agent_tasks` is an empty registry.

### Phase 4: Tools and slash commands on the parent surfaces

**User Intent:** Use the same operations from the command line and from the model.

**Actions:** Register `/tasks` and `/task`. Help lists them. Tools wrap list / wait / cancel against the same registry. Conversation owns the registry and exposes it to TUI `CommandContext` via an optional `TaskSource` (same pattern as `SessionBrowser`).

**Pain / Risk:** Commands are registered but never wired to a live registry. Tools talk to `BackgroundTaskManager` (shell) instead of the A2A registry. `CommandContext` grows a required method and breaks ACP/TUI mocks. `/task wait` lowercases ids incorrectly for ids that are already lowercase (`task-1`). Help omits the new commands.

**Success Signal:** `/tasks` lists id, spec, state. `/task wait` and `/task cancel` operate on that registry. Tools hit the same store. Shell `list_processes` is unchanged.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Blocking `Send` hangs ReAct | 1 | `SendMessageImmediate` + register + return id |
| No parent-side A2A list | 2 | Registry `List` for `/tasks` only |
| Wait on a full semaphore deadlocks | 2 | Wait outside admission; deadlock test documents it |
| Cancel is Kill-only | 2 | `tasks/cancel` then SIGTERM |
| Registry dies on the next turn | 3 | Persist on `session.Metadata` |
| Shell and A2A mixed early | 2 | Separate view until Step 21 |
| Commands have no store | 4 | Optional `TaskSource` like `SessionBrowser` |
| Tools would reuse `list_processes` | 4 | Dedicated agent-task tools on the A2A registry |

### North Star Summary

The parent can start an A2A child without blocking the ReAct loop, keep the task id in a session-backed registry, list id/spec/state, wait without taking an admission slot, and cancel via `tasks/cancel` plus SIGTERM. Shell background stays on `start_process` / `list_processes`. A later parent turn in the same session can still wait.

### Stressors

1. Non-blocking spawn still waits for a terminal Task, so the parent ReAct loop never continues.
2. Immediate send succeeds but the id is not registered, so `/task wait` cannot find it.
3. `/task wait` calls `semaphore.Acquire` and deadlocks when every slot is held by a running child.
4. `/task cancel` sends SIGKILL (or Kill) without `tasks/cancel`, so the child never sees a canceled Task.
5. `/task cancel` RPCs cancel but never SIGTERM, leaving a live `spin a2a` process.
6. `/tasks` includes shell `start_process` rows and collides with Step 21’s mixed view.
7. Registry lives only on the Manager and vanishes when the next parent turn rebuilds from the session.
8. Session JSON from before this field lacks `agent_tasks` and restore panics or drops the session.
9. Wait ignores `ctx.Done()` and hangs after the operator cancels the turn.
10. Unknown task id is swallowed; the operator cannot tell wait/cancel failed.
11. CommandContext gains a required method and ACP/TUI test mocks fail to compile.
12. `ParseCommand` lowercases args; a mixed-case id would not match if we ever emit one.
13. Background spawn holds the ReAct lock (not only the semaphore) so the parent cannot emit the next tool result.
14. Persist writes a snapshot without the live handle; Wait on a still-working restored row has nothing to poll (same-session turn must keep the in-memory handle).

## 3. UX Implementation and Assessment

### Time to First Value
- [ ] Operator can start a background child and see a task id without waiting for completion
- [ ] `/tasks` shows that id on the same turn

### Onboarding Clarity
- [ ] `/help` lists `/tasks` and `/task wait` / `/task cancel`
- [ ] Unknown id errors name the missing task

### Production-Ready Defaults
- [ ] Empty registry lists as empty, not an error
- [ ] No config flag is required to use the registry

### Golden Path Quality
- [ ] Non-blocking spawn → list → wait (or cancel) works end-to-end
- [ ] Wait returns completed/failed/canceled state

### Decision Load
- [ ] One list command; wait and cancel take only an id
- [ ] No kind=agent|shell choice yet (Step 21)

### Progressive Complexity
- [ ] Blocking `Manager.Spawn` remains for callers that want a summary now
- [ ] Background spawn is the opt-in path

### Error Quality
- [ ] Missing registry / missing id names the problem
- [ ] Context cancel surfaces as `ctx.Err()`

### Failure Safety
- [ ] Cancel is recoverable (list shows canceled)
- [ ] Cancel is explicit; list and wait are read/join only

### Runtime Transparency
- [ ] `/tasks` shows id, spec, state
- [ ] No silent registry mutations off the persist path

### Debuggability
- [ ] Task id from spawn is the same id in list and wait
- [ ] Session snapshot is inspectable on `Metadata`

### Cross-Surface Consistency
- [ ] Tools and slash commands read the same registry
- [ ] Terminology is task id / spec / state everywhere

### Workflow Consistency
- [ ] Slash commands follow `/skills` and optional-interface `/resume`
- [ ] A2A methods stay on `internal/protocol/a2a`

### Change Safety
- [ ] Session JSON without `agent_tasks` stays valid
- [ ] Shell process tools are not rewritten

### Experimentation Safety
- [ ] Blocking spawn still works if background is unused
- [ ] Registry bind is reversible (empty snapshot)

### Interaction Latency
- [ ] Non-blocking spawn returns after `message/send` immediate, not after child completion
- [ ] List does not wait on children

### Developer Feedback Speed
- [ ] Wait reports terminal state when it happens
- [ ] Cancel returns after RPC + SIGTERM, not after reap-only

### Team Scale
- [ ] Session snapshot is JSON and can be inspected in storage
- [ ] Commands are registered globally like the other slash commands

### System Scale
- [ ] Registry is a small map keyed by id
- [ ] Admission stays on the existing semaphore

### Right Behavior by Default
- [ ] Wait does not take an admission slot
- [ ] Cancel uses `tasks/cancel` before SIGTERM

### Anti-Bypass Design
- [ ] Deadlock test fails if Wait acquires the semaphore
- [ ] Persist test fails if a new turn cannot Wait the same id

## 4. Tests

### TC-01: non_blocking_spawn_returns_id

**Given** a starter that performs immediate `message/send`.
**When** `SpawnBackground` runs.
**Then** a non-empty task id is returned before the Task is terminal, and the caller continues.

### TC-02: list_id_spec_state

**Given** a registered working task.
**When** `List` / `/tasks` runs.
**Then** the row contains id, spec, and state, and no shell process columns.

### TC-03: wait_until_terminal

**Given** a working task whose handle later reports completed.
**When** `/task wait <id>` runs.
**Then** it returns only after the state is completed (or failed/canceled).

### TC-04: wait_ctx_cancel

**Given** a handle that stays working.
**When** the wait context is canceled.
**Then** Wait returns `ctx.Err()`.

### TC-05: cancel_rpc_then_sigterm

**Given** a working task with a handle that records call order.
**When** `/task cancel <id>` runs.
**Then** `tasks/cancel` runs first and SIGTERM runs second; state is canceled.

### TC-06: persist_new_parent_turn

**Given** a registry bound to a session with a registered task.
**When** the session increments the turn count (new parent turn).
**Then** Wait on the same id succeeds.

### TC-07: restore_from_session_metadata

**Given** a registry that persisted records onto session metadata.
**When** `Restore` builds a new registry from that session.
**Then** List contains the same id, spec, and state.

### TC-08: wait_outside_semaphore

**Given** a semaphore filled to capacity.
**When** Wait runs on a registered task.
**Then** Wait completes without acquiring the semaphore (documented in the test).

### TC-09: unknown_id

**Given** an empty or unrelated registry.
**When** wait or cancel is invoked with an unknown id.
**Then** a typed not-found error is returned.

### TC-10: send_immediate_working_task

**Given** a live `spin a2a` child (or MemoryHandler peer).
**When** `SendImmediate` runs.
**Then** the Task id is set and state is working, not completed.

### TC-11: tools_share_registry

**Given** the same registry used by commands.
**When** list / wait / cancel tools execute.
**Then** they observe the same rows and do not call `list_processes`.

### TC-12: old_session_without_agent_tasks

**Given** session metadata with a nil `AgentTasks` slice.
**When** Restore runs.
**Then** List is empty and no panic occurs.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 20:

- Non-blocking spawn returns a task id; parent ReAct loop continues
- `/task wait <id>` blocks until completed/failed/canceled or ctx cancel
- `/task cancel <id>` maps to `tasks/cancel` then SIGTERM
- `/tasks` lists id, spec, state
- Registry survives a new parent turn in the same session
- Wait does not hold the ReAct lock in a way that deadlocks the semaphore (test)
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 20
- Implementation files: `internal/agent/tasks/registry.go`, `internal/commands/tasks.go`, `internal/session/metadata.go`, `internal/tools/agent_tasks.go`, `internal/agent/child/handle.go`, `internal/agent/subagent/manager.go`
- Test files: `internal/agent/tasks/registry_test.go`, `internal/agent/tasks/persist_test.go`, `internal/agent/tasks/deadlock_test.go`, `internal/commands/tasks_test.go`, `internal/tools/agent_tasks_test.go`, `internal/agent/child/handle_test.go`, `internal/agent/subagent/background_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-020-agent-task-registry.md` — this journey
- `internal/agent/tasks/registry.go` — Register, List, Wait, Cancel, persist Bind/Restore
- `internal/agent/tasks/registry_test.go` — list, wait, cancel order
- `internal/agent/tasks/format.go` — id / spec / state lines
- `internal/agent/tasks/format_test.go` — format contents
- `internal/agent/tasks/persist_test.go` — new parent turn; Restore; empty AgentTasks
- `internal/agent/tasks/deadlock_test.go` — Wait does not acquire the spawn semaphore
- `internal/agent/child/handle.go` — TaskHandle: tasks/get, tasks/cancel, SIGTERM
- `internal/agent/child/handle_test.go` — live child cancel then SIGTERM
- `internal/agent/child/immediate.go` — ImmediateStarter (non-blocking message/send)
- `internal/agent/child/immediate_test.go` — working task id
- `internal/agent/subagent/background_test.go` — SpawnBackground returns id without waiting
- `internal/commands/tasks.go` — `/tasks`, `/task wait`, `/task cancel`
- `internal/commands/tasks_test.go` — list, wait, cancel, ctx cancel
- `internal/tools/agent_tasks.go` — list_agent_tasks, wait_agent_task, cancel_agent_task
- `internal/tools/agent_tasks_test.go` — same registry as commands

Files modified:
- `internal/agent/child/spawn_send.go` — SendImmediate, GetTask, CancelTask
- `internal/agent/child/spawn.go` — SignalTERM
- `internal/agent/child/spawn_test.go` — SendImmediate working id
- `internal/agent/subagent/manager.go` — SpawnBackground, SetBackgroundStarter
- `internal/session/metadata.go` — AgentTask snapshot on Metadata
- `internal/session/metadata_test.go` — AgentTasks default empty
- `internal/commands/commands.go` — register `/tasks` `/task`; help examples
- `internal/commands/commands_test.go` — help and list include `/tasks`
- `internal/tools/classification.go` — classify new tools
- `internal/conversation/conversation.go` — taskRegistry field and getter
- `internal/conversation/builder.go` — Restore, ImmediateStarter, RegisterAgentTaskTools
- `internal/protocol/acp/command_integration_test.go` — available commands include `/tasks`
- `cmd/spin/tui_command_context.go` — TaskSource
- `cmd/spin/tui_resume_test.go` — AgentTasks non-nil
- `docs/testing.md` — journey 020 row
- `specs/agent-harness/ROADMAP.md` — Step 20 DoD and traceability
