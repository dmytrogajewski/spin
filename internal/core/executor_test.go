package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestResult_Success tests the Success method
func TestResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		want     bool
	}{
		{
			name:     "zero exit code no error",
			exitCode: 0,
			err:      nil,
			want:     true,
		},
		{
			name:     "non-zero exit code",
			exitCode: 1,
			err:      nil,
			want:     false,
		},
		{
			name:     "with error",
			exitCode: 0,
			err:      errors.New("test error"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				ExitCode: tt.exitCode,
				Error:    tt.err,
			}
			if got := r.Success(); got != tt.want {
				t.Errorf("Success() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResult_Failed tests the Failed method
func TestResult_Failed(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		want     bool
	}{
		{
			name:     "zero exit code no error",
			exitCode: 0,
			err:      nil,
			want:     false,
		},
		{
			name:     "non-zero exit code",
			exitCode: 1,
			err:      nil,
			want:     true,
		},
		{
			name:     "with error",
			exitCode: 0,
			err:      errors.New("test error"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				ExitCode: tt.exitCode,
				Error:    tt.err,
			}
			if got := r.Failed(); got != tt.want {
				t.Errorf("Failed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResult_Output tests the Output method
func TestResult_Output(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{
			name:   "both stdout and stderr",
			stdout: "stdout content",
			stderr: "stderr content",
			want:   "stdout content\nstderr content",
		},
		{
			name:   "only stdout",
			stdout: "stdout only",
			stderr: "",
			want:   "stdout only",
		},
		{
			name:   "only stderr",
			stdout: "",
			stderr: "stderr only",
			want:   "stderr only",
		},
		{
			name:   "both empty",
			stdout: "",
			stderr: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				Stdout: tt.stdout,
				Stderr: tt.stderr,
			}
			if got := r.Output(); got != tt.want {
				t.Errorf("Output() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDefaultExecuteOptions tests default options
func TestDefaultExecuteOptions(t *testing.T) {
	opts := DefaultExecuteOptions()

	if opts == nil {
		t.Fatal("DefaultExecuteOptions() returned nil")
	}

	// Timeout and MaxOutputSize are 0 by default (use executor's defaults)
	if opts.Timeout != 0 {
		t.Error("Default timeout should be 0 (use executor's default)")
	}

	if opts.MaxOutputSize != 0 {
		t.Error("Default max output size should be 0 (use executor's default)")
	}

	if opts.ValidateFirst {
		t.Error("ValidateFirst should default to false")
	}
}

// TestNewExecutor tests executor creation
func TestNewExecutor(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		executor, err := NewExecutor(t.TempDir())
		if err != nil {
			t.Fatalf("NewExecutor failed: %v", err)
		}
		if executor == nil {
			t.Fatal("NewExecutor returned nil")
		}
	})

	t.Run("with options", func(t *testing.T) {
		validator := NewValidator()
		executor, err := NewExecutor(
			t.TempDir(),
			WithValidator(validator),
			WithTimeout(30*time.Second),
			WithMaxOutputSize(1024*1024),
		)
		if err != nil {
			t.Fatalf("NewExecutor with options failed: %v", err)
		}
		if executor == nil {
			t.Fatal("NewExecutor returned nil")
		}
	})

	t.Run("empty workdir", func(t *testing.T) {
		_, err := NewExecutor("")
		if err == nil {
			t.Error("Expected error for empty workdir")
		}
	})
}

// TestExecutor_Validate tests pre-execution validation
func TestExecutor_Validate(t *testing.T) {
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}

	t.Run("nil command", func(t *testing.T) {
		err := executor.Validate(nil)
		if err == nil {
			t.Error("Expected error for nil command")
		}
		if !errors.Is(err, ErrNilCommand) {
			t.Errorf("Expected ErrNilCommand, got %v", err)
		}
	})

	t.Run("empty program", func(t *testing.T) {
		cmd := &Command{
			Program: "",
			Args:    []string{},
		}
		err := executor.Validate(cmd)
		if err == nil {
			t.Error("Expected error for empty program")
		}
	})

	t.Run("valid command", func(t *testing.T) {
		cmd := &Command{
			Program: "echo",
			Args:    []string{"test"},
		}
		err := executor.Validate(cmd)
		if err != nil {
			t.Errorf("Valid command should not error: %v", err)
		}
	})
}

// TestExecutor_Validate_WithValidator tests validation with validator
func TestExecutor_Validate_WithValidator(t *testing.T) {
	validator := NewValidator()
	executor, err := NewExecutor(t.TempDir(), WithValidator(validator))
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}

	t.Run("safe command", func(t *testing.T) {
		cmd := &Command{
			Program: "ls",
			Args:    []string{"-la"},
		}
		err := executor.Validate(cmd)
		if err != nil {
			t.Errorf("Safe command should validate: %v", err)
		}
	})

	t.Run("forbidden command", func(t *testing.T) {
		cmd := &Command{
			Program: "rm",
			Args:    []string{"-rf", "/"},
		}
		err := executor.Validate(cmd)
		if err == nil {
			t.Error("Forbidden command should not validate")
		}
	})
}

// TestExecutor_Execute_Success tests successful command execution
func TestExecutor_Execute_Success(t *testing.T) {
	executor, err := NewExecutor(t.TempDir())
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}

	cmd := &Command{
		Program: "echo",
		Args:    []string{"hello world"},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if !result.Success() {
		t.Errorf("Expected success, got failure: exit code %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Stdout = %q, want to contain %q", result.Stdout, "hello world")
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	if result.StartedAt.IsZero() {
		t.Error("Expected non-zero start time")
	}

	if result.CompletedAt.IsZero() {
		t.Error("Expected non-zero completion time")
	}
}

// TestExecutor_Execute_CommandNotFound tests nonexistent command
func TestExecutor_Execute_CommandNotFound(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "nonexistent-command-xyz-12345",
		Args:    []string{},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}

	if result == nil {
		t.Fatal("Result should not be nil even on error")
	}

	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

// TestExecutor_Execute_WithExitCode tests command with non-zero exit
func TestExecutor_Execute_WithExitCode(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "exit 42"},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err == nil {
		t.Error("Expected error for non-zero exit code")
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}

	if result.Success() {
		t.Error("Command should not be successful")
	}

	if !result.Failed() {
		t.Error("Command should be failed")
	}
}

// TestExecutor_Execute_Timeout tests command timeout
func TestExecutor_Execute_Timeout(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sleep",
		Args:    []string{"10"},
	}

	opts := &ExecuteOptions{
		Timeout: 100 * time.Millisecond,
	}

	start := time.Now()
	result, err := executor.Execute(context.Background(), cmd, opts)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if duration >= 1*time.Second {
		t.Errorf("Timeout took too long: %v", duration)
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrTimeout) {
		t.Errorf("Expected timeout error, got %v", err)
	}

	if result != nil && result.Failed() {
		// Result may or may not be returned on timeout
		t.Logf("Result on timeout: exit=%d", result.ExitCode)
	}
}

// TestExecutor_Execute_ContextCancellation tests context cancellation
func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())

	cmd := &Command{
		Program: "sleep",
		Args:    []string{"10"},
	}

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := executor.Execute(ctx, cmd, nil)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected cancellation error")
	}

	if duration >= 1*time.Second {
		t.Errorf("Cancellation took too long: %v", duration)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected Canceled error, got %v", err)
	}
}

// TestExecutor_Execute_OutputCapture tests stdout/stderr capture
func TestExecutor_Execute_OutputCapture(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "echo stdout; echo stderr >&2"},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "stdout") {
		t.Errorf("Stdout missing expected content: %q", result.Stdout)
	}

	if !strings.Contains(result.Stderr, "stderr") {
		t.Errorf("Stderr missing expected content: %q", result.Stderr)
	}

	combined := result.Output()
	if !strings.Contains(combined, "stdout") || !strings.Contains(combined, "stderr") {
		t.Errorf("Combined output missing content: %q", combined)
	}
}

// TestExecutor_Execute_OutputTruncation tests output size limits
func TestExecutor_Execute_OutputTruncation(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	// Generate large output
	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "for i in $(seq 1 1000); do echo 'line '$i; done"},
	}

	opts := &ExecuteOptions{
		MaxOutputSize: 1024, // 1KB limit
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Truncated {
		t.Error("Expected output to be truncated")
	}

	if len(result.Stdout) > 2048 { // Allow some overhead
		t.Errorf("Output size %d exceeds limit", len(result.Stdout))
	}
}

