package prompt

import "github.com/dmytrogajewski/spin/pkg/ui/textbuffer"

// History is an alias for [textbuffer.History].
type History = textbuffer.History

// NewHistory creates a new command history with the given limit.
func NewHistory(limit int) *History {
	return textbuffer.NewHistory(limit)
}
