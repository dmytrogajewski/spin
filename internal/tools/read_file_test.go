package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileTool(t *testing.T) {
	t.Parallel(
	// Create temp file.
	)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewReadFileTool()

	tests := []struct {
		name       string
		params     map[string]any
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "read existing file",
			params:     map[string]any{"path": testFile},
			wantOutput: testContent,
		},
		{
			name:    "missing path parameter",
			params:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "non-existent file",
			params:  map[string]any{"path": filepath.Join(tmpDir, "nonexistent.txt")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runReadFileSubtest(t, tool, tt.params, tt.wantErr, tt.wantOutput)
		})
	}
}

func runReadFileSubtest(t *testing.T, tool Tool, params map[string]any, wantErr bool, wantOutput string) {
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

	if result.Output != wantOutput {
		t.Errorf("expected output %q, got %q", wantOutput, result.Output)
	}
}
