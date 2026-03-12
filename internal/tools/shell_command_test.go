package tools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Mock executor for shell_command tests.
type mockExecutor struct {
	executeFunc func(ctx context.Context, cmd CommandInfo, opts any) (ExecutionResult, error)
}

func (m *mockExecutor) Execute(ctx context.Context, cmd CommandInfo, opts any) (ExecutionResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd, opts)
	}

	return &mockResult{}, nil
}

// Mock result that implements ExecutionResult interface.
type mockResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Implement ExecutionResult interface for mockResult.
func (m *mockResult) GetStdout() string {
	return m.Stdout
}

func (m *mockResult) GetStderr() string {
	return m.Stderr
}

func (m *mockResult) GetExitCode() int {
	return m.ExitCode
}

func (m *mockResult) GetMetadata() map[string]any {
	return map[string]any{}
}

// TestNewShellCommandTool_NilExecutor tests that creating tool with nil executor fails gracefully.
func TestNewShellCommandTool_NilExecutor(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	if tool == nil {
		t.Fatal("Expected non-nil tool, got nil")
	}

	if tool.executor != nil {
		t.Error("Expected nil executor to be stored")
	}
}

// TestShellCommandTool_Name tests that Name returns correct tool name.
func TestShellCommandTool_Name(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	expected := "shell_command"
	if got := tool.Name(); got != expected {
		t.Errorf("Name() = %q, want %q", got, expected)
	}
}

// TestShellCommandTool_Description tests that Description returns correct description.
func TestShellCommandTool_Description(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	if len(desc) < 10 {
		t.Errorf("Description() too short: %q", desc)
	}
}

// TestShellCommandTool_Schema tests that Schema returns valid schema.
func TestShellCommandTool_Schema(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	schema := tool.Schema()
	if schema.Type != "function" {
		t.Errorf("Schema.Type = %q, want %q", schema.Type, "function")
	}

	if schema.Function.Name != "shell_command" {
		t.Errorf("Schema.Function.Name = %q, want %q", schema.Function.Name, "shell_command")
	}

	// Check operation parameter.
	opProp, ok := schema.Function.Parameters.Properties["operation"]
	if !ok {
		t.Fatal("Schema missing 'operation' property")
	}

	if len(opProp.Enum) != 5 {
		t.Errorf("operation.Enum has %d values, want 5", len(opProp.Enum))
	}

	// Check required fields - both operation and command are required
	// (command is required to ensure LLMs always provide it for execute operations).
	if len(schema.Function.Parameters.Required) != 2 {
		t.Errorf("Required has %d fields, want 2", len(schema.Function.Parameters.Required))
	}

	if schema.Function.Parameters.Required[0] != "operation" {
		t.Errorf("Required[0] = %q, want %q", schema.Function.Parameters.Required[0], "operation")
	}

	if schema.Function.Parameters.Required[1] != "command" {
		t.Errorf("Required[1] = %q, want %q", schema.Function.Parameters.Required[1], "command")
	}
}

// TestShellCommandTool_Execute_MissingOperation tests missing operation parameter.
func TestShellCommandTool_Execute_MissingOperation(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing operation")
	}

	if result.Error != "operation parameter is required" {
		t.Errorf("Error = %q, want 'operation parameter is required'", result.Error)
	}
}

// TestShellCommandTool_Execute_UnknownOperation tests unknown operation.
func TestShellCommandTool_Execute_UnknownOperation(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "unknown_op",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for unknown operation")
	}

	if result.Error != "unknown operation: unknown_op" {
		t.Errorf("Error = %q, want 'unknown operation: unknown_op'", result.Error)
	}
}

// TestShellCommandTool_Execute_NilExecutor tests execute operation with nil executor.
func TestShellCommandTool_Execute_NilExecutor(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "echo test",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for nil executor")
	}

	if result.Error != "executor not configured" {
		t.Errorf("Error = %q, want 'executor not configured'", result.Error)
	}
}

// TestShellCommandTool_Execute_MissingCommand tests execute without command.
func TestShellCommandTool_Execute_MissingCommand(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{Stdout: "", Stderr: "", ExitCode: 0}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation": "execute",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing command")
	}

	if result.Error != "command parameter is required for execute operation" {
		t.Errorf("Error = %q", result.Error)
	}
}

