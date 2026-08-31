# JOURNEY-004-parse-agent-plugins: Parse Agent Plugins 1.0 with path containment

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Parse Agent Plugins 1.0 with path containment

## 1. Journey

When **an operator (or later the plugin loader) points spin at a plugin directory** I want to **validate `plugin.json` against the closed Agent Plugins 1.0 schema and reject package paths that escape the plugin root** so I **know the package is portable and contained before any MCP start or skill-catalog merge**.

## 2. CJM

Alex is packaging a third-party Agent Plugin (or reviewing one dropped into `.spin/plugins`). Today spin has no plugin loader: a missing manifest, an unknown top-level field treated as a hard fail, a recursive `skills/**` walk, or a `../` package path would either reject a valid 1.0 plugin or leak files outside the package. This journey makes `internal/plugins.Load` the single gate that accepts a plugin root and returns a validated record plus immediate-child skills — or a typed error. User-visible value is `spin plugin validate`. MCP is not started. Skills are not merged into Discover.

### Phase 1: Locate the plugin root

**User Intent:** Feed one plugin directory into the loader.

**Actions:** Call `Load(dir)` (or `spin plugin validate <dir>`) with a filesystem path whose root should contain `plugin.json`.

**Pain / Risk:** Missing `plugin.json`; `dir` is a file; empty directory; `plugin.json` is a symlink that resolves outside the root; path with a trailing separator that changes `filepath.Clean`.

**Success Signal:** A missing manifest rejects the whole plugin with a typed not-found error. A present, contained `plugin.json` proceeds to schema validation.

### Phase 2: Validate the closed manifest schema

**User Intent:** Accept only Agent Plugins 1.0 top-level fields; require `$schema` and `name`.

**Actions:** Decode JSON; require canonical `$schema` and a valid `name`; accept permitted optional fields; report and ignore unknown top-level keys; reject other type/constraint violations.

**Pain / Risk:** `$schema` missing or a foreign URI (must not fetch); `name` violates charset/length; `author` has extra keys; `extensions` is a non-object (must report and ignore, not reject); unknown field treated as fatal (spec says report and continue).

**Success Signal:** Valid manifests load. Unknown top-level fields appear as warnings and are not stored. Wrong types, bad names, and unsupported schemas reject the plugin.

### Phase 3: Contain plugin-relative paths

**User Intent:** Package paths stay inside the plugin root. Values that are not path fields stay opaque.

**Actions:** `Contain(root, rel)` requires a `./` prefix, resolves against the root, and rejects escape. Apply containment when resolving discovered `SKILL.md` paths. Do not treat MCP `command` argv as package paths.

**Pain / Risk:** `../secret` is cleaned into a sibling and allowed; bare `data` is joined as a relative path; `./../etc/passwd` starts with `./` but escapes; a symlink skill directory points outside the root; `command` arguments are scanned as paths (spec violation).

**Success Signal:** `./` paths that stay inside resolve. `../` and bare `data` fail. Escaping skill paths are skipped, not followed. MCP is unread.

### Phase 4: Discover immediate skills only

**User Intent:** List skills the plugin actually ships, using Step 1 `Parse`, without walking nested trees.

**Actions:** If `skills/` is absent, return zero skills. If present, take only immediate child directories that contain `SKILL.md`. Parse each with `skills.Parse`. Skip invalid or escaping skills; do not reject the plugin.

**Pain / Risk:** Recursive search picks up `skills/deploy/nested/SKILL.md`; a file named `SKILL.md` directly under `skills/` is treated as a skill; missing `skills/` rejects the plugin; one bad skill unloads the package.

**Success Signal:** Nested skill fixtures contribute only the immediate child. Missing `skills/` is valid. Invalid individual skills are warnings.

### Phase 5: Report validation to the operator

**User Intent:** See whether a directory is a valid Agent Plugin without starting MCP or changing the catalog.

**Actions:** Run `spin plugin validate <dir>`. Print name, schema, skill names, and warnings. Exit nonzero on reject.

**Pain / Risk:** Success hides unknown-field warnings; failure is a generic "invalid"; validate starts MCP or writes config; help does not list the command.

**Success Signal:** Valid fixtures print `ok` and warnings when present. Missing/invalid manifests exit nonzero. No MCP process is spawned.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| No plugin loader | 1 | One `Load(dir)` entry later discovery can call |
| Closed schema vs extra keys | 2 | Report + ignore unknowns; fail only on real violations |
| Path escape via `../` or bare names | 3 | `Contain` requires `./` and post-resolve enclosure |
| Recursive skill walk | 4 | Immediate children of `skills/` only |
| Silent invalid packages | 5 | `spin plugin validate` as the operator gate |

### North Star Summary

A plugin directory is either a validated `Plugin` (closed manifest, contained package paths, immediate skills only) or a clear typed error. Unknown top-level fields are visible and ignored. Missing `skills/` is empty, not fatal. The operator can prove this with `spin plugin validate` without starting MCP or merging the catalog.

