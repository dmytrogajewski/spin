package prompt

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ui/suggest"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// Renderer is the interface for rendering the prompt.
// It allows for testing with fake renderers.
type Renderer interface {
	Redraw(model *Model, status string) error
	ClearScreen() error
	SetWidth(width int)
	SetPrefix(prefix string)
}

// Loop coordinates keyboard input, model mutations, and rendering.
// It runs an event loop that processes key events, updates the model,
// and triggers redraws.
type Loop struct {
	model      *Model
	renderer   Renderer
	keys       <-chan term.KeyEvent
	out        chan string
	onRender   func() // Callback to trigger re-render (for sticky coordination).
	skipRender bool   // If true, skip direct rendering (use callback only).
	source     *suggest.Source
	clipper    suggest.Clipper
	items      []suggest.Item
	selected   int
	workDir    string
}

// NewLoop creates a new input loop with the specified components.
// The loop does not start until Run() is called.
func NewLoop(model *Model, renderer Renderer, keys <-chan term.KeyEvent) *Loop {
	return &Loop{
		model:      model,
		renderer:   renderer,
		keys:       keys,
		out:        make(chan string, 1),
		skipRender: false,
	}
}

// Run starts the input loop. It blocks until the context is canceled,
// the key channel closes, or a quit signal is received.
// Returns a channel that emits submitted lines.
func (l *Loop) Run(ctx context.Context) <-chan string {
	go l.loop(ctx)

	return l.out
}

func (l *Loop) loop(ctx context.Context) {
	defer close(l.out)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-l.keys:
			if !ok {
				return
			}

			if shouldExit := l.handleEvent(ctx, event); shouldExit {
				return
			}
		}
	}
}

func (l *Loop) handleEvent(ctx context.Context, event term.KeyEvent) bool {
	return l.dispatchKeyEvent(ctx, event)
}

// dispatchKeyEvent dispatches key events to appropriate handlers.
func (l *Loop) dispatchKeyEvent(ctx context.Context, event term.KeyEvent) bool {
	if done, ok := l.dispatchNavigation(event); ok {
		return done
	}

	if done, ok := l.dispatchEditing(event); ok {
		return done
	}

	return l.dispatchAction(ctx, event)
}

// dispatchNavigation handles cursor movement and history navigation keys.
func (l *Loop) dispatchNavigation(event term.KeyEvent) (done, handled bool) {
	switch event.Kind {
	case term.KeyLeft:
		return l.handleLeft(), true
	case term.KeyRight:
		return l.handleRight(), true
	case term.KeyHome:
		return l.handleHome(), true
	case term.KeyEnd:
		return l.handleEnd(), true
	case term.KeyUp:
		return l.handleUp(), true
	case term.KeyDown:
		return l.handleDown(), true
	case term.KeyTab:
		l.handleTab()

		return false, true
	case term.KeyEscape:
		l.handleEscape()

		return false, true
	default:
		return false, false
	}
}

// dispatchEditing handles text insertion and deletion keys.
func (l *Loop) dispatchEditing(event term.KeyEvent) (done, handled bool) {
	switch event.Kind {
	case term.KeyRune:
		return l.handleRune(event), true
	case term.KeyBackspace:
		return l.handleBackspace(), true
	case term.KeyDelete:
		return l.handleDelete(), true
	case term.KeyCtrlU:
		return l.handleCtrlU(), true
	case term.KeyCtrlK:
		return l.handleCtrlK(), true
	case term.KeyCtrlW:
		return l.handleCtrlW(), true
	case term.KeyPaste:
		return l.handlePaste(event), true
	case term.KeyCtrlV:
		l.handleCtrlV()

		return false, true
	default:
		return false, false
	}
}

