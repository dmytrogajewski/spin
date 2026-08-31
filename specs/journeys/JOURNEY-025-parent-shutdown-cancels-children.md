# JOURNEY-025-parent-shutdown-cancels-children: Parent shutdown cancels children and ends the session

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Parent shutdown cancels children and ends the session

## 1. Journey

When **Alex is leaving a spin parent (TUI Ctrl-C, `/exit`, or ACP cancel) with A2A children still working** I want **every running child to receive `tasks/cancel` then SIGTERM, `SESSION_END` hooks to fire, and the TUI to clear the screen** so I **get a blank terminal with no orphan `spin a2a` processes, and a later `spin` start reaps leftover pid/socket files under the runtime dir**.

## 2. CJM

Alex runs spin as a parent (TUI daily, ACP from an editor). Children are OS processes on stdio or a Unix socket. Today, quit already closes the conversation (STOP then SESSION_END, Step 8) and PureTTY teardown already writes `ClearHome`. The gap is the leave path: running A2A tasks are not canceled as a set, children that outlive a crashed parent are not reaped from pid/socket files, and there is no test that a child exits when stdin or the socket closes. This journey is only that leave path. It does **not** write operator how-tos (Step 26).

Assumption: `Conversation.Close` is the single parent-quit funnel for TUI Ctrl-C and `/exit` (`stopTUILoop` already calls it). Assumption: ACP `session/cancel` cancels running A2A tasks on that session’s conversation without destroying the ACP session (prompt cancel stays usable). Assumption: `SESSION_END` stays on `Close` (Step 8) and must still run after task cancel. Assumption: pid/socket files live under `XDG_RUNTIME_DIR/spin/a2a` (fallback: temp dir). Assumption: `tasks/cancel` on an already-terminal task is ignored (race with a completing child).

### Phase 1: Decide to leave

**User Intent:** Stop the parent while children may still be `working`.

**Actions:** Press Ctrl-C. Or type `/exit`. Or the ACP host sends cancel. Observe that quit is accepted without a second signal.

**Pain / Risk:** Ctrl-C cancels the conversation context before `SESSION_END`. `/exit` and Ctrl-C take different teardown paths. ACP cancel is treated as “destroy the session” and breaks the next prompt. A second SIGINT is required (old TUI hang).

**Success Signal:** One quit action reaches `stopTUILoop` or ACP cancel. The event loop unblocks. Close is invoked once (`closeOnce`).

### Phase 2: Cancel every running child

**User Intent:** No `spin a2a` child keeps working after the parent has decided to leave.

**Actions:** Parent walks the A2A task registry. Each non-terminal row gets `tasks/cancel` then SIGTERM. Already-terminal rows are skipped.

**Pain / Risk:** A completing child returns TaskNotCancelable and abort the rest of shutdown. One hung Cancel blocks quit. SIGTERM is skipped when cancel RPC fails. Background tasks registered without a handle are left unmarked. Shell rows are confused with A2A rows.

**Success Signal:** Every `working` A2A handle is canceled then signaled. Terminal rows are untouched. Shutdown continues after a not-cancelable race.

### Phase 3: End the session and clear the screen

**User Intent:** Hooks and the terminal match a clean leave.

**Actions:** `Close` fires STOP then `SESSION_END` (already wired). PureTTY teardown still writes scroll-region reset plus `ClearHome`. Children whose stdin or socket the parent closed exit on EOF.

**Pain / Risk:** Cancel is added after Close and skips hooks. `ClearHome` teardown is rewritten or dropped. Children stay alive after stdin/socket close. `Close` is skipped because the context is already canceled.

**Success Signal:** Hook recorder sees STOP then `SESSION_END` on the quit path. TUI output still contains `ClearHome`. A child process Wait returns after stdin or client socket close.

### Phase 4: Next parent start reaps leftovers

**User Intent:** A crashed parent does not leave pid/socket files or orphan listeners forever.

**Actions:** `spin a2a` writes a pid file under the runtime dir. Next TUI / exec / ACP start calls reap. Dead pids drop their files. Live orphans receive SIGTERM; their pid and sibling socket files are removed.

**Pain / Risk:** Reap kills an unrelated pid-reused process. Missing `XDG_RUNTIME_DIR` writes to a shared temp without a spin prefix. Stale sockets block the next `unix://` bind. Reap is only hooked in TUI and exec/ACP still leak.

**Success Signal:** Stale pid files are gone after `ReapOnStart`. A live orphan in the runtime dir is signaled. A missing runtime dir is not an error.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Quit leaves working A2A children | 2 | `CancelAll` before hooks |
| `tasks/cancel` races a completing child | 2 | Ignore already-terminal cancel |
| Child leak on parent crash | 4 | Pid/socket files + reap on start |
| Leftover TUI chrome | 3 | Keep existing `ClearHome` teardown |
| Child ignores parent death | 3 | Exit on stdin/socket EOF |

