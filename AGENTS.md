# Agent Guidelines for Spin Development

**📋 Implementation Workflow:** See [instructions/istr-implement.md](instructions/istr-implement.md)  
**📚 Package Documentation:** See [docs/packages/](docs/packages/)

---

## Core Principles

1. **Vendor-Agnostic**: Compatible with Ollama, LMStudio, OpenAI, Anthropic. **Never write vendor-locked code.**
2. **TDD**: Write tests first. Coverage: ≥90% critical paths, ≥85% overall
3. **SOLID**: Single Responsibility, Open/Closed, Liskov, Interface Segregation, Dependency Inversion
4. **Clean Code**: DRY, KISS, Clean Architecture
5. **Go 1.24+**: Follow [Effective Go](https://go.dev/doc/effective_go)

---

## Implementation Workflow (14 Steps)

**⚠️ Follow ALL steps. See [istr-implement.md](instructions/istr-implement.md) for details.**

1. Read technical document (specs/)
2. Take first ROADMAP item
3. **Read ALL docs in docs/** (!!!)
4. Write FRD (specs/frds/FRD-{id}.md)
5. Read your FRD
6. Write tests (TDD)
7. Write implementation
8. Analyze: `uast parse {file} | herr analyze`
9. Lint: `make lint` (zero errors)
10. Fix all issues (no dead code)
11. Run tests: `go test -race ./...`
12. Mark ROADMAP item complete
13. Update docs/
14. Update AGENTS.md if needed

---

## Quality Gates

**Tests:**
- ✅ All passing
- ✅ Coverage ≥85% (≥90% critical)
- ✅ Race detector clean

**Code:**
- ✅ `make lint` passes (zero errors)
- ✅ Complexity ≤15
- ✅ No dead code
- ✅ Godoc on all exports

---

## Code Patterns

### Error Handling
```go
// ✅ GOOD: Structured errors with context
type Error struct {
    Op      string
    Err     error
    Code    ErrCode
    Context map[string]interface{}
}

// ❌ BAD: Simple error strings
return fmt.Errorf("error: %s", msg)
```

### Interfaces
```go
// ✅ GOOD: Small, focused
type Provider interface {
    Complete(ctx context.Context, req Request) (*Response, error)
    Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}

// ❌ BAD: Large, bloated (>5 methods)
```

### Configuration
```go
// ✅ GOOD: Functional options
func NewManager(cfg *Config, opts ...Option) (*Manager, error)

type Option func(*Manager)
func WithProvider(p Provider) Option { ... }

// Usage
mgr, _ := NewManager(cfg, WithProvider(p), WithLogger(l))
```

### Testing
```go
// ✅ GOOD: Table-driven
func TestComplete(t *testing.T) {
    tests := []struct {
        name    string
        input   Request
        want    *Response
        wantErr bool
    }{
        {"success", Request{...}, &Response{...}, false},
        {"error", Request{...}, nil, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

### Context
```go
// ✅ GOOD: Always accept context first
func (p *Provider) Complete(ctx context.Context, req Request) (*Response, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ...
}
```

### Concurrency
```go
// ✅ GOOD: Channels for communication
func Stream(ctx context.Context) (<-chan Chunk, error) {
    chunks := make(chan Chunk, 100)
    go func() {
        defer close(chunks)
        // ...
    }()
    return chunks, nil
}

// ✅ GOOD: Protect shared state
type Manager struct {
    mu    sync.RWMutex
    state State
}
func (m *Manager) GetState() State {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.state
}
```

---

## Spin Patterns

### Provider Pattern (LLM)
```go
// internal/llm/provider.go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
// Implementations: openai, ollama, lmstudio
```

### Factory Pattern
```go
// internal/llm/factory/factory.go
f := factory.NewFactory(authManager)
provider, _ := f.NewProvider(ctx, factory.ProviderConfig{
    Type:    "openai",
    KeyName: "openai-api-key", // From keystore
})
```

### Event Pattern
```go
// internal/core/event.go
for event := range events {
    switch event.Type {
    case EventTypeStreamContent:
        // Handle
    }
}
```

### State Machine
```go
// internal/core/turn/turn.go
type State string
const (
    StatePending   State = "pending"
    StateRunning   State = "running"
    StateCompleted State = "completed"
)
```

---

## Package Structure

**See [docs/packages/](docs/packages/) for full documentation.**

**Core:**
- [core](docs/packages/core.md) - Business logic, agent orchestration
- [llm](docs/packages/llm.md) - Vendor-agnostic LLM providers
- [auth](docs/packages/auth.md) - Platform keystore integration

**Infrastructure:**
- [protocol](docs/packages/protocol.md) - JSON-RPC 2.0
- [appserver](docs/packages/appserver.md) - WebSocket server
- [config](docs/packages/config.md) - Multi-format config
- [tools](docs/packages/tools.md) - Tool registry
- [mcp](docs/packages/mcp.md) - Model Context Protocol

---

## Testing

```bash
go test ./...                # All tests
go test -race ./...          # Race detection
go test -cover ./...         # Coverage
go test -bench=. ./...       # Benchmarks
```

**Coverage:**
- Critical paths: ≥90%
- Overall: ≥85%
- New code: ≥90%

---

## Commands

```bash
# Quality
make lint              # Linter (must pass)
make test              # All tests
make coverage          # Coverage report

# Analysis
uast parse {file} | herr analyze   # Complexity
gocyclo -over 15 ./...              # Cyclomatic complexity

# Testing
go test ./...          # All
go test -race ./...    # Race
go test -v ./...       # Verbose
```

---

## Checklist

### Before Commit
- [ ] All 14 workflow steps done
- [ ] **ALL docs/ read**
- [ ] FRD created and complete
- [ ] Tests pass (with `-race`)
- [ ] Coverage ≥85%
- [ ] Linter clean (zero errors)
- [ ] No dead code
- [ ] Complexity ≤15
- [ ] Godoc complete
- [ ] ROADMAP updated

### Quality
- [ ] SOLID principles
- [ ] Vendor-agnostic
- [ ] Context support
- [ ] Error handling
- [ ] Thread-safe

---

## Troubleshooting

**Tests fail:** `go test -v ./...` with `-race`  
**Linter errors:** `make lint` - fix all  
**High complexity:** `uast parse | herr analyze` - refactor  
**Low coverage:** Add edge cases and error paths

---

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Architecture Overview](specs/architecture-overview.md)
- [Implementation Workflow](instructions/istr-implement.md)

---

**Remember:**
- Quality over speed
- Follow ALL 14 steps
- Read docs/ first
- No vendor lock-in
- TDD always
