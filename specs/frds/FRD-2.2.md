# FRD-2.2: Command Executor

**Feature ID:** 2.2  
**Feature Name:** Command Executor  
**Phase:** Phase 2 - Safety & Execution  
**Priority:** P0 (Blocker - Security Critical)  
**Estimated Effort:** 14 hours  
**Status:** Ready for Implementation

---

## Overview

Implement safe command execution with sandboxing, timeouts, and result capture. This is a critical security component that safely executes validated commands with proper isolation, resource limits, and comprehensive output capture.

## Context

The **Executor** is the execution engine for Spin's command system. It takes validated commands from the Validator and executes them safely with:

- **Sandboxing:** OS-level isolation to prevent system damage
- **Timeouts:** Resource limits to prevent runaway processes
- **Output Capture:** Comprehensive stdout/stderr capture for AI feedback
- **Environment Management:** Controlled environment variable handling
- **Working Directory Control:** Restricted file system access

This is a **security-critical** component that must be thoroughly tested and hardened against exploitation.

## Definition of Ready (DoR)

- [x] Feature 2.1 (Command Validator) completed
- [ ] `internal/security` package structure defined
- [ ] Execution environment specifications documented
- [ ] Timeout policies defined
- [ ] Sandbox strategy determined per OS

## Definition of Done (DoD)

- [ ] `executor.go` fully implemented with Executor struct
- [ ] Execute() method with full execution flow
- [ ] ExecuteStreaming() for long-running commands
- [ ] Validate() pre-execution checks
- [ ] Sandbox integration (basic implementation)
- [ ] Timeout enforcement using context
- [ ] Output capture (stdout/stderr separation)
- [ ] Exit code handling
- [ ] Environment variable management
- [ ] Working directory management
- [ ] Unit tests for executor (>90% coverage)
- [ ] Integration tests with real commands
- [ ] Timeout tests with cancellation
- [ ] Error handling tests
- [ ] Godoc comments for all exported symbols
- [ ] Code analyzed with uast/herr (complexity <15)
- [ ] All linters passing

---

## Requirements

### Functional Requirements

#### FR-2.2.1: Command Execution Result

**Description:** Structure to capture command execution results.

**Type:**

```go
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

// Success returns true if the command executed successfully
func (r *Result) Success() bool

// Failed returns true if the command failed
func (r *Result) Failed() bool

// Output returns combined stdout and stderr
func (r *Result) Output() string
```

**Acceptance Criteria:**
- Captures all execution information
- Includes timing information
- Handles large outputs with truncation
- Clear success/failure determination

---

#### FR-2.2.2: Execution Options

**Description:** Configuration options for command execution.

**Type:**

```go
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
    
    // MaxOutputSize is the maximum output size in bytes (0 = unlimited)
    MaxOutputSize int64
    
    // StreamOutput enables real-time output streaming
    StreamOutput bool
    
    // ValidateFirst runs validator before execution
    ValidateFirst bool
    
    // Sandbox enables sandboxing (if available)
    Sandbox bool
}

// DefaultExecuteOptions returns default execution options
func DefaultExecuteOptions() *ExecuteOptions
```

**Acceptance Criteria:**
- Comprehensive configuration options
- Sensible defaults
- Clear documentation
- Validation of options

---

#### FR-2.2.3: Executor Structure

**Description:** Main executor structure with dependencies.

**Type:**

```go
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
//   Executor is thread-safe and can execute commands concurrently.
type Executor struct {
    validator *Validator
    workDir   string
    timeout   time.Duration
    maxOutput int64
    env       map[string]string
    mu        sync.RWMutex
}

// NewExecutor creates a new command executor with default settings
func NewExecutor(workDir string, opts ...ExecutorOption) (*Executor, error)

// ExecutorOption is a functional option for Executor
type ExecutorOption func(*Executor) error

// WithValidator sets the command validator
func WithValidator(v *Validator) ExecutorOption

// WithTimeout sets the default timeout
func WithTimeout(d time.Duration) ExecutorOption

// WithMaxOutputSize sets the maximum output size
func WithMaxOutputSize(size int64) ExecutorOption

// WithEnvironment sets the environment variables
func WithEnvironment(env map[string]string) ExecutorOption
```

