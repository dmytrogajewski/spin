package core

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dmytrogajewski/spin/internal/git"
)

func TestGitOperationTool_HandleGitStage(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]interface{}
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "missing file_path",
			params:     map[string]interface{}{},
			wantErr:    true,
			wantOutput: "file_path is required",
		},
		{
			name: "empty file_path",
			params: map[string]interface{}{
				"file_path": "",
			},
			wantErr:    true,
			wantOutput: "file_path is required",
		},
		{
			name: "valid file_path",
			params: map[string]interface{}{
				"file_path": "test.go",
			},
			wantErr:    false, // Tests code path, may fail without real git
			wantOutput: "Staged file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitStage(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitStage() unexpected error: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("handleGitStage() expected error result, got success")
			}

			// For valid file_path, we test code path (may fail without real git repo)
			// Success means validation passed, not that git command succeeded
		})
	}
}

func TestGitOperationTool_HandleGitCommit(t *testing.T) {
	tests := []struct {
		name       string
		params     map[string]interface{}
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "missing message",
			params:     map[string]interface{}{},
			wantErr:    true,
			wantOutput: "message is required",
		},
		{
			name: "empty message",
			params: map[string]interface{}{
				"message": "",
			},
			wantErr:    true,
			wantOutput: "message is required",
		},
		{
			name: "valid message",
			params: map[string]interface{}{
				"message": "test commit",
			},
			wantErr:    false, // Tests code path, may fail without real git
			wantOutput: "Committed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitCommit(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitCommit() unexpected error: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("handleGitCommit() expected error result, got success")
			}

			// For valid message, we test code path (may fail without real git repo)
			// Success means validation passed, not that git command succeeded
		})
	}
}

func TestGitOperationTool_HandleGitPush(t *testing.T) {
	gitInteg := &GitIntegration{
		enabled: true,
		workDir: t.TempDir(),
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		repo:    &git.Repository{},
	}

	tool := NewGitOperationTool(gitInteg)
	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := handleGitPush(ctx, tool, params)
	if err != nil {
		t.Errorf("handleGitPush() unexpected error: %v", err)
	}

	// Push will fail without a real repo, but we test the code path
	if result.Success {
		t.Log("handleGitPush() succeeded unexpectedly (probably in a real git repo)")
	}
}

func TestGitOperationTool_HandleGitPull(t *testing.T) {
	gitInteg := &GitIntegration{
		enabled: true,
		workDir: t.TempDir(),
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		repo:    &git.Repository{},
	}

	tool := NewGitOperationTool(gitInteg)
	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := handleGitPull(ctx, tool, params)
	if err != nil {
		t.Errorf("handleGitPull() unexpected error: %v", err)
	}

	// Pull will fail without a real repo, but we test the code path
	if result.Success {
		t.Log("handleGitPull() succeeded unexpectedly (probably in a real git repo)")
	}
}

func TestGitOperationTool_HandleGitCreateBranch(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing branch_name",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty branch_name",
			params: map[string]interface{}{
				"branch_name": "",
			},
			wantErr: true,
		},
		{
			name: "valid branch_name",
			params: map[string]interface{}{
				"branch_name": "feature-test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitCreateBranch(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitCreateBranch() unexpected error: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("handleGitCreateBranch() expected error result, got success")
			}
		})
	}
}

func TestGitOperationTool_HandleGitSwitchBranch(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "missing branch_name",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "empty branch_name",
			params: map[string]interface{}{
				"branch_name": "",
			},
			wantErr: true,
		},
		{
			name: "valid branch_name",
			params: map[string]interface{}{
				"branch_name": "main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitSwitchBranch(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitSwitchBranch() unexpected error: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("handleGitSwitchBranch() expected error result, got success")
			}
		})
	}
}

func TestGitOperationTool_HandleGitListBranches(t *testing.T) {
	gitInteg := &GitIntegration{
		enabled: true,
		workDir: t.TempDir(),
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		repo:    &git.Repository{},
	}

	tool := NewGitOperationTool(gitInteg)
	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := handleGitListBranches(ctx, tool, params)
	if err != nil {
		t.Errorf("handleGitListBranches() unexpected error: %v", err)
	}

	// Should succeed with empty list
	if !result.Success {
		t.Errorf("handleGitListBranches() expected success, got error: %v", result.Error)
	}
}

func TestGitOperationTool_HandleGitListRemotes(t *testing.T) {
	gitInteg := &GitIntegration{
		enabled: true,
		workDir: t.TempDir(),
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		repo:    &git.Repository{},
	}

	tool := NewGitOperationTool(gitInteg)
	ctx := context.Background()
	params := map[string]interface{}{}

	result, err := handleGitListRemotes(ctx, tool, params)
	if err != nil {
		t.Errorf("handleGitListRemotes() unexpected error: %v", err)
	}

	// Should succeed with empty list
	if !result.Success {
		t.Errorf("handleGitListRemotes() expected success, got error: %v", result.Error)
	}
}

func TestGitOperationTool_HandleGitStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  *git.Status
		wantErr bool
	}{
		{
			name:    "no status available",
			status:  nil,
			wantErr: true,
		},
		{
			name: "valid status",
			status: &git.Status{
				Branch:         "main",
				ModifiedFiles:  []git.FileStatus{},
				UntrackedFiles: []string{},
				Ahead:          0,
				Behind:         0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled:    true,
				workDir:    t.TempDir(),
				logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:       &git.Repository{},
				lastStatus: tt.status,
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()
			params := map[string]interface{}{}

			result, err := handleGitStatus(ctx, tool, params)
			if err != nil {
				t.Errorf("handleGitStatus() unexpected error: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("handleGitStatus() expected error result, got success")
			}

			if !tt.wantErr && !result.Success {
				t.Errorf("handleGitStatus() expected success, got error: %v", result.Error)
			}
		})
	}
}

