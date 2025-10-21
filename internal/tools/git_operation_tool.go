package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/git"
)

// GitOperationTool implements Git operations using GitIntegration.
type GitOperationTool struct {
	gitIntegration *git.GitIntegration
}

// NewGitOperationTool creates a new git operation tool.
func NewGitOperationTool(gitIntegration *git.GitIntegration) *GitOperationTool {
	return &GitOperationTool{
		gitIntegration: gitIntegration,
	}
}

// gitSuccessResult creates a successful tool result.
func gitSuccessResult(output string) ToolResult {
	return ToolResult{
		Success: true,
		Output:  output,
	}
}

// gitErrorResult creates an error tool result.
func gitErrorResult(msg string) ToolResult {
	return ToolResult{
		Success: false,
		Error:   msg,
	}
}

// gitOperationHandler is the function signature for git operation handlers.
type gitOperationHandler func(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error)

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

// Handler functions

func handleGitStage(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	filePath, _ := params["file_path"].(string)
	if filePath == "" {
		return gitErrorResult("file_path is required for stage operation"), nil
	}
	err := t.gitIntegration.StageFile(filePath)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to stage file: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Staged file: %s", filePath)), nil
}

func handleGitCommit(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	message, _ := params["message"].(string)
	if message == "" {
		return gitErrorResult("message is required for commit operation"), nil
	}
	err := t.gitIntegration.Commit(message)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to commit: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Committed: %s", message)), nil
}

func handleGitPush(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	err := t.gitIntegration.Push()
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to push: %v", err)), nil
	}
	return gitSuccessResult("Pushed changes to remote"), nil
}

func handleGitPull(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	err := t.gitIntegration.Pull()
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to pull: %v", err)), nil
	}
	return gitSuccessResult("Pulled changes from remote"), nil
}

func handleGitCreateBranch(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	branchName, _ := params["branch_name"].(string)
	if branchName == "" {
		return gitErrorResult("branch_name is required for create_branch operation"), nil
	}
	err := t.gitIntegration.CreateBranch(branchName)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to create branch: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Created branch: %s", branchName)), nil
}

func handleGitSwitchBranch(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	branchName, _ := params["branch_name"].(string)
	if branchName == "" {
		return gitErrorResult("branch_name is required for switch_branch operation"), nil
	}
	err := t.gitIntegration.SwitchBranch(branchName)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to switch branch: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Switched to branch: %s", branchName)), nil
}

func handleGitListBranches(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	branches, err := t.gitIntegration.ListBranches()
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to list branches: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Branches: %s", strings.Join(branches, ", "))), nil
}

func handleGitListRemotes(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	remotes, err := t.gitIntegration.ListRemotes()
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to list remotes: %v", err)), nil
	}
	return gitSuccessResult(fmt.Sprintf("Remotes: %s", strings.Join(remotes, ", "))), nil
}

func handleGitStatus(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	status := t.gitIntegration.GetStatus()
	if status == nil {
		return gitErrorResult("Failed to get Git status"), nil
	}
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Branch: %s\n", status.Branch))
	output.WriteString(fmt.Sprintf("Modified: %d files\n", len(status.ModifiedFiles)))
	output.WriteString(fmt.Sprintf("Untracked: %d files\n", len(status.UntrackedFiles)))
	output.WriteString(fmt.Sprintf("Ahead: %d, Behind: %d\n", status.Ahead, status.Behind))
	return gitSuccessResult(output.String()), nil
}

func handleGitDiff(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	filePath, _ := params["file_path"].(string)
	diff, err := t.gitIntegration.GetDiff(filePath)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to get diff: %v", err)), nil
	}
	return gitSuccessResult(diff), nil
}

func handleGitLog(ctx context.Context, t *GitOperationTool, params map[string]interface{}) (ToolResult, error) {
	limit := 10
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	}
	logs, err := t.gitIntegration.GetLog(limit)
	if err != nil {
		return gitErrorResult(fmt.Sprintf("Failed to get log: %v", err)), nil
	}
	var output strings.Builder
	for _, log := range logs {
		output.WriteString(fmt.Sprintf("%s: %s\n", log.Hash[:7], log.Message))
	}
	return gitSuccessResult(output.String()), nil
}

func (t *GitOperationTool) Name() string {
	return "git_operation"
}

func (t *GitOperationTool) Description() string {
	return "Perform Git operations like stage, commit, push, pull, branch management"
}

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
						Type:        "string",
						Description: "Git operation: stage, commit, push, pull, create_branch, switch_branch, list_branches, list_remotes, get_status, get_diff, get_log",
						Enum:        []string{"stage", "commit", "push", "pull", "create_branch", "switch_branch", "list_branches", "list_remotes", "get_status", "get_diff", "get_log"},
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

func (t *GitOperationTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Validate git integration
	if t.gitIntegration == nil || !t.gitIntegration.IsRepository() {
		return gitErrorResult("Not a Git repository or Git integration not available"), nil
	}

	// Extract operation
	operation, ok := params["operation"].(string)
	if !ok {
		return gitErrorResult("operation parameter is required"), nil
	}

	// Get handler from map
	handler, exists := gitOperationHandlers[operation]
	if !exists {
		return gitErrorResult(fmt.Sprintf("Unknown operation: %s", operation)), nil
	}

	// Execute handler
	return handler(ctx, t, params)
}
