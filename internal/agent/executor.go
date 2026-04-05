package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/process"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/pkg/alg/execx"
)

const (
	executorConcurrency    = 2
	streamingChannelBuffer = 10
	streamReadBufferSize   = 4096
	envKeyValueParts       = 2
)

// Executor-specific errors.
var (
	// ErrNilCommand is a sentinel error.
	ErrNilCommand = errors.New("command is nil")
	// ErrEmptyProgram is a sentinel error.
	ErrEmptyProgram = errors.New("program is empty")
	// ErrCommandNotFound is a sentinel error.
	ErrCommandNotFound = errors.New("command not found")
	// ErrOutputTooLarge is a sentinel error.
	ErrOutputTooLarge = errors.New("output too large")
	// ErrValidationFailed is a sentinel error.
	ErrValidationFailed = errors.New("validation failed")
	// ErrTimeout is a sentinel error.
	ErrTimeout = errors.New("execution timeout")
	// ErrExecutionFailed is a sentinel error.
	ErrExecutionFailed = errors.New("execution failed")
	// ErrSecurityServiceCannotBeNil is a sentinel error.
	ErrSecurityServiceCannotBeNil = errors.New("security service cannot be nil")
	// ErrTimeoutMustBePositive is a sentinel error.
	ErrTimeoutMustBePositive = errors.New("timeout must be positive")
	// ErrWorkdirCannotBeEmpty is a sentinel error.
	ErrWorkdirCannotBeEmpty = errors.New("workDir cannot be empty")
	// ErrCommandExecutionDeniedByUser is a sentinel error.
	ErrCommandExecutionDeniedByUser = errors.New("command execution denied by user")
)

// Default values for execution.
const (
	// DefaultExecutionTimeout is exported.
	DefaultExecutionTimeout = 5 * time.Minute
	// DefaultMaxOutputSize is exported.
	DefaultMaxOutputSize = 10 * 1024 * 1024 // 10MB.
)

// Result contains the outcome of command execution.
type Result struct {
	// Command is the executed command.
	Command *safety.Command

	// Stdout contains the standard output.
	Stdout string

	// Stderr contains the standard error.
	Stderr string

	// ExitCode is the process exit code.
	ExitCode int

	// Duration is the execution time.
	Duration time.Duration

	// StartedAt is when execution started.
	StartedAt time.Time

	// CompletedAt is when execution completed.
	CompletedAt time.Time

	// Error contains any execution error.
	Error error

	// Truncated indicates if output was truncated.
	Truncated bool

	// Metadata contains additional execution metadata.
	Metadata map[string]any
}

// GetMetadata returns execution metadata.
func (r *Result) GetMetadata() map[string]any {
	return r.Metadata
}

// GetStdout returns the standard output.
func (r *Result) GetStdout() string { return r.Stdout }

// GetStderr returns the standard error.
func (r *Result) GetStderr() string { return r.Stderr }

// GetExitCode returns the exit code.
func (r *Result) GetExitCode() int { return r.ExitCode }

// ToCommandResult converts Result to executor.CommandResult.
func (r *Result) ToCommandResult() *executor.CommandResult {
	return &executor.CommandResult{
		Command:     r.Command,
		Stdout:      r.Stdout,
		Stderr:      r.Stderr,
		ExitCode:    r.ExitCode,
		Duration:    r.Duration,
		StartedAt:   r.StartedAt,
		CompletedAt: r.CompletedAt,
		Error:       r.Error,
		Truncated:   r.Truncated,
	}
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
	// Timeout is the maximum execution duration.
	Timeout time.Duration

	// WorkDir is the working directory.
	WorkDir string

	// Env contains environment variables.
	Env map[string]string

	// InheritEnv determines if parent env is inherited.
	InheritEnv bool

	// MaxOutputSize is the maximum output size in bytes (0 = use default).
	MaxOutputSize int64

	// StreamOutput enables real-time output streaming.
	StreamOutput bool

	// ValidateFirst runs validator before execution.
	ValidateFirst bool

	// Sandbox enables sandboxing (if available).
	Sandbox bool
}

