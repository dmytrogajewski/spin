// Package commands provides slash command handling.
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/ds/syncmap"
)

var (
	// ErrUnknownCommand is a sentinel error.
	ErrUnknownCommand = errors.New("unknown command")
	// ErrExitCommandIsNotAvailableVia is a sentinel error.
	ErrExitCommandIsNotAvailableVia = errors.New("exit command is not available via ACP protocol")
	// ErrQuitCommandIsNotAvailableVia is a sentinel error.
	ErrQuitCommandIsNotAvailableVia = errors.New("quit command is not available via ACP protocol")
	// ErrModeCannotBeEmpty is a sentinel error.
	ErrModeCannotBeEmpty = errors.New("mode cannot be empty")
	// ErrInvalidMode is a sentinel error.
	ErrInvalidMode = errors.New("invalid mode")
)

// Command represents a command that can be executed.
type Command interface {
	// Name returns the command name (e.g., "/mode", "/help").
	Name() string
	// Description returns a human-readable description of what the command does.
	Description() string
	// Execute executes the command with the given arguments and context.
	// Returns the command output as a string and an error if execution failed.
	Execute(ctx context.Context, args []string, context CommandContext) (string, error)
}

// CommandContext provides context for command execution.
// Different implementations can provide different context (TUI, ACP, etc.).
type CommandContext interface {
	// GetCurrentMode returns the current task mode for the session.
	GetCurrentMode() string
	// SetMode sets the task mode for the session.
	SetMode(ctx context.Context, mode string) error
	// GetWorkDir returns the working directory for the session.
	GetWorkDir() string
}

// registry stores all registered commands.
var registry = syncmap.New[string, Command]()

// RegisterCommand registers a command in the global registry.
func RegisterCommand(cmd Command) {
	registry.Set(cmd.Name(), cmd)
}

// GetCommand retrieves a command by name from the registry.
// Returns the command and true if found, nil and false otherwise.
func GetCommand(name string) (Command, bool) {
	return registry.Get(name)
}

// ListCommands returns all registered commands.
func ListCommands() []Command {
	return registry.Values()
}

// ParseCommand checks if input is a command and extracts components.
// Commands start with "/" as the first character (after trimming whitespace).
// Returns the command name (lowercase), arguments, and true if input is a command.
func ParseCommand(input string) (command string, args []string, isCommand bool) {
	trimmed := strings.TrimSpace(input)

	// Check for slash prefix.
	if !strings.HasPrefix(trimmed, "/") {
		return "", nil, false
	}

	// Split into command and arguments.
	parts := strings.Fields(trimmed)
	if len(parts) == 0 || parts[0] == "/" {
		// Just a slash or slash with whitespace - not a valid command.
		return "", nil, false
	}

	// Extract command (lowercase) and args (lowercase for mode names).
	cmd := strings.ToLower(parts[0])

	commandArgs := make([]string, 0, len(parts)-1)
	for _, arg := range parts[1:] {
		commandArgs = append(commandArgs, strings.ToLower(arg))
	}

	return cmd, commandArgs, true
}

// ExecuteCommand executes a command by name with the given arguments and context.
func ExecuteCommand(ctx context.Context, commandName string, args []string, cmdCtx CommandContext) (string, error) {
	cmd, exists := GetCommand(commandName)
	if !exists {
		return "", fmt.Errorf("unknown command: %s (type /help for available commands): %w", commandName, ErrUnknownCommand)
	}

	return cmd.Execute(ctx, args, cmdCtx)
}

// ModeCommand handles the /mode command.
type ModeCommand struct{}

// Name implements the Name operation.
func (c *ModeCommand) Name() string {
	return "/mode"
}

// Description implements the Description operation.
func (c *ModeCommand) Description() string {
	return "Show current mode or switch to a different mode"
}

// Execute implements the Execute operation.
func (c *ModeCommand) Execute(ctx context.Context, args []string, cmdCtx CommandContext) (string, error) {
	// No arguments: show current mode.
	if len(args) == 0 {
		currentMode := cmdCtx.GetCurrentMode()

		return fmt.Sprintf("Current mode: %s", currentMode), nil
	}

	// One argument: switch mode.
	newMode := args[0]

	// Validate mode.
	err := validateTaskMode(newMode)
	if err != nil {
		return "", fmt.Errorf("invalid mode: %w", err)
	}

	// Switch mode.
	err = cmdCtx.SetMode(ctx, newMode)
	if err != nil {
		return "", fmt.Errorf("error switching mode: %w", err)
	}

	// Get mode description.
	description := getModeDescription(newMode)

	result := fmt.Sprintf("✓ Switched to %s mode", newMode)
	if description != "" {
		result = result + "\n" + description
	}

	return result, nil
}

