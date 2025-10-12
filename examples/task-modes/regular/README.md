# Regular Mode Example

## Overview

This example demonstrates Spin's **regular mode** - the default, full-featured interactive coding mode.

## Features

- **Token Budget**: 16,384 tokens
- **Tools**: All tools available
- **Use Case**: General development, feature implementation, debugging

## What This Example Does

1. Creates a conversation in regular mode (default)
2. Demonstrates file creation (`write_file`)
3. Shows command execution (`bash`)
4. Applies code patches (`apply_patch`)
5. Reads and searches files

## API Pattern

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dmytrogajewski/spin/internal/core"
)

func main() {
    ctx := context.Background()

    // Create manager (regular mode is default)
    mgr, err := core.NewManager(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Create conversation - defaults to "regular" mode
    conv, err := mgr.NewConversation(ctx, ".")
    if err != nil {
        log.Fatal(err)
    }

    // Verify mode
    fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
    // Output: Current mode: regular

    // Send message - agent has all tools available
    events, err := conv.SendMessage(ctx, "Create a new file hello.go with a simple Hello World program")
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for event := range events {
        switch event.Type {
        case core.EventTypeStreamContent:
            fmt.Print(event.Payload.(string))
        case core.EventTypeTurnComplete:
            fmt.Println("\n✓ Turn complete")
        }
    }
}
```

## What Makes Regular Mode Special

### Full Tool Access

Unlike restricted modes, regular mode has access to:
- ✅ `write_file` - Create and modify files
- ✅ `bash` - Execute shell commands
- ✅ `apply_patch` - Apply code patches
- ✅ All read-only tools

### Maximum Token Budget

- 16K tokens allows for complex, multi-step tasks
- Handles larger codebases and longer conversations
- No artificial restrictions

### When to Use

Use regular mode when:
- Implementing new features
- Debugging complex issues
- Refactoring code
- You need full agent capabilities

## Comparison with Other Modes

| Capability | Regular | Review | Compact | Planning |
|------------|---------|--------|---------|----------|
| Create files | ✅ | ❌ | ❌ | ❌ |
| Execute commands | ✅ | ❌ | ❌ | ❌ |
| Read files | ✅ | ✅ | ✅ | ❌ |
| Token budget | 16K | 12K | 4K | 4K |

## Try It

```bash
# Using CLI
spin --mode regular
> Create a REST API endpoint for user authentication

# Or just (regular is default)
spin
> Implement a binary search tree in Go
```

---

**Next**: Try [Review Mode](../review/) for read-only code analysis
