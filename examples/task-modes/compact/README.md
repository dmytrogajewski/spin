# Compact Mode Example

## Overview

This example demonstrates Spin's **compact mode** - optimized for fast, cost-effective queries.

## Features

- **Token Budget**: 4,096 tokens (75% cost savings)
- **Tools**: Only 3 essential tools
- **Use Case**: Quick questions, fast lookups, cost-sensitive operations

## What This Example Does

1. Creates a conversation in compact mode
2. Demonstrates fast query responses
3. Shows minimal tool usage
4. Highlights cost optimization

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

    // Create conversation in COMPACT mode (minimal tools, fast)
    conv, err := mgr.NewConversationWithTask(ctx, ".", "compact")
    if err != nil {
        log.Fatal(err)
    }

    // Verify mode
    fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
    // Output: Current mode: compact

    // Send quick query - fast response, minimal cost
    events, err := conv.SendMessage(ctx, "What does the validateUser function do?")
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for event := range events {
        switch event.Type {
        case core.EventTypeStreamContent:
            fmt.Print(event.Payload.(string))
        case core.EventTypeTurnComplete:
            fmt.Println("\n✓ Query answered")
        }
    }
}
```

## Cost Optimization

### Token Budget Comparison

| Mode | Tokens | Relative Cost |
|------|--------|---------------|
| Regular | 16,384 | 100% |
| Review | 12,288 | 75% |
| **Compact** | **4,096** | **25%** |
| Planning | 4,096 | 25% |

**Compact mode saves 75% vs regular mode!**

### Tools Available (Minimal)

Only 3 essential tools:

1. ✅ `read_file` - Read file contents
2. ✅ `get_context` - Get project context
3. ✅ `file_search` - Search for files

### Tools NOT Available

- ❌ `write_file` - No file writing
- ❌ `bash` - No command execution
- ❌ `list_directory` - Not needed for queries
- ❌ `git_context` - Not essential for quick questions
- ❌ `apply_patch` - Not relevant for queries

## When to Use Compact Mode

### Perfect For

**Quick questions:**
```bash
spin --mode compact
> What's the purpose of the User struct?
```

**Fast lookups:**
```bash
spin --mode compact
> Find the error handling pattern in auth.go
```

**High-volume queries:**
```bash
# When running many queries, compact mode reduces costs
for file in *.go; do
    spin --mode compact <<< "Summarize $file in one sentence"
done
```

### NOT Ideal For

- ❌ Implementing features (use `regular`)
- ❌ Complex analysis (use `review`)
- ❌ Architectural planning (use `planning`)
- ❌ Multi-file operations

## Performance Benefits

### Faster Responses

Smaller context = faster processing:
- **Regular mode**: ~2-5 seconds
- **Compact mode**: ~1-2 seconds (2-3x faster)

### Lower Costs

With pay-per-token pricing:
- **Regular**: $0.40 per 1000 queries
- **Compact**: $0.10 per 1000 queries

**75% cost savings on high-volume usage!**

## Real-World Example

### Before (Regular Mode)

```bash
$ spin  # Default: regular mode
> What does the main function do?
# Response time: 3.2s
# Tokens used: 15,234
# Cost: $0.015
```

### After (Compact Mode)

```bash
$ spin --mode compact
> What does the main function do?
# Response time: 1.1s
# Tokens used: 3,891
# Cost: $0.004
```

**Result:**
- 66% faster response
- 74% lower cost
- Same answer quality for simple queries

## Comparison

| Aspect | Compact | Regular | Review |
|--------|---------|---------|--------|
| Token budget | 4K | 16K | 12K |
| Tool count | 3 | 8+ | 5 |
| Speed | Very Fast | Normal | Fast |
| Cost | Lowest | Highest | Medium |
| Use case | Quick queries | Development | Code review |

## Batch Querying Example

For analyzing many files efficiently:

```go
files := []string{"auth.go", "user.go", "db.go"}
conv, _ := mgr.NewConversationWithTask(ctx, ".", "compact")

for _, file := range files {
    question := fmt.Sprintf("What's the main responsibility of %s?", file)
    events, _ := conv.SendMessage(ctx, question)

    // Process response...
    // Each query is fast and cheap!
}
```

## Try It

```bash
# CLI
spin --mode compact
> Quick question: what's the User type?

# Compare with regular mode
spin  # regular mode
> Quick question: what's the User type?

# Notice the difference in response time and token usage
```

---

**Next**: Try [Planning Mode](../planning/) for architectural design
