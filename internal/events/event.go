package events

import (
	"errors"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/google/uuid"
)

// Event represents a conversation event that can be streamed to UI.
// Events are emitted during agent execution to provide real-time feedback
// about content generation, tool calls, and system status.
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// GetType returns the event type (implements cycle.Event interface)
func (e Event) GetType() string {
	return e.Type.String()
}

// GetTimestamp returns the event timestamp (implements cycle.Event interface)
func (e Event) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetData returns the event data (implements cycle.Event interface)
func (e Event) GetData() interface{} {
	return e.Data
}

// EventType represents the category of event.
type EventType int

const (
	// Content events - text generation from LLM
	EventContentDelta EventType = iota
	EventContentComplete

	// Tool events - tool execution lifecycle
	EventToolCallStart
	EventToolCallProgress
	EventToolCallComplete

	// Turn events - turn lifecycle
	EventTurnStart
	EventTurnComplete
	EventTurnFailed
	EventTurnPaused
	EventTurnResumed

	// Approval events - user approval required
	EventCommandApproval
	EventCommandApproved
	EventCommandDenied

	// System events - system-level messages
	EventError
	EventWarning
	EventInfo
)

// String returns the string representation of EventType.
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
		"turn_paused",
		"turn_resumed",
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

// ContentDeltaData contains incremental content from LLM.
type ContentDeltaData struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

// ToolCallStartData contains tool execution start information.
type ToolCallStartData struct {
	ToolName         string          `json:"tool_name"`
	ToolID           string          `json:"tool_id"`
	Parameters       tools.Arguments `json:"parameters"`
	RequiresApproval bool            `json:"requires_approval"`
}

