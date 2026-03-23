package subagent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

const (
	// DefaultMaxConcurrent is the default concurrency cap for subagent goroutines.
	DefaultMaxConcurrent = 3
)

// ErrSpecNotFound indicates that the requested subagent spec does not exist.
var ErrSpecNotFound = errors.New("subagent: spec not found")

// ErrNilSpec indicates that a nil spec was passed to Register.
var ErrNilSpec = errors.New("subagent: spec must not be nil")

// ErrEmptySpecName indicates that a spec with an empty name was passed to Register.
var ErrEmptySpecName = errors.New("subagent: spec name must not be empty")

// ErrPanicked indicates that a subagent executor panicked during execution.
var ErrPanicked = errors.New("subagent: executor panicked")

// Executor is a function that runs a subagent with the given spec and query,
// returning a summary string. The Manager calls this function in a goroutine
// for each spawned subagent.
type Executor func(ctx context.Context, spec *Spec, query string) (string, error)

// Manager compiles and executes subagents with concurrency control.
type Manager struct {
	executor      Executor
	specs         map[string]*Spec
	mu            sync.RWMutex
	maxConcurrent int
	semaphore     *concurrency.Semaphore
}

// NewManager creates a Manager with the given executor and concurrency cap.
// Built-in subagent specs are registered automatically.
func NewManager(executor Executor, maxConcurrent int) *Manager {
	if maxConcurrent <= 0 {
		maxConcurrent = DefaultMaxConcurrent
	}

	mgr := &Manager{
		executor:      executor,
		specs:         make(map[string]*Spec),
		maxConcurrent: maxConcurrent,
		semaphore:     concurrency.NewSemaphore(maxConcurrent),
	}

	// Register built-in subagent specs.
	for _, spec := range Builtins() {
		mgr.specs[spec.Name] = spec
	}

	return mgr
}

// Register adds a custom subagent spec. Overwrites any existing spec with the same name.
func (m *Manager) Register(spec *Spec) error {
	if spec == nil {
		return ErrNilSpec
	}

	if spec.Name == "" {
		return ErrEmptySpecName
	}

	m.mu.Lock()
	m.specs[spec.Name] = spec
	m.mu.Unlock()

	return nil
}

// Spec returns the spec for the named subagent, or nil if not found.
func (m *Manager) Spec(name string) *Spec {
	m.mu.RLock()
	spec := m.specs[name]
	m.mu.RUnlock()

	return spec
}

// Spawn executes a subagent by name with the given query.
// It acquires a semaphore slot (blocking if at capacity), runs the executor,
// and recovers from panics. Each subagent runs with a fresh conversation context.
func (m *Manager) Spawn(ctx context.Context, specName, query string) (summary string, err error) {
	m.mu.RLock()
	spec, exists := m.specs[specName]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%w: %q", ErrSpecNotFound, specName)
	}

	// Acquire concurrency slot (blocks if at capacity).
	if err := m.semaphore.Acquire(ctx); err != nil {
		return "", fmt.Errorf("subagent %q: %w", specName, err)
	}

	defer m.semaphore.Release()

	// Recover from panics in the executor.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("subagent %q: %w: %v", specName, ErrPanicked, r)
			summary = ""
		}
	}()

	return m.executor(ctx, spec, query)
}
