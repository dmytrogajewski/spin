package executor

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/safety"
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

// GetWorkingDirectory implements the GetWorkingDirectory operation.
func (a *ShellContextAdapter) GetWorkingDirectory() string {
	return a.ctx.GetWorkingDirectory()
}

// GetEnvironmentVars implements the GetEnvironmentVars operation.
func (a *ShellContextAdapter) GetEnvironmentVars() map[string]string {
	return a.ctx.GetEnvironmentVars()
}

// GetContextInfo implements the GetContextInfo operation.
func (a *ShellContextAdapter) GetContextInfo() tools.ShellContextInfo {
	return &shellContextInfoAdapter{info: a.ctx.GetContextInfo()}
}

// IsShellCommand implements the IsShellCommand operation.
func (a *ShellContextAdapter) IsShellCommand(command string) bool {
	return a.ctx.IsShellCommand(command)
}

type shellContextInfoAdapter struct {
	info shellpkg.ContextInfo
}

// IsShellEnabled implements the IsShellEnabled operation.
func (a *shellContextInfoAdapter) IsShellEnabled() bool {
	return a.info.IsShellEnabled()
}

// GetShell implements the GetShell operation.
func (a *shellContextInfoAdapter) GetShell() string {
	return a.info.GetShell()
}

// GetShellPath implements the GetShellPath operation.
func (a *shellContextInfoAdapter) GetShellPath() string {
	return a.info.GetShellPath()
}

// GetShellEnv implements the GetShellEnv operation.
func (a *shellContextInfoAdapter) GetShellEnv() map[string]string {
	return a.info.GetShellEnv()
}

// ValidatorAdapter adapts safety.Validator to tools.CommandValidator interface.
type ValidatorAdapter struct {
	validator *safety.Validator
}

// NewValidatorAdapter creates a new adapter for safety.Validator to tools.CommandValidator.
func NewValidatorAdapter(v *safety.Validator) tools.CommandValidator {
	if v == nil {
		return nil
	}

	return &ValidatorAdapter{validator: v}
}

// Classify implements the Classify operation.
func (a *ValidatorAdapter) Classify(cmd tools.CommandInfo) (tools.ValidationResult, error) {
	// Convert tools.CommandInfo to *safety.Command.
	secCmd := &safety.Command{
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

// validationResultAdapter adapts safety.ValidationResult to tools.ValidationResult interface.
type validationResultAdapter struct {
	result *safety.ValidationResult
}

// GetClassification implements the GetClassification operation.
func (a *validationResultAdapter) GetClassification() int {
	return int(a.result.Classification)
}

// GetReason implements the GetReason operation.
func (a *validationResultAdapter) GetReason() string {
	return a.result.Reason
}

// TaskManagerAdapter adapts [BackgroundTaskManager] to [tools.TaskManager] interface.
type TaskManagerAdapter struct {
	mgr *BackgroundTaskManager
}

// NewTaskManagerAdapter creates a new adapter for BackgroundTaskManager to [tools.TaskManager].
func NewTaskManagerAdapter(mgr *BackgroundTaskManager) *TaskManagerAdapter {
	if mgr == nil {
		return nil
	}

	return &TaskManagerAdapter{mgr: mgr}
}

// List returns snapshots of all managed tasks.
func (a *TaskManagerAdapter) List(_ context.Context) []tools.TaskSnapshot {
	infos := a.mgr.List()
	snapshots := make([]tools.TaskSnapshot, len(infos))

	for idx, info := range infos {
		snapshots[idx] = tools.TaskSnapshot{
			ID:        info.ID,
			Command:   info.Command,
			Status:    tools.TaskStatus(info.State),
			StartedAt: info.StartedAt,
			ExitCode:  info.ExitCode,
		}
	}

	return snapshots
}

// GetOutput returns the last maxLines of output for a task.
func (a *TaskManagerAdapter) GetOutput(_ context.Context, taskID string, maxLines int) (string, error) {
	return a.mgr.GetOutput(taskID, maxLines)
}

// Kill terminates a running task.
func (a *TaskManagerAdapter) Kill(_ context.Context, taskID string) error {
	return a.mgr.Kill(taskID)
}

// Adapter adapts runtime.CommandExecutor to tools.CommandExecutor interface.
// Used by builtin runtime.
type Adapter struct {
	executor CommandExecutor
}

// Execute implements the Execute operation.
func (a *Adapter) Execute(ctx context.Context, cmd tools.CommandInfo, opts any) (tools.ExecutionResult, error) {
	secCmd := &safety.Command{
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

// GetStdout implements the GetStdout operation.
func (a *executionResultAdapter) GetStdout() string {
	if a.result == nil {
		return ""
	}

	return a.result.Stdout
}

// GetStderr implements the GetStderr operation.
func (a *executionResultAdapter) GetStderr() string {
	if a.result == nil {
		return ""
	}

	return a.result.Stderr
}

// GetExitCode implements the GetExitCode operation.
func (a *executionResultAdapter) GetExitCode() int {
	if a.result == nil {
		return -1
	}

	return a.result.ExitCode
}

// GetMetadata implements the GetMetadata operation.
func (a *executionResultAdapter) GetMetadata() map[string]any {
	return nil
}
