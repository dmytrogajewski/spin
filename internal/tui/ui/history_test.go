package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHistory(t *testing.T) {
	h := NewHistory(10)

	assert.NotNil(t, h)
	assert.Equal(t, 10, h.maxSize)
	assert.Equal(t, 0, len(h.items))
	assert.Equal(t, -1, h.position)
}

func TestHistory_Add(t *testing.T) {
	h := NewHistory(10)

	h.Add("first")
	h.Add("second")

	assert.Len(t, h.items, 2)
	assert.Equal(t, "first", h.items[0])
	assert.Equal(t, "second", h.items[1])
}

func TestHistory_Add_EmptyString(t *testing.T) {
	h := NewHistory(10)

	h.Add("")

	assert.Len(t, h.items, 0)
}

func TestHistory_Add_Deduplication(t *testing.T) {
	h := NewHistory(10)

	h.Add("duplicate")
	h.Add("other")
	h.Add("duplicate") // Should remove old "duplicate"

	assert.Len(t, h.items, 2)
	assert.Equal(t, "other", h.items[0])
	assert.Equal(t, "duplicate", h.items[1]) // Moved to end
}

func TestHistory_Add_MaxSize(t *testing.T) {
	h := NewHistory(3)

	h.Add("first")
	h.Add("second")
	h.Add("third")
	h.Add("fourth") // Should evict "first"

	assert.Len(t, h.items, 3)
	assert.Equal(t, "second", h.items[0])
	assert.Equal(t, "third", h.items[1])
	assert.Equal(t, "fourth", h.items[2])
}

func TestHistory_Previous_Empty(t *testing.T) {
	h := NewHistory(10)

	prev, ok := h.Previous()

	assert.False(t, ok)
	assert.Equal(t, "", prev)
}

func TestHistory_Previous_FromCurrent(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	prev, ok := h.Previous()

	assert.True(t, ok)
	assert.Equal(t, "second", prev)
	assert.Equal(t, 1, h.position)
}

func TestHistory_Previous_Multiple(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	// First call: go to most recent
	prev, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "third", prev)

	// Second call: go backward
	prev, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "second", prev)

	// Third call: go to oldest
	prev, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "first", prev)

	// Fourth call: at oldest, returns same
	prev, ok = h.Previous()
	assert.False(t, ok)
	assert.Equal(t, "first", prev)
}

func TestHistory_Next_Empty(t *testing.T) {
	h := NewHistory(10)

	next, ok := h.Next()

	assert.False(t, ok)
	assert.Equal(t, "", next)
}

func TestHistory_Next_NotInHistory(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")

	// Not navigating yet
	next, ok := h.Next()

	assert.False(t, ok)
	assert.Equal(t, "", next)
}

func TestHistory_Next_AfterPrevious(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	// Navigate backward
	h.Previous() // third
	h.Previous() // second
	h.Previous() // first

	// Navigate forward
	next, ok := h.Next()
	assert.True(t, ok)
	assert.Equal(t, "second", next)

	next, ok = h.Next()
	assert.True(t, ok)
	assert.Equal(t, "third", next)
}

func TestHistory_Next_ToTempBuffer(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	h.SetTempBuffer("current input")

	// Navigate backward
	h.Previous() // second
	h.Previous() // first

	// Navigate forward to temp buffer
	h.Next()             // second
	next, ok := h.Next() // should return temp buffer

	assert.True(t, ok)
	assert.Equal(t, "current input", next)
	assert.Equal(t, -1, h.position) // Back to current
}

func TestHistory_Reset(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")

	h.Previous()
	assert.Equal(t, 0, h.position)

	h.Reset()

	assert.Equal(t, -1, h.position)
	assert.Equal(t, "", h.tempBuf)
}

func TestHistory_SetTempBuffer(t *testing.T) {
	h := NewHistory(10)

	h.SetTempBuffer("temp value")

	assert.Equal(t, "temp value", h.tempBuf)
}

func TestHistory_GetAll(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	all := h.GetAll()

	assert.Len(t, all, 2)
	assert.Equal(t, "first", all[0])
	assert.Equal(t, "second", all[1])
}

func TestHistory_Clear(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")

	h.Clear()

	assert.Len(t, h.items, 0)
	assert.Equal(t, -1, h.position)
	assert.Equal(t, "", h.tempBuf)
}

func TestHistory_Navigation_FullCycle(t *testing.T) {
	h := NewHistory(10)
	h.Add("first")
	h.Add("second")
	h.Add("third")

	h.SetTempBuffer("current")

	// Go back through history
	prev, ok := h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "third", prev)

	prev, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "second", prev)

	prev, ok = h.Previous()
	assert.True(t, ok)
	assert.Equal(t, "first", prev)

	// Go forward through history
	next, ok := h.Next()
	assert.True(t, ok)
	assert.Equal(t, "second", next)

	next, ok = h.Next()
	assert.True(t, ok)
	assert.Equal(t, "third", next)

	// Back to current
	next, ok = h.Next()
	assert.True(t, ok)
	assert.Equal(t, "current", next)

	// No more forward
	_, ok = h.Next()
	assert.False(t, ok)
}