**Acceptance Criteria:**
- Clear structure with dependencies
- Functional options pattern
- Thread-safe implementation
- Well-documented

---

#### FR-2.2.4: Execute Method

**Description:** Main synchronous execution method.

**Method:**

```go
// Execute runs a command synchronously and returns the result.
//
// Execution Flow:
//   1. Validate command (if validator present)
//   2. Prepare execution environment
//   3. Apply sandbox (if enabled)
//   4. Execute command with timeout
//   5. Capture output (stdout/stderr)
//   6. Return result
//
// The method respects context cancellation and applies configured timeouts.
//
// Example:
//   executor := NewExecutor("/workspace")
//   cmd, _ := ParseCommand("ls -la")
//   result, err := executor.Execute(context.Background(), cmd, nil)
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Println(result.Stdout)
func (e *Executor) Execute(ctx context.Context, cmd *Command, opts *ExecuteOptions) (*Result, error)
```

**Execution Flow:**

```
1. Input Validation
   ├─→ Check command is not nil
   ├─→ Check program exists
   └─→ Validate with Validator (if present)

2. Options Setup
   ├─→ Merge provided options with defaults
   ├─→ Apply timeout to context
   └─→ Prepare working directory

3. Environment Setup
   ├─→ Build environment variables
   ├─→ Filter sensitive variables
   └─→ Add custom variables

4. Sandbox Preparation (if enabled)
   ├─→ Check OS support
   ├─→ Prepare sandbox profile
   └─→ Wrap command with sandbox

5. Command Execution
   ├─→ Create exec.Cmd
   ├─→ Set stdout/stderr pipes
   ├─→ Start process
   ├─→ Wait for completion or timeout
   └─→ Handle cancellation

6. Output Processing
   ├─→ Collect stdout
   ├─→ Collect stderr
   ├─→ Check output size limits
   └─→ Truncate if necessary

7. Result Construction
   ├─→ Capture exit code
   ├─→ Calculate duration
   ├─→ Add metadata
   └─→ Return result
```

**Acceptance Criteria:**
- Executes commands safely
- Respects context cancellation
- Applies timeouts correctly
- Captures all output
- Handles errors gracefully
- Thread-safe

---

#### FR-2.2.5: ExecuteStreaming Method

**Description:** Streaming execution for long-running commands.

**Method:**

```go
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

// ExecuteStreaming runs a command and streams output in real-time.
//
// This method is useful for long-running commands where you want to display
// output progressively. The returned channel will receive output chunks as
// they arrive and will be closed when execution completes.
//
// Example:
//   executor := NewExecutor("/workspace")
//   cmd, _ := ParseCommand("make build")
//   
//   chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
//   if err != nil {
//       log.Fatal(err)
//   }
//   
//   for chunk := range chunks {
//       if chunk.Error != nil {
//           log.Printf("Error: %v", chunk.Error)
//           continue
//       }
//       fmt.Print(string(chunk.Data))
//   }
func (e *Executor) ExecuteStreaming(ctx context.Context, cmd *Command, opts *ExecuteOptions) (<-chan OutputChunk, error)
```

**Acceptance Criteria:**
- Streams output in real-time
- Separates stdout and stderr
- Handles cancellation
- Closes channel on completion
- Reports errors via channel

---

#### FR-2.2.6: Validate Method

**Description:** Pre-execution validation.

**Method:**

```go
// Validate checks if a command can be executed.
//
// This method performs pre-execution checks without actually running the command:
//   - Command structure validation
//   - Program existence check (optional)
//   - Security policy validation (if validator present)
//   - Working directory validation
//
// Returns an error if the command cannot be executed.
func (e *Executor) Validate(cmd *Command) error
```

**Acceptance Criteria:**
- Validates command structure
- Checks program existence
- Uses validator if available
- Clear error messages
- No side effects

---

#### FR-2.2.7: Sandbox Integration

