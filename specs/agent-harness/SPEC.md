# SPEC: Agent harness (skills, plugins, process subagents, context)

## 1. Summary

Spin becomes a standards-compliant agent client and a multi-agent host. It loads [Agent Skills](https://agentskills.io/specification) and [Agent Plugins 1.0](https://agent-plugins.org/specification), frames each turn with a role-specific context bundle, compactes tool output using the [RTK](https://github.com/rtk-ai/rtk) strategies copied 1-1, and spawns subagents as **child processes** that the parent drives through the [A2A](https://a2a-protocol.org/latest/specification/) data model (Task, Message, Agent Card) over a local JSON-RPC binding. Target users are operators who run spin as a coding agent (TUI, `exec`, ACP) and want portable skills, isolated workers, awaitable background work, and a terminal that shows what is running. Success looks like: a skill from the open ecosystem activates with progressive disclosure; a subagent is a real OS process; the parent can wait or continue; hook scripts and plugin hooks fire on every lifecycle event already named in spin; tool transcripts stay compact without dropping exit codes.

## 2. Background & Research

### Market Context

| Product / standard | How they position it | How they describe it | Takeaway for spin |
|---|---|---|---|
| **Agent Skills** ([agentskills.io](https://agentskills.io/specification), [addyosmani/agent-skills](https://github.com/addyosmani/agent-skills)) | Portable folders of `SKILL.md` + optional `scripts/`, `references/`, `assets/` | Progressive disclosure: ~100-token `name`+`description` catalog, body on activation, resources on demand | Native skill format. Catalog always in the system prompt; body via a `skill` tool; one-level-deep relative refs. |
| **Agent Plugins 1.0** ([agent-plugins.org](https://agent-plugins.org/)) | Vendor-neutral package for skills + MCP; TSC includes Amazon, Cursor, Microsoft, OpenAI, Vercel (Google joining) | Closed `plugin.json` schema; fixed `skills/` and `mcp.json`; reverse-domain client dirs; independent component failure | Packaging and discovery layer. Spin-specific hooks live under `com.spin.agent/`, not in portable fields. |
| **Claude Code** | Skills, hooks, custom subagents, foreground/background fork | Settings + skill/agent frontmatter hooks; `context: fork` + `background`; Explore/Plan skip extra context | Event set and fork/background semantics. Do **not** copy in-process fork — spin uses OS processes + A2A. |
| **A2A 1.0** ([a2a-protocol.org](https://a2a-protocol.org/latest/specification/)) | Opaque agents collaborate via Task/Message/Artifact | Operations: send/stream message, get/list/cancel task, get Agent Card. Bindings: JSON-RPC, gRPC, HTTP+JSON, custom | Parent–child protocol. Local custom binding (stdio / Unix socket NDJSON-RPC). Same client talks to remote HTTPS cards. |
| **RTK** ([rtk-ai/rtk](https://github.com/rtk-ai/rtk)) | “Cuts up to 90% of the bash output your agent reads” | Four README strategies + eight architecture techniques; PreToolUse rewrite `git status` → `rtk git status`; fail-safe; exit-code preserve | Copy the approaches 1-1 as spin’s context-saving layer. Optional `rtk` binary when present; Go port otherwise. |
| **Cursor / Codex / Factory Droid / Gemini CLI** | Skills + plugin TSC / `rtk init --agent …` | Same skill folders; hook or plugin rewrite for compact CLI | Interop: spin must consume the same on-disk layouts those tools already write. |
| **LangChain / Anthropic multi-agent research / agentpatterns.ai** | Progressive disclosure + phase-specific context | Orchestrator gets summaries; workers get files + validation commands; vague asks explode discovery tokens | Task framing is a first-class object, not a longer system prompt. |

User complaints in the research (GitHub, docs, forums): skill-scoped hooks dropped on fork ([claude-code#40630](https://github.com/anthropics/claude-code/issues/40630)); RTK does not wrap built-in Read/Grep/Glob so savings vanish unless the agent uses shell; A2A HTTP-only mental model is heavy for local children; Agent Plugins authors still duplicate client dirs; compaction that swallows exit codes breaks CI agents.

### Technical Context

**Agent Skills directory (normative):** `skill-name/SKILL.md` required; `name` matches directory; `description` is what + when (max 1024). Optional: `license`, `compatibility`, `metadata`, experimental `allowed-tools`. Body unrestricted Markdown. Progressive disclosure is a client duty, not a file-format extra.

**Agent Plugins 1.0 (normative):** plugin root must contain `plugin.json` (`$schema` + `name` required; closed top-level fields). Skills are **immediate** children of `skills/` only — no recursive search. `mcp.json` is the only MCP config path. Path containment: plugin-relative paths start with `./` and must stay inside the plugin root. Client extensions: `extensions["com.spin.agent"]` in the manifest and/or a `com.spin.agent/` directory (hooks, spin task frames). Independent failure: a dead MCP server does not unload skills.

**A2A (normative data model, local binding):** Agent Card declares skills, URL, protocol binding, protocol version. Operations map to JSON-RPC methods (`message/send`, `message/stream`, `tasks/get`, `tasks/list`, `tasks/cancel`, `agent/getAuthenticatedExtendedCard`). Layer 3 allows a **custom binding**. Spin’s local binding is newline-delimited JSON-RPC 2.0 on the child’s stdio (or a Unix socket). The child publishes an Agent Card as the first framed message and then serves A2A methods. Remote agents keep HTTPS JSON-RPC as published.

**RTK approaches to copy 1-1** (README + [ARCHITECTURE.md](https://github.com/rtk-ai/rtk/blob/master/docs/contributing/ARCHITECTURE.md)):

| ID | Approach | Technique | Canonical commands |
|---|---|---|---|
| R1 | Smart filtering | Strip comments, whitespace, boilerplate | `read`, `smart`, linters |
| R2 | Grouping | Aggregate by directory / rule / file | `grep`/`rg`, `ruff`, `tsc` |
| R3 | Truncation | Keep relevant context, cut redundancy | `git diff`, long lines |
| R4 | Deduplication | Collapse repeated lines with `×N` | logs |
| R5 | Stats extraction | Count/aggregate, drop details | `git status`, `git log`, `pnpm list` |
| R6 | Error-only | stderr / failures, drop success stdout | runners |
| R7 | Structure-only | Keys + types, strip large JSON values | `json` |
| R8 | Code filtering | `none` / `minimal` (comments) / `aggressive` (signatures) | `read`, `smart` |
| R9 | Failure focus | Passing tests collapsed to a count | `go test`, `pytest`, `vitest`, `jest`, `playwright` |
| R10 | Tree compression | Hierarchy + per-dir counts | `ls`, `tree`, `find` |
| R11 | Auto-rewrite hook | Intercept shell argv, prefix compacting proxy | all Bash-equivalent tool calls |
| R12 | Fail-safe | Filter error → raw output | all |
| R13 | Exit-code preservation | Proxy never swallows non-zero | all |
| R14 | Unknown passthrough | Unrecognized commands inherit stdio, 0% reduction | all |
| R15 | Token estimate | `ceil(bytes/4)` savings accounting, not a real tokenizer | dashboard / TUI chip |

Per-command compact table (README, copy as the default filter registry):

| Input | Compact form |
|---|---|
| `ls` / `tree` | Tree + file counts |
| `cat` / `read` | Signatures and structure over full bodies (`-l none\|minimal\|aggressive`) |
| `grep` / `rg` | Truncated lines, grouped by file |
| `git status` | Compact stat, grouped by state |
| `git diff` | Reduced context, headers stripped |
| `git log` | Hash, author, subject |
| `git add/commit/push/pull` | One confirmation line |
| `go test` | NDJSON parsed, failures only |
| `cargo test` / `npm test` / `pytest` / `jest` / `vitest` / `playwright` | Failures only |
| `ruff check` | Grouped by rule and file |
| `docker ps` | Essential fields only |

**Spin today (evidence from the tree):**

| Area | State |
|---|---|
| Skills | `.agents/skills/*/SKILL.md` exist; **zero Go loader** |
| Plugins | Absent |
| Subagents | `internal/agent/subagent` specs + semaphore; production executor returns `ErrSubagentSpawnNotSupported` |
| Background | `start_process` / `list_processes` — **shell only**, not agent tasks |
| Hooks | 10 events defined; ~4 wired; `UpdatedInput` parsed and unused; `~` in global dir not expanded |
| Context assembly | Harness + compactor + observations live; `retrieval.Pipeline.Assemble` **never called on the turn path** |
| Task framing | `prompt.Composer` sections + `/mode`; no per-turn TaskFrame object |
| A2A | Absent (ACP is the IDE protocol) |
| UX | Timeline + palette + approval; no skill/task/subagent surfaces |

### Deep Dives

- **Progressive disclosure is a context budget, not a UX slogan.** Anthropic’s Agent Skills write-up and LangChain’s on-demand skill tutorial agree: metadata always, body on `load_skill`, references only when the body points at them. Loading every `SKILL.md` body at session start recreates the problem skills were invented to solve.
- **Phase-specific bundles beat one shared context.** [agentpatterns.ai](https://agentpatterns.ai/context-engineering/phase-specific-context-assembly/) and Anthropic’s multi-agent research post: orchestrators drift on file contents; workers drift on planning essays. Each A2A child gets a TaskFrame, not a clone of the parent transcript.
- **Opaque execution is why A2A exists.** The parent must not share internal tool implementations with the child. The child is a spin process with its own harness, tools, and card. The parent sees Tasks and Artifacts.
- **In-process subagents fail the user’s constraint.** The current `Manager` runs an `Executor` goroutine. That cannot be an A2A peer, cannot crash in isolation, and cannot outlive a hung parent cleanly. OS process + stdio JSON-RPC is the unit of isolation.
- **RTK’s hook is the adoption mechanism.** Auto-rewrite (100% of Bash-equivalent calls) beats “the model remembers to type `rtk`”. Suggest-only mode is for audit. Built-in Read/Grep/Glob in other products bypass the hook — spin must run the **same compacting functions** on those tools, not only on `exec`.
- **Claude Code’s skill→fork→background** is the UX users already know. Spin maps `context: fork` to A2A spawn, `background: true` to non-blocking `message/send`, `background: false` to blocking send / `tasks/get`.
- **ACP stays the IDE protocol.** Replacing ACP with A2A for Zed/VS Code hosts would break `spin acp`. A2A is agent-to-agent; ACP is host-to-spin.

## 3. Proposal

### Approach

Add an **agent-harness** layer that sits beside the existing ReAct harness (not inside Yaegi/RLM, not instead of ACP):

```
User / ACP host
      │
      ▼
┌─────────────────────────────────────────────┐
│  Parent spin (TUI | exec | acp)             │
│  Composer + TaskFrame + Skill catalog       │
│  Plugin loader + Hook runner (all 10 events)│
│  Compact pipeline (RTK R1–R15)              │
│  A2A client + Task registry                 │
└───────────┬─────────────────┬───────────────┘
            │                 │
            ▼                 ▼
   child `spin a2a`      remote Agent Card
   stdio / unix socket   HTTPS JSON-RPC
            │
            ▼
   isolated harness + framed context
```

### Key Decisions

| Decision | Choice | Reasoning | Alternatives |
|----------|--------|-----------|-------------|
| Skill format | Agent Skills `SKILL.md` as specified | Portable across Cursor, Claude Code, Codex, Copilot; spin already authors this format via promptkit | Invent a spin YAML skill (breaks ecosystem); load only promptkit checksums (rejects third-party skills) |
| Packaging | Agent Plugins 1.0 client | Fixed locations, closed manifest, independent failure, official TSC | Ad-hoc zip; Claude `marketplace.json` only (vendor lock-in) |
| Subagent execution | OS process + A2A data model | User requirement; crash isolation; wait/cancel via `tasks/*`; same client for remote peers | In-process goroutine (current stub — no isolation); raw pipes without A2A (reinvents Task) |
| Local A2A transport | Custom binding: NDJSON-RPC on stdio or Unix socket | A2A Layer 3 allows custom bindings; no HTTP stack for a child; works in sandbox | Localhost HTTPS (certs, ports); gRPC (heavy dep) |
| Context saving | RTK approaches R1–R15 copied 1-1; Go implementation always on; `rtk` binary optional exec | User asked for 1-1 copy; a required Rust binary violates spin’s single-static-Go-binary identity | Shell-out-only to `rtk` (breaks offline); invent different filters (not 1-1) |
| Compact apply surface | Shell **and** built-in read/grep/glob/ls | RTK’s documented hole is built-in tools bypassing the Bash hook | Hook-only rewrite (leaves Read/Grep fat) |
| Task framing | First-class `TaskFrame` on every parent turn and every child spawn | Phase-specific assembly; Composer already has stable/dynamic split | Longer system prompt; clone parent history into child |
| Hooks | Finish the existing 10-event runner; add plugin/skill frontmatter hooks; apply `UpdatedInput`; expand `~` | Events already specified in-tree; Claude-compatible mental model | Replace with HTTP-only hooks (weaker for local scripts) |

### Scope

The complete change set — every piece required for the feature to be correct and useful:

1. **Skill runtime** — Discover, validate, catalog, activate, and resolve references for Agent Skills.
2. **Plugin runtime** — Discover, validate, and load Agent Plugins 1.0 packages (skills + MCP + `com.spin.agent` extensions).
3. **Skill tool** — Model-facing `skill` / `load_skill` that injects a body; catalog stays in the system prompt.
4. **Process subagents** — Replace the stub executor with `spin a2a` children; register builtins (`explorer`, `planner`, `reviewer`, `ask_user`) and config/`agents/*.md` specs as Agent Cards.
5. **A2A client and server** — Parent client; child server; local custom binding; remote HTTPS JSON-RPC client; Agent Card documents.
6. **Background agent tasks** — Non-blocking spawn, registry, wait, list, cancel, output/artifacts; persist across parent turns in the session.
7. **Shell background tasks** — Keep `start_process` family; surface both task kinds in one registry view.
8. **Hooks completeness** — Wire `POST_TOOL_USE_FAILURE`, `SUBAGENT_START/STOP`, `PRE_COMPACT`, `STOP`, `SESSION_END`; apply `UpdatedInput`; expand home dir; skill/plugin/agent frontmatter hooks; propagate hooks into children (do not drop on spawn).
9. **RTK compact pipeline** — R1–R15 and the per-command table; auto-rewrite PreToolUse; fail-safe; exit codes; savings accounting; `--no-compact` / verbose raw escape.
10. **Task framing** — `TaskFrame` type; composer section; spawn payload; mode→frame mapping (`regular`, `review`, `compact`, `planning`).
11. **Context assembly** — Call `retrieval.Pipeline.Assemble` on the turn path; progressive disclosure layers; phase-specific bundles for children; unify TUI transcript + ACP history as the parent’s durable context (children stay isolated).
12. **Structured navigation** — Agent-facing index (skills, plugins, sessions, symbols, A2A peers); compact `ls`/`tree`/`find`/`grep`; TUI panes and slash commands (`/skills`, `/tasks`, `/agents`).
13. **UX / visuals** — Timeline blocks for skill load, A2A task state, hook veto, compact savings chip; status-bar task count; approval for child spawn when policy requires; welcome/help text lists the new commands.
14. **Config and CLI** — `spin a2a` child entry; plugin/skill search paths; compact on/off; max concurrent children; `rtk` binary path optional.
15. **Safety** — Plugin path containment; child sandbox inherits parent policy (or tighter allowlist); A2A auth for remote cards; hook veto on `SUBAGENT_START`.
16. **Tests and docs** — Unit, integration, e2e journeys; user-facing how-to for writing a skill and spawning a child.

### Anti-Goals

| Anti-goal | Substantive reason |
|---|---|
| Replace ACP with A2A for IDE hosts | ACP is the host↔agent contract already implemented in `internal/protocol/acp`. A2A is agent↔agent. Collapsing them breaks Zed/VS Code adapters. |
| In-process goroutine as the production subagent runtime | No process isolation, no real A2A peer, contradicts the spawn-as-process requirement. The current `Executor` func remains a test double only. |
| Require the Rust `rtk` binary | Spin’s identity is a single static Go binary (`CGO_ENABLED=0`). Approaches are copied; the binary is optional acceleration / reference. |
| Recursively search nested `skills/**` | Agent Plugins §7.1 forbids discovering skills deeper than immediate children of `skills/`. |
| Interpret plugin `command` args as package paths | Agent Plugins §4.1: only fields defined as plugin-relative paths are contained; treating argv as paths is a spec violation. |
| Shared writable memory between parent and child | A2A opaque execution: children do not see parent tool internals or raw transcript. They receive a TaskFrame + artifacts they are given. |
| JavaScript/Python plugin VM | Wrong primitive for a Go agent; skills already use Markdown + optional scripts the agent runs through existing tools. |
| Silent drop of skill/plugin hooks on child spawn | Documented Claude Code defect; spin must propagate or fail closed, not ignore. |

## 4. Technical Design

### Architecture

**New / extended packages**

| Package | Responsibility |
|---|---|
| `internal/skills` | Parse `SKILL.md`, validate frontmatter, progressive load, reference resolve (one level) |
| `internal/plugins` | Agent Plugins 1.0 loader, containment, MCP hand-off, `com.spin.agent` extensions |
| `internal/protocol/a2a` | Types (Card, Task, Message, Part, Artifact), JSON-RPC client/server, local + HTTPS bindings |
| `internal/agent/child` | Spawn `spin a2a`, lifecycle, env, Agent Card from `subagent.Spec` |
| `internal/agent/tasks` | Unified registry: A2A tasks + existing shell background processes |
| `internal/contexteng/compact` | RTK R1–R15 pipeline and command registry |
| `internal/agent/frame` | `TaskFrame` assemble / serialize |
| `internal/nav` | Structured navigation index |

**Existing packages to finish, not replace**

- `internal/safety/hooks` — wire remaining events; apply `UpdatedInput`; `UserHomeDir`
- `internal/agent/subagent` — keep `Spec` / builtins / semaphore as **admission control**; executor becomes process spawn
- `internal/agent/prompt` — catalog + TaskFrame as Composer sections
- `internal/contexteng/retrieval` — call `Assemble` from harness loop
- `internal/conversation` — remove `ErrSubagentSpawnNotSupported` stub
- `internal/ui/*` + `cmd/spin` — surfaces and `spin a2a`

**Discovery order (skills)**

1. Project: `<workDir>/.agents/skills/`, `<workDir>/.claude/skills/` (interop)
2. User: `~/.spin/skills/`, `~/.agents/skills/`
3. Plugins: each loaded plugin’s `skills/<name>/`
4. Bundled: spin’s own promptkit skills shipped with the binary or `$SPIN_HOME/skills`

Name collisions: project wins, then user, then plugin, then bundled. Duplicate names in the catalog include `source` so the model can disambiguate.

**Plugin discovery**

- `<workDir>/.spin/plugins/<plugin-root>/`
- `~/.spin/plugins/<plugin-root>/`
- Config `plugins.paths`

Reject plugin if `plugin.json` missing or fatally invalid. Skip individual skills/MCP entries that fail containment.

**Child spawn**

```
parent ── exec ──► spin a2a --card <path> --stdio
                 │  inherits: workdir, policy snapshot, TaskFrame JSON on stdin frame 0
                 │  or: --listen unix://$XDG_RUNTIME_DIR/spin/a2a/<id>.sock
parent A2A client ── message/send (blocking | not)
                  ── tasks/get / tasks/cancel
                  ── message/stream → EventSubagent* → TUI mapper
```

Admission: `SUBAGENT_START` hook (blocking), then semaphore (`DefaultMaxConcurrent` and config), then spawn. On exit: `SUBAGENT_STOP`, artifact summary into parent as a single tool result (not the child’s raw transcript).

**TaskFrame (parent and child)**

```text
objective          — one paragraph, what done looks like
phase              — plan | work | review | ask
output_format      — e.g. "unified diff + test names" / "bullet findings"
tools              — allowlist (from Spec / skill allowed-tools)
sources            — paths or retrieval queries, not file bodies
boundaries         — files not to touch, max turns, no network, …
success_criteria   — checkable statements
```

Composer injects a compact TaskFrame section (dynamic, non-cacheable). Children receive **only** the frame + skill body (if forked from a skill) + listed sources. They do not receive parent tool traces.

**Context assembly (parent turn)**

1. Stable Composer sections (identity, goals, tool schemas for **active** tools)
2. Skill catalog (`name` + `description` only)
3. Navigation index (counts + pointers, not trees)
4. Dynamic: AGENTS.md pointer (full file only if under size cap; else first screen + path)
5. `retrieval.Pipeline.Assemble` fragments
6. TaskFrame for this turn
7. Activated skill bodies (this turn)
8. Compacted tool observations (already in harness)
9. History after compaction / `PRE_COMPACT` hook

**Structured navigation**

Agent tool `navigate` (or composed existing tools) returns **index records**, not raw dumps:

- `kind=skill|plugin|session|peer|path|symbol`
- `id`, `title`, `why` (one line), `open` (path or A2A card URL)

Filesystem listings go through R10 tree compression. Grep goes through R2+R3.

**Hooks mapping**

| Spin event | When | Blocking |
|---|---|---|
| SESSION_START | Builder / ACP NewSession | no |
| USER_PROMPT_SUBMIT | `RunTurn` | yes |
| PRE_TOOL_USE | before every tool, including compact rewrite | yes |
| POST_TOOL_USE | after success | no |
| POST_TOOL_USE_FAILURE | after tool error | no |
| SUBAGENT_START | before spawn | yes |
| SUBAGENT_STOP | after child exit | no |
| PRE_COMPACT | before history compact | no |
| STOP | parent loop ending | no |
| SESSION_END | conversation close | no |

Skill/plugin frontmatter hooks register for the session (skill) or child lifetime (agent spec). `UpdatedInput` **must** replace tool arguments when the hook returns JSON with that field (today it is dropped).

**UX / visuals**

- Timeline block types: `SKILL`, `TASK`, `SUBAGENT`, `HOOK`, `COMPACT`
- Status bar: `tasks N/M` + compact savings chip (`−72% · 14kB`)
- Command palette entries: Skills, Tasks, Agents
- Slash: `/skills`, `/skill <name>`, `/tasks`, `/task wait <id>`, `/task cancel <id>`, `/agents`
- Child stream: thinking/content mapped like today’s mapper with `kind=a2a` discriminator (same idea as RLM’s planned ACP tag)
- Hook veto: approval-style overlay with the script’s reason
- Welcome footer documents Ctrl-C still exits the parent; running children receive `tasks/cancel` then SIGTERM

### Non-Functional Requirements

- Performance: skill catalog parse of 200 skills p99 < 50 ms; compact pipeline p99 < 15 ms per command (RTK’s 5–15 ms band); A2A local spawn ready (card received) p99 < 200 ms
- Reliability: child crash → Task state `failed` + stderr artifact; filter panic/error → raw output (R12); hook timeout does not hang the turn (existing runner budget)
- Security: plugin path containment; children inherit a **snapshot** of approval policy (cannot loosen); remote A2A requires explicit allowlist of card URLs; hooks remain workspace-trusted scripts
- Observability: every compact invocation logs command, strategy IDs, bytes in/out, exit code; every A2A method logs task id and state transition; TUI debug log already exists (`~/.spin/spin.log`)

### Testing Strategy

- **Unit:** frontmatter validation (name/dir match, hyphen rules); plugin containment (reject `../`); compact fixtures per command in the RTK table (golden stdout + preserved exit code); TaskFrame serialization; hook `UpdatedInput` applied
- **Integration:** load a sample Agent Plugin (skill + mcp.json + `com.spin.agent/hooks`); spawn `spin a2a` and `message/send` blocking; background send then `tasks/get`; `SUBAGENT_START` exit 2 vetoes spawn
- **E2E:** TUI journey — user asks for a skill-covered task, catalog activates, body loads, child explorer runs in background, parent continues, `/task wait` joins, timeline shows compact `go test` failures-only; ACP session sees the same task notifications

### Migration & Compatibility

- Existing TUI/exec/ACP sessions keep working with compact **on** by default; `SPIN_COMPACT=0` or config `compact.enabled: false` restores raw tool output
- `internal/agent/subagent.Executor` signature stays for tests; production builder injects the process executor
- Hook script filenames unchanged (`pre-tool-use`, …)
- promptkit-generated `.agents/skills` load without being repackaged as plugins
- RLM spec (`specs/rlm/SPEC.md`) remains a separate execution mode; RLM `CallLM` may later target an A2A child, but this spec does not change Yaegi

### Dependencies

| Dep | Assessment |
|---|---|
| None required for compact / skills / plugins / local A2A | JSON + yaml already in module |
| Optional `rtk` executable on PATH | If present and `compact.backend: rtk`, PreToolUse rewrite uses it; otherwise Go pipeline |
| No new gRPC/HTTP framework for local children | `pkg/protocol/jsonrpc` + stdlib `net` for Unix sockets |
| Remote A2A | `net/http` already used by MCP/web tools |

## 5. User Journey

### Persona

**Alex**, a Go engineer running `spin` in a repo that already has `.agents/skills` from promptkit and a third-party Agent Plugin for deploy. They use TUI daily and ACP from an editor. They want one Ctrl-C to leave a clean shell, compact test output, and the ability to farm review to a child while they keep typing.

### CJM Phases

**1. Arrive**
- Action: `spin` in the repo. Welcome banner + catalog count (`12 skills · 1 plugin`).
- Pain: catalog dump of full skill bodies would blow the window — must stay metadata-only.
- Success: status bar shows compact on; `/skills` lists names and one-line descriptions.

**2. Frame the task**
- Action: type “review the ACP empty-response fix and run tests”.
- Pain: vague ask would force a repo walk.
- Success: parent TaskFrame phase=`review`, sources=touched files from git compact status, success_criteria include `go test` failures-only.

**3. Skill activation**
- Action: model calls `skill` for `implement` or `bug` when the ask matches `description`.
- Pain: wrong skill, or body never loaded so the model freestyles.
- Success: timeline `SKILL` block; body in context; references unread until linked.

**4. Compact tools**
- Action: `git status`, `git diff`, `go test`.
- Pain: raw 2k-line test log.
- Success: R9/R5 output; chip `−81%`; exit code 1 still fails the turn.

**5. Spawn and wait**
- Action: parent spawns `reviewer` child (process). User keeps chatting. Later `/task wait abc12`.
- Pain: child looks “stuck”; no way to cancel; hooks missing in child.
- Success: Task block `working` → `completed`; artifact is a review list; child hooks fired; parent transcript gained one summary.

**6. Leave**
- Action: Ctrl-C.
- Pain: leftover TUI or orphan children.
- Success: `tasks/cancel` + SIGTERM to children, `SESSION_END` hooks, screen clear (existing TUI teardown).

### Friction Map

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Skill ecosystem vs promptkit-only paths | Arrive | Load both Agent Skills dirs and plugin `skills/` with source tags |
| Built-in Read bypasses Bash rewrite | Compact tools | Run R8 on `read` / grep / glob tools, not only `exec` |
| Child spawn feels like a hung parent | Spawn and wait | Explicit background vs wait; TUI task pane; `/task wait` |
| Hook veto with no UI | Spawn / tools | Overlay with script stdout; same block as approval |
| Two session stores (TUI JSONL vs ACP history) | Arrive / Leave | One parent context log; children never share it |

### North star

Alex installs a community Agent Plugin, spin lists its skills next to promptkit’s, a review child runs as a process they can wait on, `go test` appears as two failing names, and quitting leaves a blank terminal with no orphan `spin a2a` processes.

## 6. Risks & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| Compact filters hide the one line that explained a failure | Wrong agent action | medium | R12 fail-safe; R9 keeps failure bodies; `-l none` / `SPIN_COMPACT=0`; never rewrite exit codes (R13) |
| Local A2A binding drifts from HTTPS JSON-RPC types | Dual stacks | medium | One `internal/protocol/a2a` types package generated/kept in sync with A2A 1.0 objects; bindings only map transport |
| Plugin MCP start failure looks like a skill outage | User distrust | low | Independent failure (spec §); TUI reports skipped servers, skills still listed |
| Child process leak on parent crash | Resource leak | medium | Session task registry + `spin a2a` writes pid file under `XDG_RUNTIME_DIR`; parent start reaps orphans; children die when stdin/socket closes |
| Skill body + AGENTS.md + catalog exceed window | Quality drop | medium | Progressive disclosure; AGENTS.md size cap + pointer; compact history; optional RLM mode for huge inputs (existing spec) |
| Hook scripts from plugins escape the plugin root | Security | medium | Containment on package paths; hooks execute with plugin root as cwd and no extra env secrets |
| Semaphore + max shell tasks deadlock the parent | Hang | low | Separate budgets; wait API must not hold the ReAct lock while blocking on a child |

## 7. Open Questions

- Should `ask_user` remain a child Agent Card or stay an in-parent approval overlay? (Process isolation vs latency for a single question.)
- Default local binding: stdio (simpler, one fd story) vs Unix socket (inspectable, multi-client)? Spec allows both; pick a default at implement time with a test proving the other.
- Do we emit A2A `pushNotification` for long children, or only parent-side polling/`message/stream`?
- Skill `allowed-tools` is experimental in Agent Skills — enforce as a hard allowlist when present, or advisory?
- Remote A2A auth: API key header vs mTLS vs existing `internal/auth`? Needs a concrete first remote peer.

## 8. Implementation Roadmap

Ordered workstreams. Each item is in Scope; order is dependency, not deferral.

1. **Skills parse + catalog** — `internal/skills`; Composer catalog section; `/skills`; tests against agentskills.io constraints.
2. **Skill tool + progressive load** — activate body; one-level references; e2e that a third-party `SKILL.md` runs.
3. **Plugins 1.0** — `internal/plugins`; containment; MCP hand-off; `com.spin.agent` hooks dir.
4. **Hooks finish** — remaining events; `UpdatedInput`; `~` expansion; frontmatter hooks; child propagation.
5. **Compact pipeline R1–R15** — command registry + goldens; PreToolUse rewrite; apply to read/grep/glob; savings chip; optional `rtk` backend.
6. **TaskFrame + assembly** — `internal/agent/frame`; wire `retrieval.Assemble`; mode mapping.
7. **A2A types + local server** — `spin a2a`; Agent Card from `subagent.Spec`; NDJSON-RPC.
8. **A2A client + process spawn** — replace stub; `SUBAGENT_*` hooks; semaphore; artifacts as tool results.
9. **Task registry** — wait/list/cancel; unify with shell background; persist in session.
10. **Structured navigation + UX** — `internal/nav`; timeline blocks; status; slash commands; welcome/help; ACP notifications for tasks.
11. **Remote A2A client** — HTTPS JSON-RPC against a published card on the allowlist.
12. **Docs** — how-to: write a skill, package a plugin, spawn/wait a child; reference: compact table, hook events, A2A binding.

## Sources

- [Agent Skills specification](https://agentskills.io/specification)
- [addyosmani/agent-skills](https://github.com/addyosmani/agent-skills) / [skill anatomy](https://github.com/addyosmani/agent-skills/blob/main/docs/skill-anatomy.md)
- [Agent Plugins](https://agent-plugins.org/) / [specification](https://agent-plugins.org/specification) / [plugin.json](https://agent-plugins.org/plugin-authors/manifest)
- [A2A 1.0 specification](https://a2a-protocol.org/latest/specification/)
- [rtk README](https://github.com/rtk-ai/rtk) / [ARCHITECTURE.md](https://github.com/rtk-ai/rtk/blob/master/docs/contributing/ARCHITECTURE.md)
- [Claude Code skills](https://code.claude.com/docs/en/skills) / [subagents](https://code.claude.com/docs/en/sub-agents) / [hooks](https://code.claude.com/docs/en/hooks)
- [Phase-specific context assembly](https://agentpatterns.ai/context-engineering/phase-specific-context-assembly/)
- [Progressive disclosure (LangChain)](https://docs.langchain.com/oss/python/langchain/multi-agent/skills-sql-assistant)
- In-tree: `internal/agent/subagent`, `internal/safety/hooks`, `internal/conversation/builder.go`, `internal/contexteng`, `internal/agent/prompt`, `internal/protocol/acp`, `specs/rlm/SPEC.md`
