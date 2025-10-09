# 1) Goals & Scope

* Single-window **terminal UI** that renders a **timeline of blocks** (EXECUTE, PLAN, READ, GREP, APPLY PATCH, SUMMARY, TESTING).
* Inline **diff/code previews**, **command transcripts**, and **system notices** (exit code, timeout, impact).
* **Command input line** with modes and a small **statusline**.
* **Keyboard-first** interaction with discoverable keymap.
* **Zero external GUI** dependencies; pure terminal (tcell/bubbletea/blessed/ncurses-class).

# 2) Screen Layout

```
┌───────────────────────────────────────────────────────────────────────────┐
│ Timeline (scrollable, block-based feed)                                   │
│───────────────────────────────────────────────────────────────────────────│
│ [BLOCK] TYPE TAG + Title/Meta                                             │
│   Body (code, logs, diff, lists)                                          │
│   Footer (Exit code • Output lines • Duration • Impact • Timeout)         │
│ …                                                                         │
│───────────────────────────────────────────────────────────────────────────│
│ Summary / Tips row (optional, collapsible)                                │
├───────────────────────────────────────────────────────────────────────────┤
│ Input bar:  Mode • Trust • Hint • Prompt                                  │
│ > _                                                                       │
└────────────────────── Statusline (right: profile) ────────────────────────┘
```

## 2.1 Regions

* **Timeline** — vertical stack of **Blocks**; virtualized for performance.
* **Input bar** — single-line prompt with mode indicator and hints.
* **Statusline** — left: project/branch; middle: perf/pos; right: profile (“GPT-5-Codex (Auto)”), context (“IDE”), clock.

# 3) Block System

## 3.1 Block Types (visual + metadata)

* `EXECUTE` — runs a shell command.
  * Header: `EXECUTE (cmd, cwd, timeout: 120s, impact: medium)`
  * Body: stdout/stderr transcript (collapsible).
  * Footer: `↳ Exit code: 0 • Output: 19 lines • Duration: 4.2s`

* `PLAN` — planned steps list.
  * Header: `PLAN Updated: 3 total (0 pending, 0 in progress, 3 completed)`
  * Body: checklist bullets (strikethrough for done, dim for skipped).

* `READ` — file read preview.
  * Header: `READ (path, offset: X, limit: Y)`
  * Body: code snippet with line numbers.

* `GREP` — search result in content mode.
  * Header: `GREP ("pattern", content mode, context N)`
  * Body: matches with filename:line anchors.

* `APPLY PATCH` — patch result.
  * Header: `APPLY PATCH (file)`
  * Body: unified diff with red/green lines; per-hunk stats.
  * Footer: `✓ Succeeded. File edited. (+1 added)` or error details.

* `SUMMARY` — human-readable changeset notes.
  * Header: `Summary`
  * Body: paragraphs + bullet list.

* `TESTING` — rubric of how to run tests.
  * Header: `Testing`
  * Body: commands list; failing suites in yellow/red with short reasons.

* `NOTICE` — system messages (“Conversation history compressed…”).
* `ERROR` — prominent red block with cause + first lines of stderr.

## 3.2 Block Anatomy (data model)

```json
{
  "id": "blk_1738950123_07",
  "type": "EXECUTE|PLAN|READ|GREP|APPLY_PATCH|SUMMARY|TESTING|NOTICE|ERROR",
  "title": "optional concise title",
  "meta": {
    "cwd": "/home/user/project",
    "timeout_sec": 600,
    "impact": "low|medium|high",
    "exit_code": 0,
    "duration_ms": 4200,
    "lines_out": 54,
    "file": "internal/tui/ui/input.go",
    "offset": 0,
    "limit": 120
  },
  "body": "renderable text or diff",
  "fold_state": "expanded|collapsed",
  "severity": "info|warn|error"
}
```

## 3.3 Rendering Rules

* **Headers**: left tag in a colored pill (`EXECUTE`, `PLAN`, …), then concise meta string.
* **Diffs**: unified patch, `-` red dim, `+` green bright, context gray; gutters with line numbers.
* **Code**: monospace, line numbers; basic lexing for Go/JSON/YAML/Markdown (optional).
* **Bullets**: normal •, completed ~strikethrough~, pending hollow ◦.
* **Long bodies**: height-limited with `[…]` and `Enter` to expand.
* **Errors**: first line bold; `F1` opens full transcript.

