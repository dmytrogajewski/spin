# JOURNEY-002-discover-skill-catalog: Discover the skill catalog and list it to the user

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Discover the skill catalog and list it to the user

## 1. Journey

When **an operator starts spin (TUI or `exec`) in a repo that already has Agent Skills on disk** I want to **see a metadata-only catalog of those skills — the same list the model sees** so I **know which portable instructions are in play without blowing the context window with skill bodies**.

## 2. CJM

Alex is a Go engineer whose repo already has promptkit-generated `.agents/skills` plus optional `.claude/skills` interop folders. User-level skills live under `~/.spin/skills` and `~/.agents/skills`. Bundled promptkit skills may appear under `$SPIN_HOME/skills`. Step 1 can parse one directory; nothing yet scans the roots, resolves name collisions, or shows the operator what the model will be told. Dumping full `SKILL.md` bodies into the system prompt would recreate the problem skills were invented to solve. This journey makes discovery, a metadata-only Composer section, and `/skills` the single catalog the operator and the model share.

### Phase 1: Scan skill roots

**User Intent:** Collect every valid skill from project, user, and bundled roots without failing the session when a root is absent.

**Actions:** Call `Discover` with a workdir (and injectable home / bundled dirs in tests). Walk immediate children of `<workDir>/.agents/skills/`, `<workDir>/.claude/skills/`, `~/.spin/skills/`, `~/.agents/skills/`, then bundled. Parse each child with Step 1 `Parse`. Skip missing roots and unparseable children.

**Pain / Risk:** A missing `~/.spin/skills` aborts the session; recursive walk picks up nested plugin paths (forbidden later and now); `ReadDir` order is filesystem-dependent; real `$HOME` leaks into tests; invalid sibling poisons the whole catalog.

**Success Signal:** Missing roots are ignored. An empty catalog is a valid result. Only immediate children that `Parse` accepts become entries.

### Phase 2: Resolve collisions and tag source

**User Intent:** One catalog row per skill name, with a `source` tag so the operator and model know who won.

**Actions:** Scan in priority order. First writer wins. Tag `project` (workdir `.agents` then `.claude`), `user` (`~/.spin` then `~/.agents`), `bundled`. Each entry stores `name`, `description`, `location`, `source`.

**Pain / Risk:** User copy silently shadows a project skill (wrong winner); two project roots disagree and the loser is untraceable; `source` omitted so `/skills` and the prompt cannot disambiguate; location points at the loser.

**Success Signal:** On a name collision the surviving entry has `source=project` (or the highest-priority writer) and that writer’s description and location.

### Phase 3: Inject a metadata-only Composer section

**User Intent:** The model sees the same names and descriptions the operator will list — and nothing else.

**Actions:** When the catalog is non-empty, add a Composer section that renders **only** name + description. TUI and `exec` both compose through the conversation builder (ACP uses the same section helper).

**Pain / Risk:** Section dumps `Skill.Body` and blows the window; a fixture `##` heading leaks; empty catalog still injects noise; `DefaultRegularSections` grows and breaks the RegularSystemPrompt golden; only TUI or only `exec` gets the section.

**Success Signal:** Composed text contains each name and description and does not contain a fixture `##` body heading. Empty catalog omits the section. Regular default sections still match `RegularSystemPrompt`.

### Phase 4: List the catalog to the operator (`/skills`, `/help`)

**User Intent:** Type `/skills` and see one line per skill: name, source, description. Learn the command from `/help`.

**Actions:** Register `/skills` in the slash-command registry. `Execute` discovers from `CommandContext.GetWorkDir()` using the same `Discover` as Composer. `/help` lists the command (registry) and documents it in the examples.

**Pain / Risk:** `/skills` formats a different set than Composer (drift); help never mentions `/skills`; ACP hides it as TUI-only by mistake; empty catalog returns an error instead of a valid empty/no-skills result.

**Success Signal:** `/skills` prints one line per skill with name, source, and description. `/help` includes `/skills`. Empty catalog is a successful, non-error response.

### Phase 5: Prove hermetic catalog tests

**User Intent:** Collision, missing roots, and empty catalog stay true on every machine.

**Actions:** Unit tests write fixtures under `t.TempDir` (never the real home). One test plants the same name in project and user roots and asserts the project `source` tag. One test points at missing roots. One test uses existing empty dirs. One Composer test plants a `##` body heading and forbids it in the section.

**Pain / Risk:** Tests call `OptionsFor` and pick up the developer’s `~/.agents/skills`; collision test is order-flaky; body-leak test asserts a heading the section template itself uses.

