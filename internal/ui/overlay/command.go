// Package overlay provides UI overlays like command palette and file preview.
package overlay

import "context"

// Command represents an executable action in the palette.
type Command interface {
	// Name returns the primary display name (e.g., "Run...").
	Name() string

	// Description returns a short explanation (e.g., "Execute shell command").
	Description() string

	// Category returns grouping (e.g., "Edit", "View", "Tools").
	Category() string

	// Icon returns a 1-char glyph (e.g., "▶", "🔍").
	Icon() rune

	// Execute runs the command and returns error if failed.
	Execute(ctx context.Context) error
}

// CommandRegistry holds available commands.
type CommandRegistry struct {
	commands []Command
}

// NewCommandRegistry creates a new command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make([]Command, 0, 16),
	}
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(cmd Command) {
	r.commands = append(r.commands, cmd)
}

// Commands returns all registered commands in registration order.
func (r *CommandRegistry) Commands() []Command {
	return r.commands
}

// simpleCommand is a basic Command implementation for testing and built-in commands.
type simpleCommand struct {
	name        string
	description string
	category    string
	icon        rune
	exec        func(context.Context) error
}

// NewSimpleCommand creates a command from basic fields.
func NewSimpleCommand(name, description, category string, icon rune, exec func(context.Context) error) Command {
	return &simpleCommand{
		name:        name,
		description: description,
		category:    category,
		icon:        icon,
		exec:        exec,
	}
}

// Name implements the Name operation.
func (c *simpleCommand) Name() string        { return c.name }
// Description implements the Description operation.
func (c *simpleCommand) Description() string { return c.description }
// Category implements the Category operation.
func (c *simpleCommand) Category() string    { return c.category }
// Icon implements the Icon operation.
func (c *simpleCommand) Icon() rune          { return c.icon }
// Execute implements the Execute operation.
func (c *simpleCommand) Execute(ctx context.Context) error {
	if c.exec == nil {
		return nil
	}

	return c.exec(ctx)
}