**Description:** OS-specific sandboxing support.

**Interface:**

```go
// Sandbox defines the interface for command sandboxing.
type Sandbox interface {
    // Wrap wraps a command with sandbox restrictions
    Wrap(cmd *Command) (*Command, error)
    
    // Supported returns true if sandboxing is supported on this OS
    Supported() bool
    
    // Name returns the sandbox implementation name
    Name() string
}

// NoopSandbox is a no-op sandbox for unsupported platforms
type NoopSandbox struct{}

func (s *NoopSandbox) Wrap(cmd *Command) (*Command, error) {
    return cmd, nil
}

func (s *NoopSandbox) Supported() bool {
    return false
}

func (s *NoopSandbox) Name() string {
    return "none"
}
```

**Platform Support:**

| Platform | Sandbox | Status |
|----------|---------|--------|
| Linux | Landlock LSM | Phase 8 |
| macOS | sandbox-exec | Phase 8 |
| Windows | AppContainer | Phase 8 |
| Other | No-op | Phase 2 |

**Acceptance Criteria:**
- Interface defined
- No-op implementation for Phase 2
- Clear OS support indication
- Ready for Phase 8 integration

---

#### FR-2.2.8: Timeout Handling

**Description:** Robust timeout and cancellation handling.

**Implementation:**

```go
// applyTimeout applies a timeout to the context
func (e *Executor) applyTimeout(ctx context.Context, opts *ExecuteOptions) (context.Context, context.CancelFunc) {
    timeout := opts.Timeout
    if timeout == 0 {
        timeout = e.timeout
    }
    if timeout == 0 {
        timeout = DefaultTimeout // 5 minutes
    }
    
    return context.WithTimeout(ctx, timeout)
}

// handleTimeout handles timeout errors
func (e *Executor) handleTimeout(ctx context.Context, cmd *exec.Cmd) error {
    select {
    case <-ctx.Done():
        // Kill process on timeout
        if cmd.Process != nil {
            cmd.Process.Kill()
        }
        return fmt.Errorf("command timeout: %w", ctx.Err())
    }
}
```

**Acceptance Criteria:**
- Respects context cancellation
- Applies configured timeouts
- Kills processes on timeout
- Clear timeout errors
- Graceful process termination

---

#### FR-2.2.9: Environment Management

**Description:** Safe environment variable handling.

**Implementation:**

```go
// buildEnvironment constructs the environment variable list
func (e *Executor) buildEnvironment(opts *ExecuteOptions) []string {
    env := make(map[string]string)
    
    // Start with executor defaults (if inherit is enabled)
    if opts.InheritEnv {
        for _, kv := range os.Environ() {
            parts := strings.SplitN(kv, "=", 2)
            if len(parts) == 2 && !isSensitive(parts[0]) {
                env[parts[0]] = parts[1]
            }
        }
    }
    
    // Add executor environment
    for k, v := range e.env {
        env[k] = v
    }
    
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

// isSensitive checks if an environment variable is sensitive
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
```

**Acceptance Criteria:**
- Filters sensitive variables
- Supports inheritance
- Allows overrides
- Clear precedence rules
- Well-documented

---

#### FR-2.2.10: Output Capture

**Description:** Comprehensive output capture with size limits.

**Implementation:**

```go
// captureOutput captures command output with size limits
func (e *Executor) captureOutput(stdout, stderr io.Reader, maxSize int64) (string, string, bool) {
    var outBuf, errBuf bytes.Buffer
    var truncated bool
    
    // Create limited readers
    outReader := io.LimitReader(stdout, maxSize)
    errReader := io.LimitReader(stderr, maxSize)
    
    // Read stdout
    outBytes, outErr := io.ReadAll(outReader)
    if outErr != nil {
        truncated = true
    }
    if int64(len(outBytes)) >= maxSize {
        truncated = true
        outBytes = append(outBytes, []byte("\n... (output truncated)")...)
    }
    outBuf.Write(outBytes)
    
    // Read stderr
    errBytes, errErr := io.ReadAll(errReader)
    if errErr != nil {
        truncated = true
    }
    if int64(len(errBytes)) >= maxSize {
        truncated = true
        errBytes = append(errBytes, []byte("\n... (output truncated)")...)
    }
    errBuf.Write(errBytes)
    
    return outBuf.String(), errBuf.String(), truncated
}
```

