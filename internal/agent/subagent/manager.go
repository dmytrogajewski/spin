package subagent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

const (
	// DefaultMaxConcurrent is the default concurrency cap for admitted children.
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

// ErrNoBackgroundStarter indicates SpawnBackground has no immediate starter.
var ErrNoBackgroundStarter = errors.New("subagent: background starter is not set")

// BackgroundStarter starts a child and returns after non-blocking message/send.
type BackgroundStarter func(ctx context.Context, spec *Spec, query string) (id string, handle tasks.Handle, err error)

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
	background    BackgroundStarter
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

// SetBackgroundStarter installs the non-blocking spawn function.
func (m *Manager) SetBackgroundStarter(fn BackgroundStarter) {
	m.mu.Lock()
	m.background = fn
	m.mu.Unlock()
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
	if acquireErr := m.semaphore.Acquire(ctx); acquireErr != nil {
		return "", fmt.Errorf("subagent %q: %w", specName, acquireErr)
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

// SpawnBackground admits a child, sends immediately, and returns the task id.
// The caller (parent ReAct loop) continues; Wait must not be used here.
func (m *Manager) SpawnBackground(
	ctx context.Context,
	specName, query string,
	reg *tasks.Registry,
) (string, error) {
	m.mu.RLock()
	spec, exists := m.specs[specName]
	start := m.background
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%w: %q", ErrSpecNotFound, specName)
	}

	if start == nil {
		return "", ErrNoBackgroundStarter
	}

	if acquireErr := m.semaphore.Acquire(ctx); acquireErr != nil {
		return "", fmt.Errorf("subagent %q: %w", specName, acquireErr)
	}

	defer m.semaphore.Release()

	id, handle, err := start(ctx, spec, query)
	if err != nil {
		return "", err
	}

	if reg != nil {
		reg.Register(id, spec.Name, tasks.StateWorking, handle)
	}

	return id, nil
}
