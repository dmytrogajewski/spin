# FRD-UI-3.11: TUI Integration with Core

**Feature:** TUI Integration with Core Module
**Phase:** 3.11
**Status:** Draft
**Priority:** High (Blocking Phase 3.12)
**Created:** 2025-10-05

---

## Overview

This FRD defines the integration between the TUI (Bubble Tea) and the core business logic module. It establishes the communication bridge for bi-directional event flow, real-time streaming, tool approval, and state synchronization.

**Prerequisites:**
- ✅ Phase 0.1: Approval Response Mechanism (COMPLETE)
- ✅ Phase 0.2: Pause/Resume Turn Execution (COMPLETE)
- ✅ Phase 0.3: Event Backpressure Control (COMPLETE)
- ✅ Phase 0.4: Provider Factory Integration (COMPLETE)
- ✅ Phases 3.1-3.10: TUI Infrastructure (COMPLETE)

---

## Goals

1. **Real-Time Streaming**: Display AI responses as they arrive (delta updates)
2. **Tool Approval Flow**: Handle tool approval requests interactively
3. **State Synchronization**: Keep TUI state in sync with core conversation state
4. **Error Handling**: Gracefully handle and display core errors
5. **Turn Management**: Handle turn lifecycle (start, pause, resume, complete, cancel)
6. **Provider Integration**: Use real LLM providers (Ollama, OpenAI, etc.)

---

## Architecture

### Communication Pattern

```
┌─────────────────────┐          ┌─────────────────────┐
│   TUI (Bubble Tea)  │          │   Core Module       │
│                     │          │                     │
│  ┌───────────────┐  │          │  ┌───────────────┐  │
│  │  app.go       │  │          │  │  Manager      │  │
│  │  (Model)      │  │          │  │               │  │
│  └───────┬───────┘  │          │  └───────┬───────┘  │
│          │          │          │          │          │
│  ┌───────▼───────┐  │          │  ┌───────▼───────┐  │
│  │ Event Handler │  │          │  │ Conversation  │  │
│  │               │◄─┼──────────┼──│ (Events chan) │  │
│  └───────────────┘  │          │  └───────────────┘  │
│          │          │          │          ▲          │
│  ┌───────▼───────┐  │          │  ┌───────┴───────┐  │
│  │ Approval      │  │          │  │ Agent         │  │
│  │ Handler       │──┼──────────┼──►               │  │
│  └───────────────┘  │          │  └───────────────┘  │
└─────────────────────┘          └─────────────────────┘
```

### Core Integration Points

1. **Manager Creation**: Initialize core.Manager with provider
2. **Conversation Start**: Create conversation via manager
3. **Event Subscription**: Subscribe to core event channel
4. **Message Sending**: Send user messages to core
5. **Approval Callback**: Respond to approval requests
6. **Turn Control**: Pause/resume/cancel via core API

---

## Event Flow

### 1. User Message Submission

```
User: "Fix authentication bug"
  ↓
TUI: Add message to transcript
  ↓
TUI: Send to core via conv.SendMessage()
  ↓
Core: Emit EventTurnStart
  ↓
TUI: Update state → StateWaitingResponse
  ↓
Core: Stream deltas via EventContentDelta
  ↓
TUI: Append deltas to current message
  ↓
Core: Emit EventTurnComplete
  ↓
TUI: Update state → StateIdle
```

### 2. Tool Approval Flow

```
Core: Proposes tool (EventCommandApproval)
  ↓
TUI: Update state → StateToolApproval
  ↓
TUI: Show approval modal
  ↓
User: Approve/Deny/Modify
  ↓
TUI: Call approval handler
  ↓
Core: Receive ApprovalResponse
  ↓
Core: Emit EventCommandApproved/Denied
  ↓
TUI: Update state → StateWaitingResponse
```

### 3. Cancellation

```
User: Ctrl+C
  ↓
TUI: Call conv.Stop()
  ↓
Core: Cancel context
  ↓
Core: Emit EventTurnComplete (cancelled)
  ↓
TUI: Update state → StateIdle
```

---

## Implementation

### Package Structure

```
cmd/spin/
└── tui.go                  # TUI command entry point

internal/tui/
├── app.go                  # Bubble Tea model (MODIFY)
├── core_integration.go     # NEW: Core integration
├── event_handler.go        # NEW: Core event handling
└── ui/
    ├── chat.go             # Chat component (MODIFY)
    ├── approval.go         # Approval modal (existing)
    └── ...
```

