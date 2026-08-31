# JOURNEY-022-structured-navigation-index: Structured navigation index

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Structured navigation index

## 1. Journey

When **the model (or operator) needs to find a skill, plugin, session, peer, path, or symbol** I want to **receive index records (`kind`, `id`, `title`, `why`, `open`) instead of raw trees or file bodies** so I **can open a pointer and keep the context window small**.

## 2. CJM

The agent already has live catalogs (skills, plugins), a resume session index, A2A / subagent peers, and R10 tree compression for directory listings. Those sources still surface as separate dumps: catalog lines, one-line-per-file `ls`, session metadata, or full file reads. This journey adds `internal/nav` as a single index of records and a `navigate` tool the catalog can point at. It does **not** add TUI panes, ACP notifications, or slash-command surfaces (Step 23).

Assumption: `open` is always a pointer (filesystem path or A2A card URL), never a file body. Assumption: path listings go through compact R10 (`ls`/`tree`) so the model sees hierarchy + per-dir counts, not one line per file. Assumption: when no remote A2A peers exist yet, peers come from live `subagent.Builtins()` (and/or Agent Cards built from those specs). Assumption: sessions come from the existing `session.Index.List` resume index, not a new store. Assumption: skills and plugins come from the live Discover catalogs already used by `/skills` and Composer.

### Phase 1: Ask the index, not the tree

**User Intent:** Discover what exists (skills, plugins, sessions, peers) as compact records.

**Actions:** The model calls `navigate` with `kind=skill|plugin|session|peer`. The tool returns JSON records with `id`, `title`, `why` (one line), and `open` (path or card URL). Empty catalogs return an empty record list.

**Pain / Risk:** The tool dumps SKILL.md bodies or plugin.json contents into `open` or `why`. Records omit `id`/`title`/`why`/`open`. Skills or plugins are hardcoded fixtures instead of live Discover. Sessions are re-scanned from transcript files instead of `session.Index`. Empty catalogs error instead of returning zero records.

**Success Signal:** Each record has the four fields. Skill `open` is the skill directory. Plugin `open` is the plugin root. Session `open` is a path (workdir or session pointer), not transcript text. Peer `open` is a card URL. Distinctive file-body strings do not appear in any record field.

### Phase 2: List a path without exploding the window

**User Intent:** See a directory as a compressed tree, then open a single path pointer.

**Actions:** The model calls `navigate` with `kind=path` and a directory. The listing is R10-compressed (hierarchy + counts). One path record points at the directory via `open`. File contents are not inlined.

**Pain / Risk:** Path kind emits one record per file. Compact is skipped and the listing is `./file` per line. `why` or `listing` includes file bodies. Aggressive recursion walks the whole repo. Escape hatch / disabled compact is not this step’s job, but a missing R10 apply would violate the DoD.

**Success Signal:** Path output contains an R10 tree (root count line, directory counts) and is not a raw one-line-per-file listing. The path record’s `open` is the directory path.

### Phase 3: Point at a symbol without shipping its body

**User Intent:** Resolve a symbol to a file pointer the model can `read_file` later.

**Actions:** The model calls `navigate` with `kind=symbol` and a name. Records use `open` as a path (or path:line). A missing symbol source returns zero records.

**Pain / Risk:** Symbol records embed source text. The tool calls `read_file` internally. LSP failure panics instead of an empty/error result.

**Success Signal:** Symbol records have `open` as a pointer. Tests inject a body-bearing fake source and assert the body never appears in the index output.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Catalog dumps and raw `ls` blow the window | 1, 2 | Index records + R10 trees |
| `open` accidentally carries a body | 1 | Pointer-only contract + escape tests |
| Sessions hidden unless transcripts are parsed | 1 | Reuse `session.Index` |
| No remote peers yet | 1 | Builtin specs / local cards as peers |
| Symbol lookup inlines source | 3 | Pointer to path, then `read_file` |
| TUI/ACP scope creep | 1–3 | Stay on the tool + `internal/nav` |

### North Star Summary

The model asks `navigate` and gets a short list of index records. Every row says what it is (`kind`), who it is (`id`/`title`), why it exists (`why`, one line), and where to go (`open`). Filesystem listings are R10 trees. File bodies stay on disk until a later read.

