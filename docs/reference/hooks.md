# Hooks reference

<!-- Reference template: Good Docs Project (CC-BY 4.0). Lookup only; not a how-to. -->

Lifecycle hook scripts. Ten events. Filenames are `Event.ScriptName()` values, not the event tokens.

## Synopsis

```
~/.spin/hooks/<script-name>
<workdir>/.spin/hooks/<script-name>
<plugin-root>/com.spin.agent/hooks/<script-name>
```

stdin: JSON `EventContext`. stdout: optional JSON `{ "reason": "...", "updated_input": ... }`. Exit `2` vetoes on blocking events.

## Description

The runner walks global scripts, then project scripts, then plugin extras. Plugin extras come from `com.spin.agent/hooks/` or `extensions["com.spin.agent"].hooks` (file map or directory). Foreign client directories are ignored. Plugin scripts run with cwd = plugin root; paths are `Contain`'d (`./`, no `..`).

`~` in the global dir expands via `UserHomeDir`. Default global dir is `~/.spin/hooks`.

## Events

| Event | Filename | Blocking (exit 2 vetoes) |
|-------|----------|--------------------------|
| `SESSION_START` | `session-start` | no |
| `USER_PROMPT_SUBMIT` | `user-prompt-submit` | yes |
| `PRE_TOOL_USE` | `pre-tool-use` | yes |
| `POST_TOOL_USE` | `post-tool-use` | no |
| `POST_TOOL_USE_FAILURE` | `post-tool-use-failure` | no |
| `SUBAGENT_START` | `subagent-start` | yes |
| `SUBAGENT_STOP` | `subagent-stop` | no |
| `PRE_COMPACT` | `pre-compact` | no |
| `STOP` | `stop` | no |
| `SESSION_END` | `session-end` | no |

`SUBAGENT_START` exit 2 prevents the child pid (no process). `SUBAGENT_STOP` runs on success, failure, and crash.

## `updated_input`

JSON field `updated_input` on hook stdout. It is a **full replacement** of tool argument JSON, not a merge.

| Shape | Effect |
|-------|--------|
| JSON object (`{"path":"safe"}`) | Replaces the tool args with that object |
| JSON string that is itself an object (`"{\"path\":\"safe\"}"`) | Same: parsed object replaces args |
| Empty, missing, or non-object | Original args unchanged |
| Last writer wins | Project (then later scripts) override global |

Applies on `PRE_TOOL_USE` (and other scripts that return the field). Blocking exit 2 still vetoes; `reason` is the operator-visible veto text.

## stdin context

| Field | JSON key | When set |
|-------|----------|----------|
| Event | `event` | Always |
| Session id | `session_id` | Always |
| Workdir | `work_dir` | Always |
| Tool name | `tool_name` | Tool events |
| Tool input | `tool_input` | Tool events |
| Tool response | `tool_response` | Post-tool events |

## Session end vs ACP cancel

`SESSION_END` (after `STOP`) runs on `Conversation.Close`. TUI Ctrl-C and `/exit` Close. ACP `session/cancel` CancelAlls running A2A tasks and does not Close the conversation — `SESSION_END` does not fire on that path.

## See also

- [How to package an agent plugin](../how-to/agent-plugins.md)
- [How to spawn and wait on subagents](../how-to/subagents.md)
- [Compact reference](compact.md)
