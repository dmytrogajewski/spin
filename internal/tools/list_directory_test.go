package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListDirectoryTool(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create test files and directories.
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

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
			params, _ := FromMap(tt.params)
			result, err := tool.Execute(context.Background(), params)

			if tt.wantErr {
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

			for _, expected := range tt.wantContains {
				if !strings.Contains(result.Output, expected) {
					t.Errorf("expected output to contain %q, got %q", expected, result.Output)
				}
			}
		})
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
