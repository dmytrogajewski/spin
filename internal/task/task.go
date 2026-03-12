package task

import (
	"errors"
	"fmt"
	"strings"
)

const (
	TaskNameRegular  = "regular"
	TaskNameReview   = "review"
	TaskNameCompact  = "compact"
	TaskNamePlanning = "planning"
)

var (
	ErrUnknownTask                 = errors.New("unknown task")
	ErrInvalidTaskMode             = errors.New("invalid task mode")
	ErrMaxTokensMustBePositive     = errors.New("max tokens must be positive")
	ErrMaxTokensExceedsMaximumAllowed = errors.New("max tokens  exceeds maximum allowed")
)

// Task represents a task mode implementation.
type Task interface {
	// Name returns the task name.
	Name() string

	// SystemPrompt returns the system prompt for this task.
	SystemPrompt() string

	// AllowedTools returns the list of allowed tools for this task.
	AllowedTools() []string

	// MaxTokens returns the maximum token budget for this task.
	MaxTokens() int

	// Validate validates the task configuration.
	Validate() error
}

// NewTask creates a task instance by name.
// This replaces the runtime registry pattern with compile-time safety.
//
// Supported task names: "regular", "review", "compact", "planning"
// Default task: "regular".
func NewTask(name string) (Task, error) {
	switch name {
	case TaskNameRegular, "":
		return NewRegular(), nil
	case TaskNameReview:
		return NewReview(), nil
	case TaskNameCompact:
		return NewCompact(), nil
	case TaskNamePlanning:
		return NewPlanning(), nil
	default:
return nil, fmt.Errorf("unknown task: %s: %w", name, ErrUnknownTask)
	}
}

// DefaultTask returns the default task (regular).
func DefaultTask() Task {
	return NewRegular()
}

// ValidModes lists all valid task mode names.
var ValidModes = []string{
	TaskNameRegular,
	TaskNameReview,
	TaskNameCompact,
	TaskNamePlanning,
}

// validModesMap is a lookup map for O(1) validation.
var validModesMap = map[string]bool{
	TaskNameRegular:  true,
	TaskNameReview:   true,
	TaskNameCompact:  true,
	TaskNamePlanning: true,
}

// ValidateMode checks if a task mode name is valid.
// Empty string is valid (means use default).
func ValidateMode(mode string) error {
	if mode == "" {
		return nil
	}

	if !validModesMap[mode] {
return fmt.Errorf("invalid task mode: %s (must be one of: %s): %w", mode, strings.Join(ValidModes, ", "), ErrInvalidTaskMode)
	}

	return nil
}
