# Core Module

The `internal/core` package provides the core business logic and orchestration for the Spin AI coding agent.

## Overview

This package implements all the essential functionality needed for an autonomous coding agent, including conversation management, agent orchestration, task execution with safety controls, state management, and event streaming.

## Architecture

The core package is organized into several layers:

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
- **Planner**: Task decomposition
- **Validator**: Command safety classification

### Supporting Infrastructure
- **event.go**: Event types and emission
- **context.go**: Environment context gathering
- **config.go**: Configuration management
- **error.go**: Error types and handling

## Key Components

### Manager (`manager.go`)
Entry point for creating and managing conversations. Coordinates conversation lifecycle and state management.

### Conversation (`conversation.go`)
Represents an active conversation with the AI agent. Handles turn execution and event streaming.

### Agent (`agent.go`)
Implements the core agent logic and decision-making loop. Coordinates LLM interactions and tool execution.

### Executor (`executor.go`)
Manages task execution with proper isolation and safety controls. Integrates with security sandbox.

### Validator (`validator.go`)
Classifies command safety and validates commands against security policies.

### Context (`context.go`)
Gathers environment information (OS, Git, project structure) for the AI agent.

### History (`history.go`)
Manages conversation message history with token-aware truncation to fit context windows.

## Package Structure

```
internal/core/
├── doc.go                 # Package documentation
├── manager.go             # Conversation manager
├── conversation.go        # Active conversation
├── agent.go               # Agent orchestration
├── executor.go            # Task execution
├── planner.go             # Task planning
├── context.go             # Environment context
├── history.go             # Conversation history
├── validator.go           # Command validation
├── event.go               # Event types
├── error.go               # Error handling
├── config.go              # Configuration
│
├── session/               # Session state management
│   ├── session.go
│   ├── storage.go
│   └── metadata.go
│
├── turn/                  # Turn state management
│   ├── turn.go
│   ├── state.go
│   └── result.go
│
├── task/                  # Task execution modes
│   ├── task.go            # Task interface
│   ├── regular.go         # Standard mode
│   ├── review.go          # Code review mode
│   └── compact.go         # Minimal context mode
│
├── stream/                # Streaming infrastructure
│   ├── stream.go
│   ├── buffer.go
│   └── types.go
│
└── testing/               # Test utilities
    ├── mock_llm.go
    ├── mock_tools.go
    └── helpers.go
```

## Usage

Basic usage pattern:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/dmytrogajewski/spin/internal/core"
)

func main() {
    // Create a manager
    cfg := &core.Config{
        WorkDir: "/path/to/project",
        MaxTurns: 10,
    }
    
    mgr, err := core.NewManager(cfg,
        core.WithLLMProvider(provider),
        core.WithToolRegistry(tools),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Start a conversation
    ctx := context.Background()
    conv, err := mgr.NewConversation(ctx, cfg.WorkDir)
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute a turn
    err = conv.RunTurn(ctx, "Implement user authentication")
    if err != nil {
        log.Fatal(err)
    }
    
    // Stream events
    for event := range conv.Stream() {
        fmt.Printf("%s: %v\n", event.Type, event.Data)
    }
}
```

## Testing

Run tests with:

```bash
# All tests
go test ./internal/core/...

# With coverage
go test -cover ./internal/core/...

# With race detector
go test -race ./internal/core/...

# Verbose output
go test -v ./internal/core/...

# Specific test
go test -run TestConversation_RunTurn ./internal/core
```

Or use the Makefile:

```bash
make test
make test-coverage
make test-race
```

## Dependencies

### External Dependencies
- `golang.org/x/sync/errgroup`: Concurrent error handling
- `gopkg.in/yaml.v3`: Configuration file parsing

### Internal Dependencies
- `internal/llm`: LLM provider interface
- `internal/tools`: Tool implementations
- `internal/security`: Sandbox and policy enforcement
- `internal/mcp`: Model Context Protocol client

## Design Principles

The core package follows these design principles:

1. **Clean Architecture**: Dependencies point inward, core is independent of frameworks
2. **SOLID Principles**: Especially interface segregation and dependency inversion
3. **Go Idioms**: Accept interfaces, return structs
4. **Concurrency**: Safe concurrent access with proper synchronization
5. **Error Handling**: Errors are wrapped with context using `fmt.Errorf` with `%w`
6. **Context Propagation**: `context.Context` used throughout for cancellation

## Development

### Building

```bash
make build
```

### Linting

```bash
make lint
```

### Formatting

```bash
make fmt
```

### All checks

```bash
make all
```

## Documentation

- [Core Module Specification](../../specs/core-module/spec.md)
- [Core Module Roadmap](../../specs/core-module/ROADMAP.md)
- [Architecture Overview](../../specs/architecture-overview.md)

## License

Apache 2.0 - See LICENSE file in the root directory.

