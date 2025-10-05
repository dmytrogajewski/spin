# FRD-UI-3.7: Transcript View & History

**Feature:** Enhanced transcript navigation, search, and export
**Priority:** P1 (Phase 3.7)
**Status:** Draft
**Created:** 2025-10-05

---

## 1. Overview

This FRD defines the transcript view and history management features for the Spin TUI. It extends the existing Chat component (from FRD-UI-3.2) with advanced navigation, search, and export capabilities.

**Scope:**
- Scroll navigation (PgUp/PgDn/Home/End)
- Search within transcript (/)
- Export conversation (Ctrl+E)
- Scroll indicators and position tracking

**Out of Scope:**
- Persistent storage (handled by session module)
- Transcript sharing (future feature)
- Real-time transcript sync (future feature)

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR-3.7.1: Scroll Navigation
- **MUST** support PgUp/PgDn for page-based scrolling
- **MUST** support Home/End to jump to start/end
- **MUST** support mouse wheel scrolling (if terminal supports it)
- **MUST** show scroll position indicator
- **MUST** auto-scroll to bottom when new content arrives (unless user has scrolled up)

#### FR-3.7.2: Search
- **MUST** support search activation via `/` key
- **MUST** perform case-insensitive search by default
- **MUST** highlight matching text in transcript
- **MUST** support n/N for next/previous match navigation
- **MUST** show match count and current position (e.g., "3/12")
- **MUST** support Esc to exit search mode
- **SHOULD** support basic regex patterns (future enhancement)

#### FR-3.7.3: Export
- **MUST** support export via Ctrl+E
- **MUST** export to markdown format
- **MUST** include timestamps
- **MUST** preserve code blocks and formatting
- **MUST** prompt for filename (default: `spin-transcript-YYYYMMDD-HHMMSS.md`)
- **MUST** show success/error notification after export

#### FR-3.7.4: Performance
- **MUST** render scroll updates <16ms (60 FPS)
- **MUST** search complete <100ms for 1000 messages
- **MUST** export complete <500ms for 1000 messages
- **SHOULD** use lazy rendering for large transcripts

### 2.2 Non-Functional Requirements

#### NFR-3.7.1: Usability
- Keyboard shortcuts must be intuitive and documented
- Search UI must be clear and non-intrusive
- Export filename must be valid across platforms

#### NFR-3.7.2: Quality
- Test coverage ≥85%
- All tests pass with `-race` flag
- Complexity ≤15 for all functions
- Linter clean (golangci-lint)

---

## 3. Architecture

### 3.1 Component Structure

```
internal/tui/ui/
├── chat.go              # Extended with scroll/search/export (existing)
├── transcript.go        # NEW: Transcript management logic
├── search.go            # NEW: Search functionality
└── export.go            # NEW: Export functionality
```

### 3.2 Data Model

#### SearchState
```go
// SearchState represents the state of transcript search.
type SearchState struct {
    Active    bool     // Search mode active
    Query     string   // Current search query
    Matches   []int    // Indices of matching messages
    Current   int      // Current match index (0-based)
    Input     textinput.Model // Search input widget
}
```

#### ExportConfig
```go
// ExportConfig defines export settings.
type ExportConfig struct {
    Format       ExportFormat // markdown, json, txt
    IncludeTime  bool        // Include timestamps
    IncludeTools bool        // Include tool calls/results
    Filename     string      // Output filename
}

type ExportFormat string
const (
    ExportMarkdown ExportFormat = "markdown"
    ExportJSON     ExportFormat = "json"
    ExportText     ExportFormat = "text"
)
```

### 3.3 Extended Chat Component

```go
// Chat component extended with search and export
type Chat struct {
    viewport  viewport.Model
    messages  []Message
    width     int
    height    int
    formatter *Formatter

    // Existing fields
    content      string
    contentDirty bool
    atBottom     bool

    // NEW: Search state
    search       SearchState

    // NEW: Scroll tracking
    scrollPercent float64  // 0.0-100.0
}

// NEW: Search methods
func (c *Chat) StartSearch() tea.Cmd
func (c *Chat) UpdateSearch(query string)
func (c *Chat) NextMatch() tea.Cmd
func (c *Chat) PrevMatch() tea.Cmd
func (c *Chat) ExitSearch()

// NEW: Export methods
func (c *Chat) ExportTranscript(cfg ExportConfig) error
func (c *Chat) ExportToMarkdown(filename string) error
```

---

## 4. User Experience

### 4.1 Scroll Navigation

**Default behavior:**
- New messages auto-scroll to bottom
- User scrolling disables auto-scroll
- Any user input re-enables auto-scroll