**Success Signal:** Required unit tests pass without reading the real home. Body-leak assertion uses a unique fixture heading.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Skills on disk but invisible | 1 | One `Discover` walk both surfaces call |
| Missing user/bundled dirs look like hard errors | 1 | Ignore missing roots; empty catalog is valid |
| Same name in project and user | 2 | Priority order + `source` tag; project wins |
| Full bodies in the system prompt | 3 | Metadata-only Composer section; body-leak test |
| Operator cannot see what the model sees | 4 | `/skills` prints the same catalog |

### North Star Summary

Alex starts spin in a repo with `.agents/skills`. The model’s system prompt lists skill names and one-line descriptions. `/skills` prints the same names with `source` tags. A user-level duplicate of a project skill does not win. Missing `~/.spin/skills` does not fail the session. No skill body — including a `##` heading inside `SKILL.md` — enters the catalog section.

### Stressors

1. `<workDir>/.agents/skills` is absent while `.claude/skills` exists (interop-only repo).
2. `~/.spin/skills` and `~/.agents/skills` do not exist on a fresh machine.
3. The same `name` exists in project `.agents` and user `~/.spin` (project must win; `source=project`).
4. The same `name` exists in workdir `.agents` and workdir `.claude` (`.agents` wins).
5. A child directory has no `SKILL.md` or fails `Validate` (skip; do not fail the catalog).
6. A skill body contains `##` headings that must not appear in the Composer section.
7. `Discover` is called with empty `WorkDir`, `HomeDir`, and `BundledDir` (empty catalog is valid).
8. `ReadDir` order differs across filesystems (catalog listing must still be deterministic).
9. Tests accidentally scan the real `$HOME` and pick up the developer’s skills.
10. Only TUI or only `exec` is wired, so one surface’s model never sees the catalog.
11. `/help` lists registered commands but never names `/skills` in examples, so operators miss it.
12. Bundled `$SPIN_HOME/skills` collides with a project skill (project still wins).
13. Nested `skills/foo/nested/SKILL.md` (must not be discovered; immediate children only).
14. A non-directory file sitting in a skills root (must be ignored).
15. Empty catalog injected as a noisy “Skills” heading that wastes tokens.

## 3. UX Implementation and Assessment

The “user” in this journey is both the operator (`/skills`, `/help`) and the model (Composer catalog).

### Time to First Value
- [ ] Starting spin in a repo with `.agents/skills` puts names in the system prompt with no extra flag
- [ ] `/skills` lists those names on the first invocation

### Onboarding Clarity
- [ ] `/help` documents `/skills`
- [ ] Each catalog line includes `source` so origin is obvious

### Production-Ready Defaults
- [ ] Missing roots are ignored; no config is required
- [ ] Empty catalog is valid and does not fail the session

### Golden Path Quality
- [ ] Project `.agents/skills` appear in both Composer and `/skills`
- [ ] Name + description only — no bodies

### Decision Load
- [ ] Operators do not choose a search path
- [ ] Collision policy is fixed (project, then user, then bundled)

### Progressive Complexity
- [ ] Zero skills: session works as before
- [ ] User and bundled roots are additive, not required

### Error Quality
- [ ] Unparseable children are skipped, not surfaced as a session-killing error
- [ ] `/skills` on an empty catalog is a successful empty/no-skills result

### Failure Safety
- [ ] Discovery does not write to the filesystem
- [ ] Invalid skills cannot replace a valid earlier winner

### Runtime Transparency
- [ ] `source` is on every entry
- [ ] `location` records the winning skill directory

### Debuggability
- [ ] `/skills` and Composer share `Discover`
- [ ] Collision tests name the winning `source`

### Cross-Surface Consistency
- [ ] TUI and `exec` both compose the catalog when the workdir has `.agents/skills`
- [ ] `/skills` is not marked TUI-only

### Workflow Consistency
- [ ] Discovery lives in `internal/skills`; the section lives in `internal/agent/prompt`; the command lives in `internal/commands`
- [ ] Tests follow table-driven + `t.TempDir` conventions in docs/testing.md

### Change Safety
- [ ] `DefaultRegularSections` still matches `RegularSystemPrompt` (catalog is an extra section)
- [ ] Existing slash commands keep their names and help text

### Experimentation Safety
- [ ] Tests inject `HomeDir` and `BundledDir` (no real-home dependency)
- [ ] Fixtures are created under `t.TempDir`

### Interaction Latency
- [ ] Catalog is metadata-only (no body I/O beyond `Parse` of `SKILL.md`)
- [ ] Immediate children only — no recursive walk

### Developer Feedback Speed
- [ ] `go test ./internal/skills ./internal/agent/prompt ./internal/commands` isolates failures
- [ ] Body-leak test fails with the leaked heading in the message

### Team Scale
- [ ] Project skills in git win over each developer’s user skills
- [ ] Source tags make reviews of `/skills` output comparable

### System Scale
- [ ] Plugin roots are not scanned (Step 5)
- [ ] Skill tool / body activation is not implemented (Step 3)