// TestExecutor_Execute_Environment tests environment variables
func TestExecutor_Execute_Environment(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "echo $CUSTOM_VAR"},
	}

	opts := &ExecuteOptions{
		Env: map[string]string{
			"CUSTOM_VAR": "test_value",
		},
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value") {
		t.Errorf("Environment variable not set: %q", result.Stdout)
	}
}

// TestExecutor_Execute_FilterSensitiveEnv tests sensitive var filtering
func TestExecutor_Execute_FilterSensitiveEnv(t *testing.T) {
	// Set sensitive env var
	os.Setenv("MY_SECRET_TOKEN", "sensitive")
	defer os.Unsetenv("MY_SECRET_TOKEN")

	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "env",
		Args:    []string{},
	}

	opts := &ExecuteOptions{
		InheritEnv: true,
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if strings.Contains(result.Stdout, "MY_SECRET_TOKEN") {
		t.Error("Sensitive environment variable leaked")
	}
}

// TestExecutor_Execute_WorkDir tests working directory
func TestExecutor_Execute_WorkDir(t *testing.T) {
	tempDir := t.TempDir()
	executor, _ := NewExecutor(tempDir)

	cmd := &Command{
		Program: "pwd",
		Args:    []string{},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Output should contain the temp directory path
	if !strings.Contains(result.Stdout, tempDir) {
		t.Errorf("PWD output %q does not contain workdir %q", result.Stdout, tempDir)
	}
}

// TestExecutor_Execute_CustomWorkDir tests custom working directory option
func TestExecutor_Execute_CustomWorkDir(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	customDir := t.TempDir()
	cmd := &Command{
		Program: "pwd",
		Args:    []string{},
	}

	opts := &ExecuteOptions{
		WorkDir: customDir,
	}

	result, err := executor.Execute(context.Background(), cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Stdout, customDir) {
		t.Errorf("PWD output %q does not contain custom workdir %q", result.Stdout, customDir)
	}
}

// TestExecutor_ExecuteStreaming_Output tests streaming output
func TestExecutor_ExecuteStreaming_Output(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "for i in 1 2 3; do echo $i; sleep 0.1; done"},
	}

	chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("ExecuteStreaming failed: %v", err)
	}

	var output []string
	for chunk := range chunks {
		if chunk.Error != nil {
			t.Errorf("Chunk error: %v", chunk.Error)
			continue
		}
		if !chunk.Done {
			output = append(output, string(chunk.Data))
		}
	}

	combined := strings.Join(output, "")
	for _, num := range []string{"1", "2", "3"} {
		if !strings.Contains(combined, num) {
			t.Errorf("Missing output: %s", num)
		}
	}
}

