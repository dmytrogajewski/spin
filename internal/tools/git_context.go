package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/git"
)

// Git context display constants.
const (
	commitHashDisplayLen  = 8
	maxUntrackedFilesList = 20
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

const gitContextName = "git_context"

// Name implements the Name operation.
func (t *GitContextTool) Name() string {
	return gitContextName
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
	workspaceRoot := resolveWorkspaceRoot(t.workspaceRoot, params)

	// Discover git repository.
	repo, err := git.Discover(ctx, workspaceRoot)
	if err != nil {
		// Gracefully handle non-git directories.
		return NewToolResult(fmt.Sprintf("Not a Git repository: %v\n", err)), nil
	}

	var output strings.Builder
	output.WriteString("Git Repository Context:\n")
	output.WriteString("======================\n\n")

	// Get status (includes branch info).
	status, err := repo.Status(ctx)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to get git status: %w", err)), nil
	}

	// Branch info.
	fmt.Fprintf(&output, "Branch: %s\n", status.Branch)

	if status.RemoteBranch != "" {
		fmt.Fprintf(&output, "Remote: %s\n", status.RemoteBranch)
		fmt.Fprintf(&output, "Ahead: %d, Behind: %d\n", status.Ahead, status.Behind)
	}

	if status.Detached {
		output.WriteString("(detached HEAD)\n")
	}

	// Commit hash.
	if status.Hash != "" {
		hashLen := min(len(status.Hash), commitHashDisplayLen)

		fmt.Fprintf(&output, "Commit: %s\n", status.Hash[:hashLen])
	}

	// File status.
	fmt.Fprintf(&output, "\nModified files: %d\n", len(status.ModifiedFiles))
	fmt.Fprintf(&output, "Untracked files: %d\n", len(status.UntrackedFiles))

	if len(status.ModifiedFiles) > 0 {
		output.WriteString("\nModified:\n")

		for _, file := range status.ModifiedFiles {
			fmt.Fprintf(&output, "  - %s (%s)\n", file.Path, file.Worktree)
		}
	}

	if len(status.UntrackedFiles) > 0 && len(status.UntrackedFiles) < maxUntrackedFilesList {
		output.WriteString("\nUntracked:\n")

		for _, file := range status.UntrackedFiles {
			fmt.Fprintf(&output, "  - %s\n", file)
		}
	}

	return NewToolResult(output.String()), nil
}
