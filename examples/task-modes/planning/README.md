# Planning Mode Example

## Overview

This example demonstrates Spin's **planning mode** - optimized for architectural planning and task decomposition.

## Features

- **Token Budget**: 4,096 tokens
- **Tools**: Context tools only (no file reading)
- **Use Case**: Feature planning, architecture design, task breakdown

## What This Example Does

1. Creates a conversation in planning mode
2. Demonstrates high-level architectural thinking
3. Shows task decomposition
4. Plans without needing file contents

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

    // Create conversation in PLANNING mode (context-only, no file reading)
    conv, err := mgr.NewConversationWithTask(ctx, ".", "planning")
    if err != nil {
        log.Fatal(err)
    }

    // Verify mode
    fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
    // Output: Current mode: planning

    // Ask planning question - gets project context without reading files
    events, err := conv.SendMessage(ctx,
        "How should I structure a new user authentication system with OAuth2 and JWT support?")
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for event := range events {
        switch event.Type {
        case core.EventTypeStreamContent:
            fmt.Print(event.Payload.(string))
        case core.EventTypeTurnComplete:
            fmt.Println("\n✓ Plan complete")
        }
    }
}
```

## Tools Available (Context-Only)

Planning mode has 3 context tools:

1. ✅ `get_context` - Get project structure and context
2. ✅ `file_search` - Find relevant files and modules
3. ✅ `git_context` - Get repository state and branches

### Tools NOT Available

- ❌ `read_file` - No file content reading
- ❌ `write_file` - No file writing
- ❌ `bash` - No command execution
- ❌ `list_directory` - Not needed for planning
- ❌ `apply_patch` - Not relevant for planning

## Why No File Reading?

Planning mode is optimized for **high-level thinking**, not implementation details:

**Good for planning:**
- "How should I architect a microservices system?"
- "What's the best way to structure database migrations?"
- "Break down the user authentication feature into tasks"

**Not ideal (needs file reading):**
- "Review the current authentication implementation" → Use `review`
- "What does auth.go do?" → Use `compact`

## When to Use Planning Mode

### Perfect For

**Feature Planning:**
```bash
spin --mode planning
> Plan the implementation of a real-time chat feature with WebSockets
```

**Architectural Decisions:**
```bash
spin --mode planning
> Should we use microservices or monolith for our e-commerce platform?
```

**Task Breakdown:**
```bash
spin --mode planning
> Break down the 'Add payment processing' feature into implementable tasks
```

**Project Structure:**
```bash
spin --mode planning
> How should I organize a new Go project with authentication, API, and database layers?
```

### NOT Ideal For

- ❌ Code review (use `review`)
- ❌ Implementation (use `regular`)
- ❌ Quick code queries (use `compact`)

## Example Planning Session

### Input

```bash
spin --mode planning
> I need to add multi-tenant support to our SaaS application.
  Plan the architecture and break it down into tasks.
```

### Expected Output

The agent will:
1. Ask clarifying questions about requirements
2. Propose architectural approaches
3. Break down into phases:
   - Database schema changes
   - Authentication updates
   - Data isolation
   - Testing strategy
4. Provide implementation order

### Then Switch to Implementation

```bash
> /mode regular
Switched to regular mode

> Implement task 1: Add tenant_id column to users table
```

## Workflow: Plan → Implement

Typical workflow combining modes:

```go
// 1. PLAN: Use planning mode for architecture
conv, _ := mgr.NewConversationWithTask(ctx, ".", "planning")
conv.SendMessage(ctx, "Design a rate limiting system")
// Agent proposes: token bucket algorithm, Redis storage, middleware approach

// 2. REVIEW: Check existing code
conv.SetTaskMode("review")
conv.SendMessage(ctx, "Review current middleware implementation")
// Agent analyzes existing patterns

// 3. IMPLEMENT: Build the feature
conv.SetTaskMode("regular")
conv.SendMessage(ctx, "Implement the rate limiter middleware")
// Agent writes code

// 4. VERIFY: Quick checks
conv.SetTaskMode("compact")
conv.SendMessage(ctx, "Is the rate limiter thread-safe?")
// Agent confirms concurrency safety
```

## Benefits of Planning Mode

### Focused on Strategy

- No distraction from implementation details
- High-level architectural thinking
- Clear task decomposition

### Efficient

- 4K token budget (75% cheaper than regular)
- Fast responses (no file I/O)
- Minimal tool overhead

### Collaborative

Great for starting conversations:
1. Plan with the agent
2. Get structured task list
3. Switch to regular mode to implement
4. Use review mode to validate

## Example: Breaking Down a Feature

### Input

```bash
spin --mode planning
> Break down 'Add email verification' into implementable tasks
```

### Output

The agent might respond:

```
Email Verification Feature - Task Breakdown:

Phase 1: Database & Models
1. Add email_verified boolean to users table
2. Add verification_token string column
3. Add verification_expires_at timestamp
4. Create database migration

Phase 2: Token Generation
5. Implement generateVerificationToken() function
6. Add email validation logic
7. Store token with expiration (24h)

Phase 3: Email Service
8. Create email template for verification
9. Integrate email service (SendGrid/AWS SES)
10. Send verification email on signup

Phase 4: Verification Endpoint
11. Create /verify endpoint
12. Validate token and expiration
13. Update user.email_verified on success
14. Return appropriate errors

Phase 5: Middleware & Protection
15. Create RequireVerifiedEmail() middleware
16. Protect sensitive endpoints
17. Add resend verification endpoint

Phase 6: Testing
18. Unit tests for token generation
19. Integration tests for verification flow
20. E2E test with email mocking

Recommended order: 1-4, 5-7, 8-10, 11-14, 15-17, 18-20
```

Now you can tackle tasks 1-20 in regular mode!

## Comparison

| Aspect | Planning | Regular | Review | Compact |
|--------|----------|---------|--------|---------|
| Token budget | 4K | 16K | 12K | 4K |
| Tools | 3 context | 8+ all | 5 read | 3 minimal |
| Reads files | ❌ | ✅ | ✅ | ✅ |
| Use case | Architecture | Coding | Analysis | Queries |
| Speed | Very Fast | Normal | Fast | Very Fast |
| Cost | Lowest | Highest | Medium | Lowest |

## Try It

```bash
# CLI
spin --mode planning
> How should I implement user roles and permissions?

# Then implement
> /mode regular
> Implement the basic Role struct
```

---

**Complete!** You've seen all four task modes. See [Task Modes Guide](../../../docs/modes.md) for comprehensive documentation.
