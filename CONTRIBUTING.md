# Contributing to Spin

Thank you for your interest in contributing to Spin! This guide will help you get started with contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please treat all contributors with respect and professionalism.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Make
- Git
- A Unix-like environment (Linux, macOS, or WSL on Windows)

### Setting Up Development Environment

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/spin.git
   cd spin
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build the Project**
   ```bash
   make build
   ```

4. **Run Tests**
   ```bash
   make test
   ```

## Development Workflow

### 1. Create a Branch

Create a feature branch from `main`:
```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/add-new-provider` - For new features
- `fix/command-validation` - For bug fixes
- `docs/update-readme` - For documentation
- `refactor/improve-performance` - For refactoring

### 2. Make Your Changes

Follow these guidelines:
- Write clean, idiomatic Go code
- Follow the [Google Go Style Guide](https://google.github.io/styleguide/go/)
- Add tests for new functionality
- Update documentation as needed

### 3. Code Style

#### Go Style Guidelines

**Naming Conventions:**
- Use MixedCaps or mixedCaps rather than underscores
- Exported names start with capital letters
- Avoid unnecessary abbreviations
- Use clear, descriptive names

**Error Handling:**
```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("do something: %w", err)
}

// Avoid
if err := doSomething(); err != nil {
    return err  // Missing context
}
```

**Interfaces:**
```go
// Good - Small, focused interfaces
type Reader interface {
    Read([]byte) (int, error)
}

// Avoid - Large interfaces
type FileHandler interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    Seek(int64, int) (int64, error)
    // ... many more methods
}
```

#### Type Safety

Use generics where appropriate:
```go
// Good - Type-safe with generics
func Process[T any](items []T, fn func(T) error) error {
    for _, item := range items {
        if err := fn(item); err != nil {
            return err
        }
    }
    return nil
}

// Avoid - Using interface{}
func Process(items []interface{}, fn func(interface{}) error) error {
    // ...
}
```

### 4. Testing

#### Test Requirements

- Maintain test coverage above 85%
- Write both positive and negative test cases
- Use table-driven tests where appropriate
- Mock external dependencies

#### Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   "test",
            want:    "TEST",
            wantErr: false,
        },
        {
            name:    "empty input",
            input:   "",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests for specific package
go test ./internal/core/...

# Run tests with race detection
go test -race ./...

# Run benchmarks
make bench
```

### 5. Documentation

#### Code Documentation

All exported types, functions, and packages must have godoc comments:

```go
// Package core provides the core business logic for the Spin AI agent.
package core

// Agent orchestrates the interaction between the LLM provider and tools.
// It manages the conversation flow, handles tool calls, and emits events.
type Agent struct {
    // ...
}

// Execute processes a user message and returns the agent's response.
// It may invoke tools and will emit events during processing.
func (a *Agent) Execute(ctx context.Context, input string) (*Response, error) {
    // ...
}
```

#### README Updates

Update relevant documentation when:
- Adding new features
- Changing configuration options
- Modifying public APIs
- Adding new dependencies

### 6. Commit Messages

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Maintenance tasks

**Examples:**
```
feat(llm): add support for Gemini provider

Implements the Google Gemini provider with streaming support.
Includes function calling and multi-modal capabilities.

Closes #123
```

```
fix(security): prevent command injection in executor

Adds proper escaping for shell arguments to prevent
potential command injection vulnerabilities.

Security: CVE-2024-XXXXX
```

### 7. Linting and Formatting

Before submitting:

```bash
# Run linters
make lint

# Format code
go fmt ./...

# Run additional checks
go vet ./...
golangci-lint run
```

### 8. Pull Request Process

1. **Update Your Branch**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push Your Changes**
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create Pull Request**
   - Use a descriptive title
   - Reference any related issues
   - Provide a clear description
   - Include test results
   - Add screenshots if relevant

4. **PR Template**
   ```markdown
   ## Description
   Brief description of changes

   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## Testing
   - [ ] Tests pass locally
   - [ ] Added new tests
   - [ ] Coverage maintained/improved

   ## Checklist
   - [ ] Code follows style guidelines
   - [ ] Self-reviewed code
   - [ ] Updated documentation
   - [ ] No new warnings
   ```

## Project Structure

Understanding the project structure helps you contribute effectively:

```
spin/
├── cmd/               # Command-line applications
│   └── spin/         # Main CLI application
├── internal/         # Private application code
│   ├── core/        # Core business logic
│   ├── llm/         # LLM provider implementations
│   ├── security/    # Security modules
│   ├── tools/       # Tool implementations
│   ├── tui/         # Terminal UI
│   └── types/       # Shared type definitions
├── configs/         # Configuration files
├── docs/           # Documentation
├── examples/       # Example code
├── specs/          # Technical specifications
└── tests/          # Integration tests
```

## Areas for Contribution

### Good First Issues

Look for issues labeled `good first issue` - these are suitable for newcomers:
- Documentation improvements
- Test coverage improvements
- Small bug fixes
- Code cleanup

### Feature Requests

Check issues labeled `enhancement` or `feature request`:
- New LLM providers
- Additional tools
- UI improvements
- Performance optimizations

### Bug Fixes

Issues labeled `bug` need attention:
- Review issue description
- Reproduce the bug
- Write a test that fails
- Fix the bug
- Ensure test passes

### Documentation

Always welcome:
- Fix typos
- Improve clarity
- Add examples
- Update outdated information

## Performance Considerations

When contributing performance-related changes:

1. **Benchmark Before and After**
   ```go
   func BenchmarkFunction(b *testing.B) {
       for i := 0; i < b.N; i++ {
           Function()
       }
   }
   ```

2. **Profile Code**
   ```bash
   go test -bench=. -cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

3. **Consider Memory Allocations**
   - Use pooling for frequently allocated objects
   - Avoid unnecessary allocations in hot paths
   - Prefer streaming over loading everything in memory

## Security

### Reporting Security Issues

**Do not** open public issues for security vulnerabilities. Instead:
1. Email security@example.com with details
2. Include steps to reproduce
3. Allow time for patch before disclosure

### Security Guidelines

When contributing:
- Never log sensitive information
- Validate all inputs
- Use secure defaults
- Follow principle of least privilege
- Add security tests for sensitive code

## Getting Help

### Resources

- [Documentation](docs/)
- [Architecture Guide](ARCHITECTURE.md)
- [Type Safety Guide](docs/TYPE_SAFETY.md)
- [Security Modules](specs/security-modules.md)

### Communication

- Open an issue for bugs or features
- Start a discussion for questions
- Join our community chat (if available)

## Recognition

Contributors are recognized in:
- The CONTRIBUTORS file
- Release notes
- Project documentation

## License

By contributing, you agree that your contributions will be licensed under the project's MIT License.

## Thank You!

Your contributions make Spin better for everyone. We appreciate your time and effort in improving the project!