### Stressors

1. `plugin.json` is absent — the whole plugin must be rejected.
2. `$schema` is missing, empty, or not the 1.0.0 canonical URI (must not fetch a schema).
3. `name` has uppercase, leading/trailing hyphen or period, consecutive `--` or `..`, or length > 64.
4. An unknown top-level field (for example `commands`) is reported and ignored; the plugin still loads.
5. `author` contains a field other than `name` / `email` / `url` — fatal schema violation.
6. `extensions` is a string or array — reported and ignored; not fatal.
7. Plugin-relative path `../secret` or `./../secret` escapes the root.
8. Bare path `data` is rejected (not plugin-relative).
9. `skills/deploy/nested/SKILL.md` is ignored; only `skills/deploy` loads.
10. Missing `skills/` is valid and yields zero skills.
11. A skill directory that is a symlink resolving outside the plugin root is skipped.
12. MCP `command` argv is not read or treated as a package path (no `mcp.json` load in this step).
13. `plugin.json` is valid JSON but a top-level array — reject.
14. An immediate child skill fails Agent Skills `Parse` — skip that skill, keep the plugin.
15. `skills` exists but is a regular file — component type invalid; plugin still loads with zero skills.

## 3. UX Implementation and Assessment

The operator-facing surface is `spin plugin validate`. Callers of `Load` / `Contain` are later loader steps.

### Time to First Value
- [ ] `spin plugin validate <dir>` reports ok or a typed error with no extra setup
- [ ] A valid fixture loads without writing config or starting MCP

### Onboarding Clarity
- [ ] Package godoc names `Load` and `Contain` as the public surface
- [ ] Error sentinels name the failed rule (missing manifest, schema, name, path)

### Production-Ready Defaults
- [ ] Unknown top-level fields are warnings, not rejects
- [ ] Missing `skills/` is valid (zero skills)
- [ ] No network fetch of `$schema`

### Golden Path Quality
- [ ] Valid fixture → `Plugin` with name, schema, and immediate skills
- [ ] `spin plugin validate` prints name, skill list, and `ok`

### Decision Load
- [ ] Callers do not choose a parse mode
- [ ] `Load` always validates the closed schema and containment

### Progressive Complexity
- [ ] Minimal plugin is `$schema` + `name`
- [ ] Optional metadata and `extensions` are opt-in

### Error Quality
- [ ] Missing `plugin.json` is a distinct sentinel
- [ ] Path errors distinguish “not plugin-relative” from “escapes root”

### Failure Safety
- [ ] Invalid manifests do not produce a successful `Load`
- [ ] `Contain` is pure path math (no writes)

### Runtime Transparency
- [ ] `Plugin.Root` records the directory that was loaded
- [ ] Unknown fields and skipped skills appear in `Warnings`

### Debuggability
- [ ] Testdata fixtures map 1-1 to table-test names
- [ ] Validate output lists skill names that were accepted

### Cross-Surface Consistency
- [ ] Field set matches Agent Plugins 1.0 (`$schema`, `name`, metadata, `extensions`)
- [ ] Skill `Parse` reuses `internal/skills` (Step 1)

### Workflow Consistency
- [ ] Package lives at `internal/plugins` as the spec architecture table states
- [ ] Tests follow table-driven + testdata conventions in docs/testing.md

### Change Safety
- [ ] Fixtures are hermetic; validate is read-only
- [ ] Schema is not relaxed to accept unknown required-field types

### Experimentation Safety
- [ ] Symlink escape tests use `t.TempDir`
- [ ] No production catalog or MCP registry is mutated

### Interaction Latency
- [ ] Validate is one JSON decode + one `skills/` directory read
- [ ] No MCP handshake

### Developer Feedback Speed
- [ ] `go test ./internal/plugins` reports the first failing rule
- [ ] Table tests isolate one violation per invalid case

### Team Scale
- [ ] Fixtures live in git
- [ ] The same `Load` rules apply to every author

### System Scale
- [ ] Loader does not start MCP or merge Discover (later steps)
- [ ] `Contain` is reusable for later plugin-relative fields only

### Right Behavior by Default
- [ ] Paths must start with `./`
- [ ] Bare `data` and `../` fail

### Anti-Bypass Design
- [ ] `Load` cannot return a `Plugin` whose manifest failed the closed schema
- [ ] Containment cannot be skipped for discovered `SKILL.md` paths

## 4. Tests

### TC-01: valid_plugin

**Given** `testdata/valid-plugin` with `$schema`, `name`, optional metadata, and `skills/summarize/SKILL.md`.
**When** `Load` is called.
**Then** the plugin has the fixture name, canonical schema, one skill `summarize`, and no fatal error.

### TC-02: unknown_field_ignored