# 4) Input Bar

## 4.1 Modes & Indicators

* Mode chip: `Auto (High) – allow all commands` or `Manual – confirm before run`.
* Small hint tail: `shift+tab cycles` / current completion tip.
* Prompt: plain line starting with `>`.

## 4.2 Behaviors

* **Inline completion** (tab) across:
  * recent commands, file paths, block types, repo symbols.
*
 **Histories**:
  * `↑/↓` traverse; `Ctrl-R` reverse-search.

* **Trustedness toggle**:
  * `Ctrl-T` cycles: `Allow none` → `Confirm risky` → `Allow all`.
  * Visual: green/yellow/red chip next to mode.

# 5) Navigation & Interaction

## 5.1 Scrolling & Focus

* Timeline scroll: `PgUp/PgDn`, `g/G` top/bottom, `[` `]` prev/next block.
* Collapse/expand block: `Enter`.
* Expand all / collapse all: `zR` / `zM`.

## 5.2 Selection & Actions

* Open file at line: `o` (on READ/GREP/diff lines).
* Copy block body: `y`.
* Save block as file: `S`.
* Rerun EXECUTE: `r`.
* Toggle wrap: `w`.
* Filter timeline: `/` (live filter by `type:file:exit:impact`), `Esc` clear.

## 5.3 Command Palette

* `Ctrl-P` opens overlay:

  * `Run…`, `Search in repo…`, `Open recent file…`, `New plan…`, `Toggle mode…`, `Change theme…`
  * Fuzzy search with preview.

# 6) Statusline

* Left: ` project @ branch • pathDepth • dirty?`
* Middle: `blocks N • mem/cpu (optional) • scroll %`
* Right: `profile name • mode • time`
* Colors change on error in view.

# 7) Theming & Accessibility

* Themes: Dark (default), Light, High-contrast.
* Color budget works on 8-color terminals (degrade gracefully).
* Configurable **contrast** and **bold** usage; never rely on color alone (use symbols `+/-/•`).
* Optional **unicode-lite** mode for restricted fonts.

# 8) Lists, Tables & Chips

* **Meta chips**: `Exit code: 0`, `timeout: 600s`, `impact: medium`, `Output: 54 lines`.
* **Plan list**: numbered with progress counts: `Updated: 3 total (0 pending, 0 in progress, 3 completed)`.

# 9) File/Code UX

* Open file preview window (popup) with minimal editor (readonly) for context.
* Jump anchors on filename:line tokens (e.g., `input.go:42`) across READ/GREP/diff.
* From diff, `o` opens file view positioned at hunk.

# 10) History Management

* Automatic **compression** message when timeline large:

  * Renders a `NOTICE` block like “Conversation history has been compressed — previous messages may be summarized.”
* `H` opens archived session index; `Enter` to expand a previous chunk in place.

# 11) Errors & Diagnostics

* On non-zero exit: block turns red, footer shows exit code; `?` opens tail of stderr.
* Timeouts visually flagged; `t` to increase timeout and retry.
* If an APPLY PATCH fails (conflict), show a 3-way hint with suggested manual command.

# 12) Performance Requirements

* Must smoothly render **10k+ lines** via viewport virtualization.
* Stream large outputs (incremental append) for EXECUTE.
* Keep last **N** blocks fully in memory; older ones summarized with lazy expansion.

# 14) Keymap (defaults)

```
Global:  q quit • ? help • : command palette • / filter • Esc clear
Scroll:  ↑/↓ line • PgUp/PgDn page • g/G top/bottom • [ ] prev/next block
Blocks:  Enter toggle • zR expand all • zM collapse all • y copy • S save
Open:    o open file/anchor • w toggle wrap • r rerun EXECUTE • t bump timeout
Search:  / filter • n/N next/prev match • Ctrl-F fuzzy file search
Input:   Tab complete • Shift-Tab prev completion • ↑/↓ history • Ctrl-R search
Modes:   Ctrl-T trust cycle • Alt-M mode cycle (auto/manual)
```

# 16) Minimal Interface (what the TUI expects from backend)

*(Not agent logic, just the feed contract into the UI)*

```json
{
  "event": "append_block|update_block|append_stream|set_status|set_prompt_hint",
  "payload": { /* Block model, stream chunk, or status text */ }
}
```

* `append_stream` targets a block id and appends text progressively.
* TUI emits **user intents** only as:

