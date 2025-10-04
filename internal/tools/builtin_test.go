package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// Mock executor for ExecuteCommandTool tests
type mockExecutor struct {
	executeFunc func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error)
}

func (m *mockExecutor) Execute(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd, opts)
	}
	return nil, nil
}

// Mock result that matches core.Result structure
type mockResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func TestExecuteCommandTool_NilExecutor(t *testing.T) {
	tool := NewExecuteCommandTool(nil, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo hello",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure with nil executor")
	}

	if !strings.Contains(result.Error, "executor not configured") {
		t.Errorf("expected 'executor not configured' error, got: %s", result.Error)
	}
}

func TestExecuteCommandTool_InvalidCommand(t *testing.T) {
	executor := &mockExecutor{}
	tool := NewExecuteCommandTool(executor, nil)

	tests := []struct {
		name        string
		params      map[string]interface{}
		wantErrMsg  string
	}{
		{
			name:       "missing command parameter",
			params:     map[string]interface{}{},
			wantErrMsg: "command parameter must be a non-empty string",
		},
		{
			name:       "empty command",
			params:     map[string]interface{}{"command": ""},
			wantErrMsg: "command parameter must be a non-empty string",
		},
		{
			name:       "whitespace-only command",
			params:     map[string]interface{}{"command": "   "},
			wantErrMsg: "command cannot be empty",
		},
		{
			name:       "non-string command",
			params:     map[string]interface{}{"command": 123},
			wantErrMsg: "command parameter must be a non-empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.params)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Error("expected failure")
			}

			if !strings.Contains(result.Error, tt.wantErrMsg) {
				t.Errorf("expected error containing %q, got: %s", tt.wantErrMsg, result.Error)
			}
		})
	}
}

func TestExecuteCommandTool_SimpleCommand(t *testing.T) {
	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{
				Stdout:   "hello world",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo hello world",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if result.Output != "hello world" {
		t.Errorf("expected output 'hello world', got: %q", result.Output)
	}
}

func TestExecuteCommandTool_CommandWithArgs(t *testing.T) {
	var capturedCmd interface{}

	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			capturedCmd = cmd
			return &mockResult{
				Stdout:   "command executed",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "git status --short",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify command was parsed correctly using reflection
	if capturedCmd == nil {
		t.Fatal("command was not captured")
	}

	cmdVal := reflect.ValueOf(capturedCmd)
	if cmdVal.Kind() == reflect.Ptr {
		cmdVal = cmdVal.Elem()
	}

	// Check Program field
	programField := cmdVal.FieldByName("Program")
	if programField.IsValid() && programField.String() != "git" {
		t.Errorf("expected Program 'git', got: %s", programField.String())
	}

	// Check Args field
	argsField := cmdVal.FieldByName("Args")
	if argsField.IsValid() && argsField.Len() != 2 {
		t.Errorf("expected 2 args, got: %d", argsField.Len())
	}
}

func TestExecuteCommandTool_WithWorkdir(t *testing.T) {
	var capturedCmd interface{}

	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			capturedCmd = cmd
			return &mockResult{
				Stdout:   "ok",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	tmpDir := t.TempDir()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "ls",
		"workdir": tmpDir,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify workdir was set
	if capturedCmd == nil {
		t.Fatal("command was not captured")
	}

	cmdVal := reflect.ValueOf(capturedCmd)
	if cmdVal.Kind() == reflect.Ptr {
		cmdVal = cmdVal.Elem()
	}

	workDirField := cmdVal.FieldByName("WorkDir")
	if workDirField.IsValid() && workDirField.String() != tmpDir {
		t.Errorf("expected WorkDir %q, got: %s", tmpDir, workDirField.String())
	}
}

func TestExecuteCommandTool_CommandFailure(t *testing.T) {
	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{
				Stdout:   "",
				Stderr:   "command not found",
				ExitCode: 127,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "nonexistent-command",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure for non-zero exit code")
	}

	if !strings.Contains(result.Output, "command not found") {
		t.Errorf("expected stderr in output, got: %q", result.Output)
	}
}

func TestExecuteCommandTool_ExecutionError(t *testing.T) {
	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{
				Stdout:   "",
				Stderr:   "execution failed",
				ExitCode: 1,
			}, fmt.Errorf("command execution error")
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "failing-command",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure")
	}

	if !strings.Contains(result.Error, "command execution error") {
		t.Errorf("expected execution error, got: %s", result.Error)
	}
}

func TestExecuteCommandTool_StdoutAndStderr(t *testing.T) {
	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{
				Stdout:   "standard output",
				Stderr:   "standard error",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "command-with-stderr",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Both stdout and stderr should be in output
	if !strings.Contains(result.Output, "standard output") {
		t.Errorf("expected stdout in output, got: %q", result.Output)
	}

	if !strings.Contains(result.Output, "standard error") {
		t.Errorf("expected stderr in output, got: %q", result.Output)
	}
}

// GetContextTool tests

// Mock type that implements String() for testing
type mockEnvironment struct {
	data string
}

func (m *mockEnvironment) String() string {
	return m.data
}

func TestGetContextTool_Success(t *testing.T) {
	// Create a valid mock context with String() method
	env := &mockEnvironment{
		data: `Environment Context:
- OS: linux (amd64)
- Working Directory: /test/project
- Project Type: go
- Languages: Go`,
	}

	tool := NewGetContextTool(env)
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify output contains expected sections
	expectedStrings := []string{
		"Environment Context:",
		"linux",
		"amd64",
		"/test/project",
		"go",
		"Go",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result.Output, expected) {
			t.Errorf("expected output to contain %q, got: %q", expected, result.Output)
		}
	}
}

func TestGetContextTool_NilContext(t *testing.T) {
	tool := NewGetContextTool(nil)
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for nil context")
	}

	if result.Error != "context not available" {
		t.Errorf("expected error message 'context not available', got: %s", result.Error)
	}
}

