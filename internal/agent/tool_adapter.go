package agent

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ToolExecutorAdapter adapts agent.Executor to tools.CommandExecutor interface.
type ToolExecutorAdapter struct {
	executor *Executor
}

// NewToolExecutorAdapter creates a new adapter for agent.Executor to tools.CommandExecutor.
func NewToolExecutorAdapter(exec *Executor) *ToolExecutorAdapter {
	if exec == nil {
		return nil
	}

	return &ToolExecutorAdapter{executor: exec}
}

// Execute implements tools.CommandExecutor interface.
func (a *ToolExecutorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, opts any) (tools.ExecutionResult, error) {
	secCmd := safety.CommandFrom(cmd)

	// Execute using agent.Executor.
	// *Result satisfies tools.ExecutionResult directly.
	result, err := a.executor.Execute(ctx, secCmd, asExecuteOptions(opts))
	if err != nil {
		return nil, err
	}

	return result, nil
}
