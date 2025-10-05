# FRD-UI-3.4: File Picker Widget (@-trigger)

**Feature:** Fuzzy File Search and Selection Widget
**Phase:** 3.4
**Priority:** High
**Status:** In Progress
**Created:** 2025-10-05

---

## 1. Overview

Implement a file picker widget for Spin's TUI that provides fuzzy file search and instant path insertion when triggered by the `@` character. The picker must be fast, intuitive, and integrate seamlessly with the input widget.

**Goals:**
- Fuzzy file search with real-time results
- Keyboard navigation (↑↓ arrows, Tab/Enter)
- Instant path insertion at cursor position
- Gitignore-aware file filtering
- <50ms search for 10k files
- Dismissible with Esc
- Memory efficient (<10MB for file index)

---

## 2. Technical Design

### 2.1 Architecture

```
┌─────────────────────────────────────────┐
│    File Picker (filepicker.go)          │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   List (Bubble Tea Component)     │ │
│  │                                   │ │
│  │  - Filtered results display       │ │
│  │  - Keyboard navigation            │ │
│  │  - Selection highlighting         │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │     Fuzzy Matcher                 │ │
│  │                                   │ │
│  │  - Score-based ranking            │ │
│  │  - Path-aware matching            │ │
│  │  - Case-insensitive search        │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │     File Scanner                  │ │
│  │                                   │ │
│  │  - Recursive directory walk       │ │
│  │  - Gitignore filtering            │ │
│  │  - Relative path caching          │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### 2.2 Package Structure

```
internal/tui/ui/
├── filepicker.go         # Main file picker component
├── filepicker_test.go    # File picker tests

internal/filesearch/
├── scanner.go            # File system scanner
├── scanner_test.go       # Scanner tests
├── matcher.go            # Fuzzy matching algorithm
├── matcher_test.go       # Matcher tests
├── gitignore.go          # Gitignore parsing (optional)
├── doc.go                # Package documentation
```

### 2.3 File Picker Component

```go
// FilePicker represents the file picker widget.
type FilePicker struct {
    list         list.Model
    files        []string      // All available files
    filtered     []string      // Filtered results
    selected     int           // Selected index
    query        string        // Search query
    baseDir      string        // Base directory
    width        int
    height       int
    active       bool          // Picker is visible
    onSelect     func(string)  // Callback when file selected
    onCancel     func()        // Callback when dismissed
}

// NewFilePicker creates a new file picker.
func NewFilePicker(baseDir string, width, height int) FilePicker

// SetActive shows or hides the picker.
func (fp *FilePicker) SetActive(active bool)

// SetQuery updates the search query and filters results.
func (fp *FilePicker) SetQuery(query string)

// SelectNext moves selection down.
func (fp *FilePicker) SelectNext()

// SelectPrev moves selection up.
func (fp *FilePicker) SelectPrev()

// GetSelected returns the currently selected file path.
func (fp FilePicker) GetSelected() string

// Update handles Bubble Tea messages.
func (fp FilePicker) Update(msg tea.Msg) (FilePicker, tea.Cmd)

// View renders the file picker.
func (fp FilePicker) View() string
```

### 2.4 File Scanner

```go
// Scanner scans directories for files.
type Scanner struct {
    baseDir    string
    ignoreGit  bool
    maxDepth   int
    gitignore  *Gitignore // Optional gitignore parser
}

// NewScanner creates a new file scanner.
func NewScanner(baseDir string, ignoreGit bool) *Scanner

// Scan returns all files in the directory.
func (s *Scanner) Scan() ([]string, error)

// ScanAsync scans asynchronously and sends results to channel.
func (s *Scanner) ScanAsync(ctx context.Context) <-chan string
```

### 2.5 Fuzzy Matcher

```go
// Match represents a fuzzy match result.
type Match struct {
    Path     string
    Score    int
    Indices  []int // Matched character indices
}

// Matcher performs fuzzy matching.
type Matcher struct {
    caseSensitive bool
}

