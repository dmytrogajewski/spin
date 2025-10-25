package task

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
