# JOURNEY-005-load-plugins-merge-skills: Load plugins — merge skills, isolate MCP failures

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Load plugins — merge skills, isolate MCP failures

## 1. Journey

When **an operator drops Agent Plugins into project, user, or config search paths** I want to **see each plugin’s skills in `/skills` and the Composer catalog, tagged `plugin:<name>`, while MCP servers map into the existing manager** so I **can use portable skills even when a plugin’s MCP command fails**.

## 2. CJM

Alex installed a community Agent Plugin (skill + `mcp.json`) under `.spin/plugins`. Step 4 can validate one directory. Step 2 lists project/user/bundled skills only. Without this journey, plugin skills never appear next to promptkit skills, `mcp.json` is unread, and a dead MCP command would look like a catalog outage. This journey discovers plugin roots, merges immediate skills into the Step 2 catalog with `source=plugin:<name>`, maps explicit MCP transports into the existing MCP manager, and keeps skills listed when a server fails to start. `com.spin.agent` hooks are out of scope (Step 6).

### Phase 1: Discover plugin roots

**User Intent:** Find every plugin package spin should load for this workdir.

**Actions:** Scan `<workDir>/.spin/plugins/*`, `~/.spin/plugins/*`, and each config `plugins.paths` entry. Treat a path that itself contains `plugin.json` as one root; otherwise scan immediate children. Call Step 4 `Load` on each root.

**Pain / Risk:** Missing search dirs are treated as fatal; a file next to plugin dirs is loaded as a plugin; hidden `.` entries become roots; the same plugin name in project and home double-registers; config paths that are search dirs vs single roots are ambiguous.

**Success Signal:** Missing roots are ignored. Each valid plugin appears once (project first, then user, then extra paths). Fatal `plugin.json` skips that root only.

### Phase 2: Merge plugin skills into the catalog

**User Intent:** See plugin skills in the same list the model and `/skills` see.

**Actions:** Convert each loaded plugin’s immediate skills into catalog entries with `source=plugin:<name>`. Insert them after project and user skills and before bundled. First name wins.

**Pain / Risk:** Plugin bodies leak into Composer; plugin skills overwrite a project skill of the same name; source is the bare plugin name without the `plugin:` prefix; `/skills` still uses `OptionsFor` and never sees plugins; nested `skills/**` reappears.

**Success Signal:** `/skills` prints `name  plugin:<plugin-name>  description`. Composer catalog stays metadata-only. A project skill of the same name keeps `source=project`.

### Phase 3: Load `mcp.json`

**User Intent:** Read the plugin’s portable MCP config without guessing a transport.

**Actions:** If `mcp.json` is present, require `$schema` and `mcpServers`. Accept an empty `mcpServers` object. Map each server that declares `type` as `stdio`, `streamable-http`, or `sse` into `mcp.ServerConfig`. Skip entries with a missing or unknown type.

**Pain / Risk:** Empty `mcpServers` rejects the file; missing `type` defaults to stdio (spec forbids guessed transport); unknown top-level fields or a schema mismatch disable skills; `command` argv is treated as a package path; schema is fetched from the network.

**Success Signal:** Valid `mcp.json` (including empty servers) loads. Transports are the declared `type` only. Invalid MCP config disables MCP for that plugin and leaves skills in place.

### Phase 4: Isolate MCP start failures

**User Intent:** A dead MCP server must not hide the plugin’s skills.

**Actions:** Hand each mapped server to the existing MCP service (`ConnectServer`) independently. Record connection/handshake failures as warnings. Continue with sibling servers and other plugins.

**Pain / Risk:** One `Initialize` error unloads the plugin or the whole catalog; a failed connect removes already-merged skills; a later plugin is not discovered because the first MCP attach returned an error.

**Success Signal:** A fixture whose MCP command is missing or exits still lists its skills with `plugin:<name>`. Other plugins in the same scan still load.

### Phase 5: Survive a fatal neighbor

**User Intent:** One broken package does not take down the rest of the plugin set.

**Actions:** Discover a valid sample plugin beside a directory whose `plugin.json` is missing or fatally invalid. Load continues.

