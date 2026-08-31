# How to write and load an agent skill

<!-- How-to template: Good Docs Project (CC-BY 4.0). Task-oriented; not a tutorial. -->

## Goal

Add a portable Agent Skill so spin lists it in `/skills` and the model can load its body with the `skill` tool.

## Prerequisites

- A workspace you can write to (the spin workdir).
- `spin` built (`make build`) so you can open a TUI session and type `/skills`.

## Steps

1. Create a skill directory whose name is the skill id. The id must match `SKILL.md` `name`: lowercase letters, digits, and single hyphens; no leading or trailing hyphen; no `--`; at most 64 characters.

2. Write `SKILL.md` with YAML frontmatter and a Markdown body. `name` and `description` are required. `description` is what the skill does and when to use it (max 1024 characters). Optional fields: `license`, `compatibility` (max 500), `metadata` (string map), `allowed-tools` (space-separated experimental allowlist).

   ```markdown
   ---
   name: review-pr
   description: Review a pull request for regressions and missing tests.
   ---

   # Review a pull request

   Read the diff, list missing tests, and report blockers only.
   ```

3. Put the directory in one of the discovery roots (first name wins; missing roots are skipped):

   | Priority | Root | Catalog `source` |
   |----------|------|------------------|
   | Project | `<workdir>/.agents/skills/<name>/` | `project` |
   | Project (interop) | `<workdir>/.claude/skills/<name>/` | `project` |
   | User | `~/.spin/skills/<name>/` | `user` |
   | User | `~/.agents/skills/<name>/` | `user` |
   | Plugin | each loaded plugin's `skills/<name>/` | `plugin:<plugin-name>` |
   | Bundled | `$SPIN_HOME/skills/<name>/` | `bundled` |

   Collision: project wins, then user, then plugin, then bundled. Duplicate names still show `source` so you can tell which copy won.

4. Optional: add `scripts/`, `references/`, or `assets/` next to `SKILL.md`. The catalog stores name and description only. The body is not loaded until activation. A relative path is one hop and must stay inside the skill root (`../` is rejected).

5. In the TUI, type `/`. Commands and catalog skills appear above the prompt. Tab completes the highlighted row. `/skills` still prints `name  source  description`, or `No skills found.` if every root is empty.

6. Type `/review-pr` and press Enter to activate that skill on the next turn (the `SKILL.md` body is prepended). Add a remainder after the name (`/review-pr Auth.go`) to keep operator text. Registered commands win over skill names (`/help` stays help).

7. Type `@` to attach a project file (Tab completes). Paste or Ctrl-V a copied path or clipboard image to insert `@rel` tokens. Submit injects attached file text into the turn.

8. The model can still call `skill` (alias `load_skill`) with `name`. The tool returns the `SKILL.md` body and the skill root. Pass optional `path` (for example `references/extra.md`) to read one contained file. Unknown names return a typed error that lists catalog names, not other skill bodies.

## Result

`/skills` shows your skill. `/review-pr` or a `skill` / `load_skill` call injects that body into the turn. Nested `skills/**` trees are not searched — see [SPEC.md](../../specs/agent-harness/SPEC.md) (not shipped; recursive discovery is an anti-goal).

## See also

- [How to package an agent plugin](agent-plugins.md)
- [Compact reference](../reference/compact.md)
- [Agent Skills specification](https://agentskills.io/specification)
