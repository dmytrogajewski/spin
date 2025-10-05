# E2E TUI Tests

Automated end-to-end tests for Spin's terminal UI using `go-expect`.

## Overview

These tests launch the real `spin` binary in a PTY (pseudo-terminal) and interact with it programmatically by sending keyboard input and verifying output.

## Running Tests

### All E2E Tests
```bash
make test-e2e
```

### Specific Test
```bash
go test ./tests -run TestTUILaunch -v
```

### Skip E2E Tests (for CI/local quick runs)
```bash
go test -short ./...  # E2E tests are skipped in short mode
```

## Available Tests

| Test | Description | Duration |
|------|-------------|----------|
| `TestTUILaunch` | Verifies TUI starts without errors | ~2s |
| `TestTUIBasicChat` | Sends message and receives LLM response | ~15s |
| `TestTUIFilePickerTrigger` | Tests `@` key opens file picker | ~2s |
| `TestTUIHelpModal` | Tests `Ctrl+H` opens help | ~2s |
| `TestTUIExitWithCtrlD` | Tests `Ctrl+D` exits cleanly | ~1s |
| `TestTUIToolApproval` | Tests approval workflow with file creation | ~30s |
| `TestTUIMultiTurn` | Tests conversation context retention | ~25s |
| `TestTUIStopStreaming` | Tests `Ctrl+C` stops streaming | ~5s |

## Requirements

- **Built binary**: Tests require `bin/spin` to exist
  ```bash
  make build
  ```

- **Running LLM**: Tests use Ollama by default
  ```bash
  # Check if Ollama is running
  curl http://127.0.0.1:11434/api/tags

  # Pull required models
  ollama pull qwen3:0.6b
  ollama pull qwen3:1.7b
  ollama pull qwen2.5-coder:1.5b
  ```

- **PTY support**: Tests require a Unix-like system with PTY support (Linux, macOS, WSL)

## How It Works

### Test Flow
1. Create pseudo-terminal (PTY) using `go-expect`
2. Launch `bin/spin` with stdin/stdout/stderr attached to PTY
3. Send keyboard input (text, `Enter`, `Ctrl+C`, etc.)
4. Verify expected behavior (no crashes, clean exits, responses)

### Example Test Anatomy

```go
func TestTUIBasicChat(t *testing.T) {
    // 1. Create console (PTY)
    console, _ := expect.NewConsole()
    defer console.Close()

    // 2. Launch TUI
    cmd := exec.Command(getBinPath(t), "--model", "qwen3:0.6b")
    cmd.Stdin = console.Tty()
    cmd.Stdout = console.Tty()
    cmd.Start()

    // 3. Interact
    console.SendLine("Hello!")

    // 4. Verify
    console.ExpectString("Hello") // Wait for response
}
```

## Writing New Tests

### Best Practices

1. **Use `testing.Short()` skip**:
   ```go
   if testing.Short() {
       t.Skip("Skipping E2E test in short mode")
   }
   ```

2. **Set reasonable timeouts**:
   ```go
   func TestMyTUI(t *testing.T) {
       // Test timeout
       time.Sleep(2 * time.Second) // Wait for UI to render

       // Or use ExpectString with context
       ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
       defer cancel()
   }
   ```

3. **Clean up resources**:
   ```go
   defer func() {
       console.Send("\x04") // Ctrl+D to exit
       cmd.Wait()
   }()
   ```

4. **Avoid flaky tests**:
   - Use `time.Sleep()` for UI rendering (not ExpectString with fragile patterns)
   - Don't rely on exact screen output (ANSI codes, formatting vary)
   - Test behavior, not UI aesthetics

### Common Keyboard Inputs

```go
console.SendLine("text")     // Type text + Enter
console.Send("@")             // Single key
console.Send("\x03")          // Ctrl+C
console.Send("\x04")          // Ctrl+D
console.Send("\x08")          // Ctrl+H
console.Send("\x1b")          // Esc
console.Send("\x1b\x1b")      // Esc Esc (backtrack)
```

## Troubleshooting

### Test Hangs
- Increase timeout: `-timeout 60s`
- Check if LLM provider is running
- Verify binary exists: `ls bin/spin`

### "fork/exec: no such file or directory"
```bash
make build  # Rebuild binary
```

### "connection refused" / LLM errors
```bash
# Start Ollama
systemctl start ollama  # Linux
brew services start ollama  # macOS
```

### Tests fail on CI
- Use `-short` flag to skip E2E tests on CI without PTY support
- Or use GitHub Actions with full terminal support

## Integration with CI

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4

      # Unit tests (fast)
      - run: go test -short ./...

      # E2E tests (slow, requires Ollama)
      - uses: ollama/setup-ollama@v1
      - run: ollama pull qwen3:0.6b
      - run: make build
      - run: make test-e2e
```

## Technical Details

### Dependencies
- [`github.com/Netflix/go-expect`](https://github.com/Netflix/go-expect) - PTY interaction
- [`github.com/creack/pty`](https://github.com/creack/pty) - PTY creation (transitive)

### Limitations
- **Platform**: Requires Unix-like PTY support (no native Windows)
- **Speed**: E2E tests are slow (~1-30s each) due to real LLM calls
- **Flakiness**: Timing-sensitive; may fail on slow systems

### Future Improvements
- [ ] Mock LLM responses for faster tests
- [ ] Screen capture/snapshot testing
- [ ] Parallel test execution
- [ ] Headless mode for CI
- [ ] Coverage reports for TUI code
