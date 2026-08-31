# JOURNEY-006-load-com-spin-agent-extension-hooks: Load com.spin.agent extension hooks from plugins

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Load `com.spin.agent` extension hooks from plugins

## 1. Journey

When **an operator installs an Agent Plugin that ships spin lifecycle scripts** I want **those scripts registered with the existing hook runner from `com.spin.agent` only** so I **can veto or audit tools from a portable package without putting spin fields on closed `plugin.json` keys**.

## 2. CJM

Alex already has plugin skills in `/skills` (Step 5) and a closed, contained `plugin.json` (Step 4). A community plugin that wants to block a dangerous tool today has nowhere portable to put a `pre-tool-use` script: adding a top-level `hooks` key would violate Agent Plugins 1.0, and a foreign client dir (`com.example.client/`) must not run as spin. This journey discovers hook files under `com.spin.agent/hooks/` (and/or `extensions["com.spin.agent"]`), registers them with `internal/safety/hooks.Runner` using existing `Event.ScriptName()` filenames, executes them with the plugin root as cwd, and keeps package-path containment. `UpdatedInput` / `~` expansion (Step 7) and new lifecycle emitters (Step 8) stay out of scope.

### Phase 1: Discover spin hook scripts

**User Intent:** Find only spin-owned hook scripts inside a loaded plugin.

