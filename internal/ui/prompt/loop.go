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
	model    *Model
	renderer PromptRenderer
	keys     <-chan term.KeyEvent
	out      chan string
}

// NewLoop creates a new input loop with the specified components.
// The loop does not start until Run() is called.
func NewLoop(model *Model, renderer PromptRenderer, keys <-chan term.KeyEvent) *Loop {
	return &Loop{
		model:    model,
		renderer: renderer,
		keys:     keys,
		out:      make(chan string, 1),
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
	switch event.Kind {
	case term.KeyRune:
		l.model.Insert(event.Rune)
		l.renderer.Redraw(l.model, "")

	case term.KeyBackspace:
		l.model.Backspace()
		l.renderer.Redraw(l.model, "")

	case term.KeyDelete:
		l.model.Delete()
		l.renderer.Redraw(l.model, "")

	case term.KeyLeft:
		l.model.MoveLeft()
		l.renderer.Redraw(l.model, "")

	case term.KeyRight:
		l.model.MoveRight()
		l.renderer.Redraw(l.model, "")

	case term.KeyHome:
		l.model.MoveStart()
		l.renderer.Redraw(l.model, "")

	case term.KeyEnd:
		l.model.MoveEnd()
		l.renderer.Redraw(l.model, "")

	case term.KeyUp:
		l.model.PrevHistory()
		l.renderer.Redraw(l.model, "")

	case term.KeyDown:
		l.model.NextHistory()
		l.renderer.Redraw(l.model, "")

	case term.KeyCtrlU:
		l.model.ClearLineLeft()
		l.renderer.Redraw(l.model, "")

	case term.KeyCtrlK:
		l.model.ClearLineRight()
		l.renderer.Redraw(l.model, "")

	case term.KeyCtrlW:
		l.model.DeleteWord()
		l.renderer.Redraw(l.model, "")

	case term.KeyCtrlL:
		// Clear screen and redraw
		l.renderer.ClearScreen()
		l.renderer.Redraw(l.model, "")

	case term.KeyEnter:
		line := l.model.Submit()
		l.renderer.Redraw(l.model, "")
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
		l.renderer.Redraw(l.model, "")

	case term.KeyPaste:
		// Insert paste content rune by rune
		for _, r := range []rune(string(event.Paste)) {
			l.model.Insert(r)
		}
		l.renderer.Redraw(l.model, "")

	default:
		// Unknown key, ignore
	}

	return false // Continue loop
}
