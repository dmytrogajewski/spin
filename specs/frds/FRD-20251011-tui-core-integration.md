# FRD-20251011: TUI-Core Integration

**Feature:** Integrate TUI with Spin's core agent
**Phase:** 7.4 (Critical path)
**Priority:** P0
**Complexity:** Medium
**Author:** Spin Agent
**Date:** 2025-10-11
**Status:** Draft

---

## 1. Overview

### 1.1 Purpose

Wire the native TUI (Phases 1-7.3) into Spin's core agent to provide a rich, interactive terminal interface for coding sessions. Map core agent events to TUI blocks, stream LLM responses to the UI, and coordinate user input with turn submission.

### 1.2 Goals

1. **Event-to-Block Mapper**: Translate core events into TUI blocks (EXECUTE, PLAN, READ, GREP, APPLY_PATCH, SUMMARY, TESTING, NOTICE, ERROR)
2. **Streaming Integration**: Wire `LLM stream events` → `UI.PrintChunks`
3. **Input Coordination**: Wire `UI.RequestInput` → `core turn submission`
4. **End-to-End Flow**: Prove full conversation flow: `user prompt → LLM → tool → result → summary → blocks`

### 1.3 Non-Goals

- Theming system (deferred to Phase 6.4)
- File preview popup (deferred to Phase 6.3)
- Session persistence/replay (future enhancement)
- Old TUI removal (separate Phase 8.1 task)

---

## 2. Requirements

### 2.1 Functional Requirements

**FR-1: Event-to-Block Mapper**

- FR-1.1: Map `EventToolCallStart` (tool="execute_command") → EXECUTE block (append)
- FR-1.2: Map `EventToolCallComplete` → Update EXECUTE block with result (footer: exit code, duration, output lines)
- FR-1.3: Map `EventContentDelta` (role="assistant") → Stream to UI.PrintChunks
- FR-1.4: Map plan updates → PLAN block (parse task list from LLM content)
- FR-1.5: Map file read tool calls → READ block
- FR-1.6: Map grep tool calls → GREP block
- FR-1.7: Map file edit tool calls → APPLY_PATCH block (diff rendering)
- FR-1.8: Map turn completion → SUMMARY block (if LLM emits summary)
- FR-1.9: Map test execution → TESTING block
- FR-1.10: Map system messages (history compression) → NOTICE block
- FR-1.11: Map errors → ERROR block

**FR-2: Streaming Integration**

- FR-2.1: Subscribe to core `EventEmitter` on conversation start
- FR-2.2: Forward `EventContentDelta` chunks to `UI.PrintChunks` channel
- FR-2.3: Ensure prompt redraw after stream completion (via CoordinatedWriter)

**FR-3: Input Coordination**

- FR-3.1: Read from `UI.RequestInput()` channel
- FR-3.2: Submit line to `Conversation.SendMessage(ctx, line)`
- FR-3.3: Handle Ctrl-C as turn cancellation (context cancel)
- FR-3.4: Handle Ctrl-D as clean exit

**FR-4: Block Management**

- FR-4.1: Append blocks to timeline in real-time as events arrive
- FR-4.2: Update in-flight blocks when tool execution completes
- FR-4.3: Render blocks incrementally (no full refresh, append-only)
- FR-4.4: Support timeline navigation while conversation is running

**FR-5: Error Handling**

- FR-5.1: Display connection errors (LLM provider unavailable)
- FR-5.2: Display validation errors (command denied)
- FR-5.3: Display approval requests (if `RequireApproval=true`)
- FR-5.4: Graceful degradation if block rendering fails

### 2.2 Non-Functional Requirements

**NFR-1: Performance**

- NFR-1.1: Block append latency <1ms (already proven in Phase 7.2)
- NFR-1.2: Event processing overhead <100µs per event
- NFR-1.3: No UI lag during high-frequency streaming (coalescing enabled)

**NFR-2: Concurrency**

