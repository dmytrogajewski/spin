# FRD-4.1: Event Infrastructure

**Feature ID:** 4.1  
**Feature Name:** Event Infrastructure  
**Priority:** P1 (Critical)  
**Estimated Effort:** 10 hours  
**Actual Effort:** ~6 hours  
**Status:** ✅ Complete  
**Phase:** 4 - Event System

---

## Overview

Implement a robust event streaming system that enables real-time communication between the core agent logic and UI layers. This pub/sub pattern allows multiple subscribers to receive events about agent execution, tool calls, content generation, and errors in real-time.

## Rationale

The event system is critical for:
- **Real-time UI updates**: TUI/web interfaces need immediate feedback
- **Progress monitoring**: Track agent execution in real-time
- **Debugging**: Observe agent behavior and decisions as they happen
- **Multi-subscriber support**: Multiple consumers (UI, logging, metrics) can listen
- **Decoupling**: Core logic doesn't need to know about UI implementation

## Definition of Ready (DoR)

- [x] Feature 0.2 completed (Core Types & Errors)
- [x] Event types documented (in spec.md)
- [x] Event flow understanding (pub/sub pattern)

## Definition of Done (DoD)

- [ ] `event.go` implemented with Event struct
- [ ] EventType enum with all event types
- [ ] EventEmitter struct with pub/sub pattern
- [ ] Emit() method for event publication
- [ ] Subscribe() method for event subscription
- [ ] Unsubscribe() mechanism for cleanup
- [ ] Thread-safe event distribution with RWMutex
- [ ] Event buffering strategy (configurable buffer size)
- [ ] Event timestamp tracking
- [ ] Event data type safety
- [ ] Unit tests for event system (>90% coverage)
- [ ] Concurrent subscription tests
- [ ] Event ordering tests
- [ ] Subscriber cleanup tests
- [ ] Memory leak prevention tests
- [ ] Godoc comments for all exported symbols
- [ ] All linters passing
- [ ] Cyclomatic complexity ≤15 for all functions
- [ ] FRD-4.1 marked complete in ROADMAP

---

## Functional Requirements

### FR-4.1.1: Event Structure

**Description:** Define the core Event struct that carries event information.

**Acceptance Criteria:**
- Event struct with Type, Timestamp, and Data fields
- Type-safe data field (interface{})
- JSON serialization support
- Immutable after creation

**Data Structure:**
```go
type Event struct {
    Type      EventType   `json:"type"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}
```

**Test Cases:**
```go
func TestEvent_Structure(t *testing.T) {
    event := Event{
        Type:      EventContentDelta,
        Timestamp: time.Now(),
        Data:      "test content",
    }
    
    // Can serialize to JSON
    data, err := json.Marshal(event)
    require.NoError(t, err)
    
    // Can deserialize
    var decoded Event
    err = json.Unmarshal(data, &decoded)
    require.NoError(t, err)
    assert.Equal(t, EventContentDelta, decoded.Type)
}
```

---

### FR-4.1.2: Event Types

**Description:** Define all event types that the system can emit.

**Acceptance Criteria:**
- EventType as int type for efficiency
- String() method for human-readable names
- All event types defined as constants
- Event types organized by category

**Event Type Categories:**

1. **Content Events**: Text generation from LLM
   - `EventContentDelta` - Incremental content chunk
   - `EventContentComplete` - Content generation finished

2. **Tool Events**: Tool execution lifecycle
   - `EventToolCallStart` - Tool invocation begins
   - `EventToolCallProgress` - Tool execution progress
   - `EventToolCallComplete` - Tool execution finished

3. **Turn Events**: Turn lifecycle
   - `EventTurnStart` - Turn execution begins
   - `EventTurnComplete` - Turn execution finished
   - `EventTurnFailed` - Turn execution failed

4. **Approval Events**: User approval required
   - `EventCommandApproval` - Dangerous command needs approval
   - `EventCommandApproved` - User approved command
   - `EventCommandDenied` - User denied command

5. **System Events**: System-level events
   - `EventError` - Error occurred
   - `EventWarning` - Warning message
   - `EventInfo` - Information message

**Data Structure:**
```go
type EventType int

