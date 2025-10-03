# Spin Core Module - Technical Documentation

## Overview

**Package:** `internal/core`  
**Path:** `/home/dmytrogajewski/sources/spin/internal/core/`  
**Type:** Internal package  
**Purpose:** Core business logic and orchestration for the Spin AI agent

The `internal/core` package is the heart of the Spin system, implementing all business logic for the autonomous coding agent. It is designed to be consumed by various UI implementations (TUI, server, SDK) and provides a clean, idiomatic Go API.

## Architecture

### Responsibility Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Public API Layer                         │
│  Manager, Conversation, Agent                               │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│              State Management Layer                         │
│  session/, turn/, state.go                                  │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│             Task Execution Layer                            │
│  executor/, planner/, orchestrator.go                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────────────────┐
│                     │                                       │
│  ┌──────────────────▼──────┐    ┌──────────────────────┐    │
│  │   Tool Execution        │    │   LLM Client         │    │
│  │  internal/tools/        │    │   internal/llm/      │    │
│  └─────────────────────────┘    └──────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │            Safety & Execution                       │    │
│  │  internal/security/, validator.go                   │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
internal/core/
├── manager.go              # Conversation manager (entry point)
├── conversation.go         # Active conversation implementation
├── agent.go                # Agent orchestration logic
├── executor.go             # Task execution coordinator
├── planner.go              # Task planning and decomposition
├── context.go              # Environment context gathering
├── history.go              # Conversation history management
├── validator.go            # Command validation and safety
├── event.go                # Event types and emission
├── error.go                # Error types and handling
├── config.go               # Core configuration
│
├── session/                # Session state management
│   ├── session.go         # Session struct and methods
│   ├── storage.go         # Persistence layer
│   └── metadata.go        # Session metadata
│
├── turn/                   # Turn state management
│   ├── turn.go            # Turn struct and methods
│   ├── state.go           # Turn state machine
│   └── result.go          # Turn execution results
│
├── task/                   # Task execution modes
│   ├── task.go            # Task interface
│   ├── regular.go         # Standard interactive mode
│   ├── review.go          # Code review mode
│   └── compact.go         # Minimal context mode
│
└── stream/                 # Streaming infrastructure
    ├── stream.go          # Event stream handling
    ├── buffer.go          # Stream buffering
    └── types.go           # Stream event types
```

## Key Components

### 1. Conversation Management

#### `Manager`
**File:** `manager.go`

**Purpose:** High-level API for creating and managing conversations.

```go
// Manager coordinates conversation lifecycle and state
type Manager struct {
    config      *Config
    storage     Storage
    llmProvider llm.Provider
    toolRegistry *tools.Registry
    security    *security.Policy
}

// NewManager creates a conversation manager
func NewManager(cfg *Config, opts ...Option) (*Manager, error)

// NewConversation starts a new conversation thread
func (m *Manager) NewConversation(ctx context.Context, workDir string) (*Conversation, error)

// ResumeConversation resumes an existing conversation
func (m *Manager) ResumeConversation(ctx context.Context, sessionID string) (*Conversation, error)

// ListConversations returns conversation history
func (m *Manager) ListConversations(ctx context.Context, filter Filter) ([]*session.Metadata, error)

// ArchiveConversation archives a completed conversation
func (m *Manager) ArchiveConversation(ctx context.Context, sessionID string) error
```

**Responsibilities:**
- Create and initialize conversations
- Manage conversation lifecycle
- Coordinate dependencies (LLM, tools, security)
- Handle persistence operations

#### `Conversation`
**File:** `conversation.go`

**Purpose:** Represents an active conversation with the AI agent.

```go
// Conversation represents an active agent conversation
type Conversation struct {
    session     *session.Session
    agent       *Agent
    history     *History
    eventStream chan Event
    mu          sync.RWMutex
}

// RunTurn executes a single conversation turn
func (c *Conversation) RunTurn(ctx context.Context, userInput string) error

// Stream returns the event channel for real-time updates
func (c *Conversation) Stream() <-chan Event

// Stop gracefully stops the conversation
func (c *Conversation) Stop(ctx context.Context) error

