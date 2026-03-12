package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileTool(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	tool := NewWriteFileTool()

	tests := []struct {
		name        string
		params      map[string]any
		wantErr     bool
		verifyWrite bool
	}{
		{
			name: "write new file",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "new.txt"),
				"content": "test content",
			},
			verifyWrite: true,
		},
		{
			name: "overwrite existing file",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "overwrite.txt"),
				"content": "new content",
			},
			verifyWrite: true,
		},
		{
			name: "create parent directories",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "subdir", "nested", "file.txt"),
				"content": "content in nested directory",
			},
			verifyWrite: true,
		},
		{
			name:    "missing path parameter",
			params:  map[string]any{"content": "test"},
			wantErr: true,
		},
		{
			name:    "missing content parameter",
			params:  map[string]any{"path": filepath.Join(tmpDir, "test.txt")},
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

			if tt.verifyWrite {
				path, ok := tt.params["path"].(string)
				if !ok {
					t.Error("path parameter not found")

					return
				}

				content, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Errorf("failed to read written file: %v", readErr)

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
	t.Parallel()
	tool := NewWriteFileTool()

	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name:   "missing path",
			params: map[string]any{"content": "test"},
		},
		{
			name:   "missing content",
			params: map[string]any{"path": "test.txt"},
		},
		{
			name:   "invalid path type",
			params: map[string]any{"path": 123, "content": "test"},
		},
		{
			name:   "invalid content type",
			params: map[string]any{"path": "test.txt", "content": 123},
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

func TestWriteFileTool_CheckApproval_SystemPaths(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()

	tests := []struct {
		name string
		path string
		want RiskLevel
	}{
		{"etc directory", "/etc/config.conf", RiskCritical},
		{"sys directory", "/sys/kernel/param", RiskCritical},
		{"usr directory", "/usr/bin/script", RiskCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := FromMap(map[string]any{
				"path":    tt.path,
				"content": "test",
			})

			needs := tool.CheckApproval(params)

			if !needs.Required {
				t.Error("CheckApproval should require approval for system paths")
			}

			if needs.Risk != tt.want {
				t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, tt.want)
			}

			if needs.Reason == "" {
				t.Error("CheckApproval should provide a reason")
			}
		})
	}
}

func TestWriteFileTool_CheckApproval_RegularFiles(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()

	tests := []struct {
		name string
		path string
		want RiskLevel
	}{
		{"text file", "/tmp/notes.txt", RiskMedium},
		{"markdown", "/tmp/README.md", RiskMedium},
		{"json file", "/tmp/config.json", RiskMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := FromMap(map[string]any{
				"path":    tt.path,
				"content": "test",
			})

			needs := tool.CheckApproval(params)

			if !needs.Required {
				t.Error("CheckApproval should require approval for file writes")
			}

			if needs.Risk != tt.want {
				t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, tt.want)
			}

			if needs.Reason == "" {
				t.Error("CheckApproval should provide a reason")
			}
		})
	}
}

func TestWriteFileTool_CheckApproval_ExecutableFiles(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()

	tests := []struct {
		name string
		path string
		want RiskLevel
	}{
		{"shell script", "/tmp/script.sh", RiskHigh},
		{"go source", "/tmp/main.go", RiskHigh},
		{"python script", "/tmp/script.py", RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := FromMap(map[string]any{
				"path":    tt.path,
				"content": "test",
			})

			needs := tool.CheckApproval(params)

			if !needs.Required {
				t.Error("CheckApproval should require approval for executable files")
			}

			if needs.Risk != tt.want {
				t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, tt.want)
			}

			if needs.Reason == "" {
				t.Error("CheckApproval should provide a reason")
			}
		})
	}
}
