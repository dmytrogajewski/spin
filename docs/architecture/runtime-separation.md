# Runtime Separation Architecture

## Problem

The current agent construction flow in `cmd/spin/acp.go` violates separation of concerns:

```go
// ❌ WRONG: Agent created before runtime is complete
agent, emitter, storage, acpRuntime, cleanup, err := createACPAgent(...)
acpAgent, err := acppkg.NewSpinACPAgentWithStorage(agent, ...)
acpRuntime.SetACPAgent(acpAgent)  // Runtime depends on protocol adapter!
```

**Issues:**
1. Agent is built with incomplete runtime
2. Circular dependency: agent → runtime → acpAgent → agent
3. Agent knows about ACP protocol (should be agnostic)
4. Dependencies are wired AFTER construction

## Correct Architecture

### Layer Separation

```
┌─────────────────────────────────────┐
│   Protocol Layer (ACP, TUI, CLI)   │  ← Presentation/Transport
├─────────────────────────────────────┤
│         Core Agent Layer            │  ← Business Logic
├─────────────────────────────────────┤
│      Runtime Layer (ACP, Builtin)   │  ← Platform/Infrastructure
└─────────────────────────────────────┘
```

### Dependency Flow

```
Runtime (complete) 
  ↓ provides interfaces
Agent (core, runtime-agnostic)
  ↓ used by
Protocol Adapter (ACP, TUI, etc.)
```

### Construction Pattern

```go
// 1. Create runtime with ALL dependencies
runtime := runtime.NewACP(runtime.ACPConfig{
    WorkDir:      workDir,
    Emitter:      emitter,
    Storage:      storage,
    ShellService: shellService,
    GitService:   gitService,
    Logger:       logger,
})

// 2. Create core agent using runtime interfaces
agent := agent.NewBuilder().
    WithProvider(provider).
    WithRuntime(runtime).  // Runtime provides: tools, approval, executor
    Build()

// 3. Wrap in protocol adapter (presentation layer)
acpAgent := acp.NewAgent(agent, runtime)

// 4. Start protocol transport
conn := acp.NewConnection(acpAgent, stdout, stdin)
```

## Runtime Interface

The `Runtime` interface provides everything the agent needs:

```go
type Runtime interface {
    // Tool registration (runtime-specific tools)
    RegisterTools(registry *tools.Registry)
    
    // Notification handling (events → protocol)
    NotificationSender() NotificationSender
    
    // Approval handling (permissions)
    ApprovalHandler() security.ApprovalHandler
    
    // Session management
    SessionStorage() session.Storage
    SessionID() string
    
    // Terminal support (ACP only)
    SupportsTerminals() bool
    TerminalClient() TerminalClient
}
```

## Runtime Implementations

### ACP Runtime

Provides:
- **Tools**: ACP terminal tool (uses terminal protocol)
- **Approval**: ACP permission requests (via request_permission)
- **Executor**: Terminal executor (delegates to client terminals)
- **Notifications**: ACP session updates (protocol events)

### Builtin Runtime

Provides:
- **Tools**: Builtin shell command (local execution)
- **Approval**: TUI dialogs or auto-approve
- **Executor**: Local command execution
- **Notifications**: TUI events or log output

## Benefits

1. **Separation of Concerns**: Agent doesn't know about ACP/TUI
2. **No Circular Dependencies**: Linear dependency flow
3. **Testability**: Can mock runtime interface
4. **Flexibility**: Easy to add new runtimes (gRPC, HTTP, etc.)
5. **Clarity**: Each layer has clear responsibilities

## Migration Path

1. ✅ Define `Runtime` interface (done in internal/agent/runtime/runtime.go)
2. ✅ Implement `ACPRuntime` and `BuiltinRuntime` (done)
3. ⏳ Refactor cmd/spin/acp.go to use runtime-first pattern
4. ⏳ Refactor cmd/spin/tui.go to use runtime-first pattern
5. ⏳ Remove circular dependencies and late initialization
