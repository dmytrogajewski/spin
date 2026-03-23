// Package events provides event handling and emission.
package events

import (
	"errors"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// ErrEmitterIsClosed is a sentinel error.
var ErrEmitterIsClosed = errors.New("emitter is closed")

// Event represents a conversation event that can be streamed to UI.
// Events are emitted during agent execution to provide real-time feedback
// about content generation, tool calls, and system status.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// GetType returns the event type (implements cycle.Event interface).
func (e Event) GetType() string {
	return e.Type.String()
}

// GetTimestamp returns the event timestamp (implements cycle.Event interface).
func (e Event) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetData returns the event data (implements cycle.Event interface).
func (e Event) GetData() any {
	return e.Data
}

// ToolCallStartData returns the event data as ToolCallStartData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ToolCallStartData() (ToolCallStartData, bool) {
	data, ok := e.Data.(ToolCallStartData)

	return data, ok
}

// ToolCallCompleteData returns the event data as ToolCallCompleteData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ToolCallCompleteData() (ToolCallCompleteData, bool) {
	data, ok := e.Data.(ToolCallCompleteData)

	return data, ok
}

// PlanUpdateData returns the event data as PlanUpdateData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) PlanUpdateData() (PlanUpdateData, bool) {
	data, ok := e.Data.(PlanUpdateData)

	return data, ok
}

// ToolProgressData returns the event data as ToolProgressData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ToolProgressData() (ToolProgressData, bool) {
	data, ok := e.Data.(ToolProgressData)

	return data, ok
}

// ContentDeltaData returns the event data as ContentDeltaData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ContentDeltaData() (ContentDeltaData, bool) {
	data, ok := e.Data.(ContentDeltaData)

	return data, ok
}

// ThinkingDeltaData returns the event data as ThinkingDeltaData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ThinkingDeltaData() (ThinkingDeltaData, bool) {
	data, ok := e.Data.(ThinkingDeltaData)

	return data, ok
}

// TurnEventData returns the event data as TurnEventData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) TurnEventData() (TurnEventData, bool) {
	data, ok := e.Data.(TurnEventData)

	return data, ok
}

// ApprovalEventData returns the event data as ApprovalEventData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ApprovalEventData() (ApprovalEventData, bool) {
	data, ok := e.Data.(ApprovalEventData)

	return data, ok
}

// SystemEventData returns the event data as SystemEventData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) SystemEventData() (SystemEventData, bool) {
	data, ok := e.Data.(SystemEventData)

	return data, ok
}

// ErrorData returns the event data as ErrorData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ErrorData() (ErrorData, bool) {
	data, ok := e.Data.(ErrorData)

	return data, ok
}

// ACERetrievalData returns the event data as ACERetrievalData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ACERetrievalData() (ACERetrievalData, bool) {
	data, ok := e.Data.(ACERetrievalData)

	return data, ok
}

// ACELearningData returns the event data as ACELearningData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ACELearningData() (ACELearningData, bool) {
	data, ok := e.Data.(ACELearningData)

	return data, ok
}

// CompactionTriggeredData returns the event data as CompactionTriggeredData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) CompactionTriggeredData() (CompactionTriggeredData, bool) {
	data, ok := e.Data.(CompactionTriggeredData)

	return data, ok
}

// DoomLoopDetectedData returns the event data as DoomLoopDetectedData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) DoomLoopDetectedData() (DoomLoopDetectedData, bool) {
	data, ok := e.Data.(DoomLoopDetectedData)

	return data, ok
}

// ReminderInjectedData returns the event data as ReminderInjectedData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) ReminderInjectedData() (ReminderInjectedData, bool) {
	data, ok := e.Data.(ReminderInjectedData)

	return data, ok
}

// SubagentSpawnData returns the event data as SubagentSpawnData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) SubagentSpawnData() (SubagentSpawnData, bool) {
	data, ok := e.Data.(SubagentSpawnData)

	return data, ok
}

// SubagentCompleteData returns the event data as SubagentCompleteData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) SubagentCompleteData() (SubagentCompleteData, bool) {
	data, ok := e.Data.(SubagentCompleteData)

	return data, ok
}

// PhaseThinkingData returns the event data as PhaseThinkingData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) PhaseThinkingData() (PhaseThinkingData, bool) {
	data, ok := e.Data.(PhaseThinkingData)

	return data, ok
}

