package status

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSpinner_Frame(t *testing.T) {
	t.Parallel()

	s := NewSpinner(SpinnerDots)

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

	s := NewSpinner(SpinnerCircle)

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

	s := NewSpinner(SpinnerBraille)

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

	s := NewSpinner(SpinnerDots)
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

	styles := []SpinnerStyle{
		SpinnerDots,
		SpinnerBraille,
		SpinnerCircle,
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

	s := NewSpinner(SpinnerDots)

	// Change style.
	s.SetStyle(SpinnerCircle)

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

	s := NewSpinner(SpinnerDots)

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

func TestActivitySpinner_ShouldAnimate(t *testing.T) {
	t.Parallel()

	as := NewActivitySpinner(SpinnerDots)

	tests := []struct {
		state    string
		expected bool
	}{
		{"Starting", true},
		{"Thinking", true},
		{"Executing", true},
		{"Calling tools", true},
		{"Calling: Bash", true}, // Prefix match.
		{"Waiting approval", true},
		{"Ready", false},
		{"Idle", false},
		{"Complete", false},
		{"Error", false},
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

	as := NewActivitySpinner(SpinnerDots)

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

	as := NewActivitySpinner(SpinnerDots)

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
