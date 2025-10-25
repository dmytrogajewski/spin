package orchestration

import (
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/task"
)

// Registry manages task implementations for orchestration.
type Registry struct {
	tasks       map[string]task.Task
	defaultTask string
	mu          sync.RWMutex
}

// NewRegistry creates a new orchestration registry.
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]task.Task),
	}
}

// Register registers a task implementation.
func (r *Registry) Register(name string, task task.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	r.tasks[name] = task
	return nil
}

// Get retrieves a task by name.
func (r *Registry) Get(name string) (task.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		return nil, fmt.Errorf("task name cannot be empty")
	}

	task, exists := r.tasks[name]
	if !exists {
		return nil, fmt.Errorf("task %s not found", name)
	}

	return task, nil
}

// SetDefault sets the default task.
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[name]; !exists {
		return fmt.Errorf("task %s not found", name)
	}

	r.defaultTask = name
	return nil
}

// GetDefault returns the default task.
func (r *Registry) GetDefault() (task.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultTask == "" {
		return nil, fmt.Errorf("no default task set")
	}

	return r.Get(r.defaultTask)
}

// List returns all registered task names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tasks))
	for name := range r.tasks {
		names = append(names, name)
	}

	return names
}
