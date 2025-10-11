package term

// TerminalController defines the interface for terminal control operations.
// Both TTY and test fakes should implement this interface.
type TerminalController interface {
	// Enter puts the terminal into raw mode.
	Enter() error

	// Exit restores the terminal to its original state.
	Exit() error

	// Size returns the current terminal dimensions (width, height).
	Size() (int, int)

	// OnResize registers a callback to be invoked when the terminal is resized.
	OnResize(cb func(w, h int))
}

// Ensure TTY implements TerminalController
var _ TerminalController = (*TTY)(nil)
