# JOURNEY-027-prompt-slash-at-paste: Slash skills, @ files, and paste attach

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: none (operator request after agent-harness march)
- Feature: TUI prompt compose — `/` suggestions, `@` file attach, Ctrl-V / paste attach

## 1. Journey

When **Alex is in the spin TUI and wants to start a skill, pin a project file, or drop a copied path** I want **`/` to suggest commands and skills, `@` to suggest project files, and paste / Ctrl-V to attach real files** so I **do not have to wait for the model to guess a skill name or retype a path**.

## 2. CJM

Alex already has a skill catalog (`/skills`) and a `skill` tool the model can call. There is no operator shortcut: typing `/review-pr` is an unknown command. `@foo.go` is plain text. Paste inserts characters only. Other agents complete `/` and `@` in the prompt and attach clipboard files. This journey adds that compose layer on the TUI prompt. It does not change ACP hosts (they already send resource/image blocks).

Assumption: registered slash commands win over skill names (`/help` stays help). Assumption: `@token` attaches only when the path resolves inside the workdir. Assumption: Ctrl-V reads the OS clipboard when the terminal does not already emit bracketed paste; both paths share the same classifier. Assumption: GitNexus is unavailable; impact is grep.

### Phase 1: Slash suggest and invoke

**User Intent:** Start a command or skill without memorizing names.

**Actions:** Type `/`. See filtered commands and skills. Tab completes. Enter submits. `/review-pr look at auth` loads that skill body and sends the remainder as the turn.

**Pain / Risk:** Skill name collides with a command. `/` in the middle of a sentence is treated as a command. Tab steals indent. Unknown `/foo` is silent.

**Success Signal:** `/sk` suggests `/skills`. `/review-pr` with a catalog hit becomes a turn that contains the skill body. Unknown `/foo` still prints the unknown-command line.

### Phase 2: At-file suggest and attach

**User Intent:** Pin a project file to the next turn.

**Actions:** Type `@`. See workdir-relative files (gitignore respected). Tab completes `@path`. Enter injects the file contents into the turn.

**Pain / Risk:** `@user@host` looks like a mention. Paths escape the workdir. Huge files blow the context. Binary files dump garbage.

**Success Signal:** `@cmd/spin` completes a real relative path. Submit includes an attached-file block. Missing or escaped paths stay as plain text.

### Phase 3: Paste and Ctrl-V

**User Intent:** Drop a copied file path or clipboard image into the prompt.

**Actions:** Paste (bracketed) or press Ctrl-V. Existing workdir files become `@rel` tokens. PNG/JPEG bytes write under `.spin/paste/` and insert that `@` path. Other text inserts as text.

**Pain / Risk:** Ctrl-V is swallowed by the terminal (already paste). Clipboard tools missing. Image write escapes the workdir. Multi-line prose is mistaken for paths.

**Success Signal:** Pasting `foo.go` (a real file) inserts `@foo.go`. Pasting a PNG writes a contained paste file. Pasting `hello` stays `hello`.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Skills only via the model tool | 1 | `/name` activate |
| No inline `/` menu | 1 | Tab suggestions |
| Paths typed by hand | 2 | `@` index |
| Paste is text only | 3 | Path/image classifier |

### North Star Summary

Alex types `/` and sees commands plus skills, tabs a skill, optionally adds `@file` mentions, or pastes a path / image. Enter sends one turn that already contains the skill body and attached file text. Unknown slashes and non-files behave as they did before.

### Stressors

1. `/help` is a command even if a skill named help exists.
2. `Use the /api` is not a command (slash not first).
3. `/` alone is not a command; suggestions still list everything.
4. Skill remainder keeps original case (`Auth.go`).
5. `@../etc/passwd` is not attached.
6. File larger than the attach cap is skipped with a note.
7. Binary (NUL) files are skipped with a note.
8. Paste of two existing paths becomes two `@` tokens.
9. Paste of ordinary prose is unchanged.
10. Ctrl-V with empty clipboard is a no-op.
11. Tab with no suggestions inserts nothing (no indent).
12. Up/Down move the suggestion selection, not history, while a list is open.
13. Esc closes the list; history Up works again.
14. Email-like `a@b.com` is not treated as a file unless that path exists.
15. Image magic (PNG/JPEG) writes under `.spin/paste/` inside the workdir.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] `/` shows a list without extra config
- [x] Tab completes the selected row

### Onboarding Clarity
- [x] Welcome names `/`, `@`, and paste
- [x] `/help` documents skill invoke and attach

### Production-Ready Defaults
- [x] Commands win over skills
- [x] Attach stays inside the workdir

### Golden Path Quality
- [x] `/skill rest` prepends the skill body
- [x] `@path` injects file text on submit

### Decision Load
- [x] One Tab target: the highlighted row
- [x] Paste classifier chooses text vs attach

### Progressive Complexity
- [x] Plain prompts unchanged
- [x] `/` `@` and paste only activate on those tokens

### Error Quality
- [x] Unknown slash still says type /help
- [x] Skipped attach names the path and why

### Failure Safety
- [x] Esc dismisses suggestions
- [x] Failed clipboard read does not clear the buffer

