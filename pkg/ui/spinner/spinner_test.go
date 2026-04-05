package spinner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSpinner_Frame(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)

	// Not running - should return empty.
	frame := s.Frame()
	if frame != "" {
		t.Errorf("expected empty frame when not running, got %q", frame)
	}

	// Start and check frames exist.
	ctx := t.Context()

	s.Start(ctx)
	defer s.Stop()

	// Wait a bit for animation to start.
	time.Sleep(10 * time.Millisecond)

	frame = s.Frame()
	if frame == "" {
		t.Error("expected non-empty frame when running")
	}

	// Frame should be one of the dots frames.
	validFrames := map[string]bool{
		"⠋": true, "⠙": true, "⠹": true, "⠸": true,
		"⠼": true, "⠴": true, "⠦": true, "⠧": true,
		"⠇": true, "⠏": true,
	}
	if !validFrames[frame] {
		t.Errorf("unexpected frame %q", frame)
	}
}

func TestSpinner_StartStop(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleCircle)

	if s.IsRunning() {
		t.Error("spinner should not be running initially")
	}

	ctx := t.Context()

	s.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	if !s.IsRunning() {
		t.Error("spinner should be running after Start")
	}

	s.Stop()
	time.Sleep(10 * time.Millisecond)

	if s.IsRunning() {
		t.Error("spinner should not be running after Stop")
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleBraille)

	ctx := t.Context()

	s.Start(ctx)
	defer s.Stop()

	// Double start should be a no-op.
	s.Start(ctx)

	if !s.IsRunning() {
		t.Error("spinner should still be running after double Start")
	}
}

func TestSpinner_UpdateCallback(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(20 * time.Millisecond)

	var callCount atomic.Int32

	s.SetUpdateCallback(func() {
		callCount.Add(1)
	})

	ctx := t.Context()

	s.Start(ctx)

	// Wait for several animation frames.
	time.Sleep(100 * time.Millisecond)

	s.Stop()

	count := callCount.Load()
	if count < 2 {
		t.Errorf("expected callback to be called multiple times, got %d", count)
	}
}

func TestSpinner_Styles(t *testing.T) {
	t.Parallel()

	styles := []Style{
		StyleDots,
		StyleBraille,
		StyleCircle,
		SpinnerPulse,
	}

	for _, style := range styles {
		s := NewSpinner(style)
		ctx, cancel := context.WithCancel(context.Background())

		s.Start(ctx)
		time.Sleep(10 * time.Millisecond)

		frame := s.Frame()
		if frame == "" {
			t.Errorf("style %d: expected non-empty frame", style)
		}

		s.Stop()
		cancel()
	}
}

func TestSpinner_SetStyle(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)

	// Change style.
	s.SetStyle(StyleCircle)

	ctx := t.Context()

	s.Start(ctx)
	defer s.Stop()

	time.Sleep(10 * time.Millisecond)

	frame := s.Frame()

	// Should be a circle frame.
	validFrames := map[string]bool{
		"◐": true, "◓": true, "◑": true, "◒": true,
	}
	if !validFrames[frame] {
		t.Errorf("expected circle frame, got %q", frame)
	}
}

func TestSpinner_ContextCancel(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)

	ctx, cancel := context.WithCancel(context.Background())

	s.Start(ctx)

	if !s.IsRunning() {
		t.Error("spinner should be running after Start")
	}

	// Cancel context.
	cancel()

	// Wait for goroutine to notice.
	time.Sleep(50 * time.Millisecond)

	// Spinner should have stopped.
	if s.IsRunning() {
		t.Error("spinner should stop when context is canceled")
	}
}

func TestSpinner_ZeroInterval_NoPanic(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(0) // Would panic in NewTicker without guard.

	ctx := t.Context()

	s.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Goroutine should exit cleanly, not panic.
	if s.IsRunning() {
		t.Fatal("spinner should not be running with zero interval")
	}

	// Stop should not hang or panic.
	s.Stop()
}

func TestSpinner_NegativeInterval_NoPanic(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(-time.Second) // Would panic in NewTicker without guard.

	ctx := t.Context()

	s.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	if s.IsRunning() {
		t.Fatal("spinner should not be running with negative interval")
	}

	s.Stop()
}

func TestSpinner_StopStartRace(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(5 * time.Millisecond)

	ctx := t.Context()

	// Rapid Stop/Start cycles should not leave the spinner in a broken state.
	for range 20 {
		s.Start(ctx)
		s.Stop()
	}

	// Final Start should work.
	s.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	if !s.IsRunning() {
		t.Error("spinner should be running after final Start, stale goroutine may have clobbered running flag")
	}

	frame := s.Frame()
	if frame == "" {
		t.Error("expected non-empty frame from running spinner")
	}

	s.Stop()
}