### Core Types

```go
// internal/tui/core_integration.go

package tui

import (
    "context"
    "github.com/dmytrogajewski/spin/internal/core"
    "github.com/dmytrogajewski/spin/internal/llm/builder"
)

// CoreManager wraps core functionality for TUI
type CoreManager struct {
    manager  *core.Manager
    conv     *core.Conversation
    ctx      context.Context
    cancel   context.CancelFunc
}

// NewCoreManager creates a core manager for TUI
func NewCoreManager(cfg *core.Config, provider llm.Provider) (*CoreManager, error) {
    ctx, cancel := context.WithCancel(context.Background())

    mgr, err := core.NewManager(
        cfg,
        core.WithLLMProvider(provider),
    )
    if err != nil {
        cancel()
        return nil, err
    }

    return &CoreManager{
        manager: mgr,
        ctx:     ctx,
        cancel:  cancel,
    }, nil
}

// StartConversation creates a new conversation
func (cm *CoreManager) StartConversation() (<-chan core.Event, error) {
    conv, err := cm.manager.NewConversation(cm.ctx)
    if err != nil {
        return nil, err
    }
    cm.conv = conv
    return conv.Events(), nil
}

// SendMessage sends user message
func (cm *CoreManager) SendMessage(content string) error {
    if cm.conv == nil {
        return fmt.Errorf("no active conversation")
    }
    return cm.conv.SendMessage(cm.ctx, content)
}

// Stop cancels the current turn
func (cm *CoreManager) Stop() error {
    if cm.conv == nil {
        return nil
    }
    return cm.conv.Stop()
}

// Pause pauses the current turn
func (cm *CoreManager) Pause() error {
    if cm.conv == nil {
        return nil
    }
    return cm.conv.Pause()
}

// Resume resumes a paused turn
func (cm *CoreManager) Resume() error {
    if cm.conv == nil {
        return nil
    }
    return cm.conv.Resume()
}

// Close cleanup
func (cm *CoreManager) Close() error {
    cm.cancel()
    if cm.manager != nil {
        return cm.manager.Close()
    }
    return nil
}
```

### Event Handler

```go
// internal/tui/event_handler.go

package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/dmytrogajewski/spin/internal/core"
)

// CoreEventMsg wraps core events as Bubble Tea messages
type CoreEventMsg struct {
    Event core.Event
}

// waitForCoreEvent returns a Bubble Tea command that waits for next core event
func waitForCoreEvent(events <-chan core.Event) tea.Cmd {
    return func() tea.Msg {
        event, ok := <-events
        if !ok {
            return nil // Channel closed
        }
        return CoreEventMsg{Event: event}
    }
}

// handleCoreEvent processes core events and updates model
func (m Model) handleCoreEvent(msg CoreEventMsg) (Model, tea.Cmd) {
    switch msg.Event.Type {
    case core.EventTypeTurnStart:
        return m.handleTurnStart(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeStreamContent:
        return m.handleStreamDelta(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeCommandApproval:
        return m.handleApprovalRequest(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeCommandApproved:
        return m.handleApprovalResult(msg.Event, true), waitForCoreEvent(m.events)

    case core.EventTypeCommandDenied:
        return m.handleApprovalResult(msg.Event, false), waitForCoreEvent(m.events)

    case core.EventTypeTurnComplete:
        return m.handleTurnComplete(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeTurnPaused:
        return m.handleTurnPaused(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeTurnResumed:
        return m.handleTurnResumed(msg.Event), waitForCoreEvent(m.events)

    case core.EventTypeError:
        return m.handleError(msg.Event), waitForCoreEvent(m.events)

    default:
        return m, waitForCoreEvent(m.events)
    }
}

// Event-specific handlers

func (m Model) handleTurnStart(event core.Event) Model {
    m.state = StateWaitingResponse
    // Add system message if needed
    return m
}

func (m Model) handleStreamDelta(event core.Event) Model {
    if data, ok := event.Data.(*core.ContentDeltaData); ok {
        m.chat.AppendDelta(data.Content)
    }
    return m
}

func (m Model) handleApprovalRequest(event core.Event) Model {
    if data, ok := event.Data.(*core.ApprovalRequest); ok {
        m.state = StateToolApproval
        m.approval.SetRequest(data)
    }
    return m
}

func (m Model) handleApprovalResult(event core.Event, approved bool) Model {
    m.state = StateWaitingResponse
    // Add approval result to transcript
    return m
}

func (m Model) handleTurnComplete(event core.Event) Model {
    if data, ok := event.Data.(*core.TurnCompleteData); ok {
        m.statusBar.SetTokens(data.TokensUsed)
    }
    m.state = StateIdle
    m.chat.FinalizeMessage()
    return m
}

func (m Model) handleTurnPaused(event core.Event) Model {
    m.state = StateIdle // Can interact while paused
    // Show pause indicator
    return m
}

func (m Model) handleTurnResumed(event core.Event) Model {
    m.state = StateWaitingResponse
    return m
}

func (m Model) handleError(event core.Event) Model {
    if data, ok := event.Data.(*core.ErrorData); ok {
        m.chat.AddError(data.Error)
    }
    m.state = StateIdle
    return m
}
```

