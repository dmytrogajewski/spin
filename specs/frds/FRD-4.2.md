# FRD-4.2: Stream Management

**Feature ID:** 4.2  
**Feature Name:** Stream Management  
**Priority:** P1 (Critical)  
**Estimated Effort:** 8 hours  
**Actual Effort:** ~6 hours  
**Status:** ✅ Complete  
**Phase:** 4 - Event System

---

## Overview

Implement stream handling, buffering, and event type management for LLM streaming responses. This system provides the infrastructure for handling real-time data streams from LLM providers with proper backpressure management, smart buffering, and error propagation.

## Rationale

Stream management is critical for:
- **Real-time LLM responses**: Process tokens as they arrive without waiting for completion
- **Backpressure handling**: Prevent memory overflow when consumer is slower than producer
- **Buffer optimization**: Balance latency vs throughput with smart buffering
- **Error propagation**: Gracefully handle stream errors and closures
- **Resource management**: Proper cleanup of stream resources
- **Performance**: Minimize copying and allocations in hot paths

## Definition of Ready (DoR)

- [x] Feature 4.1 completed (Event Infrastructure)
- [x] Streaming requirements defined
- [x] Buffer sizing determined (based on typical LLM response patterns)

## Definition of Done (DoD)

- [ ] `stream/stream.go` implemented with Stream type
- [ ] `stream/buffer.go` with smart buffering logic
- [ ] `stream/types.go` with stream event types
- [ ] Stream event handling (read, write, close)
- [ ] Backpressure management (blocking/dropping strategies)
- [ ] Buffer overflow handling (capacity limits)
- [ ] Stream completion detection
- [ ] Error propagation in streams
- [ ] Unit tests for stream handling (>90% coverage)
- [ ] Backpressure tests
- [ ] Buffer tests
- [ ] Stream lifecycle tests
- [ ] Godoc comments for all exported symbols
- [ ] All linters passing
- [ ] Cyclomatic complexity ≤15 for all functions
- [ ] Race detector clean
- [ ] FRD-4.2 marked complete in ROADMAP

---

## Functional Requirements

### FR-4.2.1: Stream Event Types

**Description:** Define stream-specific event types for LLM streaming.

**Acceptance Criteria:**
- StreamEvent struct for stream data chunks
- ChunkType enum for different chunk types
- Metadata fields (sequence, timing, size)
- JSON serialization support

**Data Structure:**
```go
// StreamEvent represents a chunk of data in a stream
type StreamEvent struct {
    Sequence  int64       `json:"sequence"`   // Sequence number for ordering
    Type      ChunkType   `json:"type"`       // Type of chunk
    Data      []byte      `json:"data"`       // Actual data payload
    Metadata  Metadata    `json:"metadata"`   // Additional metadata
    Timestamp time.Time   `json:"timestamp"`  // When chunk was received
    Error     error       `json:"error,omitempty"` // Error if any
}

// ChunkType identifies the type of stream chunk
type ChunkType int

const (
    ChunkContent ChunkType = iota  // Content chunk from LLM
    ChunkToolCall                   // Tool call chunk
    ChunkFunctionCall              // Function call chunk
    ChunkDelta                     // Delta update
    ChunkComplete                  // Stream completion marker
    ChunkError                     // Error chunk
)

// Metadata contains chunk metadata
type Metadata struct {
    Model        string            `json:"model,omitempty"`
    Provider     string            `json:"provider,omitempty"`
    TokenCount   int               `json:"token_count,omitempty"`
    FinishReason string            `json:"finish_reason,omitempty"`
    Custom       map[string]string `json:"custom,omitempty"`
}

func (c ChunkType) String() string {
    names := []string{
        "content",
        "tool_call",
        "function_call",
        "delta",
        "complete",
        "error",
    }
    if int(c) < len(names) {
        return names[c]
    }
    return "unknown"
}
```

