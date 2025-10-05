# FRD-CORE-0.3: Event Streaming Control (Backpressure Strategies)

**Status:** Draft
**Priority:** IMPORTANT - Blocking TUI Phase 3
**Created:** 2025-10-05
**Related:** [UI Modules ROADMAP Phase 0.3](../ui-modules/ROADMAP.md#03-event-streaming-control--important)

---

## Problem Statement

The current `EventEmitter` implementation in `internal/core/event.go` uses a fire-and-forget pattern with fixed buffer sizes. When consumers (like UI rendering) are slower than event producers (like LLM streaming), events are silently dropped. This causes:

1. **Lost Critical Events**: Approval requests, errors, and state transitions can be dropped
2. **UI State Corruption**: Missing events lead to incorrect UI state
3. **No Control**: Consumers cannot choose how to handle backpressure
4. **Hard to Debug**: Silent drops make issues difficult to diagnose

**Current Implementation (lines 195-203):**
```go
for _, ch := range e.subscribers {
    select {
    case ch <- event:
        // Successfully sent
    default:
        // Subscriber slow/blocked, drop event
        // This prevents one slow subscriber from blocking others
    }
}
```

---

## Requirements

### Functional Requirements

1. **Configurable Backpressure Modes**:
   - `BackpressureDrop`: Current behavior - drop events if buffer full (fire-and-forget)
   - `BackpressureBlock`: Block emitter until consumer ready (ensures delivery)
   - `BackpressureBuffer`: Dynamic buffer growth up to configurable limit

2. **Per-Emitter Configuration**:
   - Each EventEmitter instance can have different backpressure strategy
   - Buffer size configuration
   - Buffer growth limit (for BackpressureBuffer mode)

3. **Backward Compatibility**:
   - Existing `NewEventEmitter(bufferSize)` continues to work with `BackpressureDrop`
   - New `NewEventEmitterWithConfig()` for custom configuration

4. **Thread Safety**:
   - All backpressure modes must be thread-safe
   - Race detector must pass

### Non-Functional Requirements

1. **Performance**: Backpressure overhead <1μs per event
2. **Memory**: BackpressureBuffer must respect limits to prevent unbounded growth
3. **Complexity**: Keep cyclomatic complexity ≤15
4. **Testing**: ≥90% coverage with race detector

---

## Design

### Type Definitions

```go
// BackpressureMode defines how the emitter handles slow consumers
type BackpressureMode int

const (
    // BackpressureDrop drops events when subscriber buffer is full (fire-and-forget)
    // Best for: Non-critical events, high-throughput scenarios
    BackpressureDrop BackpressureMode = iota

    // BackpressureBlock blocks the emitter until subscriber is ready
    // Best for: Critical events that must be delivered (approvals, errors)
    BackpressureBlock

    // BackpressureBuffer uses dynamic buffer growth up to limit
    // Best for: Bursty workloads where temporary slowdowns are acceptable
    BackpressureBuffer
)

// EventEmitterConfig configures event emitter behavior
type EventEmitterConfig struct {
    // BufferSize is the initial channel buffer size per subscriber
    BufferSize int

    // BackpressureMode determines how to handle slow consumers
    BackpressureMode BackpressureMode

    // BufferLimit is the maximum buffer size for BackpressureBuffer mode
    // Ignored for other modes. Default: 10000 events
    BufferLimit int
}
```

### Updated EventEmitter Struct

```go
type EventEmitter struct {
    subscribers map[string]chan Event
    mu          sync.RWMutex
    bufferSize  int
    closed      bool

    // New fields
    config      EventEmitterConfig
    buffers     map[string][]Event // Dynamic buffers for BackpressureBuffer mode
    bufferMu    sync.Mutex         // Protects buffers map
}
```

### Constructor Functions

```go
// NewEventEmitter creates an EventEmitter with default config (BackpressureDrop)
// Maintains backward compatibility with existing code
func NewEventEmitter(bufferSize int) *EventEmitter {
    return NewEventEmitterWithConfig(EventEmitterConfig{
        BufferSize:       bufferSize,
        BackpressureMode: BackpressureDrop,
        BufferLimit:      0,
    })
}

// NewEventEmitterWithConfig creates an EventEmitter with custom configuration
func NewEventEmitterWithConfig(config EventEmitterConfig) *EventEmitter {
    if config.BufferLimit == 0 {
        config.BufferLimit = 10000 // Default limit
    }

    emitter := &EventEmitter{
        subscribers: make(map[string]chan Event),
        bufferSize:  config.BufferSize,
        config:      config,
        closed:      false,
    }

    if config.BackpressureMode == BackpressureBuffer {
        emitter.buffers = make(map[string][]Event)
    }

    return emitter
}
```

### Emit() Implementation

```go
func (e *EventEmitter) Emit(event Event) {
    // Set timestamp if not provided
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }

    e.mu.RLock()
    defer e.mu.RUnlock()

    // Don't emit if closed
    if e.closed {
        return
    }

    // Fan-out to all subscribers using configured backpressure strategy
    for id, ch := range e.subscribers {
        switch e.config.BackpressureMode {
        case BackpressureDrop:
            e.emitDrop(ch, event)
        case BackpressureBlock:
            e.emitBlock(ch, event)
        case BackpressureBuffer:
            e.emitBuffer(id, ch, event)
        }
    }
}

// emitDrop implements fire-and-forget (current behavior)
func (e *EventEmitter) emitDrop(ch chan Event, event Event) {
    select {
    case ch <- event:
        // Successfully sent
    default:
        // Subscriber slow/blocked, drop event
    }
}

// emitBlock implements blocking send (ensures delivery)
func (e *EventEmitter) emitBlock(ch chan Event, event Event) {
    ch <- event // Blocks until consumer ready
}

// emitBuffer implements dynamic buffering
func (e *EventEmitter) emitBuffer(id string, ch chan Event, event Event) {
    select {
    case ch <- event:
        // Successfully sent to channel
        e.tryFlushBuffer(id, ch)
    default:
        // Channel full, add to dynamic buffer
        e.addToBuffer(id, event)
    }
}

// addToBuffer adds event to dynamic buffer (if under limit)
func (e *EventEmitter) addToBuffer(id string, event Event) {
    e.bufferMu.Lock()
    defer e.bufferMu.Unlock()

    buffer := e.buffers[id]
    if len(buffer) < e.config.BufferLimit {
        e.buffers[id] = append(buffer, event)
    }
    // If over limit, drop event (similar to BackpressureDrop)
}

// tryFlushBuffer attempts to flush buffered events to channel
func (e *EventEmitter) tryFlushBuffer(id string, ch chan Event) {
    e.bufferMu.Lock()
    defer e.bufferMu.Unlock()

    buffer := e.buffers[id]
    if len(buffer) == 0 {
        return
    }

    // Try to send buffered events (non-blocking)
    sent := 0
    for i, event := range buffer {
        select {
        case ch <- event:
            sent = i + 1
        default:
            break
        }
    }

    // Remove sent events from buffer
    if sent > 0 {
        e.buffers[id] = buffer[sent:]
    }
}
```

### Subscribe/Unsubscribe Updates

```go
func (e *EventEmitter) Subscribe() (id string, events <-chan Event, err error) {
    e.mu.Lock()
    defer e.mu.Unlock()

    if e.closed {
        return "", nil, errors.New("emitter is closed")
    }

    id = uuid.New().String()
    ch := make(chan Event, e.bufferSize)
    e.subscribers[id] = ch

    // Initialize buffer for BackpressureBuffer mode
    if e.config.BackpressureMode == BackpressureBuffer {
        e.bufferMu.Lock()
        e.buffers[id] = make([]Event, 0)
        e.bufferMu.Unlock()
    }

    return id, ch, nil
}

func (e *EventEmitter) Unsubscribe(id string) {
    e.mu.Lock()
    defer e.mu.Unlock()

    if ch, exists := e.subscribers[id]; exists {
        close(ch)
        delete(e.subscribers, id)

        // Clean up buffer
        if e.config.BackpressureMode == BackpressureBuffer {
            e.bufferMu.Lock()
            delete(e.buffers, id)
            e.bufferMu.Unlock()
        }
    }
}
```

---

## Implementation Plan

### Step 1: Types and Configuration
- Add `BackpressureMode` enum
- Add `EventEmitterConfig` struct
- Update `EventEmitter` struct with new fields
- Write tests for config validation

### Step 2: Constructor Functions
- Implement `NewEventEmitterWithConfig()`
- Update `NewEventEmitter()` to use default config
- Write tests for both constructors

### Step 3: Backpressure Implementations
- Implement `emitDrop()` (extract current behavior)
- Implement `emitBlock()` (blocking send)
- Implement `emitBuffer()` with dynamic buffering
- Implement `addToBuffer()` and `tryFlushBuffer()`
- Write tests for each mode

### Step 4: Update Core Methods
- Update `Emit()` to route to correct strategy
- Update `Subscribe()` to initialize buffers
- Update `Unsubscribe()` to clean up buffers
- Update `Close()` to clean up buffers
- Write integration tests

### Step 5: Testing and Quality
- Run all tests with race detector
- Analyze cyclomatic complexity
- Benchmark each backpressure mode
- Update documentation

---

## Testing Strategy

### Unit Tests

1. **Config Tests**:
   - Default config values
   - Config validation
   - BufferLimit defaults

2. **BackpressureDrop Tests**:
   - Events dropped when buffer full
   - Fast consumers receive all events
   - No blocking behavior

3. **BackpressureBlock Tests**:
   - Emitter blocks when consumer slow
   - All events delivered eventually
   - Blocking behavior correct

4. **BackpressureBuffer Tests**:
   - Events buffered when channel full
   - Buffer flushed when channel available
   - Buffer respects limit
   - Events dropped when over limit

5. **Concurrency Tests**:
   - Multiple subscribers with different speeds
   - Subscribe/unsubscribe during emission
   - Close during emission
   - Race detector clean

### Benchmark Tests

```go
func BenchmarkEmitDrop(b *testing.B)
func BenchmarkEmitBlock(b *testing.B)
func BenchmarkEmitBuffer(b *testing.B)
```

### Expected Coverage

- Overall: ≥90%
- Critical paths (Emit, backpressure strategies): 100%

---

## Mode Selection Guidance

| Mode | Use Case | Pros | Cons |
|------|----------|------|------|
| **Drop** | High-throughput, non-critical events | Fast, no blocking | Silent data loss |
| **Block** | Critical events (approvals, errors) | Guaranteed delivery | Can slow system |
| **Buffer** | Bursty workloads, temporary slowdowns | Handles bursts, eventual delivery | Memory usage, complexity |

**Recommendations:**
- **TUI**: Use `BackpressureBuffer` with reasonable limit (1000-5000 events)
- **Exec**: Use `BackpressureDrop` for content deltas, `BackpressureBlock` for approvals
- **Core internal**: Use `BackpressureDrop` (current behavior)

---

## Migration Path

**Existing Code (no changes needed):**
```go
emitter := core.NewEventEmitter(100)
```

**New TUI Code:**
```go
emitter := core.NewEventEmitterWithConfig(core.EventEmitterConfig{
    BufferSize:       100,
    BackpressureMode: core.BackpressureBuffer,
    BufferLimit:      5000,
})
```

---

## Performance Targets

- `BackpressureDrop`: <100ns per event (same as current)
- `BackpressureBlock`: <200ns per event
- `BackpressureBuffer`: <500ns per event (includes buffer check)
- Memory overhead: <100 bytes per subscriber (excluding buffered events)

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| BackpressureBlock deadlock | HIGH | Document usage, recommend for critical events only |
| BackpressureBuffer unbounded growth | MEDIUM | Enforce BufferLimit, drop after limit |
| Performance regression | LOW | Benchmark all modes, optimize hot paths |
| Breaking changes | LOW | Maintain backward compatibility via default config |

---

## Success Criteria

- [x] All three backpressure modes implemented
- [ ] Backward compatible with existing code
- [ ] Tests passing with ≥90% coverage
- [ ] Race detector clean
- [ ] Benchmarks show acceptable overhead
- [ ] Complexity ≤15 for all functions
- [ ] Documentation with mode selection guidance
- [ ] TUI can use BackpressureBuffer mode without drops

---

## References

- [internal/core/event.go](../../internal/core/event.go) - Current implementation
- [UI Modules ROADMAP Phase 0.3](../ui-modules/ROADMAP.md#03-event-streaming-control--important)
- [Go Channel Patterns](https://go.dev/blog/pipelines)
- [Backpressure in Streaming Systems](https://mechanical-sympathy.blogspot.com/2012/05/apply-back-pressure-when-overloaded.html)
