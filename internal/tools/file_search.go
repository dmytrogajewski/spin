package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/filesearch"
)

// FileSearchTool implements file search functionality with fuzzy matching.
type FileSearchTool struct {
	workspaceRoot string
	searcher      *filesearch.Searcher
	mu            sync.RWMutex
}

// NewFileSearchTool creates a new file search tool.
func NewFileSearchTool(workspaceRoot string) *FileSearchTool {
	return &FileSearchTool{
		workspaceRoot: workspaceRoot,
	}
}

func (t *FileSearchTool) Name() string {
	return "file_search"
}

func (t *FileSearchTool) Description() string {
	return "Search for files in the workspace using fuzzy matching with .gitignore support"
}

func (t *FileSearchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"query": {
						Type:        "string",
						Description: "The search query (fuzzy matched against file paths)",
					},
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of results to return (default: 10)",
					},
				},
				Required: []string{"query"},
			},
		},
	}
}

func (t *FileSearchTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	// Extract query parameter
	query, err := params.GetString("query")
	if err != nil || query == "" {
		return ToolResult{
			Success: false,
			Error:   "query parameter must be a non-empty string",
		}, nil
	}

	// Extract workspace_root parameter (optional)
	workspaceRoot := t.workspaceRoot
	if customRoot, err := params.GetString("workspace_root"); err == nil && customRoot != "" {
		workspaceRoot = customRoot
	}

	// Extract limit parameter (optional, default 10)
	limit := params.GetIntOr("limit", 10)

	// Get or create searcher for this workspace
	searcher, err := t.getOrCreateSearcher(workspaceRoot)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create searcher: %v", err),
		}, nil
	}

	// Index if not already indexed
	if !searcher.IsIndexed() {
		if err := searcher.IndexAsync(ctx); err != nil {
			return ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to index workspace: %v", err),
			}, nil
		}
	}

	// Search
	matches := searcher.Search(query, limit)

	// Format output
	var output strings.Builder
	if len(matches) == 0 {
		output.WriteString(fmt.Sprintf("No files found matching '%s'\n", query))
	} else {
		output.WriteString(fmt.Sprintf("Found %d file(s) matching '%s':\n\n", len(matches), query))
		for i, match := range matches {
			output.WriteString(fmt.Sprintf("%d. %s (score: %d)\n", i+1, match.Path, match.Score))
		}
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// getOrCreateSearcher returns the searcher for the given workspace, creating it if needed.
func (t *FileSearchTool) getOrCreateSearcher(workspaceRoot string) (*filesearch.Searcher, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If searcher exists and matches workspace, return it
	if t.searcher != nil && t.workspaceRoot == workspaceRoot {
		return t.searcher, nil
	}

	// Create new searcher
	searcher, err := filesearch.NewSearcher(workspaceRoot)
	if err != nil {
		return nil, err
	}

	// Update state
	t.searcher = searcher
	t.workspaceRoot = workspaceRoot

	return searcher, nil
}