// PhaseCritiqueData returns the event data as PhaseCritiqueData if possible.
// Returns the data and true if successful, zero value and false otherwise.
func (e Event) PhaseCritiqueData() (PhaseCritiqueData, bool) {
	data, ok := e.Data.(PhaseCritiqueData)

	return data, ok
}

// EventType represents the category of event.
type EventType int

const (
	// EventContentDelta represents text generation from LLM.
	EventContentDelta EventType = iota
	// EventThinkingDelta represents thinking/reasoning output.
	EventThinkingDelta
	// EventContentComplete indicates content generation is complete.
	EventContentComplete

	// EventToolCallStart indicates a tool execution has started.
	EventToolCallStart
	// EventToolCallProgress represents tool execution progress.
	EventToolCallProgress
	// EventToolCallComplete indicates a tool execution has completed.
	EventToolCallComplete
	// EventPlanUpdate represents a plan update event.
	EventPlanUpdate

	// EventTurnStart indicates a new turn has started.
	EventTurnStart
	// EventTurnProgress represents turn progress.
	EventTurnProgress
	// EventTurnComplete indicates a turn has completed.
	EventTurnComplete
	// EventTurnFailed indicates a turn has failed.
	EventTurnFailed
	// EventTurnPaused indicates a turn has been paused.
	EventTurnPaused
	// EventTurnResumed indicates a turn has been resumed.
	EventTurnResumed

	// EventCommandApproval indicates user approval is required.
	EventCommandApproval
	// EventACERetrieval represents an ACE retrieval event.
	EventACERetrieval
	// EventACELearned represents an ACE learning event.
	EventACELearned
	// EventCommandApproved indicates a command was approved.
	EventCommandApproved
	// EventCommandDenied indicates a command was denied.
	EventCommandDenied
	// EventPolicyApplied indicates an approval policy was applied.
	EventPolicyApplied
	// EventPolicySaved indicates an approval policy was saved.
	EventPolicySaved

	// EventTypeToolSelection represents a dynamic tool discovery event.
	EventTypeToolSelection

	// EventError represents a system error event.
	EventError
	// EventWarning represents a system warning event.
	EventWarning
	// EventInfo represents a system info event.
	EventInfo

	// EventCompactionTriggered indicates context compaction was activated.
	EventCompactionTriggered
	// EventDoomLoopDetected indicates a doom-loop fingerprint threshold was exceeded.
	EventDoomLoopDetected
	// EventReminderInjected indicates a system reminder was injected.
	EventReminderInjected
	// EventSubagentSpawn indicates a subagent was launched.
	EventSubagentSpawn
	// EventSubagentComplete indicates a subagent finished execution.
	EventSubagentComplete
	// EventPhaseThinking indicates a thinking LLM call started or completed.
	EventPhaseThinking
	// EventPhaseCritique indicates a critique evaluation started or completed.
	EventPhaseCritique
	// EventUndoRecorded indicates a file operation was recorded for undo.
	EventUndoRecorded
	// EventBackgroundTaskStarted indicates a background task was launched.
	EventBackgroundTaskStarted
	// EventBackgroundTaskStopped indicates a background task has stopped.
	EventBackgroundTaskStopped
	// EventSnapshotTaken indicates a working-tree snapshot was captured.
	EventSnapshotTaken
	// EventSessionIndexRebuilt indicates the session index was rebuilt from metadata files.
	EventSessionIndexRebuilt
	// EventLSPDiagnostics indicates LSP diagnostics were received from a language server.
	EventLSPDiagnostics
)

