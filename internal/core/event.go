package core

import (
	"errors"
	"sync"
	"time"

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
	Role    string `json:"role"` // assistant, tool
}

// ToolCallData contains tool execution information.
type ToolCallData struct {
	ToolName   string                 `json:"tool_name"`
	ToolID     string                 `json:"tool_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolProgressData contains tool progress updates.
type ToolProgressData struct {
	ToolID  string `json:"tool_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ToolResultData contains tool execution results.
type ToolResultData struct {
	ToolID   string `json:"tool_id"`
	ToolName string `json:"tool_name"`
	Result   string `json:"result"`
	Error    string `json:"error,omitempty"`
}

// ErrorData contains error information.
type ErrorData struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// CommandApprovalData contains approval request information.
type CommandApprovalData struct {
	Command   string `json:"command"`
	Class     string `json:"class"` // dangerous, interactive
	Reason    string `json:"reason"`
	WorkDir   string `json:"work_dir"`
	RequestID string `json:"request_id"`
}

// EventEmitter manages event subscriptions and distribution using pub/sub pattern.
// It allows multiple subscribers to receive events in real-time with thread-safe
// operations and fire-and-forget semantics to prevent slow subscribers from
// blocking the system.
type EventEmitter struct {
	subscribers map[string]chan Event
	mu          sync.RWMutex
	bufferSize  int
	closed      bool
}

// NewEventEmitter creates a new EventEmitter with the specified channel buffer size.
// The buffer size determines how many events can be queued per subscriber before
// events are dropped for slow consumers.
func NewEventEmitter(bufferSize int) *EventEmitter {
	return &EventEmitter{
		subscribers: make(map[string]chan Event),
		bufferSize:  bufferSize,
		closed:      false,
	}
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
	}
}

// Emit sends an event to all active subscribers using fire-and-forget semantics.
// If a subscriber's channel is full, the event is dropped for that subscriber
// to prevent blocking. The timestamp is automatically set if not provided.
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
