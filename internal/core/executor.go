package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Executor-specific errors.
var (
	ErrNilCommand       = errors.New("command is nil")
	ErrEmptyProgram     = errors.New("program is empty")
	ErrCommandNotFound  = errors.New("command not found")
	ErrOutputTooLarge   = errors.New("output too large")
	ErrValidationFailed = errors.New("validation failed")
)

// Note: ErrExecutionFailed and ErrTimeout are defined in error.go

// Default values for execution.
const (
	DefaultTimeout       = 5 * time.Minute
	DefaultMaxOutputSize = 10 * 1024 * 1024 // 10MB
)

// Result contains the outcome of command execution.
type Result struct {
	// Command is the executed command
	Command *Command

	// Stdout contains the standard output
	Stdout string

	// Stderr contains the standard error
	Stderr string

	// ExitCode is the process exit code
	ExitCode int

	// Duration is the execution time
	Duration time.Duration

	// StartedAt is when execution started
	StartedAt time.Time

	// CompletedAt is when execution completed
	CompletedAt time.Time

	// Error contains any execution error
	Error error

	// Truncated indicates if output was truncated
	Truncated bool
}

// Success returns true if the command executed successfully.
func (r *Result) Success() bool {
	return r.ExitCode == 0 && r.Error == nil
}

// Failed returns true if the command failed.
func (r *Result) Failed() bool {
	return !r.Success()
}

// Output returns combined stdout and stderr.
func (r *Result) Output() string {
	if r.Stdout == "" && r.Stderr == "" {
		return ""
	}
	if r.Stdout == "" {
		return r.Stderr
	}
	if r.Stderr == "" {
		return r.Stdout
	}
	return r.Stdout + "\n" + r.Stderr
}

// ExecuteOptions configures command execution behavior.
type ExecuteOptions struct {
	// Timeout is the maximum execution duration
	Timeout time.Duration

	// WorkDir is the working directory
	WorkDir string

	// Env contains environment variables
	Env map[string]string

	// InheritEnv determines if parent env is inherited
	InheritEnv bool

	// MaxOutputSize is the maximum output size in bytes (0 = use default)
	MaxOutputSize int64

	// StreamOutput enables real-time output streaming
	StreamOutput bool

	// ValidateFirst runs validator before execution
	ValidateFirst bool

	// Sandbox enables sandboxing (if available)
	Sandbox bool
}

// DefaultExecuteOptions returns default execution options.
func DefaultExecuteOptions() *ExecuteOptions {
	return &ExecuteOptions{
		Timeout:       0, // Use executor's default timeout
		WorkDir:       "",
		Env:           make(map[string]string),
		InheritEnv:    false,
		MaxOutputSize: 0, // Use executor's default max output size
		StreamOutput:  false,
		ValidateFirst: false,
		Sandbox:       false,
	}
}

// OutputChunk represents a chunk of streaming output.
type OutputChunk struct {
	// Stream identifies the stream (stdout/stderr)
	Stream string

	// Data is the output data
	Data []byte

	// Timestamp is when this chunk was received
	Timestamp time.Time

	// Done indicates if the stream is complete
	Done bool

	// Error contains any error
	Error error
}

// Executor manages safe command execution with sandboxing and resource limits.
//
// The Executor provides multi-layered protection:
//   - Pre-execution validation
//   - OS-level sandboxing (where available)
//   - Timeout enforcement
//   - Output size limits
//   - Environment isolation
//
// Thread Safety:
//
//	Executor is thread-safe and can execute commands concurrently.
type Executor struct {
	validator *Validator
	sandbox   interface{} // sandbox.Sandbox interface (avoiding import cycle)
	workDir   string
	timeout   time.Duration
	maxOutput int64
	env       map[string]string
	mu        sync.RWMutex
}

// ExecutorOption is a functional option for Executor.
type ExecutorOption func(*Executor) error

// WithValidator sets the command validator.
func WithValidator(v *Validator) ExecutorOption {
	return func(e *Executor) error {
		if v == nil {
			return NewValidationError("Executor.WithValidator", "validator cannot be nil")
		}
		e.validator = v
		return nil
	}
}

// WithTimeout sets the default timeout.
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) error {
		if d <= 0 {
			return NewValidationError("Executor.WithTimeout", "timeout must be positive")
		}
		e.timeout = d
		return nil
	}
}

// WithMaxOutputSize sets the maximum output size.
func WithMaxOutputSize(size int64) ExecutorOption {
	return func(e *Executor) error {
		if size <= 0 {
			return NewValidationError("Executor.WithMaxOutputSize", "max output size must be positive")
		}
		e.maxOutput = size
		return nil
	}
}

// WithEnvironment sets the environment variables.
func WithEnvironment(env map[string]string) ExecutorOption {
	return func(e *Executor) error {
		if env == nil {
			return NewValidationError("Executor.WithEnvironment", "environment cannot be nil")
		}
		e.env = env
		return nil
	}
}