**Pain / Risk:** The first fatal `Load` aborts the scan; skipped roots are silent with no warning; the valid plugin’s skills are omitted because the scan stopped.

**Success Signal:** The valid plugin’s skills appear. The fatal root is skipped and reported. MCP attach is not attempted for the skipped root.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Plugin skills invisible in `/skills` | 2 | Same Discover merge the Composer uses |
| MCP failure looks like a skill outage | 4 | Independent attach; skills already merged |
| Guessed stdio transport | 3 | Require explicit `type`; skip otherwise |
| One bad `plugin.json` blanks the catalog | 5 | Per-root `Load`; skip and continue |
| Extra install paths only in docs | 1 | Config `plugins.paths` as extra roots |

### North Star Summary

Alex drops a sample plugin (skill + `mcp.json`) into `.spin/plugins`. `/skills` lists the skill as `plugin:<name>`. Empty `mcpServers` is fine. `stdio`, `streamable-http`, and `sse` are used only when declared. A plugin whose MCP command fails still lists its skills. A neighbor with a fatal `plugin.json` is skipped. Step 6 hooks are untouched.

### Stressors

1. `<workDir>/.spin/plugins` is missing — discovery continues with home and config paths.
2. `~/.spin/plugins` is missing — project plugins still load.
3. Config `plugins.paths` points at a single plugin root (has `plugin.json`).
4. Config `plugins.paths` points at a directory of plugin roots (no `plugin.json` at the path itself).
5. The same plugin `name` exists in project and user trees — project wins; skills are not duplicated.
6. A plugin skill shares a name with a project skill — project keeps `source=project`.
7. `mcp.json` has `$schema` and empty `mcpServers` — MCP is valid and unused.
8. A server entry omits `type` — that entry is skipped; transport is not guessed as stdio.
9. A server declares `type: sse` or `type: streamable-http` — mapped transport matches the declaration.
10. A fixture MCP `command` is not on PATH / exits on start — skills for that plugin still appear in the catalog.
11. Fatal `plugin.json` (missing file or invalid schema) next to a valid plugin — only the fatal root is skipped.
12. Invalid `mcp.json` (bad JSON, wrong `$schema`, missing `mcpServers`) — MCP disabled for that plugin; skills remain.
13. A plugin-relative stdio `command` of `./bin/server` that escapes via `../` — that server is skipped; skills remain.
14. Hidden `.` entries and non-directory children under `.spin/plugins` are not treated as plugin roots.
15. Composer catalog lines stay name + description (no skill body headings from the plugin `SKILL.md`).

## 3. UX Implementation and Assessment

The operator-facing surface is `/skills` (and the Composer catalog). MCP attach is silent except for warnings. No new slash command.

### Time to First Value
- [ ] Dropping a valid plugin under `.spin/plugins` makes its skills appear on the next `/skills`
- [ ] No extra CLI flag is required for the project and user search paths

### Onboarding Clarity
- [ ] Source tags use `plugin:<name>` so the operator can tell plugin skills from project skills
- [ ] Fatal plugin skips name the root and the typed error

### Production-Ready Defaults
- [ ] Missing plugin search dirs are ignored
- [ ] Empty `mcpServers` is valid
- [ ] Transport is never inferred from `command` or `url`

### Golden Path Quality
- [ ] Sample plugin (skill + `mcp.json`) contributes one catalog line with a plugin source tag
- [ ] Mapped servers use the declared `type`

### Decision Load
- [ ] Operators do not choose a merge mode
- [ ] First name wins (project, user, plugin, bundled)

### Progressive Complexity
- [ ] A skill-only plugin (no `mcp.json`) is valid
- [ ] Config `plugins.paths` is opt-in

### Error Quality
- [ ] Missing `type` is reported as an invalid server entry, not silently turned into stdio
- [ ] MCP connect errors name the server and do not wrap the whole plugin as failed

### Failure Safety
- [ ] MCP attach errors do not remove catalog entries
- [ ] Fatal `plugin.json` does not abort the remaining scan

### Runtime Transparency
- [ ] Catalog `Source` is `plugin:<manifest.name>`
- [ ] Plugin `Warnings` collect skipped skills, MCP parse issues, and attach failures

