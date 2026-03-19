package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListDirectoryTool(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create test files and directories.
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0o600)
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0o600)
	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o750)

	tool := NewListDirectoryTool()

	tests := []struct {
		name         string
		params       map[string]any
		wantErr      bool
		wantContains []string
	}{
		{
			name:         "list directory",
			params:       map[string]any{"path": tmpDir},
			wantContains: []string{"file1.txt", "file2.txt", "subdir"},
		},
		{
			name:    "missing path parameter",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			params:  map[string]any{"path": filepath.Join(tmpDir, "nonexistent")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runToolTest(t, tool, tt.params, tt.wantErr, tt.wantContains)
		})
	}
}

// runToolTest executes a tool and verifies the result.
func runToolTest(t *testing.T, tool Tool, params map[string]any, wantErr bool, wantContains []string) {
	t.Helper()

	p, _ := FromMap(params)
	result, err := tool.Execute(context.Background(), p)

	if wantErr {
		if err == nil && result.Success {
			t.Error("expected error but got success")
		}

		return
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)

		return
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)

		return
	}

	for _, expected := range wantContains {
		if !strings.Contains(result.Output, expected) {
			t.Errorf("expected output to contain %q, got %q", expected, result.Output)
		}
	}
}

// TestListDirectoryTool_EmptyDirectory tests that listing an empty directory
// returns a clear message instead of empty output.
// Reproduces bug: `ls tetris/src` returned "Exit code: 0. No output." which
// confused the agent into thinking the listing failed rather than the dir being empty.
func TestListDirectoryTool_EmptyDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.Mkdir(emptyDir, 0o750))

	tool := NewListDirectoryTool()
	params, _ := FromMap(map[string]any{"path": emptyDir})

	result, err := tool.Execute(context.Background(), params)
	require.NoError(t, err)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Output should explicitly indicate the directory is empty, not be blank.
	if result.Output == "" {
		t.Error("empty directory listing should return a descriptive message, not empty string")
	}

	if !strings.Contains(strings.ToLower(result.Output), "empty") {
		t.Errorf("empty directory output should mention 'empty', got: %q", result.Output)
	}
}

func TestListDirectoryTool_ErrorCases(t *testing.T) {
	t.Parallel()

	tool := NewListDirectoryTool()

	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name:   "missing path",
			params: map[string]any{},
		},
		{
			name:   "invalid path type",
			params: map[string]any{"path": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Error("expected failure result")
			}
		})
	}
}