// DefaultExecuteOptions returns default execution options.
func DefaultExecuteOptions() *ExecuteOptions {
	return &ExecuteOptions{
		Timeout:       0, // Use executor's default timeout.
		WorkDir:       "",
		Env:           make(map[string]string),
		InheritEnv:    true, // Inherit environment by default.
		MaxOutputSize: 0,    // Use executor's default max output size.
		StreamOutput:  false,
		ValidateFirst: false,
		Sandbox:       false,
	}
}

// OutputChunk represents a chunk of streaming output.
type OutputChunk struct {
	// Stream identifies the stream (stdout/stderr).
	Stream string

	// Data is the output data.
	Data []byte

	// Timestamp is when this chunk was received.
	Timestamp time.Time

	// Done indicates if the stream is complete.
	Done bool

	// Error contains any error.
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
//   - Command result caching
//
// Thread Safety:
//
//	Executor is thread-safe and can execute commands concurrently.
type Executor struct {
	securityService *safety.Service
	approvalService *safety.ApprovalService
	cache           *CommandCache
	workDir         string
	timeout         time.Duration
	maxOutput       int64
	env             map[string]string
	mu              sync.RWMutex
}

// ExecutorOption is a functional option for Executor.
type ExecutorOption func(*Executor) error

// WithSecurityService sets the security service.
func WithSecurityService(s *safety.Service) ExecutorOption {
	return func(e *Executor) error {
		if s == nil {
			return ErrSecurityServiceCannotBeNil
		}

		e.securityService = s

		return nil
	}
}

// WithApprovalService sets the approval service for the executor.
func WithApprovalService(s *safety.ApprovalService) ExecutorOption {
	return func(e *Executor) error {
		e.approvalService = s

		return nil
	}
}

// WithTimeout sets the default timeout.
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) error {
		if d <= 0 {
			return ErrTimeoutMustBePositive
		}

		e.timeout = d

		return nil
	}
}

// WithCache sets the command cache for result caching.
func WithCache(c *CommandCache) ExecutorOption {
	return func(e *Executor) error {
		e.cache = c

		return nil
	}
}

// NewExecutor creates a new command executor with default settings.
func NewExecutor(workDir string, opts ...ExecutorOption) (*Executor, error) {
	if workDir == "" {
		return nil, ErrWorkdirCannotBeEmpty
	}

	cmdExecutor := &Executor{
		workDir:   workDir,
		timeout:   DefaultExecutionTimeout,
		maxOutput: DefaultMaxOutputSize,
		env:       make(map[string]string),
	}

	for _, opt := range opts {
		err := opt(cmdExecutor)
		if err != nil {
			return nil, fmt.Errorf("option failed: %w", err)
		}
	}

	return cmdExecutor, nil
}

// errorResult creates an error result with proper timestamps.
func (e *Executor) errorResult(cmd *safety.Command, err error) *Result {
	now := time.Now()

	return &Result{
		Command:     cmd,
		Error:       err,
		StartedAt:   now,
		CompletedAt: now,
		Duration:    0,
		ExitCode:    -1,
	}
}

// checkCache checks for a cached result.
func (e *Executor) checkCache(cmd *safety.Command) *Result {
	e.mu.RLock()
	cache := e.cache
	e.mu.RUnlock()

	if cache == nil || !cache.IsCacheable(cmd) {
		return nil
	}

	key := cache.Key(cmd)
	if cachedResult, ok := cache.Get(key); ok {
		return cachedResult
	}

	return nil
}

// cacheResultIfEligible caches the result if eligible.
func (e *Executor) cacheResultIfEligible(cmd *safety.Command, result *Result) {
	e.mu.RLock()
	cache := e.cache
	e.mu.RUnlock()

	if cache == nil || result.Error != nil || !cache.IsCacheable(cmd) {
		return
	}

	key := cache.Key(cmd)
	cache.Set(key, result)
}

// validateCommand runs the validation pipeline.
func (e *Executor) validateCommand(cmd *safety.Command, opts *ExecuteOptions) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	// If ValidateFirst option is set, run full validation.
	if opts.ValidateFirst {
		err := e.Validate(cmd)
		if err != nil {
			return err
		}
	}

	return nil
}

