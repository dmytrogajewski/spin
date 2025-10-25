# Spin Test Suite

This directory contains the end-to-end (e2e) and integration tests for the Spin project.

## Directory Structure

```
tests/
├── e2e/                    # End-to-end tests
│   ├── e2e_test.go                # Core e2e test framework
│   ├── tool_execution_e2e_test.go # Tool execution scenarios
│   └── tui_e2e_test.go            # TUI interaction tests
└── README.md               # This file
```

## Test Organization

### Unit Tests

Unit tests are located alongside the code they test in the `internal/` directory:

```
internal/
├── agent/
│   ├── agent.go
│   └── agent_test.go      # Unit tests for agent
├── llm/
│   ├── provider.go
│   └── provider_test.go   # Unit tests for LLM providers
└── ...
```

**Naming Convention:** `*_test.go` files in the same package

**Run unit tests:**
```bash
# Run all unit tests
go test ./internal/... -v

# Run tests for specific package
go test ./internal/agent/... -v

# Run with race detection
go test ./internal/... -race

# Run with coverage
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests verify interactions between components and are named `*_integration_test.go`:

```
internal/appserver/processor_integration_test.go
```

**Run integration tests:**
```bash
# Run all tests including integration
go test ./internal/... -v

# Run only integration tests
go test ./internal/... -v -run Integration
```

### End-to-End Tests

E2E tests in `tests/e2e/` verify complete workflows and user scenarios.

**Run e2e tests:**
```bash
# Run all e2e tests
go test ./tests/e2e/... -v

# Run specific e2e test
go test ./tests/e2e/... -v -run TestToolExecution

# Run with timeout
go test ./tests/e2e/... -v -timeout 30m
```

## Test Helpers

Use the `internal/testutil` package for common test utilities:

```go
import "github.com/dmytrogajewski/spin/internal/testutil"

