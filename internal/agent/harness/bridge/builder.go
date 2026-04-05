package bridge

import (
	"fmt"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	agenttool "github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Config holds dependencies for building a harness Executor.
type Config struct {
	Spec        *scaffold.Spec
	LLMCaller   *caller.LLMCaller
	Registry    *tools.Registry
	Runtime     *agenttool.Runtime
	Logger      *slog.Logger
	Guards      []harness.Guard
	Middlewares []harness.Middleware
	HarnessOpts []harness.Option
}

// BuildExecutor creates a harness.Executor from the existing agent infrastructure.
func BuildExecutor(cfg Config) (*harness.Executor, error) {
	callParams := agent.DefaultCallParams()

	callerBridge := NewCallerBridge(cfg.LLMCaller, cfg.Registry, callParams)
	dispatcherBridge := NewDispatcherBridge(cfg.Runtime)

	opts := append([]harness.Option{harness.WithRegistry(cfg.Registry)}, cfg.HarnessOpts...)

	exec, err := harness.NewExecutor(
		cfg.Spec,
		callerBridge,
		dispatcherBridge,
		cfg.Guards,
		cfg.Middlewares,
		cfg.Logger,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("build harness executor: %w", err)
	}

	return exec, nil
}
