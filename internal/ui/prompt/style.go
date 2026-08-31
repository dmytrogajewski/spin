package prompt

const (
	// DefaultPrefix is the live TUI prompt marker (1 cell + trailing space).
	DefaultPrefix = "→ "

	// ColorPromptBar is the dim grey background for the input box.
	ColorPromptBar = "\x1b[48;5;234m"

	// InputBarGapLines is the blank row between the status line and the box.
	InputBarGapLines = 1

	// ColorUserEcho is cyan for submitted user lines in the transcript.
	ColorUserEcho = "\x1b[38;5;51m"

	// ColorCursorCell is a white block painted on the input bar (hardware
	// cursor stays hidden in the scroll region).
	ColorCursorCell = "\x1b[48;5;15m"

	// InputBarPad is the left/right inset inside the grey input bar.
	InputBarPad = 3

	// InputBarLines is the painted height of the grey input box (pad, text, pad).
	InputBarLines = 3

	ansiReset = "\x1b[0m"

	// colorHintSel inverts the selected completion row.
	colorHintSel = "\x1b[7m"
)

// SetInputBar paints the prompt line with a full-width grey background.
func (r *TermRenderer) SetInputBar(on bool) {
	r.inputBar = on
}
