# How to package an agent plugin

<!-- How-to template: Good Docs Project (CC-BY 4.0). Task-oriented; not a tutorial. -->

## Goal

Ship a closed Agent Plugins 1.0 package so spin merges its immediate skills and attaches its MCP servers without one failure unloading the other.

## Prerequisites

- A plugin root directory you control.
- `spin` built so you can run `spin plugin validate <dir>`.

## Steps

1. Create a plugin root with `plugin.json` at the top. Required fields are `$schema` and `name`. The schema identifier must be:

   `https://agent-plugins.org/schemas/1.0.0/plugin.schema.json`

   `name` is lowercase alphanumeric, may include `-` or `.`, must start and end with alphanumeric, no `--` or `..`, max 64 characters.

   Closed top-level fields (unknown keys are reported and ignored): `$schema`, `name`, `version`, `description`, `author` (`name` / `email` / `url` only), `homepage`, `repository`, `license`, `keywords`, `extensions`.

   ```json
   {
     "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
     "name": "review-kit",
     "version": "1.0.0",
     "description": "Review skills and an optional MCP server"
   }
   ```

2. Put skills as **immediate** children of `skills/` only (`skills/<name>/SKILL.md`). Nested `skills/**` directories are not discovered. A skill that fails `SKILL.md` parse is skipped with a warning; the rest of the plugin still loads.

3. Keep every plugin-relative path inside the root. `Contain` requires the path to start with `./`. Bare names and `..` escapes fail. Symlinks that resolve outside the root also fail. Only fields defined as plugin-relative paths are contained — `mcp.json` `command` argv is not treated as a package path unless the token itself starts with `./`.

4. Optional: add `mcp.json` at the plugin root (the only MCP config path). Required `$schema`:

   `https://agent-plugins.org/schemas/1.0.0/mcp.schema.json`

   `mcpServers` is a closed object. Each server must set `type` to `stdio`, `streamable-http`, or `sse`. Empty `mcpServers` is valid. A missing or invalid `mcp.json` disables MCP for that plugin and records a warning; skills stay loaded. A server that fails to connect is skipped; sibling servers and skills stay.

5. Optional spin extras live under the reverse-domain namespace `com.spin.agent` (directory `com.spin.agent/hooks/<script-name>` and/or `extensions["com.spin.agent"]`). Foreign client dirs such as `com.example.client/` are ignored. Hook scripts run with cwd set to the plugin root. See [hooks reference](../reference/hooks.md).

6. Place the plugin root where spin discovers packages (first `plugin.json` `name` wins):

   | Root | How it is scanned |
   |------|-------------------|
   | `<workdir>/.spin/plugins/<plugin-root>/` | Each immediate child directory |
   | `~/.spin/plugins/<plugin-root>/` | Each immediate child directory |
   | `plugins.paths` in config | A plugin root (has `plugin.json`) or a directory of roots |

   A fatal `plugin.json` skips that root only. Other plugins still load.

7. Dry-run without starting MCP or merging skills:

   ```bash
   spin plugin validate ./my-plugin
   ```

   The report prints `plugin`, `schema`, immediate skill names, warnings, and `ok`.

## Result

`/skills` lists plugin skills as `source=plugin:<name>`. MCP failures do not unload those skills. Paths that omit `./` or escape the root never resolve.

## See also

- [How to write an agent skill](agent-skills.md)
- [How to spawn and wait on subagents](subagents.md)
- [Agent Plugins 1.0](https://agent-plugins.org/specification)