**Test Cases:**
```go
func TestStreamEvent_Creation(t *testing.T) {
    event := StreamEvent{
        Sequence:  1,
        Type:      ChunkContent,
        Data:      []byte("test content"),
        Timestamp: time.Now(),
    }
    
    assert.Equal(t, int64(1), event.Sequence)
    assert.Equal(t, ChunkContent, event.Type)
    assert.Equal(t, "test content", string(event.Data))
}

func TestChunkType_String(t *testing.T) {
    assert.Equal(t, "content", ChunkContent.String())
    assert.Equal(t, "error", ChunkError.String())
}
```

---

### FR-4.2.2: Stream Buffer

**Description:** Implement smart buffering for stream data with capacity management.

**Acceptance Criteria:**
- Ring buffer implementation for efficiency
- Configurable capacity (default 4096 bytes)
- Write operations with overflow detection
- Read operations with underflow handling
- Clear/Reset functionality
- Thread-safe operations

**Data Structure:**
```go
// Buffer provides a thread-safe ring buffer for stream data
type Buffer struct {
    data     []byte
    capacity int
    size     int
    readPos  int
    writePos int
    mu       sync.RWMutex
    full     bool
}

// NewBuffer creates a new buffer with specified capacity
func NewBuffer(capacity int) *Buffer {
    if capacity <= 0 {
        capacity = DefaultBufferCapacity
    }
    return &Buffer{
        data:     make([]byte, capacity),
        capacity: capacity,
    }
}

// Write writes data to the buffer
// Returns number of bytes written and error if buffer is full
func (b *Buffer) Write(data []byte) (int, error)

// Read reads up to len(p) bytes from buffer
// Returns number of bytes read
func (b *Buffer) Read(p []byte) (int, error)

// Available returns number of bytes available to read
func (b *Buffer) Available() int

// Capacity returns total buffer capacity
func (b *Buffer) Capacity() int

// Reset clears the buffer
func (b *Buffer) Reset()

// IsFull returns true if buffer is at capacity
func (b *Buffer) IsFull() bool

// IsEmpty returns true if buffer is empty
func (b *Buffer) IsEmpty() bool
```

**Constants:**
```go
const (
    DefaultBufferCapacity = 4096  // 4KB default
    MinBufferCapacity     = 256   // Minimum 256 bytes
    MaxBufferCapacity     = 1 << 20 // 1MB maximum
)

var (
    ErrBufferFull  = errors.New("buffer is full")
    ErrBufferEmpty = errors.New("buffer is empty")
)
```

**Test Cases:**
```go
func TestBuffer_WriteRead(t *testing.T) {
    buf := NewBuffer(1024)
    
    // Write data
    data := []byte("hello world")
    n, err := buf.Write(data)
    require.NoError(t, err)
    assert.Equal(t, len(data), n)
    
    // Read data
    read := make([]byte, 1024)
    n, err = buf.Read(read)
    require.NoError(t, err)
    assert.Equal(t, len(data), n)
    assert.Equal(t, data, read[:n])
}

func TestBuffer_Overflow(t *testing.T) {
    buf := NewBuffer(10)
    
    // Fill buffer
    data := make([]byte, 15)
    n, err := buf.Write(data)
    
    // Should detect overflow
    assert.Error(t, err)
    assert.Equal(t, ErrBufferFull, err)
    assert.Equal(t, 10, n) // Only capacity bytes written
}

func TestBuffer_Concurrent(t *testing.T) {
    buf := NewBuffer(1024)
    
    var wg sync.WaitGroup
    wg.Add(2)
    
    // Concurrent writes
    go func() {
        defer wg.Done()
        for i := 0; i < 100; i++ {
            buf.Write([]byte("test"))
        }
    }()
    
    // Concurrent reads
    go func() {
        defer wg.Done()
        read := make([]byte, 4)
        for i := 0; i < 100; i++ {
            buf.Read(read)
        }
    }()
    
    wg.Wait()
}
```

---

### FR-4.2.3: Stream Handler

**Description:** Main stream handling logic with lifecycle management.

**Acceptance Criteria:**
- Stream struct for managing data flow
- Send/Receive operations
- Stream state tracking (open/closed)
- Graceful shutdown with drain
- Context cancellation support
- Timeout handling

