package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/git"
)

// GitContextTool implements Git repository context retrieval.
type GitContextTool struct {
	workspaceRoot string
}

// NewGitContextTool creates a new git context tool.
func NewGitContextTool(workspaceRoot string) *GitContextTool {
	return &GitContextTool{
		workspaceRoot: workspaceRoot,
	}
}

// Name implements the Name operation.
func (t *GitContextTool) Name() string {
	return "git_context"
}

// Description implements the Description operation.
func (t *GitContextTool) Description() string {
	return "Get Git repository context including branch, status, and modifications"
}

// Schema implements the Schema operation.
func (t *GitContextTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"include_diff": {
						Type:        "boolean",
						Description: "If true, include diff summary (default: false)",
					},
				},
				Required: []string{},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *GitContextTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	// Get workspace root.
	workspaceRoot := t.workspaceRoot
	root, err := params.GetString("workspace_root")
	if err == nil && root != "" {
		workspaceRoot = root
	}

	// Discover git repository.
	repo, err := git.Discover(ctx, workspaceRoot)
	if err != nil {
		// Gracefully handle non-git directories.
		return ToolResult{
			Success: true,
			Output:  fmt.Sprintf("Not a Git repository: %v\n", err),
		}, nil
	}

	var output strings.Builder
	output.WriteString("Git Repository Context:\n")
	output.WriteString("======================\n\n")

	// Get status (includes branch info).
	status, err := repo.Status(ctx)
	if err != nil {
		return ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to get git status: %v\n", err),
		}, nil
	}

	// Branch info.
	output.WriteString(fmt.Sprintf("Branch: %s\n", status.Branch))

	if status.RemoteBranch != "" {
		output.WriteString(fmt.Sprintf("Remote: %s\n", status.RemoteBranch))
		output.WriteString(fmt.Sprintf("Ahead: %d, Behind: %d\n", status.Ahead, status.Behind))
	}

	if status.Detached {
		output.WriteString("(detached HEAD)\n")
	}

	// Commit hash.
	if status.Hash != "" {
		hashLen := min(len(status.Hash), 8)

		output.WriteString(fmt.Sprintf("Commit: %s\n", status.Hash[:hashLen]))
	}

	// File status.
	output.WriteString(fmt.Sprintf("\nModified files: %d\n", len(status.ModifiedFiles)))
	output.WriteString(fmt.Sprintf("Untracked files: %d\n", len(status.UntrackedFiles)))

	if len(status.ModifiedFiles) > 0 {
		output.WriteString("\nModified:\n")

		for _, file := range status.ModifiedFiles {
			output.WriteString(fmt.Sprintf("  - %s (%s)\n", file.Path, file.Worktree))
		}
	}

	if len(status.UntrackedFiles) > 0 && len(status.UntrackedFiles) < 20 {
		output.WriteString("\nUntracked:\n")

		for _, file := range status.UntrackedFiles {
			output.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}
