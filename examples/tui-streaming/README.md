# TUI Streaming Demo

This example demonstrates streaming chunks to the TUI, simulating LLM token-by-token responses.

## Purpose

Shows how to:
- Stream text word-by-word
- Stream text character-by-character
- Simulate realistic LLM response timing
- Demonstrate coalescing behavior with fast streams
- Use transient status indicators

## Running

```bash
cd examples/tui-streaming
go run main.go
```

## What You'll See

### Demo 1: Word-by-word streaming
Text appears one word at a time with 100ms delay between words.

### Demo 2: Character-by-character streaming
Text appears one character at a time with 20ms delay.

### Demo 3: Simulated LLM response
A haiku generates with realistic token timing:
- Longer pauses at line breaks (200ms)
- Short pauses for spaces (30ms)
- Normal token delays (60ms)
- Status indicator shows "generating..." during output

### Demo 4: Fast streaming (1000 chunks)
Demonstrates coalescing by sending 1000 chunks as fast as possible. The TUI buffers and flushes them efficiently, showing throughput in chunks/sec.

## Key Concepts

### 1. Creating a Chunk Channel

```go
chunks := make(chan string, 100)
```

Buffered channel for streaming data. Producer sends to this channel, consumer (TUI) reads from it.

### 2. Streaming to TUI

```go
ui.PrintChunks(ctx, chunks)
```

`PrintChunks()` reads from the channel and prints chunks as they arrive. It:
- Buffers small chunks (< 10KB)
- Flushes on newline (for prompt responsiveness)
- Coalesces with ~50ms timer (reduces flicker)
- Respects context cancellation

### 3. Producer Pattern

```go
go func() {
    defer close(chunks)  // Always close when done
    for _, token := range tokens {
        chunks <- token
        time.Sleep(delay)
    }
}()
```

**Important:** Always close the channel when done, or `PrintChunks()` will block forever.

### 4. Context Cancellation

```go
select {
case <-ctx.Done():
    return
case chunks <- token:
    // ...
}
```

Check context in producer loop to allow clean shutdown (Ctrl-C).

### 5. Coalescing Behavior

Small chunks arriving rapidly are buffered and flushed together. This:
- Reduces write syscall overhead
- Reduces terminal redraw flicker
- Improves throughput

**Exception:** Any chunk containing `\n` triggers immediate flush, keeping the prompt responsive.

### 6. Transient Status

```go
ui.SetStatus("generating...")
// ... streaming ...
ui.SetStatus("")  // Clear when done
```

Shows right-aligned status during long operations.

## Performance

On modern hardware, the TUI can handle:
- **8.7M chunks/sec** (small tokens)
- **828 MB/sec** (large chunks ≥10KB)

See [docs/performance.md](../../docs/performance.md) for benchmarks.

## Learn More

- Full TUI documentation: [docs/tui.md](../../docs/tui.md)
- Output package docs: [docs/packages/ui-output.md](../../docs/packages/ui-output.md)
- Minimal example: [examples/tui-demo/](../tui-demo/)
- Block system example: [examples/tui-blocks/](../tui-blocks/)
