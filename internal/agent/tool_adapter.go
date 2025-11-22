package agent

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ToolExecutorAdapter adapts agent.Executor to tools.CommandExecutor interface.
type ToolExecutorAdapter struct {
	executor *Executor
}

// NewToolExecutorAdapter creates a new adapter for agent.Executor to tools.CommandExecutor.
func NewToolExecutorAdapter(exec *Executor) tools.CommandExecutor {
	if exec == nil {
		return nil
	}
	return &ToolExecutorAdapter{executor: exec}
}

// Execute implements tools.CommandExecutor interface.
func (a *ToolExecutorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, opts interface{}) (tools.ExecutionResult, error) {
	// Convert tools.CommandInfo to *security.Command
	secCmd := &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}

	// Convert opts if provided
	var execOpts *ExecuteOptions
	if opts != nil {
		if eOpts, ok := opts.(*ExecuteOptions); ok {
			execOpts = eOpts
		}
	}

	// Execute using agent.Executor
	result, err := a.executor.Execute(ctx, secCmd, execOpts)
	if err != nil {
		return nil, err
	}

	// Return result adapted to tools.ExecutionResult
	return &toolExecutionResult{result: result}, nil
}

// toolExecutionResult adapts agent.Result to tools.ExecutionResult interface.
type toolExecutionResult struct {
	result *Result
}

func (r *toolExecutionResult) GetStdout() string {
	if r.result == nil {
		return ""
	}
	return r.result.Stdout
}

func (r *toolExecutionResult) GetStderr() string {
	if r.result == nil {
		return ""
	}
	return r.result.Stderr
}

func (r *toolExecutionResult) GetExitCode() int {
	if r.result == nil {
		return -1
	}
	return r.result.ExitCode
}

func (r *toolExecutionResult) GetMetadata() map[string]interface{} {
	return nil
}