const (
    // Content events
    EventContentDelta EventType = iota
    EventContentComplete
    
    // Tool events
    EventToolCallStart
    EventToolCallProgress
    EventToolCallComplete
    
    // Turn events
    EventTurnStart
    EventTurnComplete
    EventTurnFailed
    
    // Approval events
    EventCommandApproval
    EventCommandApproved
    EventCommandDenied
    
    // System events
    EventError
    EventWarning
    EventInfo
)

func (e EventType) String() string {
    names := []string{
        "content_delta",
        "content_complete",
        "tool_call_start",
        "tool_call_progress",
        "tool_call_complete",
        "turn_start",
        "turn_complete",
        "turn_failed",
        "command_approval",
        "command_approved",
        "command_denied",
        "error",
        "warning",
        "info",
    }
    
    if int(e) < len(names) {
        return names[e]
    }
    return "unknown"
}
```

**Test Cases:**
```go
func TestEventType_String(t *testing.T) {
    tests := []struct {
        eventType EventType
        expected  string
    }{
        {EventContentDelta, "content_delta"},
        {EventToolCallStart, "tool_call_start"},
        {EventError, "error"},
    }
    
    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            assert.Equal(t, tt.expected, tt.eventType.String())
        })
    }
}
```

---

### FR-4.1.3: Event Data Types

**Description:** Define specific data types for different event categories.

**Acceptance Criteria:**
- Strongly-typed data structures for common events
- Type assertion helpers
- Clear documentation of expected data types

**Data Structures:**
```go
// ContentDeltaData contains incremental content
type ContentDeltaData struct {
    Content string `json:"content"`
    Role    string `json:"role"` // assistant, tool
}

// ToolCallData contains tool execution information
type ToolCallData struct {
    ToolName   string                 `json:"tool_name"`
    ToolID     string                 `json:"tool_id"`
    Parameters map[string]interface{} `json:"parameters"`
}

// ToolProgressData contains tool progress updates
type ToolProgressData struct {
    ToolID  string `json:"tool_id"`
    Status  string `json:"status"`
    Message string `json:"message"`
}

// ToolResultData contains tool execution results
type ToolResultData struct {
    ToolID   string `json:"tool_id"`
    ToolName string `json:"tool_name"`
    Result   string `json:"result"`
    Error    string `json:"error,omitempty"`
}

// ErrorData contains error information
type ErrorData struct {
    Message string `json:"message"`
    Code    string `json:"code"`
    Details string `json:"details,omitempty"`
}

// CommandApprovalData contains approval request info
type CommandApprovalData struct {
    Command     string `json:"command"`
    Class       string `json:"class"` // dangerous, interactive
    Reason      string `json:"reason"`
    WorkDir     string `json:"work_dir"`
    RequestID   string `json:"request_id"`
}
```

---

### FR-4.1.4: Event Emitter

**Description:** Implement the core EventEmitter with pub/sub pattern.

**Acceptance Criteria:**
- EventEmitter manages multiple subscribers
- Thread-safe with RWMutex
- Configurable channel buffer size
- Proper cleanup on close

**Data Structure:**
```go
type EventEmitter struct {
    subscribers map[string]chan Event
    mu          sync.RWMutex
    bufferSize  int
    closed      bool
}

func NewEventEmitter(bufferSize int) *EventEmitter {
    return &EventEmitter{
        subscribers: make(map[string]chan Event),
        bufferSize:  bufferSize,
    }
}
```

**Test Cases:**
```go
func TestNewEventEmitter(t *testing.T) {
    emitter := NewEventEmitter(100)
    
    assert.NotNil(t, emitter)
    assert.NotNil(t, emitter.subscribers)
    assert.Equal(t, 100, emitter.bufferSize)
}
```

---

### FR-4.1.5: Subscribe Method

**Description:** Allow consumers to subscribe to events.

**Acceptance Criteria:**
- Returns unique subscriber ID and event channel
- Creates buffered channel based on configured size
- Thread-safe subscription
- Returns error if emitter is closed

**API:**
```go
func (e *EventEmitter) Subscribe() (id string, events <-chan Event, err error)
```

**Test Cases:**
```go
func TestEventEmitter_Subscribe(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    id, events, err := emitter.Subscribe()
    require.NoError(t, err)
    
    assert.NotEmpty(t, id)
    assert.NotNil(t, events)
    
    // Channel should be buffered
    assert.Equal(t, 10, cap(events))
}

