# Spin Architecture Documentation

## Executive Summary

**Spin** is an AI-powered coding assistant built in Go that provides multiple execution modes (TUI, batch execution, ACP server) with multi-provider LLM support, security sandboxing, and filesystem/shell integration. The architecture follows a service-oriented, event-driven design with layered components that enable both local and protocol-based execution.

---

## Table of Contents

1. [High-Level Architecture](#high-level-architecture)
2. [Directory Structure](#directory-structure)
3. [Entry Points and Execution Modes](#entry-points-and-execution-modes)
4. [Core Package Architecture](#core-package-architecture)
5. [Key Interfaces and Abstractions](#key-interfaces-and-abstractions)
6. [Data Flow](#data-flow)
7. [Design Patterns](#design-patterns)
8. [Architectural Decisions](#architectural-decisions)
9. [Configuration System](#configuration-system)
10. [External Dependencies](#external-dependencies)

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Application Layer                               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐   │
│  │   TUI   │  │  Exec   │  │   ACP   │  │  Debug  │  │   Config    │   │
│  │  Mode   │  │  Mode   │  │  Server │  │  Utils  │  │  Commands   │   │
│  └────┬────┘  └────┬────┘  └────┬────┘  └─────────┘  └─────────────┘   │
└───────┼────────────┼────────────┼──────────────────────────────────────┘
        │            │            │
        ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Conversation Layer                                │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    Conversation Manager                          │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │    │
│  │  │ Conversation │  │   History    │  │  Event Transformer   │  │    │
│  │  └──────┬───────┘  └──────────────┘  └──────────────────────┘  │    │
│  └─────────┼────────────────────────────────────────────────────────┘    │
└────────────┼────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Agent Layer                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                          Agent                                    │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │   │
│  │  │ ToolRuntime  │  │ ACEService   │  │   PlanTracker        │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Service Layer                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │   Security   │  │  Detection   │  │   Planning   │  │    Task    │  │
│  │   Service    │  │   Service    │  │   Service    │  │   Modes    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Infrastructure Layer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │   LLM    │  │   Git    │  │  Shell   │  │   MCP    │  │ Session  │ │
│  │ Provider │  │ Service  │  │ Service  │  │ Service  │  │ Storage  │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         External Systems                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │  OpenAI  │  │ Anthropic│  │  Ollama  │  │ LMStudio │  │Filesystem│ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
spin/
├── cmd/spin/                    # Application entry points
│   ├── main.go                  # Main entry point with special binary handling
│   ├── root.go                  # Root Cobra command with global flags
│   ├── tui.go                   # Interactive TUI mode
│   ├── exec.go                  # Non-interactive batch execution
│   ├── acp.go                   # ACP (Agent Client Protocol) server
│   ├── services.go              # Service creation and lifecycle
│   └── ...                      # Other CLI subcommands
│
├── internal/                    # Core internal packages
│   ├── agent/                   # Core agent orchestration (35KB main file)
│   │   ├── agent.go             # Main agent loop and LLM interaction
│   │   ├── builder.go           # Fluent builder for agent construction
│   │   ├── executor.go          # Command execution wrapper
│   │   ├── environment.go       # Context gathering (project structure, git)
│   │   ├── tool_runtime.go      # Tool registry and execution routing
│   │   ├── ace_service.go       # ACE (Agentic Context Engineering) integration
│   │   ├── loop.go              # Agent execution loop
│   │   ├── runtime/             # Runtime interface definitions
│   │   └── sanitizer/           # Content filtering and thought extraction
│   │
│   ├── config/                  # Configuration system
│   │   ├── config_v2.go         # Unified V2 configuration structures
│   │   ├── loader_v2.go         # Config loading (files, env, flags)
│   │   └── mcp_manager.go       # MCP server configuration
│   │
│   ├── conversation/            # High-level conversation management
│   │   ├── conversation.go      # Conversation handler wrapping Agent
│   │   ├── builder.go           # Conversation builder
│   │   ├── manager.go           # Multi-session conversation manager
│   │   └── history.go           # History wrapper with conversation context
│   │
│   ├── session/                 # Session state persistence
│   │   ├── session.go           # Session metadata and state machine
│   │   ├── storage.go           # Storage interface
│   │   └── file_storage.go      # JSON-based file storage
│   │
│   ├── history/                 # Message history management
│   │   ├── history.go           # Token-aware message storage
│   │   └── storage.go           # Persistent history storage
│   │
│   ├── events/                  # Real-time event streaming
│   │   ├── events.go            # Event type definitions
│   │   └── emitter.go           # Non-blocking event emitter
│   │
│   ├── security/                # Security and approval system
│   │   ├── approval.go          # Approval service and workflow
│   │   ├── validator.go         # Command classification
│   │   ├── service.go           # SecurityService facade
│   │   └── policy.go            # Approval policy persistence
│   │
│   ├── tools/                   # Tool system
│   │   ├── tool.go              # Tool interface and registry
│   │   ├── read_file.go         # File reading tool
│   │   ├── write_file.go        # File writing tool
│   │   ├── shell_command.go     # Command execution tool
│   │   ├── apply_patch.go       # Diff patch application
│   │   └── ...                  # Other built-in tools
│   │
│   ├── llm/                     # LLM provider abstraction
│   │   ├── provider.go          # Provider interface
│   │   ├── openai/              # OpenAI provider
│   │   ├── anthropic/           # Anthropic/Claude provider
│   │   ├── ollama/              # Ollama (local) provider
│   │   └── lmstudio/            # LMStudio (local) provider
│   │
│   ├── task/                    # Task mode system
│   │   ├── task.go              # Task interface and factory
│   │   ├── regular.go           # Full-featured mode (8K tokens)
│   │   ├── review.go            # Code review mode (12K, read-only)
│   │   ├── compact.go           # Quick tasks mode (4K, limited tools)
│   │   └── planning.go          # Planning-only mode (4K, no tools)
│   │
│   ├── ace/                     # Agentic Context Engineering
│   │   ├── bullet/              # Knowledge unit representation
│   │   ├── playbook/            # Bullet collection management
│   │   ├── retrieval/           # Semantic similarity retrieval
│   │   ├── generator/           # Bullet generation from execution
│   │   ├── reflector/           # Trajectory reflection
│   │   ├── curator/             # Quality control and deduplication
│   │   ├── adapter/             # Online learning orchestration
│   │   ├── delta/               # Change tracking
│   │   ├── refine/              # Playbook growth management
│   │   └── embedding/           # Vector embedding generation
│   │
│   ├── protocol/acp/            # ACP protocol implementation
│   │   ├── agent.go             # SpinACPAgent adapter (46KB)
│   │   ├── approval_handler.go  # ACP approval requests
│   │   ├── terminal_client.go   # Terminal operations
│   │   └── filesystem_client.go # File operations via ACP
│   │
│   ├── ui/                      # Terminal UI framework
│   │   ├── ports/               # UI interface definitions
│   │   ├── adapters/            # Concrete implementations
│   │   ├── blocks/              # Display components
│   │   ├── theme/               # Styling configuration
│   │   ├── prompt/              # Input handling
│   │   └── overlay/             # Dialogs and overlays
│   │
│   ├── git/                     # Git repository operations
│   ├── shell/                   # Shell command execution
│   ├── mcp/                     # Model Context Protocol
│   ├── planning/                # Task decomposition
│   ├── detection/               # Cycle detection
│   ├── message/                 # Message structure definitions
│   └── ...                      # Supporting packages
│
├── api/                         # API definitions
│   ├── jsonrpc/                 # JSON-RPC types
│   └── mcp/                     # MCP protocol types
│
├── build/                       # Build automation
├── configs/                     # Configuration templates
├── docs/                        # Documentation
├── examples/                    # Example implementations
├── instructions/                # LLM prompt instructions
├── specs/                       # Feature specifications
└── tests/                       # E2E and compliance tests
```

---

## Entry Points and Execution Modes

### Main Entry Point (`cmd/spin/main.go`)

The main entry point handles three scenarios:

1. **Special Binary Names**: Supports symlinked binaries for specific functions
   - `spin-apply-patch`: Standalone patch application utility
   - `spin-sandbox`: Sandbox testing mode

2. **Internal Flags**: Subprocess execution flags
   - `--spin-run-as-apply-patch`: Run in patch application mode
   - `--spin-run-as-sandbox`: Run in sandbox mode

3. **Normal CLI Execution**: Routes to Cobra command tree

```go
func main() {
    // Check for special binary names (symlinks)
    if exitCode := handleSpecialBinaryName(); exitCode >= 0 {
        os.Exit(exitCode)
    }

    // Check for internal flags (subprocess execution)
    if exitCode := handleInternalFlags(); exitCode >= 0 {
        os.Exit(exitCode)
    }

    // Normal CLI execution
    if err := execute(); err != nil {
        os.Exit(1)
    }
}
```

**Design Decision**: This pattern allows Spin to be deployed as multiple binaries from a single codebase, enabling specialized tools that share the core implementation.

### Execution Modes

| Mode | Command | Purpose | Key Features |
|------|---------|---------|--------------|
| **TUI** | `spin tui` | Interactive terminal UI | Real-time streaming, native scrollback, rich display |
| **Exec** | `spin exec` | Non-interactive batch execution | Scriptable, CI/CD integration |
| **ACP** | `spin acp` | Agent Client Protocol server | IDE integration, remote control |
| **Config** | `spin config` | Configuration management | View/edit settings |
| **Auth** | `spin auth` | Authentication management | API key storage |
| **MCP** | `spin mcp` | MCP server management | Start/stop/list servers |

### Root Command Structure (`cmd/spin/root.go`)

```go
// Global flags available to all subcommands
--model        // LLM model to use
--provider     // LLM provider (openai, anthropic, ollama, lmstudio)
--sandbox      // Sandbox mode (none, workspace-only, docker, firejail)
--cd           // Working directory
--config-file  // Custom config file path
--mode         // Task mode (regular, review, compact, planning)
```

**Design Decision**: Global flags are processed before subcommand execution, allowing consistent behavior across all modes while enabling mode-specific overrides.

---

## Core Package Architecture

### Agent Package (`internal/agent/`)

The Agent is the central orchestration engine that manages the LLM interaction loop.

#### Agent Structure

```go
type Agent struct {
    // Core LLM interaction
    llm llm.Provider

    // Service layers (dependency injection)
    security        *security.SecurityService
    detection       *detection.DetectionService
    toolRuntime     *ToolRuntime
    planningService *planning.PlanningService
    aceService      *ACEService  // Optional - persistent learning

    // Infrastructure
    context     *Environment
    emitter     *events.EventEmitter
    planner     *planning.Plan
    planTracker *PlanTracker

    // Configuration
    maxTurns        int
    timeout         time.Duration
    temperature     float64
    maxTokens       int
    requireApproval bool
}
```

**Design Decision**: The Agent uses service-based architecture rather than embedding functionality directly. This enables:
- Independent testing of each service
- Runtime service swapping (e.g., different approval handlers for TUI vs ACP)
- Clear separation of concerns
- Easier extension without modifying core agent

#### Agent Execution Flow

```go
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    // 1. Setup and validation
    ctx, resp, err := a.executeSetup(ctx, req)
    
    // 2. Resolve task mode
    task, err := a.resolveTask(req)
    
    // 3. Apply timeout
    ctx, cancel := a.applyTimeout(ctx)
    defer cancel()
    
    // 4. Build initial prompt
    messages := a.buildPrompt(req)
    
    // 5. Execute agent loop (may involve multiple LLM calls + tool executions)
    messages, resp, err = a.executeAgentLoop(ctx, messages, task, resp, trajCtx)
    
    // 6. ACE: Learn from execution (if enabled)
    if a.aceService != nil {
        learnedBullets, _ := a.aceService.GenerateBulletsWithReflectionFromTrajectory(ctx, trajectory)
    }
    
    // 7. Finalize and return
    a.finalizeResponse(resp, messages, historyLen)
    return resp, nil
}
```

### Conversation Package (`internal/conversation/`)

The Conversation package provides a higher-level abstraction over the Agent.

```go
type Conversation struct {
    // Services (optional, can be nil)
    gitService   *gitpkg.Service
    shellService *shellpkg.Service
    mcpService   *mcppkg.Service

    // Core components
    agent    *agent.Agent
    history  *history.History
    emitter  *events.EventEmitter
    taskMode string
    id       string
    workDir  string

    // Protocol-specific fields
    turnID      string
    cancel      context.CancelFunc
    transformer EventTransformer
}
```

**Design Decision**: Conversation wraps Agent to provide:
- Session management (ID, state, persistence)
- History management (token counting, truncation)
- Task mode switching
- Protocol adaptation (ACP event transformation)

This separation allows the Agent to remain focused on LLM interaction while Conversation handles session lifecycle.

### Security Package (`internal/security/`)

The security package implements a multi-layer command validation and approval system.

#### Command Classification

```go
type CommandClassification int

const (
    CommandSafe      CommandClassification = iota  // Safe to execute
    CommandNeutral                                  // Neutral (context-dependent)
    CommandDangerous                                // Requires approval
    CommandForbidden                                // Never execute
)
```

#### Approval Workflow

```
Command Submitted
       │
       ▼
┌──────────────────┐
│    Validator     │ ──► Pattern-based classification
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Policy Check    │ ──► Check session/global policies
└────────┬─────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 Cached    Not Cached
    │         │
    │         ▼
    │  ┌──────────────────┐
    │  │ Approval Handler │ ──► TUI dialog or ACP request
    │  └────────┬─────────┘
    │           │
    │           ▼
    │    User Decision
    │           │
    │           ▼
    │  ┌──────────────────┐
    │  │  Policy Store    │ ──► Persist with TTL
    │  └────────┬─────────┘
    │           │
    └─────┬─────┘
          │
          ▼
    Execute or Deny
```

**Design Decision**: The security system uses a policy persistence layer with configurable TTLs to reduce approval fatigue while maintaining security. Policies are scoped to session (8h default) or global (30 days default).

### ACE Package (`internal/ace/`)

ACE (Agentic Context Engineering) provides persistent learning capabilities.

#### ACE Pipeline

```
Execution Trajectory
         │
         ▼
┌──────────────────┐
│    Reflector     │ ──► Analyze trajectory, extract insights
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│     Curator      │ ──► Deduplicate, quality control, merge
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Playbook      │ ──► Store bullets with embeddings
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Retriever     │ ──► HNSW-based semantic similarity
└─────────────────────────────────────────────────
         │
         ▼
    Relevant Bullets
         │
         ▼
┌──────────────────┐
│  Prompt Builder  │ ──► Inject bullets into system prompt
└─────────────────────────────────────────────────
```

#### Bullet Structure

```go
type Bullet struct {
    ID           string            // Unique identifier
    Content      string            // The actual knowledge
    Embedding    []float32         // Vector embedding
    HelpfulCount int               // Positive feedback count
    HarmfulCount int               // Negative feedback count
    Tags         map[string]string // Metadata
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Design Decision**: ACE uses a dual-pipeline approach:
1. **Simple Generation**: Quick bullet creation from execution context
2. **Reflection Pipeline**: Deep analysis using Reflector → Curator for higher quality insights

The system supports ItemizedLearning where the LLM can provide explicit feedback (HELPFUL/HARMFUL markers) on retrieved bullets, creating a feedback loop for continuous improvement.

---

## Key Interfaces and Abstractions

### LLM Provider Interface

```go
type Provider interface {
    // Streaming completion (real-time response chunks)
    Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error)
    
    // Non-streaming completion
    Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)
    
    // List available models
    Models(ctx context.Context) ([]openai.Model, error)
    
    // Provider capabilities
    Capabilities() Capabilities
    
    // Provider identification
    Name() string
    
    // Resource cleanup
    Close() error
}
```

**Design Decision**: All providers use OpenAI SDK types directly, even for non-OpenAI providers (Ollama, LMStudio). This eliminates unnecessary abstraction layers since these providers implement OpenAI-compatible APIs. The result is:
- Simpler code with fewer type conversions
- Consistent behavior across providers
- Easier maintenance and debugging

### Tool Interface

```go
type Tool interface {
    // Unique tool identifier
    Name() string
    
    // Human-readable description
    Description() string
    
    // OpenAI-compatible function schema
    Schema() ToolSchema
    
    // Execute the tool
    Execute(ctx context.Context, params ToolParameters) (ToolResult, error)
}
```

**Design Decision**: Tools use OpenAI function calling schema for maximum compatibility. The registry pattern allows dynamic tool registration while maintaining type safety.

### Task Interface

```go
type Task interface {
    Name() string           // Task identifier
    SystemPrompt() string   // System prompt for this mode
    AllowedTools() []string // Tool whitelist (empty = all)
    MaxTokens() int         // Token budget
    Validate() error        // Configuration validation
}
```

**Design Decision**: Task modes use a factory pattern (`NewTask(name)`) instead of a runtime registry. This provides:
- Compile-time safety
- Clear enumeration of available modes
- Simpler code without registration ceremony

### Runtime Interface

```go
type Runtime interface {
    RegisterTools(registry *tools.Registry)
    NotificationSender() NotificationSender
    ApprovalHandler() security.ApprovalHandler
    SessionStorage() session.Storage
    SessionID() string
    SupportsTerminals() bool
    TerminalClient() TerminalClient
}
```

**Design Decision**: The Runtime interface abstracts environment-specific capabilities, enabling the same Agent code to run in different contexts (local builtin vs ACP protocol-based).

---

## Data Flow

### Conversation Turn Flow

```
┌─────────────────┐
│   User Input    │
│  (TUI/Exec/ACP) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Conversation   │
│   RunTurn()     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Get History     │ ──► Retrieve previous messages
│ Create Task     │ ──► Determine task mode
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Agent.Execute() │
└────────┬────────┘
         │
    ┌────┴────────────────────────────────┐
    │                                      │
    ▼                                      │
┌─────────────────┐                       │
│  Build Prompt   │                       │
│  (+ ACE bullets)│                       │
└────────┬────────┘                       │
         │                                 │
         ▼                                 │
┌─────────────────┐                       │
│   Call LLM      │ ◄──────────────────┐  │
│  (Streaming)    │                    │  │
└────────┬────────┘                    │  │
         │                             │  │
         ├──► EventContentDelta        │  │
         │                             │  │
         ▼                             │  │
┌─────────────────┐                    │  │
│ Parse Response  │                    │  │
└────────┬────────┘                    │  │
         │                             │  │
    ┌────┴────┐                        │  │
    │         │                        │  │
    ▼         ▼                        │  │
 Content   ToolCalls                   │  │
    │         │                        │  │
    │         ▼                        │  │
    │  ┌─────────────────┐            │  │
    │  │ Security Check  │            │  │
    │  │ + Approval      │            │  │
    │  └────────┬────────┘            │  │
    │           │                     │  │
    │           ▼                     │  │
    │  ┌─────────────────┐            │  │
    │  │ Execute Tool    │            │  │
    │  └────────┬────────┘            │  │
    │           │                     │  │
    │           ├──► EventToolCallComplete
    │           │                     │  │
    │           ▼                     │  │
    │  ┌─────────────────┐            │  │
    │  │ Add Result to   │            │  │
    │  │ Messages        │            │  │
    │  └────────┬────────┘            │  │
    │           │                     │  │
    │           └─────────────────────┘  │
    │                                     │
    └────────────────┬────────────────────┘
                     │
                     ▼
         ┌─────────────────┐
         │ ACE Learning    │ ──► Generate bullets from trajectory
         └────────┬────────┘
                  │
                  ▼
         ┌─────────────────┐
         │ Update History  │
         └────────┬────────┘
                  │
                  ▼
         ┌─────────────────┐
         │ EventTurnComplete│
         └─────────────────┘
```

### Event Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Event Sources                                 │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Agent Loop    │   Tool Exec     │   Security/Approval         │
└────────┬────────┴────────┬────────┴─────────────┬───────────────┘
         │                 │                       │
         ▼                 ▼                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    EventEmitter                                  │
│  (Buffered channel, non-blocking, backpressure handling)        │
└────────┬────────────────────────────────────────────────────────┘
         │
         ├─────────────────────────────────────────────┐
         │                                             │
         ▼                                             ▼
┌─────────────────┐                          ┌─────────────────┐
│  TUI Handler    │                          │  ACP Handler    │
│                 │                          │                 │
│ - Update display│                          │ - Transform to  │
│ - Render blocks │                          │   notifications │
│ - Handle input  │                          │ - Send to client│
└─────────────────┘                          └─────────────────┘
```

**Design Decision**: The event system uses a buffered channel with non-blocking emission. This ensures:
- Agent execution never blocks on slow consumers
- Real-time feedback to UI/protocol handlers
- Graceful backpressure handling

---

## Design Patterns

### 1. Functional Options Pattern

Used extensively for configurable components:

```go
type AgentOption func(*Agent) error

agent, err := NewAgent(provider, security, detection, runtime, planning, env, emitter,
    WithMaxTurns(50),
    WithTemperature(0.7),
    WithRequireApproval(true),
    WithACEService(aceService),
)
```

**Rationale**: Provides clear defaults while allowing extensive customization without breaking API compatibility.

### 2. Builder Pattern

Used for complex object construction:

```go
agent := agent.NewBuilder().
    WithConfig(cfg).
    WithProvider(provider).
    WithEmitter(emitter).
    BuildExecutor()
```

**Rationale**: Manages complex dependency graphs and provides fluent, readable construction code.

### 3. Service-Oriented Architecture

Components are organized as injectable services:

```go
Services:
├── git.Service       // Git operations
├── shell.Service     // Command execution  
├── mcp.Service       // MCP servers
├── security.SecurityService
├── detection.DetectionService
├── planning.PlanningService
└── agent.ACEService
```

**Rationale**: Enables independent testing, runtime swapping, and clear boundaries between concerns.

### 4. Observer Pattern (Event-Driven)

```go
// Emit events
emitter.Emit(events.Event{
    Type:      events.EventContentDelta,
    Timestamp: time.Now(),
    Data:      events.ContentDeltaData{Content: content},
})

// Consume events
for event := range emitter.Events() {
    handleEvent(event)
}
```

**Rationale**: Decouples event producers from consumers, enabling real-time UI updates without blocking execution.

### 5. Strategy Pattern

Multiple implementations of core interfaces:

```go
// LLM Providers
providers := map[string]llm.Provider{
    "openai":    openai.NewProvider(...),
    "anthropic": anthropic.NewProvider(...),
    "ollama":    ollama.NewProvider(...),
}

// Task Modes
tasks := map[string]task.Task{
    "regular":  task.NewRegular(),
    "review":   task.NewReview(),
    "compact":  task.NewCompact(),
    "planning": task.NewPlanning(),
}
```

**Rationale**: Allows runtime selection of implementation without conditional logic scattered throughout the codebase.

### 6. Adapter Pattern

Protocol adaptation for external interfaces:

```go
// ACP Agent adapts internal Agent to ACP protocol
type SpinACPAgent struct {
    agent      *agent.Agent
    connection notificationSender
    // ... transforms internal events to ACP notifications
}

// Event Transformer adapts internal events to protocol format
type EventTransformer interface {
    Transform(event events.Event) interface{}
}
```

**Rationale**: Isolates protocol-specific concerns from core business logic.

### 7. Repository Pattern

Storage abstraction for persistence:

```go
type Storage interface {
    Save(session *Session) error
    Load(id string) (*Session, error)
    List() ([]*Session, error)
    Delete(id string) error
}

// Implementations
type FileStorage struct { /* JSON files */ }
type MemoryStorage struct { /* In-memory */ }
```

**Rationale**: Decouples business logic from storage implementation, enabling easy testing and storage backend changes.

---

## Architectural Decisions

### ADR-001: OpenAI SDK as Universal Type System

**Context**: Spin supports multiple LLM providers (OpenAI, Anthropic, Ollama, LMStudio).

**Decision**: Use OpenAI SDK types directly for all providers, even non-OpenAI ones.

**Consequences**:
- (+) Single type system throughout the codebase
- (+) No conversion layers needed for OpenAI-compatible providers
- (+) Simpler debugging and maintenance
- (-) Anthropic requires type conversion layer
- (-) Coupling to OpenAI SDK versioning

### ADR-002: Service-Based Agent Architecture

**Context**: Agent needs security validation, cycle detection, planning, and learning capabilities.

**Decision**: Inject services rather than embedding functionality in Agent.

**Consequences**:
- (+) Each service can be tested independently
- (+) Services can be swapped at runtime (e.g., different approval handlers)
- (+) Clear separation of concerns
- (+) Optional features (ACE) can be nil without conditional checks
- (-) More complex construction (mitigated by Builder pattern)

### ADR-003: Event-Driven Real-Time Feedback

**Context**: Users need real-time feedback during LLM streaming and tool execution.

**Decision**: Use buffered channel-based event emitter with non-blocking emission.

**Consequences**:
- (+) Agent execution never blocks on slow consumers
- (+) Multiple consumers can process events independently
- (+) Graceful degradation under backpressure
- (-) Potential event loss if buffer overflows
- (-) Consumers must handle event ordering

### ADR-004: Task Mode System

**Context**: Different use cases require different tool access, token budgets, and prompts.

**Decision**: Implement task modes as compile-time types with factory function.

**Consequences**:
- (+) Compile-time safety for mode validation
- (+) Clear enumeration of available modes
- (+) No runtime registration needed
- (-) Adding new modes requires code changes (acceptable for core modes)

### ADR-005: Configuration V2 Schema

**Context**: Configuration was becoming complex with flat structure.

**Decision**: Organize configuration into logical sections (LLM, Agent, ACE, Security, Protocol).

**Consequences**:
- (+) Clear organization of related settings
- (+) Section-level validation
- (+) Easier to document and understand
- (+) Supports multiple sources (file, env, flags) with proper precedence
- (-) Migration required from V1 config

### ADR-006: Policy Persistence with TTL

**Context**: Requiring approval for every dangerous command creates friction.

**Decision**: Persist approval decisions with configurable TTL (session: 8h, global: 30d).

**Consequences**:
- (+) Reduced approval fatigue
- (+) Maintains security for new commands
- (+) Configurable TTL per scope
- (-) Potential security risk if TTL too long
- (-) Storage overhead for policy files

### ADR-007: ACE Dual-Pipeline Learning

**Context**: Learning from execution needs balance between speed and quality.

**Decision**: Implement two pipelines: simple generation and reflection-based.

**Consequences**:
- (+) Quick learning for simple cases
- (+) High-quality insights via reflection pipeline
- (+) Automatic deduplication via Curator
- (-) Increased complexity
- (-) LLM cost for reflection pipeline

### ADR-008: HNSW for Semantic Retrieval

**Context**: ACE needs fast semantic similarity search over learned bullets.

**Decision**: Use HNSW (Hierarchical Navigable Small World) index for retrieval.

**Consequences**:
- (+) O(log N) search complexity
- (+) Handles growing playbooks efficiently
- (+) Battle-tested algorithm
- (-) Memory overhead for index
- (-) Index rebuild on significant updates

---

## Configuration System

### Configuration Structure (V2)

```go
type ConfigV2 struct {
    Version  string           // Schema version
    LLM      LLMConfigV2      // LLM provider settings
    Agent    AgentConfigV2    // Agent behavior settings
    ACE      ACEConfigV2      // Learning system settings
    Security SecurityConfigV2 // Security and approval settings
    Protocol ProtocolConfigV2 // Protocol features (MCP, Git, Shell)
}
```

### Configuration Sources (Precedence Order)

1. **CLI Flags** (highest priority)
   ```bash
   spin --model gpt-4 --provider openai
   ```

2. **Environment Variables**
   ```bash
   export SPIN_LLM_PROVIDER=openai
   export SPIN_LLM_MODEL=gpt-4
   ```

3. **Project Config File** (`./spin.yaml`)
   ```yaml
   llm:
     provider: openai
     model: gpt-4
   ```

4. **User Config File** (`~/.config/spin/config.yaml`)
   ```yaml
   llm:
     provider: ollama
     model: qwen2.5-coder:7b
   ```

5. **Default Values** (lowest priority)

### Default Configuration

```go
func DefaultConfigV2() *ConfigV2 {
    return &ConfigV2{
        Version: "2.0",
        LLM: LLMConfigV2{
            Provider:    "ollama",
            Model:       "qwen2.5-coder:7b",
            Temperature: 0.7,
            MaxTokens:   8192,
            Timeout:     5 * time.Minute,
        },
        Agent: AgentConfigV2{
            MaxTurns:        50,
            Timeout:         60 * time.Minute,
            RequireApproval: false,
            CycleDetection: CycleDetectionConfigV2{
                Enabled:          true,
                SimilarityThresh: 0.8,
                ToolRepeatLimit:  3,
            },
        },
        ACE: ACEConfigV2{
            Enabled:  true,
            TopK:     5,
            MinScore: 0.3,
        },
        Security: SecurityConfigV2{
            SandboxMode:      "workspace-only",
            SessionPolicyTTL: 8 * time.Hour,
            GlobalPolicyTTL:  30 * 24 * time.Hour,
        },
        Protocol: ProtocolConfigV2{
            EnableGit:    true,
            EnableShell:  true,
            ShellTimeout: 5 * time.Minute,
        },
    }
}
```

---

## External Dependencies

### Core Dependencies

| Package | Purpose | Why Chosen |
|---------|---------|------------|
| `github.com/openai/openai-go` | LLM API types | Official SDK, universal type system |
| `github.com/spf13/cobra` | CLI framework | Industry standard, subcommand support |
| `github.com/spf13/viper` | Configuration | Multi-source config, env binding |
| `github.com/go-git/go-git/v5` | Git operations | Pure Go, no git binary dependency |
| `github.com/google/uuid` | ID generation | Standard UUID implementation |
| `gopkg.in/yaml.v3` | YAML parsing | Config file format |
| `github.com/coder/acp-go-sdk` | ACP protocol | IDE integration protocol |
| `github.com/mark3labs/mcp-go` | MCP protocol | Model Context Protocol |
| `github.com/coder/hnsw` | Vector index | Fast semantic search |
| `github.com/creack/pty` | PTY support | Terminal emulation |
| `github.com/zalando/go-keyring` | Credential storage | Platform keyring integration |

### Dependency Decisions

**Why OpenAI SDK for all providers?**
- Most local LLM servers (Ollama, LMStudio) implement OpenAI-compatible APIs
- Eliminates custom type systems and conversion layers
- Single set of types to learn and document

**Why go-git instead of git CLI?**
- No external dependency on git binary
- Better error handling and type safety
- Consistent behavior across platforms

**Why Cobra + Viper?**
- De facto standard for Go CLI applications
- Excellent subcommand support
- Viper provides env/file/flag merging

---

## Component Interaction Summary

```
┌─────────────────────────────────────────────────────────────────────┐
│                         User Interface                               │
│  ┌─────────┐  ┌─────────┐  ┌─────────────────────────────────────┐ │
│  │   TUI   │  │  Exec   │  │              ACP Client              │ │
│  └────┬────┘  └────┬────┘  └──────────────────┬──────────────────┘ │
└───────┼────────────┼──────────────────────────┼─────────────────────┘
        │            │                          │
        └────────────┼──────────────────────────┘
                     │
                     ▼
        ┌────────────────────────┐
        │    Conversation.       │
        │      RunTurn()         │
        └───────────┬────────────┘
                    │
                    ▼
        ┌────────────────────────┐
        │     Agent.Execute()    │◄──────────────┐
        └───────────┬────────────┘               │
                    │                            │
         ┌──────────┼──────────┐                │
         ▼          ▼          ▼                │
    ┌─────────┐ ┌─────────┐ ┌─────────┐        │
    │   LLM   │ │Security │ │  Tools  │        │
    │Provider │ │ Service │ │ Runtime │────────┘
    └────┬────┘ └────┬────┘ └────┬────┘
         │          │           │
         ▼          ▼           ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐
    │ OpenAI  │ │Approval │ │  Shell  │
    │ Ollama  │ │ Policy  │ │   Git   │
    │Anthropic│ │  Store  │ │  Files  │
    └─────────┘ └─────────┘ └─────────┘
```

---

## Conclusion

Spin's architecture is designed around several core principles:

1. **Service-Oriented Design**: All major capabilities are encapsulated in services that can be independently tested, configured, and swapped.

2. **Event-Driven Communication**: Real-time feedback through non-blocking event emission enables responsive UIs and protocol handlers.

3. **Multi-Mode Execution**: The same core agent can operate through TUI, batch execution, or ACP protocol without code changes.

4. **Extensible Tool System**: New tools can be added by implementing the Tool interface and registering with the runtime.

5. **Persistent Learning**: ACE provides automatic knowledge capture and retrieval, improving over time.

6. **Security-First**: Command validation and approval workflows ensure safe operation with configurable policies.

The architecture successfully balances flexibility with maintainability, enabling Spin to serve as both a standalone CLI tool and an IDE-integrated assistant through the ACP protocol.
