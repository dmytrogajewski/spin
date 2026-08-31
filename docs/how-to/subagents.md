# How to spawn, wait, and cancel subagents

<!-- How-to template: Good Docs Project (CC-BY 4.0). Task-oriented; not a tutorial. -->

## Goal

Start a local A2A child as an OS process, list it, wait until it is terminal, and cancel it — without treating the child as a live LLM agent.

## Prerequisites

- `spin` built (`make build`). The parent starts the same binary as `spin a2a`.
- A TUI or `exec` session so `/tasks`, `/task`, and `/agents` are available.

## Steps

1. Know the process model. The parent admits a builtin spec (`explorer`, `planner`, `reviewer`, `ask_user`) and starts:

   ```bash
   spin a2a --spec explorer --stdio
   ```

   `--spec` is required. `--stdio` (default) serves NDJSON-RPC on stdin/stdout. The first framed message is the Agent Card; later lines are `message/send` and `tasks/*`. Child logs go to stderr, never the RPC stream. Alternate bind:

   ```bash
   spin a2a --spec explorer --listen unix:///tmp/spin-a2a.sock
   ```

   There is no `spin spawn` command. Children do not receive a `spawn` tool unless a spec explicitly allowlists it (deny-by-default).

2. Treat the local child as an echo Task handler. `spin a2a` answers `message/send` with the in-process MemoryHandler: the Task artifact is the first text part of the inbound message. The child process has an isolated TaskFrame and history, but it does **not** run a live child LLM or ReAct loop. Spec prompts and tool allowlists are on the card and harness; they are not executed as a second agent in this binding. A full child LLM loop is described in [SPEC.md](../../specs/agent-harness/SPEC.md) and is not shipped.

3. List builtin specs and A2A peers:

   ```
   /agents
   ```

4. After the parent spawns a child (blocking `Spawn` or non-blocking `SpawnBackground`), list mixed work:

   ```
   /tasks
   ```

   Each row is `kind=agent|shell` plus a typed id (`agent:…` or `shell:…`), spec or command, and state. Untyped ids that exist in both stores are ambiguous — use the prefix.

5. Wait for an A2A task (shell ids are rejected here):

   ```
   /task wait agent:<id>
   ```

   The model-facing tools are `list_agent_tasks`, `wait_agent_task`, and `cancel_agent_task`. Shell rows stay on `start_process` / `list_processes` / `kill_process`.

6. Cancel:

   ```
   /task cancel agent:<id>
   /task cancel shell:<id>
   ```

   Agent cancel is `tasks/cancel` then SIGTERM. Shell cancel is SIGTERM then SIGKILL. `/task wait` does not take the spawn semaphore.

7. Leave the parent cleanly. TUI Ctrl-C and `/exit` call `Conversation.Close`: CancelAll on every working A2A task (cancel then SIGTERM), then `STOP` and `SESSION_END` hooks. ACP `session/cancel` CancelAlls running A2A tasks on that conversation and does **not** Close it — `SESSION_END` does not run on ACP cancel. The next `spin` start reaps leftover pid and socket files under `$XDG_RUNTIME_DIR/spin/a2a` (or a spin-prefixed temp dir). Children exit when stdin or the Unix client socket closes.

8. Remote HTTPS cards are a separate path. Config `a2a.allowlist` is a list of exact `https://` Agent Card URLs. The default list is empty (every remote card is forbidden). Off-list URLs are rejected before dial. Entries that are not absolute https fail config validation.

## Result

`/tasks` shows `kind=agent` rows with typed ids. `/task wait` returns a terminal row. `/task cancel` stops the child. The local A2A process is a card-plus-echo server, not a second LLM.

## TaskFrame phases

Parent turns map `/mode` onto TaskFrame `phase` 1:1: `regular`, `review`, `compact`, `planning`. Child harness maps `planner` → `planning`, `reviewer` → `review`, otherwise `regular`. SPEC phase names `plan|work|review|ask` are not shipped.

## See also

- [Hooks reference](../reference/hooks.md) (`SUBAGENT_START` / `SUBAGENT_STOP`)
- [Compact reference](../reference/compact.md)
- [A2A specification](https://a2a-protocol.org/latest/specification/)
