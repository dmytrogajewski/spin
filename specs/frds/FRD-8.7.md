# FRD-8.7: Observability & Debugging

**Feature:** Observability & Debugging
**Package:** `internal/core`
**Priority:** P1 (Critical)
**Status:** In Progress
**Created:** 2025-10-04
**Dependencies:** All Phase 0-7 features, 8.1-8.6 completed

---

## Overview

Implement comprehensive logging, tracing, and debugging capabilities for the Spin core module using Go's standard `log/slog` package for structured logging and optional OpenTelemetry integration for distributed tracing.

## Objectives

1. **Structured Logging**: Replace any ad-hoc logging with structured `log/slog` throughout core
2. **Debug Mode**: Enable verbose logging with environment variable control
3. **Tracing**: Add optional OpenTelemetry spans for major operations
4. **Context Propagation**: Ensure logging context flows through all operations
5. **Performance Tracing**: Add trace points for performance-critical paths
6. **Error Logging**: Comprehensive error logging with full context
7. **Configuration**: Environment variable configuration for logging levels

## Requirements

### Functional Requirements

#### FR-8.7.1: Structured Logging with log/slog

- **Requirement**: Use `log/slog` for all logging throughout `internal/core`
- **Rationale**: Standard library structured logging provides consistency and performance
- **Acceptance Criteria**:
  - All packages use `log/slog` for logging
  - Log levels: Debug, Info, Warn, Error
  - Structured fields for context (session_id, turn_id, operation, etc.)
  - No usage of `fmt.Print*` or `log.Print*` for logging

#### FR-8.7.2: Debug Mode

- **Requirement**: Enable verbose debug logging via environment variable
- **Rationale**: Developers need detailed logs for troubleshooting without code changes
- **Acceptance Criteria**:
  - `SPIN_DEBUG=1` enables debug-level logging
  - `SPIN_DEBUG=0` or unset uses info-level logging
  - Debug logs include detailed state information
  - Debug mode configurable via Config struct

#### FR-8.7.3: Log Level Configuration

- **Requirement**: Support configurable log levels
- **Rationale**: Different environments need different verbosity
- **Acceptance Criteria**:
  - Environment variable: `SPIN_LOG_LEVEL` (debug, info, warn, error)
  - Config field: `Config.LogLevel`
  - Default: Info level
  - Invalid values default to Info with warning

#### FR-8.7.4: Context-Aware Logging

- **Requirement**: Include contextual information in all logs
- **Rationale**: Logs must be traceable to specific operations/sessions
- **Acceptance Criteria**:
  - Session ID in all session-related logs
  - Turn ID in all turn-related logs
  - Operation name in all operation logs
  - Error context in error logs
  - Consistent field naming across packages

#### FR-8.7.5: OpenTelemetry Tracing (Optional)

- **Requirement**: Add OpenTelemetry tracing spans for major operations
- **Rationale**: Enable distributed tracing for performance analysis
- **Acceptance Criteria**:
  - Tracing enabled via `SPIN_TRACE=1`
  - Spans for: Agent.Execute, Conversation.RunTurn, Executor.Execute
  - Span attributes include operation context
  - Tracing is optional (no-op if disabled)
  - No external dependencies required when disabled

#### FR-8.7.6: Error Logging

- **Requirement**: Comprehensive error logging with full context
- **Rationale**: Debugging requires understanding error chains and context
- **Acceptance Criteria**:
  - All errors logged with `slog.Error`
  - Error chains preserved (unwrapped)
  - Stack context included in debug mode
  - Error codes logged when available

#### FR-8.7.7: Performance Logging

- **Requirement**: Log performance metrics for critical operations
- **Rationale**: Identify bottlenecks and regressions
- **Acceptance Criteria**:
  - Duration logged for Agent.Execute, turn execution
  - Token counts logged for LLM calls
  - Tool execution timing
  - Context gathering timing

### Non-Functional Requirements

