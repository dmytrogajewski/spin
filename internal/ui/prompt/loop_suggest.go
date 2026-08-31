package prompt

import (
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/ui/suggest"
)

type hintRenderer interface {
	SetHints(lines []string, selected int)
}

// SetSource enables / and @ suggestions.
func (l *Loop) SetSource(src *suggest.Source) {
	l.source = src
}

// SetWorkDir is used when classifying paste paths.
func (l *Loop) SetWorkDir(dir string) {
	l.workDir = dir
}

// SetClipper sets the Ctrl-V clipboard reader.
func (l *Loop) SetClipper(clip suggest.Clipper) {
	l.clipper = clip
}

func (l *Loop) handleTab() {
	if len(l.items) == 0 {
		l.refreshHints()
	}

	if len(l.items) == 0 {
		return
	}

	item := l.items[l.selected]
	tok := suggest.TokenAt(l.model.Text(), l.model.Cursor())
	next, cur := suggest.Apply(l.model.Text(), tok, item)
	l.model.Replace(next, cur)
	l.redraw()
}

func (l *Loop) handleEscape() {
	if len(l.items) == 0 {
		return
	}

	l.clearHints()
	l.redraw()
}

func (l *Loop) handleCtrlV() {
	if l.clipper == nil {
		return
	}

	raw, err := l.clipper()
	if err != nil || len(raw) == 0 {
		return
	}

	l.insertPaste(raw)
	l.redraw()
}

func (l *Loop) insertPaste(raw []byte) {
	got := suggest.ClassifyPaste(raw, l.workDir)

	text := got.Text
	if text == "" {
		text = string(raw)
	}

	for _, r := range text {
		if r == 0 {
			continue
		}

		l.model.Insert(r)
	}
}

func (l *Loop) moveHint(delta int) bool {
	if len(l.items) == 0 {
		return false
	}

	l.selected += delta
	if l.selected < 0 {
		l.selected = 0
	}

	if l.selected >= len(l.items) {
		l.selected = len(l.items) - 1
	}

	return true
}

func (l *Loop) refreshHints() {
	if l.source == nil {
		l.items = nil
		l.selected = 0

		return
	}

	l.items = l.source.Items(l.model.Text(), l.model.Cursor())
	if l.selected >= len(l.items) {
		l.selected = 0
	}
}

func (l *Loop) clearHints() {
	l.items = nil
	l.selected = 0
}

func (l *Loop) applyHintRenderer() {
	h, ok := l.renderer.(hintRenderer)
	if !ok {
		return
	}

	h.SetHints(l.hintLines(), l.selected)
}

func (l *Loop) hintLines() []string {
	lines := make([]string, 0, len(l.items))
	for _, item := range l.items {
		detail := strings.TrimSpace(item.Detail)
		if detail == "" {
			lines = append(lines, "  "+item.Label)

			continue
		}

		lines = append(lines, fmt.Sprintf("  %s  %s", item.Label, detail))
	}

	return lines
}
