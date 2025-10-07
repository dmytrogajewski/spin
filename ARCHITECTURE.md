# Spin Architecture

## Overview

Spin is built following clean architecture principles with clear separation of concerns, dependency injection, and interface-based design. The system is designed to be modular, testable, and maintainable.

## High-Level Architecture

```
┌──────────────────────────────────────────────────┐
│                User Interface                     │
│          Terminal UI (TUI) / CLI                  │
└──────────────────┬───────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────┐
│           Application Layer                       │
│     Manager • Conversation • Orchestration        │
└──────────────────┬───────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────┐
│             Domain Layer                          │
│   Agent • State • History • Events • Tools        │
└──────────────────┬───────────────────────────────┘
                   │
┌──────────────────▼───────────────────────────────┐
│          Infrastructure Layer                     │
│  LLM Providers • Storage • Security • MCP         │
└──────────────────────────────────────────────────┘
```

## Core Principles

### 1. Clean Architecture
- **Independence**: Business logic is independent of frameworks, UI, database, and external services
- **Testability**: Business rules can be tested without UI, database, or external services
- **Flexibility**: UI, database, and external services can be changed without affecting business logic

### 2. Dependency Injection
- All dependencies are injected through constructors or options
- Interfaces define contracts between layers
- Easy to mock for testing

### 3. Event-Driven Communication
- Loose coupling between components through events
- Real-time updates to UI through event streams
- Async operations don't block the main flow

### 4. Type Safety
- Extensive use of Go generics for compile-time type safety
- No runtime type assertions in critical paths
- Clear data contracts through typed structures

## Layer Breakdown

### User Interface Layer

**Components:**
- `cmd/spin`: CLI application
- `internal/tui`: Terminal user interface

**Responsibilities:**
- User interaction
- Command parsing
- Display formatting
- Event visualization

### Application Layer

**Components:**
- `internal/core/Manager`: High-level conversation management
- `internal/core/Conversation`: Active conversation handling

**Responsibilities:**
- Use case orchestration
- Session management
- Request/response coordination
- Business workflow

### Domain Layer

**Components:**
- `internal/core/Agent`: Core agent logic
- `internal/core/State`: State management
- `internal/core/History`: Conversation history
- `internal/core/Event`: Event system
- `internal/tools`: Tool abstractions

**Responsibilities:**
- Business rules
- Domain logic
- State transitions
- Event emission

### Infrastructure Layer

**Components:**
- `internal/llm/*`: LLM provider implementations
- `internal/session`: Persistent storage
- `internal/security`: Security implementations
- `internal/mcp`: MCP client

**Responsibilities:**
- External service integration
- Data persistence
- Security enforcement
- Protocol implementation

## Key Design Patterns

### 1. Repository Pattern
```go
type SessionRepository interface {
    Save(ctx context.Context, session *Session) error
    Load(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context) ([]*Session, error)
}
```

### 2. Factory Pattern
```go
type ProviderFactory interface {
    CreateProvider(config Config) (Provider, error)
}
```

### 3. Strategy Pattern
```go
type TaskStrategy interface {
    Execute(ctx context.Context, input string) (*Result, error)
}
```

### 4. Observer Pattern
```go
type EventEmitter interface {
    Emit(event Event)
    Subscribe(handler EventHandler) Subscription
}
```

### 5. Builder Pattern
```go
manager := NewManager(
    WithLLMProvider(provider),
    WithToolRegistry(registry),
    WithEventEmitter(emitter),
)
```

## Component Details

### Core Package (`internal/core`)

The core package is the heart of the application, containing:

```
internal/core/
├── agent.go           # Agent orchestration
├── manager.go         # High-level management
├── conversation.go    # Conversation handling
├── event.go          # Event types and emission
├── event_generic.go  # Generic event system
├── state.go          # State management
├── history.go        # History with truncation
├── executor.go       # Command execution
├── validator.go      # Command validation
├── stream/           # Streaming subsystem
├── task/            # Task strategies
└── turn/            # Turn management
```

### Security Architecture

```
internal/security/
├── policy/          # Command validation
│   ├── policy.go   # Policy interface
│   ├── rules.go    # Rule definitions
│   └── matcher.go  # Pattern matching
├── sandbox/         # Process isolation
│   ├── sandbox.go  # Sandbox interface
│   ├── landlock.go # Linux implementation
│   └── seatbelt.go # macOS implementation
└── hardening/       # Process hardening
    ├── hardening.go # Core hardening
    └── init.go      # Auto-initialization
```

### Provider Architecture

```
internal/llm/
├── provider.go      # Provider interface
├── types.go         # Common types
├── openai/         # OpenAI implementation
├── anthropic/      # Anthropic implementation
├── gemini/         # Google Gemini
├── ollama/         # Local models
└── lmstudio/       # LM Studio
```