### Model Updates

```go
// internal/tui/app.go (MODIFICATIONS)

type Model struct {
    // ... existing fields ...

    // Core integration
    coreManager *CoreManager
    events      <-chan core.Event

    // Approval handler
    approvalChan chan core.ApprovalResponse
}

// Init command
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        textinput.Blink,
        waitForCoreEvent(m.events),
    )
}

// Update message routing
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyPress(msg)

    case CoreEventMsg:
        return m.handleCoreEvent(msg)

    case tea.WindowSizeMsg:
        return m.handleResize(msg)

    default:
        return m, nil
    }
}

// Send user message
func (m Model) sendMessage() (Model, tea.Cmd) {
    content := m.input.Value()
    if content == "" {
        return m, nil
    }

    // Add to transcript
    m.chat.AddUserMessage(content)

    // Send to core
    if err := m.coreManager.SendMessage(content); err != nil {
        m.chat.AddError(err)
        return m, nil
    }

    // Clear input
    m.input.SetValue("")
    m.state = StateWaitingResponse

    return m, nil
}

// Handle Ctrl+C cancellation
func (m Model) cancelTurn() (Model, tea.Cmd) {
    if m.state != StateWaitingResponse {
        return m, nil
    }

    if err := m.coreManager.Stop(); err != nil {
        m.chat.AddError(err)
    }

    return m, nil
}
```

### Approval Handler Integration

```go
// internal/tui/approval_handler.go

package tui

import (
    "github.com/dmytrogajewski/spin/internal/core"
)

// createApprovalHandler creates approval handler for core
func (m *Model) createApprovalHandler() core.ApprovalHandler {
    return func(req core.ApprovalRequest) core.ApprovalResponse {
        // Send request to TUI via channel (non-blocking)
        select {
        case m.approvalRequestChan <- req:
            // Wait for user decision
            select {
            case resp := <-m.approvalResponseChan:
                return resp
            case <-m.coreManager.ctx.Done():
                return core.ApprovalResponse{
                    RequestID: req.ID,
                    Approved:  false,
                    Reason:    "cancelled",
                }
            }
        case <-m.coreManager.ctx.Done():
            return core.ApprovalResponse{
                RequestID: req.ID,
                Approved:  false,
                Reason:    "cancelled",
            }
        }
    }
}

// Handle approval decision from UI
func (m Model) respondToApproval(approved bool, modified string) (Model, tea.Cmd) {
    if m.approval.request == nil {
        return m, nil
    }

    resp := core.ApprovalResponse{
        RequestID: m.approval.request.ID,
        Approved:  approved,
        Timestamp: time.Now(),
    }

    if modified != "" {
        resp.ModifiedCommand = &modified
    }

    // Send response
    select {
    case m.approvalResponseChan <- resp:
    default:
    }

    m.state = StateWaitingResponse
    m.approval.Clear()

    return m, nil
}
```

### TUI Command Entry Point

```go
// cmd/spin/tui.go (MODIFICATIONS)

func runTUI(cmd *cobra.Command, args []string) error {
    // Load configuration
    cfg := loadConfig(cmd)

    // Create provider
    provider, err := builder.CreateProvider(cfg)
    if err != nil {
        return fmt.Errorf("create provider: %w", err)
    }
    defer provider.Close()

    // Create core config
    coreConfig := core.DefaultConfig()
    coreConfig.Model = cfg.Model
    coreConfig.MaxTokens = cfg.MaxTokens
    coreConfig.Temperature = cfg.Temperature

    // Create core manager
    coreMgr, err := tui.NewCoreManager(coreConfig, provider)
    if err != nil {
        return fmt.Errorf("create core manager: %w", err)
    }
    defer coreMgr.Close()

    // Start conversation and get event channel
    events, err := coreMgr.StartConversation()
    if err != nil {
        return fmt.Errorf("start conversation: %w", err)
    }

    // Create TUI model
    model := tui.NewModel(coreMgr, events)

    // Start Bubble Tea program
    p := tea.NewProgram(model, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        return fmt.Errorf("run TUI: %w", err)
    }

    return nil
}
```