// dispatchAction handles control keys that trigger actions (submit, cancel, clear).
func (l *Loop) dispatchAction(ctx context.Context, event term.KeyEvent) bool {
	switch event.Kind {
	case term.KeyCtrlL:
		return l.handleCtrlL()
	case term.KeyEnter:
		return l.handleEnter(ctx)
	case term.KeyCtrlC:
		return l.handleCtrlC()
	case term.KeyCtrlD:
		return l.handleCtrlD()
	default:
		return l.handleUnknown()
	}
}

// handleRune handles rune input.
func (l *Loop) handleRune(event term.KeyEvent) bool {
	l.model.Insert(event.Rune)
	l.redraw()

	return false
}

// handleBackspace handles backspace key.
func (l *Loop) handleBackspace() bool {
	l.model.Backspace()
	l.redraw()

	return false
}

// handleDelete handles delete key.
func (l *Loop) handleDelete() bool {
	l.model.Delete()
	l.redraw()

	return false
}

// handleLeft handles left arrow key.
func (l *Loop) handleLeft() bool {
	l.model.MoveLeft()
	l.redraw()

	return false
}

// handleRight handles right arrow key.
func (l *Loop) handleRight() bool {
	l.model.MoveRight()
	l.redraw()

	return false
}

// handleHome handles home key.
func (l *Loop) handleHome() bool {
	l.model.MoveStart()
	l.redraw()

	return false
}

// handleEnd handles end key.
func (l *Loop) handleEnd() bool {
	l.model.MoveEnd()
	l.redraw()

	return false
}

// handleUp handles up arrow key.
func (l *Loop) handleUp() bool {
	if !l.moveHint(-1) {
		l.model.PrevHistory()
	}

	l.redraw()

	return false
}

// handleDown handles down arrow key.
func (l *Loop) handleDown() bool {
	if !l.moveHint(1) {
		l.model.NextHistory()
	}

	l.redraw()

	return false
}

// handleCtrlU handles Ctrl+U (clear line left).
func (l *Loop) handleCtrlU() bool {
	l.model.ClearLineLeft()
	l.redraw()

	return false
}

// handleCtrlK handles Ctrl+K (clear line right).
func (l *Loop) handleCtrlK() bool {
	l.model.ClearLineRight()
	l.redraw()

	return false
}

// handleCtrlW handles Ctrl+W (delete word).
func (l *Loop) handleCtrlW() bool {
	l.model.DeleteWord()
	l.redraw()

	return false
}

// handleCtrlL handles Ctrl+L (clear screen).
func (l *Loop) handleCtrlL() bool {
	_ = l.renderer.ClearScreen()
	l.redraw()

	return false
}

// handleEnter handles enter key.
func (l *Loop) handleEnter(ctx context.Context) bool {
	l.clearHints()
	line := l.model.Submit()
	l.redraw()

	select {
	case l.out <- line:
	case <-ctx.Done():
		return true // Exit on context cancel.
	}

	return false
}

// handleCtrlC handles Ctrl+C (exit).
func (l *Loop) handleCtrlC() bool {
	return true // Exit loop on Ctrl-C.
}

// handleCtrlD handles Ctrl+D (EOF or delete).
func (l *Loop) handleCtrlD() bool {
	if l.model.Text() == "" {
		return true // Exit on EOF.
	}

	l.model.Delete()
	l.redraw()

	return false
}

// handlePaste handles paste events.
func (l *Loop) handlePaste(event term.KeyEvent) bool {
	l.insertPaste(event.Paste)
	l.redraw()

	return false
}

// handleUnknown handles unknown keys.
func (l *Loop) handleUnknown() bool {
	// Unknown key, ignore.
	return false
}

// redraw triggers a prompt redraw, either via callback or directly.
func (l *Loop) redraw() {
	l.refreshHints()
	l.applyHintRenderer()

	if l.skipRender && l.onRender != nil {
		// Call callback to trigger coordinated render.
		l.onRender()
	} else {
		// Direct render (backward compatibility).
		_ = l.renderer.Redraw(l.model, "")
	}
}
