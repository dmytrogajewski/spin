package task

import "fmt"

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
