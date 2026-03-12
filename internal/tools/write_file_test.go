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
			runWriteFileSubtest(t, tool, tt.params, tt.wantErr, tt.verifyWrite)
		})
	}
}

func runWriteFileSubtest(t *testing.T, tool Tool, params map[string]any, wantErr, verifyWrite bool) {
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

	if verifyWrite {
		verifyWrittenFile(t, params)
	}
}

func verifyWrittenFile(t *testing.T, params map[string]any) {
	t.Helper()

	path, _ := params["path"].(string)
	expectedContent, _ := params["content"].(string)

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Errorf("failed to read written file: %v", readErr)
		return
	}

	if string(content) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(content))
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

// writeFileApprovalCase describes a test case for write file approval checking.
type writeFileApprovalCase struct {
	name string
	path string
	want RiskLevel
}

func runWriteFileApprovalTests(t *testing.T, tool *WriteFileTool, cases []writeFileApprovalCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := FromMap(map[string]any{
				"path":    tt.path,
				"content": "test",
			})

			needs := tool.CheckApproval(params)

			if !needs.Required {
				t.Errorf("CheckApproval should require approval for %s", tt.path)
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

func TestWriteFileTool_CheckApproval_SystemPaths(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"etc directory", "/etc/config.conf", RiskCritical},
		{"sys directory", "/sys/kernel/param", RiskCritical},
		{"usr directory", "/usr/bin/script", RiskCritical},
	})
}

func TestWriteFileTool_CheckApproval_RegularFiles(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"text file", "/tmp/notes.txt", RiskMedium},
		{"markdown", "/tmp/README.md", RiskMedium},
		{"json file", "/tmp/config.json", RiskMedium},
	})
}

func TestWriteFileTool_CheckApproval_ExecutableFiles(t *testing.T) {
	t.Parallel()
	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"shell script", "/tmp/script.sh", RiskHigh},
		{"go source", "/tmp/main.go", RiskHigh},
		{"python script", "/tmp/script.py", RiskHigh},
	})
}
