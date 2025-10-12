# FRD-20251012170000: CLI REPL Mode Switching

**Status**: Ready for Implementation
**Created**: 2025-10-12 17:00
**Roadmap Item**: [P3.2] Implement REPL Mode Switching
**Depends On**: [P3.1] Add Global Task Mode Flag ✅
**Package**: `cmd/spin`
**Files**: `cmd/spin/tui.go` (modify)
**Complexity**: Medium (3-4 hours)

## Overview

Add runtime mode switching capability to the TUI interactive mode through slash commands. Users will be able to switch between task modes (`/mode`) and get help (`/help`) without restarting the session.

## Problem Statement

Currently, users can set the initial task mode via `--mode` flag, but cannot switch modes during an interactive TUI session. This requires restarting the entire session to change modes, which is disruptive to workflow.

## Goals

1. **Enable Runtime Mode Switching**: Allow users to switch task modes mid-conversation
2. **Provide Mode Information**: Users can query current mode and available modes
3. **User-Friendly Commands**: Simple `/mode` slash command interface
4. **Error Handling**: Clear error messages for invalid modes
5. **Help System**: `/help` command explains available commands

## Non-Goals

- Fancy REPL with history/autocomplete (keep it simple for v1)
- Custom mode command persistence across sessions
- Undo/rollback of mode changes
- Mode aliases or shortcuts

## Current State Analysis

### Existing TUI Implementation

The current `tui.go` (lines 18-227) provides:
- Interactive input loop (lines 182-225)
- Event streaming from conversation
- Signal handling (Ctrl-D to exit)
- Basic prompt: "Type your prompt and press Enter"

**Key code sections:**
```go
// Main input loop (lines 182-225)
inputCh := ui.RequestInput()
for {
    select {
    case line, ok := <-inputCh:
        if !ok {
            // UI closed (Ctrl-D)
            return nil
        }

        if line == "" {
            continue
        }

        // Submit prompt to conversation
        err := conv.RunTurn(turnCtx, line)
        // ... error handling ...
    }
}
```

**Missing:**
- Command parsing for `/` prefix
- Mode switching logic
- Help system
- Current mode display

## Proposed Solution

### Architecture

```
User Input
    ↓
┌─────────────────────────────┐
│  Input Parser               │
│  - Check for "/" prefix     │
│  - Route to command handler │
│  - Or pass to conversation  │
└─────────────────────────────┘
    ↓              ↓
Command        Regular
Handler        Message
    ↓              ↓
/mode   /help   RunTurn()
```

### Implementation Plan

#### 1. Add Command Parser

Create `parseCommand()` function to detect slash commands:

```go
// commandResult represents the result of parsing user input
type commandResult struct {
    isCommand bool
    command   string
    args      []string
    rawInput  string
}

// parseCommand checks if input is a command and extracts components
func parseCommand(input string) commandResult {
    trimmed := strings.TrimSpace(input)

    // Check for slash prefix
    if !strings.HasPrefix(trimmed, "/") {
        return commandResult{
            isCommand: false,
            rawInput:  input,
        }
    }

    // Split into command and arguments
    parts := strings.Fields(trimmed)
    if len(parts) == 0 {
        return commandResult{isCommand: false, rawInput: input}
    }

    cmd := strings.ToLower(parts[0])
    args := parts[1:]

    return commandResult{
        isCommand: true,
        command:   cmd,
        args:      args,
        rawInput:  input,
    }
}
```

#### 2. Add Command Handler

Create `handleCommand()` function to execute commands:

