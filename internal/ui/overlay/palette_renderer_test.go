package overlay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPaletteRenderer(t *testing.T) {
	t.Parallel()
	renderer := NewPaletteRenderer(80, 24)
	assert.NotNil(t, renderer)
}

func TestPaletteRenderer_SetSize(t *testing.T) {
	t.Parallel()
	renderer := NewPaletteRenderer(80, 24)
	renderer.SetSize(120, 40)
	// No assertion needed - just verify it doesn't panic.
}

func TestPaletteRenderer_Render_Empty(t *testing.T) {
	t.Parallel()
	registry := NewCommandRegistry() // Empty.
	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	assert.Contains(t, output, "Command Palette")
	assert.Contains(t, output, "[Esc]")
	assert.Contains(t, output, "❯")
	assert.Contains(t, output, "No commands available")
}

func TestPaletteRenderer_Render_WithCommands(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	assert.Contains(t, output, "Command Palette")
	assert.Contains(t, output, "Run...")
	assert.Contains(t, output, "Search in repo...")
	assert.Contains(t, output, "Edit")
	assert.Contains(t, output, "Tools")
}

func TestPaletteRenderer_Render_WithQuery(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()
	palette.Insert('s')
	palette.Insert('e')
	palette.Insert('a')
	palette.Insert('r')
	palette.Insert('c')
	palette.Insert('h')

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	assert.Contains(t, output, "search_") // Query with cursor.
	assert.Contains(t, output, "Search in repo...")
}

func TestPaletteRenderer_Render_NoMatch(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()
	palette.Insert('x')
	palette.Insert('y')
	palette.Insert('z')

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	assert.Contains(t, output, "No commands match 'xyz'")
}

func TestPaletteRenderer_Render_Selection(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()
	palette.MoveDown() // Select second item.

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	// Selection should be visually distinct (inverted).
	assert.Contains(t, output, colorInvert)
	assert.Contains(t, output, "Search in repo...")
}

func TestPaletteRenderer_Render_SmallTerminal(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(40, 10)
	output := renderer.Render(palette)

	// Should not panic, should render something.
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Command Palette")
}

func TestPaletteRenderer_Render_LargeTerminal(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(200, 60)
	output := renderer.Render(palette)

	// Should cap at 80 chars width per spec (plus centering padding)
	// Palette width = min(80, 200 - 2*s4) = 80
	// With centering: (200 - 80) / 2 = 60 chars left padding
	// Total line length can be up to 200 + some ANSI codes.
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Command Palette")
}

func TestPaletteRenderer_Render_MultipleItems(t *testing.T) {
	t.Parallel()
	registry := NewCommandRegistry()
	for i := range 20 {
		registry.Register(NewSimpleCommand(
			"Command "+string(rune('A'+i)),
			"Description "+string(rune('A'+i)),
			"Category",
			'•',
			nil,
		))
	}

	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	// Should render, but may truncate items based on maxHeight.
	assert.Contains(t, output, "Command A")
	assert.NotEmpty(t, output)
}

func TestPaletteRenderer_RenderItem(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	commands := registry.Commands()

	renderer := NewPaletteRenderer(80, 24)

	// Test unselected item.
	output := renderer.renderItem(commands[0], false, 80, 0)
	assert.Contains(t, output, "Run...")
	assert.Contains(t, output, "Edit")
	assert.NotContains(t, output, colorInvert)

	// Test selected item.
	output = renderer.renderItem(commands[0], true, 80, 0)
	assert.Contains(t, output, "Run...")
	assert.Contains(t, output, colorInvert)
}

func TestMin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 5, min(5, 5))
	assert.Equal(t, 0, min(0, 10))
	assert.Equal(t, -5, min(-5, 5))
}

// stripANSI removes ANSI escape codes for testing purposes.

func TestPaletteRenderer_Integration(t *testing.T) {
	t.Parallel(
	// Full integration test: registry → palette → renderer.
	)

	registry := NewCommandRegistry()
	executed := false

	registry.Register(NewSimpleCommand(
		"Test Command",
		"Test description",
		"Test",
		'T',
		func(_ context.Context) error {
			executed = true

			return nil
		},
	))

	palette := NewPalette(registry)
	palette.Open()

	renderer := NewPaletteRenderer(80, 24)
	output := renderer.Render(palette)

	// Verify rendering.
	assert.Contains(t, output, "Test Command")

	// Execute command.
	cmd := palette.SelectedCommand()
	assert.NotNil(t, cmd)
	err := cmd.Execute(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)
}
