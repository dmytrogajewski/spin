# JOURNEY-001-parse-and-validate-agent-skills: Parse and validate Agent Skills

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Parse and validate Agent Skills

## 1. Journey

When **an operator (or later the catalog loader) points spin at a skill directory** I want to **parse `SKILL.md` into a validated record** so I **know the skill is Agent Skills–compliant before it enters the catalog**.

## 2. CJM

Alex is a Go engineer whose repo already has promptkit-generated `.agents/skills/*/SKILL.md` plus third-party skill folders. Today spin has **zero Go loader**: skills exist on disk but nothing proves they parse. Invalid names, a body mixed with frontmatter, or a silent skip would leak into Step 2’s catalog. This journey makes `internal/skills.Parse` / `Validate` the single gate that accepts a directory and returns a record — or a typed error.

### Phase 1: Locate the skill directory

**User Intent:** Feed one skill root (`<name>/SKILL.md`) into the parser.

**Actions:** Call `Parse(dir)` with a filesystem path whose final component is the skill name.

**Pain / Risk:** Missing `SKILL.md`; `dir` is a file; empty directory; symlink loops; path with a trailing separator that changes `filepath.Base`.

**Success Signal:** A missing file returns a typed not-found error; a present `SKILL.md` proceeds to frontmatter split.

### Phase 2: Split frontmatter from body

**User Intent:** Separate YAML metadata from Markdown instructions.

**Actions:** Read `SKILL.md`; take YAML between the opening and closing `---` fences; treat everything after the closing fence as body.

**Pain / Risk:** File does not start with `---`; missing closing fence; CRLF vs LF; a thematic `---` later in the body; YAML that looks like Markdown; frontmatter keys leaking into `Body`.

**Success Signal:** `Body` is the Markdown after the closing `---` and does not contain the frontmatter block.

### Phase 3: Decode optional and required fields

**User Intent:** Materialize `name`, `description`, and any optional Agent Skills fields.

**Actions:** Decode YAML; require string `name` and `description`; accept `license`, `compatibility`, `metadata`, `allowed-tools` when present; ignore unknown keys (promptkit extras).

**Pain / Risk:** `name` as a YAML bool/int; `metadata` values that are not strings; `allowed-tools` as a sequence instead of a space-separated string; empty optional fields vs omitted fields.

**Success Signal:** Required fields are strings; optional fields are populated only when present and omitted (empty/nil) when absent.

### Phase 4: Validate name, description, and directory match

**User Intent:** Reject skills that violate Agent Skills identifier rules before they can be catalogued.

**Actions:** Run `Validate` (also from `Parse`): name charset/length/hyphens; `name` equals parent directory; description non-empty and ≤ 1024.

**Pain / Risk:** Uppercase leftovers from promptkit; leading/trailing/consecutive hyphens; name > 64; directory renamed without updating frontmatter; empty or > 1024 description.

**Success Signal:** Invalid skills return sentinel-wrapped errors; valid skills (including every shipped `.agents/skills/*/SKILL.md`) parse.

### Phase 5: Prove catalog-scale parse budget

**User Intent:** Confirm 200 skills stay inside the harness NFR (p99 < 50 ms per parse).

**Actions:** Parse 200 synthetic valid skills; compute p99 of per-skill `Parse` times.

**Pain / Risk:** Slow disks in CI; measuring setup instead of parse; a too-tight gate that flakes; a too-loose gate that misses regressions.

**Success Signal:** Timed test (or benchmark) asserts catalog parse of 200 synthetic skills p99 < 50 ms.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Skills on disk but no loader | 1 | One `Parse(dir)` entry that later discovery can call |
| Body mixed with YAML | 2 | Fence-only split; body may still contain `---` |
| promptkit extra keys | 3 | Ignore unknowns; do not fail shipped skills |
| Strict `name` vs existing folders | 4 | Fix `SKILL.md` / directory names; never relax Validate |
| Silent performance cliff | 5 | p99 gate on 200 synthetics |

### North Star Summary