**Acceptance Criteria:**
- Captures stdout and stderr separately
- Enforces size limits
- Indicates truncation
- Handles read errors
- No memory exhaustion

---

### Non-Functional Requirements

#### NFR-2.2.1: Performance

- Command execution overhead: <10ms
- Small command (ls): <50ms total
- Streaming latency: <100ms to first chunk
- Memory per execution: <10MB

#### NFR-2.2.2: Security

- No command injection vulnerabilities
- Proper process isolation
- Timeout enforcement (no runaway processes)
- Output size limits (no memory exhaustion)
- Environment variable filtering
- Working directory restrictions

#### NFR-2.2.3: Reliability

- Graceful timeout handling
- Proper process cleanup
- Error recovery
- Resource leak prevention
- Context cancellation support

#### NFR-2.2.4: Testability

- >90% test coverage
- Integration tests with real commands
- Timeout tests
- Cancellation tests
- Error path coverage

#### NFR-2.2.5: Maintainability

- Clear separation of concerns
- Well-documented code
- Extensible sandbox interface
- Configuration through options
- Logging for debugging

---

## Design

### Architecture

```
┌─────────────────────────────────────────┐
│             Executor                    │
├─────────────────────────────────────────┤
│  - validator *Validator                 │
│  - workDir string                       │
│  - timeout time.Duration                │
│  - maxOutput int64                      │
│  - env map[string]string                │
│  - mu sync.RWMutex                      │
├─────────────────────────────────────────┤
│  + Execute(ctx, cmd, opts) *Result     │
│  + ExecuteStreaming(ctx, cmd, opts)    │
│  + Validate(cmd) error                  │
└─────────────────────────────────────────┘
         │
         │ uses
         ▼
┌─────────────────────────────────────────┐
│           Validator                     │
│  (from Feature 2.1)                     │
└─────────────────────────────────────────┘
         │
         │ validates
         ▼
┌─────────────────────────────────────────┐
│            Command                      │
│  (from Feature 2.1)                     │
└─────────────────────────────────────────┘
         │
         │ executes to
         ▼
┌─────────────────────────────────────────┐
│            Result                       │
├─────────────────────────────────────────┤
│  - Command *Command                     │
│  - Stdout string                        │
│  - Stderr string                        │
│  - ExitCode int                         │
│  - Duration time.Duration               │
│  - Error error                          │
└─────────────────────────────────────────┘
```

### State Machine

```
┌─────────────┐
│   Created   │
└──────┬──────┘
       │
       │ Execute() / ExecuteStreaming()
       ▼
┌─────────────┐
│ Validating  │
└──────┬──────┘
       │
       │ Success
       ▼
┌─────────────┐
│  Preparing  │ (env, workdir, sandbox)
└──────┬──────┘
       │
       │ Ready
       ▼
┌─────────────┐
│  Executing  │
└──────┬──────┘
       │
       ├─→ Success ──→ ┌───────────┐
       │               │ Completed │
       │               └───────────┘
       │
       ├─→ Timeout ──→ ┌───────────┐
       │               │  Timeout  │
       │               └───────────┘
       │
       ├─→ Cancel ───→ ┌───────────┐
       │               │ Cancelled │
       │               └───────────┘
       │
       └─→ Error ────→ ┌───────────┐
                       │   Failed  │
                       └───────────┘
```

---

## Implementation Plan

### Task Breakdown

#### Task 1: Define result types (1.5 hours)
- [ ] Create Result struct
- [ ] Implement Success() and Failed() methods
- [ ] Implement Output() method
- [ ] Add godoc comments
- [ ] Write result tests

#### Task 2: Define execution options (1 hour)
- [ ] Create ExecuteOptions struct
- [ ] Implement DefaultExecuteOptions()
- [ ] Add validation for options
- [ ] Write options tests