```json
{ "ui_event": "run_command|open_file|toggle_fold|rerun_block|set_mode|set_trust|filter|copy|save", "args": {...} }
```

# 17) Visual Details Mirroring Your Screens

* **Tag pills**: `EXECUTE/PLAN/READ/GREP/APPLY PATCH/SUMMARY/TESTING` with consistent colors.
* **Footers** exactly like:
  `↳ Exit code: 0. Output: 54 lines.`
  `timeout: 600s, impact: medium`
* **“Updated: 3 total (0 pending, 0 in progress, 3 completed)”** line format preserved.
* **Unified diffs**: leading gutters, red `-`, green `+`, context dim.
* **Notice** line: “Conversation history has been compressed — previous messages may be summarized.”
Absolutely. Here’s a **pixel-level (terminal-cell-level) UI spec** for the TUI — focused purely on **elements, spacing, typography, colors, states, and layout math**. All measures are in **terminal cells**: width in `ch` (characters), height in `rows`.

# 0) Design tokens

* **Base grid**

  * Unit `u = 1ch` horizontally, `1 row` vertically.
  * Spacing scale: `s0=0u, s1=1u, s2=2u, s3=3u, s4=4u, s6=6u, s8=8u, s12=12u`.
* **Typography (monospace)**

  * Body: normal; **line height = 1 row**.
  * Heading weight: bold.
  * Secondary/meta: dim.
  * Emphasis: bold + bright.
* **Colors**
  * Named tokens (map to 256-color, degrade to 8-color):
    * `fg`: default foreground; `bg`: default background.
    * Accents: `blue`, `green`, `yellow`, `red`, `magenta`, `cyan`.
    * Neutrals: `muted` (dim fg), `border` (dim gray), `shadow` (very dim).

  * Tag color map:
    * `EXECUTE=blue`, `PLAN=magenta`, `READ=cyan`, `GREP=yellow`, `APPLY_PATCH=green`, `SUMMARY=cyan`, `TESTING=blue`, `NOTICE=muted`, `ERROR=red`

* **Dividers & borders**
  * Horizontal rule: `─` full width in `border`.
  * Box corners: `┌ ┐ └ ┘`; vertical separator `│` when needed.

* **Chips**
  * Bracketed style: `⟦ label ⟧` with label color = accent, bracket color = `border`.
  * Compact style (for meta): `[key: value]` in `muted` key, normal value.

# 1) Global screen layout

```
Top padding: s1
┌ Timeline (viewport, scrollable) ────────────────────────────────────────┐
│   Blocks stack (vertical, v-spacing = s3)                               │
└─────────────────────────────────────────────────────────────────────────┘
Optional summary/tips row (1–3 rows, muted, collapsible)
Input bar (2 rows total)
Statusline (1 row)
Bottom padding: s1
```

* **Viewport margins**: left/right margin `s2`; content max width = `W - 2*s2`.
* **Block spacing**:

  * Between blocks: `s3` (one blank row plus header’s bottom padding).
  * Inside block: see each component below.

# 2) Block anatomy

A **Block** is a vertical card with: Header → Body → Footer. No outer border by default; a subtle left **accent bar** shows type and state.

```
Accent bar (1ch wide)
│
│  [Header line(s)]
│  [Body: code/log/diff/list]
│  [Footer line(s)]
```

* **Accent bar**

  * Width: `1ch`, color = tag color.
  * Top/bottom cap: none; continuous bar spanning the block height.
* **Block padding**

  * Left inner padding from accent bar: `s2`.
  * Right padding: `s2`.
  * Top: `s1` when block first in viewport; otherwise 0 (spacing handled between blocks).
  * Bottom: `s1`.

## 2.1 Header

**Header row height:** 1 row (can wrap to 2 if needed).
**Layout:**

```
[ TagPill ] space s2 [ Title/Meta string truncated mid ]  [ right-aligned meta ]
```

* **TagPill**

  * Shape: `▐EXECUTE▌` (leading `▐` and trailing `▌` in tag color; text bold).
  * Min width = length(label)+2; no internal padding.
* **Title/Meta (left)**

  * Primary text bold; secondary info dim in parentheses.
  * Truncation: **mid-ellipsize** `…` if exceeds available width; preserve beginning and end (~60/40 split).
* **Right meta**

  * Small chips: e.g., `[timeout: 600s] [impact: medium]`.
  * Right-aligned to content area; minimum gap to left text `s3`. If not enough room, right meta collapses to first chip only; rest move to footer.

