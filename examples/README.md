# Spin Core Examples

This directory contains runnable examples demonstrating how to use the Spin core package.

## Examples

### 1. Basic Conversation

Location: `basic-conversation/`

Demonstrates:
- Creating a conversation manager
- Starting a conversation
- Sending messages to the agent
- Processing streaming events

Run:
```bash
go run examples/basic-conversation/main.go
```

### 2. Custom Tool Registration

Location: `custom-tool/`

Demonstrates:
- Defining custom tools
- Registering tools with the registry
- Tool schema definition
- Tool execution

Run:
```bash
go run examples/custom-tool/main.go
```

### 3. Task Mode Switching

Location: `task-modes/`

Demonstrates:
- Different task modes (regular, review, compact)
- Mode-specific behaviors
- Switching between modes
- Use cases for each mode

Run:
```bash
go run examples/task-modes/main.go
```

## Prerequisites

Make sure you have Go 1.24 or later installed:

```bash
go version
```

## Running Examples

All examples are self-contained and can be run directly:

```bash
# Run a specific example
go run examples/<example-name>/main.go

# Or build and run
go build -o example examples/<example-name>/main.go
./example
```

## Note on LLM Providers

These examples use mock LLM providers for demonstration purposes. In a production environment, you would:

1. Use a real LLM provider (e.g., Anthropic's Claude API)
2. Set up API credentials
3. Configure the provider in the manager

Example with real provider:
```go
import "github.com/dmytrogajewski/spin/internal/llm/anthropic"

provider, err := anthropic.NewProvider(apiKey)
if err != nil {
    log.Fatal(err)
}

mgr, err := core.NewManager(cfg,
    core.WithLLMProvider(provider),
)
```

## More Information

For more details, see:
- [Core Package README](../internal/core/README.md)
- [Project Documentation](../README.md)
- [API Reference](https://pkg.go.dev/github.com/dmytrogajewski/spin/internal/core)
