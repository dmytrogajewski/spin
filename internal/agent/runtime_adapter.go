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
	result, err := a.executor.Execute(ctx, cmd, asExecuteOptions(opts))

	cmdResult := result.ToCommandResult()

	if err != nil {
		cmdResult.Error = err

		return cmdResult, err
	}

	return cmdResult, nil
}

// asExecuteOptions extracts *ExecuteOptions from an untyped opts value.
func asExecuteOptions(opts any) *ExecuteOptions {
	if opts == nil {
		return nil
	}

	if eOpts, ok := opts.(*ExecuteOptions); ok {
		return eOpts
	}

	return nil
}
