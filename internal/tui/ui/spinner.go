package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner represents an animated loading indicator.
type Spinner struct {
	frames    []string
	frameIdx  int
	lastTick  time.Time
	interval  time.Duration
	active    bool
	style     lipgloss.Style
}

// NewSpinner creates a new spinner with default settings.
func NewSpinner() Spinner {
	// Classic spinning dots
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	return Spinner{
		frames:   frames,
		frameIdx: 0,
		lastTick: time.Now(),
		interval: 80 * time.Millisecond,
		active:   false,
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("12")), // Blue
	}
}

// Start activates the spinner.
func (s *Spinner) Start() {
	s.active = true
	s.frameIdx = 0
	s.lastTick = time.Now()
}

// Stop deactivates the spinner.
func (s *Spinner) Stop() {
	s.active = false
}

// IsActive returns whether the spinner is active.
func (s Spinner) IsActive() bool {
	return s.active
}

// Tick advances the spinner animation if enough time has passed.
func (s *Spinner) Tick() {
	if !s.active {
		return
	}

	now := time.Now()
	if now.Sub(s.lastTick) >= s.interval {
		s.frameIdx = (s.frameIdx + 1) % len(s.frames)
		s.lastTick = now
	}
}

// View renders the current spinner frame.
func (s Spinner) View() string {
	if !s.active {
		return ""
	}
	return s.style.Render(s.frames[s.frameIdx])
}

// ViewWithText renders the spinner with accompanying text.
func (s Spinner) ViewWithText(text string) string {
	if !s.active {
		return ""
	}
	return s.style.Render(s.frames[s.frameIdx]) + " " + text
}