func TestGetContextTool_InvalidType(t *testing.T) {
	// Type without String() method
	type InvalidContext struct {
		Data string
	}

	tool := NewGetContextTool(&InvalidContext{Data: "test"})
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for invalid context type")
	}

	if result.Error != "context does not implement String() method" {
		t.Errorf("expected error message about missing String() method, got: %s", result.Error)
	}
}

func TestGetContextTool_WithGitInfo(t *testing.T) {
	// Create mock environment with Git information
	env := &mockEnvironment{
		data: `Environment Context:
- OS: darwin (arm64)
- Working Directory: /Users/test/project
- Project Type: go
- Languages: Go
- Git Branch: master (dirty)`,
	}

	tool := NewGetContextTool(env)
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify Git information is in output
	if !strings.Contains(result.Output, "master") {
		t.Errorf("expected output to contain Git branch 'master', got: %q", result.Output)
	}

	if !strings.Contains(result.Output, "dirty") {
		t.Errorf("expected output to contain 'dirty' status, got: %q", result.Output)
	}
}

func TestGetContextTool_OutputFormat(t *testing.T) {
	// Verify the String() method is called correctly via reflection
	env := &mockEnvironment{
		data: `Environment Context:
- OS: linux (amd64)
- Kernel: 6.16.8
- Shell: /bin/bash
- Working Directory: /home/user/project
- Project Type: go
- Languages: Go, Python

Project Structure: 2 files
- main.go (Go, 100 lines)
- test.py (Python, 50 lines)`,
	}

	tool := NewGetContextTool(env)
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify output format matches Environment.String() structure
	expectedSections := []string{
		"Environment Context:",
		"- OS:",
		"- Kernel:",
		"- Shell:",
		"- Working Directory:",
		"- Project Type:",
		"- Languages:",
		"Project Structure:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(result.Output, section) {
			t.Errorf("expected output to contain section %q, got: %q", section, result.Output)
		}
	}

	// Verify specific values
	if !strings.Contains(result.Output, "6.16.8") {
		t.Errorf("expected kernel version in output")
	}

	if !strings.Contains(result.Output, "/bin/bash") {
		t.Errorf("expected shell in output")
	}

	if !strings.Contains(result.Output, "Go, Python") {
		t.Errorf("expected languages list in output")
	}
}

func TestGetContextTool_Schema(t *testing.T) {
	tool := NewGetContextTool(nil)
	schema := tool.Schema()

	if schema.Function.Name != "get_context" {
		t.Errorf("expected name 'get_context', got: %s", schema.Function.Name)
	}

	if schema.Function.Description == "" {
		t.Errorf("expected non-empty description")
	}

	// Tool should have no required parameters
	if len(schema.Function.Parameters.Required) != 0 {
		t.Errorf("expected no required parameters, got: %d", len(schema.Function.Parameters.Required))
	}
}