### Stressors

1. `navigate` dumps SKILL.md or plugin.json contents into `open` or the tool output.
2. Path listings stay one line per file because R10 is not applied.
3. Sessions are rebuilt from transcript JSONL instead of the resume index.
4. Skills/plugins are a static table, so live catalog changes never appear.
5. Peers require a remote HTTPS card and the index is empty in local-only mode.
6. `why` contains newlines or a whole Markdown body.
7. Unknown `kind` panics or returns a raw tree instead of a typed error.
8. A distinctive secret in a file body appears in any record field.
9. `NewDefaultRegistry` never registers `navigate`, so the catalog cannot point at it.
10. Builder never binds the live session index, so `kind=session` is always empty in production.
11. Symbol `open` is the function source instead of `path` or `path:line`.
12. Path listing recursively walks ignored trees (`.git`, `vendor`) and still one-lines every file.
13. Record JSON omits `id` or `title` under `omitempty` when those fields are required.
14. Import cycle (`nav` → `tools` → `nav`) blocks registration.

## 3. UX Implementation and Assessment

This step is agent-facing (tool + index). Operator TUI/ACP surfaces are Step 23.

### Time to First Value
- [x] `navigate` is registered on the default tool registry so the model can call it without extra config
- [x] Empty catalogs return zero records immediately (no setup required)

### Onboarding Clarity
- [x] Tool description names the six kinds
- [x] Unknown kind lists the valid kinds

### Production-Ready Defaults
- [x] Live Discover + builtin peers work with no extra config
- [x] Missing session index is a valid empty source

### Golden Path Quality
- [x] Each kind returns records with `id`, `title`, `why`, `open`
- [x] Path listings are R10-compressed

### Decision Load
- [x] One tool, one `kind` parameter
- [x] Optional `id` / `path` / `name` only when filtering

### Progressive Complexity
- [x] `kind` alone lists a catalog
- [x] `path` and `name` are opt-in

### Error Quality
- [x] Unknown kind names the problem and lists valid kinds
- [x] Missing directory on `kind=path` is a tool error, not a panic

### Failure Safety
- [x] Nil session/symbol sources yield empty lists
- [x] No write or delete operations on this tool

### Runtime Transparency
- [x] Output is JSON records the model can read
- [x] No hidden filesystem mutation

### Debuggability
- [x] `open` traces back to a path or card URL
- [x] Path listing is the same R10 shape as `list_directory`

### Cross-Surface Consistency
- [x] Skill/plugin rows use the same Discover catalogs as `/skills`
- [x] Kind vocabulary matches the spec (`skill|plugin|session|peer|path|symbol`)

### Workflow Consistency
- [x] Tool registration follows `RegisterSkillTools` / `RegisterAgentTaskTools`
- [x] Tests use `// Journey:` comments

### Change Safety
- [x] Index is read-only over existing stores
- [x] Session index schema is unchanged

### Experimentation Safety
- [x] Tests inject Sources; no live home catalog required
- [x] Distinctive body fixtures prove escape without touching production files

### Interaction Latency
- [x] Catalog list does not read skill bodies
- [x] Path list is one directory + R10, not a full-repo walk

### Developer Feedback Speed
- [x] Typed errors on bad kind / missing path
- [x] Filter by `id` without restarting discovery

### Team Scale
- [x] Catalogs come from the same workdir/plugin roots as the rest of the harness
- [x] Record JSON is stable for fixtures

### System Scale
- [x] New kinds would be another `Kind` + mapper, not a new tool
- [x] Path compression reuses compact R10

### Right Behavior by Default
- [x] `open` is a pointer
- [x] Peers default to builtin specs when remotes are absent

### Anti-Bypass Design
- [x] Tests fail if a file body appears in records
- [x] No API to attach a body to `open`

## 4. Tests

### TC-01: record_shape

**Given** a skill catalog entry.
**When** the index lists `kind=skill`.
**Then** each record has non-empty `id`, `title`, `why`, and `open`.

### TC-02: open_is_pointer_not_body