**Data Structure:**
```go
// Stream manages a data stream with buffering and flow control
type Stream struct {
    id       string
    buffer   *Buffer
    input    chan StreamEvent
    output   chan StreamEvent
    errors   chan error
    done     chan struct{}
    state    StreamState
    mu       sync.RWMutex
    ctx      context.Context
    cancel   context.CancelFunc
    sequence int64
    
    // Configuration
    config StreamConfig
}

// StreamConfig configures stream behavior
type StreamConfig struct {
    BufferSize       int                  // Buffer capacity
    InputBuffer      int                  // Input channel buffer size
    OutputBuffer     int                  // Output channel buffer size
    Backpressure     BackpressureStrategy // How to handle slow consumers
    ErrorHandler     ErrorHandler         // Error handling callback
    MaxSequenceSkip  int                  // Max allowed sequence gap
}

// StreamState represents stream lifecycle state
type StreamState int

const (
    StreamStateOpen StreamState = iota
    StreamStateDraining
    StreamStateClosed
)

// BackpressureStrategy defines how to handle backpressure
type BackpressureStrategy int

const (
    BackpressureBlock BackpressureStrategy = iota  // Block until consumer ready
    BackpressureDrop                                // Drop oldest events
    BackpressureError                              // Return error
)

// ErrorHandler is called when stream error occurs
type ErrorHandler func(err error)

// NewStream creates a new stream
func NewStream(id string, config StreamConfig) *Stream

// Send sends an event to the stream
func (s *Stream) Send(ctx context.Context, event StreamEvent) error

// Receive returns a channel for receiving events
func (s *Stream) Receive() <-chan StreamEvent

// Errors returns a channel for receiving errors
func (s *Stream) Errors() <-chan error

// Close closes the stream gracefully
func (s *Stream) Close() error

// State returns current stream state
func (s *Stream) State() StreamState

// IsOpen returns true if stream is accepting input
func (s *Stream) IsOpen() bool
```

**Default Configuration:**
```go
// DefaultStreamConfig returns default configuration
func DefaultStreamConfig() StreamConfig {
    return StreamConfig{
        BufferSize:      DefaultBufferCapacity,
        InputBuffer:     100,
        OutputBuffer:    100,
        Backpressure:    BackpressureBlock,
        ErrorHandler:    func(err error) {}, // no-op
        MaxSequenceSkip: 10,
    }
}
```

**Test Cases:**
```go
func TestStream_SendReceive(t *testing.T) {
    stream := NewStream("test", DefaultStreamConfig())
    defer stream.Close()
    
    // Send event
    event := StreamEvent{
        Type: ChunkContent,
        Data: []byte("test"),
    }
    
    err := stream.Send(context.Background(), event)
    require.NoError(t, err)
    
    // Receive event
    select {
    case received := <-stream.Receive():
        assert.Equal(t, ChunkContent, received.Type)
        assert.Equal(t, "test", string(received.Data))
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestStream_Backpressure(t *testing.T) {
    config := DefaultStreamConfig()
    config.Backpressure = BackpressureDrop
    config.OutputBuffer = 1
    
    stream := NewStream("test", config)
    defer stream.Close()
    
    // Fill buffer
    for i := 0; i < 10; i++ {
        event := StreamEvent{
            Type: ChunkContent,
            Data: []byte(fmt.Sprintf("event %d", i)),
        }
        stream.Send(context.Background(), event)
    }
    
    // Should have dropped some events
    count := 0
    timeout := time.After(100 * time.Millisecond)
    for {
        select {
        case <-stream.Receive():
            count++
        case <-timeout:
            assert.Less(t, count, 10, "should have dropped events")
            return
        }
    }
}

func TestStream_ContextCancellation(t *testing.T) {
    stream := NewStream("test", DefaultStreamConfig())
    defer stream.Close()
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    event := StreamEvent{Type: ChunkContent, Data: []byte("test")}
    err := stream.Send(ctx, event)
    
    assert.Error(t, err)
    assert.Equal(t, context.Canceled, err)
}

func TestStream_GracefulClose(t *testing.T) {
    stream := NewStream("test", DefaultStreamConfig())
    
    // Send events
    for i := 0; i < 5; i++ {
        event := StreamEvent{
            Type: ChunkContent,
            Data: []byte(fmt.Sprintf("event %d", i)),
        }
        stream.Send(context.Background(), event)
    }
    
    // Close stream
    err := stream.Close()
    require.NoError(t, err)
    
    // Drain remaining events
    count := 0
    for range stream.Receive() {
        count++
    }
    
    assert.Equal(t, 5, count, "should drain all events")
}
```

