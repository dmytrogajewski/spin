# 1) Goals & Scope

* Single-window **terminal UI** that renders a **timeline of blocks** (EXECUTE, PLAN, READ, GREP, APPLY PATCH, SUMMARY, TESTING).
* Inline **diff/code previews**, **command transcripts**, and **system notices** (exit code, timeout, impact).
* **Command input line** with modes (Auto/Manual) and a small **statusline**.
* **Keyboard-first** interaction with discoverable keymap.
* **Zero external GUI** dependencies; pure terminal (tcell/bubbletea/blessed/ncurses-class).

# 2) Screen Layout

```
┌───────────────────────────────────────────────────────────────────────────┐
│ Timeline (scrollable, block-based feed)                                   │
│ ────────────────────────────────────────────────────────────────────────  │
│ [BLOCK] TYPE TAG + Title/Meta                                             │
│   Body (code, logs, diff, lists)                                          │
│   Footer (Exit code • Output lines • Duration • Impact • Timeout)         │
│ …                                                                          │
│ ────────────────────────────────────────────────────────────────────────  │
│ Summary / Tips row (optional, collapsible)                                │
├───────────────────────────────────────────────────────────────────────────┤
│ Input bar:  Mode • Trust • Hint • Prompt                                  │
│ > _                                                                          │
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
* **Histories**:

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

# 13) Config (YAML)

```yaml
ui:
  theme: "dark"
  wrap: true
  show_line_numbers: true
  unicode_lite: false

timeline:
  max_blocks: 500
  auto_compress_after: 200
  default_fold:
    EXECUTE: expanded
    READ: collapsed
    GREP: collapsed
    APPLY_PATCH: expanded
    SUMMARY: expanded
    PLAN: expanded
    TESTING: expanded
    NOTICE: expanded
    ERROR: expanded

input:
  mode: "auto"           # auto|manual
  trust: "allow_all"     # allow_none|confirm_risky|allow_all
  history_size: 2000
```

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

# 15) Testing the TUI (not app logic)

* **Golden snapshots** for block rendering (ANSI stripped).
* **Keymap tests**: simulate sequences, assert focused block & fold state.
* **Performance tests**: inject 100k lines, measure frame time budget.
* **Color downgrade** tests for 8-color terminals.

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

    * `EXECUTE=blue`, `PLAN=magenta`, `READ=cyan`, `GREP=yellow`, `APPLY_PATCH=green`, `SUMMARY=cyan`, `TESTING=blue`, `NOTICE=muted`, `ERROR=red`.
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
▐EXECUTE▌  go test -race ./...  (cwd: ./, timeout: 600s)                        [impact: medium]
▐PLAN▌     Updated: 3 total (0 pending, 0 in progress, 3 completed)
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

* Example: `"  42 │ - assert.False(t, i.focused)"`.

## 15.3 Checklist item

```
indent s2 | marker(✓/•/◦) | space | text…
```

# 16) Performance constraints tied to UI

* Target 60 FPS at `W=160ch, H=45` while appending 1k lines/sec to a streaming EXECUTE block (virtualized).
* Rendering budget per frame ≤ 8ms; diff rendering done per hunk with lazy lexing.
* Avoid full-screen redraws; use damage-based updates: update only changed rows.

# 17) Config keys (UI-only)

```yaml
ui:
  theme: dark            # dark|light|high-contrast
  gutters:
    code_min_width: 3
    code_max_width: 6
  wrap_default: false
  show_scrollbar: true
  show_line_numbers: true
  truncate_header_strategy: mid  # start|mid|end
  chips_style: bracketed         # bracketed|square
  collapse_threshold_lines: 200
```

# 18) QA checklist (visual)

* Headers align at same x-position across all blocks (TagPill widths vary; left text start is constant after pill + s2).
* Accent bar perfectly continuous from block top to bottom, not overlapping neighbors.
* Gutter digits right-aligned; `│` always vertical with no gaps on wrapped lines.
* Chips never wrap mid-token; if wrap needed, push whole chip to next line.
* Palette downgrade to 8-color verified; red/green maintain contrast on black/white.