**Header paddings:** top `0`, bottom `s1`.

**Examples:**

```
EXECUTE▂  go test -race ./...  (cwd: ./, timeout: 600s)                        [impact: medium]
PLAN▂     Updated: 3 total (0 pending, 0 in progress, 3 completed)
```

## 2.2 Body types

### A) Transcript (logs/stdout)

* Font: normal; **wrap = off** by default; horizontal scroll enabled.
* Left gutter: none.
* Row spacing: 0.
* Streaming: append rows at bottom; keep viewport pinned to bottom when focused unless user scrolled up.
* Collapsible: default **expanded** for ≤ 200 lines; for > 200 lines: show first 80 + `[…]` + last 40; toggle with `Enter`.
* Overflow hint row (muted): `… output truncated (540 lines hidden) — press Enter to expand`.

**Paddings:** top `s1`, bottom `s1`.

### B) Code preview (READ)

* Gutter width: dynamic 3–6ch based on max line number in view; format `│NNN`.
* Syntax highlighting (basic): keywords bold, comments dim, strings cyan.
* Soft wrap: off.
* File header subline (muted): `internal/tui/ui/input.go  offset:0  limit:120`.
* Jump anchors: filename:line sequences are underlined when hovered/focused.

**Paddings:** top `s1`, bottom `s1`.

### C) Diff (APPLY PATCH)

* Unified diff; **hunk header**:

  * `@@ -a,b +c,d @@` in `border`.
* Gutter:

  * 2 gutters `a`/`b` merged into `-` / `+` indicators.
* Line colors:

  * Removed: red; Added: green; Context: muted.
  * Leading markers `-` / `+` colored; remainder slightly dimmer (so markers pop).
* Intraline highlight (optional): inverse for changed spans.
* Hunk spacing: `s1` between hunks.
* Per-hunk stats line (muted, right-aligned): `(+12/-3)`.

**Paddings:** top `s1`, bottom `s1`.

### D) List/Checklist (PLAN, SUMMARY, TESTING)

* Bullet style:

  * Pending: `•` normal; Done: `✓` green dim; Skipped: `◦` dim.
  * Strikethrough for “done” text (use `~text~` emulation if terminal lacks strikethrough).
* Indent:

  * Bullet col at `s2` from content left; text aligns at `s4`.
* Vertical spacing:

  * `s0` between list items; `s1` before nested group.

**Paddings:** top `s1`, bottom `s1`.

## 2.3 Footer

One compact line; multiple lines if chips wrap.

* Left: outcome chips (`[exit: 0] [out: 54 lines] [dur: 4.2s]`).
* Right: state labels (e.g., `[cached] [streaming] [compressed]`) in muted.
* Divider above footer: none; rely on vertical spacing.

**Paddings:** top `s1`, bottom `0`.

## 2.4 Block states

* **Focused block**: left accent bar bright + header text bold; a thin top rule `─` in `shadow`.
* **Collapsed**: show only header + a small badge `⟦ collapsed ⟧` (muted) at right.
* **Error**: accent bar red; header and footer tinted red; prepend `●` red in header.
* **Success**: subtle green tick `✓` before footer chips.

# 3) Timeline mechanics

* **Virtualization**: only render visible blocks + `±5` off-screen; replace distant blocks with a **placeholder** line:

  * `… 12 blocks hidden above — press H to expand history …`
* **Inter-block separator**: a single `─` rule spanning content width when transitioning across different block types; same type → no rule.
* **Type filter chips** (when active): a slim row above first block —

  * `Filter: [EXECUTE] [PLAN] [READ] …  (Esc to clear)`.

# 4) Input bar (2 rows)

```
Row 1 (controls):  [ModeChip] space s1 [TrustChip] space s2 [Hint]
Row 2 (prompt):    > [user text…][ghost completion]
```

* **ModeChip**:

  * Shapes:

    * Auto: `⟦ Auto ⟧` (green border).
    * Manual: `⟦ Manual ⟧` (yellow border).
* **TrustChip**:

  * `Allow all` green; `Confirm risky` yellow; `Allow none` red.
* **Hint**: muted, truncated end; examples: `shift+tab cycles • Ctrl-P palette`.
* **Prompt**:

  * `>` plus one space; cursor follows shell style.
  * Ghost completion: dim italic (if supported), otherwise dim.

