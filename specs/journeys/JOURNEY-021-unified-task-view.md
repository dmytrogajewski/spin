# JOURNEY-021-unified-task-view: Unified task view (A2A + shell background)

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Unified task view (A2A + shell background)

## 1. Journey

When **the operator has both A2A children and shell `start_process` jobs in flight** I want to **see both families in one `/tasks` list, each tagged `kind=agent|shell`** so I **can tell which row is which, cancel the right process, and keep `list_processes` / `kill_process` working unchanged**.

## 2. CJM

Alex already has two independent stores: the Step 20 A2A registry (`internal/agent/tasks.Registry`) and the existing shell `BackgroundTaskManager` behind `start_process` / `list_processes` / `kill_process`. `/tasks` lists only A2A rows. Shell jobs are invisible on that command. This journey adds a **view** that reads both stores and prints one list. It does **not** merge the implementations into a single registry, does **not** add `navigate` (Step 22), and does **not** add TUI mapper blocks (Step 23).

Assumption: one view, two kinds — `Merge` / `FormatView` compose snapshots; `Registry` and `BackgroundTaskManager` stay separate. Assumption: typed ids (`agent:<id>`, `shell:<id>`) prevent 7-hex shell ids from colliding with A2A ids in the unified list and in cancel routing. Assumption: `/task cancel` on a shell row calls the existing Kill path (SIGTERM then SIGKILL). Assumption: `list_processes` and `kill_process` keep talking only to `TaskManager`. Assumption: `/task wait` remains the A2A join; wait on a shell row is out of scope.

### Phase 1: See both families in one list

**User Intent:** Open `/tasks` and see every in-flight job, labeled by kind.

**Actions:** Operator (or the model via the same command path) runs `/tasks`. The view lists A2A records as `kind=agent` and shell snapshots as `kind=shell`. Empty stores print an empty unified message.

**Pain / Risk:** List still hides shell rows. Implementations are merged into one map and shell Kill semantics change. Display ids collide (`abc1234` as both a 7-hex shell id and an A2A id) so the operator cannot tell which row to cancel. Format drops spec/command or state. `list_processes` is rewritten to include A2A rows.

**Success Signal:** `/tasks` output contains `kind=agent` and `kind=shell` when both families have rows. Typed ids differ when raw ids match. `list_processes` still lists only shell snapshots.

### Phase 2: Cancel the right family

**User Intent:** Stop a stuck shell job or a stuck A2A child from the same command.

**Actions:** `/task cancel <id>` parses a typed id (`shell:…` / `agent:…`) or an untyped id that exists in only one store. Shell cancel calls Kill (SIGTERM then SIGKILL). Agent cancel stays `tasks/cancel` then SIGTERM.

**Pain / Risk:** Cancel always hits the A2A registry, so a shell row is “not found”. Cancel of a shell row uses A2A RPC. Untyped collision silently picks one family. Kill is skipped or replaced with SIGKILL-only. `kill_process` breaks because the manager was replaced.

**Success Signal:** Cancel on `shell:<id>` invokes Kill with the raw 7-hex id. Cancel on `agent:<id>` (or a unique untyped A2A id) still runs cancel-then-SIGTERM. `kill_process` still kills by raw shell id.

### Phase 3: Mixed namespace is testable and stable

**User Intent:** Trust the list when both stores use similar-looking ids.

**Actions:** Tests register an A2A row whose id is 7-hex and a shell row with the same raw id. The view shows two rows. Untyped cancel of that id is rejected as ambiguous.

**Pain / Risk:** Tests only cover one family. Prefix is applied to the shell store itself, breaking `list_processes` / `kill_process` callers that expect 7-hex. Prefix is applied only in docs, not in the view.

**Success Signal:** Mixed-list tests assert two rows, two kinds, two typed ids. Process tools still use unprefixed 7-hex ids.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| `/tasks` hides shell jobs | 1 | View merges snapshots, not stores |
| Two lists, two vocabularies | 1 | One list, `kind=agent\|shell` |
| 7-hex vs A2A id collision | 1, 3 | Typed ids in the view and cancel |
| Cancel always A2A | 2 | Route shell rows to Kill |
| Fear of breaking process tools | 2 | Leave `list_processes` / `kill_process` on `TaskManager` |
| Wait scope creep | 2 | Wait stays A2A-only |

### North Star Summary

The operator sees A2A children and shell background processes in one `/tasks` list, each row tagged `kind=agent` or `kind=shell` with a typed id. Cancel on a shell row still SIGTERM/SIGKILL. `list_processes` and `kill_process` keep working for shell. The two implementations remain two implementations.

