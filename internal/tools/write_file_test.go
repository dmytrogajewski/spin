package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileTool(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewWriteFileTool()

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantErr     bool
		verifyWrite bool
	}{
		{
			name: "write new file",
			params: map[string]interface{}{
				"path":    filepath.Join(tmpDir, "new.txt"),
				"content": "test content",
			},
			verifyWrite: true,
		},
		{
			name: "overwrite existing file",
			params: map[string]interface{}{
				"path":    filepath.Join(tmpDir, "overwrite.txt"),
				"content": "new content",
			},
			verifyWrite: true,
		},
		{
			name:    "missing path parameter",
			params:  map[string]interface{}{"content": "test"},
			wantErr: true,
		},
		{
			name:    "missing content parameter",
			params:  map[string]interface{}{"path": filepath.Join(tmpDir, "test.txt")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if tt.verifyWrite {
				path, ok := tt.params["path"].(string)
				if !ok {
					t.Error("path parameter not found")
					return
				}
				content, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read written file: %v", err)
					return
				}
				expectedContent, ok := tt.params["content"].(string)
				if !ok {
					t.Error("content parameter not found")
					return
				}
				if string(content) != expectedContent {
					t.Errorf("expected content %q, got %q", expectedContent, string(content))
				}
			}
		})
	}
}

func TestWriteFileTool_ErrorCases(t *testing.T) {
	tool := NewWriteFileTool()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "missing path",
			params: map[string]interface{}{"content": "test"},
		},
		{
			name:   "missing content",
			params: map[string]interface{}{"path": "test.txt"},
		},
		{
			name:   "invalid path type",
			params: map[string]interface{}{"path": 123, "content": "test"},
		},
		{
			name:   "invalid content type",
			params: map[string]interface{}{"path": "test.txt", "content": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
