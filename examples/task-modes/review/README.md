# Review Mode Example

## Overview

This example demonstrates Spin's **review mode** - a read-only mode for safe code analysis.

## Features

- **Token Budget**: 12,288 tokens
- **Tools**: Read-only tools only
- **Use Case**: Code review, security audits, understanding existing code

## What This Example Does

1. Creates a conversation in review mode
2. Demonstrates safe code analysis
3. Shows that write operations are blocked
4. Analyzes code without modification risk

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

    // Create manager
    mgr, err := core.NewManager(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Create conversation in REVIEW mode (read-only)
    conv, err := mgr.NewConversationWithTask(ctx, ".", "review")
    if err != nil {
        log.Fatal(err)
    }

    // Verify mode
    fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
    // Output: Current mode: review

    // Send message - agent can only READ, not WRITE
    events, err := conv.SendMessage(ctx, "Review auth/login.go for security vulnerabilities")
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for event := range events {
        switch event.Type {
        case core.EventTypeStreamContent:
            fmt.Print(event.Payload.(string))
        case core.EventTypeTurnComplete:
            fmt.Println("\n✓ Review complete")
        }
    }

    // Try to switch mid-conversation
    err = conv.SetTaskMode("regular")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Switched to regular mode for fixes")
}
```

## Safety Guarantees

### Tools Available (Read-Only)

- ✅ `read_file` - Read file contents
- ✅ `list_directory` - List directory contents
- ✅ `get_context` - Get project context
- ✅ `file_search` - Search for files
- ✅ `git_context` - Get git information

### Tools Blocked (Write Operations)

- ❌ `write_file` - Cannot create or modify files
- ❌ `bash` - Cannot execute commands
- ❌ `apply_patch` - Cannot apply patches

### What This Means

**Safe for:**
- Automated code reviews
- Security audits
- Third-party code analysis
- Pull request reviews

**Cannot:**
- Accidentally modify files
- Execute malicious commands
- Apply unwanted changes

## Use Cases

### Security Audit
```bash
spin --mode review
> Audit all authentication endpoints for SQL injection vulnerabilities
```

### PR Review
```bash
spin --mode review
> Review the changes in auth.go and identify potential bugs
```

### Code Understanding
```bash
spin --mode review
> Explain how the user authentication flow works in this codebase
```

## Switching Modes

If the agent finds issues, switch to regular mode to fix them:

```go
// Review code first (safe, read-only)
conv, _ := mgr.NewConversationWithTask(ctx, ".", "review")
conv.SendMessage(ctx, "Find all TODO comments")

// Switch to regular mode to address TODOs
conv.SetTaskMode("regular")
conv.SendMessage(ctx, "Implement the first TODO you found")
```

## Comparison

| Feature | Review Mode | Regular Mode |
|---------|-------------|--------------|
| Read files | ✅ | ✅ |
| Write files | ❌ | ✅ |
| Execute commands | ❌ | ✅ |
| Safety | High | Normal |
| Use case | Analysis | Implementation |
| Token budget | 12K | 16K |

## Try It

```bash
# CLI
spin --mode review
> Analyze the security of our authentication system

# Interactive switching
spin
> /mode review
> Review this codebase for best practices violations
```

---

**Next**: Try [Compact Mode](../compact/) for fast, minimal queries
