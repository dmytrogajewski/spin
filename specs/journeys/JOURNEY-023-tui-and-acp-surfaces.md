# JOURNEY-023-tui-and-acp-surfaces: TUI and ACP surfaces for skills, tasks, and agents

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: TUI and ACP surfaces for skills, tasks, and agents

## 1. Journey

When **Alex runs spin in the TUI or an ACP editor session after skills, compact, tasks, and local children already exist** I want **timeline blocks, a status-bar task count, palette entries, slash-command discovery, and ACP thought/session updates for the same events** so I **can see what skill loaded, which tasks are running, which child spawned, why a hook vetoed, and how much compact saved — without a new ACP transport**.

## 2. CJM

Alex already has a skill catalog (`/skills`), a unified task registry (`/tasks`), builtin A2A peers in `nav`, compact on by default (Step 13 chip), and `SUBAGENT_START` veto as an error/log. None of that is a first-class TUI or ACP surface. The timeline only has SKILL among the spec block types. The status bar has compact on/off + `−N%` but no `tasks N/M`. The palette lists Run/Search/theme — not Skills, Tasks, Agents. Welcome and `/help` omit `/agents`. ACP clients never see task/child state unless it is wrapped in the existing thought/session update channel with a `kind=a2a` discriminator they can ignore. This journey adds those surfaces only. It does **not** add remote HTTPS A2A (Step 24) or change shutdown (Step 25). Default compact stays **on**.

Assumption: ACP notifications reuse `UpdateAgentThoughtText` / session update helpers already used by the transformer; unknown shapes are ignored by clients. Assumption: `kind=a2a` is a discriminator in the thought text (or equivalent field), not a new JSON-RPC method. Assumption: `/agents` lists `nav` peer records, which fall back to `subagent.Builtins()` when no remote peers exist. Assumption: hook veto reason is shown on a HOOK timeline block and/or an overlay; a log line alone is not enough.

### Phase 1: See harness events on the timeline

**User Intent:** Watch skill load, task state, child spawn, hook veto, and compact as first-class timeline blocks.

**Actions:** Activate a skill. Start or complete an A2A task. Spawn a child. Hit a `SUBAGENT_START` veto. Trigger compact. Read the timeline.

**Pain / Risk:** New types are rendered as NOTICE/TOOL and look like ordinary tool output. Hook veto is only in `~/.spin/spin.log`. Compact has a status chip but no COMPACT block. SKILL exists but TASK/SUBAGENT/HOOK/COMPACT do not. Mapper ignores existing spawn/compact events.

**Success Signal:** Mapper emits `SKILL`, `TASK`, `SUBAGENT`, `HOOK`, and `COMPACT` blocks. Hook veto reason is in the HOOK block body (or overlay), not only a log line.

### Phase 2: Read task count and discover commands

**User Intent:** Know how many tasks are in flight and find Skills / Tasks / Agents without memorizing slashes.

**Actions:** Start TUI with an empty registry. Start one or more tasks. Open the command palette. Read welcome and `/help`.

**Pain / Risk:** Status always shows `tasks 0/0` and is noisy when empty. Compact chip regresses off. Palette never lists Skills/Tasks/Agents. Welcome still only lists `/mode` `/resume` `/help`. `/agents` does not exist. Compact **mode** is confused with the compact chip.

**Success Signal:** Status shows `tasks N/M` only when the registry is non-empty. Compact chip still shows on/`−N%` by default. Palette lists Skills, Tasks, Agents. Welcome and `/help` include `/skills` `/tasks` `/agents`. `/agents` lists builtin specs / nav peers.

### Phase 3: See the same events in an ACP session

**User Intent:** An editor ACP client receives the same task/skill/child updates without a new transport.

**Actions:** Run an ACP session. Spawn a child or change task state. Observe session updates.

**Pain / Risk:** A new notification method breaks clients. Unknown shapes crash the client. Task events are dropped in `convertEventToSessionUpdate`. Discriminator is missing so the client cannot tell parent thought from child/task thought.

**Success Signal:** Transformer wraps task/child/hook events as existing thought/session updates. The payload includes `kind=a2a` (or equivalent). Clients that ignore unknown text still stay up.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Timeline has no TASK/SUBAGENT/HOOK/COMPACT types | 1 | Mapper covers the spec block types |
| Hook veto is a log line / wrapped error only | 1 | HOOK block + overlay with script reason |
| Status has compact chip but no task count | 2 | `tasks N/M` when registry non-empty |
| Palette cannot jump to Skills/Tasks/Agents | 2 | Palette entries with those names |
| Welcome/help omit `/agents` | 2 | Footer + `/help` + `/agents` command |
| ACP clients never see child/task state | 3 | Thought/session wrap + `kind=a2a` |