- NFR-2.1: Thread-safe block operations (timeline has mutex)
- NFR-2.2: No race conditions between event handler and UI renderer
- NFR-2.3: Context cancellation propagates cleanly

**NFR-3: Testability**

- NFR-3.1: Inject fake event emitter for testing
- NFR-3.2: Integration tests with scripted event sequences
- NFR-3.3: E2E test: full conversation flow from prompt to blocks

---

## 3. Architecture

### 3.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                          User                               │
└──────────────────────┬──────────────────────────────────────┘
                       │ (types input, sees blocks)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                      PureTTY Adapter                        │
│  • RequestInput() → prompt loop                             │
│  • PrintChunks() ← event mapper                             │
│  • AppendBlock() ← event mapper                             │
│  • UpdateBlock() ← event mapper                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
          ┌────────────┴───────────┐
          ▼                        ▼
┌─────────────────────┐  ┌──────────────────────┐
│   Input Channel     │  │  Event Mapper        │
│   (user prompts)    │  │  (events → blocks)   │
└──────────┬──────────┘  └──────────┬───────────┘
           │                        │
           ▼                        │
┌─────────────────────────────────────────────────────────────┐
│                     Conversation                            │
│  • SendMessage(prompt) → Agent                              │
│  • Events() channel → Event stream                          │
└──────────────────────┬─────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                      Agent                                  │
│  • LLM provider                                             │
│  • Tool registry                                            │
│  • Executor                                                 │
│  • Event emitter                                            │
└──────────────────────┬─────────────────────────────────────┘
                       │
                       ▼
                  Event Stream
         (ContentDelta, ToolCallStart, ToolCallComplete, etc.)
```

### 3.2 Event-to-Block Mapping Table

| Core Event | Block Type | Trigger | Update Strategy |
|------------|-----------|---------|-----------------|
| `EventToolCallStart` (tool="execute_command") | EXECUTE | Append new block with command | Append-only |
| `EventToolCallComplete` (tool="execute_command") | EXECUTE | Update block footer | In-place update by ID |
| `EventToolCallStart` (tool="read_file") | READ | Append new block | Append-only |
| `EventToolCallStart` (tool="grep") | GREP | Append new block | Append-only |
| `EventToolCallStart` (tool="edit_file") | APPLY_PATCH | Append new block | Append-only |
| `EventContentDelta` (role="assistant") | (none) | Stream to PrintChunks | Streaming |
| `EventTurnComplete` | SUMMARY | Append if summary detected | Append-only |
| `EventError` | ERROR | Append error block | Append-only |
| `EventInfo` (history compression) | NOTICE | Append notice block | Append-only |

**Note:** Plan detection is heuristic-based. If LLM emits structured task lists, parse and create PLAN block.

### 3.3 Data Flow: User Prompt → Blocks

```
1. User types prompt → UI.RequestInput() emits line
2. Main loop reads from input channel
3. Call Conversation.SendMessage(ctx, prompt)
4. Agent starts turn, emits EventTurnStart
5. LLM streams response → EventContentDelta events
6. Event mapper forwards chunks to UI.PrintChunks
7. LLM requests tool call → EventToolCallStart
8. Event mapper creates EXECUTE block, appends to timeline
9. Executor runs command → EventToolCallProgress (optional)
10. Executor completes → EventToolCallComplete
11. Event mapper updates EXECUTE block footer (exit code, duration)
12. Agent completes turn → EventTurnComplete
13. Timeline now shows full conversation with blocks
```

### 3.4 Code Structure

**New Files:**

- `cmd/spin/tui_integration.go` - Main TUI entry point
- `internal/core/tui_mapper.go` - Event-to-block mapper
- `internal/core/tui_mapper_test.go` - Mapper tests

**Modified Files:**

- `cmd/spin/exec.go` - Add TUI mode flag
- `cmd/spin/root.go` - Wire TUI mode selection
- `internal/ui/adapters/puretty.go` - Add block operations (already implemented in Phase 6.1)

---

## 4. Detailed Design

### 4.1 Event Mapper

```go
package core

