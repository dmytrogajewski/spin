package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewReadFileTool()

	tests := []struct {
		name       string
		params     map[string]interface{}
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "read existing file",
			params:     map[string]interface{}{"path": testFile},
			wantOutput: testContent,
		},
		{
			name:    "missing path parameter",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "non-existent file",
			params:  map[string]interface{}{"path": filepath.Join(tmpDir, "nonexistent.txt")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.params)

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

			if result.Output != tt.wantOutput {
				t.Errorf("expected output %q, got %q", tt.wantOutput, result.Output)
			}
		})
	}
}

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
			result, err := tool.Execute(context.Background(), tt.params)

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

func TestListDirectoryTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files and directories
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	tool := NewListDirectoryTool()

	tests := []struct {
		name         string
		params       map[string]interface{}
		wantErr      bool
		wantContains []string
	}{
		{
			name:         "list directory",
			params:       map[string]interface{}{"path": tmpDir},
			wantContains: []string{"file1.txt", "file2.txt", "subdir"},
		},
		{
			name:    "missing path parameter",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			params:  map[string]interface{}{"path": filepath.Join(tmpDir, "nonexistent")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.params)

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

func TestToolSchemas(t *testing.T) {
	tools := []Tool{
		NewReadFileTool(),
		NewWriteFileTool(),
		NewListDirectoryTool(),
		NewExecuteCommandTool(nil, nil), // nil executor/validator for schema test
		NewGetContextTool(nil),          // nil context for schema test
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			schema := tool.Schema()

			// Verify schema structure
			if schema.Type != "function" {
				t.Errorf("expected type 'function', got %s", schema.Type)
			}

			if schema.Function.Name != tool.Name() {
				t.Errorf("expected name %s, got %s", tool.Name(), schema.Function.Name)
			}

			if schema.Function.Description == "" {
				t.Error("expected non-empty description")
			}

			if schema.Function.Parameters.Type != "object" {
				t.Errorf("expected parameters type 'object', got %s", schema.Function.Parameters.Type)
			}

			// Verify required parameters are defined
			for _, required := range schema.Function.Parameters.Required {
				if _, exists := schema.Function.Parameters.Properties[required]; !exists {
					t.Errorf("required parameter %s not defined in properties", required)
				}
			}
		})
	}
}

func TestBuiltinToolsIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a registry with built-in tools
	registry := NewRegistry()
	ctx := &Context{WorkDir: tmpDir} // Mock context

	// Register tools (with nil executor/validator since we're testing non-command tools)
	tools := []Tool{
		NewReadFileTool(),
		NewWriteFileTool(),
		NewListDirectoryTool(),
		NewGetContextTool(ctx),
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			t.Fatalf("failed to register tool %s: %v", tool.Name(), err)
		}
	}

	// Test workflow: write -> read -> list
	t.Run("write-read-list workflow", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "workflow.txt")
		testContent := "workflow test content"

		// 1. Write file
		writeResult, err := registry.Execute(context.Background(), "write_file", map[string]interface{}{
			"path":    testFile,
			"content": testContent,
		})
		if err != nil {
			t.Fatalf("write_file failed: %v", err)
		}
		if !writeResult.Success {
			t.Fatalf("write_file unsuccessful: %s", writeResult.Error)
		}

		// 2. Read file
		readResult, err := registry.Execute(context.Background(), "read_file", map[string]interface{}{
			"path": testFile,
		})
		if err != nil {
			t.Fatalf("read_file failed: %v", err)
		}
		if !readResult.Success {
			t.Fatalf("read_file unsuccessful: %s", readResult.Error)
		}
		if readResult.Output != testContent {
			t.Errorf("expected content %q, got %q", testContent, readResult.Output)
		}

		// 3. List directory
		listResult, err := registry.Execute(context.Background(), "list_directory", map[string]interface{}{
			"path": tmpDir,
		})
		if err != nil {
			t.Fatalf("list_directory failed: %v", err)
		}
		if !listResult.Success {
			t.Fatalf("list_directory unsuccessful: %s", listResult.Error)
		}
		if !strings.Contains(listResult.Output, "workflow.txt") {
			t.Errorf("expected output to contain workflow.txt, got %q", listResult.Output)
		}
	})
}

// Mock Context for testing
type Context struct {
	WorkDir string
}
