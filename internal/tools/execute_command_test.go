package tools

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

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

// Mock result that matches agent.Result structure
type mockResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func TestExecuteCommandTool_NilExecutor(t *testing.T) {
	tool := NewExecuteCommandTool(nil, nil)

	params, _ := FromMap(map[string]interface{}{
		"command": "echo hello",
	})

	result, err := tool.Execute(context.Background(), params)

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
		name       string
		params     map[string]interface{}
		wantErrMsg string
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
			params, _ := FromMap(tt.params)
			result, err := tool.Execute(context.Background(), params)

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

	params, _ := FromMap(map[string]interface{}{
		"command": "echo hello world",
	})

	result, err := tool.Execute(context.Background(), params)

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

	params, _ := FromMap(map[string]interface{}{
		"command": "git status --short",
	})

	result, err := tool.Execute(context.Background(), params)

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
	params, _ := FromMap(map[string]interface{}{
		"command":           "ls",
		"working_directory": tmpDir,
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// Verify working_directory was set
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

	params, _ := FromMap(map[string]interface{}{
		"command": "nonexistent-command",
	})

	result, err := tool.Execute(context.Background(), params)

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

	params, _ := FromMap(map[string]interface{}{
		"command": "failing-command",
	})

	result, err := tool.Execute(context.Background(), params)

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

	params, _ := FromMap(map[string]interface{}{
		"command": "command-with-stderr",
	})

	result, err := tool.Execute(context.Background(), params)

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

// TypedCommand is a typed command struct for testing typed executor
type TypedCommand struct {
	Program string
	Args    []string
	Raw     string
	WorkDir string
}

// typedExecutor is a mock executor with typed command parameter
type typedExecutor struct {
	executeFunc func(ctx context.Context, cmd *TypedCommand, opts interface{}) (interface{}, error)
}

func (t *typedExecutor) Execute(ctx context.Context, cmd *TypedCommand, opts interface{}) (interface{}, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, cmd, opts)
	}
	return nil, nil
}

func TestExecuteCommandTool_TypedExecutor(t *testing.T) {
	var capturedCmd *TypedCommand

	executor := &typedExecutor{
		executeFunc: func(ctx context.Context, cmd *TypedCommand, opts interface{}) (interface{}, error) {
			capturedCmd = cmd
			return &mockResult{
				Stdout:   "typed command executed",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	params, _ := FromMap(map[string]interface{}{
		"command":           "git status --short",
		"working_directory": "/tmp/test",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if capturedCmd == nil {
		t.Fatal("command was not captured")
	}

	if capturedCmd.Program != "git" {
		t.Errorf("expected Program 'git', got: %s", capturedCmd.Program)
	}

	if len(capturedCmd.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(capturedCmd.Args))
	}

	if capturedCmd.WorkDir != "/tmp/test" {
		t.Errorf("expected WorkDir '/tmp/test', got: %s", capturedCmd.WorkDir)
	}

	if capturedCmd.Raw != "git status --short" {
		t.Errorf("expected Raw 'git status --short', got: %s", capturedCmd.Raw)
	}
}

func TestExecuteCommandTool_TypedExecutor_WithoutWorkdir(t *testing.T) {
	var capturedCmd *TypedCommand

	executor := &typedExecutor{
		executeFunc: func(ctx context.Context, cmd *TypedCommand, opts interface{}) (interface{}, error) {
			capturedCmd = cmd
			return &mockResult{
				Stdout:   "ok",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	params, _ := FromMap(map[string]interface{}{
		"command": "echo hello",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if capturedCmd == nil {
		t.Fatal("command was not captured")
	}

	// WorkDir should be empty when not provided
	if capturedCmd.WorkDir != "" {
		t.Errorf("expected empty WorkDir, got: %s", capturedCmd.WorkDir)
	}
}

func TestExecuteCommandTool_InvalidMethodSignature(t *testing.T) {
	// Executor with no Execute method
	type badExecutor struct{}

	tool := NewExecuteCommandTool(&badExecutor{}, nil)

	params, _ := FromMap(map[string]interface{}{
		"command": "echo hello",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure result")
	}
}

func TestExecuteCommandTool_UnsupportedCommandType(t *testing.T) {
	// Executor with unsupported command parameter type (not interface or pointer)
	type unsupportedExecutor struct {
		executeFunc func(ctx context.Context, cmd string, opts interface{}) (interface{}, error)
	}

	executor := &unsupportedExecutor{
		executeFunc: func(ctx context.Context, cmd string, opts interface{}) (interface{}, error) {
			return &mockResult{Stdout: "ok", ExitCode: 0}, nil
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	params, _ := FromMap(map[string]interface{}{
		"command": "echo hello",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure result")
	}
}

func TestExecuteCommandTool_ExecuteReturnsError(t *testing.T) {
	executor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return nil, fmt.Errorf("execution failed")
		},
	}

	tool := NewExecuteCommandTool(executor, nil)

	params, _ := FromMap(map[string]interface{}{
		"command": "failing command",
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure result")
	}
}

func TestExecuteCommandTool_Timeout(t *testing.T) {
	// Create a mock executor that simulates a long-running command
	mockExecutor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			// Simulate a command that takes longer than the timeout
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
				return &mockResult{Stdout: "completed", Stderr: "", ExitCode: 0}, nil
			}
		},
	}

	tool := NewExecuteCommandTool(mockExecutor, nil)

	// Test with custom timeout parameter
	params, _ := FromMap(map[string]interface{}{
		"command": "sleep 2",
		"timeout": 1.0, // 1 second timeout
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for command that exceeds timeout")
	}

	// Verify the error indicates timeout
	if !strings.Contains(result.Error, "context deadline exceeded") &&
		!strings.Contains(result.Error, "deadline exceeded") &&
		!strings.Contains(result.Error, "timed out") {
		t.Errorf("Expected timeout error message, got: %s", result.Error)
	}
}

func TestExecuteCommandTool_CustomTimeout(t *testing.T) {
	// Create a mock executor that completes quickly
	mockExecutor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{Stdout: "quick command", Stderr: "", ExitCode: 0}, nil
		},
	}

	tool := NewExecuteCommandTool(mockExecutor, nil)

	// Test with longer timeout that should succeed
	params, _ := FromMap(map[string]interface{}{
		"command": "echo 'custom timeout test'",
		"timeout": 5.0, // 5 second timeout
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success for command within timeout, got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "quick command") {
		t.Errorf("Expected output to contain 'quick command', got: %s", result.Output)
	}
}

func TestExecuteCommandTool_InvalidTimeout(t *testing.T) {
	// Create a mock executor
	mockExecutor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			return &mockResult{Stdout: "invalid timeout test", Stderr: "", ExitCode: 0}, nil
		},
	}

	tool := NewExecuteCommandTool(mockExecutor, nil)

	// Test with invalid timeout parameter (string instead of number)
	params, _ := FromMap(map[string]interface{}{
		"command": "echo 'invalid timeout test'",
		"timeout": "invalid", // Invalid timeout type
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	// Should still succeed because invalid timeout falls back to default
	if !result.Success {
		t.Errorf("Expected success for command with invalid timeout (should fallback to default), got failure: %s", result.Error)
	}

	if !strings.Contains(result.Output, "invalid timeout test") {
		t.Errorf("Expected output to contain 'invalid timeout test', got: %s", result.Output)
	}
}

func TestExecuteCommandTool_ZeroTimeout(t *testing.T) {
	// Create a mock executor
	mockExecutor := &mockExecutor{
		executeFunc: func(ctx context.Context, cmd interface{}, opts interface{}) (interface{}, error) {
			// Zero timeout should cause immediate context cancellation
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return &mockResult{Stdout: "zero timeout test", Stderr: "", ExitCode: 0}, nil
			}
		},
	}

	tool := NewExecuteCommandTool(mockExecutor, nil)

	// Test with zero timeout
	params, _ := FromMap(map[string]interface{}{
		"command": "echo 'zero timeout test'",
		"timeout": 0.0, // Zero timeout
	})

	result, err := tool.Execute(context.Background(), params)

	if err != nil {
		t.Fatalf("Execute returned error instead of ToolResult: %v", err)
	}

	// Zero timeout should cause immediate timeout
	if result.Success {
		t.Error("Expected failure for command with zero timeout (should timeout immediately)")
	}

	// Should be a timeout error
	if !strings.Contains(result.Error, "context deadline exceeded") &&
		!strings.Contains(result.Error, "deadline exceeded") &&
		!strings.Contains(result.Error, "timed out") {
		t.Errorf("Expected timeout error for zero timeout, got: %s", result.Error)
	}
}