### Debuggability
- [ ] Testdata sample plugin maps to the integration test
- [ ] Independent-failure fixture is required evidence, not optional

### Cross-Surface Consistency
- [ ] `/skills`, Composer, and the `skill` tool registry share the same Discover merge
- [ ] MCP mapping uses existing `mcp.ServerConfig` / `ConnectServer`

### Workflow Consistency
- [ ] Discovery lives in `internal/plugins`; catalog merge stays in `internal/skills`
- [ ] `plugins` already imports `skills`; skills accept contributions without importing plugins

### Change Safety
- [ ] Step 4 `Load` / `Contain` contracts stay; MCP parse is additive and non-fatal
- [ ] Step 6 hook directories are not read

### Experimentation Safety
- [ ] Failing-MCP fixtures use a command that is not a real server
- [ ] Tests do not require a network MCP endpoint to pass

### Interaction Latency
- [ ] Discovery is filesystem + JSON decode
- [ ] MCP handshake failure is per-server and does not retry siblings

### Developer Feedback Speed
- [ ] `go test ./internal/plugins ./internal/skills` isolates parse vs merge vs attach
- [ ] Integration test names the sample plugin fixture

### Team Scale
- [ ] `plugins.paths` is a versionable config key
- [ ] The same Load rules apply to every extra path

### System Scale
- [ ] Adding another plugin root does not change skill `Parse` or MCP transport validation
- [ ] New transports require an explicit `type` — no URL sniffing

### Right Behavior by Default
- [ ] Project skills beat plugin skills on name collision
- [ ] Empty catalog remains valid when no plugins exist

### Anti-Bypass Design
- [ ] A server without `type` cannot enter the MCP manager
- [ ] `Load` still rejects a fatal manifest; Discover cannot resurrect it

## 4. Tests

### TC-01: sample_plugin_skills_in_catalog

**Given** a workdir whose `.spin/plugins/sample-plugin` has `plugin.json`, `skills/greet/SKILL.md`, and `mcp.json`.
**When** the catalog is discovered (the same path `/skills` uses).
**Then** `greet` is present with `source=plugin:sample-plugin` and the fixture description.

### TC-02: empty_mcp_servers_valid

**Given** `mcp.json` with the canonical `$schema` and `"mcpServers": {}`.
**When** the plugin is loaded.
**Then** load succeeds, MCP servers are empty, and skills (if any) remain.

### TC-03: explicit_stdio_transport

**Given** a server entry `{ "type": "stdio", "command": "true" }`.
**When** the entry is mapped to `mcp.ServerConfig`.
**Then** `Transport` is `stdio` (not empty).

### TC-04: explicit_streamable_http_and_sse

**Given** server entries with `type` `streamable-http` and `sse` and valid URLs.
**When** they are mapped.
**Then** each `Transport` equals the declared type. No URL-based guess is used.

### TC-05: missing_type_not_guessed

**Given** a server entry that has `command` but no `type`.
**When** `mcp.json` is parsed.
**Then** that server is skipped and is not mapped as stdio.

### TC-06: failing_mcp_still_lists_skills

**Given** a plugin with a valid skill and `mcp.json` whose stdio `command` does not exist.
**When** the catalog is built and MCP attach is attempted.
**Then** the skill is still listed with a plugin source tag. Attach records a warning.

### TC-07: fatal_plugin_json_skips_only_that_plugin

**Given** one valid plugin root and one sibling directory with a fatal `plugin.json`.
**When** Discover runs.
**Then** the valid plugin’s skills are in the catalog. The fatal root is skipped. Discover does not return a fatal error.

### TC-08: config_plugins_paths

**Given** `plugins.paths` pointing at a plugin root outside the default search dirs.
**When** Discover uses those extra paths.
**Then** that plugin’s skills appear with `plugin:<name>`.

### TC-09: project_skill_wins_over_plugin

**Given** a project skill and a plugin skill with the same name.
**When** Discover merges contributions.
**Then** the catalog entry has `source=project` and the project description.