### Stressors

1. `/tasks` still prints only A2A rows, so shell `start_process` jobs stay invisible.
2. Someone merges `Registry` and `BackgroundTaskManager` into one map and shell Kill becomes A2A cancel.
3. A 7-hex shell id equals an A2A id; the list shows one row and cancel hits the wrong family.
4. Typed prefix is written into `BackgroundTaskManager` ids, so `kill_process` cannot find `abc1234`.
5. `/task cancel` on a shell row calls `tasks/cancel` and never SIGTERM/SIGKILL.
6. `/task cancel` on a shell row sends SIGKILL first, changing today’s graceful Kill.
7. `list_processes` is changed to print A2A rows and the model loses a shell-only tool.
8. Untyped cancel of a colliding id silently picks agent or shell.
9. Empty mixed view still says “No agent tasks.” and implies shell is out of scope.
10. CommandContext stays A2A-only, so TUI `/tasks` never receives shell snapshots in production.
11. Format drops command text for shell rows, so the operator cannot tell which process to kill.
12. ACP or tests that only implement `TaskSource` fail to compile if `CommandContext` gains a required method.
13. Wait is accidentally required for shell rows and hangs because there is no A2A handle.
14. Restore/persist of A2A metadata is rewritten to include shell pids and session JSON breaks.

## 3. UX Implementation and Assessment

### Time to First Value
- [ ] `/tasks` shows a shell row after `start_process` without a second command
- [ ] `/tasks` still shows an A2A row after background spawn

### Onboarding Clarity
- [ ] Each row includes `kind=agent` or `kind=shell`
- [ ] Unknown or ambiguous id names the problem (not found vs use a typed id)

### Production-Ready Defaults
- [ ] Missing shell source lists A2A rows only (not an error)
- [ ] No config flag is required for the unified list

### Golden Path Quality
- [ ] Mixed list contains both families
- [ ] Cancel on a shell row uses Kill

### Decision Load
- [ ] One list command; kind is displayed, not prompted
- [ ] Cancel takes one id (typed when namespaces collide)

### Progressive Complexity
- [ ] `list_processes` / `kill_process` remain the shell-only tools
- [ ] `/task wait` stays the A2A join

### Error Quality
- [ ] Ambiguous untyped id tells the operator to use `agent:<id>` or `shell:<id>`
- [ ] Missing id still returns not-found

### Failure Safety
- [ ] Cancel is explicit; list is read-only
- [ ] Shell Kill remains the existing graceful path

### Runtime Transparency
- [ ] `/tasks` shows kind, id, spec/command, and state
- [ ] No silent rewrite of shell task ids in the process manager

### Debuggability
- [ ] Typed id in the list is the id cancel accepts
- [ ] Raw 7-hex id still works with `kill_process`

### Cross-Surface Consistency
- [ ] Slash `/tasks` is the unified view; process tools stay shell-only
- [ ] Terminology is kind / id / spec-or-command / state

### Workflow Consistency
- [ ] Optional `ShellTaskSource` follows optional `TaskSource`
- [ ] A2A registry and shell manager stay in their packages

### Change Safety
- [ ] Existing A2A list/wait/cancel tests keep passing
- [ ] Process tool tests keep passing

### Experimentation Safety
- [ ] A2A-only contexts still list agent rows
- [ ] Shell-only snapshots still list when the registry is empty

### Interaction Latency
- [ ] List does not wait on children or processes
- [ ] Cancel returns after the routed stop, not after unrelated work

### Developer Feedback Speed
- [ ] Mixed-list tests fail if a kind is dropped
- [ ] Cancel-routing tests fail if Kill is not called for shell

### Team Scale
- [ ] View types live in `internal/agent/tasks` next to the registry
- [ ] Commands stay in `internal/commands/tasks.go`

### System Scale
- [ ] View is a merge of two snapshots, not a third store
- [ ] New kinds can be added as another snapshot source later

### Right Behavior by Default
- [ ] Shell cancel is SIGTERM then SIGKILL
- [ ] Agent cancel remains `tasks/cancel` then SIGTERM

### Anti-Bypass Design
- [ ] Mixed-namespace test fails if typed ids collapse
- [ ] `list_processes` tests fail if that tool starts listing A2A rows

## 4. Tests

### TC-01: typed_id_round_trip

**Given** a kind and a raw id.
**When** `TypedID` and `SplitID` run.
**Then** `agent:abc1234` / `shell:abc1234` parse back to kind + raw, and an untyped id has empty kind.

### TC-02: merge_empty

**Given** no agent records and no shell snapshots.
**When** `Merge` runs.
**Then** the result is empty.