---

### FR-4.2.4: Backpressure Management

**Description:** Implement strategies for handling backpressure when consumers are slower than producers.

**Acceptance Criteria:**
- Block strategy: wait for consumer
- Drop strategy: discard oldest events
- Error strategy: return error to producer
- Configurable per stream
- Metrics tracking (dropped events, blocked duration)

**Implementation:**
```go
// handleBackpressure applies the configured backpressure strategy
func (s *Stream) handleBackpressure(ctx context.Context, event StreamEvent) error {
    s.mu.RLock()
    strategy := s.config.Backpressure
    s.mu.RUnlock()
    
    switch strategy {
    case BackpressureBlock:
        return s.sendBlocking(ctx, event)
        
    case BackpressureDrop:
        return s.sendDropping(event)
        
    case BackpressureError:
        return s.sendWithError(event)
        
    default:
        return ErrInvalidStrategy
    }
}

func (s *Stream) sendBlocking(ctx context.Context, event StreamEvent) error {
    select {
    case s.output <- event:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    case <-s.done:
        return ErrStreamClosed
    }
}

func (s *Stream) sendDropping(event StreamEvent) error {
    select {
    case s.output <- event:
        return nil
    default:
        // Drop oldest event
        select {
        case <-s.output:
        default:
        }
        // Try again
        select {
        case s.output <- event:
            return nil
        default:
            return ErrEventDropped
        }
    }
}

func (s *Stream) sendWithError(event StreamEvent) error {
    select {
    case s.output <- event:
        return nil
    default:
        return ErrBackpressure
    }
}
```

**Error Types:**
```go
var (
    ErrStreamClosed     = errors.New("stream is closed")
    ErrInvalidStrategy  = errors.New("invalid backpressure strategy")
    ErrEventDropped     = errors.New("event dropped due to backpressure")
    ErrBackpressure     = errors.New("backpressure detected")
    ErrSequenceGap      = errors.New("sequence gap detected")
)
```

**Test Cases:**
```go
func TestBackpressure_Block(t *testing.T) {
    config := DefaultStreamConfig()
    config.Backpressure = BackpressureBlock
    config.OutputBuffer = 1
    
    stream := NewStream("test", config)
    defer stream.Close()
    
    // Fill buffer
    event1 := StreamEvent{Type: ChunkContent, Data: []byte("1")}
    err := stream.Send(context.Background(), event1)
    require.NoError(t, err)
    
    // This should block
    done := make(chan struct{})
    go func() {
        event2 := StreamEvent{Type: ChunkContent, Data: []byte("2")}
        stream.Send(context.Background(), event2)
        close(done)
    }()
    
    // Should not complete immediately
    select {
    case <-done:
        t.Fatal("should block")
    case <-time.After(50 * time.Millisecond):
        // Expected
    }
    
    // Read event to unblock
    <-stream.Receive()
    
    // Now should complete
    select {
    case <-done:
        // Expected
    case <-time.After(time.Second):
        t.Fatal("should unblock")
    }
}

func TestBackpressure_Drop(t *testing.T) {
    config := DefaultStreamConfig()
    config.Backpressure = BackpressureDrop
    config.OutputBuffer = 2
    
    stream := NewStream("test", config)
    defer stream.Close()
    
    // Send many events quickly
    for i := 0; i < 10; i++ {
        event := StreamEvent{
            Type: ChunkContent,
            Data: []byte(fmt.Sprintf("%d", i)),
        }
        stream.Send(context.Background(), event)
    }
    
    // Verify some events were dropped
    received := 0
    timeout := time.After(100 * time.Millisecond)
    for {
        select {
        case <-stream.Receive():
            received++
        case <-timeout:
            assert.Less(t, received, 10)
            return
        }
    }
}
```

