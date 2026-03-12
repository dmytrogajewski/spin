package status

import (
	"context"
	"strings"
	"sync"
	"time"
)

const defaultSpinnerInterval = 100 * time.Millisecond

// SpinnerStyle defines the animation frames for the spinner.
type SpinnerStyle int

const (
	// SpinnerDots uses a dots animation: ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏.
	SpinnerDots SpinnerStyle = iota
	// SpinnerBraille uses a braille spinner: ⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷.
	SpinnerBraille
	// SpinnerCircle uses a simple circle: ◐ ◓ ◑ ◒.
	SpinnerCircle
	// SpinnerPulse uses a pulsing dot: · • ● ◉ ● • ·.
	SpinnerPulse
)

// spinnerFrames defines animation frames for each style.
var spinnerFrames = map[SpinnerStyle][]string{
	SpinnerDots:    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	SpinnerBraille: {"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	SpinnerCircle:  {"◐", "◓", "◑", "◒"},
	SpinnerPulse:   {"·", "•", "●", "◉", "●", "•", "·", " "},
}

// Spinner provides an animated loading indicator for the TUI.
type Spinner struct {
	mu       sync.RWMutex
	style    SpinnerStyle
	frames   []string
	index    int
	running  bool
	interval time.Duration
	onUpdate func() // Callback to trigger UI refresh.

	cancel context.CancelFunc
}

// NewSpinner creates a new spinner with the given style.
func NewSpinner(style SpinnerStyle) *Spinner {
	frames, ok := spinnerFrames[style]
	if !ok {
		frames = spinnerFrames[SpinnerDots]
	}

	return &Spinner{
		style:    style,
		frames:   frames,
		interval: defaultSpinnerInterval, // Default animation speed.
	}
}

// SetStyle changes the spinner animation style.
func (s *Spinner) SetStyle(style SpinnerStyle) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if frames, ok := spinnerFrames[style]; ok {
		s.style = style
		s.frames = frames
		s.index = 0
	}
}

// SetInterval sets the animation interval.
func (s *Spinner) SetInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interval = d
}

// SetUpdateCallback sets a callback that's called on each animation frame.
// This is used to trigger UI refreshes.
func (s *Spinner) SetUpdateCallback(callback func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.onUpdate = callback
}

// Start begins the spinner animation.
func (s *Spinner) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()

		return
	}

	s.running = true
	s.index = 0

	// Create cancelable context.
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	interval := s.interval
	s.mu.Unlock()

	// Start animation goroutine.
	go s.animate(ctx, interval)
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// IsRunning returns true if the spinner is animating.
func (s *Spinner) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running
}

// Frame returns the current animation frame.
func (s *Spinner) Frame() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running || len(s.frames) == 0 {
		return ""
	}

	return s.frames[s.index%len(s.frames)]
}

// animate runs the animation loop.
func (s *Spinner) animate(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Ensure running is set to false when animation stops.
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()

				return
			}

			s.index = (s.index + 1) % len(s.frames)
			callback := s.onUpdate
			s.mu.Unlock()

			// Trigger UI refresh if callback is set.
			if callback != nil {
				callback()
			}
		}
	}
}

// ActivitySpinner wraps Spinner with activity state awareness.
// It automatically starts/stops based on agent state.
type ActivitySpinner struct {
	*Spinner

	activeStates map[string]bool // States that trigger animation.
}

// NewActivitySpinner creates a spinner that activates for certain states.
func NewActivitySpinner(style SpinnerStyle) *ActivitySpinner {
	return &ActivitySpinner{
		Spinner: NewSpinner(style),
		activeStates: map[string]bool{
			"Starting":         true,
			"Thinking":         true,
			"Executing":        true,
			"Calling tools":    true,
			"Calling:":         true, // Prefix for "Calling: <toolname>".
			"Waiting approval": true,
		},
	}
}

// AddActiveState adds a state that should trigger animation.
func (a *ActivitySpinner) AddActiveState(state string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.activeStates[state] = true
}

// ShouldAnimate returns true if the given state should trigger animation.
func (a *ActivitySpinner) ShouldAnimate(state string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check exact match.
	if a.activeStates[state] {
		return true
	}

	// Check prefix match for states like "Calling: <toolname>".
	for activeState := range a.activeStates {
		if strings.HasPrefix(state, activeState) {
			return true
		}
	}

	return false
}

// UpdateState updates the spinner based on the new agent state.
// Starts animation for active states, stops for idle states.
func (a *ActivitySpinner) UpdateState(ctx context.Context, state string) {
	shouldAnimate := a.ShouldAnimate(state)
	isRunning := a.IsRunning()

	if shouldAnimate && !isRunning {
		a.Start(ctx)
	} else if !shouldAnimate && isRunning {
		a.Stop()
	}
}
