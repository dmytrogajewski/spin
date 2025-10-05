package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dmytrogajewski/spin/internal/filesearch"
)

// FilePicker represents the file picker widget.
type FilePicker struct {
	list     list.Model
	files    []string
	filtered []string
	matcher  *filesearch.Matcher
	query    string
	baseDir  string
	width    int
	height   int
	active   bool
	onSelect func(string)
	onCancel func()
}

// NewFilePicker creates a new file picker.
func NewFilePicker(baseDir string, width, height int) FilePicker {
	// Scan files
	scanner := filesearch.NewScanner(baseDir, true)
	files, _ := scanner.Scan()

	// Create list
	items := make([]list.Item, 0)
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, width, height-4) // -4 for borders and title
	l.Title = "Select File"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We handle filtering ourselves

	// Initial filtered list
	filtered := files
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}

	return FilePicker{
		list:     l,
		files:    files,
		filtered: filtered,
		matcher:  filesearch.NewMatcher(false),
		query:    "",
		baseDir:  baseDir,
		width:    width,
		height:   height,
		active:   false,
	}
}

// SetActive shows or hides the picker.
func (fp *FilePicker) SetActive(active bool) {
	fp.active = active
	if active {
		fp.updateList()
	}
}

// SetQuery updates the search query and filters results.
func (fp *FilePicker) SetQuery(query string) {
	fp.query = query
	fp.filter()
	fp.updateList()
}

// SetSize updates the picker dimensions.
func (fp *FilePicker) SetSize(width, height int) {
	fp.width = width
	fp.height = height
	fp.list.SetSize(width, height-4)
}

// SetOnSelect sets the callback when a file is selected.
func (fp *FilePicker) SetOnSelect(callback func(string)) {
	fp.onSelect = callback
}

// SetOnCancel sets the callback when the picker is cancelled.
func (fp *FilePicker) SetOnCancel(callback func()) {
	fp.onCancel = callback
}

// filter filters files based on query.
func (fp *FilePicker) filter() {
	if fp.query == "" {
		// Show first 20 files
		fp.filtered = fp.files
		if len(fp.filtered) > 20 {
			fp.filtered = fp.filtered[:20]
		}
		return
	}

	// Fuzzy match
	matches := fp.matcher.Match(fp.query, fp.files)

	// Take top 20
	fp.filtered = make([]string, 0, min(20, len(matches)))
	for i := 0; i < min(20, len(matches)); i++ {
		fp.filtered = append(fp.filtered, matches[i].Path)
	}
}

// updateList updates the list items.
func (fp *FilePicker) updateList() {
	items := make([]list.Item, len(fp.filtered))
	for i, path := range fp.filtered {
		items[i] = fileItem{path: path}
	}
	fp.list.SetItems(items)
}

// GetSelected returns the currently selected file path.
func (fp FilePicker) GetSelected() string {
	if len(fp.filtered) == 0 {
		return ""
	}
	item := fp.list.SelectedItem()
	if item == nil {
		return ""
	}
	fileItem, ok := item.(fileItem)
	if !ok {
		return ""
	}
	return fileItem.path
}

// Update handles Bubble Tea messages.
func (fp FilePicker) Update(msg tea.Msg) (FilePicker, tea.Cmd) {
	if !fp.active {
		return fp, nil
	}

	var cmd tea.Cmd
	fp.list, cmd = fp.list.Update(msg)
	return fp, cmd
}

// View renders the file picker.
func (fp FilePicker) View() string {
	if !fp.active {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")). // Magenta
		Padding(1)

	// Add query indicator
	queryHint := ""
	if fp.query != "" {
		queryHint = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Query: " + fp.query + " | ")
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(queryHint + "Tab/Enter to select, Esc to cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		fp.list.View(),
		hint,
	)

	return style.Render(content)
}

// fileItem implements list.Item
type fileItem struct {
	path string
}

func (f fileItem) FilterValue() string { return f.path }
func (f fileItem) Title() string       { return f.path }
func (f fileItem) Description() string { return "" }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