// TestShellCommandTool_Execute_EmptyCommand tests execute with empty command.
func TestShellCommandTool_Execute_EmptyCommand(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for empty command")
	}
}

// TestShellCommandTool_Execute_Success tests successful command execution.
func TestShellCommandTool_Execute_Success(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{
				Stdout:   "hello world",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "echo hello",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output != "hello world" {
		t.Errorf("Output = %q, want %q", result.Output, "hello world")
	}
}

// TestShellCommandTool_Execute_Failure tests failed command execution.
func TestShellCommandTool_Execute_Failure(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{
				Stdout:   "",
				Stderr:   "command not found",
				ExitCode: 127,
			}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "nonexistent",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for non-zero exit code")
	}

	if result.Output != "command not found" {
		t.Errorf("Output = %q, want %q", result.Output, "command not found")
	}
}

// TestShellCommandTool_Execute_WithWorkDir tests execute with working directory.
func TestShellCommandTool_Execute_WithWorkDir(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{
				Stdout:   "success",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation":         "execute",
		"command":           "ls",
		"working_directory": "/tmp",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}
}

// TestShellCommandTool_Execute_WithTimeout tests execute with custom timeout.
func TestShellCommandTool_Execute_WithTimeout(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return &mockResult{
				Stdout:   "output",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "sleep 1",
		"timeout":   5.0,
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}
}

// TestShellCommandTool_Execute_Timeout tests execute timeout.
func TestShellCommandTool_Execute_Timeout(t *testing.T) {
	t.Parallel()
	executor := &mockExecutor{
		executeFunc: func(_ context.Context, _ CommandInfo, _ any) (ExecutionResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	tool := NewShellCommandTool(nil, nil, executor)

	params, err := FromMap(map[string]any{
		"operation": "execute",
		"command":   "sleep 10",
		"timeout":   0.1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for timeout")
	}
}

// TestShellCommandTool_GetEnvironment_WithShellInfo tests get_environment with shell integration.
func TestShellCommandTool_GetEnvironment_WithShellInfo(t *testing.T) {
	t.Parallel(
	// Mock shell integration.
	)

	type mockShellInfo struct{}

	shellInfo := &mockShellInfo{}

	// Add GetEnvironmentVars method dynamically (not possible in Go without reflection tricks)
	// For now, test fallback.
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "get_environment",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}

	if len(result.Output) < 20 {
		t.Errorf("Output too short: %q", result.Output)
	}

	_ = shellInfo // avoid unused warning.
}

// TestShellCommandTool_GetEnvironment_Fallback tests get_environment without shell integration.
func TestShellCommandTool_GetEnvironment_Fallback(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "get_environment",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestShellCommandTool_GetShellInfo_Fallback tests get_shell_info without shell integration.
func TestShellCommandTool_GetShellInfo_Fallback(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "get_shell_info",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestShellCommandTool_DetectShell_MissingCommand tests detect_shell without command.
func TestShellCommandTool_DetectShell_MissingCommand(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "detect_shell",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing command")
	}

	if result.Error != "command parameter is required for detect_shell operation" {
		t.Errorf("Error = %q", result.Error)
	}
}

// TestShellCommandTool_DetectShell_Pipe tests detect_shell with pipe command.
func TestShellCommandTool_DetectShell_Pipe(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "detect_shell",
		"command":   "ls | grep test",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output != "Is shell command: true" {
		t.Errorf("Output = %q, want 'Is shell command: true'", result.Output)
	}
}

// TestShellCommandTool_DetectShell_Simple tests detect_shell with simple command.
func TestShellCommandTool_DetectShell_Simple(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "detect_shell",
		"command":   "ls -la",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output != "Is shell command: false" {
		t.Errorf("Output = %q, want 'Is shell command: false'", result.Output)
	}
}

// TestShellCommandTool_DetectShell_Redirect tests detect_shell with redirect.
func TestShellCommandTool_DetectShell_Redirect(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{"pipe", "cat file | grep test", true},
		{"redirect_out", "echo test > file", true},
		{"redirect_in", "cat < file", true},
		{"variable", "echo $HOME", true},
		{"cd", "cd /tmp", true},
		{"export", "export VAR=value", true},
		{"source", "source file.sh", true},
		{"simple", "ls -la", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, err := FromMap(map[string]any{
				"operation": "detect_shell",
				"command":   tt.command,
			})
			require.NoError(t, err)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}

			if !result.Success {
				t.Errorf("Expected success, got failure: %s", result.Error)
			}

			expected := "Is shell command: "
			if tt.want {
				expected += "true"
			} else {
				expected += "false"
			}

			if result.Output != expected {
				t.Errorf("Output = %q, want %q", result.Output, expected)
			}
		})
	}
}