**Given** a skill whose SKILL.md body contains a distinctive secret string.
**When** the index lists that skill.
**Then** no record field contains the secret; `open` is the skill directory.

### TC-03: plugin_from_live_catalog

**Given** a loaded plugin from Discover.
**When** the index lists `kind=plugin`.
**Then** the record `id` is the plugin name and `open` is the plugin root, not `plugin.json` contents.

### TC-04: session_from_resume_index

**Given** a `session.Index` with one resume entry.
**When** the index lists `kind=session`.
**Then** the record `id` matches the session id and `open` is the workdir path (not transcript text).

### TC-05: peers_from_builtins

**Given** no remote A2A peers.
**When** the index lists `kind=peer`.
**Then** records include builtin specs (`explorer`, `planner`, `reviewer`, `ask_user`) with `open` as a card URL.

### TC-06: path_r10_tree

**Given** a temp directory with files and a subdirectory.
**When** the index lists that path.
**Then** the listing matches compact R10 on the raw `./` listing and is not one line per file.

### TC-07: path_record_pointer

**Given** the same temp directory with a file whose body is a secret.
**When** the index lists that path.
**Then** the path record `open` is the directory and neither the record nor the listing contains the secret.

### TC-08: symbol_pointer

**Given** a symbol source that would be able to return a body.
**When** the index lists `kind=symbol`.
**Then** `open` is a path pointer and the body is absent.

### TC-09: why_one_line

**Given** a description that contains newlines.
**When** a record is built.
**Then** `why` is a single line.

### TC-10: unknown_kind

**Given** `navigate` with `kind=nope`.
**When** the tool executes.
**Then** the result is an error that lists valid kinds.

### TC-11: navigate_registered

**Given** `NewDefaultRegistry`.
**When** the registry is listed.
**Then** `navigate` is present.

### TC-12: discover_live_skills

**Given** a temp workdir with `.agents/skills/<name>/SKILL.md`.
**When** `Discover` builds an index.
**Then** `kind=skill` includes that name from the live catalog.

## Acceptance Criteria

- Index records have `id`, `title`, `why`, `open`
- Path listings are tree-compressed, not one line per file
- Sessions come from existing resume index
- Skills/plugins/peers come from live catalogs
- Unit tests for record shape and escape of file bodies
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 22
- Implementation files: `internal/nav/kind.go`, `internal/nav/sources.go`, `internal/nav/index.go`, `internal/nav/paths.go`, `internal/nav/discover.go`, `internal/tools/navigate.go`, `internal/tools/registry.go`, `internal/conversation/tools.go`, `internal/conversation/builder.go`
- Test files: `internal/nav/index_test.go`, `internal/tools/navigate_test.go`, `internal/conversation/navigate_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-022-structured-navigation-index.md` — this journey
- `internal/nav/kind.go` — kinds `skill|plugin|session|peer|path|symbol` and `Record`
- `internal/nav/sources.go` — `Sources`, `Query`, `Result`, `PluginRow`, session/symbol/peer surfaces
- `internal/nav/index.go` — `Records` / `Lookup` from live injected catalogs
- `internal/nav/paths.go` — one directory listing through compact R10
- `internal/nav/discover.go` — skill Discover + `Live` (plugins injected to avoid `nav`→`plugins`→`mcp`→`tools` cycle)
- `internal/nav/index_test.go` — record shape, body escape, R10 paths, resume index, builtin peers
- `internal/tools/navigate.go` — `navigate` tool + `RegisterNavigateTool`
- `internal/tools/navigate_test.go` — JSON records, unknown kind, path listing
- `internal/conversation/navigate_test.go` — live plugin catalog via Builder

Files modified:
- `internal/tools/registry.go` — `NewDefaultRegistry` registers `navigate`
- `internal/tools/registry_test.go` — expected tool list includes `navigate`
- `internal/tools/classification.go` — `navigate` is CategorySearch
- `internal/conversation/tools.go` — `registerNavigate` maps live plugins + session index
- `internal/conversation/builder.go` — bind resume index into `navigate` at assemble
- `docs/testing.md` — journey 022 test files
- `specs/agent-harness/ROADMAP.md` — Step 22 DoD and traceability

