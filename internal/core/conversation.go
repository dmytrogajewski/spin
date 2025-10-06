package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
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

// String returns the string representation of ControlSignal.
func (s ControlSignal) String() string {
	switch s {
	case SignalPause:
		return "pause"
	case SignalResume:
		return "resume"
	case SignalCancel:
		return "cancel"
	default:
		return "unknown"
	}
}

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
		state:         StateIdle,
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
	if c.state == StateCancelled {
		c.mu.Unlock()
		return errors.New("conversation is stopped")
	}
	c.state = StateRunning

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
		if c.state != StateCancelled {
			c.state = StateIdle
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

	req := &AgentRequest{
		Input:   userInput,
		History: historyMsgs,
		Context: c.agent.context,
		WorkDir: workDir,
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

			// Add both user and assistant messages to history
			if c.history != nil {
				_ = c.history.AddUserMessage(req.Input)

				respMu.Lock()
				if resp != nil && resp.Content != "" {
					_ = c.history.AddAssistantMessage(resp.Content)
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
	if c.state == StateCancelled {
		c.mu.Unlock()
		return nil
	}
	c.state = StateCancelled
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

// State returns the current conversation state.
func (c *Conversation) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Pause pauses the currently running turn.
// Returns an error if no turn is running or conversation is stopped.
func (c *Conversation) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate state
	if c.state != StateRunning {
		return fmt.Errorf("cannot pause: conversation is %s", c.state.String())
	}

	// Send pause signal (non-blocking)
	c.controlMu.Lock()
	if c.controlChan != nil {
		select {
		case c.controlChan <- SignalPause:
			// Signal sent
		default:
			// Channel full, pause already requested
		}
	}
	c.controlMu.Unlock()

	// Transition to paused state
	c.state = StatePaused

	// Emit event
	if c.emitter != nil {
		c.emitter.Emit(Event{
			Type:      EventTurnPaused,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"reason": "user requested"},
		})
	}

	return nil
}

// Resume resumes a paused turn.
// Returns an error if no turn is paused.
func (c *Conversation) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate state
	if c.state != StatePaused {
		return fmt.Errorf("cannot resume: conversation is %s", c.state.String())
	}

	// Send resume signal (non-blocking)
	c.controlMu.Lock()
	if c.controlChan != nil {
		select {
		case c.controlChan <- SignalResume:
			// Signal sent
		default:
			// Channel full, resume already requested
		}
	}
	c.controlMu.Unlock()

	// Transition back to running
	c.state = StateRunning

	// Emit event
	if c.emitter != nil {
		c.emitter.Emit(Event{
			Type:      EventTurnResumed,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"reason": "user requested"},
		})
	}

	return nil
}
