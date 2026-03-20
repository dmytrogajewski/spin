# Journey R4.1 — Complete Tool Kind Mapping

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Every Spin tool call reported to the client has a non-nil `ToolKind`, enabling rich per-kind UI rendering

## Context

The ACP spec defines 10 tool kinds: `read`, `edit`, `delete`, `move`, `search`, `execute`, `think`, `fetch`, `switch_mode`, `other`.
Spin's `mapToolNameToKind()` in `approval_handler.go` currently maps only 5 tool names (`read_file`, `write_file`, `list_directory`, `shell_command`, `file_search`) to 4 kinds (`read`, `edit`, `execute`, `search`). All other tools return `nil`.

## Complete Tool Inventory

| Spin Tool Name | Current Kind | Target Kind | Rationale |
|---|---|---|---|
| `read_file` | `read` | `read` | No change |
| `list_directory` | `read` | `read` | No change |
| `write_file` | `edit` | `edit` | No change |
| `edit_file` | nil | `edit` | Edits file content |
| `apply_patch` | nil | `edit` | Applies code patches |
| `shell_command` | `execute` | `execute` | No change |
| `start_process` | nil | `execute` | Starts background process |
| `file_search` | `search` | `search` | No change |
| `find_symbol` | nil | `search` | Searches for symbols |
| `find_references` | nil | `search` | Searches for references |
| `rename_symbol` | nil | `move` | Renames = moves symbol identity |
| `fetch_url` | nil | `fetch` | Fetches web content |
| `web_search` | nil | `fetch` | Fetches search results |
| `capture_web_screenshot` | nil | `fetch` | Fetches visual web content |
| `open_browser` | nil | `fetch` | Opens/fetches browser content |
| `git_context` | nil | `read` | Reads git state |
| `git_operation` | nil | `execute` | Runs git commands |
| `get_context` | nil | `read` | Reads context information |
| `memory` | nil | `think` | Stores/retrieves reasoning state |
| `scratchpad` | nil | `think` | Scratch space for reasoning |
| `get_process_output` | nil | `read` | Reads process output |
| `kill_process` | nil | `execute` | Kills a running process |
| `list_processes` | nil | `read` | Lists running processes |
| (unknown) | nil | `other` | Fallback for any unmapped tool |

## Phases

### Phase 1 — Client Receives Tool Call Notification
- **Trigger**: Agent executes any tool during a prompt turn
- **Current behavior**: Client receives `tool_call` with `kind: null` for 18 of 23 tools
- **Target behavior**: Client receives `tool_call` with a meaningful `kind` for every tool

### Phase 2 — Client Renders Tool Call UI
- **UX improvement**: IDE can show different icons/colors per kind (file icon for read, pencil for edit, terminal for execute, magnifier for search, globe for fetch, brain for think)
- **Friction removed**: No more "unknown tool type" fallback rendering

## Implementation Notes

- Single function `mapToolNameToKind()` in `approval_handler.go` — used by both `notifications.go` and `approval_handler.go`
- Change default from `nil` to `ToolKindOther` — every tool gets a kind
- No new files needed, no new dependencies
- Backward-compatible: existing mapped tools keep their kinds

## Test Plan

| # | Test | Type | Input | Expected |
|---|------|------|-------|----------|
| 1 | Existing mappings preserved | Unit | `read_file`, `write_file`, `shell_command`, `file_search`, `list_directory` | Same kinds as before |
| 2 | edit_file maps to edit | Unit | `edit_file` | `ToolKindEdit` |
| 3 | apply_patch maps to edit | Unit | `apply_patch` | `ToolKindEdit` |
| 4 | start_process maps to execute | Unit | `start_process` | `ToolKindExecute` |
| 5 | find_symbol maps to search | Unit | `find_symbol` | `ToolKindSearch` |
| 6 | find_references maps to search | Unit | `find_references` | `ToolKindSearch` |
| 7 | rename_symbol maps to move | Unit | `rename_symbol` | `ToolKindMove` |
| 8 | fetch_url maps to fetch | Unit | `fetch_url` | `ToolKindFetch` |
| 9 | web_search maps to fetch | Unit | `web_search` | `ToolKindFetch` |
| 10 | capture_web_screenshot maps to fetch | Unit | `capture_web_screenshot` | `ToolKindFetch` |
| 11 | open_browser maps to fetch | Unit | `open_browser` | `ToolKindFetch` |
| 12 | git_context maps to read | Unit | `git_context` | `ToolKindRead` |
| 13 | git_operation maps to execute | Unit | `git_operation` | `ToolKindExecute` |
| 14 | get_context maps to read | Unit | `get_context` | `ToolKindRead` |
| 15 | memory maps to think | Unit | `memory` | `ToolKindThink` |
| 16 | scratchpad maps to think | Unit | `scratchpad` | `ToolKindThink` |
| 17 | get_process_output maps to read | Unit | `get_process_output` | `ToolKindRead` |
| 18 | kill_process maps to execute | Unit | `kill_process` | `ToolKindExecute` |
| 19 | list_processes maps to read | Unit | `list_processes` | `ToolKindRead` |
| 20 | Unknown tool maps to other | Unit | `completely_unknown_tool` | `ToolKindOther` |
| 21 | All tools return non-nil | Unit | every known tool name | `!= nil` |

## Implementation

- **Modified**: `internal/protocol/acp/approval_handler.go` — `mapToolNameToKind()`
- **Modified**: `internal/protocol/acp/notifications_test.go` — `TestMapToolNameToKind`