A skill directory is either a validated `Skill` (frontmatter + body, optional fields only when authored) or a clear error. Every in-repo shipped skill parses. Invalid names and descriptions never reach the catalog. Two hundred synthetic parses stay under the 50 ms p99 gate.

### Stressors

1. Shipped promptkit `SKILL.md` files violate `name` charset or directory match.
2. Skill body contains one or more `---` thematic breaks after the frontmatter.
3. `SKILL.md` uses CRLF line endings.
4. YAML types `name` as a number or bool (`yes`).
5. Description is exactly 1024 characters (must accept) vs 1025 (must reject).
6. Name is exactly 64 characters (must accept) vs 65 (must reject).
7. Directory renamed so `name` no longer matches `filepath.Base(dir)`.
8. Optional fields omitted vs present (including `metadata` nil vs map).
9. Unknown frontmatter keys (for example `disable-model-invocation`) must not fail parse.
10. Missing `SKILL.md` or missing closing `---` fence.
11. Malformed YAML in the frontmatter block.
12. Two hundred synthetic skills on a slow CI disk vs the p99 < 50 ms gate.
13. Leading hyphen, trailing hyphen, and consecutive hyphens in `name`.
14. Uppercase letters in `name` while the directory is lowercase.
15. `allowed-tools` present on a shipped skill (`orchestrator`) must parse as a string.

## 3. UX Implementation and Assessment

The “user” in this journey is the developer/agent calling `Parse` / `Validate` (operators feel this later via `/skills`).

### Time to First Value
- [ ] `Parse(dir)` returns a `Skill` or error with no extra setup
- [ ] Shipped `.agents/skills` parse without rewriting the skill format

### Onboarding Clarity
- [ ] Package godoc names `Parse` and `Validate` as the public surface
- [ ] Error sentinels name the failed rule (name, description, mismatch, file, frontmatter)

### Production-Ready Defaults
- [ ] Unknown frontmatter keys are ignored (fail-closed only on required-rule violations)
- [ ] No config file is required to parse a directory

### Golden Path Quality
- [ ] Valid fixture → `Skill` with correct name, description, and body
- [ ] Full optional-field fixture populates license, compatibility, metadata, allowed-tools

### Decision Load
- [ ] Callers do not choose a parse mode
- [ ] `Parse` always validates; `Validate` is available for already-built records

### Progressive Complexity
- [ ] Minimal skill is name + description + body
- [ ] Optional fields are opt-in and omitted when absent

### Error Quality
- [ ] Name errors mention uppercase, hyphen, length, or charset
- [ ] Missing file and invalid YAML are distinct sentinels

### Failure Safety
- [ ] Invalid skills do not produce a partial “success” record from `Parse`
- [ ] `Validate` is pure (no filesystem writes)

### Runtime Transparency
- [ ] `Skill.Dir` records the directory that was parsed
- [ ] `Skill.Body` is inspectable Markdown, not a hidden side channel

### Debuggability
- [ ] Testdata fixtures map 1-1 to table-test names
- [ ] Shipped-skill test names each directory that failed

### Cross-Surface Consistency
- [ ] Rules match agentskills.io `name` / `description` constraints
- [ ] Field names match the spec: `allowed-tools`, not an invented alias

### Workflow Consistency
- [ ] Package lives at `internal/skills` as the spec architecture table states
- [ ] Tests follow table-driven + testdata conventions in docs/testing.md

### Change Safety
- [ ] Fixtures are hermetic; shipped-skill test is read-only
- [ ] Validate is not relaxed to paper over bad `SKILL.md` files

### Experimentation Safety
- [ ] Synthetic catalog test uses `t.TempDir`
- [ ] No production skill is rewritten except to satisfy the spec `name` rules

### Interaction Latency
- [ ] Catalog parse of 200 synthetic skills p99 < 50 ms
- [ ] Single-skill parse is a single read + YAML decode

### Developer Feedback Speed
- [ ] `go test ./internal/skills` reports the first failing rule
- [ ] Table tests isolate one violation per invalid fixture

### Team Scale
- [ ] Fixtures and shipped skills live in git
- [ ] The same `Validate` rules apply to every author

