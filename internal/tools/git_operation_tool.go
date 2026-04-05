package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/git"
	"github.com/dmytrogajewski/spin/internal/tools/dispatch"
)

const defaultGitLogLimit = 10

// Sentinel errors for git operations (unexported — internal use only).
var (
	errFilePathRequired         = errors.New("file_path is required for stage operation")
	errMessageRequired          = errors.New("message is required for commit operation")
	errBranchNameRequiredCreate = errors.New("branch_name is required for create_branch operation")
	errBranchNameRequiredSwitch = errors.New("branch_name is required for switch_branch operation")
	errGitStatusFailed          = errors.New("failed to get git status")
	errNotGitRepository         = errors.New("not a git repository or git integration not available")
	errDispatchFailed           = errors.New("dispatch failed")
)

// GitOperationTool implements Git operations using Integration.
type GitOperationTool struct {
	gitIntegration *git.Integration
}

// NewGitOperationTool creates a new git operation tool.
func NewGitOperationTool(gitIntegration *git.Integration) *GitOperationTool {
	return &GitOperationTool{
		gitIntegration: gitIntegration,
	}
}

// gitDispatcher routes git operations to their handlers.
var gitDispatcher = buildGitDispatcher()

func buildGitDispatcher() *dispatch.Dispatcher[*GitOperationTool] {
	d := dispatch.New[*GitOperationTool]()
	d.Register("stage", handleGitStage)
	d.Register("commit", handleGitCommit)
	d.Register("push", handleGitPush)
	d.Register("pull", handleGitPull)
	d.Register("create_branch", handleGitCreateBranch)
	d.Register("switch_branch", handleGitSwitchBranch)
	d.Register("list_branches", handleGitListBranches)
	d.Register("list_remotes", handleGitListRemotes)
	d.Register("get_status", handleGitStatus)
	d.Register("get_diff", handleGitDiff)
	d.Register("get_log", handleGitLog)

	return d
}

// Handler functions.

func handleGitStage(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	filePath := params.GetStringOr("file_path", "")
	if filePath == "" {
		return dispatch.Errorf("%v", errFilePathRequired)
	}

	err := t.gitIntegration.StageFile(ctx, filePath)
	if err != nil {
		return dispatch.Errorf("Failed to stage file: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Staged file: %s", filePath))
}

func handleGitCommit(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	message := params.GetStringOr("message", "")
	if message == "" {
		return dispatch.Errorf("%v", errMessageRequired)
	}

	err := t.gitIntegration.Commit(ctx, message)
	if err != nil {
		return dispatch.Errorf("Failed to commit: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Committed: %s", message))
}

func handleGitPush(ctx context.Context, t *GitOperationTool, _ dispatch.Params) (dispatch.Result, error) {
	err := t.gitIntegration.Push(ctx)
	if err != nil {
		return dispatch.Errorf("Failed to push: %v", err)
	}

	return dispatch.OK("Pushed changes to remote")
}

func handleGitPull(ctx context.Context, t *GitOperationTool, _ dispatch.Params) (dispatch.Result, error) {
	err := t.gitIntegration.Pull(ctx)
	if err != nil {
		return dispatch.Errorf("Failed to pull: %v", err)
	}

	return dispatch.OK("Pulled changes from remote")
}

func handleGitCreateBranch(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	branchName := params.GetStringOr("branch_name", "")
	if branchName == "" {
		return dispatch.Errorf("%v", errBranchNameRequiredCreate)
	}

	err := t.gitIntegration.CreateBranch(ctx, branchName)
	if err != nil {
		return dispatch.Errorf("Failed to create branch: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Created branch: %s", branchName))
}

func handleGitSwitchBranch(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	branchName := params.GetStringOr("branch_name", "")
	if branchName == "" {
		return dispatch.Errorf("%v", errBranchNameRequiredSwitch)
	}

	err := t.gitIntegration.SwitchBranch(ctx, branchName)
	if err != nil {
		return dispatch.Errorf("Failed to switch branch: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Switched to branch: %s", branchName))
}