**Keyboard shortcuts:**
```
PgUp        - Scroll up one page
PgDn        - Scroll down one page
Home        - Jump to top of transcript
End         - Jump to bottom of transcript
↑/↓         - Scroll by line (if not in input)
```

**Visual indicator (bottom-right of viewport):**
```
╭─────────────────────────────────────╮
│ [Transcript content...]             │
│                                     │
│                                90%  │  ← Scroll position
╰─────────────────────────────────────╯
```

### 4.2 Search Experience

**Activation:**
```
User presses: /
[Search bar appears at bottom]

╭─────────────────────────────────────╮
│ You │ 14:30:05                      │
│ Find all references to auth         │
│                                     │
│ Assistant │ 14:30:08                │
│ I found 3 references:               │
│ 1. internal/auth/manager.go        │ ← Highlighted match
│ 2. internal/auth/keystore.go       │
│ 3. cmd/spin/root.go                │
╰─────────────────────────────────────╯
Search: auth                    [2/3]  ← Search UI
```

**Navigation:**
```
n       - Next match
N       - Previous match
Esc     - Exit search
```

### 4.3 Export Experience

**Workflow:**
```
User presses: Ctrl+E
[Export dialog appears]

╭─────────────────────────────────────╮
│        Export Transcript            │
├─────────────────────────────────────┤
│                                     │
│ Filename:                           │
│ > spin-transcript-20251005-143015.md│
│                                     │
│ Format: [✓] Markdown  [ ] JSON      │
│ Options: [✓] Timestamps             │
│          [✓] Tool calls             │
│                                     │
│ [Enter] Export  [Esc] Cancel        │
╰─────────────────────────────────────╯
```

**Output format (Markdown):**
```markdown
# Spin Conversation - 2025-10-05 14:30

## You │ 14:30:05
Find all references to auth

## Assistant │ 14:30:08
I found 3 references:

1. `internal/auth/manager.go` - Authentication manager
2. `internal/auth/keystore.go` - Keystore implementation
3. `cmd/spin/root.go` - Root command with auth setup

### Tool Call: search
```json
{
  "pattern": "auth",
  "type": "go"
}
```

### Tool Result
✓ Found 3 matches
```
```

---

## 5. Implementation Plan

### 5.1 Phase 1: Scroll Navigation (Day 1)

**Tasks:**
1. Add PgUp/PgDn handling to Chat.Update()
2. Add Home/End handling
3. Implement scroll position tracking (scrollPercent)
4. Add scroll position indicator to Chat.View()
5. Update auto-scroll logic (respect user scrolling)
6. Write tests for scroll navigation

**Files:**
- `internal/tui/ui/chat.go` - Extend Update() and View()
- `internal/tui/ui/chat_test.go` - Add scroll tests

### 5.2 Phase 2: Search Functionality (Day 2)

**Tasks:**
1. Create `internal/tui/ui/search.go`
2. Implement SearchState struct
3. Add search input widget (textinput)
4. Implement search matching algorithm (case-insensitive)
5. Add match highlighting (lipgloss)
6. Implement n/N navigation
7. Add search UI rendering
8. Write tests for search

**Files:**
- `internal/tui/ui/search.go` - NEW
- `internal/tui/ui/search_test.go` - NEW
- `internal/tui/ui/chat.go` - Integrate search

### 5.3 Phase 3: Export Functionality (Day 3)

**Tasks:**
1. Create `internal/tui/ui/export.go`
2. Implement ExportConfig struct
3. Implement markdown exporter
4. Add filename input dialog
5. Implement file write with error handling
6. Add export success/error notification
7. Write tests for export

**Files:**
- `internal/tui/ui/export.go` - NEW
- `internal/tui/ui/export_test.go` - NEW
- `internal/tui/ui/chat.go` - Integrate export

### 5.4 Phase 4: Integration & Polish (Day 4)

**Tasks:**
1. Integrate all features into main TUI app
2. Update `internal/tui/app.go` with new key handlers
3. Add help text for new shortcuts
4. Performance testing and optimization
5. Documentation updates

---

## 6. Testing Strategy

### 6.1 Unit Tests

**Scroll Navigation:**
- Test PgUp scrolls up by viewport height
- Test PgDn scrolls down by viewport height
- Test Home goes to top
- Test End goes to bottom
- Test auto-scroll enable/disable logic
- Test scroll position calculation

**Search:**
- Test search activation (/)
- Test search matching (case-insensitive)
- Test n/N navigation
- Test Esc exits search
- Test match highlighting
- Test match count display
- Test empty query handling
- Test no matches found

**Export:**
- Test markdown generation
- Test filename generation (default)
- Test custom filename
- Test file write success
- Test file write error handling
- Test export cancellation

### 6.2 Integration Tests

- Test scroll + search interaction
- Test export with search active
- Test keyboard shortcut conflicts
- Test state transitions (idle → search → idle)

### 6.3 Performance Tests

- Benchmark scroll rendering (≥60 FPS)
- Benchmark search on 1000 messages (<100ms)
- Benchmark export on 1000 messages (<500ms)

---

## 7. Acceptance Criteria

### Definition of Done

- [ ] All functional requirements implemented
- [ ] PgUp/PgDn/Home/End navigation works
- [ ] Search with `/` works (n/N navigation)
- [ ] Export with Ctrl+E works (markdown format)
- [ ] All tests passing (≥85% coverage)
- [ ] Race detector clean
- [ ] Linter clean (golangci-lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] Performance targets met:
  - Scroll render <16ms
  - Search <100ms (1000 messages)
  - Export <500ms (1000 messages)
- [ ] Integration with TUI app complete
- [ ] Documentation updated

### Verification

```bash
# Build
make build