func TestEventEmitter_Subscribe_AfterClose(t *testing.T) {
    emitter := NewEventEmitter(10)
    emitter.Close()
    
    _, _, err := emitter.Subscribe()
    assert.Error(t, err)
}

func TestEventEmitter_Subscribe_MultipleSubscribers(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    id1, _, err := emitter.Subscribe()
    require.NoError(t, err)
    
    id2, _, err := emitter.Subscribe()
    require.NoError(t, err)
    
    // IDs should be unique
    assert.NotEqual(t, id1, id2)
}
```

---

### FR-4.1.6: Unsubscribe Method

**Description:** Allow consumers to unsubscribe and cleanup resources.

**Acceptance Criteria:**
- Removes subscriber from map
- Closes subscriber's event channel
- Thread-safe unsubscription
- No-op if subscriber doesn't exist

**API:**
```go
func (e *EventEmitter) Unsubscribe(id string)
```

**Test Cases:**
```go
func TestEventEmitter_Unsubscribe(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    id, events, _ := emitter.Subscribe()
    
    // Unsubscribe
    emitter.Unsubscribe(id)
    
    // Channel should be closed
    _, ok := <-events
    assert.False(t, ok)
}

func TestEventEmitter_Unsubscribe_NonExistent(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    // Should not panic
    emitter.Unsubscribe("non-existent-id")
}
```

---

### FR-4.1.7: Emit Method

**Description:** Publish events to all subscribers.

**Acceptance Criteria:**
- Sends event to all active subscribers
- Non-blocking with select + default
- Adds timestamp if not set
- Thread-safe emission
- Skips slow/blocked subscribers (fire-and-forget)

**API:**
```go
func (e *EventEmitter) Emit(event Event)
```

**Test Cases:**
```go
func TestEventEmitter_Emit(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    _, events, _ := emitter.Subscribe()
    
    // Emit event
    emitter.Emit(Event{
        Type: EventContentDelta,
        Data: "test",
    })
    
    // Subscriber should receive event
    select {
    case event := <-events:
        assert.Equal(t, EventContentDelta, event.Type)
        assert.Equal(t, "test", event.Data)
        assert.False(t, event.Timestamp.IsZero())
    case <-time.After(100 * time.Millisecond):
        t.Fatal("timeout waiting for event")
    }
}

func TestEventEmitter_Emit_MultipleSubscribers(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    _, events1, _ := emitter.Subscribe()
    _, events2, _ := emitter.Subscribe()
    
    // Emit event
    emitter.Emit(Event{Type: EventInfo, Data: "broadcast"})
    
    // Both should receive
    event1 := <-events1
    event2 := <-events2
    
    assert.Equal(t, EventInfo, event1.Type)
    assert.Equal(t, EventInfo, event2.Type)
}

func TestEventEmitter_Emit_SlowSubscriber(t *testing.T) {
    emitter := NewEventEmitter(2) // Small buffer
    
    _, events, _ := emitter.Subscribe()
    
    // Fill the buffer
    emitter.Emit(Event{Type: EventInfo, Data: "1"})
    emitter.Emit(Event{Type: EventInfo, Data: "2"})
    
    // This should not block (fire-and-forget)
    done := make(chan bool)
    go func() {
        emitter.Emit(Event{Type: EventInfo, Data: "3"})
        done <- true
    }()
    
    select {
    case <-done:
        // Success - didn't block
    case <-time.After(100 * time.Millisecond):
        t.Fatal("Emit() blocked on slow subscriber")
    }
    
    // Drain events
    <-events
    <-events
}
```

---

### FR-4.1.8: Close Method

**Description:** Gracefully shut down the emitter and cleanup all subscribers.

**Acceptance Criteria:**
- Closes all subscriber channels
- Marks emitter as closed
- Thread-safe closure
- Idempotent (safe to call multiple times)
- Prevents new subscriptions after close

**API:**
```go
func (e *EventEmitter) Close()
```

**Test Cases:**
```go
func TestEventEmitter_Close(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    _, events1, _ := emitter.Subscribe()
    _, events2, _ := emitter.Subscribe()
    
    // Close emitter
    emitter.Close()
    
    // All channels should be closed
    _, ok1 := <-events1
    _, ok2 := <-events2
    
    assert.False(t, ok1)
    assert.False(t, ok2)
    
    // Cannot subscribe after close
    _, _, err := emitter.Subscribe()
    assert.Error(t, err)
}