### System Scale
- [ ] Parser does not load `scripts/`, `references/`, or `assets/` (later steps)
- [ ] API is `Parse` + `Validate` only — discovery is Step 2

### Right Behavior by Default
- [ ] Directory mismatch is an error, not a warning
- [ ] Empty description is an error

### Anti-Bypass Design
- [ ] `Parse` cannot return a `Skill` that fails `Validate`
- [ ] Tests require every shipped skill to parse (no skip list)

## 4. Tests

### TC-01: valid_minimal

**Given** `testdata/valid-minimal` with name, description, and a Markdown body.
**When** `Parse` is called on that directory.
**Then** the `Skill` has matching name, description, empty optional fields, and body equal to the Markdown after the closing fence.

### TC-02: valid_full_optional_fields

**Given** `testdata/valid-full` with license, compatibility, metadata, and allowed-tools.
**When** `Parse` is called.
**Then** those optional fields are populated and body excludes frontmatter.

### TC-03: valid_body_contains_fence

**Given** a skill whose body includes a later `---` thematic break.
**When** `Parse` is called.
**Then** the body still contains that `---` and does not lose trailing sections.

### TC-04: name_rejects_uppercase

**Given** a fixture whose `name` contains uppercase letters.
**When** `Parse` is called.
**Then** the error wraps the invalid-name sentinel.

### TC-05: name_rejects_hyphen_and_length

**Given** fixtures for leading hyphen, trailing hyphen, consecutive hyphens, and name length > 64.
**When** `Parse` is called on each.
**Then** each error wraps the invalid-name sentinel.

### TC-06: name_rejects_directory_mismatch

**Given** a valid `name` that is not the parent directory.
**When** `Parse` is called.
**Then** the error wraps the name-mismatch sentinel.

### TC-07: description_rejects_empty_and_too_long

**Given** fixtures with an empty description and a description longer than 1024 characters.
**When** `Parse` is called.
**Then** each error wraps the invalid-description sentinel.

### TC-08: missing_file_and_invalid_frontmatter

**Given** a directory without `SKILL.md`, a file without fences, and invalid YAML.
**When** `Parse` is called.
**Then** missing file and invalid frontmatter sentinels are returned.

### TC-09: shipped_skills_parse

**Given** every `.agents/skills/*/SKILL.md` in this repository.
**When** a test walks those directories and calls `Parse`.
**Then** every shipped skill parses and `name` equals the directory name.

### TC-10: catalog_p99

**Given** 200 synthetic valid skill directories.
**When** each is parsed and durations are collected.
**Then** the p99 duration is < 50 ms.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 1:

- `internal/skills` exports `Parse(dir string) (Skill, error)` and `Validate(Skill) error`
- `name` rejects uppercase, leading/trailing hyphen, consecutive hyphens, length > 64, mismatch with directory
- `description` rejects empty and length > 1024
- Optional fields (`license`, `compatibility`, `metadata`, `allowed-tools`) parse when present and are omitted when absent
- Body is the Markdown after the closing `---` (frontmatter not mixed into body)
- Table tests cover valid/invalid fixtures under `internal/skills/testdata/`
- A test walks `.agents/skills/*/SKILL.md` and requires every shipped skill to parse
- Catalog parse of 200 synthetic skills p99 < 50 ms (benchmark or timed test)
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 1
- Implementation files: `internal/skills/skill.go`, `internal/skills/parse.go`, `internal/skills/validate.go`
- Test files: `internal/skills/parse_test.go`, `internal/skills/testdata/`

## Implementation

Files created:
- `internal/skills/skill.go` — `Skill` record, field/limit constants, sentinels
- `internal/skills/parse.go` — `Parse(dir string) (Skill, error)` and frontmatter decode
- `internal/skills/validate.go` — `Validate(Skill) error`
- `internal/skills/parse_test.go` — table fixtures, shipped-skill walk, catalog p99
- `internal/skills/testdata/` — valid and invalid `SKILL.md` fixtures

Files modified:
- `specs/agent-harness/ROADMAP.md` — Step 1 DoD ticks and traceability
- `docs/testing.md` — skills parser test row
- `.agents/skills/*/SKILL.md` — none (shipped names already match the spec)