// TestExecutor_ExecuteStreaming_Cancellation tests streaming cancellation
func TestExecutor_ExecuteStreaming_Cancellation(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "while true; do echo line; sleep 0.1; done"},
	}

	chunks, err := executor.ExecuteStreaming(ctx, cmd, nil)
	if err != nil {
		t.Fatalf("ExecuteStreaming failed: %v", err)
	}

	// Receive a few chunks then cancel
	count := 0
	for chunk := range chunks {
		if chunk.Error != nil && count > 0 {
			// Error after some chunks is ok (cancellation)
			break
		}
		count++
		if count >= 3 {
			cancel()
		}
	}

	if count < 3 {
		t.Errorf("Expected at least 3 chunks, got %d", count)
	}
}

// TestExecutor_ExecuteStreaming_SeparateStreams tests stdout/stderr separation
func TestExecutor_ExecuteStreaming_SeparateStreams(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	cmd := &Command{
		Program: "sh",
		Args:    []string{"-c", "echo stdout; echo stderr >&2"},
	}

	chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("ExecuteStreaming failed: %v", err)
	}

	hasStdout := false
	hasStderr := false

	for chunk := range chunks {
		if chunk.Error != nil {
			t.Errorf("Unexpected chunk error: %v", chunk.Error)
			continue
		}

		if chunk.Stream == "stdout" && strings.Contains(string(chunk.Data), "stdout") {
			hasStdout = true
		}
		if chunk.Stream == "stderr" && strings.Contains(string(chunk.Data), "stderr") {
			hasStderr = true
		}
	}

	if !hasStdout {
		t.Error("Did not receive stdout chunk")
	}
	if !hasStderr {
		t.Error("Did not receive stderr chunk")
	}
}

// TestExecutor_Integration_RealCommands tests with various real commands
func TestExecutor_Integration_RealCommands(t *testing.T) {
	tests := []struct {
		name       string
		cmd        *Command
		wantStdout string
		wantErr    bool
	}{
		{
			name: "ls command",
			cmd: &Command{
				Program: "ls",
				Args:    []string{},
			},
			wantStdout: "",
			wantErr:    false,
		},
		{
			name: "pwd command",
			cmd: &Command{
				Program: "pwd",
				Args:    []string{},
			},
			wantStdout: "",
			wantErr:    false,
		},
		{
			name: "echo command",
			cmd: &Command{
				Program: "echo",
				Args:    []string{"test message"},
			},
			wantStdout: "test message",
			wantErr:    false,
		},
		{
			name: "date command",
			cmd: &Command{
				Program: "date",
				Args:    []string{},
			},
			wantStdout: "",
			wantErr:    false,
		},
	}

	executor, _ := NewExecutor(t.TempDir())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tt.cmd, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && !result.Success() {
				t.Errorf("Command failed with exit code %d", result.ExitCode)
			}

			if tt.wantStdout != "" && !strings.Contains(result.Stdout, tt.wantStdout) {
				t.Errorf("Stdout = %q, want to contain %q", result.Stdout, tt.wantStdout)
			}
		})
	}
}

