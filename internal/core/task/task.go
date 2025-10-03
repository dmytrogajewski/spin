// Package task provides task execution mode interfaces and implementations.
//
// The task package defines different execution modes for the Spin agent,
// such as regular (full interactive), review (read-only), and compact (minimal context).
// Each mode has specific behavior, tool access, and constraints defined through
// the Task interface.
//
// Example usage:
//
//	registry := task.NewRegistry()
//	registry.Register("regular", &Regular{config: cfg})
//	registry.Register("review", &Review{config: cfg})
//	registry.SetDefault("regular")
//
//	task, err := registry.Get("review")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Task:", task.Name())
//	fmt.Println("Allowed Tools:", task.AllowedTools())
package task

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"sync"
)

// Common errors for task operations
var (
	// ErrTaskNotFound is returned when a task is not in the registry
	ErrTaskNotFound = errors.New("task not found")

	// ErrTaskAlreadyRegistered is returned when registering a duplicate task
	ErrTaskAlreadyRegistered = errors.New("task already registered")

	// ErrInvalidTaskName is returned for invalid task names
	ErrInvalidTaskName = errors.New("invalid task name")

	// ErrNilTask is returned when attempting to register a nil task
	ErrNilTask = errors.New("task cannot be nil")

	// ErrNoDefaultTask is returned when no default task is set
	ErrNoDefaultTask = errors.New("no default task set")

	// ErrInvalidTask is returned when task validation fails
	ErrInvalidTask = errors.New("invalid task configuration")
)

// Task defines different execution modes for the agent.
// Each mode has specific behavior, tool access, and constraints.
//
// Task names should be lowercase alphanumeric with optional hyphens
// and underscores (e.g., "regular", "review", "compact").
//
// All Task implementations must be safe for concurrent use by multiple goroutines.
type Task interface {
	// Name returns the unique identifier for this task mode.
	// Names should be lowercase alphanumeric (e.g., "regular", "review", "compact").
	Name() string

	// SystemPrompt returns the mode-specific system prompt that defines
	// the agent's behavior and constraints for this execution mode.
	SystemPrompt() string

	// AllowedTools returns the list of tool names that are permitted
	// in this execution mode. An empty slice means no tools allowed.
	// Tool names should match the registered tool names in the tool registry.
	AllowedTools() []string

	// MaxTokens returns the maximum token budget for this mode.
	// This affects context window size and truncation strategy.
	// Must be a positive value.
	MaxTokens() int

	// Validate validates the task configuration and returns an error
	// if the task is misconfigured or invalid.
	Validate() error
}

// Registry manages task mode registration and lookup.
// It provides thread-safe operations for registering and
// retrieving task implementations.
//
// All methods are safe for concurrent use by multiple goroutines.
type Registry struct {
	tasks       map[string]Task
	mu          sync.RWMutex
	defaultTask string
}

// NewRegistry creates a new task registry.
// The registry is initially empty with no tasks registered.
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]Task),
	}
}

// Register registers a task mode with the given name.
// Returns an error if the task is nil, the name is invalid,
// or the name is already registered.
//
// Task names must be lowercase alphanumeric with optional hyphens
// and underscores, between 1-50 characters.
//
// This method is safe for concurrent use.
func (r *Registry) Register(name string, task Task) error {
	// Validate task is not nil
	if task == nil {
		return ErrNilTask
	}

	// Validate task name
	if err := validateTaskName(name); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate
	if _, exists := r.tasks[name]; exists {
		return fmt.Errorf("%w: %s", ErrTaskAlreadyRegistered, name)
	}

	// Register task
	r.tasks[name] = task
	return nil
}

// Get retrieves a task by name.
// Returns ErrTaskNotFound if the task is not registered.
//
// This method is safe for concurrent use.
func (r *Registry) Get(name string) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, name)
	}

	return task, nil
}

// List returns all registered task names in sorted order.
// Returns an empty slice if no tasks are registered.
//
// This method is safe for concurrent use.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tasks))
	for name := range r.tasks {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Has returns true if a task with the given name is registered.
//
// This method is safe for concurrent use.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tasks[name]
	return exists
}

// SetDefault sets the default task mode name.
// Returns an error if the task is not registered.
//
// This method is safe for concurrent use.
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if task exists
	if _, exists := r.tasks[name]; !exists {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, name)
	}

	r.defaultTask = name
	return nil
}

// GetDefault returns the default task.
// Returns ErrNoDefaultTask if no default is set.
//
// This method is safe for concurrent use.
func (r *Registry) GetDefault() (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultTask == "" {
		return nil, ErrNoDefaultTask
	}

	task, exists := r.tasks[r.defaultTask]
	if !exists {
		// This shouldn't happen if SetDefault works correctly,
		// but handle it gracefully
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, r.defaultTask)
	}

	return task, nil
}

// validateTaskName validates that a task name is valid.
// Valid names are lowercase alphanumeric with optional hyphens and underscores,
// between 1-50 characters, matching the pattern: ^[a-z][a-z0-9_-]*[a-z0-9]$
//
// Invalid patterns:
//   - Empty string
//   - Uppercase letters
//   - Special characters (except - and _)
//   - Starting/ending with dash or underscore
//   - Consecutive dashes or underscores
//   - Length > 50
func validateTaskName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidTaskName)
	}

	if len(name) > 50 {
		return fmt.Errorf("%w: name too long (max 50 characters)", ErrInvalidTaskName)
	}

	// Pattern: lowercase alphanumeric, can contain - and _ in the middle
	// Must start and end with alphanumeric
	pattern := `^[a-z][a-z0-9_-]*[a-z0-9]$|^[a-z]$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return fmt.Errorf("%w: pattern matching error", ErrInvalidTaskName)
	}

	if !matched {
		return fmt.Errorf("%w: name must be lowercase alphanumeric with optional hyphens/underscores, starting and ending with alphanumeric", ErrInvalidTaskName)
	}

	// Check for consecutive dashes or underscores
	if regexp.MustCompile(`[-_]{2,}`).MatchString(name) {
		return fmt.Errorf("%w: name cannot contain consecutive dashes or underscores", ErrInvalidTaskName)
	}

	return nil
}