### TC-03: merge_agent_only

**Given** one A2A record.
**When** `Merge` runs.
**Then** the row is `kind=agent` with typed id `agent:<id>`.

### TC-04: merge_shell_only

**Given** one shell snapshot.
**When** `Merge` runs.
**Then** the row is `kind=shell` with typed id `shell:<id>` and the command as spec.

### TC-05: merge_mixed_collision

**Given** an A2A id `abc1234` and a shell id `abc1234`.
**When** `Merge` runs.
**Then** two rows exist, kinds differ, and typed ids are `agent:abc1234` and `shell:abc1234`.

### TC-06: format_view_empty_and_mixed

**Given** a mixed `Merge` result (or none).
**When** `FormatView` runs.
**Then** empty prints a unified empty message; mixed output contains `kind=agent` and `kind=shell`.

### TC-07: cancel_typed_shell_calls_kill

**Given** a shell source that records Kill.
**When** cancel runs with `shell:<id>`.
**Then** Kill receives the raw id (SIGTERM/SIGKILL path).

### TC-08: cancel_typed_agent_unchanged

**Given** a registered A2A handle that records call order.
**When** cancel runs with `agent:<id>`.
**Then** `tasks/cancel` then SIGTERM still run.

### TC-09: cancel_untyped_collision_rejected

**Given** both stores have raw id `abc1234`.
**When** cancel runs with untyped `abc1234`.
**Then** an ambiguous-id error is returned and neither store is mutated incorrectly.

### TC-10: tasks_command_mixed_list

**Given** a command context with an A2A registry and a shell source.
**When** `/tasks` executes.
**Then** the output includes `kind=agent` and `kind=shell`.

### TC-11: task_cancel_shell_row

**Given** the same mixed context.
**When** `/task cancel shell:<id>` executes.
**Then** the shell source Kill is invoked with the raw id.

### TC-12: list_processes_and_kill_process_unchanged

**Given** a `TaskManager` with a shell snapshot.
**When** `list_processes` and `kill_process` execute.
**Then** they still list/kill by raw shell id and do not require `kind=` or A2A rows.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 21:

- `/tasks` shows `kind=agent|shell` for both families
- Existing `list_processes` / `kill_process` still work for shell
- Cancel on a shell row still SIGTERM/SIGKILL as today
- Tests cover mixed lists
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 21
- Implementation files: `internal/agent/tasks/view.go`, `internal/agent/tasks/format.go`, `internal/commands/tasks.go`, `internal/tools/shell_source.go`, `internal/agent/executor/builtin.go`, `internal/conversation/conversation.go`, `internal/conversation/builder.go`, `cmd/spin/tui_command_context.go`
- Test files: `internal/agent/tasks/view_test.go`, `internal/agent/tasks/format_test.go`, `internal/commands/tasks_test.go`, `internal/tools/process_tools_test.go`, `internal/agent/executor/background_test.go`, `internal/conversation/builder_test.go`, `cmd/spin/tui_resume_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-021-unified-task-view.md` — this journey
- `internal/agent/tasks/view.go` — Kind, TypedID, SplitID, Merge, CancelView (two stores, one view)
- `internal/agent/tasks/view_test.go` — typed ids, mixed 7-hex collision, cancel routing
- `internal/tools/shell_source.go` — TaskManager → ShellSource adapter

Files modified:
- `internal/agent/tasks/format.go` — FormatView (`kind=agent|shell`)
- `internal/agent/tasks/format_test.go` — mixed FormatView
- `internal/agent/tasks/registry.go` — ErrAmbiguous; package comment
- `internal/commands/tasks.go` — `/tasks` unified list; `/task cancel` routes shell to Kill
- `internal/commands/tasks_test.go` — mixed list; shell cancel
- `internal/commands/commands.go` — help text for both families
- `internal/tools/list_processes.go` — comment: shell-only; unified view is /tasks
- `internal/tools/process_tools_test.go` — list_processes unchanged; AsShellSource Kill raw id
- `internal/agent/executor/builtin.go` — expose TaskManager after RegisterTools
- `internal/agent/executor/background_test.go` — CancelView shell row uses graceful Kill
- `internal/conversation/conversation.go` — GetShellTasks
- `internal/conversation/builder.go` — bind runtime TaskManager into the view
- `internal/conversation/builder_test.go` — Build wires shell source
- `cmd/spin/tui_command_context.go` — ShellTaskSource
- `cmd/spin/tui_resume_test.go` — ShellTasks method
- `docs/testing.md` — journey 021 row
- `specs/agent-harness/ROADMAP.md` — Step 21 DoD and traceability