func TestEventEmitter_Close_Idempotent(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    // Should not panic
    emitter.Close()
    emitter.Close()
    emitter.Close()
}
```

---

### FR-4.1.9: Event Ordering

**Description:** Ensure events are delivered in the order they are emitted.

**Acceptance Criteria:**
- Events received in same order as emitted
- Per-subscriber ordering guarantee
- No event reordering

**Test Cases:**
```go
func TestEventEmitter_EventOrdering(t *testing.T) {
    emitter := NewEventEmitter(100)
    
    _, events, _ := emitter.Subscribe()
    
    // Emit sequence
    for i := 0; i < 10; i++ {
        emitter.Emit(Event{
            Type: EventInfo,
            Data: i,
        })
    }
    
    // Receive in order
    for i := 0; i < 10; i++ {
        event := <-events
        assert.Equal(t, i, event.Data)
    }
}
```

---

### FR-4.1.10: Concurrent Safety

**Description:** Ensure thread-safe operation under concurrent access.

**Acceptance Criteria:**
- Safe concurrent Subscribe/Unsubscribe/Emit
- No data races (pass -race detector)
- No deadlocks
- Proper mutex usage

**Test Cases:**
```go
func TestEventEmitter_ConcurrentSubscribe(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    var wg sync.WaitGroup
    subscribers := 100
    
    for i := 0; i < subscribers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _, err := emitter.Subscribe()
            assert.NoError(t, err)
        }()
    }
    
    wg.Wait()
    
    // Should have all subscribers
    emitter.mu.RLock()
    count := len(emitter.subscribers)
    emitter.mu.RUnlock()
    
    assert.Equal(t, subscribers, count)
}

func TestEventEmitter_ConcurrentEmit(t *testing.T) {
    emitter := NewEventEmitter(100)
    
    _, events, _ := emitter.Subscribe()
    
    var wg sync.WaitGroup
    emitCount := 100
    
    for i := 0; i < emitCount; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            emitter.Emit(Event{
                Type: EventInfo,
                Data: n,
            })
        }(i)
    }
    
    wg.Wait()
    
    // Collect all events
    received := 0
    for received < emitCount {
        select {
        case <-events:
            received++
        case <-time.After(100 * time.Millisecond):
            t.Fatalf("only received %d/%d events", received, emitCount)
        }
    }
}

func TestEventEmitter_ConcurrentMixed(t *testing.T) {
    emitter := NewEventEmitter(50)
    
    var wg sync.WaitGroup
    
    // Concurrent subscribes
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            id, _, _ := emitter.Subscribe()
            time.Sleep(10 * time.Millisecond)
            emitter.Unsubscribe(id)
        }()
    }
    
    // Concurrent emits
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            emitter.Emit(Event{Type: EventInfo})
        }()
    }
    
    wg.Wait()
}
```

---

## Non-Functional Requirements

### NFR-4.1.1: Performance

**Requirement:** Event emission should be fast and non-blocking.

**Acceptance Criteria:**
- Emit() completes in < 1ms (fire-and-forget)
- Support 1000+ events per second
- Support 100+ concurrent subscribers
- Low memory overhead per subscriber (~1KB)

**Test:**
```go
func BenchmarkEventEmitter_Emit(b *testing.B) {
    emitter := NewEventEmitter(100)
    emitter.Subscribe() // One subscriber
    
    event := Event{Type: EventInfo, Data: "test"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        emitter.Emit(event)
    }
}