**Actions:** After `Load`, read `extensions["com.spin.agent"]` when it is an object and/or scan `com.spin.agent/hooks/` for files named by `Event.ScriptName()` (`session-start`, `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `post-tool-use-failure`, `subagent-start`, `subagent-stop`, `pre-compact`, `stop`, `session-end`). Ignore sibling extension directories.

**Pain / Risk:** A `com.example.client/hooks/pre-tool-use` script is treated as spin; unknown filenames (`pre-tool-use.sh`) are executed; a top-level `hooks` field is added to `plugin.json`; `extensions` that is not an object (already ignored) is re-parsed as hooks; missing `com.spin.agent/` looks like a load failure.

**Success Signal:** Known `ScriptName()` files under `com.spin.agent/hooks/` appear as discovered hooks. Foreign extension dirs contribute zero hooks. The plugin still loads when the spin dir is absent.

### Phase 2: Contain hook paths

**User Intent:** Hook files stay inside the plugin root, same as other package paths.

**Actions:** Resolve each hook as a plugin-relative path starting with `./` through existing `Contain`. Skip a hook (do not reject the plugin) when the path is bare, escapes with `../`, or is a symlink that resolves outside the root.

**Pain / Risk:** `./com.spin.agent/hooks/../secret` is cleaned and allowed; an extensions-declared path `../hooks/pre-tool-use` is followed; a symlink script runs from outside the package; containment is reimplemented in the runner instead of `Contain`.

**Success Signal:** Contained regular files are registered. Escaping and non-`./` paths are omitted. The plugin remains loaded.

### Phase 3: Register with the existing runner

**User Intent:** Plugin scripts fire on the same events as project and global hooks.

**Actions:** Pass discovered scripts into `hooks.Runner` (config or equivalent register API) without changing event names or inventing a second runner. `PRE_TOOL_USE` looks up `pre-tool-use`. Project and global dirs keep their current merge order; plugin scripts are extras for matching names.

**Pain / Risk:** A new runner type forks policy; script names diverge from `Event.ScriptName()`; registration requires a new portable `plugin.json` key; production `NewRunner` sites never receive the scripts so they only exist in tests.

**Success Signal:** A unit test with a fake plugin + real `Runner` records a `PRE_TOOL_USE` fire. Production builder/ACP `NewRunner` calls include plugin scripts from Discover.

### Phase 4: Execute with plugin root as cwd

**User Intent:** Scripts that read `./` files inside the package resolve against the plugin, not the operator workdir.

**Actions:** When the runner executes a plugin hook, set the process cwd to that plugin’s root. Inherit the same environment as project hooks. Do not inject extra credentials.

**Pain / Risk:** cwd stays the session workdir so `./policy` misses the package; `cmd.Env` is replaced and drops the host policy; secrets are copied into the hook env “to be helpful”; timeout/exit-2 semantics change.

**Success Signal:** A script that writes `pwd` records the plugin root. Env is the same inherit-process policy as project hooks. Exit 0 / 2 / timeout behavior is unchanged.

### Phase 5: Stay inside Step 6

**User Intent:** Ship discovery + registration + cwd/containment without pulling later hook work.

**Actions:** Do not expand `~` in `GlobalDir`. Do not apply `HookResult.UpdatedInput`. Do not add emitters for unwired events.

**Pain / Risk:** Step 7/8 land in the same change; a new top-level manifest field “for convenience”; foreign client dirs are scanned “just in case”.

**Success Signal:** Roadmap Step 6 DoD is the only closed hook work. Portable `plugin.json` keys stay closed.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| No portable place for spin hooks | 1 | `com.spin.agent/hooks/` + `extensions["com.spin.agent"]` |
| Foreign client scripts would run as spin | 1 | Ignore dirs that are not `com.spin.agent` |
| `../` hook path leaks the host | 2 | Reuse `Contain`; skip the hook |
| Plugin script `./` files miss the package | 4 | cwd = plugin root |
| Extra secrets in hook env | 4 | Same inherit-env policy as project hooks |

### North Star Summary

Alex drops a plugin that contains `com.spin.agent/hooks/pre-tool-use`. Spin discovers that file by `Event.ScriptName()`, ignores `com.example.client/`, refuses escaped hook paths, registers the script with the existing runner, and runs it with the plugin root as cwd and no extra credentials. Portable `plugin.json` keys stay closed. `UpdatedInput` and new lifecycle emitters remain later steps.

### Stressors

1. `com.spin.agent/hooks/` is missing — plugin load stays valid; zero spin hooks.
2. `com.example.client/hooks/pre-tool-use` exists beside a spin hook — only the spin file is discovered.
3. A file named `pre-tool-use.sh` sits in `com.spin.agent/hooks/` — it is not a `ScriptName()` match and is ignored.
4. `extensions["com.spin.agent"]` is absent but the conventional hooks directory is present — directory scan still finds scripts.
5. `extensions["com.spin.agent"]` names a contained hooks directory or per-event `./` paths — those files are discovered after `Contain`.
6. `extensions["com.spin.agent"].hooks` points at `../escape` or a bare `data` path — `Contain` rejects; the hook is skipped.
7. A hook file is a symlink that resolves outside the plugin root — `Contain` rejects; the hook is skipped.
8. `extensions` is a non-object (already ignored) — no crash; conventional `com.spin.agent/hooks/` can still be scanned.
9. Plugin hook `pwd` / relative file access uses the plugin root, not the session workdir.
10. Runner env for plugin hooks matches project hooks (no injected tokens or extra `SPIN_*` secrets).
11. Unknown `ScriptName` keys in an extensions hooks object are ignored.
12. Multiple plugins each ship `pre-tool-use` — each contained script is registered; cwd is that plugin’s root.
13. Global/project `pre-tool-use` still run; plugin scripts do not replace them.
14. A blocking plugin `pre-tool-use` that exits 2 still blocks (existing exit-code-2 semantics).
15. Adding a top-level `hooks` key to `plugin.json` is reported as unknown and ignored — it does not register scripts.

## 3. UX Implementation and Assessment

The operator-facing surface is unchanged: hooks remain workspace-trusted scripts. Value is that a plugin’s spin scripts fire without a new slash command or a new portable manifest field.

### Time to First Value
- [ ] A plugin with `com.spin.agent/hooks/pre-tool-use` fires on the next `PRE_TOOL_USE` after load
- [ ] No extra CLI flag is required to enable plugin hooks

### Onboarding Clarity
- [ ] Spin-only scripts live under `com.spin.agent/`, which matches the spec’s reverse-domain rule
- [ ] A skipped escaping hook does not fail the whole plugin

### Production-Ready Defaults
- [ ] Missing `com.spin.agent/` is valid (zero hooks)
- [ ] Foreign extension directories are not scanned

### Golden Path Quality
- [ ] `pre-tool-use` discovered via `Event.ScriptName()` fires on `PRE_TOOL_USE`
- [ ] Execution cwd is the plugin root

### Decision Load
- [ ] Operators do not choose a hook backend
- [ ] Filenames are the existing ten `ScriptName()` values, not a new catalog

### Progressive Complexity
- [ ] Directory convention works with no `extensions` object
- [ ] `extensions["com.spin.agent"]` hook paths are opt-in overlays

### Error Quality
- [ ] Escaping hook paths are skipped with containment, not executed
- [ ] Unknown extension hook keys are ignored, not fatal

### Failure Safety
- [ ] A bad hook path does not unload skills or MCP
- [ ] Hook timeout and non-block errors keep existing runner behavior

### Runtime Transparency
- [ ] Discovered hooks are named with `ScriptName()` strings
- [ ] Each hook records the plugin root as cwd

### Debuggability
- [ ] Unit test writes a marker from `PRE_TOOL_USE` in a fake runner
- [ ] A foreign-dir fixture proves `com.example.client/` did not run

### Cross-Surface Consistency
- [ ] TUI builder and ACP `NewRunner` both receive plugin scripts
- [ ] Event filenames match `internal/safety/hooks` (`pre-tool-use`, …)

### Workflow Consistency
- [ ] Discovery lives in `internal/plugins`; execution stays in `internal/safety/hooks`
- [ ] `Contain` remains the only package-path gate

### Change Safety
- [ ] Portable `plugin.json` keys stay closed — no new top-level hook field
- [ ] Step 7 `UpdatedInput` / `~` expansion is not implemented here

### Experimentation Safety
- [ ] Tests use temp plugin trees, not the operator’s `~/.spin`
- [ ] Hook scripts in tests do not need the executable bit (`/bin/sh` invocation)

### Interaction Latency
- [ ] Discovery is directory + `Contain` + `ScriptName()` match
- [ ] Runner timeout budget is unchanged

### Developer Feedback Speed
- [ ] `go test ./internal/plugins ./internal/safety/hooks` isolates discovery vs execute
- [ ] The PRE_TOOL_USE unit test names the fake runner path

### Team Scale
- [ ] Plugin hooks are files in the package and can be versioned with the plugin
- [ ] The same `ScriptName()` map applies to every plugin

### System Scale
- [ ] A new lifecycle event is picked up when `Event.ScriptName()` grows (later steps add emitters)
- [ ] Additional plugins append scripts; they do not replace the runner

### Right Behavior by Default
- [ ] Only `com.spin.agent` hooks run
- [ ] Env policy matches project hooks (no extra credentials)

### Anti-Bypass Design
- [ ] Paths that fail `Contain` cannot register
- [ ] A top-level `hooks` key cannot sneak past the closed schema

## 4. Tests

### TC-01: discover_pre_tool_use_script_name

**Given** a loaded plugin with `com.spin.agent/hooks/pre-tool-use`.
**When** spin-agent hooks are discovered.
**Then** one hook has `ScriptName` `pre-tool-use` and a contained path under the plugin root.

### TC-02: foreign_extension_dir_ignored

**Given** a plugin that has `com.example.client/hooks/pre-tool-use` and no `com.spin.agent/` hooks.
**When** spin-agent hooks are discovered.
**Then** the result is empty; the foreign script is not registered.

### TC-03: all_script_names_from_event_map

**Given** a plugin with a regular file for every `Event.ScriptName()`.
**When** hooks are discovered.
**Then** the set of names equals the existing map (session-start through session-end) and no extras.

### TC-04: unknown_filename_ignored

**Given** `com.spin.agent/hooks/pre-tool-use.sh` only.
**When** hooks are discovered.
**Then** zero hooks (name is not `Event.ScriptName()`).

### TC-05: extensions_object_hooks_dir

**Given** `extensions["com.spin.agent"]` with `"hooks": "./com.spin.agent/hooks"` and a `pre-tool-use` file there.
**When** hooks are discovered.
**Then** `pre-tool-use` is found via `Contain` on that directory.

### TC-06: extensions_escape_path_skipped

**Given** `extensions["com.spin.agent"].hooks` mapping `pre-tool-use` to `../secret` or `./../secret`.
**When** hooks are discovered.
**Then** that event has no hook; the plugin still loads.

### TC-07: symlink_escape_skipped

**Given** `com.spin.agent/hooks/pre-tool-use` as a symlink to a file outside the plugin root.
**When** hooks are discovered.
**Then** that hook is omitted (`Contain` / path escape).

### TC-08: top_level_hooks_key_ignored

**Given** a `plugin.json` with an unknown top-level `hooks` field and no `com.spin.agent/` dir.
**When** the plugin is loaded and hooks are discovered.
**Then** the field is reported unknown and ignored; zero spin hooks are registered.

### TC-09: plugin_hook_fires_pre_tool_use

**Given** a fake runner configured with a plugin `pre-tool-use` script that writes a marker.
**When** `Execute` runs with `EventPreToolUse`.
**Then** the marker exists (DoD unit test).

### TC-10: plugin_hook_cwd_is_plugin_root

**Given** a plugin hook that writes `pwd` to a file under the plugin root.
**When** `PRE_TOOL_USE` executes.
**Then** the file contents equal the plugin root (not the test process cwd).

### TC-11: plugin_hook_inherits_env

**Given** a plugin hook that prints a host env var and no extra credentials were configured.
**When** the hook runs.
**Then** the process env matches project-hook inherit policy (the var is visible; no injected secret keys).

### TC-12: plugin_hook_exit_2_blocks

**Given** a plugin `pre-tool-use` that prints a reason and exits 2.
**When** `Execute` runs `PRE_TOOL_USE`.
**Then** `HookResult.Blocked` is true and `Reason` is the script stdout.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 6:

- `com.spin.agent/hooks/pre-tool-use` (etc.) are discovered using existing `Event.ScriptName()` filenames
- Foreign extension dirs (`com.example.client/`) are ignored
- Plugin hook scripts execute with plugin root as cwd
- Package-path containment still applies to hook files inside the plugin
- Unit test: plugin hook fires on `PRE_TOOL_USE` in a fake runner
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 6
- Implementation files: `internal/plugins/extensions.go`, `internal/plugins/plugin.go`, `internal/safety/hooks/runner.go`, `internal/conversation/builder.go`, `internal/conversation/agent.go`, `cmd/spin/acp.go`
- Test files: `internal/plugins/extensions_test.go`, `internal/safety/hooks/runner_test.go`

## Implementation

Files created:
- `internal/plugins/extensions.go` — `DiscoverAgentHooks` + `HookScripts`; conventional `com.spin.agent/hooks/` and `extensions["com.spin.agent"]` overlays; `Contain` on every hook path
- `internal/plugins/extensions_test.go` — ScriptName discovery, foreign dir ignore, extensions paths, escape/symlink skip, PRE_TOOL_USE fake runner fire
- `internal/safety/hooks/runner_test.go` — plugin script fire, cwd = plugin root, inherit env, exit 2, unknown name ignored
- `specs/journeys/JOURNEY-006-load-com-spin-agent-extension-hooks.md` — this journey

Files modified:
- `internal/plugins/plugin.go` — `SpinAgentExtension` and `SpinAgentHooksDir` constants
- `internal/safety/hooks/runner.go` — `PluginScript` / `Config.PluginScripts`; execute extras with plugin cwd
- `internal/conversation/builder.go` — `pluginHookScripts` from Discover
- `internal/conversation/agent.go` — `NewRunner` receives plugin scripts
- `cmd/spin/acp.go` — ACP `NewRunner` receives plugin scripts
- `specs/agent-harness/ROADMAP.md` — Step 6 DoD ticks and traceability
- `docs/testing.md` — journey 006 test row
- `specs/journeys/JOURNEY-006-load-com-spin-agent-extension-hooks.md` — this implementation section