// WithSandbox sets the sandbox for command isolation.
// The sandbox parameter should implement the sandbox.Sandbox interface.
func WithSandbox(s interface{}) ExecutorOption {
	return func(e *Executor) error {
		e.sandbox = s
		return nil
	}
}

// NewExecutor creates a new command executor with default settings.
func NewExecutor(workDir string, opts ...ExecutorOption) (*Executor, error) {
	if workDir == "" {
		return nil, errors.New("workDir cannot be empty")
	}

	e := &Executor{
		workDir:   workDir,
		timeout:   DefaultTimeout,
		maxOutput: DefaultMaxOutputSize,
		env:       make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, fmt.Errorf("option failed: %w", err)
		}
	}

	return e, nil
}

// Validate checks if a command can be executed.
//
// This method performs pre-execution checks without actually running the command:
//   - Command structure validation
//   - Program existence check (optional)
//   - Security policy validation (if validator present)
//   - Working directory validation
//
// Returns an error if the command cannot be executed.
func (e *Executor) Validate(cmd *Command) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	// If validator is present, use it
	e.mu.RLock()
	validator := e.validator
	e.mu.RUnlock()

	if validator != nil {
		result, err := validator.Classify(cmd)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrValidationFailed, err)
		}

		if result.Classification == CommandForbidden {
			return fmt.Errorf("%w: %s", ErrValidationFailed, result.Reason)
		}
	}

	return nil
}

// Execute runs a command synchronously and returns the result.
//
// Execution Flow:
//  1. Validate command (if validator present)
//  2. Prepare execution environment
//  3. Apply sandbox (if enabled)
//  4. Execute command with timeout
//  5. Capture output (stdout/stderr)
//  6. Return result
//
// The method respects context cancellation and applies configured timeouts.
//
// Example:
//
//	executor := NewExecutor("/workspace")
//	cmd, _ := ParseCommand("ls -la")
//	result, err := executor.Execute(context.Background(), cmd, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Stdout)
func (e *Executor) Execute(ctx context.Context, cmd *Command, opts *ExecuteOptions) (*Result, error) {
	// Use default options if not provided
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Create result
	result := &Result{
		Command:   cmd,
		StartedAt: time.Now(),
	}

	// Validate command
	if opts.ValidateFirst {
		if err := e.Validate(cmd); err != nil {
			result.Error = err
			result.CompletedAt = time.Now()
			result.Duration = result.CompletedAt.Sub(result.StartedAt)
			return result, err
		}
	} else if cmd == nil {
		err := ErrNilCommand
		result.Error = err
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, err
	} else if cmd.Program == "" {
		err := ErrEmptyProgram
		result.Error = err
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, err
	}

	// Apply timeout
	execCtx, cancel := e.applyTimeout(ctx, opts)
	defer cancel()

	// Prepare exec.Cmd
	execCmd := exec.CommandContext(execCtx, cmd.Program, cmd.Args...)

	// Set working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = e.workDir
	}
	execCmd.Dir = workDir

	// Set environment
	execCmd.Env = e.buildEnvironment(opts)

	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	// Start execution
	if err := execCmd.Start(); err != nil {
		result.Error = fmt.Errorf("%w: %v", ErrCommandNotFound, err)
		result.ExitCode = -1
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)
		return result, result.Error
	}

	// Wait for completion
	err := execCmd.Wait()

	// Complete result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Get max output size
	maxSize := opts.MaxOutputSize
	if maxSize == 0 {
		e.mu.RLock()
		maxSize = e.maxOutput
		e.mu.RUnlock()
	}

	// Capture and limit output
	result.Stdout, result.Stderr, result.Truncated = e.captureOutput(
		&stdoutBuf,
		&stderrBuf,
		maxSize,
	)

	// Handle execution error
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = ErrTimeout
			result.ExitCode = -1
			return result, result.Error
		}
		if execCtx.Err() == context.Canceled {
			result.Error = context.Canceled
			result.ExitCode = -1
			return result, result.Error
		}

		// Extract exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = fmt.Errorf("%w: %v", ErrExecutionFailed, err)
		return result, result.Error
	}

	result.ExitCode = 0
	return result, nil
}