**Padding:** left `s2`, right `s2`.

# 5) Statusline (1 row)

* **Layout**: three zones (flex):

  * Left: `project@branch • path • dirty*`
  * Center: `blocks:N • scroll:55%`
  * Right: `profile • mode • hh:mm`
* Background can invert (slightly darker) to demarcate.
* Truncation rules:

  * Right zone highest priority; left truncates mid; center truncates end.

**Padding:** left/right `s2`.

# 6) Overlays

## 6.1 Command Palette

* Width: `min(80ch, W - 2*s4)`; centered; height: up to `H*0.6`.
* Frame: rounded box (`╭─╮` style) with `border`.
* Sections: input line (1 row), results list (10–16 rows).
* Result item:

  * Left icon (1ch), title, muted path/meta right-aligned.
  * Selected row inverted.

**Inner spacing:** frame padding `s2`; item v-spacing `s0`.

## 6.2 File Preview Popup

* Width: `min(100ch, W - 2*s4)`; height: `min(30 rows, H - 6)`.
* Header bar with filename; close hint `[Esc]`.
* Body is Code preview with gutter; soft shadow (draw with `shadow` chars).

# 7) Controls affordances (glyphs)

* Collapse/expand chevron in header tail:

  * Collapsed: `▸`; Expanded: `▾`. Color `muted`.
* Scrollbar (right 1ch lane, optional):

  * Use `█` blocks to indicate viewport; muted color.

# 8) Interaction affordances

* **Hover/focus** in terminal = **focus row** (keyboard). Focused block header gets a subtle right bracket `〉` at far right (muted).
* **Clickable anchors** (filename:line) underlined on focus; press `o` to open.

# 9) Theming details

Provide two themes:

## DARK (default)

* `bg=#0b0e12`, `fg=#dde3ea`, `muted=#9aa4b2`, `border=#2d3640`, `shadow=#1a212a`
* Accents:

  * `blue=#5aa6ff`, `green=#57d98d`, `yellow=#f5c156`, `red=#ff6b6b`, `magenta=#d08bff`, `cyan=#7adcf3`

## LIGHT

* `bg=#f7f9fc`, `fg=#1e2a35`, `muted=#6b7580`, `border=#cfd6de`, `shadow=#e9eef3`
* Accents:

  * `blue=#2a7fff`, `green=#0dbf6f`, `yellow=#c28a00`, `red=#d23a3a`, `magenta=#8e4dff`, `cyan=#1ca8c7`

**8-color fallback map**

* `fg→white`, `bg→black`, `muted→brightBlack`, `border→brightBlack`
* `blue→blue`, `green→green`, `yellow→yellow`, `red→red`, `magenta→magenta`, `cyan→cyan`.

# 10) Truncation & wrapping rules

* **Headers**: mid-ellipsize at component level:

  * left text gets `…` if `left+gap+right > width`.
* **Body**:

  * default no wrap; horizontal scroll indicator at right margin: `↔` dim.
  * optional wrap mode toggled per block (`w`): soft wrap at viewport width; continue lines prefixed with `⋮` (muted) occupying 2ch (glyph + space).

# 11) Focus & navigation indicators

* Current block index displayed transiently in statusline when navigating: `Block 7/23`.
* Jump between **types**: hold `Alt` + `[ / ]` → next/prev of same type; temporary overlay shows `PLAN (4/6)`.
* When collapsing a block, cursor remains; when expanding, keep prior scroll offset.

# 12) Keyboard discoverability

* `?` opens a two-column cheat sheet overlay:

  * Left: navigation keys; Right: block actions.
* Overlay width `min(90ch, W-8)`; each column min `30ch`; v-spacing `s1`.

# 13) Accessibility & clarity

* Never convey state by color alone:
  * Errors also show `●` and “error” text in header.
  * Success shows `✓`.

* High-contrast mode:
  * Increase accent brightness by 20%; replace muted with normal fg.
* Do not use braille or half-block characters; stick to ASCII + box-drawing + a few bullets/chevrons.

# 14) Microcopy (exact strings)

* Compression notice: `Conversation history has been compressed — previous messages may be summarized.`
* Truncation hint: `… output truncated (N lines hidden) — press Enter to expand`
* Footer chips:

  * `exit: N`, `out: N lines`, `dur: X.Ys`, `timeout: Zs`, `impact: low|medium|high`, `cached`, `streaming`, `compressed`.