### TC-10: invalid_mcp_json_keeps_skills

**Given** a valid `plugin.json` + skill and a malformed `mcp.json`.
**When** Load runs.
**Then** the plugin loads with skills; MCP is disabled; a warning is recorded.

### TC-11: missing_plugin_roots_ignored

**Given** a workdir and home that have no `.spin/plugins` directories.
**When** Discover runs.
**Then** the result has zero plugins and is not an error.

### TC-12: skills_command_lists_plugin_source

**Given** the default `/skills` command (no test options override) and a sample plugin in the workdir.
**When** Execute runs.
**Then** the output contains the skill name and `plugin:<name>`.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 5:

- Valid plugins contribute skills to `/skills` with plugin source tags
- `mcp.json` `$schema` + `mcpServers` loaded; empty `mcpServers` is valid
- Transport types `stdio`, `streamable-http`, `sse` are explicit; no guessed transport
- A fixture whose MCP command fails still lists its skills
- Fatal `plugin.json` skips that plugin only; other plugins still load
- Integration test uses a sample plugin (skill + mcp.json)
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 5
- Implementation files: `internal/plugins/discover.go`, `internal/plugins/mcpjson.go`, `internal/plugins/attach.go`, `internal/plugins/load.go`, `internal/skills/discover.go`, `internal/skills/catalog.go`, `internal/mcp/transport.go`, `internal/config/config_v2.go`, `internal/commands/skills.go`, `internal/conversation/builder.go`
- Test files: `internal/plugins/discover_test.go`, `internal/plugins/mcpjson_test.go`, `internal/skills/discover_test.go`, `internal/mcp/transport_test.go`, `internal/plugins/testdata/sample-plugin/`, `internal/plugins/testdata/failing-mcp/`

## Implementation

Files created:
- `internal/plugins/discover.go` — scan `.spin/plugins/*`, home, and `plugins.paths`; fatal `plugin.json` skips that root only
- `internal/plugins/mcpjson.go` — parse `mcp.json` (`$schema` + `mcpServers`); empty servers valid; explicit `type` only
- `internal/plugins/attach.go` — `AttachMCP` maps servers into `mcp.Service.ConnectServer` independently
- `internal/plugins/discover_test.go` — sample plugin catalog, extra paths, failing MCP, fatal neighbor, `/skills` source tag
- `internal/plugins/mcpjson_test.go` — empty servers, explicit transports, missing type not guessed, invalid MCP keeps skills
- `internal/plugins/testdata/sample-plugin/` — skill `greet` + empty `mcpServers`
- `internal/plugins/testdata/failing-mcp/` — skill `still-here` + stdio command that is not on PATH

Files modified:
- `internal/plugins/plugin.go` — `MCPFile` / `MCPServer` records and MCP sentinels
- `internal/plugins/load.go` — `Load` parses `mcp.json` without failing the plugin
- `internal/skills/catalog.go` — `PluginSource`, `PluginSkill`, `Options.PluginSkills`
- `internal/skills/discover.go` — merge plugin skills after project/user, before bundled
- `internal/skills/discover_test.go` — plugin source tag, project wins, plugin before bundled
- `internal/mcp/transport.go` — `ParsePluginTransport` rejects empty type
- `internal/mcp/transport_test.go` — explicit stdio / streamable-http / sse; empty rejected
- `internal/config/config_v2.go` — `PluginsV2.Paths`
- `internal/config/loader_v2.go` — bind `plugins.paths`
- `internal/config/config_v2_test.go` — paths unmarshal
- `internal/commands/skills.go` — default `/skills` uses `plugins.DiscoverCatalog`
- `internal/conversation/builder.go` — catalog merge + isolated MCP attach
- `internal/conversation/tools.go` — skill tool registry uses the same catalog
- `cmd/spin/acp.go` — Composer catalog and MCP attach
- `specs/agent-harness/ROADMAP.md` — Step 5 DoD ticks and traceability
- `docs/testing.md` — journey 005 test row
- `specs/journeys/JOURNEY-005-load-plugins-merge-skills.md` — this implementation section

`com.spin.agent` hooks are not loaded (Step 6).