// HelpCommand handles the /help command.
type HelpCommand struct{}

// Name implements the Name operation.
func (c *HelpCommand) Name() string {
	return "/help"
}

// Description implements the Description operation.
func (c *HelpCommand) Description() string {
	return "Show this help message"
}

// Execute implements the Execute operation.
func (c *HelpCommand) Execute(_ context.Context, _ []string, _ CommandContext) (string, error) {
	var help strings.Builder
	help.WriteString("Available commands:\n\n")

	// List all registered commands.
	commands := ListCommands()
	for _, cmd := range commands {
		fmt.Fprintf(&help, "  %s  - %s\n", cmd.Name(), cmd.Description())
	}

	help.WriteString("\nAvailable modes:\n\n")
	help.WriteString("  regular   - Full-featured interactive coding\n")
	help.WriteString("              • 16K token budget\n")
	help.WriteString("              • All tools available\n")
	help.WriteString("              • Best for: implementing features, refactoring, complex tasks\n\n")
	help.WriteString("  review    - Read-only code analysis\n")
	help.WriteString("              • 12K token budget\n")
	help.WriteString("              • Read-only tools (read_file, list_directory, get_context, file_search, git_context)\n")
	help.WriteString("              • Best for: code review, understanding codebase, documentation\n\n")
	help.WriteString("  compact   - Quick queries with minimal context\n")
	help.WriteString("              • 4K token budget\n")
	help.WriteString("              • Minimal tools (read_file, get_context, file_search)\n")
	help.WriteString("              • Best for: quick questions, fast iteration, debugging\n\n")
	help.WriteString("  planning  - Task decomposition and planning\n")
	help.WriteString("              • 4K token budget\n")
	help.WriteString("              • Context-only tools (get_context, file_search, git_context)\n")
	help.WriteString("              • Best for: breaking down large tasks, architecture planning\n\n")
	help.WriteString("Examples:\n\n")
	help.WriteString("  /mode review          # Switch to review mode\n")
	help.WriteString("  /mode                 # Show current mode\n")
	help.WriteString("  /resume               # List previous sessions\n")
	help.WriteString("  /resume last          # Continue the newest session\n")
	help.WriteString("  /help                 # Show this help\n")

	return help.String(), nil
}

// ExitCommand handles the /exit and /quit commands.
// These commands are TUI-specific and should return an error when executed via ACP.
type ExitCommand struct{}

// Name implements the Name operation.
func (c *ExitCommand) Name() string {
	return "/exit"
}

// Description implements the Description operation.
func (c *ExitCommand) Description() string {
	return "Exit the session (TUI only, not available via ACP)"
}

// Execute implements the Execute operation.
func (c *ExitCommand) Execute(_ context.Context, _ []string, _ CommandContext) (string, error) {
	return "", ErrExitCommandIsNotAvailableVia
}

// QuitCommand is an alias for ExitCommand.
type QuitCommand struct{}

// Name implements the Name operation.
func (c *QuitCommand) Name() string {
	return "/quit"
}

// Description implements the Description operation.
func (c *QuitCommand) Description() string {
	return "Quit the session (TUI only, not available via ACP)"
}

// Execute implements the Execute operation.
func (c *QuitCommand) Execute(_ context.Context, _ []string, _ CommandContext) (string, error) {
	return "", ErrQuitCommandIsNotAvailableVia
}

// validTaskModes defines the valid task modes.
var validTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// validateTaskMode validates that a mode name is valid.
func validateTaskMode(mode string) error {
	if mode == "" {
		return ErrModeCannotBeEmpty
	}

	if !validTaskModes[mode] {
		return fmt.Errorf("invalid mode: %s (valid modes: regular, review, compact, planning): %w", mode, ErrInvalidMode)
	}

	return nil
}

// getModeDescription returns a brief description of the mode.
func getModeDescription(mode string) string {
	descriptions := map[string]string{
		"regular":  "Full-featured mode with all tools (16K tokens)",
		"review":   "Read-only mode for code analysis (12K tokens)",
		"compact":  "Quick queries with minimal tools (4K tokens)",
		"planning": "Task planning and decomposition (4K tokens)",
	}

	return descriptions[mode]
}

// init registers all default commands.
func init() {
	RegisterCommand(&ModeCommand{})
	RegisterCommand(&HelpCommand{})
	RegisterCommand(&ExitCommand{})
	RegisterCommand(&QuitCommand{})
	RegisterCommand(&ResumeCommand{})
}
