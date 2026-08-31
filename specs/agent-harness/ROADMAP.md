# ROADMAP: Agent harness (skills, plugins, process subagents, context)

Source spec: [`SPEC.md`](./SPEC.md).

Each item is one user journey. Items are ordered by dependency. Every item ships value on its own and is independently testable. Integrate existing packages (`internal/agent/subagent`, `internal/safety/hooks`, `internal/agent/prompt`, `internal/contexteng`, `internal/agent/executor`, `internal/commands`, `internal/protocol/acp`) — do not rebuild them.

`/implement` authors `specs/journeys/JOURNEY-agent-harness-<n>-<slug>.md` for the item it takes.

---

### Step 1: Parse and validate Agent Skills

**Description:** Add `internal/skills` that parses a `SKILL.md` directory into a validated record (frontmatter + body). Enforce the Agent Skills name/description rules and `name` == parent directory. Prove the in-repo `.agents/skills/*/SKILL.md` files parse.

**DoR (Definition of Ready):**
- [`SPEC.md`](./SPEC.md) §2 Agent Skills directory and §4 skill runtime reviewed
- No Go skill loader exists (confirmed)

**DoD (Definition of Done):**
- [x] `internal/skills` exports `Parse(dir string) (Skill, error)` and `Validate(Skill) error`
- [x] `name` rejects uppercase, leading/trailing hyphen, consecutive hyphens, length > 64, mismatch with directory
- [x] `description` rejects empty and length > 1024
- [x] Optional fields (`license`, `compatibility`, `metadata`, `allowed-tools`) parse when present and are omitted when absent
- [x] Body is the Markdown after the closing `---` (frontmatter not mixed into body)
- [x] Table tests cover valid/invalid fixtures under `internal/skills/testdata/`
- [x] A test walks `.agents/skills/*/SKILL.md` and requires every shipped skill to parse
- [x] Catalog parse of 200 synthetic skills p99 < 50 ms (benchmark or timed test)
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-001-parse-and-validate-agent-skills.md`; implementation in `internal/skills/parse.go`, `internal/skills/validate.go`, `internal/skills/skill.go`, `internal/skills/parse_test.go`; closed at sequence [6].

**Risks:** Existing promptkit skills fail strict `name` rules. Mitigation: fix those `SKILL.md` files in the same change; do not relax the spec.

**Files likely affected:** `internal/skills/parse.go`, `internal/skills/parse_test.go`, `internal/skills/testdata/`, `.agents/skills/*/SKILL.md`

---

### Step 2: Discover the skill catalog and list it to the user

**Description:** Scan project, user, and bundled skill roots. Build a metadata-only catalog (`name`, `description`, `location`, `source`). Inject that catalog as a Composer section. Add `/skills` so the operator sees the same list the model sees.

**DoR (Definition of Ready):**
- Step 1 complete
- `internal/agent/prompt.Composer` and `internal/commands` registry reviewed

**DoD (Definition of Done):**
- [x] Discovery order: `<workDir>/.agents/skills/`, `<workDir>/.claude/skills/`, `~/.spin/skills/`, `~/.agents/skills/`, bundled — project wins on name collision; catalog entries include `source`
- [x] Composer section contains **only** name + description (no skill bodies)
- [x] `/skills` prints one line per skill (name, source, description) and is documented in `/help`
- [x] TUI and `exec` both see the catalog when a workdir has `.agents/skills`
- [x] Unit tests: collision source tag, missing roots ignored, empty catalog is valid
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-002-discover-skill-catalog.md`; implementation in `internal/skills/discover.go`, `internal/skills/catalog.go`, `internal/agent/prompt/catalog.go`, `internal/commands/skills.go`, `internal/conversation/builder.go`; closed at sequence [11].

**Risks:** Catalog dumps full bodies and blows the window. Mitigation: test that the composed section contains no `##` body heading from a fixture `SKILL.md`.

**Files likely affected:** `internal/skills/discover.go`, `internal/agent/prompt/sections.go`, `internal/commands/skills.go`, `cmd/spin/tui.go`, `cmd/spin/tui_command_context.go`

---

### Step 3: Activate a skill body with progressive disclosure

**Description:** Add a `skill` tool that loads one skill’s body into the current turn. Resolve `scripts/` / `references/` / `assets/` links one level deep from the skill root only when the model reads them. Emit a timeline `SKILL` block.

**DoR (Definition of Ready):**
- Steps 1–2 complete
- `internal/tools` registry and `internal/tui/mapper.go` block types reviewed

**DoD (Definition of Done):**
- [x] `skill` / `load_skill` tool takes a skill name, returns body + skill root path
- [x] Unknown name returns a typed error listing catalog names (no body leak)
- [x] Relative file reads from the skill stay inside the skill root (reject `../`)
- [x] Nested reference chains beyond one hop from `SKILL.md` are not auto-loaded
- [x] `allowed-tools` when present is recorded on the activation (enforced in a later step if experimental; field is not dropped)
- [x] Mapper renders a `SKILL` block with name + source
- [x] Integration test: catalog in prompt, `skill` call injects body, reference file unread until requested
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-003-activate-skill-body.md`; implementation in `internal/tools/skill.go`, `internal/skills/activate.go`, `internal/skills/resolve.go`, `internal/tui/mapper.go`, `internal/ui/blocks/`; closed at sequence [15].

**Risks:** Activating a skill reloads every reference. Mitigation: test that opening `SKILL.md` does not read `references/*.md` until a follow-up read.

**Files likely affected:** `internal/tools/skill.go`, `internal/tools/registry.go`, `internal/tui/mapper.go`, `internal/ui/blocks/`, `internal/skills/resolve.go`

---

### Step 4: Parse Agent Plugins 1.0 with path containment

**Description:** Add `internal/plugins` that loads `plugin.json` against the closed schema and rejects package paths that escape the plugin root. No MCP start and no skill merge yet — validation is the user-visible value (`spin plugin validate` or equivalent test entry).

**DoR (Definition of Ready):**
- [`SPEC.md`](./SPEC.md) §2 Agent Plugins 1.0 and §4 plugin runtime reviewed
- Step 1 available so discovered `skills/<name>/SKILL.md` can be parsed

**DoD (Definition of Done):**
- [x] `plugin.json` requires `$schema` and `name`; unknown top-level fields are reported and ignored; other schema violations reject the plugin
- [x] Permitted fields: `version`, `description`, `author`, `homepage`, `repository`, `license`, `keywords`, `extensions`
- [x] Plugin-relative paths must start with `./` and resolve inside the plugin root; `../` and bare `data` fail
- [x] Skills are immediate children of `skills/` only (no recursive search)
- [x] Missing `plugin.json` rejects the whole plugin; missing `skills/` is valid (zero skills)
- [x] Testdata fixtures: valid plugin, escape path, nested skill ignored, unknown field ignored
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-004-parse-agent-plugins.md`; implementation in `internal/plugins/plugin.go`, `internal/plugins/manifest.go`, `internal/plugins/contain.go`, `internal/plugins/load.go`, `cmd/spin/plugin.go`; closed at sequence [19].

**Risks:** Treating MCP `command` argv as package paths (spec violation). Mitigation: containment applies only to fields defined as plugin-relative paths.

**Files likely affected:** `internal/plugins/manifest.go`, `internal/plugins/contain.go`, `internal/plugins/testdata/`, `cmd/spin/plugin.go`

---

### Step 5: Load plugins — merge skills, isolate MCP failures

**Description:** Discover plugin roots (`<workDir>/.spin/plugins/*`, `~/.spin/plugins/*`, config `plugins.paths`). Merge each plugin’s skills into the Step 2 catalog with `source=plugin:<name>`. Map `mcp.json` into the existing MCP manager. A failing MCP server must not unload that plugin’s skills.

**DoR (Definition of Ready):**
- Steps 2 and 4 complete
- `internal/mcp` manager/registry reviewed

**DoD (Definition of Done):**
- [x] Valid plugins contribute skills to `/skills` with plugin source tags
- [x] `mcp.json` `$schema` + `mcpServers` loaded; empty `mcpServers` is valid
- [x] Transport types `stdio`, `streamable-http`, `sse` are explicit; no guessed transport
- [x] A fixture whose MCP command fails still lists its skills
- [x] Fatal `plugin.json` skips that plugin only; other plugins still load
- [x] Integration test uses a sample plugin (skill + mcp.json)
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-005-load-plugins-merge-skills.md`; implementation in `internal/plugins/discover.go`, `internal/plugins/mcpjson.go`, `internal/plugins/attach.go`, `internal/skills/discover.go`, `internal/config/config_v2.go`, `internal/mcp/transport.go`; closed at sequence [22].

**Risks:** One bad MCP entry disables the catalog. Mitigation: independent failure test is required DoD, not optional.

**Files likely affected:** `internal/plugins/load.go`, `internal/mcp/`, `internal/skills/discover.go`, `internal/config/config_v2.go`

---

### Step 6: Load `com.spin.agent` extension hooks from plugins

**Description:** Read `extensions["com.spin.agent"]` and/or the `com.spin.agent/hooks/` directory. Register those scripts with the existing hook runner without putting spin fields on portable `plugin.json` keys.

**DoR (Definition of Ready):**
- Step 5 complete
- `internal/safety/hooks.Runner` and script-name map reviewed

**DoD (Definition of Done):**
- [x] `com.spin.agent/hooks/pre-tool-use` (etc.) are discovered using existing `Event.ScriptName()` filenames
- [x] Foreign extension dirs (`com.example.client/`) are ignored
- [x] Plugin hook scripts execute with plugin root as cwd
- [x] Package-path containment still applies to hook files inside the plugin
- [x] Unit test: plugin hook fires on `PRE_TOOL_USE` in a fake runner
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-006-load-com-spin-agent-extension-hooks.md`; implementation in `internal/plugins/extensions.go`, `internal/safety/hooks/runner.go`, `internal/safety/hooks/runner_test.go`; closed at sequence [26].

**Risks:** Plugin hooks run with host secrets in env. Mitigation: do not inject extra credentials; inherit the same env policy as project hooks.

**Files likely affected:** `internal/plugins/extensions.go`, `internal/safety/hooks/runner.go`, `internal/safety/hooks/runner_test.go`

---

### Step 7: Finish the hook runner contract

**Description:** Expand `~` in `GlobalDir`. Apply `HookResult.UpdatedInput` to tool arguments. Keep existing script filenames and exit-code-2 block semantics.

**DoR (Definition of Ready):**
- `internal/safety/hooks` and `internal/agent/tool/runtime.go` reviewed
- Spec §4 hooks mapping reviewed

**DoD (Definition of Done):**
- [x] `filepath.Join("~", …)` is not used; `os.UserHomeDir` (or equivalent) expands the global hooks dir
- [x] ACP and TUI builders pass an expanded global dir
- [x] When a blocking hook returns JSON `updated_input`, the tool sees the replaced arguments
- [x] When `updated_input` is empty, original arguments are unchanged
- [x] Tests cover home expansion and argument replacement
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-007-finish-the-hook-runner-contract.md`; implementation in `internal/safety/hooks/dir.go`, `internal/safety/hooks/runner.go`, `internal/agent/tool/runtime.go`, `cmd/spin/acp.go`, `internal/conversation/builder.go`, `internal/conversation/agent.go`; closed at sequence [30].

**Risks:** Applying `updated_input` as a raw string to structured tool args corrupts JSON. Mitigation: define the contract (full argv/JSON object replace) and test both shell and structured tools.

**Files likely affected:** `internal/safety/hooks/runner.go`, `internal/agent/tool/runtime.go`, `cmd/spin/acp.go`, `internal/conversation/builder.go`

---

### Step 8: Wire every defined lifecycle hook event

**Description:** Fire the events already declared in `internal/safety/hooks/event.go` that have no production call site: `POST_TOOL_USE_FAILURE`, `PRE_COMPACT`, `STOP`, `SESSION_END`. `SUBAGENT_*` get call sites in Step 19; this step adds the parent-side emitters and tests with a stub manager.

**DoR (Definition of Ready):**
- Step 7 complete
- Harness loop, conversation close, and tool runtime reviewed

**DoD (Definition of Done):**
- [x] Tool error path executes `POST_TOOL_USE_FAILURE`
- [x] Compactor path executes `PRE_COMPACT` before history rewrite
- [x] Parent loop end / `Conversation.Close` execute `STOP` then `SESSION_END`
- [x] Existing wired events (`SESSION_START`, `USER_PROMPT_SUBMIT`, `PRE/POST_TOOL_USE`) still fire
- [x] Integration test registers recording scripts for all ten names and asserts the parent-side subset
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-008-wire-every-defined-lifecycle-hook-event.md`; implementation in `internal/agent/tool/runtime.go`, `internal/agent/harness/loop.go`, `internal/agent/harness/executor.go`, `internal/conversation/conversation.go`, `internal/conversation/builder.go`, `cmd/spin/tui.go`; closed at sequence [34].

**Risks:** `SESSION_END` never runs on Ctrl-C. Mitigation: hook it from the same teardown that already clears the TUI (`runTUI` / `stopTUILoop`).

**Files likely affected:** `internal/agent/tool/runtime.go`, `internal/agent/harness/loop.go`, `internal/conversation/conversation.go`, `cmd/spin/tui.go`

---

### Step 9: Compact pipeline core (fail-safe, exit codes, accounting)

**Description:** Add `internal/contexteng/compact` implementing RTK R12–R15: filter error → raw output; never change exit code; unknown commands passthrough; `ceil(bytes/4)` savings ledger. No command-specific filters yet — the identity/passthrough path is the user-visible value (raw output + ledger of 0%).

**DoR (Definition of Ready):**
- [`SPEC.md`](./SPEC.md) RTK R12–R15 reviewed
- No compact package exists (confirmed)

**DoD (Definition of Done):**
- [x] `Pipeline.Apply(cmd, stdout, stderr, exitCode) Result` returns same exit code always
- [x] Panic or filter error yields original stdout/stderr and records strategy `R12`
- [x] Unknown command yields unchanged bytes and 0% reduction (`R14`)
- [x] Ledger records bytes in/out and `ceil(in/4)-ceil(out/4)` (`R15`)
- [x] Unit tests for panic, unknown command, nonzero exit preserved
- [x] Compact p99 < 15 ms on a 64 KiB unknown-command fixture
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-009-compact-pipeline-core.md`; implementation in `internal/contexteng/compact/pipeline.go`, `internal/contexteng/compact/pipeline_test.go`, `internal/contexteng/compact/ledger.go`; closed at sequence [38].

**Risks:** Ledger uses a real tokenizer and disagrees with RTK. Mitigation: use `ceil(bytes/4)` only.

**Files likely affected:** `internal/contexteng/compact/pipeline.go`, `internal/contexteng/compact/pipeline_test.go`, `internal/contexteng/compact/ledger.go`

---

### Step 10: Compact command registry (RTK table, 1-1)

**Description:** Implement R1–R11 filters and the per-command table in the spec (ls/tree, read, grep/rg, git status/diff/log/add/commit/push/pull, go test, pytest/jest/vitest/playwright, ruff, docker ps). Goldens are the contract.

**DoR (Definition of Ready):**
- Step 9 complete
- Spec compact table and R1–R11 reviewed

**DoD (Definition of Done):**
- [x] Each table row has a golden fixture: raw stdin-like output → compact stdout, same exit code
- [x] `go test` NDJSON: passing tests collapsed, failures kept (`R9`)
- [x] `git status`: grouped by state (`R5`)
- [x] `ls`/`tree`: hierarchy + per-dir counts (`R10`)
- [x] `read` levels `none|minimal|aggressive` (`R8`)
- [x] `grep`/`rg`: truncated lines, grouped by file (`R2`+`R3`)
- [x] Dedup collapses repeated log lines with `×N` (`R4`)
- [x] Filter tests do not call the network or real git
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-010-compact-command-registry.md`; implementation in `internal/contexteng/compact/registry.go`, `internal/contexteng/compact/git.go`, `internal/contexteng/compact/gotest.go`, `internal/contexteng/compact/testdata/`; closed at sequence [42].

**Risks:** Goldens drift from upstream RTK. Mitigation: document the fixture source in each testdata README; behavior matches the spec table, not a live `rtk` binary.

**Files likely affected:** `internal/contexteng/compact/registry.go`, `internal/contexteng/compact/git.go`, `internal/contexteng/compact/gotest.go`, `internal/contexteng/compact/testdata/`

---

### Step 11: Apply compact to shell exec and PreToolUse rewrite

**Description:** Run the pipeline on `shell_command` / exec results. Auto-rewrite (R11): PreToolUse (or equivalent) prefixes compacting so the agent does not have to type a wrapper. Optional `compact.backend: rtk` execs `rtk` when present.

**DoR (Definition of Ready):**
- Steps 7 and 10 complete
- `internal/tools/shell_command.go` reviewed

**DoD (Definition of Done):**
- [x] Shell tool results are compacted before they enter conversation history
- [x] Nonzero shell exit stays nonzero after compact
- [x] Rewrite is argv-level (e.g. `git status` → compact-aware execution) with zero extra model tokens
- [x] `compact.backend: rtk` uses PATH `rtk` when the binary exists; falls back to the Go pipeline when it does not
- [x] `SPIN_COMPACT=0` / `compact.enabled: false` skips rewrite and filters
- [x] Integration test: fake `git status` blob becomes compact text in a harness observation
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-011-apply-compact-to-shell-exec.md`; implementation in `internal/tools/shell_command.go`, `internal/agent/tool/runtime.go`, `internal/config/config_v2.go`, `internal/contexteng/compact/rewrite.go`; closed at sequence [47].

**Risks:** Double-compact if rewrite and post-filter both apply. Mitigation: one apply site per tool result; test idempotence.

**Files likely affected:** `internal/tools/shell_command.go`, `internal/agent/tool/runtime.go`, `internal/config/config_v2.go`, `internal/safety/hooks/`

---

### Step 12: Apply compact to built-in read, grep, glob, and ls

**Description:** Close RTK’s documented hole: built-in tools that never go through the Bash hook still run R8/R2/R10. Same escape hatch as Step 11.

**DoR (Definition of Ready):**
- Steps 10–11 complete
- `internal/tools/read_file.go`, grep/glob/list_directory tools reviewed

**DoD (Definition of Done):**
- [x] `read_file` output uses code-filter levels from config (default `minimal`)
- [x] Directory listing uses tree compression (`R10`)
- [x] Grep/search grouping + line truncation apply
- [x] Escape hatch disables compact on these tools too
- [x] Tests use in-memory fixtures, not the user’s repo
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md`; implementation in `internal/tools/read_file.go`, `internal/tools/list_directory.go`, `internal/tools/file_search.go`, `internal/tools/grep.go`, `internal/tools/compact_apply.go`, `internal/filesearch/grep.go`; closed at sequence [51].

**Risks:** Aggressive read strips a function body the agent needed. Mitigation: default `minimal`; TaskFrame or tool arg can request `none`.

**Files likely affected:** `internal/tools/read_file.go`, `internal/tools/list_directory.go`, `internal/tools/grep.go`, `internal/filesearch/`

---

### Step 13: Compact status chip and operator escape

**Description:** Show compact state and last-turn savings on the TUI status bar. Document `/help` and env/config escape. Default remains compact **on**.

**DoR (Definition of Ready):**
- Steps 11–12 complete
- `internal/ui/status` aggregator/formatter reviewed

**DoD (Definition of Done):**
- [x] Status bar shows compact on/off and a savings chip (`−N%` from the ledger)
- [x] Chip uses ledger bytes, not a tokenizer
- [x] `/help` documents `SPIN_COMPACT=0` and config key
- [x] Welcome/status does not claim compact is on when disabled
- [x] Unit tests for formatter with zero, mid, and 100% reduction
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-013-compact-status-chip-and-operator-escape.md`; implementation in `internal/ui/status/formatter.go`, `internal/ui/status/aggregator.go`, `cmd/spin/tui.go`, `internal/commands/help.go`; closed at sequence [55].

**Risks:** Chip implies bill savings. Mitigation: label is output-bytes reduction, matching RTK’s disclaimer.

**Files likely affected:** `internal/ui/status/formatter.go`, `internal/ui/status/aggregator.go`, `cmd/spin/tui.go`, `internal/commands/help.go`

---

### Step 14: TaskFrame on every parent turn

**Description:** Add `internal/agent/frame.TaskFrame` (objective, phase, output_format, tools, sources, boundaries, success_criteria). Inject it as a dynamic Composer section. Map `/mode` values (`regular`, `review`, `compact`, `planning`) to phases.

**DoR (Definition of Ready):**
- [`SPEC.md`](./SPEC.md) TaskFrame section reviewed
- `internal/agent/prompt.Composer` and `cmd/spin/mode.go` reviewed

**DoD (Definition of Done):**
- [x] `TaskFrame` serializes to a stable JSON/text form for child spawn later
- [x] Composer dynamic part includes the frame; stable/cacheable part does not
- [x] `/mode review` yields phase `review` (and likewise for the other modes)
- [x] Sources are paths/queries, not file bodies
- [x] Unit tests for mode mapping and “no bodies in frame”
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-014-taskframe-on-every-parent-turn.md`; implementation in `internal/agent/frame/frame.go`, `internal/agent/prompt/sections.go`, `internal/conversation/conversation.go`, `internal/conversation/builder.go`, `cmd/spin/mode.go`, `cmd/spin/acp.go`; closed at sequence [59].

**Risks:** Frame duplicates AGENTS.md and explodes tokens. Mitigation: keep fields short; test max rendered size on a fixture.

**Files likely affected:** `internal/agent/frame/frame.go`, `internal/agent/prompt/sections.go`, `cmd/spin/mode.go`, `internal/conversation/`

---

### Step 15: Assemble retrieval on the turn path

**Description:** Call the existing `retrieval.Pipeline.Assemble` from the harness/conversation turn so ACE (and other) fragments actually enter context. Do not rebuild the pipeline.

**DoR (Definition of Ready):**
- Step 14 complete (frame can list `sources`)
- `internal/contexteng/retrieval` and harness loop reviewed

**DoD (Definition of Done):**
- [x] `GetRetrievalPipeline()` is no longer unused on the production turn path
- [x] Assembled fragments appear in the dynamic prompt or observation path
- [x] Nil pipeline is a no-op (ACP/TUI without ACE still run)
- [x] Integration test with a fake source proves Assemble output is present in the composed turn
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-015-assemble-retrieval-on-the-turn-path.md`; implementation in `internal/agent/harness/loop.go`, `internal/agent/harness/executor.go`, `internal/conversation/conversation.go`, `internal/agent/harness/bridge/turn_executor.go`; closed at sequence [63].

**Risks:** Double-injection with ACE middleware. Mitigation: one call site; test token-count does not duplicate the same fragment.

**Files likely affected:** `internal/agent/harness/loop.go`, `internal/conversation/`, `internal/contexteng/retrieval/`

---

### Step 16: A2A types and local JSON-RPC codec

**Description:** Add `internal/protocol/a2a` with Agent Card, Task, Message, Part, Artifact, and the methods `message/send`, `message/stream`, `tasks/get`, `tasks/list`, `tasks/cancel`, card fetch. Codec is JSON-RPC 2.0. Local custom binding: NDJSON or Content-Length (reuse `pkg/protocol/jsonrpc` if it fits). No process spawn yet — in-memory client/server tests.

**DoR (Definition of Ready):**
- [`SPEC.md`](./SPEC.md) A2A section reviewed
- `pkg/protocol/jsonrpc` reviewed for reuse vs NDJSON

**DoD (Definition of Done):**
- [x] Types cover Card, Task, Message, Part, Artifact
- [x] Client and server round-trip `message/send` and `tasks/get` over a pipe
- [x] Invalid JSON-RPC yields standard error codes; A2A domain errors use the −3200x range where specified
- [x] No HTTP listener in this step
- [x] Unit tests do not require a spin binary
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-016-a2a-types-and-local-json-rpc-codec.md`; implementation in `internal/protocol/a2a/card.go`, `internal/protocol/a2a/task.go`, `internal/protocol/a2a/message.go`, `internal/protocol/a2a/codec.go`, `internal/protocol/a2a/codec_test.go`; closed at sequence [67].

**Risks:** Dual type stacks for local vs HTTPS. Mitigation: one types package; bindings only map transport.

**Files likely affected:** `internal/protocol/a2a/types.go`, `internal/protocol/a2a/codec.go`, `internal/protocol/a2a/codec_test.go`, `pkg/protocol/jsonrpc/`

---

### Step 17: Local A2A server process (`spin a2a`)

**Description:** Add `spin a2a` that serves Step 16 over stdio (and optionally a Unix socket). Build an Agent Card from `subagent.Spec` (builtins: explorer, planner, reviewer, ask_user). The child runs an isolated harness with a TaskFrame, not the parent transcript.

**DoR (Definition of Ready):**
- Steps 14 and 16 complete
- `internal/agent/subagent` Spec/Builtins reviewed

**DoD (Definition of Done):**
- [x] `spin a2a --spec explorer --stdio` prints/serves a card then answers `message/send`
- [x] Child process has its own conversation; parent history is not copied
- [x] Card skills/capabilities derive from the Spec allowlist
- [x] `--listen unix://…` accepted or explicitly documented as the alternate binding with a test
- [x] Local spawn ready (card received) p99 < 200 ms in an integration test
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-017-local-a2a-server-process.md`; implementation in `cmd/spin/a2a.go`, `cmd/spin/root.go`, `internal/agent/child/`, `internal/agent/subagent/builtins.go`; closed at sequence [71].

**Risks:** Child inherits parent’s stdout and corrupts TUI. Mitigation: stdio A2A uses the pipes; child logs go to a file or stderr only when not the RPC stream.

**Files likely affected:** `cmd/spin/a2a.go`, `internal/agent/child/server.go`, `internal/agent/subagent/builtins.go`

---

### Step 18: Spawn process children from the parent (replace the stub)

**Description:** Production `subagent.Manager` executor starts `spin a2a`, speaks A2A, and returns the artifact summary. Keep `Executor` as the test double. Semaphore (`DefaultMaxConcurrent`) remains admission control.

**DoR (Definition of Ready):**
- Step 17 complete
- `internal/conversation/builder.go` stub and `ErrSubagentSpawnNotSupported` reviewed

**DoD (Definition of Done):**
- [x] Builder no longer returns `ErrSubagentSpawnNotSupported` on a real spawn
- [x] Spawn is an OS process (test asserts `pid > 0` / extra process)
- [x] Blocking send waits for Task completed/failed and returns artifact text
- [x] Child crash → Task `failed` + stderr artifact, parent harness survives
- [x] Events `EventSubagentSpawn` / `EventSubagentComplete` emit
- [x] Config `Subagents` map can register extra specs
- [x] Integration test uses the built `build/bin/spin` or a test helper binary
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-018-spawn-process-children-from-the-parent.md`; implementation in `internal/agent/child/spawn.go`, `internal/agent/child/spawn_send.go`, `internal/agent/child/spawn_exec.go`, `internal/agent/child/spawn_bin.go`, `internal/conversation/builder.go`, `internal/agent/subagent/manager.go`; closed at sequence [75].

**Risks:** Tests spawn forever. Mitigation: child `MaxIterations` low; test timeout; kill on `t.Cleanup`.

**Files likely affected:** `internal/conversation/builder.go`, `internal/agent/child/spawn.go`, `internal/agent/subagent/manager.go`, `internal/events/event.go`

---

### Step 19: Subagent hooks and no silent drop on spawn

**Description:** Fire `SUBAGENT_START` (blocking) before spawn and `SUBAGENT_STOP` after exit. Copy parent + skill/plugin/agent-frontmatter hooks into the child (do not drop them). Exit 2 on start vetoes spawn.

**DoR (Definition of Ready):**
- Steps 6, 8, and 18 complete

**DoD (Definition of Done):**
- [x] `SUBAGENT_START` exit 2 prevents process start (no pid)
- [x] `SUBAGENT_STOP` runs on success, failure, and crash
- [x] A skill/plugin hook that fired in the parent is registered in the child (test with a marker file)
- [x] Missing child hooks is a test failure, not a log line
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md`; implementation in `internal/agent/child/spawn_exec.go`, `internal/agent/child/spawn_hooks.go`, `internal/agent/child/harness.go`, `internal/safety/hooks/inherit.go`, `internal/conversation/builder.go`, `internal/agent/subagent/spec.go`; closed at sequence [79].

**Risks:** Recursion if the child re-spawns. Mitigation: children do not get the spawn tool unless the Spec allowlists it; test deny-by-default.

**Files likely affected:** `internal/agent/child/spawn.go`, `internal/safety/hooks/`, `internal/conversation/builder.go`

---

### Step 20: Agent task registry — wait, list, cancel, persist

**Description:** Add `internal/agent/tasks` for A2A tasks: non-blocking `message/send`, `tasks/get`, `tasks/list`, `tasks/cancel`. Persist registry in the session so a later parent turn can wait. Tools and slash commands: `/tasks`, `/task wait <id>`, `/task cancel <id>`.

**DoR (Definition of Ready):**
- Step 18 complete
- `internal/session` persistence reviewed

**DoD (Definition of Done):**
- [x] Non-blocking spawn returns a task id; parent ReAct loop continues
- [x] `/task wait <id>` blocks until completed/failed/canceled or ctx cancel
- [x] `/task cancel <id>` maps to `tasks/cancel` then SIGTERM
- [x] `/tasks` lists id, spec, state
- [x] Registry survives a new parent turn in the same session
- [x] Wait does not hold the ReAct lock in a way that deadlocks the semaphore (test)
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-020-agent-task-registry.md`; implementation in `internal/agent/tasks/registry.go`, `internal/commands/tasks.go`, `internal/session/metadata.go`, `internal/tools/agent_tasks.go`, `internal/agent/child/handle.go`, `internal/agent/subagent/manager.go`; closed at sequence [85].

**Risks:** Wait deadlocks admission. Mitigation: wait outside the spawn semaphore; documented in test.

**Files likely affected:** `internal/agent/tasks/registry.go`, `internal/commands/tasks.go`, `internal/session/`, `internal/tools/`

---

### Step 21: Unified task view (A2A + shell background)

**Description:** Present shell `start_process` tasks and A2A agent tasks in one list. Do not merge implementations — one view, two kinds.

**DoR (Definition of Ready):**
- Step 20 complete
- `internal/agent/executor` BackgroundTaskManager and process tools reviewed

**DoD (Definition of Done):**
- [x] `/tasks` shows `kind=agent|shell` for both families
- [x] Existing `list_processes` / `kill_process` still work for shell
- [x] Cancel on a shell row still SIGTERM/SIGKILL as today
- [x] Tests cover mixed lists
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-021-unified-task-view.md`; implementation in `internal/agent/tasks/view.go`, `internal/commands/tasks.go`, `internal/tools/list_processes.go`, `internal/tools/shell_source.go`; closed at sequence [89].

**Risks:** IDs collide (7-hex shell vs A2A ids). Mitigation: prefix or typed id; test mixed namespace.

**Files likely affected:** `internal/agent/tasks/`, `internal/tools/list_processes.go`, `internal/commands/tasks.go`

---

### Step 22: Structured navigation index

**Description:** Add `internal/nav` that returns index records (`kind=skill|plugin|session|peer|path|symbol`) instead of raw trees. Filesystem listings go through compact R10. Wire a `navigate` tool (or compose existing tools) that the catalog can point at.

**DoR (Definition of Ready):**
- Steps 2, 5, 10, and 20 available as sources
- Spec structured-navigation section reviewed

**DoD (Definition of Done):**
- [x] Index records have `id`, `title`, `why`, `open`
- [x] Path listings are tree-compressed, not one line per file
- [x] Sessions come from existing resume index
- [x] Skills/plugins/peers come from live catalogs
- [x] Unit tests for record shape and escape of file bodies
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-022-structured-navigation-index.md`; implementation in `internal/nav/index.go`, `internal/nav/paths.go`, `internal/tools/navigate.go`, `internal/conversation/tools.go`; closed at sequence [93].

**Risks:** `navigate` dumps file contents. Mitigation: test that `open` is a pointer, not a body.

**Files likely affected:** `internal/nav/index.go`, `internal/tools/navigate.go`, `internal/session/index.go`

---

### Step 23: TUI and ACP surfaces for skills, tasks, and agents

**Description:** Timeline blocks `SKILL`, `TASK`, `SUBAGENT`, `HOOK`, `COMPACT`. Status bar task count. Command palette entries. ACP notifications for the same task/skill events (discriminator, not a new transport). Welcome footer lists `/skills` `/tasks` `/agents`.

**DoR (Definition of Ready):**
- Steps 3, 13, 20–22 complete
- `internal/tui/mapper.go` and ACP event transformer reviewed

**DoD (Definition of Done):**
- [x] Mapper covers the new block types with tests
- [x] Status bar shows `tasks N/M` when the registry is non-empty
- [x] Palette lists Skills, Tasks, Agents
- [x] ACP session receives task state notifications (`kind=a2a` or equivalent)
- [x] Welcome/help text includes the new slash commands
- [x] Hook veto reason is visible (overlay or block), not only a log line
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md`; implementation in `internal/tui/mapper.go`, `internal/tui/mapper_harness.go`, `internal/ui/status/`, `internal/ui/overlay/`, `internal/protocol/acp/notifications.go`, `cmd/spin/tui.go`, `internal/commands/agents.go`; closed at sequence [97].

**Risks:** ACP clients ignore unknown notification shapes. Mitigation: wrap in existing thought/session update channels already used by the transformer.

**Files likely affected:** `internal/tui/mapper.go`, `internal/ui/status/`, `internal/ui/overlay/`, `internal/protocol/acp/event_transformer.go`, `cmd/spin/tui.go`

---

### Step 24: Remote A2A HTTPS client and card allowlist

**Description:** Same A2A client as local children, HTTPS JSON-RPC binding, Agent Card fetch. Only URLs on an explicit allowlist. No change to ACP.

**DoR (Definition of Ready):**
- Steps 16 and 18 complete
- `internal/auth` and config reviewed

**DoD (Definition of Done):**
- [x] Config `a2a.allowlist` (or equivalent) is required for remote cards
- [x] Off-allowlist URL is rejected before dial
- [x] `message/send` + `tasks/get` succeed against a fake HTTPS A2A server in tests
- [x] Local stdio children still work
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-024-remote-a2a-https-client-and-card-allowlist.md`; implementation in `internal/protocol/a2a/http.go`, `internal/config/config_v2.go`, `internal/agent/child/remote.go`; closed at sequence [101].

**Risks:** SSRF via card URL. Mitigation: allowlist only; no follow redirects off-list.

**Files likely affected:** `internal/protocol/a2a/http.go`, `internal/config/config_v2.go`, `internal/agent/child/`

---

### Step 25: Parent shutdown cancels children and ends the session

**Description:** On Ctrl-C / `/exit` / ACP cancel: `tasks/cancel` then SIGTERM to running children, run `SESSION_END`, keep the existing TUI screen clear. Reap orphans on next parent start via pid/socket files under the runtime dir.

**DoR (Definition of Ready):**
- Steps 8, 20, and 18 complete
- Existing TUI `stopTUILoop` / clear-on-exit reviewed

**DoD (Definition of Done):**
- [x] Parent quit sends cancel to every running A2A task
- [x] Children exit after stdin/socket close (test)
- [x] `SESSION_END` hooks run on that path
- [x] Next `spin` start reaps stale pid files
- [x] TUI still clears the screen (existing `ClearHome` teardown)
- [x] `make test` and `make lint` pass

**Traceability:** journey at `specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md`; implementation in `cmd/spin/tui.go`, `cmd/spin/exec.go`, `cmd/spin/acp.go`, `cmd/spin/a2a.go`, `cmd/spin/reap.go`, `internal/agent/tasks/registry.go`, `internal/agent/child/runtime.go`, `internal/agent/child/handle.go`, `internal/conversation/conversation.go`, `internal/protocol/acp/agent.go`, `internal/ui/adapters/puretty.go`; closed at sequence [104].

**Risks:** Cancel races with a completing child. Mitigation: ignore `tasks/cancel` on already-terminal tasks.

**Files likely affected:** `cmd/spin/tui.go`, `internal/agent/tasks/`, `internal/agent/child/`, `internal/conversation/conversation.go`

---

### Step 26: Operator documentation

**Description:** Diátaxis how-to and reference for skills, plugins, compact, spawn/wait, hooks, and the local A2A binding. Link from README. No new site generator.

**DoR (Definition of Ready):**
- Steps 1–25 code is on the branch this item documents (or the item documents only landed behavior and lists open steps honestly)

**DoD (Definition of Done):**
- [x] `docs/how-to/agent-skills.md` — write a skill, where to put it, `/skills`, `skill` tool
- [x] `docs/how-to/agent-plugins.md` — `plugin.json` layout, containment, MCP isolation
- [x] `docs/how-to/subagents.md` — spawn, wait, cancel, process model
- [x] `docs/reference/compact.md` — command table, R12–R15, escape hatch
- [x] `docs/reference/hooks.md` — ten events, filenames, `updated_input`
- [x] README points at the how-to
- [x] `make lint` pass (markdown as required by repo)

**Traceability:** journey at `specs/journeys/JOURNEY-026-operator-documentation.md`; implementation in `docs/how-to/agent-skills.md`, `docs/how-to/agent-plugins.md`, `docs/how-to/subagents.md`, `docs/reference/compact.md`, `docs/reference/hooks.md`, `README.md`; closed at sequence [108].

**Risks:** Docs describe unlanded flags. Mitigation: document only merged flags; link the spec for the rest.

**Files likely affected:** `docs/how-to/`, `docs/reference/`, `README.md`

---

## Coverage map

| Spec scope item | Steps |
|---|---|
| Skill runtime | 1–3 |
| Plugin runtime | 4–6 |
| Skill tool | 3 |
| Process subagents | 17–19 |
| A2A client/server | 16–18, 24 |
| Background agent tasks | 20–21 |
| Shell background (keep) | 21 |
| Hooks completeness | 6–8, 19, 25 |
| RTK compact R1–R15 | 9–13 |
| Task framing | 14 |
| Context assembly | 15 |
| Structured navigation | 22 |
| UX / visuals | 13, 23, 25 |
| Config / CLI | 5, 11, 17, 24 |
| Safety | 4, 7, 19, 24, 25 |
| Tests + docs | every step + 26 |

## Anti-goals (do not schedule)

- Replace ACP with A2A for IDE hosts
- In-process goroutine as the production subagent runtime
- Require the Rust `rtk` binary
- Recursive `skills/**` discovery
- Shared writable memory between parent and child
