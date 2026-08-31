# JOURNEY-003-activate-skill-body: Activate a skill body with progressive disclosure

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Activate a skill body with progressive disclosure

## 1. Journey

When **the model matches a user ask to a catalog skill** I want to **load that skill’s body into the current turn — and only the files I later request from its root** so I **follow portable instructions without dumping every `references/` file into the context window**.

## 2. CJM

Alex’s repo already has a metadata-only skill catalog (Step 2). The model sees names and descriptions. Nothing yet injects a `SKILL.md` body, records experimental `allowed-tools`, or contains relative reads to the skill root. Loading every linked `references/*.md` on activation would recreate the problem skills were invented to solve. This journey adds a `skill` / `load_skill` tool, one-hop-only resource reads, a typed unknown-name error, and a timeline `SKILL` block with name + source.

### Phase 1: Choose a skill from the catalog

**User Intent:** The model picks a catalog name that matches the ask and calls `skill` (or `load_skill`).

**Actions:** Tool takes a skill name. Catalog lookup finds the winning entry from Step 2. `Parse` loads that directory’s `SKILL.md` body and root path.

**Pain / Risk:** Unknown name is a vague string with no catalog list; error text leaks another skill’s body; empty name is accepted; `load_skill` is missing so prompts that use the LangChain name fail.

**Success Signal:** A known name returns body + skill root. An unknown name is a typed error that lists catalog names and does not contain any skill body.

### Phase 2: Inject the body without auto-loading references

**User Intent:** Activation adds this turn’s instructions. Linked `scripts/`, `references/`, and `assets/` stay unread until the model asks.

**Actions:** Activation reads `SKILL.md` only. Body may mention `references/foo.md`. Those files are not opened. `allowed-tools` is copied onto the activation when present (enforcement is a later step).

**Pain / Risk:** Activation walks the skill tree and loads every markdown file; a nested `references/a.md` → `references/b.md` chain is followed automatically; `allowed-tools` is dropped because it is experimental.

**Success Signal:** Activation output contains the body and root. A unique sentinel in `references/*.md` is absent. `allowed-tools` is present on the activation when the frontmatter has it.

### Phase 3: Read one hop from the skill root

**User Intent:** The model requests a relative file under the skill (`references/`, `scripts/`, `assets/`). That one file is returned. Nothing further is followed.

**Actions:** Optional `path` on the same tool (or a follow-up read) resolves against the skill root. Reject `../` and any path that escapes the root. Read exactly that file.

**Pain / Risk:** `../` escapes to the workdir or home; `references/../secret` is cleaned into a valid in-root path and silently allowed; reading `foo.md` also opens files it links to.

**Success Signal:** In-root relative paths return that file’s bytes. `../` is rejected. A file that links to a second reference does not cause that second file to be loaded.

### Phase 4: Show a SKILL timeline block

**User Intent:** The operator sees that a skill activated, with name and source, not a generic TOOL badge.

**Actions:** Mapper maps `skill` / `load_skill` to `BlockTypeSkill`. Meta holds name + source. Renderer labels the block `SKILL`.

**Pain / Risk:** Skill calls render as TOOL/NOTICE with no source; source is missing because the start event only has the name; complete event never updates meta.

**Success Signal:** After start + complete, the timeline block type is `SKILL` and meta/title include name and source.

### Phase 5: Prove the catalog → body → on-demand reference path

**User Intent:** One integration test walks the real operator path: catalog in the prompt, `skill` injects the body, a reference stays unread until requested.

**Actions:** Plant a project skill with a unique body heading and a unique reference sentinel. Compose the prompt. Execute `skill`. Then read the reference path.

**Pain / Risk:** Test only unit-tests `Parse` and never the tool; prompt already contains the body (Step 2 regression); reference sentinel appears in the activation output.

**Success Signal:** Composed prompt has name + description and not the body heading. Skill result has the body and not the reference sentinel. Follow-up path read has the sentinel.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Catalog visible but body never loads | 1 | `skill` / `load_skill` by name |
| Unknown name is a dead end | 1 | Typed error lists catalog names |
| Activation dumps every reference | 2 | `SKILL.md` only; resources on demand |
| `../` escapes the skill package | 3 | Containment reject before read |
| Operator cannot see what activated | 4 | Timeline `SKILL` block with name + source |

### North Star Summary

The model sees the same catalog as `/skills`. Calling `skill` for a matching name injects that skill’s Markdown body and root path into the turn. `allowed-tools` is recorded when present. Linked `references/` files stay on disk until a contained one-hop read. The timeline shows a `SKILL` block with name and source. An unknown name lists what *is* available and leaks nothing else.