```go
// handleCommand processes slash commands
// Returns:
//   - handled: true if command was recognized and processed
//   - error: non-nil if command execution failed
func handleCommand(ui *adapters.PureTTY, conv *core.Conversation, cmd commandResult) (handled bool, err error) {
    switch cmd.command {
    case "/mode":
        return handleModeCommand(ui, conv, cmd.args)

    case "/help":
        return handleHelpCommand(ui, conv, cmd.args)

    case "/exit", "/quit":
        return true, fmt.Errorf("exit requested")

    default:
        ui.PrintLine(fmt.Sprintf("Unknown command: %s (type /help for available commands)\n", cmd.command))
        return true, nil
    }
}

// handleModeCommand handles /mode command
func handleModeCommand(ui *adapters.PureTTY, conv *core.Conversation, args []string) (bool, error) {
    // No arguments: show current mode
    if len(args) == 0 {
        currentMode := conv.GetTaskMode()
        ui.PrintLine(fmt.Sprintf("Current mode: %s\n", currentMode))
        return true, nil
    }

    // One argument: switch mode
    newMode := args[0]

    // Validate mode
    if err := validateTaskMode(newMode); err != nil {
        ui.PrintLine(fmt.Sprintf("Error: %v\n", err))
        return true, nil
    }

    // Switch mode
    if err := conv.SetTaskMode(newMode); err != nil {
        ui.PrintLine(fmt.Sprintf("Error switching mode: %v\n", err))
        return true, nil
    }

    // Confirm switch with mode description
    description := getModeDescription(newMode)
    ui.PrintLine(fmt.Sprintf("✓ Switched to %s mode\n%s\n", newMode, description))
    return true, nil
}

// handleHelpCommand handles /help command
func handleHelpCommand(ui *adapters.PureTTY, conv *core.Conversation, args []string) (bool, error) {
    help := `Available commands:

  /mode [name]  - Show current mode or switch to a different mode
  /help         - Show this help message
  /exit, /quit  - Exit the session (or press Ctrl-D)

Available modes:

  regular   - Full-featured interactive coding
              • 16K token budget
              • All tools available
              • Best for: implementing features, refactoring, complex tasks

  review    - Read-only code analysis
              • 12K token budget
              • Read-only tools (read_file, list_directory, get_context, file_search, git_context)
              • Best for: code review, understanding codebase, documentation

  compact   - Quick queries with minimal context
              • 4K token budget
              • Minimal tools (read_file, get_context, file_search)
              • Best for: quick questions, fast iteration, debugging

  planning  - Task decomposition and planning
              • 4K token budget
              • Context-only tools (get_context, file_search, git_context)
              • Best for: breaking down large tasks, architecture planning

Examples:

  /mode review          # Switch to review mode
  /mode                 # Show current mode
  /help                 # Show this help

`
    ui.PrintLine(help)
    return true, nil
}

// getModeDescription returns a brief description of the mode
func getModeDescription(mode string) string {
    descriptions := map[string]string{
        "regular":  "Full-featured mode with all tools (16K tokens)",
        "review":   "Read-only mode for code analysis (12K tokens)",
        "compact":  "Quick queries with minimal tools (4K tokens)",
        "planning": "Task planning and decomposition (4K tokens)",
    }
    return descriptions[mode]
}
```

#### 3. Integrate into TUI Main Loop

Modify the main input loop in `runTUI()`:

```go
// Main input loop (update lines 182-225)
inputCh := ui.RequestInput()
for {
    select {
    case <-ctx.Done():
        <-eventDone
        return ctx.Err()

    case line, ok := <-inputCh:
        if !ok {
            // UI closed (Ctrl-D)
            <-eventDone
            return nil
        }

        if line == "" {
            continue
        }

        // NEW: Parse for commands
        cmdResult := parseCommand(line)

        if cmdResult.isCommand {
            // Handle command
            handled, err := handleCommand(ui, conv, cmdResult)
            if err != nil {
                if err.Error() == "exit requested" {
                    <-eventDone
                    return nil
                }
                ui.PrintLine(fmt.Sprintf("Command error: %v\n", err))
            }
            // Skip conversation turn for commands
            continue
        }

        // Regular message - submit to conversation
        turnCtx, turnCancel := context.WithCancel(ctx)
        defer turnCancel()

        err := conv.RunTurn(turnCtx, line)
        // ... existing streaming logic ...

        if err != nil {
            ui.PrintLine(fmt.Sprintf("✗ Error: %v\n", err))
        }
    }
}
```

#### 4. Update Welcome Message

Update the welcome message to mention commands:

```go
// In runTUI() after printing logo (line 156)
ui.PrintLine(logo)
ui.PrintLine("Type your prompt and press Enter.")
ui.PrintLine("Commands: /mode [name], /help, /exit (or press Ctrl-D)\n")
```

#### 5. Optional: Show Mode in Prompt

For better UX, could show current mode in prompt (implementation-dependent on PureTTY capabilities):

```go
// If PureTTY supports custom prompts:
currentMode := conv.GetTaskMode()
ui.SetPrompt(fmt.Sprintf("[%s] > ", currentMode))
```

**Note**: This is optional and may require changes to PureTTY interface.

## Testing Strategy

### Unit Tests