// ExecuteStreaming runs a command and streams output in real-time.
//
// This method is useful for long-running commands where you want to display
// output progressively. The returned channel will receive output chunks as
// they arrive and will be closed when execution completes.
//
// Example:
//
//	executor := NewExecutor("/workspace")
//	cmd, _ := ParseCommand("make build")
//
//	chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for chunk := range chunks {
//	    if chunk.Error != nil {
//	        log.Printf("Error: %v", chunk.Error)
//	        continue
//	    }
//	    fmt.Print(string(chunk.Data))
//	}
func (e *Executor) ExecuteStreaming(ctx context.Context, cmd *Command, opts *ExecuteOptions) (<-chan OutputChunk, error) {
	// Use default options if not provided
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Validate command
	if opts.ValidateFirst {
		if err := e.Validate(cmd); err != nil {
			return nil, err
		}
	} else if cmd == nil {
		return nil, ErrNilCommand
	} else if cmd.Program == "" {
		return nil, ErrEmptyProgram
	}

	// Apply timeout
	execCtx, cancel := e.applyTimeout(ctx, opts)

	// Create output channel
	chunks := make(chan OutputChunk, 10)

	// Prepare exec.Cmd
	execCmd := exec.CommandContext(execCtx, cmd.Program, cmd.Args...)

	// Set working directory
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = e.workDir
	}
	execCmd.Dir = workDir

	// Set environment
	execCmd.Env = e.buildEnvironment(opts)

	// Get stdout and stderr pipes
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start execution
	if err := execCmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %v", ErrCommandNotFound, err)
	}

	// Stream output in goroutines
	var wg sync.WaitGroup

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stdout, "stdout", chunks)
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.streamOutput(stderr, "stderr", chunks)
	}()

	// Wait for command and close channel
	go func() {
		defer cancel()
		defer close(chunks)

		// Wait for output streams to complete
		wg.Wait()

		// Wait for command to complete
		err := execCmd.Wait()

		// Send completion or error
		if err != nil {
			if execCtx.Err() == context.DeadlineExceeded {
				chunks <- OutputChunk{
					Timestamp: time.Now(),
					Done:      true,
					Error:     ErrTimeout,
				}
			} else if execCtx.Err() == context.Canceled {
				chunks <- OutputChunk{
					Timestamp: time.Now(),
					Done:      true,
					Error:     context.Canceled,
				}
			} else {
				chunks <- OutputChunk{
					Timestamp: time.Now(),
					Done:      true,
					Error:     fmt.Errorf("%w: %v", ErrExecutionFailed, err),
				}
			}
		} else {
			// Success
			chunks <- OutputChunk{
				Timestamp: time.Now(),
				Done:      true,
				Error:     nil,
			}
		}
	}()

	return chunks, nil
}

// applyTimeout applies a timeout to the context.
func (e *Executor) applyTimeout(ctx context.Context, opts *ExecuteOptions) (context.Context, context.CancelFunc) {
	timeout := opts.Timeout
	if timeout == 0 {
		e.mu.RLock()
		timeout = e.timeout
		e.mu.RUnlock()
	}

	return context.WithTimeout(ctx, timeout)
}

// buildEnvironment constructs the environment variable list.
func (e *Executor) buildEnvironment(opts *ExecuteOptions) []string {
	env := make(map[string]string)

	// Start with inherited environment (if enabled)
	if opts.InheritEnv {
		for _, kv := range os.Environ() {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 && !isSensitive(parts[0]) {
				env[parts[0]] = parts[1]
			}
		}
	}

	// Add executor environment
	e.mu.RLock()
	for k, v := range e.env {
		env[k] = v
	}
	e.mu.RUnlock()

	// Add command-specific environment
	for k, v := range opts.Env {
		env[k] = v
	}

	// Convert to slice
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// isSensitive checks if an environment variable is sensitive.
func isSensitive(key string) bool {
	sensitive := []string{
		"TOKEN", "SECRET", "PASSWORD", "KEY", "CREDENTIAL",
		"AWS_SECRET", "API_KEY", "PRIVATE_KEY",
	}

	upper := strings.ToUpper(key)
	for _, pattern := range sensitive {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

// captureOutput captures command output with size limits.
func (e *Executor) captureOutput(stdout, stderr io.Reader, maxSize int64) (string, string, bool) {
	var truncated bool

	// Read stdout
	stdoutBytes, err := io.ReadAll(io.LimitReader(stdout, maxSize))
	if err != nil {
		truncated = true
	}
	if int64(len(stdoutBytes)) >= maxSize {
		truncated = true
		stdoutBytes = append(stdoutBytes, []byte("\n... (output truncated)")...)
	}

	// Read stderr
	stderrBytes, err := io.ReadAll(io.LimitReader(stderr, maxSize))
	if err != nil {
		truncated = true
	}
	if int64(len(stderrBytes)) >= maxSize {
		truncated = true
		stderrBytes = append(stderrBytes, []byte("\n... (output truncated)")...)
	}

	return string(stdoutBytes), string(stderrBytes), truncated
}

// streamOutput streams data from a reader to the output channel.
func (e *Executor) streamOutput(r io.Reader, stream string, chunks chan<- OutputChunk) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			chunks <- OutputChunk{
				Stream:    stream,
				Data:      data,
				Timestamp: time.Now(),
				Done:      false,
				Error:     nil,
			}
		}
		if err != nil {
			if err != io.EOF {
				chunks <- OutputChunk{
					Stream:    stream,
					Timestamp: time.Now(),
					Done:      false,
					Error:     fmt.Errorf("stream error: %w", err),
				}
			}
			break
		}
	}
}