# 15) Element blueprints (ASCII with spacing)

## 15.1 Header line blueprint (max width = `C`)

```
[1ch bar]|s2|TagPill|s2|LeftTitle(≤L)|s3|RightMeta(≤R)
where: L = C - (1 + 2 + TagPillW + 2 + 3 + R)
```

## 15.2 Diff line blueprint

```
gutter(4ch) | space | marker(+/-/ ) | space | code…
```

## 15.3 Checklist item

```
indent s2 | marker(✓/•/◦) | space | text…
```

Got it — just the **TUI**. Below is a focused, SOLID/DRY/KISS architecture for a **native-scrollback chat TUI** (Factory Droid feel): append-only transcript + single-line prompt redraw, zero full-screen repainting.

---

# 0. Design Targets

* **No alt screen** — keep terminal scrollback intact.
* **Append-only history** — messages go to stdout as real lines.
* **Minimal redraw** — only the prompt/status line is repainted.
* **Stream-friendly** — incremental chunks print without reflow.
* **Composable** — UI is swappable (PureTTY vs BubbleteaHybrid).
* **Testable** — clear ports, deterministic state machine.

---

# 1. UI Package Layout

```
/internal/ui/
  ports/
    ui.go                // UI port (what the rest of app would call)
  term/
    tty.go               // terminal mode, signals, win size
    keyboard.go          // key events (line-editing)
    ansi.go              // escape helpers (clear line, cursor)
  prompt/
    model.go             // prompt state (buffer, cursor, history)
    renderer.go          // draw one-line prompt/status
    input_loop.go        // read/edit/submit
  output/
    printer.go           // append-only printing (stdout), stream coalescing
  adapters/
    puretty.go           // uses term/* + prompt/* + output/*
    bubbletea_hybrid.go  // optional adapter using tea.Println + one-line View
  testkit/
    fake_writer.go       // captures stdout for tests
    fake_keyboard.go     // scripted key sequences
```

**Dependency rule:** `adapters/*` depends on term/prompt/output; nothing depends on adapters.

---

# 2. UI Port (what the rest of the app would see)

```go
// /internal/ui/ports/ui.go
package ports

import "context"

type UI interface {
  // Lifecycle
  Run(ctx context.Context) error   // blocks; owns terminal mode; returns when quit
  Stop() error                     // restore terminal, show cursor, etc.

  // Output (append-only)
  PrintLine(line string)           // prints + newline, stays in scrollback
  PrintChunks(chunks <-chan string) // prints streaming chunks as they arrive

  // Prompt control
  PutSystemNotice(text string)     // prints a system line above prompt
  SetStatus(text string)           // transient right/inline status on prompt
  RequestInput(prompt string) (<-chan string, error) // emits submitted lines
}
```

Notes:

* `PrintLine` and `PrintChunks` **never** clear the viewport.
* `SetStatus` only redraws the **current** prompt line.
* `RequestInput` yields submitted lines; editing stays inside UI.

---

# 3. Terminal Primitives (term/*)

### 3.1 TTY control (enter/exit “cooked+raw-ish” mode)

```go
// tty.go
type TTY struct {
  inFD, outFD int
  origState   *term.State
  width, height int
}

func (t *TTY) Enter() error      // enable raw (or cbreak) mode, hide cursor
func (t *TTY) Exit() error       // restore mode, show cursor
func (t *TTY) Size() (w,h int)   // read winsize (SIGWINCH updates cached values)
func (t *TTY) OnResize(cb func(w,h int))
```

* Use `golang.org/x/term` for raw mode and size.
* Install SIGWINCH handler to recalc wrapping for the prompt only.

### 3.2 Keyboard events (line editing without full TUI)

```go
// keyboard.go
type KeyKind int
const (
  KeyRune KeyKind = iota
  KeyEnter
  KeyBackspace
  KeyDelete
  KeyLeft
  KeyRight
  KeyUp
  KeyDown
  KeyCtrlC
  KeyCtrlD
  KeyCtrlU // kill line left
  KeyCtrlK // kill line right
  KeyCtrlW // delete word left
  KeyPasteStart // optional bracketed paste
  KeyPasteEnd
)

type KeyEvent struct {
  Kind KeyKind
  Rune rune // if Kind==KeyRune
  Raw  []byte
}

func ReadKeys(ctx context.Context, r io.Reader, out chan<- KeyEvent) error
```

