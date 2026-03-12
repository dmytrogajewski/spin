package agent

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent/runtime"
	"github.com/dmytrogajewski/spin/internal/security"
)

// executorRuntimeAdapter adapts agent.Executor to runtime.CommandExecutor interface.
type executorRuntimeAdapter struct {
	executor *Executor
}

// NewExecutorRuntimeAdapter creates an adapter for agent.Executor to runtime.CommandExecutor.
func NewExecutorRuntimeAdapter(exec *Executor) runtime.CommandExecutor {
	if exec == nil {
		return nil
	}

	return &executorRuntimeAdapter{executor: exec}
}

// Execute implements runtime.CommandExecutor interface.
func (a *executorRuntimeAdapter) Execute(ctx context.Context, cmd *security.Command, opts any) (*runtime.CommandResult, error) {
	var execOpts *ExecuteOptions

	if opts != nil {
		if eOpts, ok := opts.(*ExecuteOptions); ok {
			execOpts = eOpts
		}
	}

	result, err := a.executor.Execute(ctx, cmd, execOpts)
	if err != nil {
		// Convert agent.Result to runtime.CommandResult.
		return &runtime.CommandResult{
			Command:     result.Command,
			Stdout:      result.Stdout,
			Stderr:      result.Stderr,
			ExitCode:    result.ExitCode,
			Duration:    result.Duration,
			StartedAt:   result.StartedAt,
			CompletedAt: result.CompletedAt,
			Error:       err,
			Truncated:   result.Truncated,
		}, err
	}

	// Convert agent.Result to runtime.CommandResult.
	return &runtime.CommandResult{
		Command:     result.Command,
		Stdout:      result.Stdout,
		Stderr:      result.Stderr,
		ExitCode:    result.ExitCode,
		Duration:    result.Duration,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
		Error:       result.Error,
		Truncated:   result.Truncated,
	}, nil
}
