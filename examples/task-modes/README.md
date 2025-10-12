# Task Modes Examples

This directory contains working examples demonstrating each of Spin's task modes.

## Overview

Spin supports four task modes, each optimized for specific workflows:

- **regular**: Full-featured interactive coding (16K tokens, all tools)
- **review**: Read-only code analysis (12K tokens, read-only tools)
- **compact**: Quick queries (4K tokens, minimal tools)
- **planning**: Task decomposition (4K tokens, context tools)

## Examples

### [Regular Mode Example](./regular/)

Demonstrates full-featured agent with all tools available.

**What it does:**
- Creates and manages conversations with write access
- Shows file creation, modification, and command execution
- Uses full token budget (16K)

**Run:**
```bash
cd regular
go run main.go
```

---

### [Review Mode Example](./review/)

Demonstrates read-only code analysis mode.

**What it does:**
- Analyzes code without modification capabilities
- Shows safe code review workflow
- Uses read-only tools (read_file, file_search, git_context)

**Run:**
```bash
cd review
go run main.go
```

---

### [Compact Mode Example](./compact/)

Demonstrates fast, minimal-context queries.

**What it does:**
- Answers quick questions with minimal overhead
- Uses only 3 essential tools (read_file, get_context, file_search)
- Demonstrates 75% cost savings vs regular mode

**Run:**
```bash
cd compact
go run main.go
```

---

### [Planning Mode Example](./planning/)

Demonstrates architectural planning and task decomposition.

**What it does:**
- Plans features and breaks down tasks
- Uses context tools without file reading
- Shows high-level architectural thinking

**Run:**
```bash
cd planning
go run main.go
```

---

## Running All Examples

```bash
# Run each example
for mode in regular review compact planning; do
    echo "=== Running $mode mode example ==="
    (cd $mode && go run main.go)
    echo
done
```

## Requirements

- Go 1.24+
- LLM provider configured (OpenAI, Ollama, or LM Studio)
- API key set in environment (if using OpenAI)

## Configuration

Examples use the default Spin configuration. To customize:

1. Copy a config file from `../` (e.g., `config-openai.yaml`)
2. Update the example code to load your config
3. Set environment variables as needed

## What You'll Learn

### Regular Mode
- Full agent capabilities
- File system operations
- Command execution
- When to use unrestricted mode

### Review Mode
- Read-only safety guarantees
- Code review workflows
- Tool access restrictions
- Security audit patterns

### Compact Mode
- Cost optimization (75% savings)
- Fast query patterns
- Minimal tool usage
- When to trade features for speed

### Planning Mode
- Architectural thinking
- Task breakdown
- Context-only operations
- High-level design workflows

## Troubleshooting

**"Provider not configured" error:**
- Set up your LLM provider (see main README.md)
- Ensure API keys are set in environment

**Examples fail to compile:**
- Run `go mod tidy` in the project root
- Ensure you're using Go 1.24+

**Examples run but no output:**
- Check provider connectivity
- Verify API keys are valid
- Review logs for errors

## Next Steps

After running these examples:

1. Read [Task Modes Guide](../../docs/modes.md)
2. Review [Core Package Docs](../../docs/packages/core.md#task-modes)
3. Explore [Tool Filtering](../../docs/packages/tools.md#tool-filtering-by-task-mode)
4. Try the CLI: `spin --mode review`

---

**Last Updated:** 2025-10-12
