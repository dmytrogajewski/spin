package conversation

import (
	"github.com/dmytrogajewski/spin/internal/history"
)

// createHistory creates a new history instance with event emitter configured.
func (b *Builder) createHistory() *history.History {
	h := history.NewHistoryWithDefaults()
	h.SetEventEmitter(b.emitter)
	_ = h.AddSystemMessage("You are a helpful AI coding assistant.")
	return h
}
