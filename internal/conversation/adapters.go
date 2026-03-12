// Package conversation provides conversation management and adapters.
package conversation

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// validatorAdapter adapts security.Service to tools.CommandValidator interface.
type validatorAdapter struct {
	securityService *security.Service
}

// Classify implements the Classify operation.
func (a *validatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
	return a.securityService.ValidateCommand(&security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	})
}

// shellContextAdapter adapts shell.Context to tools.ShellContext interface.
type shellContextAdapter struct {
	shellCtx *shell.Context
}

// GetWorkingDirectory implements the GetWorkingDirectory operation.
func (a *shellContextAdapter) GetWorkingDirectory() string {
	return a.shellCtx.GetWorkingDirectory()
}

// GetEnvironmentVars implements the GetEnvironmentVars operation.
func (a *shellContextAdapter) GetEnvironmentVars() map[string]string {
	return a.shellCtx.GetEnvironmentVars()
}

// GetContextInfo implements the GetContextInfo operation.
func (a *shellContextAdapter) GetContextInfo() tools.ShellContextInfo {
	return a.shellCtx.GetContextInfo()
}

// IsShellCommand implements the IsShellCommand operation.
func (a *shellContextAdapter) IsShellCommand(c string) bool {
	return a.shellCtx.IsShellCommand(c)
}

// executorAdapter adapts agent.Executor to tools.CommandExecutor interface.
type executorAdapter struct {
	executor *agent.Executor
}

// Execute implements the Execute operation.
func (a *executorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, _ any) (tools.ExecutionResult, error) {
	return a.executor.Execute(ctx, &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}, nil)
}
