# TUI Demo - Minimal Example

This is the simplest possible example of using the Spin TUI.

## Purpose

Demonstrates:
- Initializing the PureTTY adapter
- Printing lines to output
- Accepting user input
- Clean shutdown with Ctrl-C or Ctrl-D

## Running

```bash
cd examples/tui-demo
go run main.go
```

## Usage

1. The demo starts with a welcome message
2. Type a command and press Enter:
   - `help` - Show available commands
   - `hello` - Print a greeting
   - `quit` - Exit the demo
3. Press Ctrl-D or Ctrl-C to exit at any time

## Key Concepts

### 1. PureTTY Adapter

```go
ui := adapters.NewPureTTY()
```

The PureTTY adapter is the main TUI interface. It handles:
- Terminal raw mode management
- Keyboard input processing
- Output rendering
- Prompt redrawing

### 2. Context Management

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

Using context allows clean shutdown when:
- User presses Ctrl-C (SIGINT)
- Program calls `cancel()`

### 3. Running the TUI

```go
go ui.Run(ctx)
```

`Run()` blocks until context is canceled, so we run it in a goroutine.

### 4. Printing Output

```go
ui.PrintLine("Hello, world!")
```

`PrintLine()` prints a line and automatically redraws the prompt at the bottom.

### 5. Reading Input

```go
for line := range ui.RequestInput() {
    // Handle user input
}
```

`RequestInput()` returns a channel of submitted lines. The loop exits when:
- Channel closes (user pressed Ctrl-D)
- Context is canceled

### 6. Clean Shutdown

```go
defer ui.Stop()
```

`Stop()` restores the terminal to its original state:
- Exits raw mode
- Shows cursor
- Flushes output

## Learn More

- Full TUI documentation: [docs/tui.md](../../docs/tui.md)
- Block system example: [examples/tui-blocks/](../tui-blocks/)
- Streaming example: [examples/tui-streaming/](../tui-streaming/)