**Given** `testdata/unknown-field` with a valid required pair plus an extra top-level key.
**When** `Load` is called.
**Then** the plugin loads, the unknown key is reported in warnings, and it is not stored as a permitted field.

### TC-03: nested_skill_ignored

**Given** `testdata/nested-skill` with `skills/deploy/SKILL.md` and `skills/deploy/nested/SKILL.md`.
**When** `Load` is called.
**Then** only `deploy` is present; `nested` is absent.

### TC-04: escape_path

**Given** `testdata/escape-path` as the plugin root.
**When** `Contain` is called with `./skills/summarize/SKILL.md`, `../secret`, and `data`.
**Then** the `./` in-root path succeeds; `../secret` and `data` fail.

### TC-05: missing_plugin_json

**Given** a directory with no `plugin.json`.
**When** `Load` is called.
**Then** the error wraps the missing-manifest sentinel and no plugin is returned.

### TC-06: missing_skills_valid

**Given** a valid `plugin.json` and no `skills/` directory.
**When** `Load` is called.
**Then** the plugin loads with zero skills.

### TC-07: schema_and_name_required

**Given** manifests missing `$schema`, missing `name`, empty `name`, or a non-canonical `$schema`.
**When** `ParseManifest` / `Load` is called.
**Then** each case rejects the plugin.

### TC-08: name_constraints

**Given** names with uppercase, leading/trailing hyphen or period, consecutive `--` or `..`, and length > 64.
**When** the manifest is parsed.
**Then** each error wraps the invalid-name sentinel.

### TC-09: other_schema_violations_reject

**Given** a top-level array, a non-string `version`, or an `author` object with an unknown key.
**When** the manifest is parsed.
**Then** the plugin is rejected.

### TC-10: contain_dot_dot_prefix

**Given** a plugin root and the path `./../secret`.
**When** `Contain` is called.
**Then** the error wraps the path-escape sentinel (prefix `./` is not enough).

### TC-11: symlink_skill_escape_skipped

**Given** a temp plugin whose `skills/leak` is a symlink to a directory outside the root.
**When** `Load` is called.
**Then** the plugin loads and `leak` is not in the skill list.

### TC-12: spin_plugin_validate

**Given** the valid-plugin and missing-manifest paths.
**When** `spin plugin validate <dir>` is executed.
**Then** the valid path prints the plugin name and `ok` (exit 0); the missing path exits nonzero and does not start MCP.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 4:

- `plugin.json` requires `$schema` and `name`; unknown top-level fields are reported and ignored; other schema violations reject the plugin
- Permitted fields: `version`, `description`, `author`, `homepage`, `repository`, `license`, `keywords`, `extensions`
- Plugin-relative paths must start with `./` and resolve inside the plugin root; `../` and bare `data` fail
- Skills are immediate children of `skills/` only (no recursive search)
- Missing `plugin.json` rejects the whole plugin; missing `skills/` is valid (zero skills)
- Testdata fixtures: valid plugin, escape path, nested skill ignored, unknown field ignored
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 4
- Implementation files: `internal/plugins/plugin.go`, `internal/plugins/manifest.go`, `internal/plugins/contain.go`, `internal/plugins/load.go`, `cmd/spin/plugin.go`
- Test files: `internal/plugins/load_test.go`, `cmd/spin/plugin_test.go`, `internal/plugins/testdata/`

## Implementation

Files created:
- `internal/plugins/plugin.go` — `Plugin` / `Manifest` / `Author` records, field constants, sentinels
- `internal/plugins/manifest.go` — `ParseManifest` closed-schema decode; unknown fields reported and ignored
- `internal/plugins/contain.go` — `Contain(root, rel)` requires `./` and stays inside the plugin root
- `internal/plugins/load.go` — `Load(dir)` reads `plugin.json`, discovers immediate `skills/` children via `skills.Parse`
- `internal/plugins/load_test.go` — fixture table, name/schema violations, symlink escape skip
- `internal/plugins/testdata/valid-plugin/` — full permitted-field plugin with `skills/summarize`
- `internal/plugins/testdata/escape-path/` — Contain root for `./` success and `../` / `data` reject
- `internal/plugins/testdata/nested-skill/` — `skills/deploy` plus ignored `skills/deploy/nested`
- `internal/plugins/testdata/unknown-field/` — extra top-level `commands` reported and ignored; no `skills/`
- `cmd/spin/plugin.go` — `spin plugin validate <dir>`
- `cmd/spin/plugin_test.go` — CLI validate on fixtures

Files modified:
- `cmd/spin/root.go` — register `plugin` command
- `cmd/spin/stubs_test.go` — expect `plugin` on root
- `specs/agent-harness/ROADMAP.md` — Step 4 DoD ticks and traceability
- `docs/testing.md` — journey 004 test row
- `specs/journeys/JOURNEY-004-parse-agent-plugins.md` — this implementation section

MCP is not started. Skills are not merged into Discover.