## Data Flow

### Request Processing Flow

```
User Input
    ↓
TUI/CLI Parser
    ↓
Manager.SendMessage()
    ↓
Conversation.Process()
    ↓
Agent.Execute()
    ↓
LLM Provider.Complete()
    ↓
Tool Execution (if needed)
    ↓
Response Assembly
    ↓
Event Stream
    ↓
UI Update
```

### Event Flow

```
Domain Event Occurs
    ↓
Event Creation (Typed)
    ↓
EventEmitter.Emit()
    ↓
Channel Distribution
    ↓
Handler Processing
    ↓
UI Update / State Change
```

## Concurrency Model

### Goroutine Usage

1. **Main Goroutine**: UI and user interaction
2. **Agent Goroutine**: LLM interaction and orchestration
3. **Tool Goroutines**: Parallel tool execution
4. **Event Goroutine**: Event distribution
5. **Stream Goroutine**: Response streaming

### Synchronization

- **Channels**: Primary communication mechanism
- **Mutexes**: State protection
- **Context**: Cancellation and timeout
- **WaitGroups**: Parallel operation coordination

## Error Handling

### Error Types

```go
// Domain errors
type ValidationError struct {
    Field   string
    Message string
}

// Infrastructure errors
type ProviderError struct {
    Provider string
    Cause    error
}

// Application errors
type ConversationError struct {
    Turn  int
    Cause error
}
```

### Error Flow

1. Errors bubble up through layers
2. Each layer adds context
3. Top layer decides on user messaging
4. Critical errors trigger graceful shutdown

## Testing Strategy

### Unit Tests
- Test individual components in isolation
- Mock external dependencies
- Focus on business logic

### Integration Tests
- Test component interactions
- Use real implementations where possible
- Verify data flow

### End-to-End Tests
- Test complete user scenarios
- Include all layers
- Verify observable behavior

## Performance Considerations

### Optimization Points

1. **Token Management**: Efficient history truncation
2. **Streaming**: Chunked response processing
3. **Caching**: Response and tool result caching
4. **Pooling**: Connection and resource pooling
5. **Batching**: Batch tool calls when possible

### Benchmarks

Key operations are benchmarked:
- Turn creation: ~100ns
- Event emission: ~50ns
- Message processing: <100ms (excluding LLM)
- Tool execution: Varies by tool

## Scalability

### Horizontal Scaling
- Stateless design allows multiple instances
- Session storage enables cross-instance continuity
- Event system supports distributed processing

### Vertical Scaling
- Efficient memory usage through streaming
- Goroutine pools for parallel processing
- Resource limits prevent runaway consumption

## Security Layers

### Defense in Depth

1. **Input Validation**: All user input sanitized
2. **Command Classification**: Safety levels assigned
3. **Approval Flow**: User confirmation required
4. **Sandboxing**: Process isolation
5. **Hardening**: Runtime protections

### Trust Boundaries

```
Untrusted: User Input → Validation → Trusted: Internal
Untrusted: LLM Output → Validation → Trusted: Execution
Untrusted: Tool Results → Sanitization → Trusted: Display
```

## Extension Points

### Adding Providers

1. Implement `Provider` interface
2. Register in factory
3. Add configuration

### Adding Tools

1. Implement `Tool` interface
2. Register in registry
3. Define schema

### Adding Task Strategies

1. Implement `TaskStrategy` interface
2. Register in task registry
3. Configure selection

## Configuration

### Hierarchical Configuration

```
1. Default values (code)
2. Configuration file (~/.spin/config.yaml)
3. Environment variables (SPIN_*)
4. Command-line flags
```

### Configuration Scopes

- **Global**: Application-wide settings
- **Provider**: Provider-specific settings
- **Session**: Per-conversation settings
- **Security**: Security policies

## Monitoring & Observability

### Logging
- Structured logging with slog
- Configurable levels
- JSON output for parsing

### Tracing
- OpenTelemetry integration
- Distributed tracing support
- Performance profiling

### Metrics
- Operation counters
- Latency histograms
- Resource usage

## Future Architecture Goals

### Short Term
- [ ] Plugin system for tools
- [ ] Enhanced caching layer
- [ ] Improved streaming performance

### Medium Term
- [ ] Distributed execution
- [ ] Multi-agent coordination
- [ ] Advanced security policies

### Long Term
- [ ] Cloud-native deployment
- [ ] Federated learning
- [ ] Self-improvement capabilities

## Conclusion

Spin's architecture is designed to be:
- **Maintainable**: Clear separation of concerns
- **Testable**: Interface-based design
- **Extensible**: Plugin points throughout
- **Secure**: Multiple security layers
- **Performant**: Optimized critical paths

The architecture continues to evolve while maintaining backward compatibility and clean design principles.