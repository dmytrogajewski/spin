package harness

import (
	"errors"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Sentinel errors for executor validation.
var (
	ErrNilSpec       = errors.New("spec cannot be nil")
	ErrNilCaller     = errors.New("caller cannot be nil")
	ErrNilDispatcher = errors.New("tool dispatcher cannot be nil")
	ErrEmptyResponse = errors.New("LLM returned empty response (no content, no tool calls)")
)

// Option configures an Executor.
type Option func(*Executor)

// WithCompactor sets the context compactor for Phase 0.
func WithCompactor(c ContextCompactor) Option {
	return func(e *Executor) {
		e.compactor = c
	}
}

// WithReminderInjector sets the reminder injector.
func WithReminderInjector(r ReminderInjector) Option {
	return func(e *Executor) {
		e.reminderInjector = r
	}
}

// WithObservationSummarizer sets the observation summarizer for Phase 3.
func WithObservationSummarizer(o ObservationSummarizer) Option {
	return func(e *Executor) {
		e.observationSummarizer = o
	}
}

// WithEmitter sets the event emitter for phase event emission.
func WithEmitter(em *events.EventEmitter) Option {
	return func(e *Executor) {
		e.emitter = em
	}
}

// Executor runs the Extended ReAct execution loop.
// It consumes a compiled scaffold.Spec and orchestrates LLM calls,
// tool dispatch, guard checks, and middleware hooks.
type Executor struct {
	spec                  *scaffold.Spec
	caller                Caller
	dispatcher            ToolDispatcher
	guards                []Guard
	middlewares           []Middleware
	toolSchemas           []tools.ToolSchema
	maxTurns              int
	logger                *slog.Logger
	compactor             ContextCompactor
	reminderInjector      ReminderInjector
	observationSummarizer ObservationSummarizer
	emitter               *events.EventEmitter
}

// NewExecutor creates an Executor from a compiled spec and its dependencies.
// Returns an error if required dependencies are nil.
// Optional contexteng components can be provided via functional options.
func NewExecutor(
	spec *scaffold.Spec,
	caller Caller,
	dispatcher ToolDispatcher,
	guards []Guard,
	middlewares []Middleware,
	logger *slog.Logger,
	opts ...Option,
) (*Executor, error) {
	if spec == nil {
		return nil, ErrNilSpec
	}

	if caller == nil {
		return nil, ErrNilCaller
	}

	if dispatcher == nil {
		return nil, ErrNilDispatcher
	}

	if logger == nil {
		logger = slog.Default()
	}

	exec := &Executor{
		spec:        spec,
		caller:      caller,
		dispatcher:  dispatcher,
		guards:      guards,
		middlewares: middlewares,
		toolSchemas: spec.ToolSchemas,
		maxTurns:    spec.Config.MaxTurns,
		logger:      logger,
	}

	for _, opt := range opts {
		opt(exec)
	}

	return exec, nil
}