// State returns current conversation state
func (c *Conversation) State() State
```

**Responsibilities:**
- Execute user turns
- Stream events to UI
- Manage conversation state
- Coordinate agent execution

### 2. Agent Orchestration

#### `Agent`
**File:** `agent.go`

**Purpose:** Core agent logic and decision-making loop.

```go
// Agent implements the autonomous coding agent
type Agent struct {
    llm         llm.Provider
    tools       *tools.Registry
    executor    *Executor
    validator   *Validator
    context     *Context
    maxTurns    int
    timeout     time.Duration
}

// Execute runs the agent loop for a user request
func (a *Agent) Execute(ctx context.Context, req Request) (*Response, error)

// ProcessToolCall handles AI tool invocations
func (a *Agent) ProcessToolCall(ctx context.Context, call ToolCall) (*ToolResult, error)

// ShouldApprove determines if a command needs user approval
func (a *Agent) ShouldApprove(cmd Command) (bool, string)
```

**Responsibilities:**
- Implement agent decision loop
- Coordinate LLM interactions
- Execute tool calls
- Apply safety policies
- Manage agent state

**Agent Loop:**
```go
func (a *Agent) Execute(ctx context.Context, req Request) (*Response, error) {
    for i := 0; i < a.maxTurns; i++ {
        // 1. Build prompt with context
        prompt := a.buildPrompt(req, a.context)
        
        // 2. Call LLM
        stream, err := a.llm.Stream(ctx, llm.CompletionRequest{
            Messages: prompt,
            Tools:    a.tools.Schemas(),
        })
        
        // 3. Process streaming response
        for chunk := range stream {
            switch chunk.Type {
            case llm.ContentDelta:
                // Emit content
            case llm.ToolCallDelta:
                // Accumulate tool call
            case llm.ToolCallComplete:
                // Execute tool
                result := a.ProcessToolCall(ctx, chunk.ToolCall)
                // Add result to context
            case llm.Done:
                // Check if complete
                if chunk.FinishReason == "stop" {
                    return response, nil
                }
            }
        }
    }
}
```

### 3. State Management

#### Session (`session/`)

**File:** `session/session.go`

```go
// Session represents a persistent conversation session
type Session struct {
    ID          string
    WorkDir     string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    Turns       []*turn.Turn
    Metadata    Metadata
    State       State
}

// NewSession creates a new session
func NewSession(workDir string) *Session

// AddTurn appends a turn to the session
func (s *Session) AddTurn(t *turn.Turn)

// Save persists session to storage
func (s *Session) Save(storage Storage) error

// Load retrieves session from storage
func Load(storage Storage, id string) (*Session, error)
```

**Storage:** `~/.spin/sessions/`

**Format:** JSON or Protocol Buffers

#### Turn (`turn/`)

**File:** `turn/turn.go`

```go
// Turn represents a single user-AI interaction
type Turn struct {
    ID           string
    SessionID    string
    UserInput    string
    AIResponse   string
    ToolCalls    []ToolCall
    ToolResults  []ToolResult
    State        TurnState
    StartedAt    time.Time
    CompletedAt  time.Time
    TokensUsed   TokenUsage
}

// TurnState represents turn execution state
type TurnState int

const (
    TurnStatePending TurnState = iota
    TurnStateRunning
    TurnStateWaitingApproval
    TurnStateCompleted
    TurnStateFailed
    TurnStateCancelled
)

// Execute runs the turn
func (t *Turn) Execute(ctx context.Context, agent *Agent) error
```

### 4. Task Execution Modes

**Directory:** `task/`

#### Task Interface

```go
// Task defines different execution modes
type Task interface {
    // Name returns task type name
    Name() string
    
    // SystemPrompt returns mode-specific system prompt
    SystemPrompt() string
    
    // AllowedTools returns tools available in this mode
    AllowedTools() []string
    
    // MaxTokens returns token budget for this mode
    MaxTokens() int
    
    // Validate validates task configuration
    Validate() error
}
```

#### Regular Task (`task/regular.go`)
```go
// Regular implements standard interactive coding mode
type Regular struct {
    config *Config
}

func (r *Regular) Name() string { return "regular" }
func (r *Regular) SystemPrompt() string { return regularPrompt }
func (r *Regular) AllowedTools() []string { 
    return []string{"read_file", "write_file", "shell", "search", "git"}
}
```

#### Review Task (`task/review.go`)
```go
// Review implements code review mode
type Review struct {
    config *Config
    files  []string
}

