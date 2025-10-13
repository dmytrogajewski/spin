package prompt

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// PromptRenderer is the interface for rendering the prompt.
// It allows for testing with fake renderers.
type PromptRenderer interface {
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
	renderer   PromptRenderer
	keys       <-chan term.KeyEvent
	out        chan string
	onRender   func() // Callback to trigger re-render (for sticky coordination)
	skipRender bool   // If true, skip direct rendering (use callback only)
}

// NewLoop creates a new input loop with the specified components.
// The loop does not start until Run() is called.
func NewLoop(model *Model, renderer PromptRenderer, keys <-chan term.KeyEvent) *Loop {
	return &Loop{
		model:      model,
		renderer:   renderer,
		keys:       keys,
		out:        make(chan string, 1),
		skipRender: false,
	}
}

// SetRenderCallback sets a callback to be called when render is needed.
// If skipDirect is true, the loop will call the callback instead of rendering directly.
func (l *Loop) SetRenderCallback(callback func(), skipDirect bool) {
	l.onRender = callback
	l.skipRender = skipDirect
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
	switch event.Kind {
	case term.KeyRune:
		l.model.Insert(event.Rune)
		l.redraw()

	case term.KeyBackspace:
		l.model.Backspace()
		l.redraw()

	case term.KeyDelete:
		l.model.Delete()
		l.redraw()

	case term.KeyLeft:
		l.model.MoveLeft()
		l.redraw()

	case term.KeyRight:
		l.model.MoveRight()
		l.redraw()

	case term.KeyHome:
		l.model.MoveStart()
		l.redraw()

	case term.KeyEnd:
		l.model.MoveEnd()
		l.redraw()

	case term.KeyUp:
		l.model.PrevHistory()
		l.redraw()

	case term.KeyDown:
		l.model.NextHistory()
		l.redraw()

	case term.KeyCtrlU:
		l.model.ClearLineLeft()
		l.redraw()

	case term.KeyCtrlK:
		l.model.ClearLineRight()
		l.redraw()

	case term.KeyCtrlW:
		l.model.DeleteWord()
		l.redraw()

	case term.KeyCtrlL:
		// Clear screen and redraw
		l.renderer.ClearScreen()
		l.redraw()

	case term.KeyEnter:
		line := l.model.Submit()
		l.redraw()
		select {
		case l.out <- line:
		case <-ctx.Done():
			return true // Exit on context cancel
		}

	case term.KeyCtrlC:
		// Exit loop on Ctrl-C
		return true

	case term.KeyCtrlD:
		// EOF on empty buffer, otherwise delete
		if l.model.Text() == "" {
			return true // Exit on EOF
		}
		l.model.Delete()
		l.redraw()

	case term.KeyPaste:
		// Insert paste content rune by rune
		for _, r := range []rune(string(event.Paste)) {
			l.model.Insert(r)
		}
		l.redraw()

	default:
		// Unknown key, ignore
	}

	return false // Continue loop
}

// redraw triggers a prompt redraw, either via callback or directly.
func (l *Loop) redraw() {
	if l.skipRender && l.onRender != nil {
		// Call callback to trigger coordinated render
		l.onRender()
	} else {
		// Direct render (backward compatibility)
		l.renderer.Redraw(l.model, "")
	}
}
