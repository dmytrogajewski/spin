package runtime

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/security"
	shellpkg "github.com/dmytrogajewski/spin/internal/shell"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ShellContextAdapter adapts shell.Context to tools.ShellContext interface.
type ShellContextAdapter struct {
	ctx *shellpkg.Context
}

// NewShellContextAdapter creates a new adapter for shell.Context to tools.ShellContext.
func NewShellContextAdapter(ctx *shellpkg.Context) tools.ShellContext {
	if ctx == nil {
		return nil
	}

	return &ShellContextAdapter{ctx: ctx}
}

func (a *ShellContextAdapter) GetWorkingDirectory() string {
	return a.ctx.GetWorkingDirectory()
}

func (a *ShellContextAdapter) GetEnvironmentVars() map[string]string {
	return a.ctx.GetEnvironmentVars()
}

func (a *ShellContextAdapter) GetContextInfo() tools.ShellContextInfo {
	return &shellContextInfoAdapter{info: a.ctx.GetContextInfo()}
}

func (a *ShellContextAdapter) IsShellCommand(command string) bool {
	return a.ctx.IsShellCommand(command)
}

type shellContextInfoAdapter struct {
	info shellpkg.ContextInfo
}

func (a *shellContextInfoAdapter) IsShellEnabled() bool {
	return a.info.IsShellEnabled()
}

func (a *shellContextInfoAdapter) GetShell() string {
	return a.info.GetShell()
}

func (a *shellContextInfoAdapter) GetShellPath() string {
	return a.info.GetShellPath()
}

func (a *shellContextInfoAdapter) GetShellEnv() map[string]string {
	return a.info.GetShellEnv()
}

// ValidatorAdapter adapts security.Validator to tools.CommandValidator interface.
type ValidatorAdapter struct {
	validator *security.Validator
}

// NewValidatorAdapter creates a new adapter for security.Validator to tools.CommandValidator.
func NewValidatorAdapter(v *security.Validator) tools.CommandValidator {
	if v == nil {
		return nil
	}

	return &ValidatorAdapter{validator: v}
}

func (a *ValidatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
	// Convert tools.CommandInfo to *security.Command.
	secCmd := &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}

	result, err := a.validator.Classify(secCmd)
	if err != nil {
		return nil, err
	}

	return &validationResultAdapter{result: result}, nil
}

// validationResultAdapter adapts security.ValidationResult to tools.ValidationResult interface.
type validationResultAdapter struct {
	result *security.ValidationResult
}

func (a *validationResultAdapter) GetClassification() int {
	return int(a.result.Classification)
}

func (a *validationResultAdapter) GetReason() string {
	return a.result.Reason
}

// ExecutorAdapter adapts runtime.CommandExecutor to tools.CommandExecutor interface.
// Used by builtin runtime.
type ExecutorAdapter struct {
	executor CommandExecutor
}

func (a *ExecutorAdapter) Execute(ctx context.Context, cmd tools.CommandInfo, opts any) (tools.ExecutionResult, error) {
	secCmd := &security.Command{
		Program: cmd.GetProgram(),
		Args:    cmd.GetArgs(),
		Raw:     cmd.GetRaw(),
		WorkDir: cmd.GetWorkDir(),
	}

	result, err := a.executor.Execute(ctx, secCmd, opts)
	if err != nil {
		return &executionResultAdapter{result: result}, err
	}

	return &executionResultAdapter{result: result}, nil
}

type executionResultAdapter struct {
	result *CommandResult
}

func (a *executionResultAdapter) GetStdout() string {
	if a.result == nil {
		return ""
	}

	return a.result.Stdout
}

func (a *executionResultAdapter) GetStderr() string {
	if a.result == nil {
		return ""
	}

	return a.result.Stderr
}

func (a *executionResultAdapter) GetExitCode() int {
	if a.result == nil {
		return -1
	}

	return a.result.ExitCode
}

func (a *executionResultAdapter) GetMetadata() map[string]any {
	return nil
}