func (r *Review) Name() string { return "review" }
func (r *Review) SystemPrompt() string { return reviewPrompt }
func (r *Review) AllowedTools() []string { 
    return []string{"read_file", "search"}  // Read-only
}
```

#### Compact Task (`task/compact.go`)
```go
// Compact implements minimal context mode
type Compact struct {
    config *Config
}

func (c *Compact) Name() string { return "compact" }
func (c *Compact) MaxTokens() int { return 4096 }  // Reduced budget
```

### 5. Task Execution

#### `Executor`
**File:** `executor.go`

**Purpose:** Coordinates task execution with proper isolation.

```go
// Executor manages task execution
type Executor struct {
    sandbox  *security.Sandbox
    policy   *security.Policy
    workDir  string
    env      map[string]string
    timeout  time.Duration
}

// Execute runs a command with sandboxing
func (e *Executor) Execute(ctx context.Context, cmd Command) (*Result, error)

// ExecuteStreaming runs long-running command with output streaming
func (e *Executor) ExecuteStreaming(ctx context.Context, cmd Command) (<-chan Output, error)

// Validate checks if command is allowed
func (e *Executor) Validate(cmd Command) error
```

**Execution Flow:**
```go
func (e *Executor) Execute(ctx context.Context, cmd Command) (*Result, error) {
    // 1. Validate command against policy
    if err := e.policy.Validate(cmd); err != nil {
        return nil, fmt.Errorf("policy violation: %w", err)
    }
    
    // 2. Apply sandbox
    sandboxedCmd, err := e.sandbox.Wrap(cmd)
    if err != nil {
        return nil, fmt.Errorf("sandbox error: %w", err)
    }
    
    // 3. Execute with timeout
    ctx, cancel := context.WithTimeout(ctx, e.timeout)
    defer cancel()
    
    result, err := e.run(ctx, sandboxedCmd)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}
```

#### `Planner`
**File:** `planner.go`

**Purpose:** Task decomposition and planning.

```go
// Planner implements task planning and decomposition
type Planner struct {
    llm llm.Provider
}

// Plan decomposes a complex task into steps
func (p *Planner) Plan(ctx context.Context, task string) (*Plan, error)

// Plan represents a task execution plan
type Plan struct {
    Task        string
    Steps       []Step
    Dependencies map[string][]string
    Estimated   time.Duration
}

// Step represents a single plan step
type Step struct {
    ID          string
    Description string
    Action      string
    DependsOn   []string
    Status      StepStatus
}
```

### 6. Command Validation

#### `Validator`
**File:** `validator.go`

**Purpose:** Command safety classification and validation.

```go
// Validator classifies command safety
type Validator struct {
    policy *security.Policy
}

// CommandClass represents safety classification
type CommandClass int

const (
    CommandSafe CommandClass = iota        // Auto-execute
    CommandInteractive                     // Needs approval
    CommandDangerous                       // Needs strong approval
    CommandForbidden                       // Never execute
)

// Classify determines command safety class
func (v *Validator) Classify(cmd Command) CommandClass

// IsSafe returns true for safe commands
func (v *Validator) IsSafe(cmd Command) bool

// IsDangerous returns true for dangerous commands
func (v *Validator) IsDangerous(cmd Command) bool
```

**Safe Commands:**
- Read operations: `ls`, `cat`, `grep`, `find`
- Git read: `git status`, `git log`, `git diff`
- Build/test: `go build`, `go test`, `make test`
- Info: `pwd`, `whoami`, `which`

**Interactive Commands:**
- Write operations: `echo >`, `touch`, `mkdir`
- Git write: `git add`, `git commit`
- Package install: `go get`, `npm install`

**Dangerous Commands:**
- Deletion: `rm -rf`, `rmdir`
- System: `sudo`, `chmod +x`
- Network write: `curl -X POST`, `wget -O`
- Git force: `git push --force`, `git reset --hard`

**Forbidden Commands:**
- System damage: `:(){ :|:& };:` (fork bomb)
- Privilege escalation: `sudo su`
- Unsafe network: `curl | bash`

### 7. Environment Context

#### `Context`
**File:** `context.go`

**Purpose:** Gather system and project context for LLM.

```go
// Context contains environment information for AI
type Context struct {
    OS           OSInfo
    Git          *GitInfo
    WorkDir      string
    Files        []FileInfo
    Environment  map[string]string
    ProjectType  string
    Languages    []string
}

