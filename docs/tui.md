# Spin Terminal UI (TUI)

**Version:** 1.0
**Last Updated:** 2025-10-11
**Status:** ✅ Complete (Phases 1-7.2)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Getting Started](#3-getting-started)
4. [Keymap Reference](#4-keymap-reference)
5. [Block Types](#5-block-types)
6. [Advanced Features](#6-advanced-features)
7. [Troubleshooting](#7-troubleshooting)
8. [Examples](#8-examples)
9. [References](#9-references)

---

## 1. Overview

### 1.1 What is the Spin TUI?

The Spin TUI is a **native-scrollback terminal user interface** for interactive coding agent sessions. Unlike traditional full-screen TUIs (like vim, htop), Spin preserves your terminal's native scrollback history—nothing disappears when you exit.

**Key features:**
- **Block-based timeline**: Visual representation of agent actions (commands, diffs, plans, summaries)
- **Streaming output**: Real-time LLM response rendering with smooth coalescing
- **Keyboard-first navigation**: Scroll, filter, collapse, copy—all without a mouse
- **Zero alt-screen**: Works seamlessly in tmux, SSH, and screen sessions
- **Performance**: Handles 100k+ blocks without lag

---

### 1.2 Design Principles

**Factory Droid Philosophy:**
1. **Append-only transcript**: Every interaction adds to the timeline, nothing is erased
2. **Single-line prompt redraw**: Prompt stays at the bottom, redrawn after each output
3. **Zero full-screen repainting**: Terminal scrollback is sacred

**Why this matters:**
- ✅ Review conversation history by scrolling up
- ✅ Copy any output with terminal's native selection
- ✅ Survive SSH drops, tmux detach, screen sessions
- ✅ Integrate with terminal multiplexers and logging

---

### 1.3 Screen Layout

```
┌───────────────────────────────────────────────────────────────────────────┐
│ Timeline (scrollable, block-based feed)                                   │
│───────────────────────────────────────────────────────────────────────────│
│ │ ▐EXECUTE▌  go test ./... (cwd: "./")                     [impact: medium]│
│ │   === RUN   TestFoo                                                     │
│ │   --- PASS: TestFoo (0.00s)                                             │
│ │   PASS                                                                  │
│ │ ✓ [exit: 0] [out: 3 lines] [dur: 0.1s]                                 │
│───────────────────────────────────────────────────────────────────────────│
│ │ ▐APPLY_PATCH▌  (file: main.go)                                         │
│ │   @@ -42,6 +42,7 @@ func main() {                                         │
│ │    func process() {                                                     │
│ │        fmt.Println("processing")                                        │
│ │   +    log.Info("started")                                              │
│ │    }                                                                    │
│ │ ✓ Succeeded. File edited. (+1 added)                                   │
│───────────────────────────────────────────────────────────────────────────│
│ > _                                                                       │
└───────────────────────────────────────────────────────────────────────────┘
```

**Regions:**
- **Timeline**: Vertically scrolling blocks (virtualized for performance)
- **Prompt line**: Single-line input with `>` prefix (always at bottom)
- **Status** (optional): Right-aligned transient status indicators

---

## 2. Architecture

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         PureTTY Adapter                         │
│                   (implements UI port interface)                │
└────────────┬───────────────────────────────────────────┬────────┘
             │                                           │
    ┌────────▼────────┐                         ┌────────▼────────┐
    │  Terminal       │                         │  Block System   │
    │  Subsystem      │                         │                 │
    ├─────────────────┤                         ├─────────────────┤
    │ • TTY (raw mode)│                         │ • Block Model   │
    │ • ANSI escapes  │                         │ • Timeline      │
    │ • Window resize │                         │ • Renderer      │
    └─────────────────┘                         └─────────────────┘
             │                                           │
    ┌────────▼────────┐                         ┌────────▼────────┐
    │  Prompt         │                         │  Output         │
    │  Subsystem      │                         │  Subsystem      │
    ├─────────────────┤                         ├─────────────────┤
    │ • Buffer/Cursor │                         │ • Printer       │
    │ • History       │                         │ • Streaming     │
    │ • Renderer      │                         │ • Coordinator   │
    │ • Input Loop    │                         └─────────────────┘
    └─────────────────┘
```

---

### 2.2 Package Structure

| Package | Purpose | Lines | Coverage |
|---------|---------|-------|----------|
| [`internal/ui/term`](packages/ui-term.md) | Terminal control (raw mode, ANSI, keyboard) | ~400 | 81% |
| [`internal/ui/prompt`](packages/ui-prompt.md) | Prompt state (buffer, history, renderer, loop) | ~600 | 92% |
| [`internal/ui/output`](packages/ui-output.md) | Append-only printing with streaming | ~300 | 89% |
| [`internal/ui/blocks`](packages/ui-blocks.md) | Block data model, timeline, renderer | ~1200 | 88% |
| [`internal/ui/overlay`](packages/ui-overlay.md) | Command palette overlay | ~300 | 99% |
| [`internal/ui/adapters`](packages/ui-adapters.md) | PureTTY adapter (port implementation) | ~400 | 63% |

**Total:** ~3200 lines, 85%+ coverage across all packages

---

### 2.3 Data Flow

```
User Input  →  Keyboard Events  →  Prompt Model  →  Submit Channel
                                                         │
                                                         ▼
                                              ┌──────────────────┐
                                              │  Core Agent      │
                                              │  (future Phase)  │
                                              └──────────────────┘
                                                         │
                    ┌────────────────────────────────────┴────────┐
                    ▼                                             ▼
            ┌───────────────┐                           ┌────────────────┐
            │  Block Events │                           │ Stream Chunks  │
            │  (append/update)                          │ (LLM response) │
            └───────┬───────┘                           └────────┬───────┘
                    │                                            │
                    ▼                                            ▼
            ┌───────────────┐                           ┌────────────────┐
            │  Timeline     │                           │  Printer       │
            │  (append)     │                           │  (stream)      │
            └───────┬───────┘                           └────────┬───────┘
                    │                                            │
                    └─────────────────┬──────────────────────────┘
                                      ▼
                              ┌───────────────┐
                              │  Coordinated  │
                              │  Writer       │
                              │  (output +    │
                              │   redraw)     │
                              └───────┬───────┘
                                      ▼
                                ┌──────────┐
                                │  stdout  │
                                └──────────┘
```

**Key insight:** Every output write automatically triggers a prompt redraw, ensuring the prompt stays at the bottom without tearing.

---

## 3. Getting Started

### 3.1 Running the TUI

```bash
# Start Spin TUI (default mode)
spin

# Or explicitly specify TUI mode
spin --mode tui

# With specific model
spin --provider ollama --model qwen3:1.7b
```

---

### 3.2 Basic Interaction

**1. Type your prompt:**
```
> Write a function to calculate fibonacci
```

**2. Press `Enter` to submit**

The agent responds with streaming output and creates blocks for each action.

**3. Navigate the timeline:**
- `PgUp` / `PgDn` - Scroll by page
- `g` - Jump to top
- `G` - Jump to bottom

**4. Interact with blocks:**
- `Enter` - Toggle fold/expand
- `y` - Copy block body to clipboard
- `S` - Save block to file

**5. Exit:**
- `Ctrl-D` - Clean exit
- `Ctrl-C` - Cancel current operation or exit if idle

---

### 3.3 First Session Example

```
$ spin

Spin v1.0.0 - Coding agent with native TUI
Press ? for help, Ctrl-D to exit

> list files in current directory

│ ▐EXECUTE▌  ls -la (cwd: ".")                           [impact: low]
│   total 48
│   drwxr-xr-x  12 user  staff   384 Oct 11 10:30 .
│   drwxr-xr-x   8 user  staff   256 Oct 10 15:20 ..
│   -rw-r--r--   1 user  staff  1043 Oct 11 09:15 main.go
│   -rw-r--r--   1 user  staff   256 Oct 10 18:42 README.md
│ ✓ [exit: 0] [out: 4 lines] [dur: 0.02s]

> fix the import in main.go

│ ▐READ▌  (file: main.go)
│   │1  package main
│   │2
│   │3  import "fmtt"  // typo here
│   │4
│   │5  func main() {

│ ▐APPLY_PATCH▌  (file: main.go)
│   @@ -1,3 +1,3 @@
│    package main
│
│   -import "fmtt"
│   +import "fmt"
│
│    func main() {
│ ✓ Succeeded. File edited. (-1 removed, +1 added)

> _
```

---

## 4. Keymap Reference

### 4.1 Global Shortcuts

| Key | Action | Mode | Notes |
|-----|--------|------|-------|
| `Ctrl-D` | Exit | Any | Clean shutdown |
| `Ctrl-C` | Cancel/Exit | Any | Cancel input or exit if idle |
| `?` | Help | Any | Show keymap (future) |
| `Ctrl-L` | Clear screen | Any | Redraw everything |

---

### 4.2 Input Mode

| Key | Action | Notes |
|-----|--------|-------|
| `Enter` | Submit line | Send prompt to agent |
| `Backspace` | Delete char left | Standard editing |
| `Delete` | Delete char right | |
| `Left` / `Right` | Move cursor | Character navigation |
| `Home` / `End` | Jump to start/end | Line boundaries |
| `Ctrl-U` | Clear line left | Kill to start of line |
| `Ctrl-K` | Clear line right | Kill to end of line |
| `Ctrl-W` | Delete word left | Word-boundary deletion |
| `Up` / `Down` | History navigation | Previous/next command |

---

### 4.3 Timeline Navigation

| Key | Action | Notes |
|-----|--------|-------|
| `PgUp` | Scroll up one page | Viewport-aware |
| `PgDn` | Scroll down one page | |
| `g` | Jump to top | First block |
| `G` | Jump to bottom | Last block |
| `[` | Previous block | Focus navigation |
| `]` | Next block | |

---

### 4.4 Block Actions

| Key | Action | Notes |
|-----|--------|-------|
| `Enter` | Toggle fold/expand | Collapse/expand block body |
| `y` | Copy block body | Copies to system clipboard |
| `S` | Save block to file | Prompts for filename |
| `r` | Rerun EXECUTE block | Re-executes command |
| `o` | Open file at line | Works on READ/GREP/diff anchors |

---

### 4.5 Advanced

| Key | Action | Notes |
|-----|--------|-------|
| `Ctrl-P` | Command palette | Fuzzy search commands |
| `/` | Filter timeline | Enter filter mode |
| `Esc` | Clear filter | Exit filter mode |
| `zR` | Expand all blocks | Bulk expand |
| `zM` | Collapse all blocks | Bulk collapse |

---

## 5. Block Types

### 5.1 EXECUTE - Shell Command Execution

**Purpose:** Displays shell command execution results.

**Metadata:**
- Command, working directory, timeout
- Exit code, duration, output line count
- Impact level (low/medium/high)

**Visual example:**
```
│ ▐EXECUTE▌  go test ./... (cwd: "./", timeout: 600s)    [impact: medium]
│   === RUN   TestFoo
│   --- PASS: TestFoo (0.00s)
│   === RUN   TestBar
│   --- PASS: TestBar (0.15s)
│   PASS
│   ok      github.com/user/project    0.151s
│ ✓ [exit: 0] [out: 6 lines] [dur: 0.2s]
```

**Error example:**
```
│ ▐EXECUTE▌  make build (cwd: "./")                      [impact: high]
│   go build -o bin/app ./cmd/app
│   # github.com/user/project/cmd/app
│   cmd/app/main.go:15:2: undefined: foo
│   make: *** [build] Error 2
│ ✗ [exit: 2] [out: 4 lines] [dur: 1.3s]
```

---

### 5.2 PLAN - Task Checklist

**Purpose:** Shows planned tasks with progress tracking.

**Metadata:**
- Total tasks, pending, in-progress, completed counts

**Visual example:**
```
│ ▐PLAN▌  Updated: 5 total (1 pending, 1 in progress, 3 completed)
│   ✓ Install dependencies (completed)
│   ✓ Create main.go skeleton (completed)
│   ◦ Write tests (in progress)
│   • Add documentation (pending)
│   • Deploy to production (pending)
```

**Bullets:**
- `✓` - Completed (green, strikethrough)
- `◦` - In progress (yellow)
- `•` - Pending (dim)

---

### 5.3 READ - File Content Preview

**Purpose:** Shows file content with line numbers.

**Metadata:**
- File path, offset, limit (line range)

**Visual example:**
```
│ ▐READ▌  (file: internal/tui/input.go, offset: 0, limit: 50)
│   │ 1  package tui
│   │ 2
│   │ 3  import (
│   │ 4      "context"
│   │ 5      "io"
│   │ 6  )
│   │ 7
│   │ 8  // Input handles user input with history and completion
│   │ 9  type Input struct {
│   │10      buffer  []rune
│   │11      cursor  int
│   │12      history []string
│   │13  }
```

**Features:**
- Dynamic gutter width (adapts to line numbers: 3-6 characters)
- Syntax highlighting (future)
- Jump to line with `o` key

---

### 5.4 GREP - Search Results

**Purpose:** Displays search matches across files.

**Metadata:**
- Pattern, mode (content/files), context lines

**Visual example:**
```
│ ▐GREP▌  ("TODO", content mode, context: 2)
│   main.go:42:
│   40:  func process() {
│   41:      // TODO: Add error handling
│   42:      fmt.Println("processing")
│   43:  }
│
│   utils.go:18:
│   16:  func helper() {
│   17:      // TODO: Optimize this loop
│   18:      for i := 0; i < n; i++ {
│   19:  }
```

**Features:**
- Clickable anchors (`filename:line`)
- Press `o` to open file at match

---

### 5.5 APPLY_PATCH - File Modification

**Purpose:** Shows diff of file changes.

**Metadata:**
- File path, success status
- Lines added/removed, error message

**Visual example (success):**
```
│ ▐APPLY_PATCH▌  (file: main.go)
│   @@ -15,6 +15,7 @@ func main() {
│    func process() {
│        fmt.Println("start")
│   +    log.Info("processing started")
│        doWork()
│        fmt.Println("done")
│    }
│ ✓ Succeeded. File edited. (+1 added)
```

**Visual example (failure):**
```
│ ▐APPLY_PATCH▌  (file: config.yaml)
│   @@ -5,3 +5,4 @@
│    port: 8080
│    host: localhost
│   +timeout: 30s
│ ✗ Failed. Patch conflict at line 7.
│   Run: git apply --reject patch.diff
```

**Diff rendering:**
- Red lines: Deletions (prefix: `-`)
- Green lines: Additions (prefix: `+`)
- Gray lines: Context (no prefix)
- Hunk headers: `@@ -15,6 +15,7 @@` (muted)

---

### 5.6 SUMMARY - Changeset Summary

**Purpose:** Human-readable summary of changes.

**Visual example:**
```
│ ▐SUMMARY▌  Changes applied
│   Added error handling to the process() function:
│   • Log entry at function start
│   • Panic recovery with defer
│   • Structured error return
│
│   Files modified:
│   • main.go (+12 lines, -3 lines)
│   • errors.go (new file, +45 lines)
```

**Rendering:**
- Paragraphs and bullet lists
- Plain text, no special formatting

---

### 5.7 TESTING - Test Execution Guide

**Purpose:** Shows how to run tests and results.

**Metadata:**
- Total suites, passed, failed counts

**Visual example:**
```
│ ▐TESTING▌  Test plan (3 suites)
│   ✓ go test -race ./internal/... (passed, 0.5s)
│   ✓ go test -bench=. ./... (passed, 2.1s)
│   ✗ integration tests (failed, 5.2s)
│       Error: database connection timeout
│       Re-run: make test-integration
```

---

### 5.8 NOTICE - System Messages

**Purpose:** Informational system notices.

**Visual example:**
```
│ ▐NOTICE▌  Conversation history compressed
│   Previous messages have been summarized to reduce context size.
│   Full history available in session log: ~/.spin/sessions/20251011-103042.json
```

**Styling:**
- Muted colors (gray text)
- No special formatting
- Info-level severity

---

### 5.9 ERROR - Error Messages

**Purpose:** Prominent error display.

**Visual example:**
```
│ ▐ERROR▌  Command execution failed
│   ● Error: exit status 1
│
│   Stack trace:
│   at processFile (internal/core/exec.go:142)
│   at runCommand (internal/core/agent.go:89)
│   at handleTurn (internal/core/session.go:67)
│
│   Suggestion: Check file permissions and retry
│ [exit: 1]
```

**Styling:**
- Red accent bar
- First line bold (error message)
- Subsequent lines dim (stack trace)
- Footer with exit code

---

## 6. Advanced Features

### 6.1 Command Palette (Ctrl-P)

**Activation:** Press `Ctrl-P`

**Features:**
- Fuzzy search across commands
- Categories: Run, Search, Open, Plan, Settings
- Real-time filtering as you type

**Available commands:**
- `Run: Execute shell command`
- `Search: Find in repository`
- `Open: Recent file`
- `Plan: New task list`
- `Toggle: Execution mode (auto/manual)`
- `Theme: Change color scheme`

**Navigation:**
- `Up`/`Down` - Select command
- `Enter` - Execute selected command
- `Esc` - Close palette
- `Backspace` - Edit filter query

**Visual example:**
```
┌─────────────────────────────────────────┐
│  Command Palette                  Ctrl-P │
├─────────────────────────────────────────┤
│  > run                                   │
│                                          │
│  ▸ Run: Execute shell command            │
│    Run: Recent command                   │
│    Search: Run grep                      │
│                                          │
│  3 results                               │
└─────────────────────────────────────────┘
```

---

### 6.2 Timeline Filtering (`/`)

**Activation:** Press `/` to enter filter mode

**Filter syntax:**
```
type:EXECUTE                    # Show only EXECUTE blocks
file:main.go                    # Blocks related to main.go
exit:1                          # Failed commands (exit code 1)
impact:high                     # High-impact operations
type:EXECUTE exit:0             # Successful commands (AND logic)
```

**Filter chips display:**
```
> _                    [Filters: type:EXECUTE exit:0]
```

**Clear filter:** Press `Esc`

**Supported fields:**
- `type:<BLOCK_TYPE>` - Block type (EXECUTE, PLAN, READ, etc.)
- `file:<path>` - File path (partial match)
- `exit:<code>` - Exit code (exact match)
- `impact:<level>` - Impact level (low/medium/high)

---

### 6.3 Block Actions

#### Copy Block (`y`)

Copies block body to system clipboard (if available) or prints to stdout.

**Use cases:**
- Copy error messages
- Copy diff for manual review
- Copy command output for documentation

---

#### Save Block (`S`)

Saves block body to a file.

**Flow:**
1. Press `S` on focused block
2. Prompt: `Save to: _`
3. Enter filename
4. Block saved

**Default filenames:**
- EXECUTE: `execute_<timestamp>.log`
- APPLY_PATCH: `patch_<timestamp>.diff`
- READ: `<original_filename>_<timestamp>.txt`

---

#### Rerun Command (`r`)

Re-executes an EXECUTE block's command.

**Flow:**
1. Focus EXECUTE block
2. Press `r`
3. Confirmation: `Rerun: <command>? (y/n)`
4. Press `y` to confirm
5. New EXECUTE block appended

**Use cases:**
- Retry failed command
- Re-run tests after fix
- Check command output again

---

### 6.4 Collapse/Expand

**Single block:**
- `Enter` - Toggle focused block

**Bulk operations:**
- `zR` - Expand all blocks
- `zM` - Collapse all blocks

**Visual cues:**
- Collapsed: `[…]` with line count
- Expanded: Full body visible

**Example (collapsed):**
```
│ ▐EXECUTE▌  go test ./... (cwd: "./")    [impact: medium]
│   […] (42 lines)
│ ✓ [exit: 0] [dur: 2.1s]
```

---

## 7. Troubleshooting

### 7.1 Terminal Compatibility

#### Issue: Broken cursor positioning

**Symptoms:**
- Cursor appears in wrong position
- Text overlaps
- Prompt not at bottom

**Causes:**
- Terminal doesn't support ANSI cursor control
- Wide characters (emoji, CJK) miscalculated

**Solutions:**
```bash
# Check terminal type
echo $TERM  # Should be xterm-256color or similar

# Force UTF-8 locale
export LC_ALL=en_US.UTF-8
export LANG=en_US.UTF-8

# Use compatible terminal emulator
# Recommended: kitty, alacritty, iTerm2, Windows Terminal
```

---

#### Issue: Colors not displaying

**Symptoms:**
- All text is white/black
- No syntax highlighting
- No block colors

**Cause:** Terminal doesn't support 256-color mode

**Solutions:**
```bash
# Check color support
tput colors  # Should output 256

# Upgrade TERM
export TERM=xterm-256color

# Use 8-color fallback (future)
spin --theme=8color
```

---

### 7.2 SSH / Tmux Edge Cases

#### Issue: TUI breaks in SSH session

**Symptoms:**
- Raw mode doesn't activate
- Keys not recognized
- Terminal frozen

**Solutions:**
```bash
# Enable tty allocation for SSH
ssh -t user@host spin

# Check PTY availability
tty  # Should output /dev/pts/X

# Use non-interactive mode if PTY unavailable
spin --mode=exec  # Headless execution
```

---

#### Issue: Resize doesn't work in tmux

**Symptoms:**
- Viewport doesn't update on window resize
- Blocks cut off

**Cause:** tmux not forwarding SIGWINCH

**Solutions:**
```bash
# Detach and reattach
tmux detach
tmux attach

# Send manual resize signal
kill -WINCH $(pgrep spin)

# Redraw with Ctrl-L
```

---

### 7.3 Unicode / Emoji Display

#### Issue: Emoji renders as boxes or double-width

**Symptoms:**
- �� instead of emoji
- Text alignment broken

**Cause:** Font doesn't support emoji or terminal calculates width incorrectly

**Solutions:**
```bash
# Use monospace font with emoji support
# Recommended: JetBrains Mono, Fira Code, Cascadia Code

# Enable Unicode-lite mode (future)
spin --unicode-lite

# Disable emoji in blocks (config)
# ~/.spin/config.yaml
ui:
  emoji: false
```

---

### 7.4 Performance Issues

#### Issue: Scrolling lags with many blocks

**Symptoms:**
- Viewport render >16ms (below 60fps)
- Noticeable stutter when scrolling

**Cause:** Extremely large timeline (>100k blocks)

**Solutions:**
```bash
# Clear old sessions
rm ~/.spin/sessions/*.json

# Enable aggressive history compression (future)
# config.yaml
session:
  max_blocks: 10000
  auto_compress: true

# Check performance metrics
spin --debug-perf
```

**Note:** Current implementation handles 100k blocks smoothly (<1ms viewport render). See [docs/performance.md](performance.md).

---

### 7.5 Cursor Visibility

#### Issue: Cursor invisible after exit

**Symptoms:**
- Cursor doesn't appear in shell after exiting Spin
- Terminal seems broken

**Cause:** Spin crashed before restoring cursor (ANSI ShowCursor not sent)

**Solutions:**
```bash
# Manually restore cursor
echo -e "\e[?25h"

# Or
reset

# Prevent issue: always use clean exit (Ctrl-D)
```

**Fixed in:** Phase 1.1 (TTY cleanup on panic/signal)

---

## 8. Examples

### 8.1 Minimal TUI Demo

**Location:** [examples/tui-demo/](../examples/tui-demo/)

**Purpose:** Simplest possible TUI usage

**Run:**
```bash
cd examples/tui-demo
go run main.go
```

**Features:**
- Initialize PureTTY adapter
- Print a few lines
- Accept user input
- Clean shutdown

**Code snippet:**
```go
ui := adapters.NewPureTTY()
ctx := context.Background()

go ui.Run(ctx)
defer ui.Stop()

ui.PrintLine("Welcome to Spin TUI Demo!")
ui.PrintLine("Type 'quit' to exit")

for line := range ui.RequestInput() {
    if line == "quit" {
        break
    }
    ui.PrintLine(fmt.Sprintf("You typed: %s", line))
}
```

---

### 8.2 Streaming Demo

**Location:** [examples/tui-streaming/](../examples/tui-streaming/)

**Purpose:** Demonstrate streaming chunks (simulated LLM response)

**Run:**
```bash
cd examples/tui-streaming
go run main.go
```

**Features:**
- Print static lines
- Stream chunks with delay (simulate LLM tokens)
- Show coalescing behavior
- Use context cancellation

**Code snippet:**
```go
// Simulate LLM token stream
chunks := make(chan string, 100)
go func() {
    defer close(chunks)
    text := "The quick brown fox jumps over the lazy dog."
    for _, word := range strings.Fields(text) {
        chunks <- word + " "
        time.Sleep(50 * time.Millisecond)
    }
}()

// Stream to TUI
err := ui.PrintChunks(ctx, chunks)
```

---

### 8.3 Interactive Block Demo

**Location:** [examples/tui-blocks/](../examples/tui-blocks/)

**Purpose:** Showcase all 9 block types

**Run:**
```bash
cd examples/tui-blocks
go run main.go
```

**Features:**
- Create timeline with sample blocks of each type
- Demonstrate navigation (PgUp/PgDn, g/G)
- Demonstrate collapse/expand (Enter, zR, zM)
- Demonstrate filtering (`/`)

**Code snippet:**
```go
// Create timeline
timeline := blocks.NewTimeline()
timeline.SetViewportHeight(20)

// Add EXECUTE block
execBlock := blocks.NewBlock(blocks.BlockTypeExecute)
execBlock.Title = "Run tests"
execBlock.Body = "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)"
blocks.SetExecuteMeta(execBlock, &blocks.ExecuteMeta{
    Command: "go test ./...",
    ExitCode: ptr.Int(0),
})
timeline.Append(execBlock)

// Add APPLY_PATCH block
patchBlock := blocks.NewBlock(blocks.BlockTypeApplyPatch)
patchBlock.Body = "@@ -1,3 +1,4 @@\n package main\n+import \"log\"\n"
timeline.Append(patchBlock)

// ... (all 9 types)

// Render visible blocks
visible := timeline.GetVisibleBlocks()
renderer := blocks.NewRenderer(80)
for _, block := range visible {
    output, _ := renderer.Render(block)
    fmt.Print(output)
}
```

---

## 9. References

### 9.1 Package Documentation

- [internal/ui/term](packages/ui-term.md) - Terminal control
- [internal/ui/prompt](packages/ui-prompt.md) - Prompt subsystem
- [internal/ui/output](packages/ui-output.md) - Append-only output
- [internal/ui/blocks](packages/ui-blocks.md) - Block system & timeline
- [internal/ui/overlay](packages/ui-overlay.md) - Command palette
- [internal/ui/adapters](packages/ui-adapters.md) - PureTTY adapter

---

### 9.2 Specifications

- [tui-new.md](../specs/tui-implementation/tui-new.md) - Full TUI specification
- [ROADMAP.md](../specs/tui-implementation/ROADMAP.md) - Implementation roadmap

---

### 9.3 FRDs (Feature Requirements Documents)

**Phase 1 - Foundation:**
- [FRD-20251009-tui-terminal-control.md](../specs/frds/FRD-20251009-tui-terminal-control.md)
- [FRD-20251010-keyboard-events.md](../specs/frds/FRD-20251010-keyboard-events.md)

**Phase 2 - Prompt:**
- [FRD-20251010-prompt-model.md](../specs/frds/FRD-20251010-prompt-model.md)
- [FRD-20251010-prompt-renderer.md](../specs/frds/FRD-20251010-prompt-renderer.md)
- [FRD-20251010-input-loop.md](../specs/frds/FRD-20251010-input-loop.md)

**Phase 3 - Output:**
- [FRD-20251010-append-only-printer.md](../specs/frds/FRD-20251010-append-only-printer.md)
- [FRD-20251010-output-prompt-coordination.md](../specs/frds/FRD-20251010-output-prompt-coordination.md)

**Phase 4 - Blocks:**
- [FRD-20251010-block-types-data-model.md](../specs/frds/FRD-20251010-block-types-data-model.md)
- [FRD-20251010-block-rendering-rules.md](../specs/frds/FRD-20251010-block-rendering-rules.md)
- [FRD-20251010-block-timeline.md](../specs/frds/FRD-20251010-block-timeline.md)

**Phase 5-6 - Integration:**
- [FRD-20251010-puretty-adapter.md](../specs/frds/FRD-20251010-puretty-adapter.md)
- [FRD-20251010-block-timeline-ui-integration.md](../specs/frds/FRD-20251010-block-timeline-ui-integration.md)
- [FRD-20251010-command-palette-overlay.md](../specs/frds/FRD-20251010-command-palette-overlay.md)

**Phase 7 - Testing & QA:**
- [FRD-20251010-e2e-tui-tests.md](../specs/frds/FRD-20251010-e2e-tui-tests.md)
- [FRD-20251011-performance-virtualization-validation.md](../specs/frds/FRD-20251011-performance-virtualization-validation.md)

---

### 9.4 Performance

- [docs/performance.md](performance.md) - Benchmarks and optimization guide

**Key metrics:**
- Viewport render: **0.52ms** (31x faster than 60fps target)
- Timeline with 100k blocks: **3ns** GetVisibleBlocks (O(1))
- Streaming: **8.7M chunks/sec** (8700x faster than target)

---

### 9.5 Architecture

- [specs/architecture-overview.md](../specs/architecture-overview.md) - High-level architecture
- [AGENTS.md](../AGENTS.md) - Development workflow and patterns

---

## 10. FAQ

**Q: Can I use Spin TUI over SSH?**

A: Yes, with `ssh -t user@host spin` to allocate a PTY. See [Troubleshooting](#72-ssh--tmux-edge-cases).

---

**Q: Does it work in tmux/screen?**

A: Yes, TUI is designed to work seamlessly in terminal multiplexers. Resize may require manual signal (Ctrl-L).

---

**Q: How do I copy block output?**

A: Press `y` on focused block, or use terminal's native selection (mouse drag + Ctrl-Shift-C).

---

**Q: Can I disable colors?**

A: Future feature. For now, terminal with `TERM=xterm` will show minimal colors.

---

**Q: What's the maximum timeline size?**

A: Tested up to 100k blocks without performance issues. Practical limit is 1M blocks (filter becomes ~2ms).

---

**Q: How do I report bugs?**

A: File issues at https://github.com/dmytrogajewski/spin/issues with:
- Terminal type (`echo $TERM`)
- OS and version
- Reproduction steps
- Screenshot if visual issue

---

## 11. Roadmap (Future)

### Phase 7.4 - Core Integration (Next)

- [ ] Wire TUI to core agent
- [ ] Map core events to blocks
- [ ] LLM stream → PrintChunks
- [ ] Tool execution → EXECUTE blocks
- [ ] File edits → APPLY_PATCH blocks

---

### Phase 8 - Polish (Future)

- [ ] Theming system (Dark/Light/High-contrast)
- [ ] File preview popup (`o` on anchors)
- [ ] Syntax highlighting (tree-sitter/chroma)
- [ ] Session persistence and replay
- [ ] Terminal recordings (asciinema)

---

### Phase 9 - Advanced (Future)

- [ ] Collaborative sessions (multi-user TUI)
- [ ] Voice input integration
- [ ] Plugin system for custom block types
- [ ] LLM response diff (show edits in real-time)

---

**Last Updated:** 2025-10-11
**Version:** 1.0
**Status:** Production-ready (Phases 1-7.2 complete)

---

## Phase 7.4 Completion (2025-10-11)

### Integration with Core Agent

The TUI is now fully integrated with Spin's core agent! 🎉

**What's New:**
- **Event-to-Block Mapper**: Core events are automatically translated into visual blocks
  - `execute_command` → EXECUTE blocks with command, output, exit code
  - `read_file` → READ blocks with file path and content
  - `write_file` → APPLY_PATCH blocks with diff view
  - LLM streaming → Real-time text output via PrintChunks
  - Errors → ERROR blocks with stack traces
  - System messages → NOTICE blocks (history compression, etc.)

- **Full Conversation Flow**: Prompt → LLM → Tool Execution → Results → Blocks
  - User types prompt
  - Agent processes request
  - Tool calls create blocks in real-time
  - Results update blocks with exit codes and duration
  - LLM responses stream smoothly

- **Command Integration**: `spin tui` and `spin` (default) now launch the new TUI
  - No more old TUI code
  - Factory Droid principles upheld (append-only, native scrollback)
  - All tests passing with zero race conditions

**Files:**
- `internal/core/tui_mapper.go` - Event mapper (370 lines)
- `internal/core/tui_mapper_test.go` - Comprehensive tests (437 lines, 12 tests)
- `cmd/spin/tui.go` - TUI command (220 lines)

**Metrics:**
- ✅ All tests pass with `-race`
- ✅ make lint clean
- ✅ Build compiles successfully
- ✅ 12 comprehensive integration tests
- ✅ Zero race conditions

**Next Steps:**
- Phase 8.1: Deprecate old TUI code
- Phase 8.2: Manual QA on diverse terminals
- Production dogfooding and feedback

**Try It:**
```bash
# Build and run
make build
./bin/spin tui

# Or just
./bin/spin
```

The TUI now provides a rich, interactive coding experience with real-time block-based feedback!

---