# Run tests
go test -race ./internal/tui/ui/...
go test -cover ./internal/tui/ui/...

# Lint
make lint

# Complexity check
gocyclo -over 15 ./internal/tui/ui/

# Manual testing
./bin/spin tui
# Test: PgUp/PgDn, Home/End
# Test: / to search, n/N to navigate
# Test: Ctrl+E to export
```

---

## 8. Dependencies

### External Libraries
- `github.com/charmbracelet/bubbles/viewport` - Already used
- `github.com/charmbracelet/bubbles/textinput` - Already used
- `github.com/charmbracelet/lipgloss` - Already used

### Internal Modules
- `internal/tui/ui/chat.go` - Extend existing component
- `internal/tui/ui/message.go` - Existing message types

---

## 9. Risks & Mitigations

### Risk 1: Performance Degradation with Large Transcripts
**Likelihood:** Medium
**Impact:** High
**Mitigation:**
- Implement lazy rendering
- Cache rendered content
- Limit visible messages to viewport height + buffer
- Test with 1000+ messages

### Risk 2: Search Complexity
**Likelihood:** Low
**Impact:** Medium
**Mitigation:**
- Start with simple case-insensitive substring match
- Defer regex support to future enhancement
- Optimize with pre-compiled search index if needed

### Risk 3: Export File Permissions
**Likelihood:** Medium
**Impact:** Low
**Mitigation:**
- Check write permissions before export
- Show clear error message on failure
- Default to ~/Downloads or ~/Documents
- Allow custom directory selection

---

## 10. Future Enhancements

### Phase 2 (Future)
- Advanced search (regex, case-sensitive toggle)
- Search history (previous queries)
- Incremental search (search-as-you-type)
- JSON export format
- Custom export templates
- Export to clipboard
- Syntax highlighting in exported markdown
- Persistent search settings

---

## 11. References

- [AGENTS.md](../../AGENTS.md) - Implementation workflow
- [FRD-UI-3.1](FRD-UI-3.1.md) - TUI application setup
- [FRD-UI-3.2](FRD-UI-3.2.md) - Chat interface components
- [specs/ui-modules/spec.md](../ui-modules/spec.md) - UI modules specification
- [Bubble Tea viewport](https://github.com/charmbracelet/bubbles/tree/master/viewport) - Viewport component docs

---

## Appendix A: Message Format

### Markdown Export Format

```markdown
# Spin Conversation

**Started:** 2025-10-05 14:30:15
**Model:** llama3.1
**Messages:** 42

---

## You │ 14:30:05

Find all references to auth in the codebase.

---

## Assistant │ 14:30:08

I'll search for "auth" references across the project.

### 🔧 Tool Call: search
```json
{
  "pattern": "auth",
  "type": "go",
  "output_mode": "files_with_matches"
}
```

### ✓ Tool Result
```
internal/auth/manager.go
internal/auth/keystore.go
cmd/spin/root.go
```

I found 3 files with "auth" references. Let me read each one...

---
```

---

## Appendix B: Keyboard Shortcuts Summary

| Key | Action | State |
|-----|--------|-------|
| PgUp | Scroll up one page | Any |
| PgDn | Scroll down one page | Any |
| Home | Jump to top | Any |
| End | Jump to bottom | Any |
| `/` | Start search | Idle |
| `n` | Next match | Search |
| `N` | Previous match | Search |
| Esc | Exit search | Search |
| Ctrl+E | Export transcript | Idle |

---

**End of FRD-UI-3.7**
