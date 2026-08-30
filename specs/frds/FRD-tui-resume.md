# FRD: TUI `/resume` — continue a previous session

## Problem

A graceful TUI stop flushes the transcript and session index, but the next
`spin tui` always mints a new session. Users cannot pick up the conversation
they just left.

## Goal

Add a TUI slash command `/resume` that lists prior sessions for the current
working directory and restores one so the next prompt continues that history.

## Non-goals

- ACP `session/load` already resumes; this FRD does not change ACP.
- Cross-workdir browsing, session delete/rename, and full transcript replay
  in the scrollback.

## Behavior

1. `/resume` with no args lists resumable sessions (newest first), excluding
   the current session and sessions with an empty transcript.
2. `/resume <n>` selects by 1-based list index.
3. `/resume last` selects the newest resumable session.
4. `/resume <id>` selects by full ID or unique prefix.
5. On success the live conversation ID, in-memory history, and transcript
   writer switch to the chosen session. Later turns append to that transcript.
6. The status bar conversation ID and token count refresh after a resume.
7. ACP hides `/resume` (TUI-only), same as `/exit`.

## Acceptance

- Listing never overflows a single summary line per session.
- A resumed session's next `RunTurn` sees the restored messages in
  `MessagesForLLM`.
- New messages after resume append to the existing `transcript.jsonl`.
- Unknown selector, empty store, and already-current ID return a clear error
  without mutating history.