// NewMatcher creates a new fuzzy matcher.
func NewMatcher(caseSensitive bool) *Matcher

// Match finds fuzzy matches for the query.
func (m *Matcher) Match(query string, paths []string) []Match

// Score calculates the match score for a path.
func (m *Matcher) Score(query, path string) (int, []int)
```

---

## 3. Implementation Details

### 3.1 Dependencies

**Required Packages:**
```go
// Bubble Tea ecosystem
"github.com/charmbracelet/bubbles/list"
"github.com/charmbracelet/lipgloss"

// Standard library
"path/filepath"
"strings"
"sort"
```

**Note:** `bubbles/list` is already available via existing dependencies

### 3.2 Fuzzy Matching Algorithm

Use a simple but effective scoring algorithm:

```go
// Score calculates fuzzy match score
// Higher score = better match
func (m *Matcher) Score(query, path string) (int, []int) {
    if query == "" {
        return 0, nil
    }

    query = strings.ToLower(query)
    path = strings.ToLower(path)

    score := 0
    indices := []int{}
    queryIdx := 0

    for pathIdx, ch := range path {
        if queryIdx >= len(query) {
            break
        }

        if rune(query[queryIdx]) == ch {
            // Match found
            indices = append(indices, pathIdx)
            queryIdx++

            // Bonus for consecutive matches
            if len(indices) > 1 && indices[len(indices)-1] == indices[len(indices)-2]+1 {
                score += 5
            }

            // Bonus for match after separator
            if pathIdx > 0 && (path[pathIdx-1] == '/' || path[pathIdx-1] == '_' || path[pathIdx-1] == '-') {
                score += 10
            }

            // Bonus for exact case match (if case-sensitive)
            if !m.caseSensitive || query[queryIdx-1] == byte(ch) {
                score += 1
            }
        }
    }

    // All query chars must match
    if queryIdx != len(query) {
        return -1, nil
    }

    // Bonus for shorter paths
    score += 100 - len(path)

    return score, indices
}
```

### 3.3 File Scanner Implementation

```go
package filesearch

import (
    "context"
    "os"
    "path/filepath"
)

// Scanner scans directories for files.
type Scanner struct {
    baseDir   string
    ignoreGit bool
    maxDepth  int
}

// NewScanner creates a new file scanner.
func NewScanner(baseDir string, ignoreGit bool) *Scanner {
    return &Scanner{
        baseDir:   baseDir,
        ignoreGit: ignoreGit,
        maxDepth:  10, // Reasonable default
    }
}

// Scan returns all files in the directory.
func (s *Scanner) Scan() ([]string, error) {
    var files []string

    err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return nil // Skip errors
        }

        // Skip directories
        if d.IsDir() {
            // Skip .git directories
            if s.ignoreGit && d.Name() == ".git" {
                return filepath.SkipDir
            }
            return nil
        }

        // Get relative path
        relPath, err := filepath.Rel(s.baseDir, path)
        if err != nil {
            return nil
        }

        files = append(files, relPath)
        return nil
    })

    return files, err
}
```

### 3.4 File Picker Implementation

```go
package ui

import (
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/dmytrogajewski/spin/internal/filesearch"
)

// FilePicker represents the file picker widget.
type FilePicker struct {
    list      list.Model
    files     []string
    filtered  []string
    matcher   *filesearch.Matcher
    query     string
    baseDir   string
    width     int
    height    int
    active    bool
    onSelect  func(string)
    onCancel  func()
}