* Support **bracketed paste** to treat pasted blocks as literal text.
* Translate ESC sequences to navigation keys; keep it minimal.

### 3.3 ANSI helpers (no alt screen)

```go
// ansi.go
const (
  ClearLine     = "\x1b[2K"
  Carriage      = "\r"
  HideCursor    = "\x1b[?25l"
  ShowCursor    = "\x1b[?25h"
  SaveCursor    = "\x1b[s"
  RestoreCursor = "\x1b[u"
)

func MoveCursorToCol(col int) string  // "\x1b[<n>G"
```

---

# 4. Prompt Subsystem (prompt/*)

### 4.1 Prompt model (single source of truth)

```go
// model.go
type Buffer struct {
  runes  []rune
  cursor int // index in runes
}

type History struct {
  items []string
  pos   int // -1 = at "new"
}

type Model struct {
  Prefix      string // e.g. "> "
  StatusRight string // transient status (rendered right-aligned if fits)
  Width       int
  Buf         Buffer
  Hist        History
  multiline   bool // if enabled, submit on Enter with e.g. Ctrl+Enter gating
}

func (m *Model) Insert(r rune)
func (m *Model) Backspace()
func (m *Model) Delete()
func (m *Model) MoveLeft()
func (m *Model) MoveRight()
func (m *Model) PrevHistory()
func (m *Model) NextHistory()
func (m *Model) ClearLineLeft()
func (m *Model) ClearLineRight()
func (m *Model) ClearAll()
func (m *Model) Submit() string // returns string, pushes to history
```

### 4.2 Prompt renderer (one line, wrapped if needed)

```go
// renderer.go
type Renderer struct {
  out io.Writer
}

func (r *Renderer) Redraw(m *Model) {
  // 1) \r + ClearLine
  // 2) write Prefix + buffer content
  // 3) compute visible cursor column (account for wide chars)
  // 4) reposition cursor to correct col
  // 5) optionally draw inline/right-aligned status (truncate if needed)
}
```

* Use `rivo/uniseg` to measure grapheme width; avoid broken cursor math.
* Never touch prior lines; only current line.

### 4.3 Input loop (edit → submit)

```go
// input_loop.go
type Loop struct {
  tty   *term.TTY
  rend  *Renderer
  model *Model
  keys  <-chan KeyEvent
  out   chan string // submitted lines
}

func (l *Loop) Run(ctx context.Context) <-chan string {
  go func() {
    for {
      select {
      case ev := <-l.keys:
        // mutate model based on ev
        // Enter → submit to l.out
        l.rend.Redraw(l.model)
      case <-ctx.Done():
        close(l.out); return
      }
    }
  }()
  return l.out
}
```

---

# 5. Output (append-only printer)

```go
// output/printer.go
type Printer struct {
  out io.Writer
  // Optional: small coalescing buffer for streaming (flush by timer / newline)
}

func (p *Printer) PrintLine(s string) {
  fmt.Fprintln(p.out, s) // real newline → scrollback
}

func (p *Printer) PrintChunks(ch <-chan string, flushEvery time.Duration) {
  // Append chunks immediately; if you want cleaner wrapping, buffer until newline
  t := time.NewTicker(flushEvery)
  var buf strings.Builder
  for {
    select {
    case c, ok := <-ch:
      if !ok {
        if buf.Len() > 0 { fmt.Fprint(p.out, buf.String()) }
        return
      }
      buf.WriteString(c)
      // fast-path: flush on newline to avoid prompt collision
      if strings.Contains(c, "\n") { fmt.Fprint(p.out, buf.String()); buf.Reset() }
    case <-t.C:
      if buf.Len() > 0 { fmt.Fprint(p.out, buf.String()); buf.Reset() }
    }
  }
}
```

**Important:** printing chunks may “push” the prompt line down. The prompt renderer must **repaint after every output** event. Strategy:

* Before any `Print*`, emit `ansi.SaveCursor`.
* After printing, call `Renderer.Redraw(model)` or write `ansi.RestoreCursor` + `Renderer.Redraw`.
* Keep it minimal—just one line redraw.

---

# 6. Adapters

## 6.1 PureTTY (recommended)

Responsibilities:

