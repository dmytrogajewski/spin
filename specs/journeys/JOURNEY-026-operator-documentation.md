# JOURNEY-026-operator-documentation: Operator documentation

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Operator documentation

## 1. Journey

When **Alex is operating spin (TUI, `exec`, or ACP) and needs to write a skill, load a plugin, disable compact, hook a lifecycle event, or wait on a child process** I want **Diátaxis how-tos and references that name only landed commands and flags** so I **can complete the task without inventing a flag that does not exist, and I can look up the compact table or hook event names without reading Go**.

## 2. CJM

Alex already has Steps 1–25 on disk: skill discovery, plugins, compact, process children, hooks, and a local A2A binding. The gap is operator-facing prose. `docs/` today is a testing guide. `/help` lists `/skills`, `/tasks`, `/task wait|cancel`, `/agents`, and `SPIN_COMPACT=0`, but there is no how-to for writing a `SKILL.md`, no plugin layout page, no spawn/wait process model, and no reference tables for compact rules or the ten hook events. This journey writes those pages and links them from the README. It does **not** add a docs site generator, rewrite `docs/testing.md`, or change product Go code.

Assumption: document only flags and commands that exist in the landed tree. If [SPEC.md](../agent-harness/SPEC.md) names a surface that is not in code, the page links the spec and says it is not shipped. Assumption: GitNexus is unavailable; impact is grep and file reads. Assumption: `make lint` is golangci-lint plus deadcode (no markdownlint job in the Makefile); docs stay green by not touching Go.

### Phase 1: Find the right page

**User Intent:** Start from the README or `/help` and reach the how-to or reference that answers the task.

**Actions:** Open README SYNOPSIS / USAGE / SEE ALSO. Follow a how-to link. Or type `/help` and match the same command names in the docs.

**Pain / Risk:** README never points at `docs/how-to/`. A how-to is buried under explanation. A reference page starts with a tutorial narrative. Links invent paths that do not exist.

**Success Signal:** README names at least one how-to path. How-tos live under `docs/how-to/`. References live under `docs/reference/`. Quadrants are not mixed.

### Phase 2: Complete a landed task

**User Intent:** Write a skill, package a plugin, spawn/wait/cancel a child, disable compact, or install a hook script using only shipped surfaces.

**Actions:** Follow the how-to steps. Place files in the documented roots. Run `/skills`, `/tasks`, `/task wait|cancel`, `/agents`, or `spin a2a --spec <name> --stdio`. Set `SPIN_COMPACT=0` or `compact.enabled: false`. Drop a script named after `Event.ScriptName()`.

**Pain / Risk:** The page names `--no-compact` (spec only). It claims `spin spawn`. It lists hook files as `PRE_TOOL_USE` instead of `pre-tool-use`. It says TaskFrame phases are `plan|work|review|ask`. It tells the operator the child runs a live LLM/ReAct loop.

**Success Signal:** Each how-to step maps to a landed path, flag, or slash command. Honesty notes call out the echo Task handler and ACP-vs-Close hook difference.

### Phase 3: Look up a contract

**User Intent:** Confirm a compact command, R12–R15 rule, hook event, filename, or `updated_input` shape without reading source.

**Actions:** Open `docs/reference/compact.md` or `docs/reference/hooks.md`. Scan the table. Copy a filename or env var.

**Pain / Risk:** Command table drifts from `internal/contexteng/compact` registry. Event list is not the ten landed names. `updated_input` is omitted. Escape hatch is missing.

**Success Signal:** Compact table matches the Default registry. Hook table lists ten events and ten filenames. Escape hatch matches `/help`.

### Phase 4: Trust the honesty boundary

**User Intent:** Know what is shipped versus what the spec still describes.

**Actions:** Read a "not shipped" note that links [SPEC.md](../agent-harness/SPEC.md). Do not treat spec-only flags as operator commands.

**Pain / Risk:** Docs silently invent remote-A2A without `a2a.allowlist`. Docs claim `SESSION_END` on ACP cancel. Docs invent recursive `skills/**` discovery.

**Success Signal:** Unshipped surfaces are named as unshipped and linked to the spec. Landed allowlist, task-id typing, and Close-vs-ACP behavior are explicit.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| README has no how-to pointer | 1 | SEE ALSO / USAGE link |
| Spec flags leak into operator pages | 2 | Document only merged flags |
| Compact/hook contracts live only in Go | 3 | Reference tables |
| Echo child looks like a full agent | 2 | Honest process-model note |
| ACP cancel confused with session end | 4 | Close vs cancel paragraph |

### North Star Summary