import (
    "github.com/dmytrogajewski/spin/internal/ui/blocks"
    "github.com/dmytrogajewski/spin/internal/ui/ports"
    "strings"
    "time"
)

// TUIMapper maps core events to TUI blocks.
type TUIMapper struct {
    ui            ports.UI
    blockRegistry map[string]*blocks.Block // toolID → block (for updates)
}

// NewTUIMapper creates a TUI event mapper.
func NewTUIMapper(ui ports.UI) *TUIMapper {
    return &TUIMapper{
        ui:            ui,
        blockRegistry: make(map[string]*blocks.Block),
    }
}

// MapEvent processes a core event and updates the TUI.
func (m *TUIMapper) MapEvent(event Event) error {
    switch event.Type {
    case EventToolCallStart:
        return m.handleToolStart(event)
    case EventToolCallComplete:
        return m.handleToolComplete(event)
    case EventContentDelta:
        return m.handleContentDelta(event)
    case EventError:
        return m.handleError(event)
    case EventInfo:
        return m.handleInfo(event)
    // ... (other events)
    }
    return nil
}

func (m *TUIMapper) handleToolStart(event Event) error {
    data := event.Data.(ToolCallStartData)

    switch data.ToolName {
    case "execute_command":
        block := blocks.NewBlock(blocks.BlockTypeExecute)
        block.ID = data.ToolID
        block.Title = extractCommand(data.Parameters)
        block.Meta = map[string]interface{}{
            "command": extractCommand(data.Parameters),
            "cwd":     extractCwd(data.Parameters),
        }
        m.blockRegistry[data.ToolID] = block
        m.ui.AppendBlock(block)

    case "read_file":
        block := blocks.NewBlock(blocks.BlockTypeRead)
        block.ID = data.ToolID
        block.Title = extractFilePath(data.Parameters)
        // ... populate metadata
        m.ui.AppendBlock(block)

    // ... (other tools)
    }
    return nil
}

func (m *TUIMapper) handleToolComplete(event Event) error {
    data := event.Data.(ToolCallCompleteData)

    block, exists := m.blockRegistry[data.ToolID]
    if !exists {
        return nil // Block not tracked (shouldn't happen)
    }

    // Update block with result
    block.Body = data.Output
    if !data.Success {
        block.Severity = blocks.SeverityError
    }

    // Update metadata footer
    if data.ToolName == "execute_command" {
        meta := block.Meta.(map[string]interface{})
        meta["exit_code"] = extractExitCode(data.Output)
        meta["duration"] = calculateDuration(event.Timestamp) // needs start time
    }

    m.ui.UpdateBlock(block.ID, block)
    delete(m.blockRegistry, data.ToolID)
    return nil
}

func (m *TUIMapper) handleContentDelta(event Event) error {
    data := event.Data.(ContentDeltaData)
    if data.Role == "assistant" {
        // Stream to UI (needs channel-based API)
        // For now, print line by line
        m.ui.PrintLine(data.Content)
    }
    return nil
}

func (m *TUIMapper) handleError(event Event) error {
    data := event.Data.(ErrorData)
    block := blocks.NewBlock(blocks.BlockTypeError)
    block.Title = "Error: " + data.Message
    block.Body = data.Details
    block.Severity = blocks.SeverityError
    m.ui.AppendBlock(block)
    return nil
}

