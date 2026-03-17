package safety

import (
	"errors"
	"strings"
)

// validator_types.go defines the core types used by the command validator.
//
// This file contains:
//   - CommandClass: Safety classification levels
//   - Command: Parsed command structure
//   - ValidationResult: Classification result with reason
//   - Pattern: Pattern matching rules
//   - ParseCommand: Command parsing function

// Common validator errors.
var (
	// ErrInvalidCommand is a sentinel error.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrParseError is a sentinel error.
	ErrParseError = errors.New("command parse error")
	// ErrEmptyCommand is a sentinel error.
	ErrEmptyCommand = errors.New("empty command")
)

// CommandClass represents the safety classification of a command.
type CommandClass int

const (
	// CommandSafe - Read-only operations that can execute automatically.
	CommandSafe CommandClass = iota

	// CommandInteractive - Write operations that need user approval.
	CommandInteractive

	// CommandDangerous - Destructive operations requiring strong approval.
	CommandDangerous

	// CommandForbidden - Commands that should never execute.
	CommandForbidden

	// CommandUnverified - Unknown commands with indeterminate safety.
	CommandUnverified
)

// String returns the string representation of the classification.
func (c CommandClass) String() string {
	switch c {
	case CommandSafe:
		return "safe"
	case CommandInteractive:
		return "interactive"
	case CommandDangerous:
		return "dangerous"
	case CommandForbidden:
		return "forbidden"
	case CommandUnverified:
		return "unverified"
	default:
		return "unknown"
	}
}

// NeedsApproval returns true if the command requires user approval.
//
// Safe commands don't need approval (auto-execute).
// Interactive, Dangerous, Forbidden, and Unverified commands need approval.
func (c CommandClass) NeedsApproval() bool {
	return c == CommandInteractive || c == CommandDangerous || c == CommandForbidden || c == CommandUnverified
}

// Command represents a parsed shell command for validation.
type Command struct {
	// Program is the command name (e.g., "ls", "git", "rm").
	Program string

	// Args are the command arguments.
	Args []string

	// Env contains environment variables.
	Env map[string]string

	// WorkDir is the working directory.
	WorkDir string

	// Raw is the original unparsed command string.
	Raw string
}

// GetProgram returns the command program name.
func (c *Command) GetProgram() string { return c.Program }

// GetArgs returns the command arguments.
func (c *Command) GetArgs() []string { return c.Args }

// GetRaw returns the raw command string.
func (c *Command) GetRaw() string { return c.Raw }

// GetWorkDir returns the working directory.
func (c *Command) GetWorkDir() string { return c.WorkDir }

// ValidationResult contains the result of command validation.
type ValidationResult struct {
	// Classification is the determined safety level.
	Classification CommandClass

	// Reason explains why this classification was chosen.
	Reason string

	// MatchedRule is the rule that matched (if any).
	MatchedRule string

	// Confidence is how confident the validator is (0.0-1.0).
	Confidence float64

	// Suggestions for safer alternatives (optional).
	Suggestions []string
}

// GetClassification returns the classification as an int.
func (r *ValidationResult) GetClassification() int { return int(r.Classification) }

// GetReason returns the validation reason.
func (r *ValidationResult) GetReason() string { return r.Reason }

// Pattern represents a command pattern for matching.
type Pattern struct {
	// Program is the command name.
	Program string

	// ArgPatterns are patterns that must match in arguments.
	ArgPatterns []string

	// ForbiddenPatterns are patterns that must NOT appear.
	ForbiddenPatterns []string

	// Description explains this pattern.
	Description string
}

// ParseCommand parses a command string into a Command struct.
//
// The parser handles basic shell syntax including quoted arguments
// and environment variables. Complex shell constructs like pipes,
// redirects, and command substitution are partially supported.
func ParseCommand(cmdStr string) (*Command, error) {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil, ErrEmptyCommand
	}

	// Simple tokenization (space-separated).
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, ErrEmptyCommand
	}

	cmd := &Command{
		Program: strings.ToLower(parts[0]),
		Args:    []string{},
		Env:     make(map[string]string),
		Raw:     cmdStr,
	}

	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}

	return cmd, nil
}
