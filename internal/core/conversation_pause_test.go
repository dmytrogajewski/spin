package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestConversation_Pause_WhenRunning tests pausing a running turn
func TestConversation_Pause_WhenRunning(t *testing.T) {
	// Create conversation with mock LLM that has long delay
	provider := llm.NewMockProvider("test",
		llm.WithDelay(500*time.Millisecond),
		llm.WithResponse("Long running response"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	history := NewHistoryWithDefaults()
	conv := NewConversation(agent, history, NewEventEmitter(100))

	// Subscribe to events
	_, events, _ := conv.emitter.Subscribe()

	// Start turn in goroutine
	done := make(chan error, 1)
	go func() {
		done <- conv.RunTurn(context.Background(), "test input")
	}()

	// Wait for turn to start running
	time.Sleep(50 * time.Millisecond)

	// Verify state is running
	if conv.State() != StateRunning {
		t.Errorf("expected state Running, got %s", conv.State())
	}

	// Pause the turn
	err := conv.Pause()
	if err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	// Verify state transitioned to Paused
	if conv.State() != StatePaused {
		t.Errorf("expected state Paused after Pause(), got %s", conv.State())
	}

	// Verify EventTurnPaused was emitted
	timeout := time.After(200 * time.Millisecond)
	pausedEventFound := false
	for !pausedEventFound {
		select {
		case event := <-events:
			if event.Type == EventTurnPaused {
				pausedEventFound = true
			}
		case <-timeout:
			t.Error("EventTurnPaused not emitted within timeout")
			pausedEventFound = true // break loop
		}
	}

	// Resume to allow turn to complete
	_ = conv.Resume()

	// Wait for turn to complete
	select {
	case <-done:
		// Turn completed
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not complete after resume")
	}
}

// TestConversation_Pause_WhenNotRunning tests pausing when not running
func TestConversation_Pause_WhenNotRunning(t *testing.T) {
	provider := llm.NewMockProvider("test")
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Try to pause when Idle
	err := conv.Pause()
	if err == nil {
		t.Error("expected error when pausing Idle conversation, got nil")
	}

	// Verify state unchanged
	if conv.State() != StateIdle {
		t.Errorf("state should remain Idle, got %s", conv.State())
	}
}

// TestConversation_Resume_WhenPaused tests resuming a paused turn
func TestConversation_Resume_WhenPaused(t *testing.T) {
	// Create conversation with delayed mock LLM
	provider := llm.NewMockProvider("test",
		llm.WithDelay(300*time.Millisecond),
		llm.WithResponse("Response after pause/resume"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	history := NewHistoryWithDefaults()
	conv := NewConversation(agent, history, NewEventEmitter(100))

	// Subscribe to events
	_, events, _ := conv.emitter.Subscribe()

	// Start turn
	done := make(chan error, 1)
	go func() {
		done <- conv.RunTurn(context.Background(), "test input")
	}()

	// Wait for turn to be running
	time.Sleep(50 * time.Millisecond)

	// Pause
	err := conv.Pause()
	if err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	// Verify paused
	if conv.State() != StatePaused {
		t.Fatalf("expected StatePaused, got %s", conv.State())
	}

	// Resume
	err = conv.Resume()
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}

	// Verify state transitioned to Running
	if conv.State() != StateRunning {
		t.Errorf("expected state Running after Resume(), got %s", conv.State())
	}

	// Verify EventTurnResumed was emitted
	timeout := time.After(200 * time.Millisecond)
	resumedEventFound := false
	for !resumedEventFound {
		select {
		case event := <-events:
			if event.Type == EventTurnResumed {
				resumedEventFound = true
			}
		case <-timeout:
			t.Error("EventTurnResumed not emitted within timeout")
			resumedEventFound = true
		}
	}

	// Wait for turn to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("turn error after resume: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not complete after resume")
	}

	// Verify turn completed successfully
	if conv.State() != StateIdle {
		t.Errorf("expected state Idle after completion, got %s", conv.State())
	}
}

// TestConversation_Resume_WhenNotPaused tests resuming when not paused
func TestConversation_Resume_WhenNotPaused(t *testing.T) {
	provider := llm.NewMockProvider("test")
	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Try to resume when Idle
	err := conv.Resume()
	if err == nil {
		t.Error("expected error when resuming non-paused conversation, got nil")
	}

	// Try to resume when Running
	go conv.RunTurn(context.Background(), "test")
	time.Sleep(50 * time.Millisecond)

	err = conv.Resume()
	if err == nil {
		t.Error("expected error when resuming running conversation, got nil")
	}
}

// TestConversation_PauseResumeCycle tests full pause/resume cycle
func TestConversation_PauseResumeCycle(t *testing.T) {
	provider := llm.NewMockProvider("test",
		llm.WithDelay(400*time.Millisecond),
		llm.WithResponse("Response"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Start turn
	done := make(chan error, 1)
	go func() {
		done <- conv.RunTurn(context.Background(), "test")
	}()

	// Wait for running
	time.Sleep(50 * time.Millisecond)

	// Pause
	if err := conv.Pause(); err != nil {
		t.Fatalf("Pause() error: %v", err)
	}
	if conv.State() != StatePaused {
		t.Errorf("expected Paused, got %s", conv.State())
	}

	// Wait a bit while paused
	time.Sleep(100 * time.Millisecond)

	// Resume
	if err := conv.Resume(); err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if conv.State() != StateRunning {
		t.Errorf("expected Running, got %s", conv.State())
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("turn error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not complete")
	}

	// Verify final state
	if conv.State() != StateIdle {
		t.Errorf("expected Idle after completion, got %s", conv.State())
	}
}

// TestConversation_StopWhilePaused tests calling Stop() while paused
func TestConversation_StopWhilePaused(t *testing.T) {
	provider := llm.NewMockProvider("test",
		llm.WithDelay(500*time.Millisecond),
		llm.WithResponse("Response"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Start turn
	done := make(chan error, 1)
	go func() {
		done <- conv.RunTurn(context.Background(), "test")
	}()

	// Wait for running
	time.Sleep(50 * time.Millisecond)

	// Pause
	if err := conv.Pause(); err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	// Stop while paused
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := conv.Stop(stopCtx); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Verify state is Cancelled
	if conv.State() != StateCancelled {
		t.Errorf("expected StateCancelled, got %s", conv.State())
	}

	// Wait for turn to finish (should return with cancellation error)
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error from cancelled turn, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not finish after Stop()")
	}
}

// TestConversation_ContextCancellationWhilePaused tests context cancellation while paused
func TestConversation_ContextCancellationWhilePaused(t *testing.T) {
	provider := llm.NewMockProvider("test",
		llm.WithDelay(500*time.Millisecond),
		llm.WithResponse("Response"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start turn
	done := make(chan error, 1)
	go func() {
		done <- conv.RunTurn(ctx, "test")
	}()

	// Wait for running
	time.Sleep(50 * time.Millisecond)

	// Pause
	if err := conv.Pause(); err != nil {
		t.Fatalf("Pause() error: %v", err)
	}

	// Cancel context while paused
	cancel()

	// Turn should return with context.Canceled
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not return after context cancellation")
	}
}

// TestConversation_ConcurrentPauseResume tests concurrent Pause/Resume calls
func TestConversation_ConcurrentPauseResume(t *testing.T) {
	provider := llm.NewMockProvider("test",
		llm.WithDelay(1*time.Second),
		llm.WithResponse("Response"),
	)

	workDir := t.TempDir()
	executor, _ := NewExecutor(workDir)
	agent, _ := NewAgent(provider, executor, NewValidator(), &Environment{WorkDir: workDir}, NewEventEmitter(100))
	conv := NewConversation(agent, NewHistoryWithDefaults(), NewEventEmitter(100))

	// Start turn
	go conv.RunTurn(context.Background(), "test")

	// Wait for running
	time.Sleep(50 * time.Millisecond)

	// Concurrent pause calls (should not panic)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			conv.Pause()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-done:
		// Success - no panic
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent Pause() calls deadlocked")
	}

	// Resume
	_ = conv.Resume()

	// Concurrent resume calls (should not panic)
	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		for i := 0; i < 10; i++ {
			conv.Resume()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-done2:
		// Success - no panic
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent Resume() calls deadlocked")
	}
}