// Helper functions
func extractCommand(params types.ToolCallArguments) string {
    if cmd, ok := params["command"].(string); ok {
        return cmd
    }
    return "unknown"
}
```

### 4.2 TUI Integration Main Loop

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/dmytrogajewski/spin/internal/core"
    "github.com/dmytrogajewski/spin/internal/ui/adapters"
)

func runTUIMode(cfg *core.Config, mgr *core.Manager) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Initialize TUI
    ui := adapters.NewPureTTY()
    go ui.Run(ctx)
    defer ui.Stop()

    // Start conversation
    conv, err := mgr.NewConversation(ctx)
    if err != nil {
        return err
    }

    // Create event mapper
    mapper := core.NewTUIMapper(ui)

    // Subscribe to conversation events
    events := conv.Events()

    // Handle signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    // Main event loop
    go func() {
        for event := range events {
            if err := mapper.MapEvent(event); err != nil {
                ui.PrintLine(fmt.Sprintf("Mapper error: %v", err))
            }
        }
    }()

    // Input loop
    inputCh := ui.RequestInput()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()

        case <-sigCh:
            ui.PrintLine("\n^C received, exiting...")
            return nil

        case line, ok := <-inputCh:
            if !ok {
                return nil // UI closed (Ctrl-D)
            }

            if line == "" {
                continue
            }

            // Submit to agent
            turnCtx, turnCancel := context.WithCancel(ctx)
            defer turnCancel()

            _, err := conv.SendMessage(turnCtx, line)
            if err != nil {
                ui.PrintLine(fmt.Sprintf("Error: %v", err))
            }
        }
    }
}
```

### 4.3 Streaming Content

**Challenge:** `EventContentDelta` emits chunks rapidly, but `UI.PrintChunks` expects a channel.

**Solution:** Buffer chunks and flush periodically or on turn completion.

```go
func (m *TUIMapper) handleContentDelta(event Event) error {
    data := event.Data.(ContentDeltaData)
    if data.Role != "assistant" {
        return nil
    }

    // Stream directly to PrintChunks (needs channel API adjustment)
    // For MVP: accumulate and print on EventContentComplete
    if m.contentBuffer == nil {
        m.contentBuffer = new(strings.Builder)
    }
    m.contentBuffer.WriteString(data.Content)
    return nil
}

func (m *TUIMapper) handleContentComplete(event Event) error {
    if m.contentBuffer != nil {
        m.ui.PrintLine(m.contentBuffer.String())
        m.contentBuffer = nil
    }
    return nil
}
```

**Better approach:** Create chunk channel and wire to `PrintChunks`.

```go
func (m *TUIMapper) StartStreaming() chan string {
    m.streamCh = make(chan string, 100)
    go func() {
        // Forward chunks to UI.PrintChunks
        m.ui.PrintChunks(context.Background(), m.streamCh)
    }()
    return m.streamCh
}

func (m *TUIMapper) handleContentDelta(event Event) error {
    data := event.Data.(ContentDeltaData)
    if data.Role == "assistant" && m.streamCh != nil {
        select {
        case m.streamCh <- data.Content:
        default:
            // Drop if channel full (UI.PrintChunks has coalescing)
        }
    }
    return nil
}
```

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Test Cases:**

1. **Event Mapper:**
   - `TestMapEvent_ToolCallStart_Execute` - Creates EXECUTE block
   - `TestMapEvent_ToolCallComplete_Execute` - Updates block footer
   - `TestMapEvent_ToolCallStart_Read` - Creates READ block
   - `TestMapEvent_ToolCallStart_Grep` - Creates GREP block
   - `TestMapEvent_ToolCallStart_EditFile` - Creates APPLY_PATCH block
   - `TestMapEvent_ContentDelta` - Streams to UI
   - `TestMapEvent_Error` - Creates ERROR block
   - `TestMapEvent_Info` - Creates NOTICE block

2. **Edge Cases:**
   - Block ID collision (same toolID twice)
   - Missing block on ToolCallComplete
   - Nil data payloads
   - Empty command strings

### 5.2 Integration Tests

**Scenario 1: Simple Command Execution**