Create `cmd/spin/tui_commands_test.go`:

```go
package main

import (
    "testing"
)

func TestParseCommand(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantCmd   bool
        wantName  string
        wantArgs  []string
    }{
        {
            name:     "slash command with no args",
            input:    "/mode",
            wantCmd:  true,
            wantName: "/mode",
            wantArgs: []string{},
        },
        {
            name:     "slash command with args",
            input:    "/mode review",
            wantCmd:  true,
            wantName: "/mode",
            wantArgs: []string{"review"},
        },
        {
            name:     "regular message",
            input:    "Write a test",
            wantCmd:  false,
        },
        {
            name:     "slash in middle of line (not a command)",
            input:    "Use the /api endpoint",
            wantCmd:  false,
        },
        {
            name:     "empty input",
            input:    "",
            wantCmd:  false,
        },
        {
            name:     "just slash",
            input:    "/",
            wantCmd:  false,
        },
        {
            name:     "help command",
            input:    "/help",
            wantCmd:  true,
            wantName: "/help",
            wantArgs: []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := parseCommand(tt.input)

            if got.isCommand != tt.wantCmd {
                t.Errorf("isCommand = %v, want %v", got.isCommand, tt.wantCmd)
            }

            if tt.wantCmd {
                if got.command != tt.wantName {
                    t.Errorf("command = %v, want %v", got.command, tt.wantName)
                }
                if len(got.args) != len(tt.wantArgs) {
                    t.Errorf("args length = %d, want %d", len(got.args), len(tt.wantArgs))
                }
            }
        })
    }
}

func TestGetModeDescription(t *testing.T) {
    tests := []struct {
        mode string
        want string
    }{
        {"regular", "Full-featured mode with all tools (16K tokens)"},
        {"review", "Read-only mode for code analysis (12K tokens)"},
        {"compact", "Quick queries with minimal tools (4K tokens)"},
        {"planning", "Task planning and decomposition (4K tokens)"},
    }

    for _, tt := range tests {
        t.Run(tt.mode, func(t *testing.T) {
            got := getModeDescription(tt.mode)
            if got != tt.want {
                t.Errorf("getModeDescription(%s) = %v, want %v", tt.mode, got, tt.want)
            }
        })
    }
}
```

### Integration Tests

These would test the actual TUI behavior, but are complex due to UI dependencies. For now, focus on:

1. Manual testing of `/mode` and `/help` commands
2. E2E test script (see below)

### E2E Test Script

Create `e2e/test_tui_mode_switching.sh`:

```bash
#!/bin/bash
# E2E test for TUI mode switching

set -e

echo "Testing TUI mode switching..."

# Test 1: /mode command shows current mode
echo "/mode" | timeout 5s spin tui --provider mock > output.txt 2>&1 || true
if grep -q "Current mode: regular" output.txt; then
    echo "✓ /mode shows current mode"
else
    echo "✗ /mode failed to show current mode"
    cat output.txt
    exit 1
fi

# Test 2: /mode review switches mode
echo -e "/mode review\n/mode" | timeout 5s spin tui --provider mock > output.txt 2>&1 || true
if grep -q "Switched to review mode" output.txt && grep -q "Current mode: review" output.txt; then
    echo "✓ /mode review switches mode"
else
    echo "✗ /mode review failed"
    cat output.txt
    exit 1
fi

# Test 3: /mode invalid shows error
echo "/mode invalid" | timeout 5s spin tui --provider mock > output.txt 2>&1 || true
if grep -q "Error.*invalid task mode" output.txt; then
    echo "✓ /mode invalid shows error"
else
    echo "✗ /mode invalid should show error"
    cat output.txt
    exit 1
fi

# Test 4: /help shows help
echo "/help" | timeout 5s spin tui --provider mock > output.txt 2>&1 || true
if grep -q "Available commands:" output.txt && grep -q "/mode" output.txt; then
    echo "✓ /help shows command list"
else
    echo "✗ /help failed"
    cat output.txt
    exit 1
fi

rm -f output.txt
echo "All tests passed!"
```

## Implementation Checklist

### Code Changes

- [ ] Add `commandResult` struct to `tui.go`
- [ ] Implement `parseCommand()` function
- [ ] Implement `handleCommand()` function
- [ ] Implement `handleModeCommand()` function
- [ ] Implement `handleHelpCommand()` function
- [ ] Implement `getModeDescription()` function
- [ ] Update main input loop to call command parser
- [ ] Update welcome message to mention commands
- [ ] Optional: Add mode display in prompt

