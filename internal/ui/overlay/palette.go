package overlay

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

// Palette state machine for command search/selection.
type Palette struct {
	registry  *CommandRegistry
	query     []rune
	cursor    int
	filtered  []fuzzy.Match // filtered commands with scores.
	selection int           // index in filtered list.
	visible   bool
}

// NewPalette creates a new command palette.
func NewPalette(registry *CommandRegistry) *Palette {
	p := &Palette{
		registry:  registry,
		query:     make([]rune, 0, 64),
		cursor:    0,
		filtered:  make([]fuzzy.Match, 0),
		selection: 0,
		visible:   false,
	}
	p.updateFilter()

	return p
}

// Open opens the palette and resets state.
func (p *Palette) Open() {
	p.visible = true
	p.query = p.query[:0]
	p.cursor = 0
	p.selection = 0
	p.updateFilter()
}

// Close closes the palette.
func (p *Palette) Close() {
	p.visible = false
}

// IsOpen returns whether the palette is currently visible.
func (p *Palette) IsOpen() bool {
	return p.visible
}

// Insert adds a rune at the cursor position.
func (p *Palette) Insert(r rune) {
	p.query = append(p.query[:p.cursor], append([]rune{r}, p.query[p.cursor:]...)...)
	p.cursor++
	p.updateFilter()
	p.selection = 0 // Reset selection on query change.
}

// Backspace deletes the rune before the cursor.
func (p *Palette) Backspace() {
	if p.cursor > 0 {
		p.query = append(p.query[:p.cursor-1], p.query[p.cursor:]...)
		p.cursor--
		p.updateFilter()
		p.selection = 0
	}
}

// ClearQuery clears the entire query.
func (p *Palette) ClearQuery() {
	p.query = p.query[:0]
	p.cursor = 0
	p.updateFilter()
	p.selection = 0
}

// Query returns the current search query as a string.
func (p *Palette) Query() string {
	return string(p.query)
}

// MoveUp moves selection up one item (wraps to bottom).
func (p *Palette) MoveUp() {
	if len(p.filtered) == 0 {
		return
	}

	p.selection--
	if p.selection < 0 {
		p.selection = len(p.filtered) - 1
	}
}

// MoveDown moves selection down one item (wraps to top).
func (p *Palette) MoveDown() {
	if len(p.filtered) == 0 {
		return
	}

	p.selection++
	if p.selection >= len(p.filtered) {
		p.selection = 0
	}
}

// SelectedCommand returns the currently selected command, or nil if no selection.
func (p *Palette) SelectedCommand() Command {
	if len(p.filtered) == 0 || p.selection < 0 || p.selection >= len(p.filtered) {
		return nil
	}

	cmdIndex := p.filtered[p.selection].Index

	commands := p.registry.Commands()
	if cmdIndex >= 0 && cmdIndex < len(commands) {
		return commands[cmdIndex]
	}

	return nil
}

// FilteredCommands returns the filtered command list (for rendering).
func (p *Palette) FilteredCommands() []Command {
	commands := p.registry.Commands()

	result := make([]Command, 0, len(p.filtered))
	for _, match := range p.filtered {
		if match.Index >= 0 && match.Index < len(commands) {
			result = append(result, commands[match.Index])
		}
	}

	return result
}

// Selection returns the current selection index (0-based in filtered list).
func (p *Palette) Selection() int {
	return p.selection
}

// updateFilter re-runs fuzzy search and updates filtered list.
func (p *Palette) updateFilter() {
	commands := p.registry.Commands()
	if len(commands) == 0 {
		p.filtered = nil

		return
	}

	queryStr := string(p.query)
	if p.isEmptyQuery(queryStr) {
		p.setAllCommands(commands)

		return
	}

	p.performFuzzySearch(commands, queryStr)
	p.clampSelection()
}

// isEmptyQuery checks if the query is empty or whitespace only.
func (p *Palette) isEmptyQuery(queryStr string) bool {
	return strings.TrimSpace(queryStr) == ""
}

// setAllCommands sets all commands as filtered results.
func (p *Palette) setAllCommands(commands []Command) {
	p.filtered = make([]fuzzy.Match, len(commands))
	for i := range commands {
		p.filtered[i] = fuzzy.Match{Index: i, Score: 0}
	}
}

// performFuzzySearch performs fuzzy search on commands.
func (p *Palette) performFuzzySearch(commands []Command, queryStr string) {
	searchStrings := p.buildSearchStrings(commands)
	matches := fuzzy.Find(queryStr, searchStrings)
	p.filtered = matches
}

// buildSearchStrings builds searchable strings from commands.
func (p *Palette) buildSearchStrings(commands []Command) []string {
	searchStrings := make([]string, len(commands))
	for i, cmd := range commands {
		searchStrings[i] = cmd.Name() + " " + cmd.Description()
	}

	return searchStrings
}

// clampSelection ensures selection is within bounds.
func (p *Palette) clampSelection() {
	if p.selection >= len(p.filtered) {
		p.selection = 0
	}
}