// Gather collects context information
func Gather(workDir string, opts ...ContextOption) (*Context, error)

// OSInfo contains operating system details
type OSInfo struct {
    OS      string  // linux, darwin, windows
    Arch    string  // amd64, arm64
    Kernel  string
    Shell   string
}

// GitInfo contains repository information
type GitInfo struct {
    Root           string
    Branch         string
    HasChanges     bool
    UntrackedFiles []string
    Remotes        []Remote
}

// FileInfo contains file metadata
type FileInfo struct {
    Path     string
    Size     int64
    Language string
    Lines    int
}
```

**Context Gathering:**
```go
func Gather(workDir string, opts ...ContextOption) (*Context, error) {
    ctx := &Context{
        WorkDir: workDir,
    }
    
    // OS information
    ctx.OS = gatherOSInfo()
    
    // Git repository
    if gitRoot, err := findGitRoot(workDir); err == nil {
        ctx.Git = gatherGitInfo(gitRoot)
    }
    
    // Project structure
    ctx.Files = scanProjectFiles(workDir)
    ctx.ProjectType = detectProjectType(ctx.Files)
    ctx.Languages = detectLanguages(ctx.Files)
    
    // Filtered environment variables
    ctx.Environment = filterEnvironment(os.Environ())
    
    return ctx, nil
}
```

### 8. History Management

#### `History`
**File:** `history.go`

**Purpose:** Manage conversation history and context window.

```go
// History manages conversation message history
type History struct {
    messages    []Message
    maxTokens   int
    tokenizer   Tokenizer
    mu          sync.RWMutex
}

// AddMessage appends a message to history
func (h *History) AddMessage(msg Message)

// Messages returns all messages
func (h *History) Messages() []Message

// Truncate reduces history to fit token budget
func (h *History) Truncate(budget int) error

// Export saves history to file
func (h *History) Export(path string) error

// Message represents a conversation message
type Message struct {
    Role      string      // system, user, assistant, tool
    Content   string
    ToolCalls []ToolCall
    ToolResult *ToolResult
    Timestamp time.Time
}
```

**Truncation Strategy:**
```go
func (h *History) Truncate(budget int) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    // Always keep system message
    systemMsg := h.messages[0]
    
    // Keep recent messages that fit in budget
    truncated := []Message{systemMsg}
    tokens := h.tokenizer.Count(systemMsg.Content)
    
    // Reverse iteration (keep recent)
    for i := len(h.messages) - 1; i > 0; i-- {
        msgTokens := h.tokenizer.Count(h.messages[i].Content)
        if tokens + msgTokens > budget {
            break
        }
        truncated = append([]Message{h.messages[i]}, truncated...)
        tokens += msgTokens
    }
    
    h.messages = truncated
    return nil
}
```

### 9. Event Streaming

#### Event System
**File:** `event.go`

**Purpose:** Real-time event emission to UI layer.

```go
// Event represents a conversation event
type Event struct {
    Type      EventType
    Timestamp time.Time
    Data      interface{}
}

// EventType defines event categories
type EventType int

const (
    EventContentDelta EventType = iota
    EventToolCallStart
    EventToolCallProgress
    EventToolCallComplete
    EventCommandApproval
    EventError
    EventComplete
)

// EventEmitter sends events to subscribers
type EventEmitter struct {
    subscribers []chan<- Event
    mu          sync.RWMutex
}

// Emit sends event to all subscribers
func (e *EventEmitter) Emit(event Event)

// Subscribe adds a new event subscriber
func (e *EventEmitter) Subscribe() <-chan Event
```

### 10. Configuration

#### `Config`
**File:** `config.go`

```go
// Config contains core configuration
type Config struct {
    // LLM Provider
    Provider        string
    Model           string
    ProviderConfig  map[string]interface{}
    
    // Execution
    MaxTurns        int
    Timeout         time.Duration
    WorkDir         string
    
    // Security
    SandboxMode     security.SandboxMode
    PolicyFile      string
    AllowedCommands []string
    
    // Features
    EnableMCP       bool
    MCPServers      []MCPServerConfig
    EnableGit       bool
    EnableShell     bool
    
    // Performance
    MaxTokens       int
    StreamBuffer    int
    CacheCommands   bool
    
    // Storage
    SessionDir      string
    HistoryLimit    int
}

// Load loads configuration from file
func Load(path string) (*Config, error)

