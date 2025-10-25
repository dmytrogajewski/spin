package orchestration

// Task represents a task mode that defines how the agent should behave.
// Different task modes have different capabilities, constraints, and behaviors.
type Task interface {
	// Name returns the unique identifier for this task mode
	Name() string

	// SystemPrompt returns the system prompt that defines the agent's behavior
	// for this specific task mode
	SystemPrompt() string

	// AllowedTools returns the list of tool names that are permitted
	// in this task mode. Empty slice means all tools are allowed.
	AllowedTools() []string

	// MaxTokens returns the maximum token budget for this task mode
	MaxTokens() int

	// Validate validates the task configuration and returns an error
	// if the task is misconfigured or invalid.
	Validate() error
}

// Registry manages task mode registration and lookup.
