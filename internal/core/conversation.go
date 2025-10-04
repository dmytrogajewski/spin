package core

import (
	"context"
	"errors"
	"sync"
	"time"
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
	mu         sync.RWMutex
	state      string // idle | running | stopped
	turnCancel context.CancelFunc
	turnGuard  chan struct{} // binary semaphore to prevent overlap
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
		state:         "idle",
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
	if c.state == "stopped" {
		c.mu.Unlock()
		return errors.New("conversation is stopped")
	}
	c.state = "running"

	// Prepare a cancellable context for the turn
	turnCtx, cancel := context.WithCancel(ctx)
	c.turnCancel = cancel
	c.mu.Unlock()

	// Ensure state cleanup
	defer func() {
		c.mu.Lock()
		c.state = "idle"
		c.turnCancel = nil
		c.mu.Unlock()
	}()

	// Record user message
	if c.history != nil {
		_ = c.history.AddUserMessage(userInput)
	}

	// WorkDir preference: agent context
	workDir := ""
	if c.agent != nil && c.agent.context != nil {
		workDir = c.agent.context.WorkDir
	}

	// Build request
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

	// Execute agent
	resp, err := c.agent.Execute(turnCtx, req)
	if err != nil {
		// Error already meaningful from agent; propagate
		return err
	}

	// Append assistant response if present
	if resp != nil && resp.Content != "" && c.history != nil {
		_ = c.history.AddAssistantMessage(resp.Content)
	}

	return nil
}

// Stream returns the per-conversation event stream.
func (c *Conversation) Stream() <-chan Event {
	return c.events
}

// Stop gracefully stops the conversation, cancels any running turn,
// unsubscribes from the shared emitter, and closes the stream.
func (c *Conversation) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.state == "stopped" {
		c.mu.Unlock()
		return nil
	}
	c.state = "stopped"
	cancel := c.turnCancel
	c.turnCancel = nil
	subID := c.subscriptionID
	c.subscriptionID = ""
	c.mu.Unlock()

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

// State returns the current conversation state (idle | running | stopped).
func (c *Conversation) State() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