#### Task 3: Implement Executor struct (1 hour)
- [ ] Define Executor struct
- [ ] Implement NewExecutor() constructor
- [ ] Implement functional options
- [ ] Add godoc comments
- [ ] Write constructor tests

#### Task 4: Implement Validate method (1 hour)
- [ ] Implement pre-execution validation
- [ ] Integrate with Validator
- [ ] Check program existence
- [ ] Write validation tests

#### Task 5: Implement environment management (1.5 hours)
- [ ] Implement buildEnvironment()
- [ ] Implement isSensitive() filtering
- [ ] Handle environment inheritance
- [ ] Write environment tests

#### Task 6: Implement output capture (1.5 hours)
- [ ] Implement captureOutput()
- [ ] Add size limit enforcement
- [ ] Handle truncation
- [ ] Write capture tests

#### Task 7: Implement Execute method (2.5 hours)
- [ ] Implement full execution flow
- [ ] Add timeout handling
- [ ] Integrate output capture
- [ ] Add error handling
- [ ] Write execution tests

#### Task 8: Implement ExecuteStreaming (2 hours)
- [ ] Implement streaming execution
- [ ] Create output channel
- [ ] Handle real-time output
- [ ] Add completion detection
- [ ] Write streaming tests

#### Task 9: Implement no-op sandbox (0.5 hours)
- [ ] Define Sandbox interface
- [ ] Implement NoopSandbox
- [ ] Integrate with executor
- [ ] Write sandbox tests

#### Task 10: Integration tests (1.5 hours)
- [ ] Write integration tests with real commands
- [ ] Test timeout scenarios
- [ ] Test cancellation
- [ ] Test error handling
- [ ] Test concurrent execution

#### Task 11: Testing and polish (1 hour)
- [ ] Achieve >90% test coverage
- [ ] Add benchmarks
- [ ] Run linters
- [ ] Analyze with uast/herr
- [ ] Complete documentation

---

## Testing Strategy

### Unit Tests

#### Basic Execution Tests

```go
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
    
    if !result.Success() {
        t.Errorf("Expected success, got failure")
    }
    
    if !strings.Contains(result.Stdout, "hello world") {
        t.Errorf("Stdout = %q, want %q", result.Stdout, "hello world")
    }
}

func TestExecutor_Execute_CommandNotFound(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    cmd := &Command{
        Program: "nonexistent-command-12345",
        Args:    []string{},
    }
    
    result, err := executor.Execute(context.Background(), cmd, nil)
    if err == nil {
        t.Error("Expected error for nonexistent command")
    }
    if result != nil && result.ExitCode == 0 {
        t.Error("Expected non-zero exit code")
    }
}

func TestExecutor_Execute_WithExitCode(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    // sh -c "exit 42"
    cmd := &Command{
        Program: "sh",
        Args:    []string{"-c", "exit 42"},
    }
    
    result, err := executor.Execute(context.Background(), cmd, nil)
    if err == nil {
        t.Error("Expected error for non-zero exit code")
    }
    
    if result.ExitCode != 42 {
        t.Errorf("ExitCode = %d, want %d", result.ExitCode, 42)
    }
}
```

#### Timeout Tests

```go
func TestExecutor_Execute_Timeout(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    // sleep 10
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
    
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("Expected DeadlineExceeded, got %v", err)
    }
}

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
    
    result, err := executor.Execute(ctx, cmd, nil)
    
    if err == nil {
        t.Error("Expected cancellation error")
    }
    
    if !errors.Is(err, context.Canceled) {
        t.Errorf("Expected Canceled, got %v", err)
    }
}
```

#### Output Capture Tests

```go
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
}

func TestExecutor_Execute_OutputTruncation(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    // Generate 1MB of output
    cmd := &Command{
        Program: "sh",
        Args:    []string{"-c", "dd if=/dev/zero bs=1024 count=1024 | base64"},
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
```

#### Environment Tests

```go
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

func TestExecutor_Execute_FilterSensitive(t *testing.T) {
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
```

#### Streaming Tests