---

### FR-4.2.5: Error Propagation

**Description:** Proper error handling and propagation throughout the stream lifecycle.

**Acceptance Criteria:**
- Separate error channel
- Error does not stop stream (unless fatal)
- Error context preservation
- Error recovery strategies
- Error callbacks

**Implementation:**
```go
// propagateError sends error to error channel and optionally to handler
func (s *Stream) propagateError(err error) {
    if err == nil {
        return
    }
    
    // Call error handler if configured
    s.mu.RLock()
    handler := s.config.ErrorHandler
    s.mu.RUnlock()
    
    if handler != nil {
        handler(err)
    }
    
    // Send to error channel (non-blocking)
    select {
    case s.errors <- err:
    default:
        // Error channel full, log or discard
    }
}

// RecoverableError wraps errors that can be recovered from
type RecoverableError struct {
    Err      error
    Recoverable bool
}

func (e *RecoverableError) Error() string {
    return fmt.Sprintf("stream error (recoverable=%v): %v", e.Recoverable, e.Err)
}

func (e *RecoverableError) Unwrap() error {
    return e.Err
}

// IsRecoverable checks if error can be recovered from
func IsRecoverable(err error) bool {
    var re *RecoverableError
    if errors.As(err, &re) {
        return re.Recoverable
    }
    return false
}
```

**Test Cases:**
```go
func TestStream_ErrorPropagation(t *testing.T) {
    errorReceived := make(chan error, 1)
    config := DefaultStreamConfig()
    config.ErrorHandler = func(err error) {
        errorReceived <- err
    }
    
    stream := NewStream("test", config)
    defer stream.Close()
    
    // Simulate error
    testErr := errors.New("test error")
    stream.propagateError(testErr)
    
    // Should receive error in both handler and channel
    select {
    case err := <-errorReceived:
        assert.Equal(t, testErr, err)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for error")
    }
    
    select {
    case err := <-stream.Errors():
        assert.Equal(t, testErr, err)
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for error channel")
    }
}

func TestRecoverableError(t *testing.T) {
    err := &RecoverableError{
        Err:         errors.New("network timeout"),
        Recoverable: true,
    }
    
    assert.True(t, IsRecoverable(err))
    assert.Contains(t, err.Error(), "recoverable=true")
}
```

---

## Non-Functional Requirements

### NFR-4.2.1: Performance
- Stream operations complete in <1ms under normal load
- Zero-copy operations where possible
- Minimal allocations in hot paths
- Buffer reuse to reduce GC pressure

### NFR-4.2.2: Concurrency
- All operations thread-safe
- No race conditions (verified with `go test -race`)
- Proper mutex usage (RWMutex for read-heavy operations)
- Deadlock-free design

### NFR-4.2.3: Resource Management
- Proper cleanup on stream close
- No goroutine leaks
- No channel leaks
- Bounded memory usage

### NFR-4.2.4: Observability
- Metrics for throughput, latency, drops
- Debug logging for stream lifecycle
- Trace points for performance analysis

---

## Implementation Plan

### Phase 1: Types and Buffer (2 hours)
1. Implement `stream/types.go` with event types
2. Implement `stream/buffer.go` with ring buffer
3. Write unit tests for buffer operations
4. Verify thread safety with race detector

### Phase 2: Stream Handler (3 hours)
1. Implement `stream/stream.go` with core Stream type
2. Implement Send/Receive operations
3. Implement lifecycle management (open/close)
4. Write unit tests for stream operations

### Phase 3: Backpressure (2 hours)
1. Implement backpressure strategies
2. Add configuration support
3. Write tests for each strategy
4. Verify behavior under load

