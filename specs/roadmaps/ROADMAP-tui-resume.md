# Roadmap: TUI `/resume`

Journey: [JOURNEY-tui-resume.md](../journeys/JOURNEY-tui-resume.md)
FRD: [FRD-tui-resume.md](../frds/FRD-tui-resume.md)

## 1. Session catalog

- [x] `ReadTranscript` loads JSONL without creating the file
- [x] `ListResumable` skips current ID and empty transcripts
- [x] `ResolveSelector` accepts index, `last`, full ID, unique prefix
- [x] `FormatResumeList` is one line per session and fits 80 columns

## 2. Conversation switch

- [x] `Conversation.Resume` replaces history, ID, and transcript writer
- [x] Next `RunTurn` sees restored messages
- [x] New messages append to the resumed transcript
- [x] Index `message_count` updates as turns persist

## 3. Slash command

- [x] `/resume` lists; `/resume <sel>` restores
- [x] TUI refreshes conversation ID and token count
- [x] `/help` documents `/resume`
- [x] ACP hides `/resume`

## DoD

- Unit tests for list/select/resume
- TUI command test covering list + restore
- `make lint` clean, race-clean on touched packages
