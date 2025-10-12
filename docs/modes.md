# Task Modes Guide

## Overview

Spin supports four task modes, each optimized for specific workflows. Task modes control:
- **Token budget**: How much context the LLM can process
- **Tool access**: Which tools are available to the agent
- **System prompt**: Task-specific instructions

Choosing the right mode improves performance, reduces costs, and ensures safety for your workflow.

## Available Modes

### Regular Mode (Default)
**Token Budget**: 16,384 tokens | **Tools**: All tools available

**Use when:**
- Implementing new features
- Debugging complex issues
- General interactive coding
- You need full tool access (read, write, execute)

**Tools Available:**
- File operations: `read_file`, `write_file`, `list_directory`
- Execution: `bash`, `apply_patch`
- Context: `get_context`, `file_search`, `git_context`

**Example:**
```bash
spin --mode regular
> Create a new API endpoint for user authentication with JWT tokens
```

---

### Review Mode
**Token Budget**: 12,288 tokens | **Tools**: Read-only tools

**Use when:**
- Reviewing code changes or pull requests
- Understanding existing code architecture
- Conducting security audits
- Analyzing code quality

**Tools Available** (Read-only):
- `read_file` - Read file contents
- `list_directory` - List directory contents
- `get_context` - Get project context
- `file_search` - Search for files
- `git_context` - Get git repository information

**Example:**
```bash
spin --mode review
> Review the authentication logic in auth.go for potential security vulnerabilities
```

**Safety Guarantee:** Cannot modify files, execute commands, or apply patches.

---

### Compact Mode
**Token Budget**: 4,096 tokens | **Tools**: Minimal (3 essential tools)

**Use when:**
- Quick questions about code
- Fast lookups and simple queries
- Cost-sensitive operations
- You need rapid responses

**Tools Available** (Minimal):
- `read_file` - Read file contents
- `get_context` - Get project context
- `file_search` - Search for files

**Example:**
```bash
spin --mode compact
> What does the validateUser function in auth/user.go do?
```

**Benefits:**
- **75% token cost savings** vs regular mode (4K vs 16K)
- **Faster responses** due to smaller context
- **Focused** on quick information retrieval

---

### Planning Mode
**Token Budget**: 4,096 tokens | **Tools**: Context tools only

**Use when:**
- Planning feature implementations
- Breaking down large tasks
- Understanding project structure
- Making architectural decisions
- Creating implementation roadmaps

**Tools Available** (Context):
- `get_context` - Get project context and structure
- `file_search` - Find relevant files and modules
- `git_context` - Understand repository state and branches

**Example:**
```bash
spin --mode planning
> How should I structure a new user authentication system with SSO support?
```

**Optimized for:**
- High-level architectural thinking
- Task decomposition
- Context gathering without file reading

---

## Usage

### CLI Flag

Start Spin with a specific mode:

```bash
# Long form
spin --mode review

# Short form
spin -m compact

# Default (no flag needed)
spin
# → Uses "regular" mode
```

### Interactive Mode Switching

Switch modes during a session using the `/mode` command:

```bash
$ spin
Welcome to Spin! Type /help for commands.

> /mode
Current mode: regular

> /mode review
Switched to review mode

> Read and analyze src/auth/login.go
[Agent responds using only read-only tools]

> /mode compact
Switched to compact mode

> What's the main function in main.go?
[Agent responds with minimal tools, fast response]

> /help
Commands:
  /mode [name]  - Show or switch task mode
  /help         - Show this help
  /exit         - Exit the session
```

### Mode Information Command

Get detailed information about available modes:

```bash
# List all modes
$ spin mode list
Available modes:
  regular   - Full-featured interactive coding (16K tokens, all tools)
  review    - Read-only code review (12K tokens, read-only tools)
  compact   - Quick queries (4K tokens, 3 essential tools)
  planning  - Task decomposition (4K tokens, context tools)

# Describe specific mode
$ spin mode describe review
Mode: review
Description: Read-only code analysis mode
Max Tokens: 12288
Allowed Tools:
  - read_file
  - list_directory
  - get_context
  - file_search
  - git_context
Best For: Code review, security audits, understanding existing code
```

### WebSocket/JSON-RPC API

Programmatic access via JSON-RPC protocol:

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "send_message",
  "params": {
    "message": "Review this authentication code",
    "task_mode": "review"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "conversation_id": "conv-abc123",
    "turn_id": "turn-xyz789",
    "task_mode": "review"
  }
}
```

**Mode Switching:**
```json
{
  "method": "send_message",
  "params": {
    "conversation_id": "conv-abc123",
    "message": "Now create a test file",
    "task_mode": "regular"
  }
}
```

---

## Best Practices

### Cost Optimization

**Use compact mode for simple queries:**
```bash
# ❌ Expensive
spin --mode regular
> What version is in package.json?