```go
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
        output = append(output, string(chunk.Data))
    }
    
    combined := strings.Join(output, "")
    for _, num := range []string{"1", "2", "3"} {
        if !strings.Contains(combined, num) {
            t.Errorf("Missing output: %s", num)
        }
    }
}

func TestExecutor_ExecuteStreaming_Cancellation(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    ctx, cancel := context.WithCancel(context.Background())
    
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
        count++
        if count >= 3 {
            cancel()
        }
    }
    
    if count < 3 {
        t.Errorf("Expected at least 3 chunks, got %d", count)
    }
}
```

#### Validation Tests

```go
func TestExecutor_Validate_WithValidator(t *testing.T) {
    validator := NewValidator()
    executor, _ := NewExecutor(t.TempDir(), WithValidator(validator))
    
    // Safe command
    cmd := &Command{
        Program: "ls",
        Args:    []string{"-la"},
    }
    
    err := executor.Validate(cmd)
    if err != nil {
        t.Errorf("Safe command should validate: %v", err)
    }
    
    // Dangerous command
    cmd = &Command{
        Program: "rm",
        Args:    []string{"-rf", "/"},
    }
    
    err = executor.Validate(cmd)
    if err == nil {
        t.Error("Dangerous command should not validate")
    }
}

func TestExecutor_Validate_NilCommand(t *testing.T) {
    executor, _ := NewExecutor(t.TempDir())
    
    err := executor.Validate(nil)
    if err == nil {
        t.Error("Nil command should not validate")
    }
}
```

### Integration Tests

```go
func TestExecutor_Integration_RealCommands(t *testing.T) {
    tests := []struct {
        name       string
        cmd        *Command
        wantStdout string
        wantErr    bool
    }{
        {
            name: "ls current directory",
            cmd: &Command{
                Program: "ls",
                Args:    []string{},
            },
            wantStdout: "",
            wantErr:    false,
        },
        {
            name: "pwd",
            cmd: &Command{
                Program: "pwd",
                Args:    []string{},
            },
            wantStdout: "",
            wantErr:    false,
        },
        {
            name: "echo",
            cmd: &Command{
                Program: "echo",
                Args:    []string{"test message"},
            },
            wantStdout: "test message",
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
            
            if tt.wantStdout != "" && !strings.Contains(result.Stdout, tt.wantStdout) {
                t.Errorf("Stdout = %q, want %q", result.Stdout, tt.wantStdout)
            }
        })
    }
}

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
```

### Benchmark Tests

```go
func BenchmarkExecutor_Execute_Simple(b *testing.B) {
    executor, _ := NewExecutor(os.TempDir())
    
    cmd := &Command{
        Program: "echo",
        Args:    []string{"test"},
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        executor.Execute(context.Background(), cmd, nil)
    }
}

func BenchmarkExecutor_Execute_WithValidation(b *testing.B) {
    validator := NewValidator()
    executor, _ := NewExecutor(os.TempDir(), WithValidator(validator))
    
    cmd := &Command{
        Program: "echo",
        Args:    []string{"test"},
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        executor.Execute(context.Background(), cmd, nil)
    }
}
```

---

## Error Handling

### Error Types

```go
var (
    ErrNilCommand       = errors.New("command is nil")
    ErrEmptyProgram     = errors.New("program is empty")
    ErrCommandNotFound  = errors.New("command not found")
    ErrExecutionFailed  = errors.New("execution failed")
    ErrTimeout          = errors.New("execution timeout")
    ErrOutputTooLarge   = errors.New("output too large")
    ErrValidationFailed = errors.New("validation failed")
)
```

### Error Wrapping

```go
func (e *Executor) Execute(ctx context.Context, cmd *Command, opts *ExecuteOptions) (*Result, error) {
    if cmd == nil {
        return nil, ErrNilCommand
    }
    
    if err := e.Validate(cmd); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrValidationFailed, err)
    }
    
    // ... execution ...
    
    if err := execCmd.Wait(); err != nil {
        return result, fmt.Errorf("%w: %v", ErrExecutionFailed, err)
    }
    
    return result, nil
}
```

