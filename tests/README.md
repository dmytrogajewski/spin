# Spin Test Suite

Comprehensive test coverage for the Spin AI coding assistant.

## Directory Structure

```
tests/
├── e2e/                    # End-to-end tests (hermetic, fast)
│   ├── e2e_test.go        # Core functionality (config, exec, MCP)
│   ├── tui_e2e_test.go    # TUI mode tests
│   ├── statusbar_interactive_test.go  # Status bar user flows
│   ├── statusbar_regression_test.go   # Bug regression tests
│   ├── statusbar_diagnostic_test.go   # Rendering diagnostics
│   └── README.md          # E2E test documentation
│
├── emulator/              # Real terminal emulator tests (slow, requires Ollama)
│   ├── statusbar_pty_test.go  # PTY-based status bar tests
│   ├── test_config.yaml       # Test configuration
│   └── README.md              # Emulator test documentation
│
└── README.md              # This file
```

## Test Types

### 1. Unit Tests

Located in package directories (e.g., `internal/ui/sticky/*_test.go`):
- Test individual components in isolation
- Fast, deterministic, hermetic
- **Run**: `go test ./internal/...`

### 2. Integration Tests

Located in package directories with `_integration_test.go` suffix:
- Test component interactions
- May use test fixtures or mocks
- **Run**: `go test ./internal/... -run Integration`

### 3. End-to-End Tests (`tests/e2e/`)

Complete user workflows with simulated terminals:
- Uses **fake TTY** for speed and determinism
- Tests real user interactions
- Hermetic (no external services)
- **Run**: `go test ./tests/e2e/... -v`

### 4. Emulator Tests (`tests/emulator/`)

Real pseudo-terminal tests using go-expect:
- Uses **real PTY** for terminal validation
- Catches ANSI rendering bugs
- **Requires Ollama running**
- **Run**: `go test ./tests/emulator/... -v`

## Quick Start

```bash
# Run all tests
make test

# Run with race detection
make test-race

# Run only fast tests (skip emulator)
go test ./... -short

# Run specific test suite
go test ./tests/e2e/... -v
go test ./tests/emulator/... -v

# Run specific test
go test ./tests/e2e/... -v -run TestInteractiveFlow_UserTypesAndSeesPrompt

# Coverage report
make coverage
```

## Test Categories by Feature

### Status Bar Feature

| Test File | Type | What It Tests |
|-----------|------|---------------|
| `internal/ui/sticky/statusbar_test.go` | Unit | StatusBar data model |
| `internal/ui/sticky/coordinator_test.go` | Unit | Sticky bottom coordinator |
| `internal/ui/sticky/renderer_test.go` | Unit | Adaptive rendering |
| `internal/ui/sticky/statusbar_integration_test.go` | Integration | Status bar + aggregator |
| `tests/e2e/statusbar_interactive_test.go` | E2E | User typing & interaction |
| `tests/e2e/statusbar_regression_test.go` | E2E | Bug regressions |
| `tests/e2e/statusbar_diagnostic_test.go` | E2E | Rendering diagnostics |
| `tests/emulator/statusbar_pty_test.go` | Emulator | Real PTY validation |

### Core Functionality

| Test File | Type | What It Tests |
|-----------|------|---------------|
| `tests/e2e/e2e_test.go` | E2E | Config, exec mode, MCP |
| `tests/e2e/tui_e2e_test.go` | E2E | TUI modes, navigation |
| `internal/core/*_test.go` | Unit | Core logic |
| `internal/llm/*_test.go` | Unit | LLM providers |

## Coverage Targets

- **Unit tests**: ≥90% coverage
- **Integration tests**: ≥85% coverage
- **Critical paths**: ≥90% coverage
- **New features**: ≥90% coverage

Check coverage:
```bash
make coverage
# Opens HTML report in browser
```

## Test Infrastructure

### Test Utilities (`internal/ui/testkit/`)

- `fake_tty.go` - Simulated terminal
- `safe_buffer.go` - Thread-safe output capture
- `interactive_tui_test.go` - TUI interaction helpers
- `fake_keyboard.go` - Keyboard event simulation

### Mock Providers (`internal/llm/mock/`)

- `provider.go` - Mock LLM for hermetic tests
- No external API calls
- Configurable responses
- Call history tracking

## Prerequisites

### For E2E Tests

None! E2E tests are fully hermetic.

### For Emulator Tests

1. **Ollama running**: `ollama serve`
2. **Model available**: `ollama pull qwen3:1.7b`
3. **Network access**: http://localhost:11434

## Continuous Integration

```bash
# CI pipeline should run:
make lint          # Linter checks
make test          # All unit + integration + e2e tests
make test-race     # Race detector

# Emulator tests should run separately (require Ollama):
go test ./tests/emulator/... -v
```

## Writing New Tests

### 1. Unit Test (Component Package)

```go
// internal/ui/sticky/newfeature_test.go
package sticky

func TestNewFeature(t *testing.T) {
    // Test isolated component
}
```

### 2. Integration Test (Component Package)

```go
// internal/ui/sticky/integration_test.go
package sticky

func TestFeatureIntegration(t *testing.T) {
    // Test multiple components together
}
```

### 3. E2E Test (tests/e2e/)

```go
// tests/e2e/feature_test.go
package e2e

func TestFeature_UserFlow(t *testing.T) {
    // Simulate complete user workflow
}
```

### 4. Emulator Test (tests/emulator/)

```go
// tests/emulator/feature_pty_test.go
package emulator

func TestPTY_Feature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping PTY test in short mode")
    }
    // Test with real PTY
}
```

## Debugging Test Failures

### Race Conditions

```bash
go test ./tests/e2e/... -race -v
```

Look for concurrent access to shared state.

### Flaky Tests

```bash
# Run test 100 times
go test ./tests/e2e/... -run TestFlaky -count 100
```

### PTY Issues

```bash
# See raw ANSI output
go test ./tests/emulator/... -v -run TestPTY 2>&1 | cat -v
```

## Best Practices

1. **Test pyramid**: More unit tests, fewer e2e tests, minimal emulator tests
2. **Hermetic first**: Use fake TTY for most tests, real PTY only when necessary
3. **Fast feedback**: Keep tests fast (<100ms for unit, <1s for e2e)
4. **Clear names**: Test names should describe what they verify
5. **Deterministic**: No sleeps, use synchronization primitives
6. **Isolated**: Each test should be independent

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "No such file or directory: bin/spin" | Run `make build` first |
| "Ollama connection refused" | Start Ollama: `ollama serve` |
| "Model not found" | Pull model: `ollama pull qwen3:1.7b` |
| Race detector failures | Fix concurrent access, use `SafeBuffer` |
| Test timeouts | Increase timeout or check for deadlocks |

---

**For more details, see:**
- `tests/e2e/README.md` - E2E test documentation
- `tests/emulator/README.md` - Emulator test documentation
- `docs/testing.md` - Testing philosophy and guidelines