### North Star Summary

Alex opens the TUI and sees SKILL/TASK/SUBAGENT/HOOK/COMPACT on the timeline, `tasks N/M` on the status bar when work is registered, and the compact chip still on by default. Palette and welcome name Skills, Tasks, Agents. `/agents` lists builtin peers. A hook veto shows its reason on a block or overlay. The same task/child events reach ACP as thought/session updates tagged `kind=a2a`. No remote HTTPS A2A. No shutdown change.

### Stressors

1. Empty task registry — status omits `tasks N/M` (no `tasks 0/0` noise).
2. Non-empty registry — status shows `tasks N/M` and still shows the compact on/`−N%` chip (Step 13 does not regress).
3. `SPIN_COMPACT=0` — compact chip stays `off`; new surfaces still work.
4. Mapper receives `EventSubagentSpawn` — timeline appends a `SUBAGENT` block, not NOTICE.
5. Mapper receives `EventBackgroundTaskStarted` — timeline appends a `TASK` block with id/state.
6. Mapper receives `EventCompactionTriggered` — timeline appends a `COMPACT` block.
7. `SUBAGENT_START` exit 2 — HOOK block or overlay contains the script reason; not only a slog line.
8. ACP client ignores unknown notification methods — updates arrive as existing thought/session shapes.
9. ACP task/child thought includes `kind=a2a` so the client can filter child stream from parent thought.
10. Palette empty query lists Skills, Tasks, and Agents by those names.
11. Welcome footer and `/help` both list `/skills`, `/tasks`, and `/agents`.
12. `/agents` with no remote peers lists `subagent.Builtins()` via nav (explorer/planner/reviewer/ask_user).
13. Unknown mapper event types remain ignored (no panic).
14. ACP `kind=a2a` wrap does not add an HTTP/HTTPS transport (Step 24 stays closed).

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Welcome lists `/skills` `/tasks` `/agents` on first TUI paint
- [x] Status shows `tasks N/M` as soon as the registry is non-empty

### Onboarding Clarity
- [x] Palette names Skills, Tasks, Agents
- [x] Hook veto reason is visible in the UI, not only a log line

### Production-Ready Defaults
- [x] Compact chip remains on by default (Step 13)
- [x] `/agents` works with builtin specs when no remote peers exist

### Golden Path Quality
- [x] Mapper covers SKILL, TASK, SUBAGENT, HOOK, COMPACT
- [x] ACP session receives task/child updates on existing channels

### Decision Load
- [x] Empty registry hides the task count
- [x] Palette adds three entries without replacing existing commands

### Progressive Complexity
- [x] Slash commands stay optional; palette and welcome discover them
- [x] `kind=a2a` is a discriminator, not a new transport

### Error Quality
- [x] Hook veto names the reason from the script
- [x] Unknown events are ignored, not fatal

### Failure Safety
- [x] ACP wrap uses thought/session updates clients already accept
- [x] No remote dial or allowlist change

### Runtime Transparency
- [x] Timeline shows skill/task/child/hook/compact as distinct types
- [x] Status shows live `tasks N/M` when work exists

### Debuggability
- [x] HOOK body/overlay traces back to the veto reason
- [x] ACP thought text includes `kind=a2a` plus task/child identity

### Cross-Surface Consistency
- [x] TUI blocks and ACP thoughts cover the same event families
- [x] `/skills` `/tasks` `/agents` match SPEC slash names

### Workflow Consistency
- [x] Palette uses existing `CommandRegistry` / `NewSimpleCommand`
- [x] Mapper follows the SKILL-block pattern

### Change Safety
- [x] Default compact stays on
- [x] Existing palette commands remain registered

### Experimentation Safety
- [x] Tests construct events; no live child or remote card required
- [x] Hook veto overlay/block is unit-tested with a fixture reason

### Interaction Latency
- [x] Mapper appends blocks without a new IO round trip
- [x] ACP wrap is a thought update, not a new handshake

### Developer Feedback Speed
- [x] Status updates from metrics setters
- [x] `/help` and welcome show the new commands immediately

### Team Scale
- [x] Slash commands are registered in the shared commands package
- [x] Journey comments point at this document

### System Scale
- [x] New block types are Valid() constants, not ad-hoc NOTICE titles
- [x] ACP discriminator can tag future child events without a new method

### Right Behavior by Default
- [x] Compact remains on
- [x] Task count hidden until the registry has rows

### Anti-Bypass Design
- [x] Hook veto reason must appear in a block or overlay assertion
- [x] ACP tests fail if `kind=a2a` is missing from the wrap

