package overlay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestRegistry() *CommandRegistry {
	registry := NewCommandRegistry()
	registry.Register(NewSimpleCommand("Run...", "Execute shell command", "Edit", '▶', nil))
	registry.Register(NewSimpleCommand("Search in repo...", "Grep/search files", "Tools", '🔍', nil))
	registry.Register(NewSimpleCommand("Open recent file...", "File picker", "File", '📄', nil))
	registry.Register(NewSimpleCommand("New plan...", "Create plan block", "Edit", '📋', nil))
	registry.Register(NewSimpleCommand("Toggle mode...", "Switch Auto/Manual", "System", '🔄', nil))
	registry.Register(NewSimpleCommand("Change theme...", "Switch Dark/Light", "System", '🎨', nil))

	return registry
}

func TestNewPalette(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)

	assert.NotNil(t, palette)
	assert.False(t, palette.IsOpen())
	assert.Equal(t, "", palette.Query())
	assert.Equal(t, 0, palette.Selection())
	// Should have all commands when query is empty.
	assert.Len(t, palette.FilteredCommands(), 6)
}

func TestPalette_OpenClose(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)

	// Initially closed.
	assert.False(t, palette.IsOpen())

	// Open.
	palette.Open()
	assert.True(t, palette.IsOpen())
	assert.Equal(t, "", palette.Query())
	assert.Equal(t, 0, palette.Selection())

	// Close.
	palette.Close()
	assert.False(t, palette.IsOpen())
}

func TestPalette_Insert(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	palette.Insert('s')
	assert.Equal(t, "s", palette.Query())

	palette.Insert('e')
	palette.Insert('a')
	palette.Insert('r')
	palette.Insert('c')
	palette.Insert('h')
	assert.Equal(t, "search", palette.Query())
}

func TestPalette_Backspace(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	palette.Insert('t')
	palette.Insert('e')
	palette.Insert('s')
	palette.Insert('t')
	assert.Equal(t, "test", palette.Query())

	palette.Backspace()
	assert.Equal(t, "tes", palette.Query())

	palette.Backspace()
	palette.Backspace()
	assert.Equal(t, "t", palette.Query())

	palette.Backspace()
	assert.Equal(t, "", palette.Query())

	// Backspace on empty does nothing.
	palette.Backspace()
	assert.Equal(t, "", palette.Query())
}

func TestPalette_ClearQuery(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	palette.Insert('t')
	palette.Insert('e')
	palette.Insert('s')
	palette.Insert('t')
	assert.Equal(t, "test", palette.Query())

	palette.ClearQuery()
	assert.Equal(t, "", palette.Query())
	assert.Equal(t, 0, palette.Selection())
}

func TestPalette_MoveUpDown_EmptyResults(t *testing.T) {
	t.Parallel()
	registry := NewCommandRegistry() // Empty registry.
	palette := NewPalette(registry)
	palette.Open()

	// Movement on empty results should be no-op.
	palette.MoveDown()
	assert.Equal(t, 0, palette.Selection())

	palette.MoveUp()
	assert.Equal(t, 0, palette.Selection())
}

func TestPalette_MoveDown(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	assert.Equal(t, 0, palette.Selection())

	palette.MoveDown()
	assert.Equal(t, 1, palette.Selection())

	palette.MoveDown()
	assert.Equal(t, 2, palette.Selection())
}

func TestPalette_MoveDown_Wrapping(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Move to last item.
	for range 5 {
		palette.MoveDown()
	}

	assert.Equal(t, 5, palette.Selection())

	// Wrap to first.
	palette.MoveDown()
	assert.Equal(t, 0, palette.Selection())
}

func TestPalette_MoveUp(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Move down a few times.
	palette.MoveDown()
	palette.MoveDown()
	palette.MoveDown()
	assert.Equal(t, 3, palette.Selection())

	// Move up.
	palette.MoveUp()
	assert.Equal(t, 2, palette.Selection())

	palette.MoveUp()
	assert.Equal(t, 1, palette.Selection())
}

func TestPalette_MoveUp_Wrapping(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	assert.Equal(t, 0, palette.Selection())

	// Wrap to last.
	palette.MoveUp()
	assert.Equal(t, 5, palette.Selection())
}

func TestPalette_SelectedCommand_NoSelection(t *testing.T) {
	t.Parallel()
	registry := NewCommandRegistry() // Empty.
	palette := NewPalette(registry)
	palette.Open()

	cmd := palette.SelectedCommand()
	assert.Nil(t, cmd)
}