Alex opens the README, follows a how-to, and finishes the task with the same command names `/help` prints. When Alex needs a compact rule or hook filename, a reference table answers without a tutorial. Nothing on those pages claims a flag, event, or child LLM that the tree does not run.

### Stressors

1. A page documents `--no-compact` or a verbose-raw CLI flag from SPEC.md that is not in `/help` or `cmd/spin`.
2. A hook page uses the wrong event token (`PRE_TOOL` instead of `PRE_TOOL_USE`) or the wrong filename (`pre_tool_use` instead of `pre-tool-use`).
3. Subagent how-to invents `spin spawn` or a `spawn` CLI; the landed child entry is `spin a2a --spec <name> --stdio` or `unix://`.
4. Subagent how-to claims the local child runs a live LLM/ReAct loop; the landed server answers with the in-process echo Task handler.
5. TaskFrame phases are documented as SPEC `plan|work|review|ask` instead of `/mode` names `regular` / `review` / `compact` / `planning`.
6. `/tasks` ids are shown without `agent:` / `shell:` typing, or `kind=` is omitted.
7. `SESSION_END` is claimed on ACP `session/cancel`; that path CancelAlls tasks and does not Close the conversation.
8. Skill discovery is documented as recursive `skills/**`; plugins load only immediate `skills/` children.
9. Compact escape hatch omits `SPIN_COMPACT=0` or `compact.enabled: false`.
10. Remote A2A is documented without `a2a.allowlist`; off-list URLs must be rejected before dial.
11. `updated_input` is missing or described as a merge instead of a full argument replacement.
12. A reference page opens with a tutorial narrative (quadrant leak).
13. README links a how-to path that is empty or absent.
14. Plugin page omits `Contain` (`./` prefix, no `..` escape) or claims a failing MCP server unloads skills.
15. Parent quit is documented without CancelAll then SIGTERM, or without next-start pid-file reap under the runtime dir.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] README SEE ALSO or USAGE reaches a how-to in one click
- [x] `/help` command names match the how-to headings

### Onboarding Clarity
- [x] How-tos state the goal in one sentence
- [x] Unshipped spec surfaces are labeled not shipped

### Production-Ready Defaults
- [x] Compact documented as default on
- [x] Empty `a2a.allowlist` documented as deny-all remote cards

### Golden Path Quality
- [x] Skill how-to covers write, place, `/skills`, `skill` tool
- [x] Plugin how-to covers `plugin.json`, containment, MCP isolation

### Decision Load
- [x] How-tos do not offer unshipped alternate flags
- [x] Reference pages are tables, not decision essays

### Progressive Complexity
- [x] Simple skill is a single `SKILL.md` directory
- [x] Advanced plugin MCP and hooks are after the minimal layout

### Error Quality
- [x] Unknown skill / path-escape / allowlist reject are named
- [x] Hook exit 2 is named as veto on blocking events

### Failure Safety
- [x] Compact R12 fail-safe (raw output) is in the reference
- [x] MCP failure does not unload plugin skills

### Runtime Transparency
- [x] `/tasks` `kind=agent|shell` is documented
- [x] Child echo handler is documented as echo, not an LLM

### Debuggability
- [x] Hook script filenames are copyable
- [x] Compact command table is copyable

### Cross-Surface Consistency
- [x] Slash commands match `internal/commands` help text
- [x] TaskFrame phases match `/mode` names

### Workflow Consistency
- [x] How-tos follow Diátaxis task shape (goal, steps, result)
- [x] References follow Diátaxis lookup shape (synopsis, tables)

### Change Safety
- [x] `docs/testing.md` is not rewritten
- [x] No product Go code changes except README links required by DoD

### Experimentation Safety
- [x] Compact escape hatch is documented before any rewrite claim
- [x] `spin plugin validate` is the dry-run for a plugin root

### Interaction Latency
- [x] Pages are short enough to scan; tables first on references
- [x] No site-generator setup step

### Developer Feedback Speed
- [x] `make lint` remains the docs-adjacent gate
- [x] File-existence probes confirm the five DoD paths

### Team Scale
- [x] Skill and plugin roots are shareable via the workdir tree
- [x] Config keys use the landed YAML names

### System Scale
- [x] Collision `source` tags (project / user / plugin / bundled) are listed
- [x] Plugin extra `plugins.paths` is documented

### Right Behavior by Default
- [x] Compact on; remote A2A off until allowlisted
- [x] Child `spawn` tool deny-by-default is stated

### Anti-Bypass Design
- [x] Off-list remote card rejected before dial
- [x] Plugin paths that omit `./` or escape `..` are rejected

## 4. Tests

### TC-01: skills_how_to_exists

