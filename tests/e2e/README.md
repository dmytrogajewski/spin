# End-to-End Tests

End-to-end tests that verify complete user workflows using simulated terminals.

## Test Categories

### Core E2E Tests (`e2e_test.go`)

Tests for basic Spin functionality:
- **Config commands**: `config show`, `config list-keys`
- **Exec mode**: Basic prompts, tool usage, config integration
- **MCP integration**: Server lifecycle, resource access

These tests use **real binaries** and **real config files** to validate production behavior.

### TUI E2E Tests (`tui_e2e_test.go`)

Interactive terminal UI tests:
- **Launch and initialization**: Logo display, prompt rendering
- **Mode switching**: Input mode, Timeline mode, Command Palette
- **Navigation**: Keyboard shortcuts, scrolling, filtering

Uses **fake TTY** for hermetic, fast testing.

### Status Bar Tests

#### Interactive Flow (`statusbar_interactive_test.go`)

User interaction scenarios:
- **Typing and echoing**: Characters appear as you type
- **Submit and response**: Enter key submits, prompt resets
- **Backspace editing**: Character deletion works correctly
- **Status bar updates**: Metrics update in real-time
- **Scrolling output**: Output scrolls above sticky area

**Purpose**: Verify the sticky bottom area works correctly with user input.

#### Regression Tests (`statusbar_regression_test.go`)

Tests for previously fixed bugs:
- **Bug: Typing shows nothing** - Cursor restoration issue
- **Bug: Status bar not visible** - ANSI positioning
- **Bug: Prompt rendering** - Debug output validation

**Purpose**: Prevent regressions of critical fixes.

#### Diagnostic Tests (`statusbar_diagnostic_test.go`)

Low-level rendering validation:
- **Render count**: How many times sticky area redraws
- **Prompt position**: Cursor at correct line
- **Status bar multiplicity**: No duplicate status bars
- **Raw prompt rendering**: Direct renderer output

**Purpose**: Debug rendering issues and performance.

## Test Infrastructure

### Fake TTY (`internal/ui/testkit/fake_tty.go`)

Simulated terminal for hermetic tests:
- Configurable dimensions (width × height)
- Raw mode simulation
- Resize event handling
- Thread-safe

### Safe Buffer (`internal/ui/testkit/safe_buffer.go`)

Thread-safe output capture for concurrent UI rendering.

### Interactive TUI Test (`internal/ui/testkit/interactive_tui_test.go`)

Helper for simulating user interactions:
```go
test := testkit.NewInteractiveTUITest(t, ui, keyboard)
test.TypeString("hello")
test.PressEnter()
output := test.GetOutput()
```

## Running Tests

```bash
# All e2e tests
go test ./tests/e2e/... -v

# Specific test suite
go test ./tests/e2e/... -v -run TestInteractiveFlow

# With race detection
go test ./tests/e2e/... -v -race

# Short mode (skip slow tests)
go test ./tests/e2e/... -v -short
```

## Test Data

- `test_config.yaml` - Minimal config for testing (uses Ollama with qwen3:1.7b)

## Coverage Targets

- Critical paths: ≥90%
- Overall: ≥85%
- New features: ≥90%

## Adding New Tests

1. **User workflow test** → `statusbar_interactive_test.go`
2. **Bug regression** → `statusbar_regression_test.go`
3. **Rendering debug** → `statusbar_diagnostic_test.go`
4. **Core functionality** → `e2e_test.go`

## Known Issues

- Tests require **fake TTY** - real TTY breaks hermetic testing
- Some tests use **SafeBuffer** to prevent race conditions
- **PTY tests** are in `tests/emulator/` (require real pseudo-terminal)

---

**See also:**
- `tests/emulator/README.md` - Real terminal emulator tests
- `internal/ui/testkit/` - Test utilities