### Testing

- [ ] Write unit tests for `parseCommand()`
- [ ] Write unit tests for `getModeDescription()`
- [ ] Create E2E test script `e2e/test_tui_mode_switching.sh`
- [ ] Manually test all commands in TUI
- [ ] Test edge cases (empty input, whitespace, case sensitivity)

### Quality Gates

- [ ] `make lint` passes (zero errors)
- [ ] All unit tests pass with `go test ./cmd/spin/...`
- [ ] Race detector clean: `go test -race ./cmd/spin/...`
- [ ] E2E script passes
- [ ] Godoc comments on all new functions
- [ ] Code complexity ≤15 per function

## Definition of Done

**Functional Requirements:**
- [ ] `/mode` with no args shows current mode
- [ ] `/mode <name>` switches to valid mode and shows confirmation
- [ ] `/mode <invalid>` shows clear error message
- [ ] `/help` displays command list and mode descriptions
- [ ] `/exit` and `/quit` exit the session
- [ ] Regular messages (non-commands) work as before
- [ ] Welcome message mentions available commands

**Quality Requirements:**
- [ ] Unit tests cover all command parsing logic
- [ ] E2E test verifies mode switching workflow
- [ ] All tests pass (no flakes)
- [ ] `make lint` clean
- [ ] Race detector clean
- [ ] Godoc complete on all new functions

**Documentation:**
- [ ] Update `cmd/spin/README.md` (if exists) with command list
- [ ] Update roadmap to mark P3.2 complete

## Acceptance Criteria

```bash
# User starts TUI
$ spin tui

# User sees welcome with command hints
> Type your prompt and press Enter.
> Commands: /mode [name], /help, /exit (or press Ctrl-D)

# User checks current mode
> /mode
Current mode: regular

# User switches to review mode
> /mode review
✓ Switched to review mode
Read-only mode for code analysis (12K tokens)

# User verifies mode changed
> /mode
Current mode: review

# User tries invalid mode
> /mode invalid
Error: invalid task mode: invalid (valid modes: regular, review, compact, planning)

# User gets help
> /help
Available commands:

  /mode [name]  - Show current mode or switch to a different mode
  /help         - Show this help message
  /exit, /quit  - Exit the session (or press Ctrl-D)

Available modes:
  regular   - Full-featured interactive coding (16K tokens, all tools)
  ...

# User exits
> /exit
$
```

## Risks and Mitigations

### Risk 1: PureTTY API Limitations
**Impact**: May not support custom prompts or formatting
**Mitigation**: Focus on basic functionality first, enhance later
**Contingency**: Omit optional prompt display feature

### Risk 2: Command Parsing Edge Cases
**Impact**: Users might accidentally trigger commands
**Mitigation**: Require `/` as first character (after trim)
**Contingency**: Add escape mechanism if needed (e.g., `\/` for literal slash)

### Risk 3: Mode Switching During Active Turn
**Impact**: Unexpected behavior if mode changes mid-execution
**Mitigation**: Commands are processed before RunTurn(), so no conflict
**Contingency**: Add conversation lock if needed

### Risk 4: E2E Test Flakiness
**Impact**: CI failures due to timing issues
**Mitigation**: Use mock provider, add reasonable timeouts
**Contingency**: Mark as manual test if too flaky

## Future Enhancements

**Post-v1:**
1. Command history with up/down arrows
2. Tab completion for mode names
3. `/modes` alias for `/mode` (list all)
4. Custom mode definitions via `/mode add <name>`
5. Mode switching confirmation for destructive operations
6. Colorized mode indicator in prompt
7. `/undo` to revert last mode change

## References

- [ROADMAP.md P3.2](../../specs/task-modes/ROADMAP.md#p32-implement-repl-mode-switching)
- [Specification 3.3](../../specs/task-modes/specification.md) (lines 647-721)
- [P2.1 FRD](./FRD-20251012140000-conversation-task-mode.md) - Conversation.SetTaskMode()
- [P3.1 FRD](./FRD-20251012160000-cli-global-task-mode-flag.md) - validateTaskMode()

---

**Status**: ✅ Ready for Implementation
**Next Step**: Begin implementation following checklist above
**Estimated Time**: 3-4 hours
**Target Completion**: 2025-10-12 EOD