# ✅ Efficient (75% cheaper)
spin --mode compact
> What version is in package.json?
```

**Use planning mode for architecture discussions:**
```bash
# Planning doesn't need file reading
spin --mode planning
> Design a microservices architecture for our e-commerce platform
```

### Safety and Security

**Use review mode for code audits:**
```bash
# Prevents accidental modifications during review
spin --mode review
> Audit all authentication endpoints for SQL injection vulnerabilities
```

**Restrict mode for untrusted operations:**
- Review mode guarantees read-only access
- No risk of file modification or command execution
- Ideal for automated security scans

### Performance

**Smaller modes respond faster:**
- Compact and planning modes (4K tokens) process faster than regular (16K)
- Fewer tools = less context for LLM = quicker responses
- Use the smallest mode that fits your task

**Mode selection guide:**
```
Need to write code?        → regular
Just reading/reviewing?    → review
Quick question?            → compact
Planning/architecture?     → planning
```

---

## Comparison Table

| Feature | Regular | Review | Compact | Planning |
|---------|---------|--------|---------|----------|
| **Token Budget** | 16,384 | 12,288 | 4,096 | 4,096 |
| **Read Files** | ✅ | ✅ | ✅ | ❌ |
| **Write Files** | ✅ | ❌ | ❌ | ❌ |
| **Execute Commands** | ✅ | ❌ | ❌ | ❌ |
| **Apply Patches** | ✅ | ❌ | ❌ | ❌ |
| **Search Files** | ✅ | ✅ | ✅ | ✅ |
| **Git Context** | ✅ | ✅ | ❌ | ✅ |
| **List Directory** | ✅ | ✅ | ❌ | ❌ |
| **Get Context** | ✅ | ✅ | ✅ | ✅ |
| **Relative Cost** | 100% | 75% | 25% | 25% |
| **Response Speed** | Normal | Fast | Very Fast | Very Fast |

---

## Migration from Pre-Mode Version

If you were using Spin before task modes were introduced:

### No Breaking Changes

**Old behavior still works:**
```bash
# This works exactly as before
spin
> Create a new file test.txt with content "hello"
```
→ Automatically uses "regular" mode (same behavior as before)

### Opt-In New Features

**Explicitly choose modes:**
```bash
# New capability: read-only mode
spin --mode review
> Analyze this code

# New capability: fast queries
spin --mode compact
> Quick question about the code?
```

### Backward Compatibility

- **Default mode**: "regular" (identical to pre-mode behavior)
- **No API changes**: All existing code works unchanged
- **No config changes**: No configuration updates required
- **Opt-in**: Task modes are completely optional

---

## Troubleshooting

### "Invalid task mode" Error

**Error:**
```
Error: invalid task mode: reveiw (valid: regular, review, compact, planning)
```

**Solutions:**
1. Check spelling: `review` not `reveiw`
2. Use exact mode names: `regular`, `review`, `compact`, `planning`
3. Mode names are case-sensitive (use lowercase)

### Tools Not Available

**Problem:** "Tool 'write_file' not found" in review mode

**Reason:** Review mode is read-only

**Solution:**
```bash
> /mode regular
Switched to regular mode

> Now you can write files
```

### Which Mode Should I Use?

**Decision tree:**

```
Start here: What do you want to do?

├─ Write/modify code?
│  └─ Use: regular
│
├─ Read/analyze code?
│  └─ Use: review
│
├─ Quick question?
│  └─ Use: compact
│
└─ Plan/design work?
   └─ Use: planning
```

**Still not sure?** Use `regular` mode (default).

### Mode Won't Switch

**Problem:** Mode command doesn't work

**Check:**
1. Are you in interactive mode? (Started with `spin`, not `spin <file>`)
2. Is the syntax correct? `/mode review` not `mode review`
3. Is the mode name valid? Use `/mode` without arguments to see current mode

---

## Examples

See [`examples/task-modes/`](../examples/task-modes/) for complete working examples demonstrating each mode:

- [`regular/main.go`](../examples/task-modes/regular/) - Full-featured coding example
- [`review/main.go`](../examples/task-modes/review/) - Code review example
- [`compact/main.go`](../examples/task-modes/compact/) - Quick query example
- [`planning/main.go`](../examples/task-modes/planning/) - Planning mode example

---

## API Reference

For programmatic usage, see:
- [Core Package - Task Modes](./packages/core.md#task-modes)
- [Tools Package - Tool Filtering](./packages/tools.md#tool-filtering-by-task-mode)
- [Protocol Package](./packages/protocol.md) - JSON-RPC API details

---

**Last Updated:** 2025-10-12
**Version:** 1.0.0
**Status:** ✅ Production Ready