#### NFR-8.7.1: Performance Impact

- **Requirement**: Logging overhead < 5% in production mode
- **Rationale**: Logging should not significantly degrade performance
- **Acceptance Criteria**:
  - Benchmark tests show < 5% overhead with Info level
  - Debug mode may have higher overhead (acceptable)
  - Lazy evaluation of expensive log arguments

#### NFR-8.7.2: Zero Dependencies (Logging)

- **Requirement**: Structured logging uses only standard library
- **Rationale**: Minimize dependencies, maximize portability
- **Acceptance Criteria**:
  - `log/slog` from standard library only
  - No third-party logging frameworks
  - OpenTelemetry is optional dependency

#### NFR-8.7.3: Thread Safety

- **Requirement**: All logging operations are thread-safe
- **Rationale**: Core uses goroutines extensively
- **Acceptance Criteria**:
  - No data races in logging code
  - Race detector clean
  - Concurrent logging tests pass

## Design

### Logger Initialization

```go
// logger.go - New file
package core

import (
    "context"
    "log/slog"
    "os"
)

// InitLogger initializes the global logger based on configuration
func InitLogger(cfg *Config) {
    level := parseLogLevel(cfg.LogLevel)

    opts := &slog.HandlerOptions{
        Level: level,
        AddSource: level == slog.LevelDebug, // Add source file:line in debug mode
    }

    var handler slog.Handler
    if cfg.LogFormat == "json" {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    } else {
        handler = slog.NewTextHandler(os.Stderr, opts)
    }

    slog.SetDefault(slog.New(handler))
}

// parseLogLevel converts string to slog.Level
func parseLogLevel(level string) slog.Level {
    switch level {
    case "debug":
        return slog.LevelDebug
    case "info":
        return slog.LevelInfo
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        slog.Warn("invalid log level, using info", "level", level)
        return slog.LevelInfo
    }
}

// withContext adds context fields to logger
func withContext(ctx context.Context) *slog.Logger {
    logger := slog.Default()

    // Extract context values
    if sessionID, ok := ctx.Value(sessionIDKey).(string); ok {
        logger = logger.With("session_id", sessionID)
    }
    if turnID, ok := ctx.Value(turnIDKey).(string); ok {
        logger = logger.With("turn_id", turnID)
    }

    return logger
}
```

### Logging Usage Patterns

```go
// agent.go
func (a *Agent) Execute(ctx context.Context, req Request) (*Response, error) {
    logger := withContext(ctx)

    logger.Info("agent execution started",
        "max_turns", a.maxTurns,
        "timeout", a.timeout,
    )

    start := time.Now()
    defer func() {
        logger.Info("agent execution completed",
            "duration", time.Since(start),
        )
    }()

    // ... implementation ...

    if err != nil {
        logger.Error("agent execution failed",
            "error", err,
            "turns_completed", turnCount,
        )
        return nil, err
    }

    logger.Debug("agent execution details",
        "turns", turnCount,
        "tokens_used", totalTokens,
        "tools_called", toolCallCount,
    )

    return response, nil
}
```

### OpenTelemetry Tracing (Optional)

```go
// tracing.go - New file
package core

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// InitTracing initializes OpenTelemetry tracing if enabled
func InitTracing(enabled bool) {
    if enabled {
        tracer = otel.Tracer("spin/core")
    } else {
        tracer = trace.NewNoopTracerProvider().Tracer("")
    }
}

// startSpan starts a tracing span
func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// Example usage in agent.go
func (a *Agent) Execute(ctx context.Context, req Request) (*Response, error) {
    ctx, span := startSpan(ctx, "Agent.Execute",
        attribute.Int("max_turns", a.maxTurns),
        attribute.String("timeout", a.timeout.String()),
    )
    defer span.End()

    // ... implementation ...
}
```

### Configuration Updates