// Validate validates configuration
func (c *Config) Validate() error

// Merge merges configurations (CLI flags override file)
func (c *Config) Merge(other *Config) *Config
```

**Configuration Precedence:**
1. CLI flags (highest)
2. Environment variables
3. Config file (`~/.spin/config.yaml`)
4. Defaults (lowest)

### 11. Error Handling

#### Error Types
**File:** `error.go`

```go
// Core error types
var (
    ErrInvalidInput      = errors.New("invalid input")
    ErrSessionNotFound   = errors.New("session not found")
    ErrExecutionFailed   = errors.New("execution failed")
    ErrPolicyViolation   = errors.New("policy violation")
    ErrLLMError          = errors.New("llm error")
    ErrToolNotFound      = errors.New("tool not found")
    ErrContextTooLarge   = errors.New("context too large")
    ErrTimeout           = errors.New("timeout")
)

// Error wraps errors with context
type Error struct {
    Op   string  // Operation that failed
    Err  error   // Underlying error
    Code int     // Error code
}

func (e *Error) Error() string {
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
    return e.Err
}

// Is implements error matching
func (e *Error) Is(target error) bool {
    t, ok := target.(*Error)
    if !ok {
        return false
    }
    return e.Code == t.Code
}
```

**Error Handling Pattern:**
```go
func (c *Conversation) RunTurn(ctx context.Context, input string) error {
    if err := c.validate(input); err != nil {
        return &Error{
            Op:   "RunTurn.validate",
            Err:  err,
            Code: ErrCodeInvalidInput,
        }
    }
    
    // ... operation ...
    
    if err := c.agent.Execute(ctx, req); err != nil {
        return fmt.Errorf("agent execution: %w", err)
    }
    
    return nil
}
```

## Data Flow

### Complete Turn Execution

```
1. User Input (via TUI/Server/SDK)
   ↓
2. Manager.NewConversation() or ResumeConversation()
   ↓
3. Conversation.RunTurn(userInput)
   ↓
4. Agent.Execute(request)
   ↓
5. Build context (Context.Gather())
   ↓
6. Call LLM (llm.Provider.Stream())
   ↓
7. Process stream chunks:
   a. ContentDelta → Emit event
   b. ToolCall → Process tool
      ├─→ Validator.Classify()
      ├─→ Check approval (if needed)
      ├─→ Executor.Execute()
      └─→ Return result to LLM
   c. Done → Complete turn
   ↓
8. Save to history (History.AddMessage())
   ↓
9. Persist session (Session.Save())
   ↓
10. Emit completion event
```

### Tool Execution Flow

```
1. LLM emits ToolCall
   ↓
2. Agent.ProcessToolCall()
   ↓
3. Validator.Classify(command)
   ↓
4. [If needs approval]
   ├─→ Emit EventCommandApproval
   ├─→ Wait for user response
   └─→ Continue or cancel
   ↓
5. Executor.Validate(command)
   ↓
6. Executor.Execute()
   ├─→ Policy check (security.Policy)
   ├─→ Sandbox wrap (security.Sandbox)
   ├─→ Run command (os/exec)
   └─→ Capture output
   ↓
7. Return ToolResult
   ↓
8. Add result to context
   ↓
9. Continue LLM stream
```

## LLM Provider Integration

### Provider Interface

```go
// Provider interface (from internal/llm)
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
    Models(ctx context.Context) ([]Model, error)
    Capabilities() Capabilities
}

// Core interacts with provider
func (a *Agent) callLLM(ctx context.Context, msgs []Message) (<-chan StreamChunk, error) {
    req := llm.CompletionRequest{
        Messages: msgs,
        Tools:    a.tools.Schemas(),
        Stream:   true,
    }
    return a.llm.Stream(ctx, req)
}
```

### Multi-Provider Support

```go
// Core supports provider switching
type ProviderManager struct {
    providers map[string]llm.Provider
    default   string
}

func (m *Manager) WithProvider(name string) *Manager {
    if p, ok := m.providers[name]; ok {
        m.llmProvider = p
    }
    return m
}
```

## Testing

### Test Structure

```
internal/core/
├── manager_test.go
├── conversation_test.go
├── agent_test.go
├── executor_test.go
├── validator_test.go
├── history_test.go
├── testdata/
│   ├── sessions/
│   ├── conversations/
│   └── fixtures/
└── testing/
    ├── mock_llm.go
    ├── mock_tools.go
    └── helpers.go