func handleGitListBranches(ctx context.Context, t *GitOperationTool, _ dispatch.Params) (dispatch.Result, error) {
	branches, err := t.gitIntegration.ListBranches(ctx)
	if err != nil {
		return dispatch.Errorf("Failed to list branches: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Branches: %s", strings.Join(branches, ", ")))
}

func handleGitListRemotes(ctx context.Context, t *GitOperationTool, _ dispatch.Params) (dispatch.Result, error) {
	remotes, err := t.gitIntegration.ListRemotes(ctx)
	if err != nil {
		return dispatch.Errorf("Failed to list remotes: %v", err)
	}

	return dispatch.OK(fmt.Sprintf("Remotes: %s", strings.Join(remotes, ", ")))
}

func handleGitStatus(_ context.Context, t *GitOperationTool, _ dispatch.Params) (dispatch.Result, error) {
	status := t.gitIntegration.GetStatus()
	if status == nil {
		return dispatch.Errorf("%v", errGitStatusFailed)
	}

	var output strings.Builder
	fmt.Fprintf(&output, "Branch: %s\n", status.Branch)
	fmt.Fprintf(&output, "Modified: %d files\n", len(status.ModifiedFiles))
	fmt.Fprintf(&output, "Untracked: %d files\n", len(status.UntrackedFiles))
	fmt.Fprintf(&output, "Ahead: %d, Behind: %d\n", status.Ahead, status.Behind)

	return dispatch.OK(output.String())
}

func handleGitDiff(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	filePath := params.GetStringOr("file_path", "")

	diff, err := t.gitIntegration.GetDiff(ctx, filePath)
	if err != nil {
		return dispatch.Errorf("Failed to get diff: %v", err)
	}

	return dispatch.OK(diff)
}

func handleGitLog(ctx context.Context, t *GitOperationTool, params dispatch.Params) (dispatch.Result, error) {
	limit := params.GetIntOr("limit", defaultGitLogLimit)

	logs, err := t.gitIntegration.GetLog(ctx, limit)
	if err != nil {
		return dispatch.Errorf("Failed to get log: %v", err)
	}

	var output strings.Builder
	for _, log := range logs {
		fmt.Fprintf(&output, "%s: %s\n", log.Hash[:7], log.Message)
	}

	return dispatch.OK(output.String())
}

// Name implements the Name operation.
func (t *GitOperationTool) Name() string {
	return "git_operation"
}

// Description implements the Description operation.
func (t *GitOperationTool) Description() string {
	return "Perform Git operations like stage, commit, push, pull, branch management"
}

// Schema implements the Schema operation.
func (t *GitOperationTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"operation": {
						Type: "string",
						Description: "Git operation: stage, commit, push, pull, create_branch, " +
							"switch_branch, list_branches, list_remotes, get_status, get_diff, get_log",
						Enum: []string{
							"stage", "commit", "push", "pull", "create_branch",
							"switch_branch", "list_branches", "list_remotes",
							"get_status", "get_diff", "get_log",
						},
					},
					"file_path": {
						Type:        "string",
						Description: "File path for stage operation (optional)",
					},
					"message": {
						Type:        "string",
						Description: "Commit message for commit operation (optional)",
					},
					"branch_name": {
						Type:        "string",
						Description: "Branch name for create_branch or switch_branch operations (optional)",
					},
					"limit": {
						Type:        "integer",
						Description: "Limit for get_log operation (optional, default: 10)",
					},
				},
				Required: []string{"operation"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *GitOperationTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	// Validate git integration.
	if t.gitIntegration == nil || !t.gitIntegration.IsRepository() {
		return NewToolError(errNotGitRepository), nil
	}

	// Dispatch to the appropriate handler.
	result, err := gitDispatcher.Dispatch(ctx, t, "operation", params)
	if err != nil {
		return ToolResult{}, err
	}

	if result.IsError {
		return NewToolError(fmt.Errorf("%w: %s", errDispatchFailed, result.Content)), nil
	}

	return NewToolResult(result.Content), nil
}