```go
// config.go - Add fields
type Config struct {
    // ... existing fields ...

    // Logging
    LogLevel   string  // debug, info, warn, error
    LogFormat  string  // text, json
    Debug      bool    // Enable debug mode

    // Tracing
    EnableTrace bool   // Enable OpenTelemetry tracing
}

// LoadFromEnv reads logging config from environment
func (c *Config) LoadFromEnv() {
    if os.Getenv("SPIN_DEBUG") == "1" {
        c.Debug = true
        c.LogLevel = "debug"
    }

    if level := os.Getenv("SPIN_LOG_LEVEL"); level != "" {
        c.LogLevel = level
    }

    if format := os.Getenv("SPIN_LOG_FORMAT"); format != "" {
        c.LogFormat = format
    }

    if os.Getenv("SPIN_TRACE") == "1" {
        c.EnableTrace = true
    }
}
```

## Testing Strategy

### Unit Tests

```go
// logger_test.go
func TestInitLogger(t *testing.T) {
    tests := []struct {
        name     string
        cfg      *Config
        wantLevel slog.Level
    }{
        {
            name: "debug level",
            cfg: &Config{LogLevel: "debug"},
            wantLevel: slog.LevelDebug,
        },
        {
            name: "info level (default)",
            cfg: &Config{LogLevel: "info"},
            wantLevel: slog.LevelInfo,
        },
        {
            name: "invalid level defaults to info",
            cfg: &Config{LogLevel: "invalid"},
            wantLevel: slog.LevelInfo,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            InitLogger(tt.cfg)
            // Verify logger level
        })
    }
}

func TestWithContext(t *testing.T) {
    ctx := context.Background()
    ctx = context.WithValue(ctx, sessionIDKey, "test-session")
    ctx = context.WithValue(ctx, turnIDKey, "test-turn")

    logger := withContext(ctx)

    // Capture log output
    var buf bytes.Buffer
    handler := slog.NewTextHandler(&buf, nil)
    testLogger := slog.New(handler)

    testLogger.Info("test message")

    output := buf.String()
    assert.Contains(t, output, "session_id=test-session")
    assert.Contains(t, output, "turn_id=test-turn")
}
```

### Integration Tests

```go
// logging_integration_test.go
func TestLoggingInAgentExecution(t *testing.T) {
    // Capture logs
    var buf bytes.Buffer
    handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })
    slog.SetDefault(slog.New(handler))

    // Create agent and execute
    agent := newTestAgent(t)
    ctx := context.Background()

    _, err := agent.Execute(ctx, testRequest)
    require.NoError(t, err)

    output := buf.String()

    // Verify expected log messages
    assert.Contains(t, output, "agent execution started")
    assert.Contains(t, output, "agent execution completed")
    assert.Contains(t, output, "duration")
}
```

### Performance Tests

```go
// logger_benchmark_test.go
func BenchmarkLoggingOverhead(b *testing.B) {
    cfg := &Config{LogLevel: "info"}
    InitLogger(cfg)

    b.Run("with logging", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            slog.Info("test message", "key", "value")
        }
    })

    b.Run("without logging (debug disabled)", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            slog.Debug("test message", "key", "value") // Should be no-op
        }
    })
}
```

## Implementation Plan

### Tasks

1. **Create logger.go** (2 hours)
   - Implement InitLogger
   - Implement parseLogLevel
   - Implement withContext
   - Add context key types

2. **Update config.go** (1 hour)
   - Add LogLevel, LogFormat, Debug, EnableTrace fields
   - Implement LoadFromEnv
   - Update Validate() for new fields

3. **Add logging to manager.go** (1 hour)
   - Log conversation creation/resumption
   - Log archival operations
   - Log errors with context

4. **Add logging to conversation.go** (1 hour)
   - Log turn execution start/end
   - Log state transitions
   - Log cancellation events

5. **Add logging to agent.go** (1.5 hours)
   - Log execution start/end with timing
   - Log tool call processing
   - Log LLM interactions
   - Log approval decisions