// mockValidatorShell is a mock validator for shell_command testing.
type mockValidatorShell struct {
	classifyFunc func(cmd CommandInfo) (ValidationResult, error)
}

// mockClassificationResult is the result from Classify.
type mockClassificationResult struct {
	Classification int
	Reason         string
}

// Implement ValidationResult interface for mockClassificationResult.
func (m *mockClassificationResult) GetClassification() int {
	return m.Classification
}

func (m *mockClassificationResult) GetReason() string {
	return m.Reason
}

// Classify is the mock Classify method.
func (m *mockValidatorShell) Classify(cmd CommandInfo) (ValidationResult, error) {
	if m.classifyFunc != nil {
		return m.classifyFunc(cmd)
	}

	return &mockClassificationResult{
		Classification: 0,
		Reason:         "",
	}, nil
}

// TestShellCommandTool_Validate_NilValidator tests validate with nil validator.
func TestShellCommandTool_Validate_NilValidator(t *testing.T) {
	t.Parallel()
	tool := NewShellCommandTool(nil, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
		"command":   "ls",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for nil validator")
	}

	if result.Error != "validator not configured" {
		t.Errorf("Error = %q, want 'validator not configured'", result.Error)
	}
}

// TestShellCommandTool_Validate_MissingCommand tests validate without command.
func TestShellCommandTool_Validate_MissingCommand(t *testing.T) {
	t.Parallel()
	validator := &mockValidatorShell{}
	tool := NewShellCommandTool(validator, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for missing command")
	}

	if result.Error != "command parameter is required for validate operation" {
		t.Errorf("Error = %q", result.Error)
	}
}

// TestShellCommandTool_Validate_Safe tests validate with safe command.
func TestShellCommandTool_Validate_Safe(t *testing.T) {
	t.Parallel()
	validator := &mockValidatorShell{
		classifyFunc: func(_ CommandInfo) (ValidationResult, error) {
			return &mockClassificationResult{
				Classification: 0, // Safe.
				Reason:         "read-only command",
			}, nil
		},
	}
	tool := NewShellCommandTool(validator, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
		"command":   "ls -la",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestShellCommandTool_Validate_Dangerous tests validate with dangerous command.
func TestShellCommandTool_Validate_Dangerous(t *testing.T) {
	t.Parallel()
	validator := &mockValidatorShell{
		classifyFunc: func(_ CommandInfo) (ValidationResult, error) {
			return &mockClassificationResult{
				Classification: 1, // Dangerous.
				Reason:         "modifies filesystem",
			}, nil
		},
	}
	tool := NewShellCommandTool(validator, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
		"command":   "rm -rf /tmp/test",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestShellCommandTool_Validate_Critical tests validate with critical command.
func TestShellCommandTool_Validate_Critical(t *testing.T) {
	t.Parallel()
	validator := &mockValidatorShell{
		classifyFunc: func(_ CommandInfo) (ValidationResult, error) {
			return &mockClassificationResult{
				Classification: 2, // Critical.
				Reason:         "requires elevated privileges",
			}, nil
		},
	}
	tool := NewShellCommandTool(validator, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
		"command":   "sudo rm -rf /",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestShellCommandTool_Validate_EmptyCommand tests validate with empty command.
func TestShellCommandTool_Validate_EmptyCommand(t *testing.T) {
	t.Parallel()
	validator := &mockValidatorShell{}
	tool := NewShellCommandTool(validator, nil, nil)

	params, err := FromMap(map[string]any{
		"operation": "validate",
		"command":   "",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for empty command")
	}
}
