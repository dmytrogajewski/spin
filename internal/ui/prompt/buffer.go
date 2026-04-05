package prompt

import "github.com/dmytrogajewski/spin/pkg/ui/textbuffer"

// Buffer is an alias for [textbuffer.Buffer].
type Buffer = textbuffer.Buffer

// NewBuffer creates a new empty text buffer.
func NewBuffer() *Buffer {
	return textbuffer.NewBuffer()
}
