package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/state"
	"github.com/google/uuid"
)

// ControlSignal represents a control signal sent to a running turn.
type ControlSignal int

const (
	// SignalPause requests the turn to pause execution
	SignalPause ControlSignal = iota
	// SignalResume requests the turn to resume from paused state
	SignalResume
	// SignalCancel requests the turn to cancel immediately
	SignalCancel
)

// Conversation represents an active conversation with the AI agent.
//
// It coordinates turn execution using the underlying Agent, provides a
// per-conversation event stream, and ensures turns are executed serially.
// Conversation is safe for concurrent use. The Stream channel remains open
// for the lifetime of the conversation and is closed when Stop() is called.
type Conversation struct {
	// Core components
	agent   *Agent
	history *History
	emitter *EventEmitter

	// Events
	events         chan Event
	subscriptionID string
	forwarderDone  chan struct{}

	// State & control
	mu          sync.RWMutex
	state       State // idle | running | paused | cancelled (stopped)
	turnCancel  context.CancelFunc
	turnGuard   chan struct{} // binary semaphore to prevent overlap
	controlChan chan ControlSignal
	controlMu   sync.Mutex // Protects controlChan creation/access

	// Task mode tracking
	currentTask Task   // Current task object (resolved from taskName)
	taskName    string // Current task mode name (for queries/UI)

	// Session tracking
	sessionID string // Unique session identifier (lazy-generated)
}

// NewConversation creates a new Conversation instance wired to the provided
// session, agent, history and shared EventEmitter.
func NewConversation(agent *Agent, history *History, emitter *EventEmitter) *Conversation {
	if history == nil {
		history = NewHistoryWithDefaults()
	}

	c := &Conversation{
		agent:         agent,
		history:       history,
		emitter:       emitter,
		events:        make(chan Event, DefaultEventBufferSize),
		forwarderDone: make(chan struct{}),
		state:         state.StateIdle,
		turnGuard:     make(chan struct{}, 1),
	}

	// Subscribe to shared emitter and forward into conversation stream
	if emitter != nil {
		id, sub, err := emitter.Subscribe()
		if err == nil {
			c.subscriptionID = id
			go c.forwardEvents(sub)
		}
	}

	return c
}

// forwardEvents relays events from the shared emitter to the per-conversation
// stream. It uses fire-and-forget semantics for slow consumers.
func (c *Conversation) forwardEvents(sub <-chan Event) {
	defer close(c.forwarderDone)
	for ev := range sub {
		select {
		case c.events <- ev:
		default:
			// Drop if consumer is slow to avoid blocking
		}
	}
	// Upstream closed; close our stream as well if Stop wasn't responsible
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.events:
		// channel was already closed by Stop; put back value — cannot. So skip.
		// We cannot reliably detect; prefer guarded close below.
		// No-op fallthrough
	default:
	}
	// Close only if not already closed by Stop
	// Recover from potential double close
	safeClose := func() {
		defer func() { _ = recover() }()
		close(c.events)
	}
	safeClose()
}