// ToolProgressData contains tool progress updates.
type ToolProgressData struct {
	ToolID  string `json:"tool_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ToolCallCompleteData contains tool execution results.
type ToolCallCompleteData struct {
	ToolID   string `json:"tool_id"`
	ToolName string `json:"tool_name"`
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// TurnEventData contains turn lifecycle information.
type TurnEventData struct {
	Turn       int    `json:"turn"`
	TurnID     string `json:"turn_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message,omitempty"`
	TurnsUsed  int    `json:"turns_used,omitempty"`
	TokensUsed int    `json:"tokens_used,omitempty"`
	MaxTurns   int    `json:"max_turns,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ApprovalEventData contains approval request and status information.
type ApprovalEventData struct {
	RequestID       string    `json:"request_id"`
	Command         string    `json:"command"`
	WorkDir         string    `json:"work_dir,omitempty"`
	Reason          string    `json:"reason,omitempty"`
	Status          string    `json:"status,omitempty"`
	ModifiedCommand string    `json:"modified_command,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// SystemEventData contains informational or warning messages.
type SystemEventData struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ErrorData contains error information.
type ErrorData struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// BackpressureMode defines how the emitter handles slow consumers.
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

// String returns the string representation of BackpressureMode.
func (m BackpressureMode) String() string {
	switch m {
	case BackpressureDrop:
		return "drop"
	case BackpressureBlock:
		return "block"
	case BackpressureBuffer:
		return "buffer"
	default:
		return "unknown"
	}
}

// EventEmitterConfig configures event emitter behavior.
type EventEmitterConfig struct {
	// BufferSize is the initial channel buffer size per subscriber
	BufferSize int

	// BackpressureMode determines how to handle slow consumers
	BackpressureMode BackpressureMode

	// BufferLimit is the maximum buffer size for BackpressureBuffer mode
	// Ignored for other modes. Default: 10000 events
	BufferLimit int
}

// EventEmitter manages event subscriptions and distribution using pub/sub pattern.
// It allows multiple subscribers to receive events in real-time with thread-safe
// operations and configurable backpressure strategies to handle slow consumers.
type EventEmitter struct {
	subscribers map[string]chan Event
	mu          sync.RWMutex
	bufferSize  int
	closed      bool

	// Backpressure configuration
	config   EventEmitterConfig
	buffers  map[string][]Event // Dynamic buffers for BackpressureBuffer mode
	bufferMu sync.Mutex         // Protects buffers map
}

// NewEventEmitter creates a new EventEmitter with default config (BackpressureDrop).
// The buffer size determines how many events can be queued per subscriber before
// events are dropped for slow consumers.
// Maintains backward compatibility with existing code.
func NewEventEmitter(bufferSize int) *EventEmitter {
	return NewEventEmitterWithConfig(EventEmitterConfig{
		BufferSize:       bufferSize,
		BackpressureMode: BackpressureDrop,
		BufferLimit:      0,
	})
}

// NewEventEmitterWithConfig creates a new EventEmitter with custom configuration.
// This allows selecting different backpressure strategies for handling slow consumers.
func NewEventEmitterWithConfig(config EventEmitterConfig) *EventEmitter {
	// Set default buffer limit if not specified
	if config.BufferLimit == 0 {
		config.BufferLimit = 10000
	}

	emitter := &EventEmitter{
		subscribers: make(map[string]chan Event),
		bufferSize:  config.BufferSize,
		config:      config,
		closed:      false,
	}

	// Initialize buffers map for BackpressureBuffer mode
	if config.BackpressureMode == BackpressureBuffer {
		emitter.buffers = make(map[string][]Event)
	}

	return emitter
}

// Subscribe creates a new subscription and returns a unique ID and event channel.
// The subscriber can receive events by reading from the returned channel.
// Returns an error if the emitter has been closed.
func (e *EventEmitter) Subscribe() (id string, events <-chan Event, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return "", nil, errors.New("emitter is closed")
	}

	// Generate unique subscriber ID
	id = uuid.New().String()

	// Create buffered channel for subscriber
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

// Unsubscribe removes a subscriber and closes their event channel.
// It's safe to call with a non-existent ID (no-op).
func (e *EventEmitter) Unsubscribe(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ch, exists := e.subscribers[id]; exists {
		close(ch)
		delete(e.subscribers, id)

		// Clean up buffer for BackpressureBuffer mode
		if e.config.BackpressureMode == BackpressureBuffer {
			e.bufferMu.Lock()
			delete(e.buffers, id)
			e.bufferMu.Unlock()
		}
	}
}

// Emit sends an event to all active subscribers using the configured backpressure strategy.
// The timestamp is automatically set if not provided.
func (e *EventEmitter) Emit(event Event) {
	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	e.mu.RLock()
	// Don't emit if closed
	if e.closed {
		e.mu.RUnlock()
		return
	}

	mode := e.config.BackpressureMode

	// For non-blocking modes, keep lock and emit directly
	if mode == BackpressureDrop || mode == BackpressureBuffer {
		for id, ch := range e.subscribers {
			switch mode {
			case BackpressureDrop:
				e.emitDrop(ch, event)
			case BackpressureBuffer:
				e.emitBuffer(id, ch, event)
			}
		}
		e.mu.RUnlock()
		return
	}

	// For BackpressureBlock, we need to release the lock to avoid deadlock
	// Take snapshot of subscribers
	subscribers := make(map[string]chan Event)
	for id, ch := range e.subscribers {
		subscribers[id] = ch
	}
	e.mu.RUnlock()

	// Emit with blocking (might take time)
	for _, ch := range subscribers {
		e.emitBlock(ch, event)
	}
}

// emitDrop implements fire-and-forget (drops events if channel full).
func (e *EventEmitter) emitDrop(ch chan Event, event Event) {
	select {
	case ch <- event:
		// Successfully sent
	default:
		// Subscriber slow/blocked, drop event
	}
}

// emitBlock implements blocking send (ensures delivery).
func (e *EventEmitter) emitBlock(ch chan Event, event Event) {
	ch <- event // Blocks until consumer ready
}

// emitBuffer implements dynamic buffering (buffers events up to limit).
func (e *EventEmitter) emitBuffer(id string, ch chan Event, event Event) {
	// First try to flush any existing buffered events
	e.tryFlushBuffer(id, ch)

	// Then try to send current event
	select {
	case ch <- event:
		// Successfully sent
	default:
		// Channel full, add to dynamic buffer
		e.addToBuffer(id, event)
	}
}

// addToBuffer adds event to dynamic buffer (if under limit).
func (e *EventEmitter) addToBuffer(id string, event Event) {
	e.bufferMu.Lock()
	defer e.bufferMu.Unlock()

	buffer := e.buffers[id]
	if len(buffer) < e.config.BufferLimit {
		e.buffers[id] = append(buffer, event)
	}
	// If over limit, drop event (similar to BackpressureDrop)
}

// tryFlushBuffer attempts to flush buffered events to channel (non-blocking).
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
			goto done
		}
	}
done:

	// Remove sent events from buffer
	if sent > 0 {
		e.buffers[id] = buffer[sent:]
	}
}

// Close shuts down the emitter and closes all subscriber channels.
// It's safe to call multiple times (idempotent).
// After Close(), Subscribe() will return an error.
func (e *EventEmitter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return // Already closed
	}

	e.closed = true

	// Close all subscriber channels
	for id, ch := range e.subscribers {
		close(ch)
		delete(e.subscribers, id)
	}

	// Clean up buffers for BackpressureBuffer mode
	if e.config.BackpressureMode == BackpressureBuffer {
		e.bufferMu.Lock()
		for id := range e.buffers {
			delete(e.buffers, id)
		}
		e.bufferMu.Unlock()
	}
}

// Events returns a channel for receiving events (convenience method for testing).
// This is a simple wrapper around Subscribe() that returns only the channel.
// The subscription ID is not returned, so Unsubscribe cannot be called.
// Use Subscribe() directly if you need to unsubscribe later.
func (e *EventEmitter) Events() <-chan Event {
	_, ch, err := e.Subscribe()
	if err != nil {
		// Return closed channel if subscription fails
		closedCh := make(chan Event)
		close(closedCh)
		return closedCh
	}
	return ch
}
