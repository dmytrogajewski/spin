package main

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// commandResult represents the result of parsing user input.
type commandResult struct {
	isCommand bool
	command   string
	args      []string
	rawInput  string
}

// parseCommand checks if input is a command and extracts components.
// Commands start with "/" as the first character (after trimming whitespace).
// Returns commandResult with isCommand=true if input is a valid command.
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
	if len(parts) == 0 || parts[0] == "/" {
		// Just a slash or slash with whitespace - not a valid command
		return commandResult{
			isCommand: false,
			rawInput:  input,
		}
	}

	// Extract command (lowercase) and args (lowercase for mode names)
	cmd := strings.ToLower(parts[0])
	args := make([]string, 0, len(parts)-1)
	for _, arg := range parts[1:] {
		args = append(args, strings.ToLower(arg))
	}

	return commandResult{
		isCommand: true,
		command:   cmd,
		args:      args,
		rawInput:  input,
	}
}

// handleCommand processes slash commands.
// Returns:
//   - handled: true if command was recognized and processed
//   - error: non-nil if command execution failed or exit was requested
func handleCommand(ui *adapters.PureTTY, conv *conversation.Conversation, cmd commandResult) (handled bool, err error) {
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

// handleModeCommand handles /mode command.
// With no arguments: shows current mode.
// With one argument: switches to the specified mode.
func handleModeCommand(ui *adapters.PureTTY, conv *conversation.Conversation, args []string) (bool, error) {
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
	ui.PrintLine(fmt.Sprintf("✓ Switched to %s mode\n", newMode))
	if description != "" {
		ui.PrintLine(fmt.Sprintf("%s\n", description))
	}
	ui.PrintLine("")

	return true, nil
}

// handleHelpCommand handles /help command.
// Displays available commands and mode descriptions.
func handleHelpCommand(ui *adapters.PureTTY, conv *conversation.Conversation, args []string) (bool, error) {
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

// getModeDescription returns a brief description of the mode.
// Returns empty string for unknown modes.
func getModeDescription(mode string) string {
	descriptions := map[string]string{
		"regular":  "Full-featured mode with all tools (16K tokens)",
		"review":   "Read-only mode for code analysis (12K tokens)",
		"compact":  "Quick queries with minimal tools (4K tokens)",
		"planning": "Task planning and decomposition (4K tokens)",
	}
	return descriptions[mode]
}