// RunTurn executes a single conversation turn with the given user input.
// It prevents overlapping executions and emits events via the shared emitter.
func (c *Conversation) RunTurn(ctx context.Context, userInput string) error {
	if userInput == "" {
		return errors.New("user input cannot be empty")
	}

	// Try to acquire turn guard (non-blocking). If busy, return error immediately.
	select {
	case c.turnGuard <- struct{}{}:
		// Acquired
	default:
		return errors.New("a turn is already running")
	}
	// Ensure guard release
	defer func() {
		select {
		case <-c.turnGuard:
		default:
		}
	}()

	// Ensure only one turn runs at a time
	c.mu.Lock()
	if c.state == state.StateCancelled {
		c.mu.Unlock()
		return errors.New("conversation is stopped")
	}
	c.state = state.StateRunning

	// Prepare a cancellable context for the turn
	turnCtx, cancel := context.WithCancel(ctx)
	c.turnCancel = cancel
	c.mu.Unlock()

	// Create control channel for this turn
	c.controlMu.Lock()
	c.controlChan = make(chan ControlSignal, 1)
	controlChan := c.controlChan
	c.controlMu.Unlock()

	// Ensure state and control channel cleanup
	defer func() {
		c.mu.Lock()
		// Only set to Idle if not already Cancelled by Stop()
		if c.state != state.StateCancelled {
			c.state = state.StateIdle
		}
		c.turnCancel = nil
		c.mu.Unlock()

		c.controlMu.Lock()
		if c.controlChan != nil {
			close(c.controlChan)
			c.controlChan = nil
		}
		c.controlMu.Unlock()
	}()

	// WorkDir preference: agent context
	workDir := ""
	if c.agent != nil && c.agent.context != nil {
		workDir = c.agent.context.WorkDir
	}

	// Build request with history (BEFORE adding current user message)
	var historyMsgs []Message
	if c.history != nil {
		historyMsgs = c.history.MessagesForLLM()
	}

	// Get current task mode for this turn
	c.mu.RLock()
	task := c.currentTask
	taskName := c.taskName
	c.mu.RUnlock()

	req := &AgentRequest{
		Input:    userInput,
		History:  historyMsgs,
		Context:  c.agent.context,
		WorkDir:  workDir,
		Task:     task,     // Pass current task object
		TaskName: taskName, // Pass current task name
	}

	// Execute turn with control signal checking
	return c.runTurnWithControl(turnCtx, req, controlChan)
}

// runTurnWithControl executes agent with control signal monitoring
func (c *Conversation) runTurnWithControl(ctx context.Context, req *AgentRequest, controlChan <-chan ControlSignal) error {
	// Execute agent in goroutine
	done := make(chan error, 1)
	var resp *AgentResponse
	var respMu sync.Mutex

	go func() {
		r, err := c.agent.Execute(ctx, req)
		respMu.Lock()
		resp = r
		respMu.Unlock()
		done <- err
	}()

	// Monitor for completion or control signals
	for {
		select {
		case err := <-done:
			// Turn completed
			if err != nil {
				return err
			}

			// Add all messages to history: user input + all turn messages
			if c.history != nil {
				// Add user message first
				_ = c.history.AddUserMessage(req.Input)

				// Add all messages generated during the turn (tool calls, tool results, etc.)
				respMu.Lock()
				if resp != nil && len(resp.Messages) > 0 {
					for _, msg := range resp.Messages {
						// Skip user messages (already added above)
						if msg.Role != RoleUser {
							_ = c.history.AddMessage(msg)
						}
					}
				}
				respMu.Unlock()
			}

			return nil

		case signal := <-controlChan:
			switch signal {
			case SignalPause:
				// Enter paused state, wait for resume or cancel
				if err := c.waitForResume(ctx, controlChan); err != nil {
					return err
				}

			case SignalCancel:
				// Cancel requested
				return context.Canceled

			case SignalResume:
				// Already running, ignore
				continue
			}

		case <-ctx.Done():
			// Context cancelled (e.g., by Stop())
			return ctx.Err()
		}
	}
}

// waitForResume blocks until resume or cancel signal is received
func (c *Conversation) waitForResume(ctx context.Context, controlChan <-chan ControlSignal) error {
	for {
		select {
		case signal := <-controlChan:
			switch signal {
			case SignalResume:
				// Resume execution
				return nil

			case SignalCancel:
				// Cancel while paused
				return context.Canceled

			case SignalPause:
				// Already paused, ignore
				continue
			}

		case <-ctx.Done():
			// Context cancelled while paused
			return ctx.Err()
		}
	}
}

// Stream returns the per-conversation event stream.
func (c *Conversation) Stream() <-chan Event {
	return c.events
}

