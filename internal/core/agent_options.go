package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// AgentOption is a functional option for configuring an Agent.
type AgentOption func(*Agent) error

// WithMaxTurns sets the maximum number of agent turns.
func WithMaxTurns(maxTurns int) AgentOption {
	return func(a *Agent) error {
		if maxTurns <= 0 {
			return fmt.Errorf("max turns must be positive, got %d", maxTurns)
		}
		a.config.MaxTurns = maxTurns
		return nil
	}
}

// WithAgentTimeout sets the agent execution timeout.
func WithAgentTimeout(timeout time.Duration) AgentOption {
	return func(a *Agent) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive, got %v", timeout)
		}
		a.config.Timeout = timeout
		return nil
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(temperature float64) AgentOption {
	return func(a *Agent) error {
		if temperature < 0 || temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2, got %f", temperature)
		}
		a.config.Temperature = temperature
		return nil
	}
}

// WithMaxTokens sets the maximum tokens per LLM call.
func WithMaxTokens(maxTokens int) AgentOption {
	return func(a *Agent) error {
		if maxTokens <= 0 {
			return fmt.Errorf("max tokens must be positive, got %d", maxTokens)
		}
		a.config.MaxTokens = maxTokens
		return nil
	}
}

// WithRequireApproval sets whether dangerous commands require approval.
func WithRequireApproval(require bool) AgentOption {
	return func(a *Agent) error {
		a.config.RequireApproval = require
		return nil
	}
}

// WithApprovalHandler sets the approval handler for the agent.
// The handler is called when a command requires user approval.
// If no handler is set, commands requiring approval are automatically denied.
func WithApprovalHandler(handler ApprovalHandler) AgentOption {
	return func(a *Agent) error {
		a.approvalHandler = handler
		return nil
	}
}

// WithPatternDetector sets the pattern detector for advanced cycle detection.
func WithPatternDetector(pd *cycle.PatternDetector) AgentOption {
	return func(a *Agent) error {
		a.patternDetector = pd
		return nil
	}
}

// WithToolRegistry merges a custom tool registry with the agent's default tools.
// Custom tools will override default tools with the same name.
// This ensures default tools (execute_command, get_context, etc.) are always available.
func WithToolRegistry(registry *tools.Registry) AgentOption {
	return func(a *Agent) error {
		if registry == nil {
			return errors.New("tool registry cannot be nil")
		}

		// Merge custom tools into the agent's existing registry
		// Custom tools override defaults with the same name
		for _, tool := range registry.List() {
			if err := a.toolRegistry.RegisterOrReplace(tool); err != nil {
				return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
			}
		}

		return nil
	}
}

// WithTaskRegistry sets a custom task registry for the agent.
// This replaces the default registry with all built-in modes (regular, review, compact, planning).
// Use this option to provide custom task modes or override default behavior.
//
// Example:
//
//	customRegistry := task.NewRegistry()
//	customRegistry.Register("custom", myCustomTask)
//	agent := NewAgent(llm, exec, val, ctx, emitter, WithTaskRegistry(customRegistry))
func WithTaskRegistry(registry *task.Registry) AgentOption {
	return func(a *Agent) error {
		if registry == nil {
			return errors.New("task registry cannot be nil")
		}
		a.taskRegistry = registry
		return nil
	}
}