### Stressors

1. Unknown skill name — typed error lists catalog names and must not include any `SKILL.md` body.
2. Activation of a skill whose body links to `references/extra.md` — extra file must not be read.
3. Follow-up read of `references/extra.md` that itself links to `references/nested.md` — nested file must not be auto-loaded.
4. Relative path `../secrets.md` or `references/../../../etc/passwd` — rejected; no bytes from outside the skill root.
5. Path `references/../SKILL.md` contains `../` — rejected even if cleaned it would stay inside the root.
6. Absolute path `/etc/passwd` passed as the skill-relative path — rejected.
7. Empty skill name — rejected before catalog lookup.
8. `allowed-tools` present on the fixture — must appear on the activation metadata; must not be dropped.
9. `allowed-tools` absent — activation still succeeds; field omitted rather than invented.
10. `load_skill` alias — same activation behavior as `skill`.
11. Mapper start event has name only — complete event must still attach `source` from result metadata.
12. Catalog in the composed prompt still metadata-only after the tool exists (no body leak via Composer).
13. Two catalog entries — unknown-name error lists both names, neither body.
14. Missing reference file after a valid contained path — typed/not-found error, not a panic.
15. Skill tool registered on the conversation builder path so TUI and `exec` can both call it.

## 3. UX Implementation and Assessment

### Time to First Value
- [ ] Catalog name → `skill` call returns body in one tool round-trip
- [ ] No extra configuration required to activate a project skill

### Onboarding Clarity
- [ ] Tool description says to load a catalog skill by name
- [ ] Unknown-name error lists available names

### Production-Ready Defaults
- [ ] Activation reads `SKILL.md` only
- [ ] `allowed-tools` recorded when present; not enforced in this step

### Golden Path Quality
- [ ] Known name returns body + root
- [ ] Follow-up path read returns the requested file only

### Decision Load
- [ ] One required parameter: `name`
- [ ] Optional `path` only when a resource is needed

### Progressive Complexity
- [ ] Simple activate stays one call
- [ ] Resource reads are opt-in

### Error Quality
- [ ] Unknown name names the problem and lists catalog names
- [ ] Escape paths name containment as the reason

### Failure Safety
- [ ] Failed activate does not inject a body
- [ ] Escape attempts do not read outside the skill root

### Runtime Transparency
- [ ] Timeline `SKILL` block shows name + source
- [ ] Tool output includes root path

### Debuggability
- [ ] Activation output traces name → source → root → body
- [ ] Path reads echo the resolved relative path

### Cross-Surface Consistency
- [ ] `skill` and `load_skill` share one implementation
- [ ] TUI mapper and tool result use the same name/source fields

### Workflow Consistency
- [ ] Catalog still comes from Step 2 `Discover`
- [ ] Body still comes from Step 1 `Parse`

### Change Safety
- [ ] Activation does not write skill files
- [ ] Containment is reject, not rewrite-and-allow

### Experimentation Safety
- [ ] `allowed-tools` is advisory in this step (recorded only)
- [ ] Enforcement deferred; field is not dropped

### Interaction Latency
- [ ] Activate is one `Parse` of `SKILL.md`
- [ ] No directory walk of `references/`

### Developer Feedback Speed
- [ ] Tool result is immediate (success or typed error)
- [ ] Operator sees the SKILL block on the same turn

### Team Scale
- [ ] Project skills activate from `.agents/skills` without extra config
- [ ] Source tag still distinguishes project / user / bundled

### System Scale
- [ ] Catalog snapshot at registry build; no per-file preload
- [ ] One-hop reads stay O(1) files

### Right Behavior by Default
- [ ] Progressive disclosure is the default, not an opt-in
- [ ] `../` is rejected by default

### Anti-Bypass Design
- [ ] Path containment cannot be skipped via `filepath.Clean` of `../`
- [ ] Unknown-name path cannot be used to `Parse` an off-catalog directory

## 4. Tests

### TC-01: activate_returns_body_and_root

**Given** a catalog entry for `alpha` with a known body.
**When** `Activate` is called with `alpha`.
**Then** the activation has that body and the skill directory as root.

### TC-02: unknown_name_typed_error_lists_catalog

**Given** a catalog with `alpha` and `beta`.
**When** `Activate` is called with `missing`.
**Then** the error unwraps to `ErrUnknownSkill`, lists `alpha` and `beta`, and does not contain either skill body.

### TC-03: activate_does_not_read_references

**Given** a skill body that links to `references/extra.md` whose contents are a unique sentinel.
**When** the skill is activated.
**Then** the activation output contains the body and does not contain the sentinel.

### TC-04: path_read_one_hop_only