### North Star Summary

Alex hits Ctrl-C (or `/exit`, or ACP cancel). Every running A2A child gets `tasks/cancel` then SIGTERM, `SESSION_END` scripts run, and the terminal is wiped with the existing `ClearHome` sequence. If a parent crashed first, the next `spin` start reaps stale pid and socket files under the runtime dir so no orphan `spin a2a` remains.

### Stressors

1. Child completes between list and `tasks/cancel` — TaskNotCancelable is ignored; other children still cancel.
2. Registry state is already `completed` / `failed` / `canceled` — `Cancel` is a no-op (no RPC, no SIGTERM).
3. Conversation context is already canceled (Ctrl-C) — cancel RPC and `SESSION_END` still run via `WithoutCancel`.
4. `Close` is called twice (defer + `stopTUILoop`) — hooks and cancel run once.
5. Child stdin is closed — `spin a2a --stdio` Serve returns on EOF and the process exits.
6. Unix client socket is closed — `ListenAndServe` returns; no orphan listener for that connection.
7. Stale pid file whose process is gone — reap removes the pid file and sibling socket.
8. Live orphan pid in the runtime dir — reap sends SIGTERM and removes files.
9. `XDG_RUNTIME_DIR` unset — files go under a spin-prefixed temp path, not `/`.
10. Nil task registry on Close — no panic; `SESSION_END` still fires.
11. Handle is nil on a `working` snapshot (restored metadata) — row is marked canceled, no SIGTERM.
12. ACP cancel with no in-progress prompt — still CancelAll if a conversation exists; session remains.
13. TUI quit must still write `ClearHome` (scroll-region reset + home/erase).
14. Pid reuse: reap only touches files under the spin A2A runtime subdir.
15. `CancelAll` error on one id must not skip remaining ids.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] One quit action starts child cancel without an extra command
- [x] No second SIGINT required to leave the TUI

### Onboarding Clarity
- [x] Welcome footer still documents Ctrl-C exits the parent
- [x] `/exit` remains the typed quit command

### Production-Ready Defaults
- [x] Quit cancels children without extra config
- [x] Runtime dir defaults from `XDG_RUNTIME_DIR` or a spin-prefixed temp

### Golden Path Quality
- [x] Ctrl-C / `/exit` / ACP cancel cancel every working A2A task
- [x] `SESSION_END` still runs on `Close`

### Decision Load
- [x] User does not choose cancel-vs-kill; parent always cancel then SIGTERM
- [x] Reap on start is automatic

### Progressive Complexity
- [x] No children: quit is still Close + ClearHome only
- [x] Many children: same quit; CancelAll walks the registry

### Error Quality
- [x] Already-terminal cancel is not surfaced as a quit failure
- [x] Missing runtime dir on reap is not a hard error

### Failure Safety
- [x] One child cancel failure does not skip the others
- [x] `closeOnce` prevents double SESSION_END

### Runtime Transparency
- [x] Registry states become `canceled` for rows that were working
- [x] Pid files make leftovers inspectable under the runtime dir

### Debuggability
- [x] Pid path is derived from pid (filename + contents)
- [x] Hook log still records STOP then SESSION_END

### Cross-Surface Consistency
- [x] TUI and exec Close share the same Conversation cancel+hooks path
- [x] ACP cancel uses the same registry CancelAll

### Workflow Consistency
- [x] Single-task `/task cancel` still uses `Registry.Cancel`
- [x] Shutdown reuses cancel-then-SIGTERM (Step 20)

### Change Safety
- [x] ClearHome teardown is not replaced with a new screen model
- [x] Step 8 hook order (STOP then SESSION_END) is unchanged

### Experimentation Safety
- [x] Tests use temp runtime dirs, not the user XDG dir
- [x] Live-orphan tests only signal processes the test started

### Interaction Latency
- [x] CancelAll does not wait for child Wait/reap
- [x] SIGTERM is non-blocking

### Developer Feedback Speed
- [x] Tests fail on leftover processes after stdin/socket close
- [x] Tests fail if SESSION_END is dropped from Close

### Team Scale
- [x] Pid/socket layout is a convention under one runtime subdir
- [x] No operator how-to in this journey (Step 26)

### System Scale
- [x] CancelAll is O(n) over the in-memory registry
- [x] Reap is O(n) over pid files in the runtime subdir

### Right Behavior by Default
- [x] Terminal tasks are not re-signaled
- [x] Children die on stdin/socket close without an extra flag

### Anti-Bypass Design
- [x] TUI quit cannot skip Close (`stopTUILoop` always closes)
- [x] Reap on start is in the parent entry, not an opt-in flag

## 4. Tests

### TC-01: cancel_ignores_terminal

