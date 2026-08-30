# Journey: Resume a previous TUI session

User: engineer who quit `spin` (Ctrl-D, `/exit`, or SIGTERM) and wants the
same conversation back without re-explaining context.

## CJM

1. **Stop** — session transcript and index are on disk under `session_dir`.
2. **Restart** — `spin tui` opens a fresh empty session.
3. **Discover** — `/help` mentions `/resume`.
4. **Browse** — `/resume` prints a numbered list: short ID, message count,
   relative age, first user-line preview.
5. **Select** — `/resume 2` or `/resume a1b2` or `/resume last`.
6. **Continue** — status bar shows the restored ID; the next prompt uses the
   restored history; new turns append to that session's transcript.

## Stressors

- Empty session store
- Current session is the only entry
- Stale index with `message_count: 0` but a real transcript
- Ambiguous ID prefix
- Out-of-range index
- Selecting the live session
- Missing transcript file
- Tilde-expanded vs raw `session_dir`
- ACP clients must not see `/resume`

## Implementation

- `internal/session/resume.go` — list, resolve, format, read transcript
- `internal/conversation/resume.go` — switch ID / history / writer
- `internal/commands/commands.go` — `/resume` command
- `cmd/spin/tui_command_context.go` — TUI handler + UI refresh
- `internal/protocol/acp/agent.go` — hide TUI-only commands

## Traceability

Roadmap: `specs/roadmaps/ROADMAP-tui-resume.md`
FRD: `specs/frds/FRD-tui-resume.md`
