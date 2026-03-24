package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/git"
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
	errOperationRequired        = errors.New("operation parameter is required")
	errUnknownGitOperation      = errors.New("unknown git operation")
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

// gitOperationHandler is the function signature for git operation handlers.
type gitOperationHandler func(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error)

// gitOperationHandlers maps operation names to their handler functions.
var gitOperationHandlers = map[string]gitOperationHandler{
	"stage":         handleGitStage,
	"commit":        handleGitCommit,
	"push":          handleGitPush,
	"pull":          handleGitPull,
	"create_branch": handleGitCreateBranch,
	"switch_branch": handleGitSwitchBranch,
	"list_branches": handleGitListBranches,
	"list_remotes":  handleGitListRemotes,
	"get_status":    handleGitStatus,
	"get_diff":      handleGitDiff,
	"get_log":       handleGitLog,
}

// Handler functions.

func handleGitStage(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	filePath := params.GetStringOr("file_path", "")
	if filePath == "" {
		return NewToolError(errFilePathRequired), nil
	}

	err := t.gitIntegration.StageFile(ctx, filePath)
	if err != nil {
		return ErrToResultf("Failed to stage file: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Staged file: %s", filePath)), nil
}

func handleGitCommit(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	message := params.GetStringOr("message", "")
	if message == "" {
		return NewToolError(errMessageRequired), nil
	}

	err := t.gitIntegration.Commit(ctx, message)
	if err != nil {
		return ErrToResultf("Failed to commit: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Committed: %s", message)), nil
}

func handleGitPush(ctx context.Context, t *GitOperationTool, _ ToolParameters) (ToolResult, error) {
	err := t.gitIntegration.Push(ctx)
	if err != nil {
		return ErrToResultf("Failed to push: %v", err)
	}

	return NewToolResult("Pushed changes to remote"), nil
}

func handleGitPull(ctx context.Context, t *GitOperationTool, _ ToolParameters) (ToolResult, error) {
	err := t.gitIntegration.Pull(ctx)
	if err != nil {
		return ErrToResultf("Failed to pull: %v", err)
	}

	return NewToolResult("Pulled changes from remote"), nil
}

func handleGitCreateBranch(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	branchName := params.GetStringOr("branch_name", "")
	if branchName == "" {
		return NewToolError(errBranchNameRequiredCreate), nil
	}

	err := t.gitIntegration.CreateBranch(ctx, branchName)
	if err != nil {
		return ErrToResultf("Failed to create branch: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Created branch: %s", branchName)), nil
}

func handleGitSwitchBranch(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	branchName := params.GetStringOr("branch_name", "")
	if branchName == "" {
		return NewToolError(errBranchNameRequiredSwitch), nil
	}

	err := t.gitIntegration.SwitchBranch(ctx, branchName)
	if err != nil {
		return ErrToResultf("Failed to switch branch: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Switched to branch: %s", branchName)), nil
}

func handleGitListBranches(ctx context.Context, t *GitOperationTool, _ ToolParameters) (ToolResult, error) {
	branches, err := t.gitIntegration.ListBranches(ctx)
	if err != nil {
		return ErrToResultf("Failed to list branches: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Branches: %s", strings.Join(branches, ", "))), nil
}

func handleGitListRemotes(ctx context.Context, t *GitOperationTool, _ ToolParameters) (ToolResult, error) {
	remotes, err := t.gitIntegration.ListRemotes(ctx)
	if err != nil {
		return ErrToResultf("Failed to list remotes: %v", err)
	}

	return NewToolResult(fmt.Sprintf("Remotes: %s", strings.Join(remotes, ", "))), nil
}

func handleGitStatus(_ context.Context, t *GitOperationTool, _ ToolParameters) (ToolResult, error) {
	status := t.gitIntegration.GetStatus()
	if status == nil {
		return NewToolError(errGitStatusFailed), nil
	}

	var output strings.Builder
	fmt.Fprintf(&output, "Branch: %s\n", status.Branch)
	fmt.Fprintf(&output, "Modified: %d files\n", len(status.ModifiedFiles))
	fmt.Fprintf(&output, "Untracked: %d files\n", len(status.UntrackedFiles))
	fmt.Fprintf(&output, "Ahead: %d, Behind: %d\n", status.Ahead, status.Behind)

	return NewToolResult(output.String()), nil
}

func handleGitDiff(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	filePath := params.GetStringOr("file_path", "")

	diff, err := t.gitIntegration.GetDiff(ctx, filePath)
	if err != nil {
		return ErrToResultf("Failed to get diff: %v", err)
	}

	return NewToolResult(diff), nil
}

func handleGitLog(ctx context.Context, t *GitOperationTool, params ToolParameters) (ToolResult, error) {
	limit := params.GetIntOr("limit", defaultGitLogLimit)

	logs, err := t.gitIntegration.GetLog(ctx, limit)
	if err != nil {
		return ErrToResultf("Failed to get log: %v", err)
	}

	var output strings.Builder
	for _, log := range logs {
		fmt.Fprintf(&output, "%s: %s\n", log.Hash[:7], log.Message)
	}

	return NewToolResult(output.String()), nil
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

	// Extract operation.
	operation := params.GetStringOr("operation", "")
	if operation == "" {
		return NewToolError(errOperationRequired), nil
	}

	// Get handler from map.
	handler, exists := gitOperationHandlers[operation]
	if !exists {
		return NewToolError(fmt.Errorf("unknown operation: %s: %w", operation, errUnknownGitOperation)), nil
	}

	// Execute handler.
	return handler(ctx, t, params)
}
