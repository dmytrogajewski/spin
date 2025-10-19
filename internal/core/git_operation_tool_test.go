package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initRepo initializes a git repository in the given directory
func initRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run(), "git init should succeed")

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	// Create initial commit
	readmePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# Test"), 0644))

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

// TestGitOperationTool_Execute_Integration tests all git operations work correctly.
// This serves as safety net before refactoring.
func TestGitOperationTool_Execute_Integration(t *testing.T) {
	// Setup: Create a real git repository in temp directory
	workDir := t.TempDir()

	// Initialize git repo
	initRepo(t, workDir)

	gitIntegration := NewGitIntegration(true, workDir, testLogger())
	require.NoError(t, gitIntegration.Initialize(context.Background()))

	tool := NewGitOperationTool(gitIntegration)
	require.NotNil(t, tool)

	ctx := context.Background()

	// Test all supported operations
	tests := []struct {
		name      string
		operation string
		params    map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "get_status",
			operation: "get_status",
			params:    map[string]interface{}{"operation": "get_status"},
			wantErr:   false,
		},
		{
			name:      "list_branches",
			operation: "list_branches",
			params:    map[string]interface{}{"operation": "list_branches"},
			wantErr:   false,
		},
		{
			name:      "list_remotes",
			operation: "list_remotes",
			params:    map[string]interface{}{"operation": "list_remotes"},
			wantErr:   false,
		},
		{
			name:      "unknown_operation",
			operation: "invalid_op",
			params:    map[string]interface{}{"operation": "invalid_op"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(ctx, tt.params)

			assert.NoError(t, err, "Execute should not return error")
			assert.NotNil(t, result)

			if tt.wantErr {
				assert.False(t, result.Success, "Result should indicate failure")
				assert.NotEmpty(t, result.Error, "Error message should be present")
			} else {
				assert.True(t, result.Success, "Result should indicate success")
				assert.Empty(t, result.Error, "Error should be empty on success")
			}
		})
	}
}

// TestGitOperationTool_Execute_ValidatesGitIntegration tests validation logic
func TestGitOperationTool_Execute_ValidatesGitIntegration(t *testing.T) {
	tool := NewGitOperationTool(nil)
	ctx := context.Background()
	params := map[string]interface{}{"operation": "get_status"}

	result, err := tool.Execute(ctx, params)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "Not a Git repository")
}

// TestGitOperationTool_Execute_RequiresOperation tests operation parameter validation
func TestGitOperationTool_Execute_RequiresOperation(t *testing.T) {
	workDir := t.TempDir()
	initRepo(t, workDir)

	gitIntegration := NewGitIntegration(true, workDir, testLogger())
	require.NoError(t, gitIntegration.Initialize(context.Background()))

	tool := NewGitOperationTool(gitIntegration)
	ctx := context.Background()
	params := map[string]interface{}{} // Missing operation

	result, err := tool.Execute(ctx, params)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "operation parameter is required")
}

// Test helper functions
func TestGitOperationTool_Helpers(t *testing.T) {
	t.Run("successResult", func(t *testing.T) {
		result := gitSuccessResult("test output")
		assert.True(t, result.Success)
		assert.Equal(t, "test output", result.Output)
		assert.Empty(t, result.Error)
	})

	t.Run("errorResult", func(t *testing.T) {
		result := gitErrorResult("test error")
		assert.False(t, result.Success)
		assert.Empty(t, result.Output)
		assert.Equal(t, "test error", result.Error)
	})
}
