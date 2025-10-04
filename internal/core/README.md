# Core Package

The `core` package provides the core business logic and orchestration for the Spin AI coding agent.

## Overview

This package implements all the essential functionality needed for an autonomous coding agent, including conversation management, agent orchestration, task execution with safety controls, state management, event streaming, and command validation.

## Features

- **Conversation Management**: Create and manage multi-turn conversations with context persistence
- **Agent Orchestration**: Coordinate LLM interactions, tool calls, and decision-making
- **Task Execution**: Safe command execution with validation and sandboxing
- **State Management**: Track sessions, turns, and conversation history
- **Event Streaming**: Real-time UI updates through event channels
- **Security**: Command validation and safety classification
- **Observability**: Structured logging and OpenTelemetry tracing

## Architecture

The package is organized into several layers:

### Public API Layer
- **Manager**: High-level conversation manager (entry point)
- **Conversation**: Active conversation instance
- **Agent**: Core agent orchestration

### State Management Layer
- **session/**: Persistent session state
- **turn/**: Turn state machine
- **History**: Conversation history with token-aware truncation

### Task Execution Layer
- **Executor**: Safe command execution
- **Validator**: Command safety classification

### Supporting Infrastructure
- **event.go**: Event types and emission
- **config.go**: Configuration management
- **logger.go**: Structured logging
- **tracing.go**: Distributed tracing

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"

    "github.com/dmytrogajewski/spin/internal/core"
    "github.com/dmytrogajewski/spin/internal/llm"
    "github.com/dmytrogajewski/spin/internal/tools"
)

func main() {
    // Create configuration
    cfg := &core.Config{
        MaxTurns:    10,
        Temperature: 0.7,
        MaxTokens:   2048,
        Model:       "claude-3-5-sonnet-20241022",
    }

    // Create manager
    mgr, err := core.NewManager(
        cfg,
        core.WithLLMProvider(llmProvider),
        core.WithToolRegistry(toolRegistry),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Start conversation
    ctx := context.Background()
    conv, err := mgr.NewConversation(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Send message
    events, err := conv.SendMessage(ctx, "Hello!")
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for event := range events {
        switch event.Type {
        case core.EventTypeStreamContent:
            // Handle streaming content
        case core.EventTypeTurnComplete:
            // Handle completion
        case core.EventTypeError:
            // Handle error
        }
    }
}
```

### With Custom Tools

```go
// Create tool registry
registry := tools.NewRegistry()

// Register custom tool
customTool := &tools.Tool{
    Name:        "my_tool",
    Description: "Does something useful",
    InputSchema: tools.InputSchema{
        Type: "object",
        Properties: map[string]tools.Property{
            "param": {
                Type:        "string",
                Description: "A parameter",
            },
        },
        Required: []string{"param"},
    },
}
registry.Register(customTool)

// Use in manager
mgr, _ := core.NewManager(cfg,
    core.WithToolRegistry(registry),
)
```

### Task Modes

```go
// Create task registry
taskRegistry := task.NewRegistry()

// Register modes
taskRegistry.Register("regular", task.NewRegular(cfg))
taskRegistry.Register("review", task.NewReview(cfg))
taskRegistry.Register("compact", task.NewCompact(cfg))
taskRegistry.SetDefault("regular")

// Create manager with task modes
mgr, _ := core.NewManager(cfg,
    core.WithTaskRegistry(taskRegistry),
)
```

### With Observability

```go
// Enable structured logging
cfg.Debug = true
cfg.LogLevel = "debug"
cfg.LogFormat = "json"

// Enable tracing
cfg.EnableTrace = true

// Initialize logger
core.InitLogger(cfg)

// Initialize tracing
core.InitOtelTracing(cfg)

// Use manager - all operations are now traced and logged
mgr, _ := core.NewManager(cfg, ...)
```

## Configuration

### Config Options

```go
type Config struct {
    // LLM Settings
    Model       string  // Model name (default: claude-3-5-sonnet-20241022)
    Temperature float64 // Temperature (0-1, default: 0.7)
    MaxTokens   int     // Max output tokens (default: 4096)

    // Agent Settings
    MaxTurns    int           // Max turns per conversation (default: 50)
    Timeout     time.Duration // Agent timeout (default: 5m)

    // Task Settings
    DefaultTask string // Default task mode (default: "regular")

    // Security Settings
    RequireApproval bool     // Require approval for commands
    SafeCommands    []string // List of safe commands

    // Logging Settings
    Debug     bool   // Enable debug logging
    LogLevel  string // Log level: debug, info, warn, error
    LogFormat string // Log format: text, json

    // Tracing Settings
    EnableTrace bool // Enable OpenTelemetry tracing
}
```

### Manager Options

```go
// WithLLMProvider sets the LLM provider
core.WithLLMProvider(provider llm.Provider)

// WithToolRegistry sets the tool registry
core.WithToolRegistry(registry *tools.Registry)

// WithTaskRegistry sets the task registry
core.WithTaskRegistry(registry *task.Registry)

// WithEventEmitter sets custom event emitter
core.WithEventEmitter(emitter EventEmitter)
```

## Events

The core package emits events for real-time updates:

```go
type EventType string

const (
    EventTypeStreamStart   EventType = "stream_start"
    EventTypeStreamContent EventType = "stream_content"
    EventTypeStreamEnd     EventType = "stream_end"
    EventTypeTurnStart     EventType = "turn_start"
    EventTypeTurnComplete  EventType = "turn_complete"
    EventTypeToolCall      EventType = "tool_call"
    EventTypeToolResult    EventType = "tool_result"
    EventTypeError         EventType = "error"
    EventTypeThinking      EventType = "thinking"
)
```

## State Management

### Conversation State

```go
type State string

const (
    StateIdle       State = "idle"
    StateActive     State = "active"
    StateWaiting    State = "waiting"
    StateCompleted  State = "completed"
    StateError      State = "error"
)

// Get current state
state := conv.GetState()

// Check if active
isActive := conv.IsActive()
```

### Turn State

Each turn progresses through states:
- `pending` → `running` → `completed`
- Or `pending` → `running` → `failed`

### History Management

```go
// Get conversation history
history := conv.GetHistory()

// History automatically truncates based on token limits
// to prevent context overflow
```

## Security

### Command Validation

```go
// Commands are classified as:
// - Safe: Auto-approved
// - Neutral: May require approval
// - Dangerous: Requires approval

validator := core.NewValidator(cfg)
safety := validator.ClassifyCommand("rm -rf /")
// Returns: SafetyDangerous
```

### Executor Sandbox

```go
// Commands execute in a controlled environment
executor := core.NewExecutor(cfg, validator)
result, err := executor.Execute(ctx, command)
```

## Testing

### Running Tests

```bash
# All tests
go test ./internal/core/...

# With coverage
go test -cover ./internal/core/...

# With race detection
go test -race ./internal/core/...

# Specific package
go test ./internal/core/task/
```

### Test Coverage

Current coverage:
- Overall: 85.1%
- Turn: 95.6%
- Task: 96.6%
- Stream: 89.8%
- Core: 83.1%

## Examples

See the `examples/` directory for complete examples:
- `basic-conversation/`: Basic usage example
- `custom-tool/`: Custom tool registration
- `task-modes/`: Task mode switching

## Performance

### Benchmarks

```bash
go test -bench=. ./internal/core/...
```

Key metrics:
- Turn creation: ~100ns
- Event emission: ~50ns
- Logging overhead: <5%
- Tracing overhead: ~4ns per span

### Optimization Tips

1. **Use Compact Mode** for simple queries
2. **Enable Tracing Conditionally** in production
3. **Configure Token Limits** based on use case
4. **Batch Tool Calls** when possible

## Troubleshooting

### Common Issues

**Issue**: High memory usage
- **Solution**: Reduce `MaxTurns` or implement history pruning

**Issue**: Slow response times
- **Solution**: Use compact mode or reduce `MaxTokens`

**Issue**: Too many tool calls
- **Solution**: Adjust agent prompt or reduce available tools

**Issue**: Context overflow
- **Solution**: History auto-truncates, but check `MaxTokens`

### Debug Mode

Enable debug logging:
```go
cfg.Debug = true
cfg.LogLevel = "debug"
core.InitLogger(cfg)
```

Enable tracing:
```go
cfg.EnableTrace = true
core.InitOtelTracing(cfg)
```

## Architecture Patterns

### Clean Architecture
- **Domain Layer**: Core business logic (agent, state, history)
- **Application Layer**: Use cases (manager, conversation)
- **Infrastructure Layer**: External dependencies (LLM, tools)

### SOLID Principles
- **Single Responsibility**: Each component has one clear purpose
- **Open/Closed**: Extensible through options and interfaces
- **Liskov Substitution**: Interface-based design
- **Interface Segregation**: Minimal, focused interfaces
- **Dependency Inversion**: Depend on abstractions

### Design Patterns
- **Builder**: Manager creation with options
- **Observer**: Event emission system
- **State Machine**: Turn and conversation states
- **Strategy**: Task mode selection
- **Factory**: Task and tool registries

## Contributing

When contributing to the core package:

1. **Maintain Test Coverage**: Keep coverage above 85%
2. **Follow Go Idioms**: Use standard library patterns
3. **Document Exports**: All public APIs must have godoc
4. **Run Linters**: `make lint` must pass
5. **Benchmark Changes**: Performance-sensitive code needs benchmarks

## Related Packages

- **internal/llm**: LLM provider interfaces and implementations
- **internal/tools**: Tool registry and definitions
- **internal/core/task**: Task mode implementations
- **internal/core/turn**: Turn state management
- **internal/core/stream**: Event streaming

## License

See project LICENSE file.