// Stop gracefully stops the conversation, cancels any running turn,
// unsubscribes from the shared emitter, and closes the stream.
func (c *Conversation) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.state == state.StateCancelled {
		c.mu.Unlock()
		return nil
	}
	c.state = state.StateCancelled
	cancel := c.turnCancel
	c.turnCancel = nil
	subID := c.subscriptionID
	c.subscriptionID = ""
	c.mu.Unlock()

	// Send cancel signal if control channel exists
	c.controlMu.Lock()
	if c.controlChan != nil {
		select {
		case c.controlChan <- SignalCancel:
			// Signal sent
		default:
			// Channel full or closed, ignore
		}
	}
	c.controlMu.Unlock()

	// Cancel running turn if any
	if cancel != nil {
		cancel()
	}

	// Unsubscribe and wait for forwarder to finish, then close events
	if subID != "" && c.emitter != nil {
		c.emitter.Unsubscribe(subID)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-c.forwarderDone
	}()

	select {
	case <-done:
		// Forwarder completed and should have closed stream; ensure closure
		safeClose := func() {
			defer func() { _ = recover() }()
			close(c.events)
		}
		safeClose()
	case <-ctx.Done():
		// Timeout waiting; ensure stream closed to prevent leaks
		safeClose := func() {
			defer func() { _ = recover() }()
			close(c.events)
		}
		safeClose()
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		// Safety timeout
		safeClose := func() {
			defer func() { _ = recover() }()
			close(c.events)
		}
		safeClose()
	}

	return nil
}

// SetTaskMode switches the conversation to a different task mode.
// Returns an error if the task mode is not registered in the agent's task registry.
//
// This method is thread-safe and can be called concurrently with other operations.
// Mode switching takes effect on the next turn execution.
//
// Example:
//
//	if err := conv.SetTaskMode("review"); err != nil {
//	    return fmt.Errorf("failed to switch mode: %w", err)
//	}
func (c *Conversation) SetTaskMode(taskName string) error {
	// Validate that agent has task registry
	if c.agent == nil {
		return errors.New("conversation agent is nil")
	}

	registry := c.agent.GetTaskRegistry()
	if registry == nil {
		return errors.New("agent task registry not initialized")
	}

	// Validate mode exists in agent's registry
	task, err := registry.Get(taskName)
	if err != nil {
		return fmt.Errorf("invalid task mode %q: %w", taskName, err)
	}

	// Validate task
	if err := task.Validate(); err != nil {
		return fmt.Errorf("task %q validation failed: %w", taskName, err)
	}

	// Update state (thread-safe)
	c.mu.Lock()
	c.currentTask = task
	c.taskName = taskName
	c.mu.Unlock()

	// Emit system event for UI/logging
	if c.emitter != nil {
		c.emitter.Emit(Event{
			Type:      EventInfo,
			Timestamp: time.Now(),
			Data: SystemEventData{
				Level:   "info",
				Message: fmt.Sprintf("Switched to %s mode", taskName),
			},
		})
	}

	return nil
}

// GetTaskMode returns the name of the current task mode.
// Returns "regular" if no mode has been explicitly set.
//
// This method is thread-safe and can be called concurrently with other operations.
//
// Example:
//
//	currentMode := conv.GetTaskMode()
//	fmt.Printf("Current mode: %s\n", currentMode)
func (c *Conversation) GetTaskMode() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return explicitly set mode, or default to "regular"
	if c.taskName != "" {
		return c.taskName
	}
	return "regular"
}

// GetMaxTokens returns the maximum token limit for the current task mode.
// This method is thread-safe and can be called concurrently.
func (c *Conversation) GetMaxTokens() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentTask != nil {
		return c.currentTask.MaxTokens()
	}

	// Fallback: return default (16K for regular mode)
	return 16384
}

// GetSessionID returns the session ID for this conversation.
// Generates a new UUID if not already set (lazy initialization).
// This method is thread-safe and can be called concurrently.
func (c *Conversation) GetSessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionID == "" {
		// Lazy generate session ID
		c.sessionID = uuid.New().String()
	}

	return c.sessionID
}

// GetTokenCount returns the total token count from conversation history.
// This method is thread-safe and can be called concurrently.
func (c *Conversation) GetTokenCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.history != nil {
		return c.history.TokenCount()
	}

	return 0
}