// NewFilePicker creates a new file picker.
func NewFilePicker(baseDir string, width, height int) FilePicker {
    // Scan files
    scanner := filesearch.NewScanner(baseDir, true)
    files, _ := scanner.Scan()

    // Create list
    items := make([]list.Item, 0)
    delegate := list.NewDefaultDelegate()
    l := list.New(items, delegate, width, height)
    l.Title = "Select File"
    l.SetShowHelp(false)

    return FilePicker{
        list:     l,
        files:    files,
        filtered: files[:min(20, len(files))], // Show first 20
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

// filter filters files based on query.
func (fp *FilePicker) filter() {
    if fp.query == "" {
        fp.filtered = fp.files[:min(20, len(fp.files))]
        return
    }

    // Fuzzy match
    matches := fp.matcher.Match(fp.query, fp.files)

    // Sort by score (descending)
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
    return item.(fileItem).path
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

    return style.Render(fp.list.View())
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
```

### 3.5 Integration with Input Widget

```go
// In internal/tui/app.go

type Model struct {
    // ... existing fields ...
    filePicker ui.FilePicker // File picker (Phase 3.4)
}

func NewModel() Model {
    m := Model{
        // ... existing fields ...
        filePicker: ui.NewFilePicker(".", 60, 15),
    }

    // Set @ trigger callback to open file picker
    m.input.SetTriggerCallback(func() {
        m.state = StateFilePickerOpen
        m.filePicker.SetActive(true)
    })

    return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... existing code ...

    // File picker state handling
    if m.state == StateFilePickerOpen {
        switch msg := msg.(type) {
        case tea.KeyMsg:
            switch msg.String() {
            case "esc":
                // Close file picker
                m.filePicker.SetActive(false)
                m.state = StateIdle
                return m, nil

            case "enter", "tab":
                // Select file and insert path
                selected := m.filePicker.GetSelected()
                if selected != "" {
                    // Insert at cursor position
                    current := m.input.GetValue()
                    m.input.SetValue(current + selected)
                }
                m.filePicker.SetActive(false)
                m.state = StateIdle
                return m, nil
            }
        }

        // Update file picker query from input
        // Extract text after @
        value := m.input.GetValue()
        // ... extract query after @ ...
        m.filePicker.SetQuery(query)
    }

    // Update file picker
    var cmd tea.Cmd
    m.filePicker, cmd = m.filePicker.Update(msg)

    return m, cmd
}
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

```go
// Matcher tests
func TestMatcher_Score(t *testing.T) {
    m := NewMatcher(false)

    score, indices := m.Score("abc", "a/b/c.txt")
    assert.Greater(t, score, 0)
    assert.Len(t, indices, 3)

    score, _ = m.Score("xyz", "a/b/c.txt")
    assert.Equal(t, -1, score) // No match
}

func TestMatcher_Match(t *testing.T) {
    m := NewMatcher(false)
    paths := []string{
        "src/app.go",
        "src/main.go",
        "internal/app/app.go",
    }

    matches := m.Match("ap", paths)

    assert.Greater(t, len(matches), 0)
    // "app.go" should rank higher than "main.go"
}

// Scanner tests
func TestScanner_Scan(t *testing.T) {
    tmpDir := t.TempDir()

    // Create test files
    os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644)
    os.MkdirAll(filepath.Join(tmpDir, "dir"), 0755)
    os.WriteFile(filepath.Join(tmpDir, "dir/file2.txt"), []byte(""), 0644)

    scanner := NewScanner(tmpDir, false)
    files, err := scanner.Scan()

    assert.NoError(t, err)
    assert.Len(t, files, 2)
}

// FilePicker tests
func TestFilePicker_SetQuery(t *testing.T) {
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644)

    fp := NewFilePicker(tmpDir, 60, 15)
    fp.SetQuery("test")

    assert.Contains(t, fp.filtered, "test.go")
}
```

### 4.2 Integration Tests

```go
func TestFilePicker_Integration(t *testing.T) {
    m := NewModel()

    // Type @ to trigger
    m.input.SetValue("@")
    // Trigger callback
    assert.Equal(t, StateFilePickerOpen, m.state)
    assert.True(t, m.filePicker.active)

    // Type query
    m.filePicker.SetQuery("test")

    // Press Enter to select
    newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    m = newModel.(Model)

    assert.False(t, m.filePicker.active)
    assert.Equal(t, StateIdle, m.state)
}
```

### 4.3 Performance Tests

```go
func BenchmarkMatcher_Match_10k(b *testing.B) {
    m := NewMatcher(false)

    // Generate 10k file paths
    paths := make([]string, 10000)
    for i := 0; i < 10000; i++ {
        paths[i] = fmt.Sprintf("src/pkg%d/file%d.go", i/100, i)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.Match("pkg", paths)
    }
}

func TestFilePicker_SearchPerformance(t *testing.T) {
    // Create 10k files
    paths := make([]string, 10000)
    for i := 0; i < 10000; i++ {
        paths[i] = fmt.Sprintf("dir/file%d.txt", i)
    }

    m := NewMatcher(false)

    start := time.Now()
    m.Match("file", paths)
    elapsed := time.Since(start)

    assert.Less(t, elapsed, 50*time.Millisecond, "Search should be <50ms")
}
```

---

## 5. Performance Requirements

### 5.1 Search Performance

**Targets:**
- Initial scan: <500ms for 10k files
- Fuzzy match: <50ms for 10k files
- UI update: <16ms (60 FPS)
- Memory: <10MB for file index

**Optimizations:**
- Async file scanning
- Limit results to top 20 matches
- Efficient scoring algorithm
- Path caching

### 5.2 Memory Management

```go
// Limit file index size
const MaxFiles = 50000

func (s *Scanner) Scan() ([]string, error) {
    files := make([]string, 0, 1000)

    err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
        if len(files) >= MaxFiles {
            return filepath.SkipAll
        }
        // ... rest of scan ...
    })

    return files, err
}
```

---

## 6. Quality Checklist

### 6.1 Definition of Ready (DoR)

- [x] File search requirements reviewed
- [x] Fuzzy search algorithm selected (score-based)
- [x] UI/UX flow designed (overlay modal)
- [x] Dependencies identified (bubbles/list)

### 6.2 Definition of Done (DoD)

- [ ] Tests for fuzzy matching (≥90% coverage)
- [ ] Tests for file scanning (≥90% coverage)
- [ ] Tests for file picker UI (≥85% coverage)
- [ ] Search <50ms for 10k files
- [ ] UI updates in real-time
- [ ] File picker dismissible with Esc
- [ ] Tab/Enter selects file
- [ ] Path inserted at cursor
- [ ] Gitignore filtering works
- [ ] All tests passing with race detector
- [ ] Linter clean (make lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] ROADMAP updated

---

## 7. Risks and Mitigations

### 7.1 Risks

1. **Large directory performance** - 100k+ files could be slow
   - **Mitigation:** Async scanning, result limits, depth limits, async file loading

2. **Gitignore parsing complexity** - Full .gitignore support is complex
   - **Mitigation:** Start simple (skip .git only), add gitignore later if needed

3. **Fuzzy match quality** - Simple algorithm may not rank well
   - **Mitigation:** Iterate on scoring, add tests for common cases

4. **Memory usage** - Large file lists could consume memory
   - **Mitigation:** Limit to 50k files, use lazy loading

---

## 8. Success Criteria

Phase 3.4 is complete when:

1. ✅ @ character triggers file picker overlay
2. ✅ Fuzzy search filters files in real-time
3. ✅ Keyboard navigation works (↑↓, Tab/Enter)
4. ✅ Selected file path inserted at cursor
5. ✅ Esc dismisses picker
6. ✅ Search <50ms for 10k files
7. ✅ Gitignore filtering (at least .git)
8. ✅ All tests passing with ≥90% coverage
9. ✅ Memory usage <10MB
10. ✅ Integration with TUI complete

---

## 9. Future Enhancements (Out of Scope)

- Full .gitignore parsing
- Multi-file selection
- File preview pane
- Recent files list
- Workspace-aware search
- Symbol search (@symbol syntax)
- Command palette integration

---

## 10. References

- [Bubble Tea List](https://github.com/charmbracelet/bubbles/tree/master/list)
- [FZF Algorithm](https://github.com/junegunn/fzf/blob/master/ALGORITHM.md)
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md)
- [FRD-UI-3.3.md](FRD-UI-3.3.md) - Input Widget
- [AGENTS.md](../../AGENTS.md) - Quality standards

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** Spin Development Team
