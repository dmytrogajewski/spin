package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

var ErrExitRequested = errors.New("exit requested")

// tuiCommandContext implements commands.CommandContext for TUI.
type tuiCommandContext struct {
	conv *conversation.Conversation
}

// GetCurrentMode returns the current task mode for the session.
func (c *tuiCommandContext) GetCurrentMode() string {
	return c.conv.GetTaskMode()
}

// SetMode sets the task mode for the session.
func (c *tuiCommandContext) SetMode(mode string) error {
	return c.conv.SetTaskMode(mode)
}

// GetWorkDir returns the working directory for the session.
func (c *tuiCommandContext) GetWorkDir() string {
	// Conversation doesn't expose workDir directly, but we can get it from session if needed
	// Return empty string as it's not currently used in TUI commands.
	return ""
}

// handleCommand processes slash commands using the command system.
// Returns:
//   - handled: true if command was recognized and processed
//   - error: non-nil if command execution failed or exit was requested
func handleCommand(
	ctx context.Context, ui *adapters.PureTTY,
	conv *conversation.Conversation, cmdName string, args []string,
) (handled bool, err error) {
	// Create command context.
	cmdCtx := &tuiCommandContext{conv: conv}

	// Special handling for exit/quit commands (TUI-only).
	if cmdName == "/exit" || cmdName == "/quit" {
		return true, ErrExitRequested
	}

	// Execute command via command system.
	result, err := commands.ExecuteCommand(ctx, cmdName, args, cmdCtx)
	if err != nil {
		// Check if it's an unknown command.
		if strings.Contains(err.Error(), "unknown command") {
			_ = ui.PrintLine(fmt.Sprintf("Unknown command: %s (type /help for available commands)\n", cmdName))

			return true, nil
		}
		// Other errors.
		_ = ui.PrintLine(fmt.Sprintf("Error: %v\n", err))

		return true, nil
	}

	// Print command output.
	_ = ui.PrintLine(result)

	if !strings.HasSuffix(result, "\n") {
		_ = ui.PrintLine("")
	}

	return true, nil
}
