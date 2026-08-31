package harness

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/contexteng/retrieval"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
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

// WithRegistry sets a live registry for dynamic tool schema resolution.
// When set, tool schemas are read from the registry on each turn
// instead of using the frozen snapshot from the spec.
func WithRegistry(r *tools.Registry) Option {
	return func(e *Executor) {
		e.registry = r
	}
}

// HookRunner executes lifecycle hook events. *hooks.Runner implements this.
type HookRunner interface {
	Execute(ctx context.Context, event hooks.Event, evtCtx hooks.EventContext) hooks.HookResult
}

// WithHookRunner sets the lifecycle hook runner for PRE_COMPACT and STOP.
func WithHookRunner(r HookRunner) Option {
	return func(e *Executor) {
		e.hookRunner = r
	}
}

// WithRetrievalPipeline sets the context retrieval pipeline for Assemble.
// A nil pipeline is a no-op on the turn path.
func WithRetrievalPipeline(p *retrieval.Pipeline) Option {
	return func(e *Executor) {
		e.retrievalPipeline = p
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
	registry              *tools.Registry // Live registry for dynamic tool schemas.
	maxTurns              int
	logger                *slog.Logger
	compactor             ContextCompactor
	reminderInjector      ReminderInjector
	observationSummarizer ObservationSummarizer
	emitter               *events.EventEmitter
	hookRunner            HookRunner
	retrievalPipeline     *retrieval.Pipeline
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

// SetRetrievalPipeline replaces the retrieval pipeline. A nil value disables Assemble.
func (e *Executor) SetRetrievalPipeline(p *retrieval.Pipeline) {
	e.retrievalPipeline = p
}

// currentToolSchemas returns live tool schemas from registry if available,
// falling back to the frozen snapshot from the spec.
func (e *Executor) currentToolSchemas() []tools.ToolSchema {
	if e.registry != nil {
		return e.registry.ListSchemas()
	}

	return e.toolSchemas
}
