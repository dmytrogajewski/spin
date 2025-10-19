# Testing Patterns in Spin

This document describes the testing patterns and best practices used throughout the Spin codebase.

## Table-Driven Tests

The primary pattern used in Spin is **table-driven tests**, following Go best practices:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "happy path",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "error case",
            input:   invalidInput,
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Test Organization

### File Naming

- Test files: `*_test.go`
- Coverage-focused tests: `*_coverage_test.go`
- Integration tests: `*_integration_test.go`
- E2E tests: `*_e2e_test.go`

### Package Structure

Tests are co-located with the code they test in the same package:

```
internal/core/
  ├── validator.go
  ├── validator_test.go
  ├── validator_coverage_test.go
  ├── git_integration.go
  └── git_integration_coverage_test.go
```

## Common Patterns

### Testing with Mock Dependencies

```go
type mockTokenizer struct{}

func (m *mockTokenizer) Count(text string) int {
    return len(text) / 4
}

func (m *mockTokenizer) CountMessages(messages []Message) int {
    total := 0
    for _, msg := range messages {
        total += m.Count(msg.Content)
    }
    return total
}
```

### Testing with Temporary Directories

```go
func TestWithTempDir(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    // Use tmpDir for file operations
}
```

### Testing Concurrent Code

```go
func TestConcurrentAccess(t *testing.T) {
    done := make(chan bool, 10)
    
    for i := 0; i < 10; i++ {
        go func() {
            // Test concurrent operations
            done <- true
        }()
    }
    
    timeout := time.After(5 * time.Second)
    for i := 0; i < 10; i++ {
        select {
        case <-done:
            // Success
        case <-timeout:
            t.Fatal("Test timed out")
        }
    }
}
```

### Testing Event Emission

```go
func TestEventEmission(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    // Run operation in goroutine to allow event emission
    done := make(chan bool, 1)
    go func() {
        operation()
        done <- true
    }()
    
    // Check for event
    select {
    case event := <-emitter.Events():
        if event.Type != ExpectedType {
            t.Errorf("event type = %v, want %v", event.Type, ExpectedType)
        }
    case <-done:
        t.Error("operation completed without emitting event")
    }
}
```

### Testing Error Paths

Always test both success and failure paths:

```go
func TestWithErrorPaths(t *testing.T) {
    tests := []struct {
        name    string
        setup   func()
        wantErr bool
    }{
        {
            name:    "success case",
            setup:   func() { /* setup for success */ },
            wantErr: false,
        },
        {
            name:    "nil input",
            setup:   func() { /* setup nil */ },
            wantErr: true,
        },
        {
            name:    "invalid state",
            setup:   func() { /* setup invalid state */ },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            err := operation()
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Coverage Goals

- **Overall:** ≥85%
- **Critical paths:** ≥90%
- **New code:** ≥90%

### Running Coverage

```bash
# Run tests with coverage
go test -cover ./internal/core/...

# Generate coverage profile
go test -coverprofile=coverage.out ./internal/core/...

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep "^total"
```

### Race Detection

Always run tests with race detector:

```bash
go test -race ./...
```

## Test Timeouts

Set appropriate timeouts to prevent hanging tests:

```bash
go test -timeout 30s ./...
```

In code:

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Use ctx for operations
}
```

## Examples from Codebase

### Git Integration Tests

See `internal/core/git_integration_coverage_test.go` for examples of:
- Testing file system operations
- Testing Git commands with mock repositories
- Testing concurrent access
- Testing edge cases (renamed files, mixed status)

### History Compression Tests

See `internal/core/history_coverage_test.go` for examples of:
- Testing compression algorithms
- Testing event emission
- Testing message transformation
- Testing with mock tokenizers

### Validator Tests

See `internal/core/validator_test.go` and `internal/core/final_coverage_test.go` for examples of:
- Testing security-critical code
- Testing command classification
- Testing pattern matching
- Testing concurrency safety

## Best Practices

1. **Test names should read like requirements:** `TestValidator_ClassifiesDangerousCommands`
2. **One assertion per test case** (in table-driven tests)
3. **Test edge cases:** empty input, nil pointers, boundary conditions
4. **Test concurrency** where applicable
5. **Use t.Helper()** for test helper functions
6. **Clean up resources** with `defer` or `t.Cleanup()`
7. **Avoid test interdependence** - tests should run independently
8. **Use descriptive test case names** in table-driven tests
9. **Test error messages** for clarity and actionability
10. **Document complex test setups** with comments

## Test Fixtures

For complex test data, use separate files:

```
internal/core/testdata/
  ├── valid_config.yaml
  ├── invalid_config.yaml
  └── sample_messages.json
```

Load with:

```go
data, err := os.ReadFile("testdata/valid_config.yaml")
if err != nil {
    t.Fatalf("failed to load test fixture: %v", err)
}
```

## Continuous Integration

Tests are run automatically on:
- Every commit
- Pull requests
- Before merges

CI checks:
- ✓ All tests pass
- ✓ Race detector clean
- ✓ Coverage ≥85%
- ✓ Linter passes

## References

- [Effective Go - Testing](https://go.dev/doc/effective_go#testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Testing Best Practices](https://go.dev/blog/table-driven-tests)
- [AGENTS.md](../AGENTS.md) - Project testing philosophy