// String returns the string representation of EventType.
func (e EventType) String() string {
	names := []string{
		"content_delta",
		"thinking_delta",
		"content_complete",
		"tool_call_start",
		"tool_call_progress",
		"tool_call_complete",
		"plan_update",
		"turn_start",
		"turn_progress",
		"turn_complete",
		"turn_failed",
		"turn_paused",
		"turn_resumed",
		"command_approval",
		"ace_retrieval",
		"ace_learned",
		"command_approved",
		"command_denied",
		"policy_applied",
		"policy_saved",
		"tool_selection",
		"error",
		"warning",
		"info",
		"compaction_triggered",
		"doom_loop_detected",
		"reminder_injected",
		"subagent_spawn",
		"subagent_complete",
		"phase_thinking",
		"phase_critique",
		"undo_recorded",
		"background_task_started",
		"background_task_stopped",
		"snapshot_taken",
		"session_index_rebuilt",
		"lsp_diagnostics",
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

// ThinkingDeltaData contains incremental thinking content from LLM.
type ThinkingDeltaData struct {
	Content string `json:"content"`
}

// ToolCallStartData contains tool execution start information.
type ToolCallStartData struct {
	ToolName         string               `json:"tool_name"`
	ToolID           string               `json:"tool_id"`
	Parameters       tools.ToolParameters `json:"parameters"`
	RequiresApproval bool                 `json:"requires_approval"`
}

// ToolProgressData contains tool progress updates.
type ToolProgressData struct {
	ToolID  string `json:"tool_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ToolCallCompleteData contains tool execution results.
type ToolCallCompleteData struct {
	ToolID   string         `json:"tool_id"`
	ToolName string         `json:"tool_name"`
	Success  bool           `json:"success"`
	Output   string         `json:"output"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PlanUpdateData contains the updated plan with current step statuses.
type PlanUpdateData struct {
	Plan any `json:"plan"`
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
	RequestID       string         `json:"request_id"`
	Command         string         `json:"command"`
	WorkDir         string         `json:"work_dir,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	Status          ApprovalStatus `json:"status,omitempty"`
	ModifiedCommand string         `json:"modified_command,omitempty"`
	Timestamp       time.Time      `json:"timestamp"`
}

// ApprovalStatus represents the status of an approval request.
type ApprovalStatus string

const (
	// ApprovalStatusPending indicates the approval is pending.
	ApprovalStatusPending ApprovalStatus = "pending"
	// ApprovalStatusApproved defines a ApprovalStatusApproved constant.
	ApprovalStatusApproved ApprovalStatus = "approved"
	// ApprovalStatusDenied indicates the approval was denied.
	ApprovalStatusDenied ApprovalStatus = "denied"
)

// BulletData represents a single ACE bullet for display.
type BulletData struct {
	Content  string `json:"content"`
	Category string `json:"category,omitempty"` // Optional category for grouping (success_pattern, error_mode, etc.)
}

// ACERetrievalData contains ACE progressive retrieval information.
type ACERetrievalData struct {
	Turn             int          `json:"turn"`
	Trigger          string       `json:"trigger"`
	Query            string       `json:"query"`
	BulletsRetrieved int          `json:"bullets_retrieved"`
	BulletsNew       int          `json:"bullets_new"`
	CacheSize        int          `json:"cache_size"`
	CacheHitRate     float64      `json:"cache_hit_rate"`
	Bullets          []BulletData `json:"bullets"` // Actual bullet content for rendering.
}

// ACELearningData contains ACE learning information after trajectory execution.
type ACELearningData struct {
	Success bool         `json:"success"` // Whether the execution was successful.
	Bullets []BulletData `json:"bullets"` // Learned bullets to display.
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

// CompactionTriggeredData contains context compaction event information.
type CompactionTriggeredData struct {
	Turn  int    `json:"turn"`
	Stage string `json:"stage"`
}

// DoomLoopDetectedData contains doom-loop detection event information.
type DoomLoopDetectedData struct {
	Turn        int    `json:"turn"`
	Fingerprint string `json:"fingerprint"`
	Count       int    `json:"count"`
	ToolName    string `json:"tool_name"`
}

// ReminderInjectedData contains reminder injection event information.
type ReminderInjectedData struct {
	Turn  int `json:"turn"`
	Count int `json:"count"`
}

// SubagentSpawnData contains subagent launch event information.
type SubagentSpawnData struct {
	AgentType string `json:"agent_type"`
	Query     string `json:"query"`
}

// SubagentCompleteData contains subagent completion event information.
type SubagentCompleteData struct {
	AgentType    string `json:"agent_type"`
	Summary      string `json:"summary"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// PhaseThinkingData contains thinking phase event information.
type PhaseThinkingData struct {
	Turn   int    `json:"turn"`
	Status string `json:"status"`
}

// PhaseCritiqueData contains critique phase event information.
type PhaseCritiqueData struct {
	Turn   int    `json:"turn"`
	Status string `json:"status"`
}

// BackpressureMode defines how the emitter handles slow consumers.
type BackpressureMode int

const (
	// BackpressureDrop drops events when subscriber buffer is full (fire-and-forget)
	// Best for: Non-critical events, high-throughput scenarios.
	BackpressureDrop BackpressureMode = iota

	// BackpressureBlock blocks the emitter until subscriber is ready
	// Best for: Critical events that must be delivered (approvals, errors).
	BackpressureBlock

	// BackpressureBuffer uses dynamic buffer growth up to limit
	// Best for: Bursty workloads where brief slowdowns are acceptable.
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
	// BufferSize is the initial channel buffer size per subscriber.
	BufferSize int

	// BackpressureMode determines how to handle slow consumers.
	BackpressureMode BackpressureMode

	// BufferLimit is the maximum buffer size for BackpressureBuffer mode
	// Ignored for other modes. Default: 10000 events.
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

	// Backpressure configuration.
	config   EventEmitterConfig
	buffers  map[string][]Event // Dynamic buffers for BackpressureBuffer mode.
	bufferMu sync.Mutex         // Protects buffers map.
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
	// Set default buffer limit if not specified.
	if config.BufferLimit == 0 {
		config.BufferLimit = 10000
	}

	emitter := &EventEmitter{
		subscribers: make(map[string]chan Event),
		bufferSize:  config.BufferSize,
		config:      config,
		closed:      false,
	}

	// Initialize buffers map for BackpressureBuffer mode.
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
		return "", nil, ErrEmitterIsClosed
	}

	// Generate unique subscriber ID.
	id = uuid.New().String()

	// Create buffered channel for subscriber.
	ch := make(chan Event, e.bufferSize)
	e.subscribers[id] = ch

	// Initialize buffer for BackpressureBuffer mode.
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

		// Clean up buffer for BackpressureBuffer mode.
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
	// Set timestamp if not provided.
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	e.mu.RLock()
	// Don't emit if closed.
	if e.closed {
		e.mu.RUnlock()

		return
	}

	mode := e.config.BackpressureMode

	// For non-blocking modes, keep lock and emit directly.
	if mode == BackpressureDrop || mode == BackpressureBuffer {
		for id, ch := range e.subscribers {
			switch mode {
			case BackpressureDrop:
				concurrency.TrySend(ch, event)
			case BackpressureBuffer:
				e.emitBuffer(id, ch, event)
			}
		}

		e.mu.RUnlock()

		return
	}

	// For BackpressureBlock, we need to release the lock to avoid deadlock
	// Take snapshot of subscribers.
	subscribers := make(map[string]chan Event)
	maps.Copy(subscribers, e.subscribers)

	e.mu.RUnlock()

	// Emit with blocking (might take time).
	for _, ch := range subscribers {
		concurrency.SendWithTimeout(ch, event, emitBlockTimeout)
	}
}

// emitBlockTimeout is the maximum time to wait for a blocking emit before dropping.
const emitBlockTimeout = 5 * time.Second

// emitBuffer implements dynamic buffering (buffers events up to limit).
func (e *EventEmitter) emitBuffer(id string, ch chan Event, event Event) {
	// First try to flush any existing buffered events.
	e.tryFlushBuffer(id, ch)

	// Then try to send current event.
	select {
	case ch <- event:
		// Successfully sent.
	default:
		// Channel full, add to dynamic buffer.
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
	// If over limit, drop event (similar to BackpressureDrop).
}

// tryFlushBuffer attempts to flush buffered events to channel (non-blocking).
func (e *EventEmitter) tryFlushBuffer(id string, ch chan Event) {
	e.bufferMu.Lock()
	defer e.bufferMu.Unlock()

	buffer := e.buffers[id]
	if len(buffer) == 0 {
		return
	}

	// Try to send buffered events (non-blocking).
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

	// Remove sent events from buffer.
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
		return // Already closed.
	}

	e.closed = true

	// Close all subscriber channels.
	for id, ch := range e.subscribers {
		close(ch)
		delete(e.subscribers, id)
	}

	// Clean up buffers for BackpressureBuffer mode.
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
// Use Subscribe() directly if you need to unsubscribe.
func (e *EventEmitter) Events() <-chan Event {
	_, ch, err := e.Subscribe()
	if err != nil {
		// Return closed channel if subscription fails.
		closedCh := make(chan Event)
		close(closedCh)

		return closedCh
	}

	return ch
}
