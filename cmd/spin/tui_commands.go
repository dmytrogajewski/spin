package main

import (
	"github.com/dmytrogajewski/spin/internal/commands"
)

// parseCommand checks if input is a command using the command system.
// Returns commandResult with isCommand=true if input is a valid command.
func parseCommand(input string) commandResult {
	cmd, args, isCmd := commands.ParseCommand(input)
	if !isCmd {
		return commandResult{
			isCommand: false,
			rawInput:  input,
		}
	}

	return commandResult{
		isCommand: true,
		command:   cmd,
		args:      args,
		rawInput:  input,
	}
}

// commandResult represents the result of parsing user input.
// Kept for backward compatibility with existing TUI code.
type commandResult struct {
	isCommand bool
	command   string
	args      []string
	rawInput  string
}