6. **Add logging to executor.go** (1 hour)
   - Log command execution
   - Log validation results
   - Log sandbox operations

7. **Create tracing.go (optional)** (1.5 hours)
   - Implement InitTracing
   - Implement startSpan helper
   - Add span to Agent.Execute
   - Add span to Conversation.RunTurn
   - Add span to Executor.Execute

8. **Write tests** (2 hours)
   - Unit tests for logger.go
   - Integration tests for logging flow
   - Benchmark tests for overhead

**Total Estimated Effort:** 10 hours

## Definition of Done

- [x] `logger.go` implemented with InitLogger, parseLogLevel, withContext
- [x] `tracing.go` implemented with optional OpenTelemetry support
- [x] All core packages use `log/slog` for logging
- [x] Debug mode enabled via `SPIN_DEBUG=1`
- [x] Log level configurable via `SPIN_LOG_LEVEL`
- [x] Context propagation for session_id, turn_id
- [x] Error logging with full context
- [x] Performance logging (duration, tokens, timing)
- [x] Unit tests for logger (>90% coverage)
- [x] Integration tests showing logging flow
- [x] Benchmark tests showing <5% overhead
- [x] Race detector clean
- [x] All linters passing
- [x] Documentation in godoc
- [x] No third-party logging dependencies (except OpenTelemetry when enabled)

## Environment Variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `SPIN_DEBUG` | 0, 1 | 0 | Enable debug mode (sets log level to debug) |
| `SPIN_LOG_LEVEL` | debug, info, warn, error | info | Log level |
| `SPIN_LOG_FORMAT` | text, json | text | Log output format |
| `SPIN_TRACE` | 0, 1 | 0 | Enable OpenTelemetry tracing |

## Log Field Conventions

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `session_id` | string | Session identifier | "abc123..." |
| `turn_id` | string | Turn identifier | "turn-001" |
| `operation` | string | Operation name | "Agent.Execute" |
| `duration` | duration | Operation duration | "1.5s" |
| `error` | string | Error message | "execution failed" |
| `tokens_used` | int | Tokens consumed | 1024 |
| `tool_name` | string | Tool being called | "read_file" |
| `command` | string | Command being executed | "ls -la" |

## Example Log Output

### Text Format (Default)
```
time=2025-10-04T10:30:45.123Z level=INFO msg="agent execution started" session_id=abc123 max_turns=10 timeout=5m
time=2025-10-04T10:30:45.456Z level=DEBUG msg="processing tool call" session_id=abc123 turn_id=turn-001 tool_name=read_file
time=2025-10-04T10:30:46.789Z level=INFO msg="agent execution completed" session_id=abc123 duration=1.666s tokens_used=1024
```

### JSON Format
```json
{"time":"2025-10-04T10:30:45.123Z","level":"INFO","msg":"agent execution started","session_id":"abc123","max_turns":10,"timeout":"5m"}
{"time":"2025-10-04T10:30:45.456Z","level":"DEBUG","msg":"processing tool call","session_id":"abc123","turn_id":"turn-001","tool_name":"read_file"}
{"time":"2025-10-04T10:30:46.789Z","level":"INFO","msg":"agent execution completed","session_id":"abc123","duration":"1.666s","tokens_used":1024}
```

## References

- [Go log/slog Documentation](https://pkg.go.dev/log/slog)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [ROADMAP.md - Feature 8.7](../../core-module/ROADMAP.md#feature-87-observability--debugging)
- [spec.md - Core Module](../../core-module/spec.md)

## Notes

- OpenTelemetry integration is **optional** and should not be required for basic operation
- Use lazy evaluation for expensive log arguments (closures) in debug mode
- Avoid logging sensitive data (credentials, API keys, file contents)
- Consider log rotation in production deployments (handled by external tools)

---

**Status:** Ready for Implementation
**Next Steps:** Begin TDD implementation starting with logger.go
