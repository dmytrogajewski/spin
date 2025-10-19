# Core Package

The `core` package provides the core business logic and orchestration for the Spin AI coding agent.

## Overview

This package implements all the essential functionality needed for an autonomous coding agent, including conversation management, agent orchestration, task execution with safety controls, state management, event streaming, and command validation.

## Features

- **Conversation Management**: Create and manage multi-turn conversations with context persistence
- **Agent Orchestration**: Coordinate LLM interactions, tool calls, and decision-making
- **Task Execution**: Safe command execution with validation and sandboxing
- **State Management**: Track sessions, turns, and conversation history
- **Event Streaming**: Real-time UI updates through strongly typed event payloads
- **Security**: Command validation and safety classification
- **Observability**: Structured logging and OpenTelemetry tracing

## Architecture

The package is organized into several layers:

### Public API Layer
- **Manager**: High-level conversation manager (entry point)
  - Refactored (2025-10-19): NewConversation method complexity reduced from 47 to 5
  - Clean separation of concerns with 15+ helper methods
- **Conversation**: Active conversation instance
- **Agent**: Core agent orchestration

### State Management Layer
- **turn/**: Turn state machine
- **History**: Conversation history with token-aware truncation

### Task Execution Layer
- **Executor**: Safe command execution
- **Validator**: Command safety classification

- **event.go**: Event emitter and typed payload definitions
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

Task modes provide specialized agent behavior with specific tool access and token budgets. The agent automatically initializes with 4 built-in modes:

- **regular**: Full-featured interactive coding (16K tokens, all tools) - default
- **review**: Read-only code analysis (12K tokens, read-only tools)
- **compact**: Quick queries (4K tokens, minimal tools)
- **planning**: Task decomposition (4K tokens, context tools)

```go
// Agent automatically includes all 4 modes by default
agent, _ := core.NewAgent(llm, executor, validator, ctx, emitter)

// List available modes
modes := agent.ListTaskModes()
// Returns: ["compact", "planning", "regular", "review"]

// Get the task registry
registry := agent.GetTaskRegistry()
task, _ := registry.Get("review")
fmt.Println("Review mode allows:", task.AllowedTools())

// Provide custom task registry (replaces defaults)
customRegistry := task.NewRegistry()
customRegistry.Register("my-mode", task.NewCompact())
customRegistry.SetDefault("my-mode")

agent, _ := core.NewAgent(llm, executor, validator, ctx, emitter,
    core.WithTaskRegistry(customRegistry),
)
```

See [task package documentation](./task.md) for detailed information on task modes.

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

// WithToolRegistry sets the tool registry for all conversations
core.WithManagerToolRegistry(registry *tools.Registry)

// WithManagerTaskRegistry sets the task registry for all conversations
core.WithManagerTaskRegistry(registry *task.Registry)

// WithEventEmitter sets custom event emitter
core.WithEventEmitter(emitter EventEmitter)

// WithApprovalHandler sets the command approval handler
core.WithManagerApprovalHandler(handler ApprovalHandler)
```

### Creating Conversations with Task Modes

The Manager provides two methods for creating conversations:

**NewConversation** - Creates conversation with default task mode (regular):
```go
conv, err := mgr.NewConversation(ctx, workDir)
// Conversation starts in "regular" mode
```

**NewConversationWithTask** - Creates conversation in specific mode:
```go
// Start directly in review mode (read-only)
conv, err := mgr.NewConversationWithTask(ctx, workDir, "review")

// All 4 built-in modes available:
// - "regular"  : Full-featured interactive coding
// - "review"   : Read-only code analysis
// - "compact"  : Quick queries with minimal tools
// - "planning" : Task decomposition
```

You can also customize available modes using WithManagerTaskRegistry:
```go
registry := task.NewRegistry()
registry.Register("custom", task.NewCompact())
registry.SetDefault("custom")

mgr := NewManager(cfg, WithManagerTaskRegistry(registry))
conv, _ := mgr.NewConversationWithTask(ctx, workDir, "custom")
```

### Approval Handler

The approval handler allows UI modules to intercept dangerous command requests and obtain user approval:

```go
// Define approval handler
handler := func(req core.ApprovalRequest) core.ApprovalResponse {
    // Display approval dialog to user
    approved := showDialog(req.Command.Raw, req.Reason)

    return core.ApprovalResponse{
        RequestID: req.ID,
        Approved:  approved,
        Reason:    "user decision",
        Timestamp: time.Now(),
    }
}

// Create agent with handler
agent, _ := core.NewAgent(llm, executor, validator, ctx, emitter,
    core.WithApprovalHandler(handler),
)
```

**ApprovalRequest fields:**
- `ID` - Unique request identifier (UUID)
- `Command` - Command requiring approval
- `Reason` - Why approval is needed (from Validator)
- `WorkDir` - Working directory
- `Timestamp` - When request was created

**ApprovalResponse fields:**
- `RequestID` - Must match request ID
- `Approved` - true = approve, false = deny
- `Reason` - User-provided reason (optional)
- `ModifiedCommand` - Modified command (optional, will be re-validated)
- `Timestamp` - When response was created

**Features:**
- **Timeout**: Default 60s, configurable via `config.ApprovalTimeout`
- **Context Cancellation**: Respects context cancellation
- **Command Modification**: User can modify command before approval
- **Auto-Deny**: If no handler is set, commands are automatically denied
- **Events**: Emits `EventCommandApproval`, `EventCommandApproved`, or `EventCommandDenied`

## Events

### Event Payloads

The event stream delivers concrete structs instead of loosely typed maps. Key payload types include:

- `ContentDeltaData` – incremental assistant/tool content
- `ToolCallStartData`, `ToolCallProgressData`, `ToolCallCompleteData`
- `TurnEventData` – turn counters, status, and limits
- `ApprovalEventData` – approval requests and decisions
- `ErrorData` / `SystemEventData` – structured status messages

`ToolCallStartData` exposes parameters via `types.ToolCallArguments`, allowing consumers to safely decode tool arguments in the UI and protocol layers.

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

// History automatically compresses when approaching token limits
// to prevent context overflow
```

### Context Compression

**Automatic history compression** prevents context overflow in long conversations.

**How It Works:**
- Triggers automatically at 80% token capacity
- Two compression strategies available:
  - **Hybrid** (default): Importance-weighted message selection
  - **LLM Summary**: Uses LLM to summarize old messages (optional)
- Critical messages (user requests, tool results, errors) always preserved
- Compression ratio: 40-60% reduction

**Configuration:**
```go
// Create history with custom compression config
config := &core.HistoryConfig{
    CompressionEnabled:   true,   // Enable compression (default)
    CompressionThreshold: 0.8,    // Compress at 80% capacity
    PreserveCritical:     true,   // Always keep critical messages
    MinRetention:         0.3,    // Keep at least 30% of messages
}

history := core.NewHistoryWithConfig(16384, tokenizer, config)
```

**Compression Strategies:**

**1. Composite Strategy (Recommended):**
- LLM summarization (primary) + Hybrid compression (fallback)
- Best semantic preservation with reliability guarantee
- Automatically falls back if LLM unavailable
- Recommended for production

```go
// Recommended: Composite strategy with LLM + hybrid fallback
history := core.NewHistoryWithLLMSummarization(16384, tokenizer, llmProvider, nil)
```

**2. Hybrid Strategy (Default/Fast):**
- Importance-weighted selection only
- Fast (<2ms for 1000 messages)
- No LLM calls required
- Good for offline/fast scenarios

```go
// Fast compression without LLM
history := core.NewHistory(16384, tokenizer) // Uses hybrid by default
```

**3. LLM Summary Only (Advanced):**
- Uses only LLM summarization (no hybrid fallback)
- Maximum semantic preservation
- May fail if LLM unavailable

```go
// Advanced: LLM-only (no fallback)
import (
    "github.com/dmytrogajewski/spin/internal/core/history"
    "github.com/dmytrogajewski/spin/internal/core/history/compress"
)

adapter := history.NewLLMProviderAdapter(llmProvider)
summarizer := compress.NewDefaultLLMSummarizer(adapter)
hist.SetCompressor(summarizer)
```

**Importance Levels:**
- **Critical** (100% retention): User messages, tool results, errors
- **High** (prioritized): Code blocks, diffs, decisions
- **Medium** (included if space): Regular assistant responses
- **Low** (compressed first): Verbose reasoning, "thinking" content

**Performance:**
- Hybrid compression: <2ms for 1000 messages (74x faster than target!)
- LLM summarization: ~100-500ms (depends on LLM latency)
- Composite (recommended): LLM latency + hybrid fallback guarantee
- Zero blocking: runs asynchronously in background
- Thread-safe: all operations use mutex protection

**Observability:**
- Compression events emitted via EventEmitter
- Metrics: before/after message count, token count, compression ratio
- Events show in TUI as INFO messages

**Example:**
```go
history := core.NewHistory(16384, tokenizer)

// Optional: Set event emitter for compression notifications
history.SetEventEmitter(emitter)

// Add 200 turns of conversation
for i := 0; i < 200; i++ {
    history.AddUserMessage("Question " + i)
    history.AddAssistantMessage("Response " + i)
    history.AddToolMessage("tool_1", "Tool result")
}

// History automatically compressed when exceeding 80% capacity
// Final token count stays under budget
// All user messages preserved
// Compression events emitted to UI
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
- Compress: 89.3%

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
- **internal/session**: Persistent session state management
- **internal/core/task**: Task mode implementations
- **internal/core/turn**: Turn state management
- **internal/core/stream**: Event streaming

## License

See project LICENSE file.