**Given** `references/extra.md` that links to `references/nested.md`.
**When** the skill tool is called with `path=references/extra.md`.
**Then** the result contains extra’s sentinel and not nested’s sentinel.

### TC-05: reject_dotdot_escape

**Given** an activated skill root.
**When** `Resolve` is called with `../secrets.md` or `references/../../../outside`.
**Then** the error unwraps to a path-escape sentinel and no file is read.

### TC-06: reject_dotdot_even_if_clean_stays_inside

**Given** a skill root.
**When** `Resolve` is called with `references/../SKILL.md`.
**Then** the path is rejected because it contains `../`.

### TC-07: allowed_tools_recorded

**Given** a skill whose frontmatter has `allowed-tools: read_file grep`.
**When** it is activated.
**Then** `Activation.AllowedTools` is `read_file grep`.

### TC-08: skill_and_load_skill_alias

**Given** a catalog with one skill.
**When** both `skill` and `load_skill` execute with that name.
**Then** both succeed and return the same body and root.

### TC-09: mapper_skill_block_name_and_source

**Given** a `skill` tool start with `name=alpha` and a complete event whose metadata has `source=project`.
**When** the mapper handles both events.
**Then** the UI block type is `SKILL` and meta/title include `alpha` and `project`.

### TC-10: integration_catalog_prompt_then_body_then_reference

**Given** a workdir with `.agents/skills/alpha` (body heading + `references/extra.md` sentinel).
**When** the composer builds the prompt and the `skill` tool activates `alpha`, then reads `references/extra.md`.
**Then** the prompt has name + description and not the body heading; activation has the body and not the sentinel; the path read has the sentinel.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 3:

- `skill` / `load_skill` tool takes a skill name, returns body + skill root path
- Unknown name returns a typed error listing catalog names (no body leak)
- Relative file reads from the skill stay inside the skill root (reject `../`)
- Nested reference chains beyond one hop from `SKILL.md` are not auto-loaded
- `allowed-tools` when present is recorded on the activation (enforced in a later step if experimental; field is not dropped)
- Mapper renders a `SKILL` block with name + source
- Integration test: catalog in prompt, `skill` call injects body, reference file unread until requested
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 3
- Implementation files: `internal/skills/activate.go`, `internal/skills/resolve.go`, `internal/tools/skill.go`, `internal/tools/registry.go`, `internal/tools/classification.go`, `internal/conversation/tools.go`, `internal/tui/mapper.go`, `internal/ui/blocks/`
- Test files: `internal/skills/activate_test.go`, `internal/skills/resolve_test.go`, `internal/tools/skill_test.go`, `internal/tools/skill_integration_test.go`, `internal/tui/mapper_skill_test.go`, `internal/conversation/builder_test.go`

## Implementation

Files created:
- `internal/skills/activate.go` — `Activation`, `Activate`, `UnknownSkillError`
- `internal/skills/resolve.go` — `Resolve` (reject `../`), `ReadResource` (one file, no follow)
- `internal/skills/activate_test.go` — body+root, unknown-name list, no reference leak, `allowed-tools`
- `internal/skills/resolve_test.go` — in-root accept, `../` reject including clean-stays-inside, one-hop read
- `internal/tools/skill.go` — `skill` / `load_skill` tool, `RegisterSkillTools`
- `internal/tools/skill_test.go` — activate, alias, unknown error, path read, escape
- `internal/tools/skill_integration_test.go` — Discover + activate + on-demand reference
- `internal/tui/mapper_skill_test.go` — SKILL block name + source

Files modified:
- `internal/skills/skill.go` — `ErrUnknownSkill`, `ErrPathEscape`, `ErrEmptyName`, `ErrEmptyPath`
- `internal/tools/registry.go` — register skill tools on `NewDefaultRegistry`
- `internal/tools/registry_test.go` — default registry includes `skill` and `load_skill`
- `internal/tools/classification.go` — `skill` / `load_skill` as notice
- `internal/conversation/tools.go` — register skill tools from the builder workdir catalog
- `internal/conversation/builder_test.go` — catalog in prompt, body on `skill`, reference unread until path
- `internal/tui/mapper.go` — `SKILL` block; complete event fills source
- `internal/ui/blocks/block.go` — `BlockTypeSkill`
- `internal/ui/blocks/metadata.go` — `SkillMeta`
- `internal/ui/blocks/model.go` — Get/Set SkillMeta
- `internal/ui/blocks/renderer.go` / `tokens.go` / tests — SKILL rendering
- `docs/testing.md` — journey 003 row
- `specs/agent-harness/ROADMAP.md` — Step 3 DoD ticks and traceability