```go
func TestIntegration_SimpleCommand(t *testing.T) {
    // Setup fake UI
    fakeUI := &FakeUI{}
    mapper := core.NewTUIMapper(fakeUI)

    // Emit events
    mapper.MapEvent(Event{
        Type: EventToolCallStart,
        Data: ToolCallStartData{
            ToolID:   "tool-1",
            ToolName: "execute_command",
            Parameters: map[string]interface{}{
                "command": "ls -la",
            },
        },
    })

    mapper.MapEvent(Event{
        Type: EventToolCallComplete,
        Data: ToolCallCompleteData{
            ToolID:   "tool-1",
            ToolName: "execute_command",
            Success:  true,
            Output:   "total 8\ndrwxr-xr-x 2 user user 4096 Oct 11 10:00 .\n",
        },
    })

    // Assertions
    assert.Equal(t, 1, len(fakeUI.Blocks))
    block := fakeUI.Blocks[0]
    assert.Equal(t, blocks.BlockTypeExecute, block.Type)
    assert.Equal(t, "ls -la", block.Title)
    assert.Contains(t, block.Body, "total 8")
}
```

**Scenario 2: Full Conversation Flow**

```go
func TestIntegration_FullConversation(t *testing.T) {
    // Setup: fake LLM, real conversation, fake UI
    // 1. User prompt
    // 2. LLM emits ContentDelta events
    // 3. LLM requests tool call
    // 4. Tool executes
    // 5. LLM emits summary
    // 6. Verify: UI has EXECUTE block + summary text
}
```

### 5.3 E2E Tests

**Test: End-to-End TUI Flow**

```bash
# Setup PTY
pty := NewTestPTY()

# Start Spin in TUI mode
cmd := exec.Command("./bin/spin", "--mode=tui", "--provider=fake")
cmd.Stdin = pty.Master
cmd.Stdout = pty.Master

# Send prompt
pty.Write("list files\n")

# Wait for block rendering
pty.WaitFor("▐EXECUTE▌")
pty.WaitFor("ls")

# Verify output
output := pty.ReadAll()
assert.Contains(t, output, "EXECUTE")
assert.Contains(t, output, "exit: 0")
```

---

## 6. Acceptance Criteria

### 6.1 Functional Acceptance

- [ ] **AC-1:** User types prompt → EXECUTE block appears on tool call
- [ ] **AC-2:** Command completes → Block footer shows exit code, duration, line count
- [ ] **AC-3:** LLM streaming content appears in real-time without lag
- [ ] **AC-4:** File read → READ block with line numbers
- [ ] **AC-5:** File edit → APPLY_PATCH block with diff
- [ ] **AC-6:** Error occurs → ERROR block with stack trace
- [ ] **AC-7:** History compression → NOTICE block
- [ ] **AC-8:** Timeline navigable during conversation (PgUp/PgDn work)
- [ ] **AC-9:** Ctrl-C cancels current turn cleanly
- [ ] **AC-10:** Ctrl-D exits with clean shutdown

### 6.2 Quality Gates

- [ ] All tests pass with `-race` (zero race conditions)
- [ ] Coverage ≥85% for mapper and integration code
- [ ] `make lint` clean (zero errors, complexity ≤15)
- [ ] Godoc on all exports
- [ ] No visual glitches (prompt always at bottom, blocks render correctly)
- [ ] Performance: Block append <1ms, event processing <100µs

### 6.3 Manual QA

- [ ] Test on Linux, macOS terminals (xterm, alacritty, kitty)
- [ ] Test in tmux and screen sessions
- [ ] Test with slow LLM responses (high latency)
- [ ] Test with rapid tool calls (stress test)
- [ ] Test Ctrl-C mid-execution
- [ ] Test timeline scrolling during active conversation

---

## 7. Implementation Plan

### 7.1 Task Breakdown

1. **Write TUIMapper (2-3 hours)**
   - Implement event-to-block mapping logic
   - Write unit tests for each event type
   - Test edge cases (nil data, missing blocks)

2. **Write Integration Main Loop (1-2 hours)**
   - Create `cmd/spin/tui_integration.go`
   - Wire UI.RequestInput → Conversation.SendMessage
   - Wire Conversation.Events → Mapper → UI blocks
   - Handle signals (Ctrl-C, Ctrl-D)