func TestActivitySpinner_ShouldAnimate(t *testing.T) {
	t.Parallel()

	as := NewActivitySpinner(StyleDots)

	tests := []struct {
		state    string
		expected bool
	}{
		{"Starting", true},
		{"Thinking", true},
		{"Executing", true},
		{"Calling tools", true},
		{"Calling: Bash", true}, // Prefix match (delimiter-based).
		{"Waiting approval", true},
		{"Ready", false},
		{"Idle", false},
		{"Complete", false},
		{"Error", false},
		// Non-delimiter states must NOT prefix-match.
		{"StartingCleanup", false},
		{"ExecutingShutdown", false},
		{"ThinkingAboutStopping", false},
		// Delimiter-based prefix matching still works.
		{"Calling: Read", true},
		{"Calling: Write", true},
	}

	for _, tt := range tests {
		result := as.ShouldAnimate(tt.state)
		if result != tt.expected {
			t.Errorf("ShouldAnimate(%q) = %v, want %v", tt.state, result, tt.expected)
		}
	}
}

func TestActivitySpinner_UpdateState(t *testing.T) {
	t.Parallel()

	as := NewActivitySpinner(StyleDots)

	ctx := t.Context()

	// Initially not running.
	if as.IsRunning() {
		t.Error("should not be running initially")
	}

	// Update to active state.
	as.UpdateState(ctx, "Thinking")
	time.Sleep(10 * time.Millisecond)

	if !as.IsRunning() {
		t.Error("should be running for Thinking state")
	}

	// Update to idle state.
	as.UpdateState(ctx, "Ready")
	time.Sleep(10 * time.Millisecond)

	if as.IsRunning() {
		t.Error("should stop for Ready state")
	}
}

func TestActivitySpinner_AddActiveState(t *testing.T) {
	t.Parallel()

	as := NewActivitySpinner(StyleDots)

	// Custom state not active by default.
	if as.ShouldAnimate("CustomState") {
		t.Error("CustomState should not be active by default")
	}

	// Add it.
	as.AddActiveState("CustomState")

	if !as.ShouldAnimate("CustomState") {
		t.Error("CustomState should be active after AddActiveState")
	}
}

func TestSpinner_ContextCancel_NoContextLeak(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)

	// Start many spinners with zero interval — each should clean up its context.
	for range 100 {
		ctx, cancel := context.WithCancel(context.Background())

		s.Start(ctx)
		time.Sleep(5 * time.Millisecond) // Let animate exit due to zero interval... but we have default interval.

		s.Stop()
		cancel()
	}

	// After Stop+cancel, the spinner should not be running and cancel should be nil.
	s.mu.RLock()
	cancelIsNil := s.cancel == nil
	s.mu.RUnlock()

	if !cancelIsNil {
		t.Error("cancel function should be nil after Stop, potential context leak")
	}
}

func TestSpinner_ZeroInterval_NoContextLeak(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(0)

	// Start with zero interval — animate exits immediately.
	s.Start(context.Background())
	time.Sleep(50 * time.Millisecond)

	// animate's defer should have cleaned up.
	s.mu.RLock()
	cancelIsNil := s.cancel == nil
	running := s.running
	s.mu.RUnlock()

	if running {
		t.Error("spinner should not be running with zero interval")
	}

	if !cancelIsNil {
		t.Error("cancel function should be nil after animate exits, context leak")
	}
}

func TestSpinner_SetInterval_WhileRunning(t *testing.T) {
	t.Parallel()

	s := NewSpinner(StyleDots)
	s.SetInterval(50 * time.Millisecond) // Initial interval.

	var callCount atomic.Int32

	s.SetUpdateCallback(func() {
		callCount.Add(1)
	})

	ctx := t.Context()

	s.Start(ctx)

	// Wait for the initial interval's in-flight timer to fire.
	time.Sleep(70 * time.Millisecond)

	// Change to fast interval.
	s.SetInterval(5 * time.Millisecond)

	callCount.Store(0) // Reset counter after interval change.

	// Wait long enough for the in-flight timer to expire and fast ticks to accumulate.
	time.Sleep(150 * time.Millisecond)

	s.Stop()

	count := callCount.Load()
	if count < 5 {
		t.Errorf("expected multiple fast callbacks after SetInterval, got %d", count)
	}
}

func TestActivitySpinner_UpdateState_Concurrent(t *testing.T) {
	t.Parallel()

	as := NewActivitySpinner(StyleDots)
	as.SetInterval(5 * time.Millisecond)

	ctx := t.Context()

	// Run many concurrent UpdateState calls to verify no races.
	done := make(chan struct{})

	go func() {
		defer close(done)

		for range 100 {
			as.UpdateState(ctx, "Thinking")
			as.UpdateState(ctx, "Ready")
		}
	}()

	for range 100 {
		as.UpdateState(ctx, "Executing")
		as.UpdateState(ctx, "Idle")
	}

	<-done

	// Final state: set to a known state and verify.
	as.UpdateState(ctx, "Thinking")
	time.Sleep(20 * time.Millisecond)

	if !as.IsRunning() {
		t.Error("spinner should be running after final UpdateState(Thinking)")
	}

	as.UpdateState(ctx, "Ready")
	time.Sleep(20 * time.Millisecond)

	if as.IsRunning() {
		t.Error("spinner should be stopped after final UpdateState(Ready)")
	}
}
