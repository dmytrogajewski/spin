package agent

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

// executorRuntimeAdapter adapts agent.Executor to executor.CommandExecutor interface.
type executorRuntimeAdapter struct {
	executor *Executor
}

// NewExecutorRuntimeAdapter creates an adapter for agent.Executor to executor.CommandExecutor.
func NewExecutorRuntimeAdapter(exec *Executor) executor.CommandExecutor {
	if exec == nil {
		return nil
	}

	return &executorRuntimeAdapter{executor: exec}
}

// Execute implements executor.CommandExecutor interface.
func (a *executorRuntimeAdapter) Execute(ctx context.Context, cmd *safety.Command, opts any) (*executor.CommandResult, error) {
	var execOpts *ExecuteOptions

	if opts != nil {
		if eOpts, ok := opts.(*ExecuteOptions); ok {
			execOpts = eOpts
		}
	}

	result, err := a.executor.Execute(ctx, cmd, execOpts)
	if err != nil {
		// Convert agent.Result to executor.CommandResult.
		return &executor.CommandResult{
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

	// Convert agent.Result to executor.CommandResult.
	return &executor.CommandResult{
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
