# Agent Guidelines for Spin Development

This document provides guidelines for AI coding agents working on the Spin project. Following these guidelines ensures consistent, high-quality code that adheres to Go best practices and project standards.

## Table of Contents

- [Development Philosophy](#development-philosophy)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Implementation Workflow](#implementation-workflow)
- [Quality Checks](#quality-checks)
- [Common Patterns](#common-patterns)
- [Error Handling](#error-handling)
- [Performance Guidelines](#performance-guidelines)

## Development Philosophy

### Core Principles

1. **Test-Driven Development (TDD)**: Always write tests first
2. **SOLID Principles**: Single Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, Dependency Inversion
3. **DRY (Don't Repeat Yourself)**: Eliminate code duplication
4. **KISS (Keep It Simple, Stupid)**: Prefer simple, readable solutions
5. **Clean Architecture**: Core logic independent of frameworks
6. **Effective Go**: Follow idiomatic Go patterns

### Go Version

- **Go 1.24+** required
- Use latest standard library features
- Leverage improved type inference
- Use `log/slog` for structured logging

## Code Standards

### Project Layout

Follow the [Go Standard Project Layout](https://github.com/golang-standards/project-layout):

```
spin/
├── cmd/                 # Application entry points
├── internal/            # Private application code
│   └── core/           # Core business logic (current focus)
├── pkg/                # Public library code
├── configs/            # Configuration examples
├── specs/              # Specifications and FRDs
└── docs/               # Documentation
```

### Naming Conventions

```go
// Package names: lowercase, single word, no underscores
package core

// Exported types: PascalCase
type Config struct {}
type Manager struct {}

// Unexported types: camelCase
type validationError struct {}

// Functions: PascalCase (exported), camelCase (unexported)
func NewManager() *Manager {}
func loadFromEnv() *Config {}

// Constants: PascalCase or SCREAMING_SNAKE_CASE for special cases
const MaxRetries = 3
const DefaultTimeout = 5 * time.Minute

// Error variables: ErrPrefix
var ErrNotFound = errors.New("not found")
var ErrInvalidInput = errors.New("invalid input")
```

### Code Organization

```go
// 1. Package declaration
package core

// 2. Imports (grouped: stdlib, external, internal)
import (
    "context"
    "fmt"
    "time"

    "gopkg.in/yaml.v3"

    "github.com/dmytrogajewski/spin/internal/security"
)

// 3. Constants
const (
    DefaultMaxTurns = 50
)

// 4. Variables (package-level)
var (
    ErrSessionNotFound = errors.New("session not found")
)

// 5. Types (interfaces first, then structs)
type Manager interface {}
type Config struct {}

// 6. Constructors
func NewConfig() *Config {}

// 7. Methods
func (c *Config) Validate() error {}

// 8. Private functions
func loadFromEnv() *Config {}
```

### Documentation

Every exported symbol must have godoc comments:

```go
// Config contains core configuration for the Spin agent.
// It supports loading from YAML files, environment variables,
// and provides validation and merging capabilities.
type Config struct {
    // Provider is the LLM provider type (ollama, openai, etc.)
    Provider string `yaml:"provider"`
    
    // Model is the model name specific to the provider
    Model string `yaml:"model"`
}

// Validate validates the configuration and returns all validation errors.
// It checks required fields, value ranges, and cross-field dependencies.
func (c *Config) Validate() error {
    // Implementation
}
```

## Testing Requirements

### Test-Driven Development

**Always write tests BEFORE implementation:**

```go
// 1. Write the test
func TestConfig_Validate_MissingProvider(t *testing.T) {
    cfg := &Config{
        Model: "test-model",
    }
    
    err := cfg.Validate()
    if err == nil {
        t.Error("Validate() should return error for missing provider")
    }
}

// 2. Run the test (it should fail)
// 3. Implement the code to make it pass
// 4. Refactor if needed
```

### Test Structure

```go
// Test file naming: <file>_test.go
// internal/core/config.go -> internal/core/config_test.go

package core

import "testing"

// Test naming: Test<Type>_<Method>_<Scenario>
func TestConfig_Validate_MissingProvider(t *testing.T) {}
func TestConfig_Merge_BasicFields(t *testing.T) {}

// Table-driven tests for multiple scenarios
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *Config
        wantErr bool
    }{
        {
            name: "valid config",
            cfg:  &Config{Provider: "ollama", Model: "test"},
            wantErr: false,
        },
        {
            name: "missing provider",
            cfg:  &Config{Model: "test"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Coverage Requirements

- **Minimum 90% coverage** for critical paths
- **Minimum 85% coverage** overall
- Run tests with coverage:
  ```bash
  go test -cover ./internal/core/...
  go test -coverprofile=coverage.out ./internal/core/...
  go tool cover -html=coverage.out
  ```

### Test Types

1. **Unit Tests**: Test individual functions/methods
2. **Integration Tests**: Test component interactions
3. **Race Detection**: Always run with `-race` flag
4. **Benchmarks**: For performance-critical code

```bash
# Run all tests
make test

# With coverage
make test-coverage

# With race detector
make test-race

# Benchmarks
go test -bench=. ./internal/core/...
```

## Implementation Workflow

### Feature Implementation Process

1. **Read Specifications**
   - Read the FRD (Feature Requirements Document)
   - Understand DoR (Definition of Ready)
   - Review DoD (Definition of Done)

2. **Write Tests First (TDD)**
   - Create test file with all test cases
   - Include edge cases and error scenarios
   - Run tests to see them fail

3. **Implement Code**
   - Write minimal code to make tests pass
   - Follow SOLID principles
   - Keep functions small and focused

4. **Run Tests**
   - `go test ./internal/core/...`
   - Iterate until all tests pass
   - Check coverage

5. **Code Analysis**
   - Run linter: `make lint`
   - Analyze with uast/herr: `uast parse <file> | herr analyze`
   - Fix any issues

6. **Refactor**
   - Reduce complexity if needed
   - Eliminate duplication
   - Improve readability

7. **Documentation**
   - Add godoc comments
   - Update README if needed
   - Create example code

8. **Update Tracking**
   - Mark feature complete in ROADMAP
   - Update SUMMARY
   - Update FRD status

## Quality Checks

### Linter Configuration

The project uses `golangci-lint` with these key checks:

- `gofmt` - Code formatting
- `govet` - Go vet checks
- `errcheck` - Unchecked errors
- `staticcheck` - Static analysis
- `gosec` - Security issues
- `gocyclo` - Cyclomatic complexity (max 15)

Run linter:
```bash
make lint
```

### Code Complexity

**Target Complexity Levels:**
- Simple: cyclomatic complexity ≤ 5
- Moderate: cyclomatic complexity ≤ 10
- Complex: cyclomatic complexity ≤ 15 (maximum allowed)

**If complexity exceeds 15:**
1. Extract helper functions
2. Use early returns
3. Simplify conditional logic
4. Consider refactoring

Example refactoring:

```go
// Before: High complexity (20+)
func Process(data []Item) error {
    for _, item := range data {
        if item.Type == "A" {
            if item.Valid {
                if item.Status == "active" {
                    // lots of nested logic
                }
            }
        } else if item.Type == "B" {
            // more nested logic
        }
    }
}

// After: Reduced complexity
func Process(data []Item) error {
    for _, item := range data {
        if err := processItem(item); err != nil {
            return err
        }
    }
    return nil
}

func processItem(item Item) error {
    switch item.Type {
    case "A":
        return processTypeA(item)
    case "B":
        return processTypeB(item)
    default:
        return nil
    }
}

func processTypeA(item Item) error {
    if !item.Valid {
        return nil
    }
    if item.Status != "active" {
        return nil
    }
    // Simple logic here
    return nil
}
```

## Common Patterns

### Constructor Pattern

```go
// Basic constructor
func NewManager() *Manager {
    return &Manager{
        config: DefaultConfig(),
    }
}

// Constructor with options (functional options pattern)
type Option func(*Manager) error

func WithConfig(cfg *Config) Option {
    return func(m *Manager) error {
        if err := cfg.Validate(); err != nil {
            return err
        }
        m.config = cfg
        return nil
    }
}

func NewManager(opts ...Option) (*Manager, error) {
    m := &Manager{
        config: DefaultConfig(),
    }
    
    for _, opt := range opts {
        if err := opt(m); err != nil {
            return nil, err
        }
    }
    
    return m, nil
}
```

### Interface Design

```go
// Accept interfaces, return structs
func NewAgent(llm Provider) *Agent {
    return &Agent{llm: llm}
}

// Small, focused interfaces
type Provider interface {
    Complete(ctx context.Context, req Request) (*Response, error)
}

type Validator interface {
    Validate(cmd Command) error
}
```

### Context Propagation

```go
// Always accept context as first parameter
func (m *Manager) LoadSession(ctx context.Context, id string) (*Session, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context for timeouts
    ctx, cancel := context.WithTimeout(ctx, m.timeout)
    defer cancel()
    
    // Pass context to downstream calls
    return m.storage.Load(ctx, id)
}
```

## Error Handling

### Error Creation

```go
// Use the Error struct for rich context
return &Error{
    Op:   "Manager.LoadSession",
    Err:  ErrSessionNotFound,
    Code: ErrCodeNotFound,
    Context: map[string]interface{}{
        "session_id": id,
    },
}
```

### Error Wrapping

```go
// Wrap errors with context
if err := validateInput(input); err != nil {
    return fmt.Errorf("input validation failed: %w", err)
}

// Use Error struct for wrapping
if err := m.storage.Save(session); err != nil {
    return &Error{
        Op:  "Session.Save",
        Err: err,
        Code: ErrCodeInternal,
    }
}
```

### Error Checking

```go
// Use errors.Is for sentinel errors
if errors.Is(err, ErrSessionNotFound) {
    // Handle not found
}

// Use errors.As for error types
var e *Error
if errors.As(err, &e) {
    log.Printf("Operation: %s, Code: %d", e.Op, e.Code)
}
```

### Multiple Errors

```go
// Collect and join multiple errors
func (c *Config) Validate() error {
    var errs []error
    
    if c.Provider == "" {
        errs = append(errs, fmt.Errorf("provider is required"))
    }
    if c.Model == "" {
        errs = append(errs, fmt.Errorf("model is required"))
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    
    return nil
}
```

## Performance Guidelines

### Avoid Unnecessary Allocations

```go
// Bad: Creates new slice on each append
var items []Item
for _, data := range largeDataset {
    items = append(items, process(data))
}

// Good: Pre-allocate with capacity
items := make([]Item, 0, len(largeDataset))
for _, data := range largeDataset {
    items = append(items, process(data))
}
```

### Use Appropriate Data Structures

```go
// For lookups: use maps
cache := make(map[string]*Result)

// For ordered data: use slices
history := make([]Message, 0, 100)

// For unique items: use map with empty struct
seen := make(map[string]struct{})
```

### Concurrency Patterns

```go
// Use errgroup for concurrent operations
import "golang.org/x/sync/errgroup"

func (m *Manager) LoadMultiple(ctx context.Context, ids []string) ([]*Session, error) {
    g, ctx := errgroup.WithContext(ctx)
    sessions := make([]*Session, len(ids))
    
    for i, id := range ids {
        i, id := i, id // Capture loop variables
        g.Go(func() error {
            session, err := m.Load(ctx, id)
            if err != nil {
                return err
            }
            sessions[i] = session
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    
    return sessions, nil
}
```

### Channel Buffering

```go
// Unbuffered: synchronous communication
ch := make(chan Event)

// Buffered: asynchronous communication
events := make(chan Event, 100)

// Rule of thumb:
// - Use unbuffered for synchronization
// - Use buffered to prevent blocking
// - Buffer size = expected burst size
```

## File Organization

### Package Structure

Each package should be cohesive and focused:

```
internal/core/
├── config.go          # Configuration
├── config_test.go     # Configuration tests
├── error.go           # Error types
├── error_test.go      # Error tests
├── manager.go         # Manager (main API)
├── manager_test.go    # Manager tests
└── session/           # Sub-package for sessions
    ├── session.go
    └── session_test.go
```

### Test Data Organization

```
internal/core/
├── testdata/          # Test fixtures
│   ├── valid_config.yaml
│   ├── invalid_config.yaml
│   └── fixtures/
└── testing/           # Test utilities
    ├── mock_llm.go
    └── helpers.go
```

## Git Commit Guidelines

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `test`: Tests
- `refactor`: Code refactoring
- `style`: Formatting
- `chore`: Maintenance

**Example:**
```
feat: implement configuration system

- Add Config struct with YAML support
- Implement Load() and Validate() methods
- Add environment variable support
- Complete test coverage (78.6%)

Closes #123
```

## Troubleshooting

### Common Issues

**Issue: Tests fail after refactoring**
- Run `go mod tidy` to update dependencies
- Check if interfaces changed
- Verify mock implementations match

**Issue: Linter fails**
- Run `make fmt` to format code
- Check cyclomatic complexity with uast/herr
- Refactor complex functions

**Issue: Race detector fails**
- Add proper mutex protection
- Use channels for communication
- Review goroutine usage

**Issue: Coverage too low**
- Add tests for error paths
- Test edge cases
- Cover all branches

## Resources

### Official Go Documentation
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

### Project Documentation
- [Architecture Overview](specs/architecture-overview.md)
- [Core Module Spec](specs/core-module/spec.md)
- [ROADMAP](specs/core-module/ROADMAP.md)

### Tools
- `golangci-lint`: Code linting
- `uast/herr`: Code analysis
- `go test`: Testing
- `go tool cover`: Coverage analysis

## Quick Reference Checklist

Before marking a feature complete:

- [ ] FRD created and read
- [ ] Tests written first (TDD)
- [ ] All tests passing
- [ ] Coverage ≥90% for critical paths, ≥85% overall
- [ ] Race detector clean
- [ ] Linter passing
- [ ] Code analyzed with uast/herr
- [ ] Complexity ≤15 for all functions
- [ ] Godoc comments on all exports
- [ ] Example code provided
- [ ] ROADMAP updated
- [ ] SUMMARY updated
- [ ] FRD marked complete

---

**Remember:** Quality over speed. Write clean, tested, maintainable code.