### Runtime Transparency
- [x] Suggestion list shows name + detail
- [x] Attached blocks are visible in the submitted prompt

### Debuggability
- [x] Skill block includes name and source
- [x] Paste images land at a visible `@.spin/paste/…` path

### Cross-Surface Consistency
- [x] ACP file/image blocks unchanged
- [x] `/skills` list still works

### Workflow Consistency
- [x] Slash parse still uses the command registry first
- [x] File index uses the existing filesearch scanner

### Change Safety
- [x] Mid-sentence `/api` is not rewritten
- [x] Non-file `@tokens` stay as typed

### Experimentation Safety
- [x] Tests use temp workdirs
- [x] Clipboard reader is injectable

### Interaction Latency
- [x] File index is cached after first `@`
- [x] Suggestion filter is in-memory prefix/contains

### Developer Feedback Speed
- [x] List updates on each key
- [x] Tab result is visible immediately

### Team Scale
- [x] Skill names match the catalog
- [x] Relative paths are workdir-relative

### System Scale
- [x] Suggestion cap keeps the list short
- [x] Attach byte cap keeps turns bounded

### Right Behavior by Default
- [x] No new config keys
- [x] Gitignored files stay out of `@` suggestions

### Anti-Bypass Design
- [x] Path jail rejects `..`
- [x] Image writes stay under `.spin/paste/`

## 4. Tests

### TC-01: token_slash

**Given** buffer `/sk` with cursor at end.
**When** TokenAt runs.
**Then** kind is slash and query is `sk`.

### TC-02: token_at

**Given** buffer `see @cmd/s` with cursor at end.
**When** TokenAt runs.
**Then** kind is file and query is `cmd/s`.

### TC-03: filter_prefix

**Given** items `/skills` and `/mode`.
**When** Filter query `sk`.
**Then** `/skills` is first.

### TC-04: expand_skill

**Given** catalog skill `review-pr` and line `/review-pr Auth.go`.
**When** Expand runs.
**Then** prompt contains the skill body and `Auth.go` (original case). Not a command.

### TC-05: expand_command_wins

**Given** `/help`.
**When** Expand runs.
**Then** IsCommand is true even if a skill named help exists.

### TC-06: attach_file

**Given** workdir file `a.go` and line `look at @a.go`.
**When** Expand runs.
**Then** prompt includes an attached-file block with `a.go` contents.

### TC-07: attach_escape

**Given** line `@../secret`.
**When** Expand runs.
**Then** no attached-file block.

### TC-08: paste_paths

**Given** paste of two existing relative paths.
**When** ClassifyPaste runs.
**Then** insert text is `@p1 @p2`.

### TC-09: paste_text

**Given** paste `hello`.
**When** ClassifyPaste runs.
**Then** insert text is `hello`.

### TC-10: paste_png

**Given** PNG magic bytes.
**When** ClassifyPaste runs.
**Then** a file is written under `.spin/paste/` and insert is `@.spin/paste/…`.

### TC-11: tab_completes

**Given** loop source with `/skills` and buffer `/sk`.
**When** Tab is pressed.
**Then** buffer is `/skills`.

### TC-12: ctrl_v_key

**Given** byte 0x16.
**When** the key parser runs.
**Then** Kind is KeyCtrlV.

## Acceptance Criteria

- `/` suggests registered commands and catalog skills; Tab completes
- `/<skill>` [rest] activates the skill body on the turn; commands still win
- `@` suggests workdir files; submit attaches contained file text
- Bracketed paste and Ctrl-V attach existing paths or clipboard images
- Welcome and `/help` document the three gestures
- `make test` and `make lint` pass

## Traceability
- Roadmap item: operator request (no ROADMAP step)
- Implementation files: `internal/ui/suggest/`, `internal/ui/prompt/loop.go`, `internal/ui/prompt/loop_suggest.go`, `internal/ui/prompt/renderer.go`, `internal/ui/term/keyboard.go`, `internal/ui/adapters/puretty.go`, `cmd/spin/tui.go`, `cmd/spin/tui_compose.go`, `internal/commands/commands.go`, `docs/how-to/agent-skills.md`
- Test files: `internal/ui/suggest/suggest_test.go`, `internal/ui/prompt/loop_test.go`, `internal/ui/term/keyboard_test.go`, `cmd/spin/tui_compose_test.go`, `cmd/spin/tui_welcome_test.go`, `internal/commands/commands_test.go`

## Implementation

Files created:
- `internal/ui/suggest/` — token, filter, expand, paste, clipboard
- `internal/ui/prompt/loop_suggest.go` — Tab / Esc / Ctrl-V
- `cmd/spin/tui_compose.go` — submit expansion
- `specs/journeys/JOURNEY-027-prompt-slash-at-paste.md` — this journey

Files modified:
- `internal/ui/prompt/loop.go`, `renderer.go`, `model.go`
- `internal/ui/term/keyboard.go`
- `internal/ui/adapters/puretty.go`
- `cmd/spin/tui.go`, `internal/commands/commands.go`
- `docs/how-to/agent-skills.md`