**Given** a registry row already `completed` with a handle.
**When** `Cancel` is called for that id.
**Then** the handle is not canceled or SIGTERM’d and the call returns nil.

### TC-02: cancel_all_working

**Given** one `working` row and one `completed` row.
**When** `CancelAll` runs.
**Then** only the working handle receives cancel then SIGTERM.

### TC-03: close_cancels_then_session_end

**Given** a conversation with a working task and a hook recorder.
**When** `Close` runs (including on a canceled context).
**Then** the handle was cancel+SIGTERM’d and hooks are STOP then SESSION_END.

### TC-04: close_nil_registry

**Given** a conversation with a nil task registry.
**When** `Close` runs.
**Then** it does not panic and SESSION_END still fires.

### TC-05: child_exits_after_stdin_close

**Given** a spawned `spin a2a --stdio` child.
**When** the parent closes stdin.
**Then** the child process exits.

### TC-06: child_exits_after_socket_close

**Given** `ListenAndServe` accepted a Unix client.
**When** the client closes the socket.
**Then** the server goroutine returns.

### TC-07: session_end_on_tui_quit_path

**Given** `stopTUILoop` with a closer.
**When** quit runs.
**Then** the closer runs (existing SESSION_END contract) and the event loop unblocks.

### TC-08: reap_stale_dead_pid

**Given** a pid file for a process that is not running.
**When** `ReapOnStart` / `ReapStale` runs.
**Then** the pid file and sibling socket are removed.

### TC-09: reap_live_orphan

**Given** a pid file for a live process the test owns.
**When** reap runs.
**Then** the process is signaled and the pid file is removed.

### TC-10: tui_clearhome_teardown

**Given** the existing PureTTY exit sequence.
**When** the teardown writer runs.
**Then** the bytes include `term.ClearHome`.

### TC-11: acp_cancel_cancels_tasks

**Given** an ACP session whose conversation has a working A2A task.
**When** ACP Cancel is sent.
**Then** that task’s handle received cancel then SIGTERM and the session is still valid.

### TC-12: handle_cancel_already_terminal

**Given** a child task already canceled.
**When** `TaskHandle.Cancel` is called again.
**Then** the call returns nil (race).

## Acceptance Criteria

- Parent quit sends cancel to every running A2A task
- Children exit after stdin/socket close (test)
- `SESSION_END` hooks run on that path
- Next `spin` start reaps stale pid files
- TUI still clears the screen (existing `ClearHome` teardown)
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [Step 25](../agent-harness/ROADMAP.md)
- Implementation files: `cmd/spin/tui.go`, `cmd/spin/exec.go`, `cmd/spin/acp.go`, `cmd/spin/a2a.go`, `cmd/spin/reap.go`, `internal/agent/tasks/registry.go`, `internal/agent/child/runtime.go`, `internal/agent/child/handle.go`, `internal/conversation/conversation.go`, `internal/protocol/acp/agent.go`, `internal/ui/adapters/puretty.go`
- Test files: `internal/agent/tasks/registry_test.go`, `internal/conversation/parent_shutdown_test.go`, `internal/agent/child/shutdown_test.go`, `internal/agent/child/runtime_test.go`, `internal/agent/child/handle_test.go`, `cmd/spin/reap_test.go`, `cmd/spin/a2a_test.go`, `cmd/spin/tui_quit_test.go`, `internal/protocol/acp/cancel_test.go`, `internal/ui/adapters/exit_clear_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md` — this journey
- `cmd/spin/reap.go` — parent-start `reapParentOrphans`
- `cmd/spin/reap_test.go` — stale pid removed via `ReapOnStart`
- `internal/agent/child/runtime.go` — runtime dir, pid/socket paths, write, reap
- `internal/agent/child/runtime_test.go` — XDG dir, write pid, dead/live reap
- `internal/agent/child/shutdown_test.go` — Serve/process/socket exit on close
- `internal/conversation/parent_shutdown_test.go` — Close cancels tasks then SESSION_END
- `internal/ui/adapters/exit_clear_test.go` — teardown still writes ClearHome

Files modified:
- `internal/agent/tasks/registry.go` — skip terminal `Cancel`; add `CancelAll`
- `internal/conversation/conversation.go` — Close runs `CancelAll` then STOP/SESSION_END
- `internal/agent/child/handle.go` — ignore already-terminal `tasks/cancel`
- `cmd/spin/tui.go`, `cmd/spin/exec.go`, `cmd/spin/acp.go` — reap orphans on parent start
- `cmd/spin/a2a.go` — child writes pid file and removes it on exit
- `internal/protocol/acp/agent.go` — ACP cancel CancelAlls running A2A tasks
- `internal/ui/adapters/puretty.go` — extract `writeExitClear` (same ClearHome sequence)
- `specs/agent-harness/ROADMAP.md` — Step 25 DoD and traceability
