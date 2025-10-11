# TUI Blocks Demo

This example demonstrates all 9 block types in the Spin TUI with realistic examples.

## Purpose

Shows:
- All 9 block types (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR)
- Block metadata and rendering
- Timeline navigation
- Block actions (fold/expand, copy, save, rerun)
- Filtering by type, file, exit code

## Running

```bash
cd examples/tui-blocks
go run main.go
```

## What You'll See

The demo creates a timeline with 9 blocks, one of each type:

### 1. EXECUTE Block (Success)
```
│ ▐EXECUTE▌  Run tests (cmd: "go test -race ./...")    [impact: medium]
│   === RUN   TestCalculate
│   --- PASS: TestCalculate (0.00s)
│   === RUN   TestValidate
│   --- PASS: TestValidate (0.05s)
│   PASS
│   ok      github.com/user/project    0.051s
│ ✓ [exit: 0] [out: 6 lines] [dur: 0.1s]
```

### 2. PLAN Block
```
│ ▐PLAN▌  Implementation plan: 6 total (3 pending, 1 in progress, 2 completed)
│   ✓ Install dependencies (completed)
│   ✓ Create main.go skeleton (completed)
│   ◦ Write unit tests (in progress)
│   • Add integration tests (pending)
│   • Write documentation (pending)
│   • Deploy to staging (pending)
```

### 3. READ Block
```
│ ▐READ▌  (file: internal/tui/input.go)
│   │ 1  package tui
│   │ 2
│   │ 3  import (
│   │ 4      "context"
│   │ 5      "io"
│   │ 6  )
│   ...
```

### 4. GREP Block
```
│ ▐GREP▌  ("TODO", content mode, context: 2)
│   main.go:42:
│   40:  func process() {
│   41:      // TODO: Add error handling
│   42:      fmt.Println("processing")
│   43:  }
│   ...
```

### 5. APPLY_PATCH Block (Success)
```
│ ▐APPLY_PATCH▌  (file: main.go)
│   @@ -15,6 +15,9 @@ func main() {
│    func process() {
│        fmt.Println("start")
│   +    if err := validate(); err != nil {
│   +        log.Fatal(err)
│   +    }
│        doWork()
│   ✓ Succeeded. File edited. (+3 added)
```

### 6. SUMMARY Block
```
│ ▐SUMMARY▌  Changes summary
│   Added error handling to the process() function:
│
│   • Added validation check before doWork()
│   • Log fatal error if validation fails
│   ...
```

### 7. TESTING Block
```
│ ▐TESTING▌  Test plan (3 suites, 2 passed, 1 failed)
│   ✓ go test -race ./internal/... (passed, 0.5s)
│   ✓ go test -bench=. ./... (passed, 2.1s)
│   ✗ integration tests (failed, 5.2s)
│       Error: database connection timeout
```

### 8. NOTICE Block
```
│ ▐NOTICE▌  System notice
│   Conversation history has been compressed to reduce context size.
│
│   Previous messages have been summarized. Full history available in:
│   ~/.spin/sessions/20251011-103042.json
```

### 9. ERROR Block
```
│ ▐ERROR▌  Command failed
│   ● Error: exit status 1
│
│   Stack trace:
│     at processFile (internal/core/exec.go:142)
│     at runCommand (internal/core/agent.go:89)
│   ...
│ [exit: 1]
```

## Key Concepts

### 1. Creating Blocks

```go
block := blocks.NewBlock(blocks.BlockTypeExecute)
block.Title = "Run tests"
block.Body = "command output..."
```

Each block has:
- **Type**: Determines rendering style and metadata schema
- **Title**: Optional concise description
- **Body**: Renderable content (logs, code, diffs, lists)
- **Metadata**: Type-specific structured data

### 2. Setting Metadata

```go
meta := &blocks.ExecuteMeta{
    Command:    "go test ./...",
    ExitCode:   ptr.Int(0),
    DurationMS: ptr.Int64(51),
}
blocks.SetExecuteMeta(block, meta)
```

Metadata is type-specific:
- `ExecuteMeta`: Command, exit code, duration, impact
- `PlanMeta`: Total, pending, in-progress, completed counts
- `ReadMeta`: File path, offset, limit
- `GrepMeta`: Pattern, mode, context lines
- `PatchMeta`: File, success, lines added/removed
- `TestingMeta`: Total suites, passed, failed

### 3. Appending to Timeline

```go
ui.AppendBlock(block)
```

Adds block to the timeline and renders it immediately.

### 4. Block Rendering

Blocks are rendered with:
- **Header**: Tag pill (colored by type) + title + metadata chips
- **Body**: Type-specific rendering (diff, code, list, transcript)
- **Footer**: Outcome chips (exit code, duration, lines, etc.)

### 5. Navigation Keys

Try these while the demo is running:

**Scrolling:**
- `PgUp` / `PgDn` - Scroll by page
- `g` - Jump to top
- `G` - Jump to bottom
- `[` / `]` - Previous/next block

**Block actions:**
- `Enter` - Toggle fold/expand
- `y` - Copy block body
- `S` - Save block to file
- `r` - Rerun (EXECUTE blocks only)

**Advanced:**
- `Ctrl-P` - Command palette
- `/` - Filter timeline (e.g., `type:EXECUTE`, `exit:1`)
- `zR` / `zM` - Expand/collapse all

### 6. Filtering

Press `/` to filter the timeline:

```
/type:EXECUTE          # Show only EXECUTE blocks
/exit:1                # Show failed commands
/file:main.go          # Blocks related to main.go
/type:EXECUTE exit:0   # Successful commands (AND logic)
```

Press `Esc` to clear filters.

## Learn More

- Full TUI documentation: [docs/tui.md](../../docs/tui.md)
- Block system docs: [docs/packages/ui-blocks.md](../../docs/packages/ui-blocks.md)
- Timeline API: [docs/packages/ui-blocks.md#timeline](../../docs/packages/ui-blocks.md#timeline)
- Minimal example: [examples/tui-demo/](../tui-demo/)
- Streaming example: [examples/tui-streaming/](../tui-streaming/)

## Extending This Example

To add custom block types or modify rendering:

1. Define new `BlockType` constant
2. Add metadata struct (if needed)
3. Implement rendering in `blocks.Renderer`
4. Update `blocks.Validate()` for new type

See [FRD-20251010-block-types-data-model.md](../../specs/frds/FRD-20251010-block-types-data-model.md) for design details.