func TestMyFeature(t *testing.T) {
    // Build test fixtures with sensible defaults
    agent := testutil.NewAgentBuilder(t).
        WithMaxTurns(5).
        WithTimeout(10 * time.Second).
        Build()

    // Use test context with timeout
    ctx, cancel := testutil.ContextWithTimeout(t)
    defer cancel()

    // Execute and assert
    result, err := agent.Execute(ctx, req)
    testutil.RequireNoError(t, err)
    testutil.AssertNotNil(t, result)
}
```

## Writing Tests

### Table-Driven Tests

Prefer table-driven tests for testing multiple scenarios:

```go
func TestAgentExecute(t *testing.T) {
    tests := []struct {
        name    string
        input   *AgentRequest
        want    *AgentResponse
        wantErr bool
    }{
        {
            name: "successful execution",
            input: &AgentRequest{
                Input: "test input",
            },
            want: &AgentResponse{
                Output: "test output",
            },
            wantErr: false,
        },
        {
            name: "error case",
            input: &AgentRequest{
                Input: "",
            },
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := testutil.NewAgentBuilder(t).Build()
            
            got, err := agent.Execute(context.Background(), tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Execute() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Using Subtests

Use `t.Run()` for subtests to organize related test cases:

```go
func TestAgent(t *testing.T) {
    t.Run("Creation", func(t *testing.T) {
        t.Run("ValidConfig", func(t *testing.T) {
            // Test valid configuration
        })
        
        t.Run("InvalidConfig", func(t *testing.T) {
            // Test invalid configuration
        })
    })
    
    t.Run("Execution", func(t *testing.T) {
        t.Run("Success", func(t *testing.T) {
            // Test successful execution
        })
        
        t.Run("Timeout", func(t *testing.T) {
            // Test timeout handling
        })
    })
}
```

### Mock Providers

Use mock providers for testing without external dependencies:

```go
func TestWithMockLLM(t *testing.T) {
    mockLLM := testutil.NewMockLLMProvider("test")
    
    agent := testutil.NewAgentBuilder(t).
        WithProvider(mockLLM).
        Build()
    
    // Test agent behavior with mock LLM
}
```

### Test Context and Timeouts

Always use context with timeout to prevent hanging tests:

```go
func TestOperation(t *testing.T) {
    ctx, cancel := testutil.ContextWithTimeout(t)
    defer cancel()
    
    result, err := operation(ctx)
    testutil.RequireNoError(t, err)
}
```

## Test Best Practices

### 1. Use t.Helper()

Mark helper functions with `t.Helper()` for better error messages:

```go
func setupAgent(t *testing.T) *Agent {
    t.Helper()
    return testutil.NewAgentBuilder(t).Build()
}
```

### 2. Clean Up Resources

Always clean up resources using `defer`:

```go
func TestWithFile(t *testing.T) {
    f, err := os.CreateTemp("", "test")
    testutil.RequireNoError(t, err)
    defer os.Remove(f.Name())
    
    // Test code
}
```

### 3. Use Parallel Tests

Mark independent tests as parallel:

```go
func TestParallel(t *testing.T) {
    t.Parallel()
    
    // Test code that doesn't depend on global state
}
```

### 4. Test Error Cases

Always test both success and error paths:

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        errMsg  string
    }{
        {
            name:    "success",
            input:   "valid",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
            errMsg:  "input cannot be empty",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate(tt.input)
            
            if tt.wantErr {
                testutil.RequireError(t, err)
                testutil.AssertContains(t, err.Error(), tt.errMsg)
            } else {
                testutil.RequireNoError(t, err)
            }
        })
    }
}
```

### 5. Use Descriptive Test Names

Test names should describe what is being tested:

```go
// Good
func TestAgent_Execute_WithTimeout_ReturnsError(t *testing.T) {}
func TestLLM_Complete_WithInvalidModel_ReturnsNotFoundError(t *testing.T) {}

// Avoid
func TestAgent1(t *testing.T) {}
func TestError(t *testing.T) {}
```

## Running Tests

### All Tests

```bash
# Run all tests
make test

# With coverage
make test-coverage

# With race detection
make test-race
```

### Specific Packages

```bash
# Test specific package
go test ./internal/agent/... -v

# Test with filter
go test ./internal/... -v -run TestAgent

# Test with short mode (skip long-running tests)
go test ./... -short
```

### Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# Coverage by package
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Benchmarks

```bash
# Run benchmarks
go test ./... -bench=. -benchmem

# Run specific benchmark
go test ./internal/agent/... -bench=BenchmarkExecute

# With memory allocations
go test ./... -bench=. -benchmem -benchtime=10s
```

## Continuous Integration

Tests run automatically on GitHub Actions (see `.github/workflows/test.yml`):

- **On Push:** All tests run on Linux, macOS, and Windows
- **On PR:** Full test suite with coverage reporting
- **Nightly:** Extended test suite with race detection

## Test Coverage Goals

| Package | Target Coverage | Current |
|---------|----------------|---------|
| `internal/agent` | 85% | Check CI |
| `internal/llm` | 80% | Check CI |
| `internal/tools` | 80% | Check CI |
| `internal/security` | 90% | Check CI |
| Overall | 80% | Check CI |

## Troubleshooting

### Tests Hang

- Check for missing context cancellation
- Verify timeouts are set appropriately
- Look for deadlocks in concurrent code

### Flaky Tests

- Identify race conditions with `-race` flag
- Check for timing dependencies
- Verify proper cleanup in `defer` statements

### Slow Tests

- Use `-short` flag to skip slow tests during development
- Profile tests with `-cpuprofile` and `-memprofile`
- Consider mocking expensive operations

## Contributing

When adding new features:

1. Write tests first (TDD)
2. Aim for >80% coverage
3. Include both success and error cases
4. Use table-driven tests for multiple scenarios
5. Add integration/e2e tests for user-facing features
6. Update this README if adding new test categories

## Further Reading

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Advanced Testing with Go](https://www.youtube.com/watch?v=8hQG7QlcLBk)
- [Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