func BenchmarkEventEmitter_EmitMultipleSubscribers(b *testing.B) {
    emitter := NewEventEmitter(100)
    
    // 10 subscribers
    for i := 0; i < 10; i++ {
        emitter.Subscribe()
    }
    
    event := Event{Type: EventInfo, Data: "test"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        emitter.Emit(event)
    }
}
```

---

### NFR-4.1.2: Memory Safety

**Requirement:** No memory leaks from unclosed subscriptions.

**Acceptance Criteria:**
- Unsubscribe properly cleans up resources
- Close() cleans up all subscribers
- No goroutine leaks
- Channels are properly closed

**Test:**
```go
func TestEventEmitter_NoMemoryLeaks(t *testing.T) {
    emitter := NewEventEmitter(10)
    
    // Create and cleanup many subscribers
    for i := 0; i < 1000; i++ {
        id, _, _ := emitter.Subscribe()
        emitter.Unsubscribe(id)
    }
    
    emitter.mu.RLock()
    count := len(emitter.subscribers)
    emitter.mu.RUnlock()
    
    // Should be empty
    assert.Equal(t, 0, count)
}
```

---

### NFR-4.1.3: Reliability

**Requirement:** System should handle edge cases gracefully.

**Acceptance Criteria:**
- No panics on nil data
- Safe to call methods after Close()
- Handle subscriber channel closure gracefully
- Proper error handling

---

## Technical Design

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Agent / Turn                          │
│                 (Event Producers)                       │
└────────┬────────────────────┬────────────────┬─────────┘
         │                    │                │
         │    Emit Events     │                │
         ▼                    ▼                ▼
┌─────────────────────────────────────────────────────────┐
│                   EventEmitter                          │
│              (Central Event Hub)                        │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│   │ Sub 1 ch │  │ Sub 2 ch │  │ Sub N ch │            │
│   └──────────┘  └──────────┘  └──────────┘            │
└────────┬────────────────────┬────────────────┬─────────┘
         │                    │                │
         │ Receive Events     │                │
         ▼                    ▼                ▼
┌─────────────┐      ┌──────────────┐  ┌──────────────┐
│     TUI     │      │  Web Server  │  │   Logger     │
│ (Subscriber)│      │ (Subscriber) │  │ (Subscriber) │
└─────────────┘      └──────────────┘  └──────────────┘
```

### Thread Safety Model

```go
// EventEmitter protects subscribers map with RWMutex
type EventEmitter struct {
    subscribers map[string]chan Event
    mu          sync.RWMutex  // Protects subscribers map
    bufferSize  int
    closed      bool           // Atomic flag
}

// Subscribe: Write lock
func (e *EventEmitter) Subscribe() {
    e.mu.Lock()
    defer e.mu.Unlock()
    // Add subscriber
}

// Unsubscribe: Write lock
func (e *EventEmitter) Unsubscribe() {
    e.mu.Lock()
    defer e.mu.Unlock()
    // Remove subscriber
}

// Emit: Read lock (allows concurrent emissions)
func (e *EventEmitter) Emit() {
    e.mu.RLock()
    defer e.mu.RUnlock()
    // Send to all subscribers
}
```

### Fire-and-Forget Pattern

```go
func (e *EventEmitter) Emit(event Event) {
    // Set timestamp
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    
    e.mu.RLock()
    defer e.mu.RUnlock()
    
    // Fan-out to all subscribers (non-blocking)
    for _, ch := range e.subscribers {
        select {
        case ch <- event:
            // Successfully sent
        default:
            // Subscriber slow/blocked, drop event
            // This prevents one slow subscriber from blocking others
        }
    }
}
```

---

## Dependencies

### Standard Library
- `sync` - RWMutex for thread safety
- `time` - Timestamp tracking
- `encoding/json` - JSON serialization
- `github.com/google/uuid` - Unique subscriber IDs

---

## Testing Strategy

### Unit Tests (>90% coverage)
- [x] Event structure and serialization
- [x] EventType String() method
- [x] EventEmitter creation
- [x] Subscribe/Unsubscribe lifecycle
- [x] Emit to single subscriber
- [x] Emit to multiple subscribers
- [x] Event ordering
- [x] Thread safety (concurrent operations)
- [x] Memory cleanup
- [x] Edge cases (close, nil data, etc.)

### Concurrency Tests
- [x] Concurrent Subscribe
- [x] Concurrent Unsubscribe
- [x] Concurrent Emit
- [x] Mixed concurrent operations
- [x] Race detector clean

### Benchmark Tests
- [x] Emit performance
- [x] Multi-subscriber emit
- [x] Subscribe/Unsubscribe overhead

---

## Implementation Plan

### Step 1: Define Types (30 minutes)
- Define Event struct
- Define EventType constants
- Add String() method for EventType
- Define event data types

### Step 2: Implement EventEmitter (1 hour)
- Implement NewEventEmitter()
- Add basic structure with mutex
- Implement buffer size configuration

### Step 3: Subscribe/Unsubscribe (1.5 hours)
- Implement Subscribe() with UUID generation
- Implement Unsubscribe() with cleanup
- Add subscriber map management
- Write subscribe/unsubscribe tests