// requestApprovalIfNeeded requests approval if command needs it.
func (e *Executor) requestApprovalIfNeeded(ctx context.Context, cmd *safety.Command, opts *ExecuteOptions) error {
	e.mu.RLock()
	securityService := e.securityService
	e.mu.RUnlock()

	// Use SecurityService's high-level approval method (handles validation + approval).
	if securityService == nil {
		return nil // No security service, skip approval.
	}

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = e.workDir
	}

	// Use SecurityService's canonical approval method (handles safe/forbidden/dangerous correctly).
	approved, err := securityService.ValidateAndApprove(ctx, cmd, workDir)
	if err != nil {
		return fmt.Errorf("approval request failed: %w", err)
	}

	if !approved {
		return ErrCommandExecutionDeniedByUser
	}

	return nil
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
func (e *Executor) Validate(cmd *safety.Command) error {
	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	// If security service is present, use it.
	e.mu.RLock()
	securityService := e.securityService
	e.mu.RUnlock()

	if securityService != nil {
		result, err := securityService.ValidateCommand(cmd)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}

		if result.Classification == safety.CommandForbidden {
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
func (e *Executor) Execute(ctx context.Context, cmd *safety.Command, opts *ExecuteOptions) (*Result, error) {
	// Use default options if not provided.
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Run pre-execution pipeline (validation + approval).
	pc := executor.NewPipelineContext(cmd)

	pipeline := e.buildPipeline(opts)
	if err := pipeline.Run(ctx, pc); err != nil {
		return e.errorResult(cmd, err), err
	}

	if pc.Halted {
		return e.errorResult(cmd, pc.HaltErr), pc.HaltErr
	}

	// Check cache.
	if cached := e.checkCache(cmd); cached != nil {
		return cached, nil
	}

	// Execute command.
	result := e.executeCommand(ctx, cmd, opts)

	// Cache successful results.
	e.cacheResultIfEligible(cmd, result)

	return result, result.Error
}

// buildPipeline creates the pre-execution pipeline with configured stages.
func (e *Executor) buildPipeline(opts *ExecuteOptions) *executor.Pipeline {
	return executor.NewPipeline(
		e.validationStage(opts),
		executor.NewPrepareStage(),
		executor.NewDetectStage(),
		e.approvalStage(opts),
	)
}

// validationStage wraps command validation as a pipeline stage.
func (e *Executor) validationStage(opts *ExecuteOptions) executor.Stage {
	return func(_ context.Context, pc *executor.PipelineContext) error {
		return e.validateCommand(pc.Command, opts)
	}
}

// approvalStage wraps approval request as a pipeline stage.
func (e *Executor) approvalStage(opts *ExecuteOptions) executor.Stage {
	return func(ctx context.Context, pc *executor.PipelineContext) error {
		return e.requestApprovalIfNeeded(ctx, pc.Command, opts)
	}
}

// executeCommand performs the actual command execution.
func (e *Executor) executeCommand(ctx context.Context, cmd *safety.Command, opts *ExecuteOptions) *Result {
	result := &Result{
		Command:   cmd,
		StartedAt: time.Now(),
	}

	execCtx, cancel := e.applyTimeout(ctx, opts)
	defer cancel()

	return e.executeAndCapture(execCtx, cmd, opts, result)
}

// executeAndCapture runs the command, waits, and captures output.
func (e *Executor) executeAndCapture(execCtx context.Context, cmd *safety.Command, opts *ExecuteOptions, result *Result) *Result {
	execCmd := e.prepareExecCmd(execCtx, cmd, opts)

	var stdoutBuf, stderrBuf bytes.Buffer

	execCmd.Stdout = &stdoutBuf
	execCmd.Stderr = &stderrBuf

	err := execCmd.Start()
	if err != nil {
		result.Error = fmt.Errorf("%w: %w", ErrCommandNotFound, err)
		result.ExitCode = -1
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(result.StartedAt)

		return result
	}

	err = execCmd.Wait()
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	maxSize := e.getMaxOutputSize(opts)
	result.Stdout, result.Stderr, result.Truncated = e.captureOutput(&stdoutBuf, &stderrBuf, maxSize)

	if err != nil {
		e.handleExecError(execCtx, err, result)

		return result
	}

	result.ExitCode = 0

	return result
}

// prepareExecCmd creates and configures an [exec.Cmd].
func (e *Executor) prepareExecCmd(execCtx context.Context, cmd *safety.Command, opts *ExecuteOptions) *exec.Cmd {
	execCmd := exec.CommandContext(execCtx, cmd.Program, cmd.Args...)

	workDir := opts.WorkDir
	if workDir == "" {
		workDir = e.workDir
	}

	execCmd.Dir = workDir
	execCmd.Env = e.buildEnvironment(opts)

	ensurePATH(execCmd)
	process.SetGroup(execCmd)

	// Override context cancellation to kill the entire process group.
	execCmd.Cancel = func() error {
		return process.KillGroup(execCmd)
	}

	return execCmd
}

// ensurePATH ensures the [exec.Cmd] has a PATH environment variable.
func ensurePATH(execCmd *exec.Cmd) {
	for _, env := range execCmd.Env {
		if strings.HasPrefix(env, "PATH=") {
			return
		}
	}

	execCmd.Env = append(execCmd.Env, "PATH=/usr/bin:/bin")
}

// getMaxOutputSize returns the max output size from opts or the executor default.
func (e *Executor) getMaxOutputSize(opts *ExecuteOptions) int64 {
	if opts.MaxOutputSize > 0 {
		return opts.MaxOutputSize
	}

	e.mu.RLock()
	maxSize := e.maxOutput
	e.mu.RUnlock()

	return maxSize
}

// classifyExecError maps a context and exec error to the appropriate sentinel error.
// It checks for timeout and cancellation via the context, and falls back to wrapping
// the original error as an execution failure.
func classifyExecError(execCtx context.Context, err error) error {
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return ErrTimeout
	}

	if errors.Is(execCtx.Err(), context.Canceled) {
		return context.Canceled
	}

	return fmt.Errorf("%w: %w", ErrExecutionFailed, err)
}

// handleExecError sets error and exit code on the result based on exec error type.
func (e *Executor) handleExecError(execCtx context.Context, err error, result *Result) {
	result.Error = classifyExecError(execCtx, err)

	classified := result.Error
	if errors.Is(classified, ErrTimeout) || errors.Is(classified, context.Canceled) {
		result.ExitCode = -1

		return
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
}

// ExecuteStreaming runs a command and streams output in real-time.
//
// This method is useful for long-running commands where you want to display
// output progressively. The returned channel receives output chunks as
// they arrive and is closed when execution completes.
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
func (e *Executor) ExecuteStreaming(ctx context.Context, cmd *safety.Command, opts *ExecuteOptions) (<-chan OutputChunk, error) {
	// Use default options if not provided.
	if opts == nil {
		opts = DefaultExecuteOptions()
	}

	// Validate command.
	if err := e.validateStreamingCommand(cmd, opts); err != nil {
		return nil, err
	}

	// Apply timeout.
	execCtx, cancel := e.applyTimeout(ctx, opts)

	// Prepare and start the command with pipes.
	execCmd, stdout, stderr, err := e.prepareStreamingCmd(execCtx, cmd, opts)
	if err != nil {
		cancel()

		return nil, err
	}

	err = execCmd.Start()
	if err != nil {
		cancel()

		return nil, fmt.Errorf("%w: %w", ErrCommandNotFound, err)
	}

	// Create output channel and run streaming goroutines.
	chunks := make(chan OutputChunk, streamingChannelBuffer)
	e.runStreamingGoroutines(execCtx, execCmd, stdout, stderr, chunks, cancel)

	return chunks, nil
}

// validateStreamingCommand validates a command for streaming execution.
func (e *Executor) validateStreamingCommand(cmd *safety.Command, opts *ExecuteOptions) error {
	if opts.ValidateFirst {
		return e.Validate(cmd)
	}

	if cmd == nil {
		return ErrNilCommand
	}

	if cmd.Program == "" {
		return ErrEmptyProgram
	}

	return nil
}

// prepareStreamingCmd creates an [exec.Cmd] with stdout/stderr pipes for streaming.
func (e *Executor) prepareStreamingCmd(
	execCtx context.Context, cmd *safety.Command, opts *ExecuteOptions,
) (execCmd *exec.Cmd, stdout, stderr io.Reader, err error) {
	execCmd = e.prepareExecCmd(execCtx, cmd, opts)

	stdout, err = execCmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err = execCmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	return execCmd, stdout, stderr, nil
}

// runStreamingGoroutines starts goroutines to stream stdout/stderr and wait for completion.
func (e *Executor) runStreamingGoroutines(
	execCtx context.Context, execCmd *exec.Cmd,
	stdout, stderr io.Reader, chunks chan OutputChunk,
	cancel context.CancelFunc,
) {
	var wg sync.WaitGroup

	wg.Add(executorConcurrency)

	go func() {
		defer wg.Done()

		e.streamOutput(execCtx, stdout, "stdout", chunks)
	}()

	go func() {
		defer wg.Done()

		e.streamOutput(execCtx, stderr, "stderr", chunks)
	}()

	go func() {
		defer cancel()
		defer close(chunks)

		wg.Wait()

		waitErr := execCmd.Wait()
		chunks <- e.buildStreamingCompletion(execCtx, waitErr)
	}()
}

// buildStreamingCompletion creates the final OutputChunk for a streaming command.
func (e *Executor) buildStreamingCompletion(execCtx context.Context, waitErr error) OutputChunk {
	if waitErr == nil {
		return OutputChunk{Timestamp: time.Now(), Done: true}
	}

	completionErr := classifyExecError(execCtx, waitErr)

	return OutputChunk{Timestamp: time.Now(), Done: true, Error: completionErr}
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

	// Start with inherited environment (if enabled).
	if opts.InheritEnv {
		for _, kv := range os.Environ() {
			parts := strings.SplitN(kv, "=", envKeyValueParts)
			if len(parts) == envKeyValueParts && !isSensitive(parts[0]) {
				env[parts[0]] = parts[1]
			}
		}
	}

	// Add executor environment.
	e.mu.RLock()

	maps.Copy(env, e.env)

	e.mu.RUnlock()

	// Add command-specific environment.
	maps.Copy(env, opts.Env)

	// Convert to slice.
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// isSensitive checks if an environment variable is sensitive.
// Delegates to execx.IsSensitiveKey using the canonical lists from environment.go.
func isSensitive(key string) bool {
	return execx.IsSensitiveKey(key, sensitivePrefixes, sensitiveSubstrings)
}

// captureOutput captures command output with size limits.
func (e *Executor) captureOutput(stdout, stderr io.Reader, maxSize int64) (stdoutStr, stderrStr string, truncated bool) {
	stdoutStr, t1 := captureLimited(stdout, maxSize)
	stderrStr, t2 := captureLimited(stderr, maxSize)

	return stdoutStr, stderrStr, t1 || t2
}

// captureLimited reads up to maxSize bytes from r, marking output as truncated if exceeded.
func captureLimited(r io.Reader, maxSize int64) (string, bool) {
	data, err := io.ReadAll(io.LimitReader(r, maxSize))
	truncated := err != nil

	if int64(len(data)) >= maxSize {
		truncated = true

		data = append(data, []byte("\n... (output truncated)")...)
	}

	return string(data), truncated
}

// streamOutput streams data from a reader to the output channel.
// It checks context cancellation periodically to allow graceful shutdown.
func (e *Executor) streamOutput(ctx context.Context, r io.Reader, stream string, chunks chan<- OutputChunk) {
	buf := make([]byte, streamReadBufferSize)

	for {
		// Check context cancellation before reading.
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := r.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			if !sendChunk(ctx, chunks, OutputChunk{Stream: stream, Data: data, Timestamp: time.Now()}) {
				return
			}
		}

		if err != nil {
			if err != io.EOF {
				sendChunk(ctx, chunks, OutputChunk{
					Stream:    stream,
					Timestamp: time.Now(),
					Error:     fmt.Errorf("stream error: %w", err),
				})
			}

			break
		}
	}
}

// sendChunk sends a chunk to the channel or returns false if context is canceled.
func sendChunk(ctx context.Context, chunks chan<- OutputChunk, chunk OutputChunk) bool {
	select {
	case chunks <- chunk:
		return true
	case <-ctx.Done():
		return false
	}
}