---

## Data Flow Diagrams

### Message Flow

```
┌─────────┐   Enter    ┌─────────┐   SendMessage()   ┌─────────┐
│  User   │───────────►│   TUI   │──────────────────►│  Core   │
└─────────┘            └─────────┘                   └─────────┘
                            │                             │
                            │                        ┌────▼────┐
                            │                        │  Agent  │
                            │                        └────┬────┘
                            │                             │
                            │   EventStreamContent        │
                            │◄────────────────────────────┤
                            │                             │
                            │   (multiple deltas...)      │
                            │◄────────────────────────────┤
                            │                             │
                            │   EventTurnComplete         │
                            │◄────────────────────────────┘
                            ▼
                        [Display]
```

### Approval Flow

```
┌─────────┐            ┌─────────┐            ┌─────────┐
│  Core   │            │   TUI   │            │  User   │
└────┬────┘            └────┬────┘            └────┬────┘
     │                      │                      │
     │  EventCommandApproval│                      │
     ├─────────────────────►│                      │
     │                      │                      │
     │                      │  [Show Modal]        │
     │                      ├─────────────────────►│
     │                      │                      │
     │                      │      [A/D/M]         │
     │                      │◄─────────────────────┤
     │                      │                      │
     │  ApprovalResponse    │                      │
     │◄─────────────────────┤                      │
     │                      │                      │
     │  EventCommandApproved│                      │
     ├─────────────────────►│                      │
     │                      │                      │
     ▼                      ▼                      ▼
```

---

## Testing Strategy

### Unit Tests

```go
// internal/tui/event_handler_test.go

func TestHandleStreamDelta(t *testing.T) {
    m := NewTestModel()

    event := core.Event{
        Type: core.EventTypeStreamContent,
        Data: &core.ContentDeltaData{
            Content: "Hello",
        },
    }

    m, _ = m.handleStreamDelta(event)

    assert.Contains(t, m.chat.CurrentMessage(), "Hello")
}

func TestHandleApprovalRequest(t *testing.T) {
    m := NewTestModel()

    req := &core.ApprovalRequest{
        ID: "test-123",
        Command: core.Command{Raw: "rm -rf node_modules"},
    }

    event := core.Event{
        Type: core.EventTypeCommandApproval,
        Data: req,
    }

    m, _ = m.handleApprovalRequest(event)

    assert.Equal(t, StateToolApproval, m.state)
    assert.NotNil(t, m.approval.request)
}
```

### Integration Tests

```go
// internal/tui/integration_test.go

func TestFullConversationFlow(t *testing.T) {
    // Create mock provider
    mockProvider := llm.NewMockProvider("test",
        llm.WithResponse("I'll help you"),
    )

    // Create core manager
    cfg := core.DefaultConfig()
    coreMgr, err := NewCoreManager(cfg, mockProvider)
    require.NoError(t, err)
    defer coreMgr.Close()

    // Start conversation
    events, err := coreMgr.StartConversation()
    require.NoError(t, err)

    // Create model
    m := NewModel(coreMgr, events)

    // Send message
    m, _ = m.sendMessage("Hello")

    // Wait for events
    timeout := time.After(2 * time.Second)
    for {
        select {
        case event := <-events:
            m, _ = m.handleCoreEvent(CoreEventMsg{Event: event})
            if event.Type == core.EventTypeTurnComplete {
                assert.Equal(t, StateIdle, m.state)
                return
            }
        case <-timeout:
            t.Fatal("timeout waiting for turn complete")
        }
    }
}
```

---

## Configuration

### Core Config

```go
type CoreConfig struct {
    // LLM Settings
    Model       string
    Temperature float64
    MaxTokens   int

    // Backpressure for TUI
    BackpressureMode core.BackpressureMode
    BufferSize       int
    BufferLimit      int

    // Approval
    ApprovalTimeout time.Duration
}

func DefaultTUIConfig() *CoreConfig {
    return &CoreConfig{
        Model:            "gpt-4",
        Temperature:      0.7,
        MaxTokens:        4096,
        BackpressureMode: core.BackpressureBuffer, // Buffer for bursty UI
        BufferSize:       100,
        BufferLimit:      5000,
        ApprovalTimeout:  60 * time.Second,
    }
}
```