### Step 4: Emit Implementation (1.5 hours)
- Implement Emit() with fan-out
- Add timestamp auto-setting
- Implement fire-and-forget pattern
- Write emit tests

### Step 5: Close and Cleanup (1 hour)
- Implement Close() method
- Add closed flag check
- Clean up all subscribers
- Write cleanup tests

### Step 6: Concurrency Tests (2 hours)
- Write concurrent subscribe tests
- Write concurrent emit tests
- Write mixed operation tests
- Run race detector

### Step 7: Edge Cases (1 hour)
- Test after-close behavior
- Test nil data handling
- Test empty subscriber list
- Test double close

### Step 8: Documentation (1 hour)
- Add godoc comments
- Create usage examples
- Document patterns

### Step 9: Benchmarking (1 hour)
- Write performance benchmarks
- Optimize hot paths
- Document performance characteristics

---

## Usage Examples

### Basic Usage
```go
// Create emitter
emitter := NewEventEmitter(100) // 100-event buffer per subscriber
defer emitter.Close()

// Subscribe
id, events, err := emitter.Subscribe()
if err != nil {
    panic(err)
}
defer emitter.Unsubscribe(id)

// Listen for events
go func() {
    for event := range events {
        switch event.Type {
        case EventContentDelta:
            data := event.Data.(ContentDeltaData)
            fmt.Print(data.Content)
            
        case EventToolCallStart:
            data := event.Data.(ToolCallData)
            fmt.Printf("Executing tool: %s\n", data.ToolName)
            
        case EventError:
            data := event.Data.(ErrorData)
            fmt.Fprintf(os.Stderr, "Error: %s\n", data.Message)
        }
    }
}()

// Emit events
emitter.Emit(Event{
    Type: EventContentDelta,
    Data: ContentDeltaData{
        Content: "Hello, ",
        Role:    "assistant",
    },
})
```

### In Agent
```go
type Agent struct {
    emitter *EventEmitter
    // ... other fields
}

func (a *Agent) Execute(ctx context.Context, req Request) error {
    // Emit turn start
    a.emitter.Emit(Event{
        Type: EventTurnStart,
        Data: map[string]interface{}{
            "request": req.Input,
        },
    })
    
    // Stream LLM response
    for chunk := range llmStream {
        a.emitter.Emit(Event{
            Type: EventContentDelta,
            Data: ContentDeltaData{
                Content: chunk.Content,
                Role:    "assistant",
            },
        })
    }
    
    // Emit turn complete
    a.emitter.Emit(Event{
        Type: EventTurnComplete,
        Data: map[string]interface{}{
            "tokens_used": totalTokens,
        },
    })
    
    return nil
}
```

### Multiple Subscribers
```go
emitter := NewEventEmitter(100)

// TUI subscriber
tui_id, tui_events, _ := emitter.Subscribe()
go func() {
    for event := range tui_events {
        updateUI(event)
    }
}()

// Logger subscriber
logger_id, log_events, _ := emitter.Subscribe()
go func() {
    for event := range log_events {
        log.Printf("[%s] %v", event.Type, event.Data)
    }
}()

// Metrics subscriber
metrics_id, metrics_events, _ := emitter.Subscribe()
go func() {
    for event := range metrics_events {
        recordMetric(event)
    }
}()
```

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Memory leaks from unclosed subscriptions | High | Medium | Implement Close() cleanup, document proper usage |
| Slow subscribers block system | High | Medium | Fire-and-forget pattern, drop events for slow subscribers |
| Race conditions | High | Low | RWMutex protection, extensive race detector testing |
| Event ordering issues | Medium | Low | Single goroutine per subscriber, buffered channels |

---

## Success Metrics

- [ ] All tests passing with >90% coverage
- [ ] Race detector clean
- [ ] No memory leaks in tests
- [ ] Emit() < 1ms per call
- [ ] Supports 100+ concurrent subscribers
- [ ] All linters passing
- [ ] Godoc complete

---

## References

- [Core Module Spec](../core-module/spec.md) - Section 9: Event Streaming
- [ROADMAP](../core-module/ROADMAP.md) - Feature 4.1
- [Go sync package](https://pkg.go.dev/sync)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)

---

## Changelog

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-10-03 | 1.0 | Initial FRD creation | AI Agent |

---

**Status:** 🚧 Ready for Implementation  
**Next Steps:** Write tests (TDD), implement code, achieve DoD

