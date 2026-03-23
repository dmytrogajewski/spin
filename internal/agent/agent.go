// Package agent provides the Agent type — a thin service holder
// that delegates turn execution to a harness.Executor via bridge.TurnExecutor.
package agent

import (
	"errors"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/agentsmd"
	spinerrors "github.com/dmytrogajewski/spin/pkg/apperr"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/safety"
)

// Default agent configuration values.
const (
	DefaultMaxTurns        = 50
	DefaultAgentTimeout    = 60 * time.Minute
	DefaultTemperature     = 0.7
	DefaultMaxTokens       = 8192
	DefaultEventBufferSize = 100
)

// Common agent errors.
var (
	// ErrNilLLM is a sentinel error.
	ErrNilLLM = errors.New("LLM provider cannot be nil")
	// ErrNilSecurity is a sentinel error.
	ErrNilSecurity = errors.New("security service cannot be nil")
	// ErrNilDetection is a sentinel error.
	ErrNilDetection = errors.New("detection service cannot be nil")
	// ErrNilToolRuntime is a sentinel error.
	ErrNilToolRuntime = errors.New("tool runtime cannot be nil")
	// ErrNilContext is a sentinel error.
	ErrNilContext = errors.New("context cannot be nil")
	// ErrNilEmitter is a sentinel error.
	ErrNilEmitter = errors.New("event emitter cannot be nil")
	// ErrInvalidTaskMode is a sentinel error.
	ErrInvalidTaskMode = errors.New("invalid task mode")
)

// CallParams holds pre-resolved task parameters for LLM calls.
type CallParams struct {
	// SystemPrompt is the base system prompt from the resolved task.
	SystemPrompt string
	// MaxTokens is the effective token budget from the resolved task.
	MaxTokens int
}

// Agent is a thin service holder that provides access to security,
// tool runtime, and configuration services. Turn execution is handled
// by the harness executor via the conversation layer.
type Agent struct {
	// Core LLM interaction.
	llm llm.Provider

	// Service layers.
	security     *safety.Service
	detection    *cycle.Service
	toolRuntime  *tool.Runtime
	aceService   *ace.Service      // ACE (Agentic Context Engineering) - optional.
	agentsMD     *agentsmd.Service // AGENTS.md project instructions - optional.
	toolSelector *tool.Selector    // Dynamic tool selection - optional.

	// Infrastructure.
	context *Environment
	emitter *events.EventEmitter

	// Configuration (options-based).
	logger      *slog.Logger
	maxTurns    int
	timeout     time.Duration
	temperature float64
	maxTokens   int
	aceConfig   *ace.Config
}

// Option is a functional option for configuring an Agent.
type Option func(*Agent) error

// WithACEService sets the ACE service for the agent.
func WithACEService(aceService *ace.Service) Option {
	return func(a *Agent) error {
		a.aceService = aceService

		return nil
	}
}

// WithACEConfig sets the ACE configuration on the agent.
func WithACEConfig(aceConfig *ace.Config) Option {
	return func(a *Agent) error {
		a.aceConfig = aceConfig

		return nil
	}
}

// WithAgentsMDService sets the AGENTS.md service for the agent.
func WithAgentsMDService(svc *agentsmd.Service) Option {
	return func(a *Agent) error {
		a.agentsMD = svc

		return nil
	}
}

// WithToolSelector sets the dynamic tool selector for the agent.
func WithToolSelector(selector *tool.Selector) Option {
	return func(a *Agent) error {
		a.toolSelector = selector

		return nil
	}
}

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(maxTurns int) Option {
	return func(a *Agent) error {
		if maxTurns <= 0 {
			return spinerrors.Newf(
				spinerrors.CodeValidation, "Agent.WithMaxTurns", nil,
				"max turns must be positive, got %d", maxTurns,
			)
		}

		a.maxTurns = maxTurns

		return nil
	}
}

// WithAgentTimeout sets the agent execution timeout.
func WithAgentTimeout(timeout time.Duration) Option {
	return func(a *Agent) error {
		if timeout <= 0 {
			return spinerrors.Newf(
				spinerrors.CodeValidation, "Agent.WithAgentTimeout", nil,
				"timeout must be positive, got %v", timeout,
			)
		}

		a.timeout = timeout

		return nil
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(temperature float64) Option {
	return func(a *Agent) error {
		if temperature < 0 || temperature > 2 {
			return spinerrors.Newf(
				spinerrors.CodeValidation, "Agent.WithTemperature", nil,
				"temperature must be between 0 and 2, got %f", temperature,
			)
		}

		a.temperature = temperature

		return nil
	}
}

// WithMaxTokens sets the maximum tokens per LLM call.
func WithMaxTokens(maxTokens int) Option {
	return func(a *Agent) error {
		if maxTokens <= 0 {
			return spinerrors.Newf(
				spinerrors.CodeValidation, "Agent.WithMaxTokens", nil,
				"max tokens must be positive, got %d", maxTokens,
			)
		}

		a.maxTokens = maxTokens

		return nil
	}
}

// WithRequireApproval is a no-op option preserved for backward compatibility.
// Approval handling is now configured through the security service.
func WithRequireApproval(_ bool) Option {
	return func(_ *Agent) error {
		return nil
	}
}

// NewAgent creates a new agent with service-based architecture.
//
// The agent is a thin service holder providing access to security,
// tool runtime, and configuration. Turn execution is handled by the
// harness executor via the conversation layer.
func NewAgent(
	provider llm.Provider,
	securitySvc *safety.Service,
	detectionSvc *cycle.Service,
	runtime *tool.Runtime,
	env *Environment,
	emitter *events.EventEmitter,
	opts ...Option,
) (*Agent, error) {
	if provider == nil {
		return nil, ErrNilLLM
	}

	if securitySvc == nil {
		return nil, ErrNilSecurity
	}

	if detectionSvc == nil {
		return nil, ErrNilDetection
	}

	if runtime == nil {
		return nil, ErrNilToolRuntime
	}

	if env == nil {
		return nil, ErrNilContext
	}

	if emitter == nil {
		return nil, ErrNilEmitter
	}

	logger := slog.Default()

	a := &Agent{
		llm:         provider,
		security:    securitySvc,
		detection:   detectionSvc,
		toolRuntime: runtime,
		context:     env,
		emitter:     emitter,
		logger:      logger,
		maxTurns:    DefaultMaxTurns,
		timeout:     DefaultAgentTimeout,
		temperature: DefaultTemperature,
		maxTokens:   DefaultMaxTokens,
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, spinerrors.New(
				spinerrors.CodeValidation, "Agent.NewAgent",
				"applying option failed", err,
			)
		}
	}

	return a, nil
}

// SecurityService returns the agent's security service.
func (a *Agent) SecurityService() *safety.Service {
	return a.security
}

// ToolRuntime returns the agent's tool runtime.
func (a *Agent) ToolRuntime() *tool.Runtime {
	return a.toolRuntime
}

// ApprovalService updates the approval service on the tool runtime.
func (a *Agent) ApprovalService(service *safety.ApprovalService) {
	if a.toolRuntime != nil {
		a.toolRuntime.ApprovalService(service)
	}
}