**Given** the Step 26 DoD.
**When** the operator opens `docs/how-to/agent-skills.md`.
**Then** the file is non-empty and covers writing a skill, discovery roots, `/skills`, and the `skill` / `load_skill` tool.

### TC-02: plugins_how_to_exists

**Given** the Step 26 DoD.
**When** the operator opens `docs/how-to/agent-plugins.md`.
**Then** the file is non-empty and covers `plugin.json` layout, `Contain`, and MCP isolation.

### TC-03: subagents_how_to_exists

**Given** the Step 26 DoD.
**When** the operator opens `docs/how-to/subagents.md`.
**Then** the file is non-empty and covers spawn, wait, cancel, the `spin a2a` process model, typed task ids, and the echo handler honesty note.

### TC-04: compact_reference_exists

**Given** the Step 26 DoD.
**When** the operator opens `docs/reference/compact.md`.
**Then** the file is a lookup (not a tutorial) with the command table, R12–R15, and `SPIN_COMPACT=0` / `compact.enabled: false`.

### TC-05: hooks_reference_exists

**Given** the Step 26 DoD.
**When** the operator opens `docs/reference/hooks.md`.
**Then** the file lists the ten landed events, their script filenames, and `updated_input`.

### TC-06: readme_points_at_howto

**Given** the root README man-page voice.
**When** the operator reads SYNOPSIS / USAGE / SEE ALSO.
**Then** at least one `docs/how-to/` path is linked.

### TC-07: no_unshipped_flags

**Given** SPEC.md names `--no-compact` and TaskFrame `plan|work|review|ask`.
**When** the five operator pages are scanned.
**Then** those tokens are either absent or labeled not shipped with a spec link.

### TC-08: hook_filenames_match_code

**Given** `internal/safety/hooks/event.go` `eventScriptNames`.
**When** `docs/reference/hooks.md` is read.
**Then** filenames are `session-start`, `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `post-tool-use-failure`, `subagent-start`, `subagent-stop`, `pre-compact`, `stop`, `session-end`.

### TC-09: echo_handler_honest

**Given** `a2a.NewMemoryHandler` is the local child Task handler.
**When** `docs/how-to/subagents.md` is read.
**Then** it states the child answers with the in-process echo Task handler and does not claim a live child LLM/ReAct loop.

### TC-10: testing_guide_untouched_intent

**Given** `docs/testing.md` is the existing testing voice.
**When** Step 26 lands.
**Then** that file is not rewritten; new pages are only the five named DoD paths plus README links.

### TC-11: make_lint_green

**Given** a docs-only change.
**When** `make lint` runs.
**Then** it exits 0 (golangci-lint + deadcode; no markdownlint job in the Makefile).

### TC-12: test_count_holds

**Given** baseline `Test*` count 3647.
**When** `CGO_ENABLED=0 go test ./... -list . | grep -c '^Test'` and `make test` run.
**Then** the count is ≥ 3647 and `make test` is green.

## Acceptance Criteria

- `docs/how-to/agent-skills.md` — write a skill, where to put it, `/skills`, `skill` tool
- `docs/how-to/agent-plugins.md` — `plugin.json` layout, containment, MCP isolation
- `docs/how-to/subagents.md` — spawn, wait, cancel, process model
- `docs/reference/compact.md` — command table, R12–R15, escape hatch
- `docs/reference/hooks.md` — ten events, filenames, `updated_input`
- README points at the how-to
- `make lint` pass (markdown as required by repo)

## Traceability
- Roadmap item: [Step 26](../agent-harness/ROADMAP.md)
- Implementation files: `docs/how-to/agent-skills.md`, `docs/how-to/agent-plugins.md`, `docs/how-to/subagents.md`, `docs/reference/compact.md`, `docs/reference/hooks.md`, `README.md`
- Test files: docs-only step; gates are `make lint` and `make test` (count ≥ 3647)

## Implementation

Files created:
- `specs/journeys/JOURNEY-026-operator-documentation.md` — this journey
- `docs/how-to/agent-skills.md` — write a skill, roots, `/skills`, `skill` / `load_skill`
- `docs/how-to/agent-plugins.md` — `plugin.json`, `Contain`, MCP isolation, `spin plugin validate`
- `docs/how-to/subagents.md` — `spin a2a`, wait/cancel, typed ids, echo handler, Close vs ACP
- `docs/reference/compact.md` — command table, R12–R15, `SPIN_COMPACT=0`
- `docs/reference/hooks.md` — ten events, filenames, `updated_input`

Files modified:
- `README.md` — SEE ALSO points at the how-tos
- `specs/agent-harness/ROADMAP.md` — Step 26 DoD and traceability