---

## Dependencies

### Internal Dependencies
- `internal/core/validator.go` - Command validation
- `internal/core/error.go` - Error types

### Standard Library
- `os/exec` - Command execution
- `context` - Cancellation and timeouts
- `io` - Output capture
- `time` - Duration and timing
- `sync` - Thread safety

### Future Dependencies
- `internal/security/sandbox` - Sandboxing (Phase 8)

---

## Examples

### Basic Execution

```go
// Create executor
executor, err := NewExecutor("/workspace")
if err != nil {
    log.Fatal(err)
}

// Parse command
cmd := &Command{
    Program: "ls",
    Args:    []string{"-la"},
}

// Execute
result, err := executor.Execute(context.Background(), cmd, nil)
if err != nil {
    log.Printf("Execution failed: %v", err)
    return
}

// Check result
if result.Success() {
    fmt.Println("Output:")
    fmt.Println(result.Stdout)
} else {
    fmt.Printf("Command failed with exit code: %d\n", result.ExitCode)
    fmt.Println("Error output:")
    fmt.Println(result.Stderr)
}
```

### With Options

```go
executor, _ := NewExecutor("/workspace")

cmd := &Command{
    Program: "make",
    Args:    []string{"build"},
}

opts := &ExecuteOptions{
    Timeout:       5 * time.Minute,
    MaxOutputSize: 10 * 1024 * 1024, // 10MB
    Env: map[string]string{
        "CGO_ENABLED": "1",
        "GOOS":        "linux",
    },
}

result, err := executor.Execute(context.Background(), cmd, opts)
// ...
```

### Streaming Execution

```go
executor, _ := NewExecutor("/workspace")

cmd := &Command{
    Program: "go",
    Args:    []string{"test", "-v", "./..."},
}

chunks, err := executor.ExecuteStreaming(context.Background(), cmd, nil)
if err != nil {
    log.Fatal(err)
}

for chunk := range chunks {
    if chunk.Error != nil {
        log.Printf("Error: %v", chunk.Error)
        continue
    }
    
    if chunk.Stream == "stdout" {
        fmt.Print(string(chunk.Data))
    } else {
        fmt.Fprint(os.Stderr, string(chunk.Data))
    }
}
```

### With Validation

```go
validator := NewValidator()
executor, _ := NewExecutor("/workspace", WithValidator(validator))

cmd, _ := ParseCommand("rm -rf /")

// Validation happens automatically
result, err := executor.Execute(context.Background(), cmd, nil)
if err != nil {
    // Will fail validation
    log.Printf("Blocked: %v", err)
}
```

### With Timeout and Cancellation

```go
executor, _ := NewExecutor("/workspace")

// Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

cmd := &Command{
    Program: "npm",
    Args:    []string{"install"},
}

// Execute with context
result, err := executor.Execute(ctx, cmd, nil)
if errors.Is(err, context.DeadlineExceeded) {
    fmt.Println("Command timed out")
} else if errors.Is(err, context.Canceled) {
    fmt.Println("Command canceled")
}
```

---

## Success Criteria

- [ ] All DoD items checked off
- [ ] Test coverage >90%
- [ ] All execution paths tested
- [ ] Timeout handling verified
- [ ] Output capture works correctly
- [ ] Environment management secure
- [ ] Linters passing
- [ ] Code complexity <15 (verified with uast/herr)
- [ ] Documentation complete
- [ ] Integration tests pass
- [ ] Concurrent execution safe
- [ ] Ready for use by Feature 6.1 (Agent)

---

## References

- [Core Module Spec](../core-module/spec.md)
- [Security Modules Spec](../security-modules.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [FRD-2.1: Command Validator](./FRD-2.1.md)
- [Go os/exec Package](https://pkg.go.dev/os/exec)
- [Go context Package](https://pkg.go.dev/context)

---

**Created:** 2025-10-03  
**Author:** Development Team  
**Status:** 🚧 Ready for Implementation  
**Dependencies:** Feature 2.1 (Command Validator)