3. **Streaming Content (1 hour)**
   - Implement chunk buffering and forwarding
   - Wire to UI.PrintChunks
   - Test coalescing behavior

4. **Integration Tests (2 hours)**
   - Write fake UI for testing
   - Scenario: simple command execution
   - Scenario: full conversation with multiple tools
   - Verify block rendering

5. **E2E Test (1 hour)**
   - PTY-based test with real binary
   - Scripted interaction
   - Verify ANSI output

6. **Lint & Analysis (30 min)**
   - Run `uast parse | herr analyze`
   - Run `make lint`
   - Fix any issues

7. **Documentation (30 min)**
   - Update ROADMAP.md (mark Phase 7.4 complete)
   - Update docs/tui.md (add integration section)
   - Write usage examples

**Total Estimate:** 8-10 hours

### 7.2 Dependencies

- Phases 1-7.3 must be complete (✅ Done)
- Core event system (✅ Already implemented)
- Block system (✅ Already implemented)
- PureTTY adapter (✅ Already implemented)

### 7.3 Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Event mapper complexity | Medium | Medium | Incremental implementation, test-driven |
| Streaming lag | Low | Medium | Use coalescing (already proven in Phase 3.1) |
| Race conditions | Low | High | Use `-race` detector throughout |
| Tool parameter extraction | Medium | Low | Use type assertions with fallbacks |

---

## 8. API Design

### 8.1 TUIMapper Interface

```go
package core

// TUIMapper maps core events to TUI blocks.
type TUIMapper interface {
    // MapEvent processes a core event and updates the TUI.
    MapEvent(event Event) error

    // Close cleans up resources (closes stream channels).
    Close() error
}

// NewTUIMapper creates a new TUI event mapper.
func NewTUIMapper(ui ports.UI) TUIMapper
```

### 8.2 UI Port Extensions (Already in Phase 5.1)

```go
package ports

// UI is the port interface for terminal user interfaces.
type UI interface {
    // ... (existing methods)

    // Block operations (added in Phase 6.1)
    AppendBlock(block *blocks.Block) error
    UpdateBlock(blockID string, block *blocks.Block) error
    DeleteBlock(blockID string) error
}
```

---

## 9. Metrics & Observability

### 9.1 Metrics to Track

- **Event processing latency:** Time from event emit to block append
- **Block append throughput:** Blocks/sec
- **Streaming lag:** Time from ContentDelta to UI render
- **Input-to-response latency:** Time from prompt submit to first block

### 9.2 Logging

```go
slog.Info("TUI mapper processing event",
    "event_type", event.Type.String(),
    "tool_id", data.ToolID,
    "block_type", blockType,
)
```

### 9.3 Tracing (Optional)

- Span per event: `TUIMapper.MapEvent`
- Child spans: block creation, UI update
- Trace full conversation flow

---

## 10. Future Enhancements (Out of Scope)

- **Session persistence:** Save/restore timeline to disk
- **Session replay:** Replay events from log file
- **Live collaboration:** Multi-user TUI with shared timeline
- **Voice input:** Integrate with speech-to-text
- **Custom block types:** Plugin system for user-defined blocks
- **LLM diff view:** Show real-time edits as LLM streams

---

## 11. References

- [ROADMAP.md](../tui-implementation/ROADMAP.md) - Phase 7.4
- [docs/tui.md](../../docs/tui.md) - TUI documentation
- [docs/packages/core.md](../../docs/packages/core.md) - Core package documentation
- [internal/core/event.go](../../internal/core/event.go) - Event types
- [internal/ui/ports/ui.go](../../internal/ui/ports/ui.go) - UI port interface
- [internal/ui/blocks/model.go](../../internal/ui/blocks/model.go) - Block data model

---

**Last Updated:** 2025-10-11
**Next Steps:** Write tests, implement mapper, wire main loop, run lint/analysis