```

### Test Utilities

```go
// testing/mock_llm.go
type MockProvider struct {
    responses []StreamChunk
    err       error
}

func (m *MockProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
    ch := make(chan StreamChunk)
    go func() {
        defer close(ch)
        for _, resp := range m.responses {
            ch <- resp
        }
    }()
    return ch, m.err
}

// testing/helpers.go
func NewTestManager(t *testing.T) *Manager {
    cfg := &Config{
        WorkDir:    t.TempDir(),
        MaxTurns:   10,
        Timeout:    5 * time.Second,
    }
    
    mgr, err := NewManager(cfg,
        WithLLMProvider(&MockProvider{}),
        WithToolRegistry(tools.NewRegistry()),
    )
    require.NoError(t, err)
    return mgr
}
```

### Example Test

```go
func TestConversation_RunTurn(t *testing.T) {
    mgr := NewTestManager(t)
    
    // Create conversation
    conv, err := mgr.NewConversation(context.Background(), t.TempDir())
    require.NoError(t, err)
    
    // Run turn
    err = conv.RunTurn(context.Background(), "List files in current directory")
    require.NoError(t, err)
    
    // Check events
    events := collectEvents(conv.Stream(), 1*time.Second)
    assert.True(t, containsEventType(events, EventComplete))
}
```

### Running Tests

```bash
# All core tests
go test ./internal/core/...

# With coverage
go test -cover ./internal/core/...

# Verbose
go test -v ./internal/core/...

# Specific test
go test -run TestConversation_RunTurn ./internal/core

# Race detector
go test -race ./internal/core/...

# Integration tests
go test -tags=integration ./internal/core/...
```

## Performance Considerations

### 1. Streaming
```go
// Always stream LLM responses for lower latency
stream, err := a.llm.Stream(ctx, req)
for chunk := range stream {
    // Process immediately
    a.processChunk(chunk)
}
```

### 2. Concurrency
```go
// Use goroutines for parallel operations
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error {
    return a.gatherContext(ctx)
})

g.Go(func() error {
    return a.loadHistory(ctx)
})

if err := g.Wait(); err != nil {
    return err
}
```

### 3. Context Management
```go
// Aggressive truncation for large contexts
if h.TokenCount() > h.maxTokens {
    h.Truncate(h.maxTokens * 0.8)  // 80% of budget
}
```

### 4. Caching
```go
// Cache command results
type ExecutorCache struct {
    cache map[string]*Result
    ttl   time.Duration
}

func (c *ExecutorCache) Get(cmd string) (*Result, bool) {
    // Return cached result if fresh
}
```

### 5. Buffer Sizing
```go
// Appropriate channel buffers
eventChan := make(chan Event, 100)  // Buffer UI events
streamChan := make(chan StreamChunk, 10)  // Small buffer for LLM
```

## Security Model

### Defense Layers

1. **Input Validation**
   ```go
   func (v *Validator) ValidateInput(input string) error {
       if len(input) > maxInputSize {
           return ErrInputTooLarge
       }
       // Sanitize input
   }
   ```

2. **Command Classification**
   ```go
   func (v *Validator) Classify(cmd Command) CommandClass {
       // Pattern matching against policy
       for _, pattern := range v.policy.Safe {
           if pattern.Match(cmd) {
               return CommandSafe
           }
       }
       // Check dangerous patterns
   }
   ```

3. **Execution Sandboxing**
   ```go
   func (e *Executor) Execute(ctx context.Context, cmd Command) (*Result, error) {
       sandboxed, err := e.sandbox.Wrap(cmd)
       // Execute in restricted environment
   }
   ```

4. **Credential Protection**
   ```go
   // Never log or expose credentials
   func (c *Context) filterEnvironment(env []string) map[string]string {
       filtered := make(map[string]string)
       for _, e := range env {
           key, val := split(e)
           if !isSensitive(key) {
               filtered[key] = val
           }
       }
       return filtered
   }
   ```

## Extension Points

### 1. Custom Tools
```go
// Implement tools.Tool interface
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "..." }
func (t *MyTool) Schema() *jsonschema.Schema { return schema }
func (t *MyTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Implementation
}

// Register tool
registry.Register(&MyTool{})
```

### 2. Custom Tasks
```go
// Implement task.Task interface
type CustomTask struct{}

