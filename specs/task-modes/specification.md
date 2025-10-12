# Task Mode System Integration Specification

**Status**: Design Specification
**Created**: 2025-10-12
**Author**: System Analysis
**Related**: `internal/core/task/`, `internal/core/agent.go`

## Overview

This specification describes how to integrate the existing task mode system into the Spin agent. The task mode system is **fully implemented** but **never wired up** to production code. This document provides a complete technical plan for activation.

## Current State

### Implemented Components

1. **Task Interface** ([agent.go:57-72](../internal/core/agent.go#L57-L72))
   ```go
   type Task interface {
       Name() string
       SystemPrompt() string
       AllowedTools() []string
       MaxTokens() int
       Validate() error
   }
   ```

2. **Four Task Modes** (all in `internal/core/task/`)
   - **Regular**: Full-featured interactive coding (16K tokens, all tools)
   - **Review**: Read-only code analysis (12K tokens, read-only tools)
   - **Compact**: Quick queries (4K tokens, 3 essential tools)
   - **Planning**: Task decomposition (4K tokens, context tools only)

3. **Registry System** ([task.go:89-216](../internal/core/task/task.go#L89-L216))
   - Thread-safe task registration
   - Default task management
   - Name validation and lookup

4. **AgentRequest.Task Field** ([agent.go:151](../internal/core/agent.go#L151))
   - Field exists but is never set

### Missing Integration

```go
// Current state: Task is always nil
req := &AgentRequest{
    Input:   userInput,
    History: historyMsgs,
    Context: c.agent.context,
    WorkDir: workDir,
    // Task: nil  <- Never set!
}
```

**Result**: Default prompt always used, tool filtering never applied, token budgets ignored.

## Integration Architecture

### Phase 1: Core Wiring

#### 1.1 Add Task Registry to Agent

**File**: `internal/core/agent.go`

```go
type Agent struct {
    llm             llm.Provider
    executor        *Executor
    validator       *Validator
    context         *Environment
    emitter         *EventEmitter
    config          *AgentConfig
    toolRegistry    *tools.Registry
    taskRegistry    *TaskRegistry      // NEW: Add task registry
    approvalHandler ApprovalHandler
}

// TaskRegistry manages available task modes
type TaskRegistry struct {
    tasks       map[string]Task
    defaultTask string
    mu          sync.RWMutex
}
```

**Constructor Update**:
```go
func NewAgent(
    llm llm.Provider,
    executor *Executor,
    validator *Validator,
    context *Environment,
    emitter *EventEmitter,
    opts ...AgentOption,
) (*Agent, error) {
    // ... existing validation ...

    agent := &Agent{
        llm:          llm,
        executor:     executor,
        validator:    validator,
        context:      context,
        emitter:      emitter,
        toolRegistry: tools.NewRegistry(),
        taskRegistry: newDefaultTaskRegistry(), // NEW: Initialize with defaults
        config: &AgentConfig{
            MaxTurns:        DefaultMaxTurns,
            Timeout:         DefaultAgentTimeout,
            Temperature:     DefaultTemperature,
            MaxTokens:       DefaultMaxTokens,
            RequireApproval: false,
            ApprovalTimeout: DefaultApprovalTimeout,
        },
    }

    // Apply options
    for _, opt := range opts {
        if err := opt(agent); err != nil {
            return nil, err
        }
    }

    return agent, nil
}

// newDefaultTaskRegistry creates a registry with all built-in modes
func newDefaultTaskRegistry() *TaskRegistry {
    registry := &TaskRegistry{
        tasks:       make(map[string]Task),
        defaultTask: "regular",
    }

    // Register built-in modes
    registry.Register("regular", task.NewRegular())
    registry.Register("review", task.NewReview())
    registry.Register("compact", task.NewCompact())
    registry.Register("planning", task.NewPlanning())

    return registry
}
```

**Agent Option**:
```go
// WithTaskRegistry sets a custom task registry
func WithTaskRegistry(registry *TaskRegistry) AgentOption {
    return func(a *Agent) error {
        if registry == nil {
            return fmt.Errorf("task registry cannot be nil")
        }
        a.taskRegistry = registry
        return nil
    }
}
```

#### 1.2 Update AgentRequest to Support Task Selection

**File**: `internal/core/agent.go`

Add helper method to resolve task:
```go
// resolveTask returns the task to use for this request.
// Priority: req.Task > req.TaskName > agent.taskRegistry.default
func (a *Agent) resolveTask(req *AgentRequest) (Task, error) {
    // 1. Explicit task object provided
    if req.Task != nil {
        return req.Task, nil
    }

    // 2. Task name provided
    if req.TaskName != "" {
        task, err := a.taskRegistry.Get(req.TaskName)
        if err != nil {
            return nil, fmt.Errorf("task not found: %s", req.TaskName)
        }
        return task, nil
    }

    // 3. Use default task
    task, err := a.taskRegistry.GetDefault()
    if err != nil {
        return nil, fmt.Errorf("no default task available: %w", err)
    }

    return task, nil
}
```

**Update AgentRequest**:
```go
type AgentRequest struct {
    // Input is the user's request
    Input string

    // History is the conversation history
    History []Message

    // Context is the environment context (optional, will use agent's context if nil)
    Context *Environment

    // Task is the task mode (optional, resolved from TaskName or default if nil)
    Task Task

    // TaskName is the name of the task mode to use (optional, resolved from registry)
    // Takes precedence over agent's default but is overridden by Task field
    TaskName string  // NEW: Add task name field

    // WorkDir is the working directory
    WorkDir string
}
```

#### 1.3 Implement Tool Filtering

**File**: `internal/core/agent.go`, in `callLLM()` method

**Current code** (lines 579-595):
```go
// Add tool schemas if tool registry is available
if a.toolRegistry != nil {
    toolSchemas := a.toolRegistry.ListSchemas()
    req.Tools = make([]llm.Tool, len(toolSchemas))
    for i, schema := range toolSchemas {
        // Convert ParameterSchema struct to map[string]interface{}
        params := convertParameterSchemaToMap(schema.Function.Parameters)

        req.Tools[i] = llm.Tool{
            Type: schema.Type,
            Function: llm.Function{
                Name:        schema.Function.Name,
                Description: schema.Function.Description,
                Parameters:  params,
            },
        }
    }
}
```

**NEW: Filter tools based on task mode**:
```go
// buildToolsForRequest constructs the tool list for the LLM request,
// filtered by the task mode's allowed tools.
func (a *Agent) buildToolsForRequest(req *AgentRequest) ([]llm.Tool, error) {
    if a.toolRegistry == nil {
        return nil, nil
    }

    // Get all available tools
    allSchemas := a.toolRegistry.ListSchemas()

    // Resolve task mode
    task, err := a.resolveTask(req)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve task: %w", err)
    }

    // Get allowed tools for this mode
    allowedTools := task.AllowedTools()

    // Build allowed tool set for O(1) lookup
    allowedSet := make(map[string]bool, len(allowedTools))
    for _, name := range allowedTools {
        allowedSet[name] = true
    }

    // Filter tools
    filtered := make([]llm.Tool, 0, len(allSchemas))
    for _, schema := range allSchemas {
        // Check if tool is allowed in this mode
        if !allowedSet[schema.Function.Name] {
            continue
        }

        // Convert ParameterSchema struct to map[string]interface{}
        params := convertParameterSchemaToMap(schema.Function.Parameters)

        filtered = append(filtered, llm.Tool{
            Type: schema.Type,
            Function: llm.Function{
                Name:        schema.Function.Name,
                Description: schema.Function.Description,
                Parameters:  params,
            },
        })
    }

    return filtered, nil
}
```

**Update callLLM()**:
```go
func (a *Agent) callLLM(ctx context.Context, messages []Message) (*llm.CompletionResponse, error) {
    // ... existing message conversion code ...

    // Build LLM request
    req := llm.CompletionRequest{
        Messages:    llmMessages,
        Temperature: a.config.Temperature,
        MaxTokens:   a.config.MaxTokens,
    }

    // Add filtered tool schemas
    tools, err := a.buildToolsForRequest(req)  // NEW: Use filtered tools
    if err != nil {
        return nil, fmt.Errorf("failed to build tools: %w", err)
    }
    req.Tools = tools

    // Call LLM with streaming
    chunks, err := a.llm.Stream(ctx, req)
    // ... rest of method ...
}
```

**PROBLEM**: `callLLM()` doesn't have access to `AgentRequest`!

**Solution**: Pass the task through the call chain:

```go
// In Execute() method, store task in agent state or pass through
func (a *Agent) Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
    // ... existing code ...

    // Resolve task once at start
    task, err := a.resolveTask(req)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve task: %w", err)
    }

    // Store in request for use throughout execution
    req.Task = task

    // ... rest of execute logic ...
}
```

Then update `callLLM()`:
```go
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
    // ... message conversion ...

    // Filter tools by task
    tools, err := a.buildToolsForTask(task)
    if err != nil {
        return nil, fmt.Errorf("failed to build tools: %w", err)
    }

    req := llm.CompletionRequest{
        Messages:    llmMessages,
        Temperature: a.config.Temperature,
        MaxTokens:   a.config.MaxTokens,
        Tools:       tools,
    }

    // ... rest of method ...
}

func (a *Agent) buildToolsForTask(task Task) ([]llm.Tool, error) {
    if a.toolRegistry == nil {
        return nil, nil
    }

    allSchemas := a.toolRegistry.ListSchemas()
    allowedTools := task.AllowedTools()

    allowedSet := make(map[string]bool, len(allowedTools))
    for _, name := range allowedTools {
        allowedSet[name] = true
    }

    filtered := make([]llm.Tool, 0, len(allSchemas))
    for _, schema := range allSchemas {
        if !allowedSet[schema.Function.Name] {
            continue
        }

        params := convertParameterSchemaToMap(schema.Function.Parameters)
        filtered = append(filtered, llm.Tool{
            Type: schema.Type,
            Function: llm.Function{
                Name:        schema.Function.Name,
                Description: schema.Function.Description,
                Parameters:  params,
            },
        })
    }

    return filtered, nil
}
```

**Update all callLLM invocations** in Execute():
```go
// Call LLM
llmResp, err := a.callLLM(ctx, messages, req.Task)
```

#### 1.4 Apply Token Budget from Task

**File**: `internal/core/agent.go`, in `callLLM()` method

```go
func (a *Agent) callLLM(ctx context.Context, messages []Message, task Task) (*llm.CompletionResponse, error) {
    // ... message conversion ...

    // Determine token budget: task overrides agent config
    maxTokens := a.config.MaxTokens
    if task != nil {
        taskMaxTokens := task.MaxTokens()
        if taskMaxTokens > 0 {
            maxTokens = taskMaxTokens
        }
    }

    req := llm.CompletionRequest{
        Messages:    llmMessages,
        Temperature: a.config.Temperature,
        MaxTokens:   maxTokens,  // Use task-specific budget
        Tools:       tools,
    }

    // ... rest of method ...
}
```

### Phase 2: Conversation Integration

#### 2.1 Add Task Selection to Conversation

**File**: `internal/core/conversation.go`

```go
type Conversation struct {
    id            string
    agent         *Agent
    history       *History
    state         State
    mu            sync.RWMutex
    currentTask   Task          // NEW: Track current task mode
    taskName      string        // NEW: Track task name for switching
}

// SetTaskMode switches the conversation to a different task mode
func (c *Conversation) SetTaskMode(taskName string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    task, err := c.agent.taskRegistry.Get(taskName)
    if err != nil {
        return fmt.Errorf("failed to set task mode: %w", err)
    }

    c.currentTask = task
    c.taskName = taskName
    return nil
}

// GetTaskMode returns the current task mode name
func (c *Conversation) GetTaskMode() string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if c.taskName != "" {
        return c.taskName
    }
    return "regular" // default
}
```

**Update SendMessage()**:
```go
func (c *Conversation) sendMessageInternal(
    turnCtx context.Context,
    userInput string,
    controlChan <-chan ControlSignal,
) (<-chan Event, error) {
    // ... existing validation and setup ...

    var historyMsgs []Message
    if c.history != nil {
        historyMsgs = c.history.MessagesForLLM()
    }

    // NEW: Include task mode in request
    c.mu.RLock()
    task := c.currentTask
    taskName := c.taskName
    c.mu.RUnlock()

    req := &AgentRequest{
        Input:    userInput,
        History:  historyMsgs,
        Context:  c.agent.context,
        WorkDir:  workDir,
        Task:     task,      // NEW: Pass task object
        TaskName: taskName,  // NEW: Or task name
    }

    // Execute turn with control signal checking
    return c.runTurnWithControl(turnCtx, req, controlChan)
}
```

#### 2.2 Add Task Mode to Manager

**File**: `internal/core/manager.go` (if it exists, or conversation.go)

```go
type Manager struct {
    config       *Config
    llmProvider  llm.Provider
    toolRegistry *tools.Registry
    taskRegistry *TaskRegistry  // NEW: Add task registry
    // ... other fields ...
}

type ManagerOption func(*Manager) error

// WithTaskRegistry sets a custom task registry for the manager
func WithTaskRegistry(registry *TaskRegistry) ManagerOption {
    return func(m *Manager) error {
        if registry == nil {
            return fmt.Errorf("task registry cannot be nil")
        }
        m.taskRegistry = registry
        return nil
    }
}

// NewConversationWithTask creates a conversation with a specific task mode
func (m *Manager) NewConversationWithTask(ctx context.Context, taskName string) (*Conversation, error) {
    conv, err := m.NewConversation(ctx)
    if err != nil {
        return nil, err
    }

    if taskName != "" {
        if err := conv.SetTaskMode(taskName); err != nil {
            return nil, fmt.Errorf("failed to set task mode: %w", err)
        }
    }

    return conv, nil
}
```

### Phase 3: CLI Integration

#### 3.1 Add Task Mode Flag to CLI

**File**: `cmd/spin/root.go`

```go
var (
    taskMode string  // NEW: Global flag for task mode
)

func newRootCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "spin",
        Short: "Spin - AI-powered coding assistant",
        // ... existing config ...
    }

    // NEW: Add task mode flag
    cmd.PersistentFlags().StringVarP(
        &taskMode,
        "mode",
        "m",
        "regular",
        "Task mode: regular, review, compact, or planning",
    )

    // ... add subcommands ...
    return cmd
}
```

#### 3.2 Add Mode Switching Command

**File**: `cmd/spin/mode.go` (NEW FILE)

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newModeCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mode [mode-name]",
        Short: "Switch task mode or show current mode",
        Long: `Switch the agent's task mode or display the current mode.

Available modes:
  regular  - Full-featured interactive coding (default)
  review   - Read-only code review and analysis
  compact  - Quick queries with minimal context
  planning - Task decomposition and planning

Examples:
  spin mode            # Show current mode
  spin mode review     # Switch to review mode
  spin mode compact    # Switch to compact mode`,
        Args: cobra.MaximumNArgs(1),
        RunE: runMode,
    }

    return cmd
}

func runMode(cmd *cobra.Command, args []string) error {
    // If no argument, show current mode
    if len(args) == 0 {
        fmt.Printf("Current mode: %s\n", taskMode)
        return nil
    }

    // Validate mode
    validModes := map[string]bool{
        "regular":  true,
        "review":   true,
        "compact":  true,
        "planning": true,
    }

    newMode := args[0]
    if !validModes[newMode] {
        return fmt.Errorf("invalid mode: %s (valid: regular, review, compact, planning)", newMode)
    }

    // Update global mode
    taskMode = newMode
    fmt.Printf("Switched to %s mode\n", newMode)

    // TODO: If in active conversation, send mode switch command
    // to update the current conversation's task mode

    return nil
}
```

#### 3.3 Update Interactive REPL

**File**: `cmd/spin/repl.go` (or wherever REPL is implemented)

```go
func startREPL(conv *core.Conversation) error {
    reader := bufio.NewReader(os.Stdin)

    // Show current mode
    fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
    fmt.Println("Type /mode <name> to switch modes, /help for help, /exit to quit")

    for {
        fmt.Print("> ")
        input, err := reader.ReadString('\n')
        if err != nil {
            return err
        }

        input = strings.TrimSpace(input)

        // Handle commands
        if strings.HasPrefix(input, "/") {
            if err := handleCommand(conv, input); err != nil {
                fmt.Printf("Error: %v\n", err)
            }
            continue
        }

        // Send message to agent
        // ... existing message handling ...
    }
}

func handleCommand(conv *core.Conversation, cmd string) error {
    parts := strings.Fields(cmd)
    if len(parts) == 0 {
        return nil
    }

    switch parts[0] {
    case "/mode":
        if len(parts) == 1 {
            fmt.Printf("Current mode: %s\n", conv.GetTaskMode())
            return nil
        }

        newMode := parts[1]
        if err := conv.SetTaskMode(newMode); err != nil {
            return err
        }
        fmt.Printf("Switched to %s mode\n", newMode)
        return nil

    case "/help":
        fmt.Println(`Commands:
  /mode [name]  - Show or switch task mode
  /help         - Show this help
  /exit         - Exit the session

Available modes:
  regular   - Full-featured interactive coding
  review    - Read-only code review
  compact   - Quick queries
  planning  - Task planning`)
        return nil

    case "/exit":
        os.Exit(0)
        return nil

    default:
        return fmt.Errorf("unknown command: %s", parts[0])
    }
}
```

### Phase 4: AppServer Integration

#### 4.1 Add Task Mode to Request Protocol

**File**: `internal/appserver/protocol.go` (or equivalent)

```go
type SendMessageRequest struct {
    ConversationID string `json:"conversation_id"`
    Message        string `json:"message"`
    TaskMode       string `json:"task_mode,omitempty"`  // NEW: Optional task mode
}

type ConversationStatus struct {
    ID      string `json:"id"`
    State   string `json:"state"`
    TaskMode string `json:"task_mode"`  // NEW: Current task mode
    // ... other fields ...
}
```

#### 4.2 Update Processor to Use Task Mode

**File**: `internal/appserver/processor.go`

```go
func (p *Processor) processSendMessage(ctx context.Context, req SendMessageRequest) error {
    // ... existing conversation lookup ...

    // NEW: Switch task mode if specified
    if req.TaskMode != "" {
        if err := conv.SetTaskMode(req.TaskMode); err != nil {
            p.emitError(req.ConversationID, fmt.Errorf("invalid task mode: %w", err))
            return err
        }
    }

    // Create agent request
    agentReq := &core.AgentRequest{
        Input:   req.Message,
        History: conv.History,
        // Task will be set by conversation based on current mode
    }

    // ... existing processing ...
}
```

## Testing Strategy

### Unit Tests

#### Test 1: Task Resolution
```go
func TestAgent_ResolveTask(t *testing.T) {
    tests := []struct {
        name        string
        req         *AgentRequest
        setupAgent  func(*Agent)
        wantTask    string
        wantErr     bool
    }{
        {
            name: "explicit task object",
            req: &AgentRequest{
                Task: task.NewReview(),
            },
            wantTask: "review",
        },
        {
            name: "task by name",
            req: &AgentRequest{
                TaskName: "compact",
            },
            wantTask: "compact",
        },
        {
            name: "default task",
            req: &AgentRequest{},
            wantTask: "regular",
        },
        {
            name: "invalid task name",
            req: &AgentRequest{
                TaskName: "invalid",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := newTestAgent(t)

            task, err := agent.resolveTask(tt.req)
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if task.Name() != tt.wantTask {
                t.Errorf("got task %s, want %s", task.Name(), tt.wantTask)
            }
        })
    }
}
```

#### Test 2: Tool Filtering
```go
func TestAgent_BuildToolsForTask(t *testing.T) {
    agent := newTestAgent(t)

    // Register test tools
    agent.toolRegistry.Register(&tools.Tool{Name: "read_file"})
    agent.toolRegistry.Register(&tools.Tool{Name: "write_file"})
    agent.toolRegistry.Register(&tools.Tool{Name: "shell"})

    tests := []struct {
        name      string
        task      Task
        wantTools []string
    }{
        {
            name:      "regular mode - all tools",
            task:      task.NewRegular(),
            wantTools: []string{"read_file", "write_file", "shell"},
        },
        {
            name:      "review mode - read only",
            task:      task.NewReview(),
            wantTools: []string{"read_file"},
        },
        {
            name:      "compact mode - minimal",
            task:      task.NewCompact(),
            wantTools: []string{"read_file"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tools, err := agent.buildToolsForTask(tt.task)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            gotNames := make([]string, len(tools))
            for i, tool := range tools {
                gotNames[i] = tool.Function.Name
            }

            if !reflect.DeepEqual(gotNames, tt.wantTools) {
                t.Errorf("got tools %v, want %v", gotNames, tt.wantTools)
            }
        })
    }
}
```

#### Test 3: Token Budget Application
```go
func TestAgent_TaskTokenBudget(t *testing.T) {
    tests := []struct {
        name          string
        task          Task
        agentMaxTokens int
        wantMaxTokens  int
    }{
        {
            name:          "regular mode",
            task:          task.NewRegular(),
            agentMaxTokens: 4096,
            wantMaxTokens:  16384, // task overrides
        },
        {
            name:          "compact mode",
            task:          task.NewCompact(),
            agentMaxTokens: 16384,
            wantMaxTokens:  4096, // task restricts
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            agent := newTestAgentWithConfig(t, &AgentConfig{
                MaxTokens: tt.agentMaxTokens,
            })

            // Verify token budget is applied correctly
            // in callLLM - this requires mocking the LLM
            // to inspect the request
        })
    }
}
```

### Integration Tests

#### Test 4: End-to-End Mode Switching
```go
func TestConversation_ModeSwitch(t *testing.T) {
    mgr := newTestManager(t)
    conv, err := mgr.NewConversation(context.Background())
    require.NoError(t, err)

    // Start in regular mode
    assert.Equal(t, "regular", conv.GetTaskMode())

    // Switch to review mode
    err = conv.SetTaskMode("review")
    require.NoError(t, err)
    assert.Equal(t, "review", conv.GetTaskMode())

    // Send message and verify review mode is used
    events, err := conv.SendMessage(context.Background(), "Review this code")
    require.NoError(t, err)

    // Verify only read-only tools are available
    // by checking tool call events
}
```

### E2E Tests

#### Test 5: CLI Mode Switching
```bash
#!/bin/bash
# e2e/test_mode_switching.sh

# Start spin in regular mode
spin chat --mode regular <<EOF
Create a file test.txt
EOF

# Verify file was created (write operation works in regular mode)
[ -f test.txt ] || exit 1

# Start spin in review mode
spin chat --mode review <<EOF
Create a file test2.txt
EOF

# Verify file was NOT created (write operations blocked in review mode)
[ ! -f test2.txt ] || exit 1

echo "Mode switching works correctly"
```

## Rollout Plan

### Step 1: Core Implementation (1-2 days)
- [ ] Add task registry to Agent
- [ ] Implement task resolution
- [ ] Add tool filtering
- [ ] Apply token budgets
- [ ] Write unit tests

### Step 2: Conversation Integration (1 day)
- [ ] Add task tracking to Conversation
- [ ] Implement SetTaskMode()
- [ ] Update SendMessage()
- [ ] Write integration tests

### Step 3: CLI Integration (1 day)
- [ ] Add `--mode` flag
- [ ] Implement `/mode` REPL command
- [ ] Add mode switching help
- [ ] Write E2E tests

### Step 4: AppServer Integration (1 day)
- [ ] Update protocol with task_mode field
- [ ] Update processor to handle mode switching
- [ ] Test with frontend clients

### Step 5: Documentation & Polish (1 day)
- [ ] Update user documentation
- [ ] Add examples for each mode
- [ ] Update API docs
- [ ] Performance testing

## Performance Considerations

### Tool Filtering Overhead
- **Impact**: O(n) filter per LLM call where n = number of tools
- **Mitigation**: Cache filtered tool lists per mode
- **Benchmark target**: < 100μs for filtering

### Mode Switching
- **Impact**: One map lookup + mutex lock
- **Mitigation**: Already using RWMutex for read-heavy access
- **Benchmark target**: < 1μs for GetTaskMode()

### Memory
- **Impact**: 4 task objects per agent (~1KB each)
- **Mitigation**: Singleton task instances shared across conversations
- **Target**: < 10KB overhead per agent

## Security Considerations

### Tool Access Control
- **Risk**: Malicious mode switch to gain tool access
- **Mitigation**:
  - Validate mode names against registry
  - Log all mode switches
  - Consider requiring approval for mode switches in restricted environments

### Token Budget Bypass
- **Risk**: Switch to high-token mode to exhaust resources
- **Mitigation**:
  - Enforce global max token limit regardless of mode
  - Rate limit mode switches
  - Track token usage across modes

## Migration Path

### For Existing Users
1. **No breaking changes**: Default behavior unchanged (regular mode)
2. **Opt-in**: Users must explicitly set mode to use new features
3. **Gradual adoption**: Can start with simple mode flag, then explore mode switching

### Compatibility
- All existing code continues to work unchanged
- New features are additive only
- No changes to existing APIs (only additions)

## Future Enhancements

### Dynamic Mode Creation
Allow users to define custom modes via config:
```yaml
custom_modes:
  security_audit:
    base: review
    system_prompt: "You are a security auditor..."
    allowed_tools: [read_file, search_code, list_dir]
    max_tokens: 8192
```

### Mode Auto-Selection
Automatically select mode based on user input:
```go
// Detect intent and switch modes automatically
if strings.Contains(input, "review") || strings.Contains(input, "analyze") {
    conv.SetTaskMode("review")
}
```

### Mode Composition
Allow combining modes or inheriting from base modes:
```go
customMode := task.NewRegular().
    WithExcludedTools([]string{"shell"}).
    WithMaxTokens(8192).
    WithCustomPrompt("...")
```

### Per-Tool Permissions
More granular control than all-or-nothing:
```go
type ToolPermission struct {
    Tool    string
    Actions []string // ["read", "write", "execute"]
}
```

## Appendix: Complete File Changes

### A. internal/core/agent.go
**Lines to add**: ~200
**Lines to modify**: ~20
**Key changes**:
- Add taskRegistry field
- Add resolveTask() method
- Add buildToolsForTask() method
- Update callLLM() signature
- Update all callLLM() call sites

### B. internal/core/conversation.go
**Lines to add**: ~40
**Lines to modify**: ~10
**Key changes**:
- Add currentTask, taskName fields
- Add SetTaskMode(), GetTaskMode() methods
- Update sendMessageInternal()

### C. cmd/spin/root.go
**Lines to add**: ~10
**Key changes**:
- Add --mode flag

### D. cmd/spin/mode.go (NEW FILE)
**Lines to add**: ~80
**Key changes**:
- Complete new file for mode command

### E. cmd/spin/repl.go
**Lines to add**: ~50
**Key changes**:
- Add command parsing
- Add /mode command handler

### F. internal/appserver/protocol.go
**Lines to add**: ~5
**Key changes**:
- Add task_mode field to requests/responses

### G. internal/appserver/processor.go
**Lines to add**: ~10
**Key changes**:
- Handle task_mode in requests

## Summary

This specification provides a complete, production-ready plan to activate the existing task mode system. The implementation is:

- **Non-breaking**: Fully backward compatible
- **Tested**: Comprehensive test strategy
- **Performant**: Minimal overhead
- **Secure**: Proper validation and controls
- **Extensible**: Clear path for future enhancements

**Total effort estimate**: 5-6 development days + 2 days testing/documentation = ~1.5 weeks

**Priority**: Medium - valuable feature, but no urgent business need
