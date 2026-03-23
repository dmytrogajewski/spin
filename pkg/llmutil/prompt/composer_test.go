package prompt

// Journey: specs/journeys/JOURNEY-S14.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComposer_priority_ordering(t *testing.T) {
	t.Parallel()

	comp := NewComposer()
	comp.AddSection(Section{Name: "low", Priority: 100, Template: "second"})
	comp.AddSection(Section{Name: "high", Priority: 0, Template: "first"})

	got := comp.Compose()
	require.Equal(t, "first\n\nsecond", got)
}

func TestComposer_conditional_sections(t *testing.T) {
	t.Parallel()

	comp := NewComposer()
	comp.AddSection(Section{Name: "always", Priority: 0, Template: "hello"})
	comp.AddSection(Section{Name: "never", Priority: 1, Template: "hidden", Active: func() bool { return false }})
	comp.AddSection(Section{Name: "active", Priority: 2, Template: "world", Active: func() bool { return true }})

	got := comp.Compose()
	require.Equal(t, "hello\n\nworld", got)
}

func TestComposer_var_substitution(t *testing.T) {
	t.Parallel()

	comp := NewComposer()
	comp.SetVar("NAME", "spin")
	comp.AddSection(Section{Name: "greeting", Priority: 0, Template: "Hello ${NAME}!"})

	got := comp.Compose()
	require.Equal(t, "Hello spin!", got)
}

func TestComposer_two_part(t *testing.T) {
	t.Parallel()

	comp := NewComposer()
	comp.AddSection(Section{Name: "stable", Priority: 0, Template: "cached", Cacheable: true})
	comp.AddSection(Section{Name: "dynamic", Priority: 1, Template: "fresh"})

	stable, dynamic := comp.ComposeTwoPart()
	require.Equal(t, "cached", stable)
	require.Equal(t, "fresh", dynamic)
}

func TestComposer_empty(t *testing.T) {
	t.Parallel()

	comp := NewComposer()

	got := comp.Compose()
	require.Empty(t, got)
}
