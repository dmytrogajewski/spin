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
	result, err := a.executor.Execute(ctx, secCmd, asExecuteOptions(opts))
	if err != nil {
		return nil, err
	}

	// Return result adapted to tools.ExecutionResult.
	return &toolExecutionResult{result: result}, nil
}

// toolExecutionResult adapts agent.Result to tools.ExecutionResult interface.
type toolExecutionResult struct {
	result *Result
}

// GetStdout implements the GetStdout operation.
func (r *toolExecutionResult) GetStdout() string {
	if r.result == nil {
		return ""
	}

	return r.result.Stdout
}

// GetStderr implements the GetStderr operation.
func (r *toolExecutionResult) GetStderr() string {
	if r.result == nil {
		return ""
	}

	return r.result.Stderr
}

// GetExitCode implements the GetExitCode operation.
func (r *toolExecutionResult) GetExitCode() int {
	if r.result == nil {
		return -1
	}

	return r.result.ExitCode
}

// GetMetadata implements the GetMetadata operation.
func (r *toolExecutionResult) GetMetadata() map[string]any {
	return nil
}