* Enter/exit raw mode, hide/show cursor.
* Start `ReadKeys()` → feed `prompt.Loop`.
* Expose `UI` port by composing:

  * `Printer.PrintLine / PrintChunks`
  * `Renderer.Redraw` for status/prompt
  * `RequestInput()` returns the prompt loop’s submit channel.

```go
// adapters/puretty.go (sketch)
type PureTTY struct {
  tty   *term.TTY
  rend  *prompt.Renderer
  model *prompt.Model
  keys  chan KeyEvent
  prn   *output.Printer
}

func (u *PureTTY) Run(ctx context.Context) error {
  if err := u.tty.Enter(); err != nil { return err }
  defer u.tty.Exit()
  fmt.Fprint(os.Stdout, ansi.HideCursor)
  defer fmt.Fprint(os.Stdout, ansi.ShowCursor)

  // start keys, prompt loop
  inputs := u.promptLoop.Run(ctx)

  // main event pump — minimal: nothing to repaint except prompt on changes
  for {
    select {
    case line, ok := <-inputs:
      if !ok { return nil }
      u.prn.PrintLine(u.model.Prefix + line) // echo user line as transcript
      // upstream will call PrintChunks/PrintLine for bot; we just keep prompt fresh
      u.rend.Redraw(u.model)
    case <-ctx.Done():
      return ctx.Err()
    }
  }
}
```

**Resize:** on SIGWINCH → update `model.Width` → `Renderer.Redraw(model)`.

## 6.2 BubbleteaHybrid (optional)

* Run Bubbletea with `tea.WithAltScreen(false)`.
* Use `tea.Println()` for every history line (this appends to real scrollback).
* `View()` returns a **single prompt line** only.
* All key handling mirrors `prompt.Model` to keep behavior consistent.

This adapter is a convenience if you want Bubbletea’s message/update model while preserving native scrollback.

---

# 7. Prompt/Output Coordination (race-free redraw)

**Rule:** Any output to stdout must be followed by **prompt redraw**. Two safe options:

1. **Centralized writer**

   * Wrap `io.Writer` with a mutex.
   * `Printer` and `Renderer` share it.
   * `Printer` does: `SaveCursor → write → RestoreCursor → Renderer.Redraw(model)` within the same lock.

2. **Event bus (lightweight)**

   * On any `Print*`, post `OutputFlushed` event.
   * Prompt loop listens and triggers `Renderer.Redraw`.
   * Still keep a write mutex.

KISS advice: start with **centralized writer + lock**.

---

# 8. Streaming without flicker

* Default behavior: **immediate** chunk print (no special effects), then `Redraw(prompt)`.
* Optional nicety: coalesce chunks for N ms to reduce redraw frequency.
* Avoid carriage returns for reflow; keep transcript immutable.

---

# 9. Status line

* Render within the same prompt line (suffix), truncated if needed.
* For right alignment: compute `width - len(prefix+buffer)` and place status if space allows; else fall back to a compact indicator (e.g., “[…]”).

---

# 10. Testing Strategy (only UI)

* **Prompt model unit tests**: editing ops, history nav, word delete, submit.
* **Renderer golden tests**: for given model state & width → expected ANSI sequence (use `testkit.FakeWriter` to capture bytes).
* **Output printer tests**: streaming coalescer, newline flush.
* **Coordination tests**: interleave `PrintLine` and key events → ensure final byte stream is `[HideCursor][user lines with \n][assistant chunks][prompt redraw][ShowCursor]`.
* **Resize tests**: simulate `Size()` change → assert prompt redraw with correct wrapping.
* **Paste tests**: bracketed paste on/off; large payloads shouldn’t lag prompt beyond redraw cadence.

---

# 11. Minimal Public API (for the rest of your codebase)

```go
type Settings struct {
  PromptPrefix string
  Multiline    bool
}

func NewPureTTY(settings Settings) (ports.UI, error)
func NewBubbleteaHybrid(settings Settings) (ports.UI, error)
```

---

# 12. Factory Droid Parity Checklist

* [ ] **No alt screen** (scrollback preserved).
* [ ] **Append-only** transcript (`PrintLine`, `tea.Println`).
* [ ] **One-line prompt** with ANSI `\r` + `ESC[2K` repaint.
* [ ] **Streaming** prints push the prompt down; prompt is immediately repainted.
* [ ] **Resize aware** prompt; history untouched.
* [ ] **Bracketed paste**; big pastes treated as raw text.
* [ ] **No frame re-rendering**; only the prompt line is touched.