## 4. Tests

### TC-01: mapper_skill_block

**Given** a `skill` / `load_skill` tool start.
**When** the mapper handles the event.
**Then** the timeline block type is `SKILL`.

### TC-02: mapper_task_block

**Given** `EventBackgroundTaskStarted` with a task id and state.
**When** the mapper handles the event.
**Then** the timeline block type is `TASK` and the title includes the id.

### TC-03: mapper_subagent_block

**Given** `EventSubagentSpawn` with an agent type.
**When** the mapper handles the event.
**Then** the timeline block type is `SUBAGENT`.

### TC-04: mapper_hook_veto_block

**Given** `EventHookVeto` with reason `veto`.
**When** the mapper handles the event.
**Then** the timeline block type is `HOOK` and the body contains `veto`.

### TC-05: mapper_compact_block

**Given** `EventCompactionTriggered`.
**When** the mapper handles the event.
**Then** the timeline block type is `COMPACT`.

### TC-06: status_tasks_hidden_when_empty

**Given** metrics with `TasksTotal == 0`.
**When** the status bar formats.
**Then** the string does not contain `tasks `.

### TC-07: status_tasks_n_of_m

**Given** metrics with 1 active and 2 total tasks.
**When** the status bar formats.
**Then** the string contains `tasks 1/2` and still contains the compact chip.

### TC-08: palette_harness_entries

**Given** the default palette registry after harness registration.
**When** the palette is opened with an empty query.
**Then** filtered commands include Skills, Tasks, and Agents.

### TC-09: acp_kind_a2a_thought

**Given** a subagent spawn or task-state event.
**When** the ACP converter maps it.
**Then** the update is an agent thought on the existing channel and the text contains `kind=a2a`.

### TC-10: welcome_and_help_slash_commands

**Given** welcome footer and `/help` output.
**When** the operator reads them.
**Then** both include `/skills`, `/tasks`, and `/agents`.

### TC-11: agents_lists_builtins

**Given** no remote peers.
**When** `/agents` runs.
**Then** output lists builtin specs (explorer, planner, reviewer, or ask_user).

### TC-12: hook_veto_overlay_reason

**Given** a hook veto reason string.
**When** the overlay renders.
**Then** the reason is visible in the overlay text.

## Traceability
- Roadmap item: [Step 23](../agent-harness/ROADMAP.md)
- Implementation files: see Implementation below
- Test files: `internal/tui/mapper_harness_test.go`, `internal/ui/status/formatter_test.go`, `internal/ui/status/aggregator_test.go`, `internal/ui/overlay/harness_test.go`, `internal/ui/overlay/hook_veto_test.go`, `internal/protocol/acp/a2a_notifications_test.go`, `internal/commands/agents_test.go`, `cmd/spin/tui_welcome_test.go`, `internal/agent/child/spawn_hooks_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md` — this journey
- `internal/tui/mapper_harness.go` — TASK/SUBAGENT/HOOK/COMPACT mapper
- `internal/tui/mapper_harness_test.go` — mapper block-type tests
- `internal/ui/overlay/harness.go` — palette Skills/Tasks/Agents
- `internal/ui/overlay/harness_test.go` — palette name assertions
- `internal/ui/overlay/hook_veto.go` — overlay with veto reason
- `internal/ui/overlay/hook_veto_test.go` — reason visible
- `internal/commands/agents.go` — `/agents` from nav peers / builtins
- `internal/commands/agents_test.go` — lists explorer
- `internal/protocol/acp/a2a_notifications_test.go` — `kind=a2a` thought wrap

Files modified:
- `internal/ui/blocks/block.go` — TASK, SUBAGENT, HOOK, COMPACT types
- `internal/ui/blocks/tokens.go`, `renderer.go`, `*_test.go` — colors and Valid()
- `internal/events/event.go` — `EventHookVeto`, `HookVetoData`, `TaskStateData`
- `internal/tui/mapper.go` — harness event dispatch
- `internal/ui/status/manager.go`, `formatter.go`, `aggregator.go` — `tasks N/M`
- `internal/ui/adapters/puretty.go` — register harness palette commands
- `internal/protocol/acp/notifications.go` — thought wrap with `kind=a2a`
- `internal/commands/commands.go` — register `/agents`, help examples
- `cmd/spin/tui.go` — welcome lists `/skills` `/tasks` `/agents`
- `internal/agent/child/spawn_exec.go` — emit `EventHookVeto` on start block
- `docs/testing.md` — journey 023 row
- `specs/agent-harness/ROADMAP.md` — Step 23 DoD and traceability
