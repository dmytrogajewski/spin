package conversation

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// validatorAdapter adapts security.SecurityService to tools.CommandValidator interface.
type validatorAdapter struct {
	securityService *security.SecurityService
}

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

func (a *shellContextAdapter) GetWorkingDirectory() string {
	return a.shellCtx.GetWorkingDirectory()
}

func (a *shellContextAdapter) GetEnvironmentVars() map[string]string {
	return a.shellCtx.GetEnvironmentVars()
}

func (a *shellContextAdapter) GetContextInfo() tools.ShellContextInfo {
	return a.shellCtx.GetContextInfo()
}

func (a *shellContextAdapter) IsShellCommand(c string) bool {
	return a.shellCtx.IsShellCommand(c)
}

// executorAdapter adapts agent.Executor to tools.CommandExecutor interface.
type executorAdapter struct {
	executor *agent.Executor
}

func (a *executorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, _ interface{}) (tools.ExecutionResult, error) {
	return a.executor.Execute(ctx, &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}, nil)
}
