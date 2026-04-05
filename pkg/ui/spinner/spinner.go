// Package spinner provides terminal spinner animations for indicating progress.
package spinner

import (
	"context"
	"strings"
	"sync"
	"time"
)

const defaultSpinnerInterval = 100 * time.Millisecond

// Style defines the animation frames for the spinner.
type Style int

const (
	// StyleDots uses a dots animation: ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏.
	StyleDots Style = iota
	// StyleBraille uses a braille spinner: ⣾ ⣽ ⣻ ⢿ ⡿ ⣟ ⣯ ⣷.
	StyleBraille
	// StyleCircle uses a simple circle: ◐ ◓ ◑ ◒.
	StyleCircle
	// SpinnerPulse uses a pulsing dot: · • ● ◉ ● • ·.
	SpinnerPulse
)

// spinnerFrames defines animation frames for each style.
var spinnerFrames = map[Style][]string{
	StyleDots:    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	StyleBraille: {"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	StyleCircle:  {"◐", "◓", "◑", "◒"},
	SpinnerPulse: {"·", "•", "●", "◉", "●", "•", "·", " "},
}

// Spinner provides an animated loading indicator for the TUI.
type Spinner struct {
	mu       sync.RWMutex
	style    Style
	frames   []string
	index    int
	running  bool
	interval time.Duration
	onUpdate func() // Callback to trigger UI refresh.
	gen      uint64 // Generation counter to prevent stale goroutine cleanup.

	cancel context.CancelFunc
}

// NewSpinner creates a new spinner with the given style.
func NewSpinner(style Style) *Spinner {
	frames, ok := spinnerFrames[style]
	if !ok {
		frames = spinnerFrames[StyleDots]
	}

	return &Spinner{
		style:    style,
		frames:   frames,
		interval: defaultSpinnerInterval, // Default animation speed.
	}
}

// SetStyle changes the spinner animation style.
func (s *Spinner) SetStyle(style Style) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if frames, ok := spinnerFrames[style]; ok {
		s.style = style
		s.frames = frames
		s.index = 0
	}
}

// SetInterval sets the animation interval.
// Takes effect on the next animation tick if the spinner is running.
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

// startLocked begins the spinner animation. Caller must hold s.mu.
func (s *Spinner) startLocked(ctx context.Context) {
	if s.running {
		return
	}

	s.running = true
	s.index = 0
	s.gen++

	// Create cancelable context.
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	gen := s.gen

	// Start animation goroutine.
	go s.animate(ctx, gen)
}

// stopLocked stops the spinner animation. Caller must hold s.mu.
func (s *Spinner) stopLocked() {
	if !s.running {
		return
	}

	s.running = false
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startLocked(ctx)
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopLocked()
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
func (s *Spinner) animate(ctx context.Context, gen uint64) {
	// Cleanup must run even on early exit to release the child context
	// and reset state for this generation.
	defer s.cleanupGeneration(gen)

	for {
		// Read current interval each tick so SetInterval takes effect dynamically.
		s.mu.RLock()
		interval := s.interval
		s.mu.RUnlock()

		if interval <= 0 {
			return
		}

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()

			return
		case <-timer.C:
			if !s.tick() {
				return
			}
		}
	}
}

// cleanupGeneration resets state for the given generation on goroutine exit.
func (s *Spinner) cleanupGeneration(gen uint64) {
	s.mu.Lock()
	if s.gen == gen {
		s.running = false
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
	}

	s.mu.Unlock()
}

// tick advances the animation frame and triggers the update callback.
// Returns false if the spinner was stopped and the loop should exit.
func (s *Spinner) tick() bool {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()

		return false
	}

	s.index = (s.index + 1) % len(s.frames)
	callback := s.onUpdate
	s.mu.Unlock()

	if callback != nil {
		callback()
	}

	return true
}

// ActivitySpinner wraps Spinner with activity state awareness.
// It automatically starts/stops based on agent state.
type ActivitySpinner struct {
	*Spinner

	activeStates map[string]bool // States that trigger animation.
}

// NewActivitySpinner creates a spinner that activates for certain states.
func NewActivitySpinner(style Style) *ActivitySpinner {
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

// shouldAnimateLocked returns true if the given state should trigger animation.
// Caller must hold a.mu (at least RLock).
func (a *ActivitySpinner) shouldAnimateLocked(state string) bool {
	// Check exact match.
	if a.activeStates[state] {
		return true
	}

	// Only prefix-match states that end with a delimiter (e.g. "Calling:").
	for activeState := range a.activeStates {
		if strings.HasSuffix(activeState, ":") && strings.HasPrefix(state, activeState) {
			return true
		}
	}

	return false
}

// ShouldAnimate returns true if the given state should trigger animation.
func (a *ActivitySpinner) ShouldAnimate(state string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.shouldAnimateLocked(state)
}

// UpdateState updates the spinner based on the new agent state.
// Starts animation for active states, stops for idle states.
// The entire check-and-act is atomic to prevent TOCTOU races.
func (a *ActivitySpinner) UpdateState(ctx context.Context, state string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	shouldAnimate := a.shouldAnimateLocked(state)

	if shouldAnimate && !a.running {
		a.startLocked(ctx)
	} else if !shouldAnimate && a.running {
		a.stopLocked()
	}
}