### Phase 4: Error Handling (1 hour)
1. Implement error propagation
2. Add error recovery mechanisms
3. Write error handling tests
4. Verify error contexts

---

## Test Strategy

### Unit Tests
- Buffer operations (write, read, overflow, underflow)
- Stream lifecycle (create, send, receive, close)
- Backpressure strategies (block, drop, error)
- Error propagation
- Concurrent access

### Integration Tests
- Stream with real event emitter
- Multi-stream coordination
- Long-running streams
- Error recovery scenarios

### Performance Tests
- Throughput benchmarks
- Latency measurements
- Memory allocation profiling
- Stress testing with many streams

### Race Detection
- Run all tests with `-race` flag
- Verify no data races
- Check proper synchronization

---

## Acceptance Tests

```go
func TestFeature_4_2_Complete(t *testing.T) {
    // Test complete stream lifecycle
    stream := NewStream("acceptance", DefaultStreamConfig())
    
    // Should accept events
    for i := 0; i < 100; i++ {
        event := StreamEvent{
            Type: ChunkContent,
            Data: []byte(fmt.Sprintf("event %d", i)),
        }
        err := stream.Send(context.Background(), event)
        require.NoError(t, err)
    }
    
    // Should receive all events
    received := 0
    done := make(chan struct{})
    go func() {
        for range stream.Receive() {
            received++
            if received == 100 {
                close(done)
                return
            }
        }
    }()
    
    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Fatal("timeout receiving events")
    }
    
    // Should close gracefully
    err := stream.Close()
    require.NoError(t, err)
}
```

---

## Dependencies

### Internal
- `internal/core/event.go` - Event types

### Standard Library
- `context` - Context cancellation
- `sync` - Synchronization primitives
- `time` - Timestamps and timeouts
- `errors` - Error handling

---

## Documentation

### Godoc
All exported types and functions must have comprehensive godoc comments:
```go
// Stream manages a data stream with buffering and flow control.
// It provides thread-safe operations for sending and receiving events
// with configurable backpressure handling.
//
// Example:
//   stream := NewStream("example", DefaultStreamConfig())
//   defer stream.Close()
//
//   event := StreamEvent{Type: ChunkContent, Data: []byte("hello")}
//   stream.Send(context.Background(), event)
//
//   for evt := range stream.Receive() {
//       fmt.Println(string(evt.Data))
//   }
```

### Examples
Provide runnable examples in test files:
```go
func ExampleStream() {
    stream := NewStream("example", DefaultStreamConfig())
    defer stream.Close()
    
    go func() {
        event := StreamEvent{
            Type: ChunkContent,
            Data: []byte("Hello, Stream!"),
        }
        stream.Send(context.Background(), event)
    }()
    
    evt := <-stream.Receive()
    fmt.Println(string(evt.Data))
    
    // Output: Hello, Stream!
}
```

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Memory leaks from unclosed streams | High | Implement finalizers, document proper usage |
| Race conditions in concurrent access | High | Extensive testing with `-race`, proper locking |
| Deadlocks in backpressure handling | Medium | Careful design, timeout tests |
| Performance bottlenecks | Medium | Benchmarking, profiling, optimization |
| Buffer overflow in high-load scenarios | Medium | Backpressure strategies, monitoring |

---

## Success Criteria

- [ ] All DoD items completed
- [ ] >90% test coverage
- [ ] Zero race conditions
- [ ] All linters passing
- [ ] Performance benchmarks acceptable
- [ ] Documentation complete
- [ ] Integration with event system verified

---

## References

- [ROADMAP.md](../core-module/ROADMAP.md) - Feature 4.2
- [spec.md](../core-module/spec.md) - Core module specification
- [FRD-4.1.md](./FRD-4.1.md) - Event Infrastructure (dependency)
- [Go Blog: Share Memory By Communicating](https://go.dev/blog/codelab-share)

---

**Created:** October 3, 2025  
**Author:** AI Agent  
**Status:** 🚧 In Progress