func TestPalette_SelectedCommand(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Initially at 0.
	cmd := palette.SelectedCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "Run...", cmd.Name())

	// Move to 1.
	palette.MoveDown()
	cmd = palette.SelectedCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "Search in repo...", cmd.Name())

	// Move to 2.
	palette.MoveDown()
	cmd = palette.SelectedCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "Open recent file...", cmd.Name())
}

func TestPalette_FuzzySearch_EmptyQuery(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Empty query returns all commands.
	filtered := palette.FilteredCommands()
	assert.Len(t, filtered, 6)
}

func TestPalette_FuzzySearch_PartialMatch(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Search for "search".
	palette.Insert('s')
	palette.Insert('e')
	palette.Insert('a')
	palette.Insert('r')
	palette.Insert('c')
	palette.Insert('h')

	filtered := palette.FilteredCommands()
	assert.Greater(t, len(filtered), 0, "Should match at least 'Search in repo...'")

	// First result should be "Search in repo...".
	assert.Equal(t, "Search in repo...", filtered[0].Name())
}

func TestPalette_FuzzySearch_MultipleMatches(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Search for "mode" - should match "Toggle mode...".
	palette.Insert('m')
	palette.Insert('o')
	palette.Insert('d')
	palette.Insert('e')

	filtered := palette.FilteredCommands()
	assert.Greater(t, len(filtered), 0)

	found := false

	for _, cmd := range filtered {
		if cmd.Name() == "Toggle mode..." {
			found = true

			break
		}
	}

	assert.True(t, found, "Should find 'Toggle mode...'")
}

func TestPalette_FuzzySearch_NoMatch(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Search for something that doesn't exist.
	palette.Insert('x')
	palette.Insert('y')
	palette.Insert('z')
	palette.Insert('z')
	palette.Insert('z')

	filtered := palette.FilteredCommands()
	assert.Len(t, filtered, 0)
}

func TestPalette_FuzzySearch_Description(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Search for "grep" - should match description of "Search in repo...".
	palette.Insert('g')
	palette.Insert('r')
	palette.Insert('e')
	palette.Insert('p')

	filtered := palette.FilteredCommands()
	assert.Greater(t, len(filtered), 0)

	found := false

	for _, cmd := range filtered {
		if cmd.Name() == "Search in repo..." {
			found = true

			break
		}
	}

	assert.True(t, found, "Should find 'Search in repo...' by description 'Grep/search files'")
}

func TestPalette_SelectionResetOnQueryChange(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Move selection down.
	palette.MoveDown()
	palette.MoveDown()
	assert.Equal(t, 2, palette.Selection())

	// Insert character - selection should reset to 0.
	palette.Insert('s')
	assert.Equal(t, 0, palette.Selection())
}

func TestPalette_SelectionClampOnFilter(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Move to last item.
	for range 5 {
		palette.MoveDown()
	}

	assert.Equal(t, 5, palette.Selection())

	// Filter down to 1 result.
	palette.Insert('r')
	palette.Insert('u')
	palette.Insert('n')

	// Selection should be clamped to 0.
	assert.Equal(t, 0, palette.Selection())
}

func TestPalette_FilteredCommands_Order(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)
	palette.Open()

	// Empty query preserves order.
	filtered := palette.FilteredCommands()
	assert.Equal(t, "Run...", filtered[0].Name())
	assert.Equal(t, "Search in repo...", filtered[1].Name())
	assert.Equal(t, "Open recent file...", filtered[2].Name())
	assert.Equal(t, "New plan...", filtered[3].Name())
	assert.Equal(t, "Toggle mode...", filtered[4].Name())
	assert.Equal(t, "Change theme...", filtered[5].Name())
}

func TestPalette_OpenResetsState(t *testing.T) {
	t.Parallel()
	registry := createTestRegistry()
	palette := NewPalette(registry)

	// Set some state.
	palette.Open()
	palette.Insert('t')
	palette.Insert('e')
	palette.Insert('s')
	palette.Insert('t')
	palette.MoveDown()
	palette.MoveDown()

	// Close and reopen.
	palette.Close()
	palette.Open()

	// State should be reset.
	assert.Equal(t, "", palette.Query())
	assert.Equal(t, 0, palette.Selection())
	assert.Len(t, palette.FilteredCommands(), 6)
}

func TestPalette_ExecuteSelectedCommand(t *testing.T) {
	t.Parallel()
	executed := false
	registry := NewCommandRegistry()
	registry.Register(NewSimpleCommand("Test", "Test command", "Test", 'T', func(_ context.Context) error {
		executed = true

		return nil
	}))

	palette := NewPalette(registry)
	palette.Open()

	cmd := palette.SelectedCommand()
	assert.NotNil(t, cmd)

	err := cmd.Execute(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)
}
