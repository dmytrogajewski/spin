package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// slowProvider delays completion to keep RunTurn in-progress
type slowProvider struct{ delay time.Duration }

var _ llm.Provider = (*slowProvider)(nil)

func (s *slowProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &llm.CompletionResponse{Content: "ok", FinishReason: "stop"}, nil
}

func (s *slowProvider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	return nil, nil
}

func (s *slowProvider) Models(ctx context.Context) ([]llm.Model, error) {
	return nil, nil
}

func (s *slowProvider) Capabilities() llm.Capabilities {
	return llm.Capabilities{Streaming: false, FunctionCalling: false}
}

func (s *slowProvider) Name() string {
	return "slow"
}

func (s *slowProvider) Close() error {
	return nil
}

func TestConversation_RunTurn_Basic(t *testing.T) {
	// Setup dependencies
	llm := llm.NewMockProvider("assistant reply")
	emitter := NewEventEmitter(100)
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctxEnv := &Context{WorkDir: workDir}

	agent, err := NewAgent(llm, executor, validator, ctxEnv, emitter)
	if err != nil {
		t.Fatalf("NewAgent() error: %v", err)
	}

	history := NewHistoryWithDefaults()
	if err := history.AddSystemMessage("You are helpful"); err != nil {
		t.Fatalf("failed to init history: %v", err)
	}

	// Minimal session substitute for test; not used directly in behavior
	conv := &Conversation{}
	_ = conv // to satisfy lint for now until implementation

	// Create conversation via constructor (to be implemented)
	c := NewTestConversation(t, agent, history, emitter)

	// Subscribe to stream
	events := c.Stream()

	// Run a turn
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.RunTurn(ctx, "List files"); err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}

	// Drain some events
	deadline := time.After(500 * time.Millisecond)
	received := 0
loop:
	for {
		select {
		case _, ok := <-events:
			if !ok {
				break loop
			}
			received++
		case <-deadline:
			break loop
		}
	}

	if received == 0 {
		t.Error("expected events to be emitted during RunTurn")
	}
}

func TestConversation_RunTurn_NoOverlap(t *testing.T) {
	llm := &slowProvider{delay: 300 * time.Millisecond}
	emitter := NewEventEmitter(100)
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctxEnv := &Context{WorkDir: workDir}
	agent, _ := NewAgent(llm, executor, validator, ctxEnv, emitter)
	history := NewHistoryWithDefaults()
	_ = history.AddSystemMessage("sys")
	c := NewTestConversation(t, agent, history, emitter)

	// Start first turn but do not wait
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()
	done := make(chan error)
	go func() { done <- c.RunTurn(ctx1, "first") }()

	// Wait until the first turn actually starts (observe EventTurnStart)
	started := false
	waitDeadline := time.After(500 * time.Millisecond)
	for !started {
		select {
		case ev := <-c.Stream():
			if ev.Type == EventTurnStart {
				started = true
			}
		case <-waitDeadline:
			t.Fatal("timeout waiting for first turn to start")
		}
	}

	// Now try to start another turn which should be rejected
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := c.RunTurn(ctx2, "second"); err == nil {
		t.Error("expected error when starting overlapping RunTurn")
	}

	<-done
}

func TestConversation_Stop_ClosesStream(t *testing.T) {
	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)
	validator := NewValidator()
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	ctxEnv := &Context{WorkDir: workDir}
	agent, _ := NewAgent(llm, executor, validator, ctxEnv, emitter)
	history := NewHistoryWithDefaults()
	_ = history.AddSystemMessage("sys")
	c := NewTestConversation(t, agent, history, emitter)

	ch := c.Stream()

	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("stream channel should be closed after Stop()")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for stream close")
	}
}

// NewTestConversation is a testing helper that will be implemented in conversation.go
func NewTestConversation(t *testing.T, agent *Agent, history *History, emitter *EventEmitter) *Conversation {
	t.Helper()
	// Create minimal session surrogate (nil ok)
	return NewConversation(agent, history, emitter)
}