// TestExecutor_Integration_ConcurrentExecution tests concurrent execution
func TestExecutor_Integration_ConcurrentExecution(t *testing.T) {
	executor, _ := NewExecutor(t.TempDir())

	const concurrency = 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			cmd := &Command{
				Program: "echo",
				Args:    []string{fmt.Sprintf("test-%d", n)},
			}

			result, err := executor.Execute(context.Background(), cmd, nil)
			if err != nil {
				errors <- err
				return
			}

			expected := fmt.Sprintf("test-%d", n)
			if !strings.Contains(result.Stdout, expected) {
				errors <- fmt.Errorf("unexpected output: %s", result.Stdout)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}
}

// TestExecutor_WithOptions tests functional options
func TestExecutor_WithOptions(t *testing.T) {
	t.Run("WithValidator", func(t *testing.T) {
		validator := NewValidator()
		executor, err := NewExecutor(t.TempDir(), WithValidator(validator))
		if err != nil {
			t.Fatalf("NewExecutor failed: %v", err)
		}

		// Forbidden command should fail validation
		cmd := &Command{
			Program: "rm",
			Args:    []string{"-rf", "/"},
		}

		opts := &ExecuteOptions{
			ValidateFirst: true,
		}

		_, err = executor.Execute(context.Background(), cmd, opts)
		if err == nil {
			t.Error("Expected validation error for forbidden command")
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		executor, err := NewExecutor(t.TempDir(), WithTimeout(100*time.Millisecond))
		if err != nil {
			t.Fatalf("NewExecutor failed: %v", err)
		}

		cmd := &Command{
			Program: "sleep",
			Args:    []string{"10"},
		}

		// Should timeout with default timeout from constructor
		start := time.Now()
		_, err = executor.Execute(context.Background(), cmd, nil)
		duration := time.Since(start)

		if err == nil {
			t.Error("Expected timeout error")
		}

		if duration >= 1*time.Second {
			t.Errorf("Should have timed out faster: %v", duration)
		}
	})

	t.Run("WithMaxOutputSize", func(t *testing.T) {
		executor, err := NewExecutor(t.TempDir(), WithMaxOutputSize(512))
		if err != nil {
			t.Fatalf("NewExecutor failed: %v", err)
		}

		cmd := &Command{
			Program: "sh",
			Args:    []string{"-c", "for i in $(seq 1 1000); do echo 'line '$i; done"},
		}

		result, err := executor.Execute(context.Background(), cmd, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !result.Truncated {
			t.Error("Expected output to be truncated")
		}
	})

	t.Run("WithEnvironment", func(t *testing.T) {
		env := map[string]string{
			"TEST_VAR": "test_value",
		}

		executor, err := NewExecutor(t.TempDir(), WithEnvironment(env))
		if err != nil {
			t.Fatalf("NewExecutor failed: %v", err)
		}

		cmd := &Command{
			Program: "sh",
			Args:    []string{"-c", "echo $TEST_VAR"},
		}

		result, err := executor.Execute(context.Background(), cmd, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !strings.Contains(result.Stdout, "test_value") {
			t.Errorf("Environment variable not set: %q", result.Stdout)
		}
	})
}

// BenchmarkExecutor_Execute_Simple benchmarks simple command execution
func BenchmarkExecutor_Execute_Simple(b *testing.B) {
	executor, _ := NewExecutor(os.TempDir())

	cmd := &Command{
		Program: "echo",
		Args:    []string{"test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(context.Background(), cmd, nil)
	}
}

// BenchmarkExecutor_Execute_WithValidation benchmarks with validation
func BenchmarkExecutor_Execute_WithValidation(b *testing.B) {
	validator := NewValidator()
	executor, _ := NewExecutor(os.TempDir(), WithValidator(validator))

	cmd := &Command{
		Program: "echo",
		Args:    []string{"test"},
	}

	opts := &ExecuteOptions{
		ValidateFirst: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(context.Background(), cmd, opts)
	}
}