---

## Error Handling

### Error Types

1. **Provider Errors**: LLM API failures → Display in chat, allow retry
2. **Core Errors**: Internal errors → Display in chat, log to stderr
3. **Network Errors**: Connection issues → Show status bar warning, auto-retry
4. **Timeout Errors**: Request timeout → Display in chat, suggest retry
5. **Approval Timeout**: User didn't respond → Auto-deny, notify user

### Error Display

```go
func (m Model) handleError(event core.Event) Model {
    if data, ok := event.Data.(*core.ErrorData); ok {
        // Determine error type
        switch {
        case errors.Is(data.Error, llm.ErrRateLimited):
            m.chat.AddSystemMessage("Rate limited. Retrying in 5s...")
            return m.scheduleRetry(5 * time.Second)

        case errors.Is(data.Error, context.DeadlineExceeded):
            m.chat.AddError(fmt.Errorf("request timeout - try again"))

        default:
            m.chat.AddError(data.Error)
        }
    }
    m.state = StateIdle
    return m
}
```

---

## Performance Considerations

### Buffering Strategy

**TUI uses BackpressureBuffer mode:**
- Buffer size: 100 events
- Buffer limit: 5000 events
- Allows bursty AI output without blocking core
- UI remains responsive during fast streaming

### Optimization

1. **Event Batching**: Batch multiple deltas before re-render
2. **Lazy Rendering**: Only render visible viewport
3. **String Building**: Use strings.Builder for efficiency
4. **Channel Sizing**: Properly sized buffers prevent blocking

---

## Security

### Approval Validation

1. Command re-validation after modification
2. Timeout enforcement (60s default)
3. Context cancellation support
4. No command execution without explicit approval

### Logging

- Sensitive data not logged
- Approval decisions audited
- Error details sanitized

---

## Definition of Done

### Functional Requirements

- [ ] TUI can send messages to core ✅
- [ ] TUI receives and displays streaming deltas ✅
- [ ] Tool approval flow works (approve/deny/modify) ✅
- [ ] Cancellation works (Ctrl+C stops turn) ✅
- [ ] State synchronization accurate ✅
- [ ] Error handling graceful ✅
- [ ] Provider integration works (Ollama, OpenAI) ✅

### Quality Requirements

- [ ] Tests passing (≥90% coverage)
- [ ] Race detector clean (`go test -race`)
- [ ] Linter clean (`make lint`)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] No memory leaks (channel cleanup)

### Performance Requirements

- [ ] Event processing <10ms
- [ ] UI responsive during streaming
- [ ] No dropped events (buffering works)
- [ ] Graceful shutdown <500ms

---

## Dependencies

### Internal Packages

- `internal/core` - Core business logic
- `internal/llm` - LLM providers
- `internal/llm/builder` - Provider creation
- `internal/tui/ui` - UI components

### External Libraries

- `github.com/charmbracelet/bubbletea` - TUI framework
- No additional dependencies

---

## Rollout Plan

### Phase 1: Core Integration (This FRD)
1. Create CoreManager wrapper
2. Implement event handler
3. Update Model for core integration
4. Wire up in cmd/spin/tui.go

### Phase 2: Testing
1. Unit tests for event handlers
2. Integration tests with mock provider
3. Manual testing with real providers

### Phase 3: Documentation
1. Update ROADMAP.md
2. Update docs/packages/
3. Add usage examples

---

## Related Documents

- [ROADMAP.md](../../ui-modules/ROADMAP.md) - Implementation roadmap
- [FRD-CORE-0.1.md](FRD-CORE-0.1.md) - Approval response mechanism
- [FRD-CORE-0.2.md](FRD-CORE-0.2.md) - Pause/resume turn execution
- [FRD-CORE-0.3.md](FRD-CORE-0.3.md) - Event backpressure control
- [FRD-CORE-0.4.md](FRD-CORE-0.4.md) - Provider factory integration
- [docs/packages/core.md](../../docs/packages/core.md) - Core package docs
- [docs/packages/llm.md](../../docs/packages/llm.md) - LLM package docs

---

**Author:** Claude (AI Agent)
**Reviewer:** TBD
**Approved:** TBD