func (t *CustomTask) Name() string { return "custom" }
func (t *CustomTask) SystemPrompt() string { return customPrompt }
func (t *CustomTask) AllowedTools() []string { return tools }
```

### 3. Custom Validators
```go
// Extend Validator
type CustomValidator struct {
    *Validator
}

func (v *CustomValidator) Classify(cmd Command) CommandClass {
    // Custom logic
    if myCustomCheck(cmd) {
        return CommandSafe
    }
    return v.Validator.Classify(cmd)
}
```

### 4. Event Handlers
```go
// Subscribe to events
events := conversation.Stream()
for event := range events {
    switch event.Type {
    case EventToolCallStart:
        // Custom handling
    }
}
```

## Debugging

### Logging
```go
import "log/slog"

// Structured logging throughout core
slog.Info("turn started",
    "session_id", session.ID,
    "turn_id", turn.ID,
)

slog.Error("execution failed",
    "error", err,
    "command", cmd,
)
```

### Debug Mode
```go
// Enable verbose logging
if cfg.Debug {
    slog.SetLogLoggerLevel(slog.LevelDebug)
}
```

### Tracing
```go
// OpenTelemetry integration
import "go.opentelemetry.io/otel"

func (a *Agent) Execute(ctx context.Context, req Request) (*Response, error) {
    ctx, span := otel.Tracer("spin").Start(ctx, "Agent.Execute")
    defer span.End()
    
    // ... implementation ...
}
```

### Environment Variables
- `SPIN_DEBUG=1` - Enable debug logging
- `SPIN_HOME` - Override config directory
- `SPIN_TRACE=1` - Enable tracing

## Dependencies

### Standard Library
- `context` - Cancellation and timeouts
- `os/exec` - Command execution
- `encoding/json` - Serialization
- `sync` - Concurrency primitives
- `time` - Timing and durations

### Internal Packages
- `internal/llm` - LLM provider interface
- `internal/tools` - Tool implementations
- `internal/security` - Sandbox and policy
- `internal/protocol` - Type definitions
- `internal/mcp` - MCP client

### External Dependencies
- `golang.org/x/sync/errgroup` - Concurrent error handling
- `go.opentelemetry.io/otel` - Observability (optional)

## Future Enhancements

- [ ] Multi-agent conversations (agent collaboration)
- [ ] Model ensemble (multiple LLMs voting)
- [ ] Distributed execution (remote workers)
- [ ] Enhanced caching (Redis/disk)
- [ ] Parallel tool execution (goroutine pool)
- [ ] Streaming file operations
- [ ] WebSocket event streaming
- [ ] Agent memory (long-term context)

## Related Packages

- `internal/llm` - LLM provider implementations
- `internal/tools` - Tool implementations
- `internal/security` - Sandbox and policy
- `internal/mcp` - MCP integration
- `cmd/tui` - TUI consumer of core
- `cmd/server` - Server consumer of core
- `pkg/sdk` - SDK wrapper around core

## Key Design Patterns

1. **Interface Segregation:** Small, focused interfaces (Provider, Tool, Task)
2. **Dependency Injection:** Pass dependencies via constructor options
3. **Event-Driven:** Streaming events to decouple UI from core
4. **Context Propagation:** Use context.Context throughout
5. **Error Wrapping:** fmt.Errorf with %w for error chains
6. **Builder Pattern:** Functional options for configuration

## Go Idioms Applied

1. **Accept interfaces, return structs**
   ```go
   func NewAgent(llm llm.Provider) *Agent
   ```

2. **Functional options**
   ```go
   func NewManager(cfg *Config, opts ...Option) (*Manager, error)
   ```

3. **Error handling**
   ```go
   if err != nil {
       return fmt.Errorf("operation failed: %w", err)
   }
   ```

4. **Goroutines and channels**
   ```go
   events := make(chan Event)
   go func() {
       // Emit events
   }()
   ```

5. **Context cancellation**
   ```go
   func (c *Conversation) RunTurn(ctx context.Context, input string) error {
       select {
       case <-ctx.Done():
           return ctx.Err()
       case <-c.done:
           return nil
       }
   }
   ```

## Documentation

- **Godoc:** Full documentation in code comments
- **Examples:** `examples/` directory with runnable code
- **Architecture:** This document
- **API Reference:** Generated via `godoc` or `pkgsite`