### Right Behavior by Default
- [ ] Project wins on name collision
- [ ] Composer omits bodies even when `Parse` loaded them

### Anti-Bypass Design
- [ ] Tests forbid a fixture `##` heading in the composed section
- [ ] `/skills` cannot be registered without a description (help lists it)

## 4. Tests

### TC-01: empty_catalog_is_valid

**Given** existing but empty skill roots (or all roots omitted).
**When** `Discover` runs.
**Then** the result is an empty catalog and not a failure.

### TC-02: missing_roots_ignored

**Given** `WorkDir`, `HomeDir`, and `BundledDir` that do not exist.
**When** `Discover` runs.
**Then** those roots are skipped and the catalog is empty.

### TC-03: project_agents_discovered

**Given** a workdir with `.agents/skills/<name>/SKILL.md` that parses.
**When** `Discover` runs with that workdir and empty home/bundled.
**Then** the catalog has one entry with `source=project`, matching name, description, and location.

### TC-04: collision_project_wins_source_tag

**Given** the same skill name in `<workDir>/.agents/skills` and `<home>/.spin/skills` with different descriptions.
**When** `Discover` runs.
**Then** exactly one entry exists, `source` is `project`, and description/location belong to the project copy.

### TC-05: user_and_bundled_sources

**Given** distinct names in `~/.spin/skills`, `~/.agents/skills`, and bundled.
**When** `Discover` runs.
**Then** each entry’s `source` is `user` or `bundled` as appropriate.

### TC-06: composer_section_metadata_only

**Given** a parsed catalog entry whose `SKILL.md` body contains `## MustNotAppearInCatalog`.
**When** the Composer skill-catalog section is composed.
**Then** the section contains the name and description and does not contain `## MustNotAppearInCatalog`.

### TC-07: empty_catalog_omits_section

**Given** an empty catalog.
**When** `ApplyCatalog` is called on a Composer.
**Then** composed output has no Skills heading.

### TC-08: skills_command_one_line_per_entry

**Given** a workdir with one project skill.
**When** `/skills` executes with that workdir.
**Then** output has one line containing name, `project`, and description.

### TC-09: help_documents_skills

**Given** the default command registry.
**When** `/help` executes.
**Then** the text includes `/skills`.

### TC-10: tui_and_exec_share_composer_catalog

**Given** a workdir with `.agents/skills`.
**When** the shared compose path used by TUI and `exec` (conversation builder) builds the system prompt.
**Then** the prompt contains the skill name and description and not the fixture body heading.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 2:

- Discovery order: `<workDir>/.agents/skills/`, `<workDir>/.claude/skills/`, `~/.spin/skills/`, `~/.agents/skills/`, bundled — project wins on name collision; catalog entries include `source`
- Composer section contains **only** name + description (no skill bodies)
- `/skills` prints one line per skill (name, source, description) and is documented in `/help`
- TUI and `exec` both see the catalog when a workdir has `.agents/skills`
- Unit tests: collision source tag, missing roots ignored, empty catalog is valid
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 2
- Implementation files: `internal/skills/catalog.go`, `internal/skills/discover.go`, `internal/agent/prompt/catalog.go`, `internal/commands/skills.go`, `internal/conversation/builder.go`, `cmd/spin/acp.go`
- Test files: `internal/skills/discover_test.go`, `internal/agent/prompt/catalog_test.go`, `internal/commands/commands_test.go`, `internal/conversation/builder_test.go`

## Implementation

Files created:
- `internal/skills/catalog.go` — `Entry`, `Source`, `Catalog`, `Options`, `Format`
- `internal/skills/discover.go` — `Discover`, `OptionsFor` (project → user → bundled; first name wins)
- `internal/agent/prompt/catalog.go` — `SkillCatalogSection`, `ApplyCatalog` (name + description only)
- `internal/commands/skills.go` — `/skills` slash command
- `internal/skills/discover_test.go` — empty catalog, missing roots, collision `source`, skip invalid/nested
- `internal/agent/prompt/catalog_test.go` — metadata-only section; fixture `##` heading must not leak

Files modified:
- `internal/commands/commands.go` — register `/skills`; document it in `/help` examples
- `internal/commands/commands_test.go` — `/skills` list/help/empty tests
- `internal/conversation/builder.go` — TUI and `exec` compose the catalog via `composeSystemPrompt`
- `internal/conversation/builder_test.go` — workdir `.agents/skills` appears in the composed prompt
- `cmd/spin/acp.go` — same catalog injection on the ACP compose path
- `internal/protocol/acp/command_integration_test.go` — `/skills` advertised
- `docs/testing.md` — journey 002 row
- `specs/agent-harness/ROADMAP.md` — Step 2 DoD ticks and traceability
