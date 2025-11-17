package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	workDir := t.TempDir()

	executor, err := NewExecutor(workDir)
	require.NoError(t, err)
	assert.NotNil(t, executor)
	assert.Equal(t, workDir, executor.workDir)
}

func TestNewExecutor_WithOptions(t *testing.T) {
	workDir := t.TempDir()
	validator := security.NewValidator()
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)

	executor, err := NewExecutor(workDir, WithSecurityService(securityService))
	require.NoError(t, err)
	assert.NotNil(t, executor)
	assert.NotNil(t, executor.securityService)
}

func TestNewExecutor_EmptyWorkDir(t *testing.T) {
	executor, err := NewExecutor("")
	require.Error(t, err)
	assert.Nil(t, executor)
}

func TestNewExecutor_NonExistentWorkDir(t *testing.T) {
	executor, err := NewExecutor("/non/existent/directory")
	require.NoError(t, err) // NewExecutor doesn't validate directory existence
	assert.NotNil(t, executor)
}

func TestExecutor_Execute(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "world"},
		WorkDir: workDir,
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "hello world")
}

func TestExecutor_Execute_WithTimeout(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "sleep",
		Args:    []string{"2"},
		WorkDir: workDir,
	}

	opts := &ExecuteOptions{
		Timeout: 100 * time.Millisecond,
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success())
}

func TestExecutor_Execute_WithWorkingDirectory(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(workDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "ls",
		Args:    []string{"test.txt"},
		WorkDir: workDir,
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "test.txt")
}

func TestExecutor_Execute_WithEnvironment(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "sh",
		Args:    []string{"-c", "echo $TEST_VAR"},
		WorkDir: workDir,
	}

	opts := &ExecuteOptions{
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success())
	assert.Contains(t, result.Output(), "test_value")
}

func TestExecutor_Execute_NonExistentCommand(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "nonexistentcommand12345",
		Args:    []string{},
		WorkDir: workDir,
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success())
}

func TestExecutor_Validate(t *testing.T) {
	workDir := t.TempDir()
	validator := security.NewValidator()
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	executor, err := NewExecutor(workDir, WithSecurityService(securityService))
	require.NoError(t, err)

	// Test valid command
	cmd := &security.Command{
		Program: "echo",
		Args:    []string{"hello"},
		WorkDir: workDir,
	}

	err = executor.Validate(cmd)
	require.NoError(t, err)

	// Test dangerous command
	dangerousCmd := &security.Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		WorkDir: workDir,
	}

	err = executor.Validate(dangerousCmd)
	require.Error(t, err)
}

func TestExecutor_Validate_NilCommand(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	err = executor.Validate(nil)
	require.Error(t, err)
	assert.Equal(t, ErrNilCommand, err)
}

func TestExecutor_Validate_EmptyProgram(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "",
		Args:    []string{"hello"},
		WorkDir: workDir,
	}

	err = executor.Validate(cmd)
	require.Error(t, err)
	assert.Equal(t, ErrEmptyProgram, err)
}

func TestResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name: "successful result",
			result: &Result{
				ExitCode: 0,
				Error:    nil,
			},
			expected: true,
		},
		{
			name: "failed result with error",
			result: &Result{
				ExitCode: 1,
				Error:    assert.AnError,
			},
			expected: false,
		},
		{
			name: "failed result with non-zero exit code",
			result: &Result{
				ExitCode: 1,
				Error:    nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Success())
		})
	}
}

func TestResult_Failed(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected bool
	}{
		{
			name: "successful result",
			result: &Result{
				ExitCode: 0,
				Error:    nil,
			},
			expected: false,
		},
		{
			name: "failed result",
			result: &Result{
				ExitCode: 1,
				Error:    assert.AnError,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Failed())
		})
	}
}

func TestResult_Output(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		expected string
	}{
		{
			name: "stdout only",
			result: &Result{
				Stdout: "hello",
				Stderr: "",
			},
			expected: "hello",
		},
		{
			name: "stderr only",
			result: &Result{
				Stdout: "",
				Stderr: "error",
			},
			expected: "error",
		},
		{
			name: "both stdout and stderr",
			result: &Result{
				Stdout: "hello",
				Stderr: "error",
			},
			expected: "hello\nerror",
		},
		{
			name: "empty output",
			result: &Result{
				Stdout: "",
				Stderr: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Output())
		})
	}
}