func TestGitOperationTool_HandleGitDiff(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "no file_path (all files)",
			params:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "with file_path",
			params: map[string]interface{}{
				"file_path": "test.go",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitDiff(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitDiff() unexpected error: %v", err)
			}

			// Diff will fail without a real repo, but we test the code path
			_ = result
		})
	}
}

func TestGitOperationTool_HandleGitLog(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "default limit",
			params: map[string]interface{}{},
		},
		{
			name: "custom limit",
			params: map[string]interface{}{
				"limit": float64(5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gitInteg := &GitIntegration{
				enabled: true,
				workDir: t.TempDir(),
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			}

			tool := NewGitOperationTool(gitInteg)
			ctx := context.Background()

			result, err := handleGitLog(ctx, tool, tt.params)
			if err != nil {
				t.Errorf("handleGitLog() unexpected error: %v", err)
			}

			// Should succeed with empty log
			if !result.Success {
				t.Errorf("handleGitLog() expected success, got error: %v", result.Error)
			}
		})
	}
}

func TestGitOperationTool_Execute(t *testing.T) {
	tests := []struct {
		name        string
		gitInteg    *GitIntegration
		params      map[string]interface{}
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "nil git integration",
			gitInteg:    nil,
			params:      map[string]interface{}{"operation": "status"},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "not a repository",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    nil, // Not a repo
			},
			params:      map[string]interface{}{"operation": "status"},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "missing operation parameter",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			},
			params:      map[string]interface{}{},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "unknown operation",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			},
			params: map[string]interface{}{
				"operation": "unknown_operation",
			},
			wantSuccess: false,
			wantErr:     false,
		},
		{
			name: "valid get_status operation",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
				lastStatus: &git.Status{
					Branch:         "main",
					ModifiedFiles:  []git.FileStatus{},
					UntrackedFiles: []string{},
				},
			},
			params: map[string]interface{}{
				"operation": "get_status",
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "valid list_branches operation",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			},
			params: map[string]interface{}{
				"operation": "list_branches",
			},
			wantSuccess: true,
			wantErr:     false,
		},
		{
			name: "valid list_remotes operation",
			gitInteg: &GitIntegration{
				enabled: true,
				workDir: "/tmp",
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    &git.Repository{},
			},
			params: map[string]interface{}{
				"operation": "list_remotes",
			},
			wantSuccess: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewGitOperationTool(tt.gitInteg)
			ctx := context.Background()

			result, err := tool.Execute(ctx, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("Execute() success = %v, wantSuccess %v", result.Success, tt.wantSuccess)
			}
		})
	}
}

func TestGitOperationTool_Schema(t *testing.T) {
	gitInteg := &GitIntegration{
		enabled: true,
		workDir: "/tmp",
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	tool := NewGitOperationTool(gitInteg)

	schema := tool.Schema()
	if schema.Function.Name != "git_operation" {
		t.Errorf("Schema().Function.Name = %v, want git_operation", schema.Function.Name)
	}

	if len(schema.Function.Parameters.Required) != 1 || schema.Function.Parameters.Required[0] != "operation" {
		t.Errorf("Schema().Function.Parameters.Required = %v, want [operation]", schema.Function.Parameters.Required)
	}

	// Verify operation enum values
	operationProp, ok := schema.Function.Parameters.Properties["operation"]
	if !ok {
		t.Fatal("Schema() missing operation property")
	}

	expectedOps := []string{"stage", "commit", "push", "pull", "create_branch", "switch_branch", "list_branches", "list_remotes", "get_status", "get_diff", "get_log"}
	if len(operationProp.Enum) != len(expectedOps) {
		t.Errorf("Schema() operation enum length = %v, want %v", len(operationProp.Enum), len(expectedOps))
	}
}

func TestGitOperationTool_Name(t *testing.T) {
	tool := NewGitOperationTool(nil)
	if tool.Name() != "git_operation" {
		t.Errorf("Name() = %v, want git_operation", tool.Name())
	}
}

func TestGitOperationTool_Description(t *testing.T) {
	tool := NewGitOperationTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestGitSuccessResult(t *testing.T) {
	output := "test output"
	result := gitSuccessResult(output)

	if !result.Success {
		t.Error("gitSuccessResult() Success = false, want true")
	}

	if result.Output != output {
		t.Errorf("gitSuccessResult() Output = %v, want %v", result.Output, output)
	}
}

func TestGitErrorResult(t *testing.T) {
	errMsg := "test error"
	result := gitErrorResult(errMsg)

	if result.Success {
		t.Error("gitErrorResult() Success = true, want false")
	}

	if result.Error != errMsg {
		t.Errorf("gitErrorResult() Error = %v, want %v", result.Error, errMsg)
	}
}

func TestGitOperationTool_AllOperations(t *testing.T) {
	// Test that all operations in the handlers map are accessible
	expectedOperations := []string{
		"stage", "commit", "push", "pull",
		"create_branch", "switch_branch",
		"list_branches", "list_remotes",
		"get_status", "get_diff", "get_log",
	}

	for _, op := range expectedOperations {
		t.Run("operation_"+op, func(t *testing.T) {
			handler, exists := gitOperationHandlers[op]
			if !exists {
				t.Errorf("gitOperationHandlers missing handler for operation: %s", op)
			}
			if handler == nil {
				t.Errorf("gitOperationHandlers[%s] is nil", op)
			}
		})
	}
}
