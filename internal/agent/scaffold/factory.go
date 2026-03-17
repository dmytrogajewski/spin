// Package scaffold provides compile-time agent specification assembly.
// It produces immutable Spec values from configuration, tool registries,
// and provider connections.
package scaffold

import (
	"errors"
	"fmt"
	"slices"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const (
	// AgentTypeMain is the primary agent type with full tool access.
	AgentTypeMain = "main"

	// defaultSystemPrompt is the baseline system prompt for the main agent.
	defaultSystemPrompt = "You are a helpful coding assistant " +
		"with access to tools for reading, writing, and searching files, " +
		"executing shell commands, and managing projects."
)

var (
	// ErrNilConfig indicates that a nil config was passed to NewFactory.
	ErrNilConfig = errors.New("scaffold: config must not be nil")

	// ErrNilRegistry indicates that a nil registry was passed to NewFactory.
	ErrNilRegistry = errors.New("scaffold: registry must not be nil")

	// ErrUnknownAgentType indicates an unrecognized agent type was requested.
	ErrUnknownAgentType = errors.New("scaffold: unknown agent type")
)

// Factory compiles agent specifications from configuration, tool registry,
// and provider connections. It produces immutable Spec values.
type Factory struct {
	config    *config.V2
	registry  *tools.Registry
	providers map[string]llm.Provider
}

// NewFactory creates a Factory for compiling agent specifications.
// Both config and registry are required; providers may be nil.
func NewFactory(cfg *config.V2, registry *tools.Registry, providers map[string]llm.Provider) (*Factory, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if registry == nil {
		return nil, ErrNilRegistry
	}

	return &Factory{
		config:    cfg,
		registry:  registry,
		providers: providers,
	}, nil
}

// Compile produces an immutable Spec for the named agent type.
// Supported types: "main" and all builtin subagent names.
func (f *Factory) Compile(agentType string) (*Spec, error) {
	if agentType == AgentTypeMain {
		return f.compileMain()
	}

	// Check if it's a known subagent type.
	for _, sub := range subagent.Builtins() {
		if sub.Name == agentType {
			return f.compileSubagent(sub), nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrUnknownAgentType, agentType)
}

// compileMain assembles a Spec for the main agent with full tool access.
func (f *Factory) compileMain() (*Spec, error) {
	schemas := f.registry.ListSchemas()

	spec := &Spec{
		SystemPrompt:  defaultSystemPrompt,
		ToolSchemas:   schemas,
		AllowedTools:  nil, // Main agent has access to all tools.
		Providers:     f.providers,
		SubagentSpecs: nil,
		IsSubagent:    false,
		Config: SpecConfig{
			MaxTurns:    f.config.Agent.MaxTurns,
			Timeout:     f.config.Agent.Timeout,
			Temperature: f.config.LLM.Temperature,
			MaxTokens:   f.config.LLM.MaxTokens,
		},
	}

	return spec, nil
}

// compileSubagent assembles a Spec for a subagent with filtered tool access.
// Only tools in the subagent's AllowedTools list are included in ToolSchemas.
func (f *Factory) compileSubagent(sub *subagent.Spec) *Spec {
	allSchemas := f.registry.ListSchemas()
	filtered := make([]tools.ToolSchema, 0, len(sub.AllowedTools))

	for _, schema := range allSchemas {
		if slices.Contains(sub.AllowedTools, schema.Function.Name) {
			filtered = append(filtered, schema)
		}
	}

	return &Spec{
		SystemPrompt: sub.SystemPrompt,
		ToolSchemas:  filtered,
		AllowedTools: sub.AllowedTools,
		Providers:    f.providers,
		IsSubagent:   true,
		Config: SpecConfig{
			MaxTurns:    sub.MaxIterations,
			Temperature: f.config.LLM.Temperature,
			MaxTokens:   f.config.LLM.MaxTokens,
		},
	}
}