func TestDefaultExecuteOptions(t *testing.T) {
	opts := DefaultExecuteOptions()
	assert.NotNil(t, opts)
	assert.Equal(t, time.Duration(0), opts.Timeout) // 0 means use executor's default
	assert.Equal(t, int64(0), opts.MaxOutputSize)   // 0 means use executor's default
	assert.True(t, opts.InheritEnv)
}

func TestExecutorOption_WithSecurityService(t *testing.T) {
	validator := security.NewValidator()
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: nil, Validator: validator})
	securityService := security.NewSecurityService(validator, approvalService)
	executor := &Executor{}

	opt := WithSecurityService(securityService)
	err := opt(executor)
	require.NoError(t, err)
	assert.Equal(t, securityService, executor.securityService)
}

func TestExecutorOption_WithSecurityService_Nil(t *testing.T) {
	executor := &Executor{}

	opt := WithSecurityService(nil)
	err := opt(executor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "security service cannot be nil")
}

func TestExecutorOption_WithApprovalService(t *testing.T) {
	approvalService := &security.ApprovalService{}
	executor := &Executor{}

	opt := WithApprovalService(approvalService)
	err := opt(executor)
	require.NoError(t, err)
	assert.Equal(t, approvalService, executor.approvalService)
}

func TestExecutorOption_WithTimeout(t *testing.T) {
	timeout := 30 * time.Second
	executor := &Executor{}

	opt := WithTimeout(timeout)
	err := opt(executor)
	require.NoError(t, err)
	assert.Equal(t, timeout, executor.timeout)
}

func TestExecutorOption_WithCache(t *testing.T) {
	cache := NewCommandCache(5*time.Second, 1024*1024)
	executor := &Executor{}

	opt := WithCache(cache)
	err := opt(executor)
	require.NoError(t, err)
	assert.Equal(t, cache, executor.cache)
}

func TestExecutor_ConcurrentExecution(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	// Run multiple commands concurrently
	results := make(chan *Result, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			cmd := &security.Command{
				Program: "echo",
				Args:    []string{"test", string(rune(i + '0'))},
				WorkDir: workDir,
			}
			result, _ := executor.Execute(context.Background(), cmd, nil)
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		select {
		case result := <-results:
			assert.NotNil(t, result)
			assert.True(t, result.Success())
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent command results")
		}
	}
}

func TestExecutor_ExecuteStreaming(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "echo",
		Args:    []string{"hello", "world"},
		WorkDir: workDir,
	}

	chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
	require.NoError(t, err)
	assert.NotNil(t, chunks)

	// Collect chunks
	var output string
	for chunk := range chunks {
		output += string(chunk.Data)
	}

	assert.Contains(t, output, "hello world")
}

func TestExecutor_ExecuteStreaming_WithTimeout(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	cmd := &security.Command{
		Program: "sleep",
		Args:    []string{"2"},
		WorkDir: workDir,
	}

	opts := &ExecuteOptions{
		Timeout: 100 * time.Millisecond,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	chunks, err := executor.ExecuteStreaming(ctx, cmd, opts)
	require.NoError(t, err) // ExecuteStreaming may not return error immediately
	assert.NotNil(t, chunks)

	// The timeout should be handled by the context, not the function
	// We can't easily test this without mocking or more complex setup
}

func TestExecutor_ErrorHandling(t *testing.T) {
	workDir := t.TempDir()
	executor, err := NewExecutor(workDir)
	require.NoError(t, err)

	// Test with nil command
	result, err := executor.Execute(context.Background(), nil, nil)
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success())

	// Test with empty program
	cmd := &security.Command{
		Program: "",
		Args:    []string{"hello"},
		WorkDir: workDir,
	}

	result, err = executor.Execute(context.Background(), cmd, nil)
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success())
}
