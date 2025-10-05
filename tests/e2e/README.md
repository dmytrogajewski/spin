# End-to-End Tests

This directory contains end-to-end (e2e) tests for the Spin binary.

## Overview

E2E tests execute the actual `spin` binary and verify its behavior from a user's perspective. These tests catch integration issues that unit tests might miss.

## Running Tests

### Prerequisites

1. **Go 1.21+** installed
2. **Ollama** running on `http://127.0.0.1:11434` (for exec mode tests)
   - Install from: https://ollama.ai
   - Ensure model `qwen3:0.6b` is available: `ollama pull qwen3:0.6b`

### Quick Start

```bash
# From project root
make test-e2e

# Or manually
cd tests/e2e
go test -v -timeout 5m
```

### Test Options

```bash
# Run specific test
go test -v -run TestConfigCommands

# Run without Ollama (skips exec tests)
go test -v -run "^Test(Config|MCP|Debug|Version)"

# Verbose output
go test -v -timeout 5m

# See all test names
go test -list .
```

## Test Coverage

### Regression Tests

These tests specifically catch bugs that were fixed:

#### Bug #1: Config Loader Reading Binary Files
- **Test:** `TestConfigCommands/config_show_with_binary_file_in_cwd`
- **What it tests:** Config loader should ignore binary files named "spin"
- **How it fails if bug returns:** Error message contains "control characters are not allowed"

#### Bug #2: Exec Mode Config Integration
- **Test:** `TestExecMode/exec_basic_prompt`
- **What it tests:** Exec mode should use config file for provider/model
- **How it fails if bug returns:** Error message contains "provider is required" or "model is required"

#### Bug #3: Event Data Type Mismatch
- **Test:** `TestExecMode/exec_basic_prompt`
- **What it tests:** Exec mode should print LLM output to stdout
- **How it fails if bug returns:** stdout is empty despite successful execution

### Feature Tests

| Test Suite | What It Tests | Requirements |
|------------|---------------|--------------|
| `TestConfigCommands` | Config show, validate, path | None |
| `TestMCPCommands` | MCP add, list, get, remove | None |
| `TestDebugCommands` | Platform checks for debug commands | None |
| `TestExecMode` | Non-interactive execution | Ollama running |
| `TestVersionAndHelp` | Version and help output | None |
| `TestJSONOutput` | JSON output format validity | None |

## Test Architecture

### Test Structure

```go
func TestFeature(t *testing.T) {
    t.Run("specific case", func(t *testing.T) {
        // Arrange
        stdout, stderr, err := runSpin(t, "command", "args...")

        // Assert
        if err != nil {
            t.Fatalf("command failed: %v", err)
        }

        if !strings.Contains(stdout, "expected output") {
            t.Errorf("unexpected output: %s", stdout)
        }
    })
}
```

### Helper Functions

- `runSpin(t, args...)` - Execute spin binary with args
- `runSpinWithInput(t, input, args...)` - Execute with stdin
- `isOllamaAvailable(t)` - Check if Ollama is running

### TestMain

The `TestMain` function builds the binary before running tests:

```go
func TestMain(m *testing.M) {
    // Build binary
    exec.Command("go", "build", "-o", binPath, "../../cmd/spin").Run()

    // Run tests
    code := m.Run()

    os.Exit(code)
}
```

## Writing New E2E Tests

### Template

```go
func TestNewFeature(t *testing.T) {
    t.Run("test case name", func(t *testing.T) {
        // Execute command
        stdout, stderr, err := runSpin(t, "command", "args")

        // Verify behavior
        if err != nil {
            t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
        }

        // Check output
        if !strings.Contains(stdout, "expected") {
            t.Errorf("unexpected output: %s", stdout)
        }
    })
}
```

### Best Practices

1. **Test from user perspective** - Execute the actual binary
2. **Use descriptive names** - `TestConfigCommands/config_show_without_config_file`
3. **Check both success and error paths**
4. **Verify output format** - stdout vs stderr
5. **Use timeouts** - `context.WithTimeout` for long-running commands
6. **Clean up resources** - Remove temp files, restore configs
7. **Skip gracefully** - If Ollama unavailable, skip (don't fail)

### Example: Testing a New Command

```go
func TestNewCommand(t *testing.T) {
    t.Run("basic usage", func(t *testing.T) {
        stdout, stderr, err := runSpin(t, "newcmd", "--flag", "value")

        if err != nil {
            t.Fatalf("newcmd failed: %v\nstderr: %s", err, stderr)
        }

        if !strings.Contains(stdout, "Success") {
            t.Errorf("Expected success message, got: %s", stdout)
        }
    })

    t.Run("error handling", func(t *testing.T) {
        _, stderr, err := runSpin(t, "newcmd", "--invalid-flag")

        if err == nil {
            t.Error("Expected error with invalid flag")
        }

        if !strings.Contains(stderr, "unknown flag") {
            t.Errorf("Expected error message, got: %s", stderr)
        }
    })
}
```

## Debugging Failed Tests

### View Full Output

```bash
go test -v -timeout 5m 2>&1 | tee test-output.log
```

### Run Single Test

```bash
go test -v -run "TestConfigCommands/config_show_with_binary"
```

### Check Binary

```bash
# Verify binary exists
ls -lh ../../bin/spin

# Test manually
../../bin/spin config show
```

### Debug with Delve

```bash
dlv test -- -test.run TestConfigCommands
```

## CI Integration

### GitHub Actions

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install Ollama
        run: curl https://ollama.ai/install.sh | sh

      - name: Pull model
        run: ollama pull qwen3:0.6b

      - name: Run e2e tests
        run: make test-e2e
```

## Troubleshooting

### "Ollama not available"

```bash
# Check Ollama is running
curl http://127.0.0.1:11434/api/tags

# Start Ollama
ollama serve &

# Pull test model
ollama pull qwen3:0.6b
```

### "Binary not found"

```bash
# Build manually
cd ../..
make build

# Check binary
ls -lh bin/spin
```

### "Context deadline exceeded"

- Model loading might be slow on first run
- Increase timeout in test
- Use a smaller/faster model

## Performance

Typical test execution times (with warm Ollama):

- Config tests: ~0.1s
- MCP tests: ~0.2s
- Debug tests: ~0.1s
- **Exec tests: ~6s** (includes LLM inference)
- Version/Help tests: ~0.1s
- JSON tests: ~0.1s

**Total: ~7 seconds**

## Future Improvements

- [ ] Add TUI interaction tests (using expect/pexpect)
- [ ] Test tool execution and approval flow
- [ ] Test multi-turn conversations
- [ ] Add performance benchmarks
- [ ] Test concurrent execution
- [ ] Add snapshot testing for output formatting
- [ ] Test config file validation edge cases
- [ ] Test signal handling (SIGINT, SIGTERM)

## Related Documentation

- [MANUAL_TEST_PLAN.md](../../MANUAL_TEST_PLAN.md) - Manual testing guide
- [AGENTS.md](../../AGENTS.md) - Development workflow
- [Architecture Overview](../../specs/architecture-overview.md)

---

**Last Updated:** 2025-10-06
**Maintainer:** Spin Development Team
