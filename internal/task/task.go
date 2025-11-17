package task

import (
	"fmt"
	"strings"
)

// Task represents a task mode implementation.
type Task interface {
	// Name returns the task name
	Name() string

	// SystemPrompt returns the system prompt for this task
	SystemPrompt() string

	// AllowedTools returns the list of allowed tools for this task
	AllowedTools() []string

	// MaxTokens returns the maximum token budget for this task
	MaxTokens() int

	// Validate validates the task configuration
	Validate() error
}

// NewTask creates a task instance by name.
// This replaces the runtime registry pattern with compile-time safety.
//
// Supported task names: "regular", "review", "compact", "planning"
// Default task: "regular"
func NewTask(name string) (Task, error) {
	switch name {
	case "regular", "":
		return NewRegular(), nil
	case "review":
		return NewReview(), nil
	case "compact":
		return NewCompact(), nil
	case "planning":
		return NewPlanning(), nil
	default:
		return nil, fmt.Errorf("unknown task: %s", name)
	}
}

// DefaultTask returns the default task (regular).
func DefaultTask() Task {
	return NewRegular()
}

// ValidModes lists all valid task mode names.
var ValidModes = []string{
	"regular",
	"review",
	"compact",
	"planning",
}

// validModesMap is a lookup map for O(1) validation.
var validModesMap = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// ValidateMode checks if a task mode name is valid.
// Empty string is valid (means use default).
func ValidateMode(mode string) error {
	if mode == "" {
		return nil
	}
	if !validModesMap[mode] {
		return fmt.Errorf("invalid task mode: %s (must be one of: %s)", mode, strings.Join(ValidModes, ", "))
	}
	return nil
}
