# FRD-UI-3.6: Status Bar Component

**Feature:** Status Bar for Interactive TUI
**Phase:** 3.6
**Priority:** High
**Status:** In Progress
**Created:** 2025-10-05

---

## 1. Overview

Implement a status bar component for Spin's interactive TUI mode that displays critical runtime information. The status bar provides users with at-a-glance awareness of the system's state, current model, sandbox mode, working directory, and resource usage.

**Goals:**
- Display current model and provider
- Show sandbox mode with visual indicators
- Display working directory
- Show connection/activity status
- Display token usage (current turn / session total)
- Responsive layout that adapts to terminal width
- Color-coded status indicators
- Performance: <1ms render time

---

## 2. Technical Design

### 2.1 Architecture

**Status Bar Position:**
```
┌────────────────────────────────────────────────┐
│                                                │
│           Chat Transcript Area                 │
│                                                │
├────────────────────────────────────────────────┤
│                                                │
│              Input Widget                      │
│                                                │
├────────────────────────────────────────────────┤
│ llama3.1 | 🔒 read-only | ~/project | 1.2K/5K │ ← Status Bar
└────────────────────────────────────────────────┘
```

### 2.2 Package Structure

```
internal/tui/ui/
├── statusbar.go       # Status bar component
└── statusbar_test.go  # Tests (≥85% coverage)
```

### 2.3 Information Display

**Left Section:**
- Model name (e.g., "llama3.1", "gpt-4o", "mixtral")
- Provider indicator (optional, inferred from model)

**Middle Section:**
- Sandbox mode icon + text:
  - 🔒 read-only
  - 📝 workspace-write
  - 🔓 unrestricted
- Working directory (relative or abbreviated)

**Right Section:**
- Connection/activity status:
  - ⚡ active (streaming response)
  - ⏸ idle
  - ⚠ error
  - 🔄 connecting
- Token usage: `<turn>/<session>` (e.g., "1.2K/5.4K")

**Example Layouts:**

Wide terminal (120 cols):
```
llama3.1 | 🔒 read-only | ~/dev/spin | ⚡ active | 1.2K / 5.4K tokens
```

Medium terminal (80 cols):
```
llama3.1 | 🔒 read-only | ~/spin | ⚡ | 1.2K/5.4K
```

Narrow terminal (60 cols):
```
llama3.1 | 🔒 | ~/spin | 1.2K/5K
```

### 2.4 Data Model

```go
// StatusInfo contains all information displayed in the status bar.
type StatusInfo struct {
	// Model information
	Model    string // Model name (e.g., "llama3.1")
	Provider string // Provider name (e.g., "ollama", "openai")

	// Sandbox information
	SandboxMode string // "read-only", "workspace-write", "unrestricted"

	// Location
	WorkingDir string // Current working directory

	// Connection status
	Status ConnectionStatus // Current activity status

	// Token usage
	TurnTokens    int // Tokens used in current turn
	SessionTokens int // Total tokens used in session
}

// ConnectionStatus represents the current activity state.
type ConnectionStatus int

const (
	StatusIdle ConnectionStatus = iota
	StatusActive
	StatusConnecting
	StatusError
)

// String returns the string representation of the status.
func (s ConnectionStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusActive:
		return "active"
	case StatusConnecting:
		return "connecting"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Icon returns the icon for the status.
func (s ConnectionStatus) Icon() string {
	switch s {
	case StatusIdle:
		return "⏸"
	case StatusActive:
		return "⚡"
	case StatusConnecting:
		return "🔄"
	case StatusError:
		return "⚠"
	default:
		return "?"
	}
}
```

### 2.5 Status Bar Component

```go
// StatusBar represents the status bar component.
type StatusBar struct {
	info  StatusInfo
	width int
	style StatusBarStyle
}

// StatusBarStyle defines the visual styling for the status bar.
type StatusBarStyle struct {
	Normal lipgloss.Style
	Active lipgloss.Style
	Error  lipgloss.Style
}

// NewStatusBar creates a new status bar.
func NewStatusBar(width int) StatusBar {
	return StatusBar{
		info:  StatusInfo{Status: StatusIdle},
		width: width,
		style: DefaultStatusBarStyle(),
	}
}

// SetInfo updates the status information.
func (s *StatusBar) SetInfo(info StatusInfo) {
	s.info = info
}

// SetWidth updates the status bar width.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// View renders the status bar.
func (s StatusBar) View() string {
	// Render based on width and information
	return s.render()
}

// render creates the status bar content.
func (s StatusBar) render() string {
	// Format each section
	left := s.renderLeft()
	middle := s.renderMiddle()
	right := s.renderRight()

	// Layout based on available width
	return s.layout(left, middle, right)
}
```

---

## 3. Implementation Details

### 3.1 Rendering Logic

**Adaptive Layout:**

```go
func (s StatusBar) layout(left, middle, right string) string {
	// Calculate required width
	separator := " | "
	requiredWidth := lipgloss.Width(left) + lipgloss.Width(middle) + lipgloss.Width(right) + len(separator)*2

	if s.width < 40 {
		// Very narrow: only essential info
		return s.renderCompact()
	} else if requiredWidth > s.width {
		// Medium: abbreviate middle section
		middle = s.renderMiddleAbbreviated()
	}

	// Join with separators
	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		left,
		separator,
		middle,
		separator,
		right,
	)

	// Pad to full width
	return s.style.Normal.Width(s.width).Render(content)
}
```

**Section Renderers:**

```go
func (s StatusBar) renderLeft() string {
	// Model name (potentially truncated)
	model := s.info.Model
	if model == "" {
		model = "no-model"
	}
	return model
}

func (s StatusBar) renderMiddle() string {
	// Sandbox icon + mode + working dir
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)

	// Abbreviate working directory
	workDir := s.abbreviateDir(s.info.WorkingDir)

	return fmt.Sprintf("%s %s | %s", sandboxIcon, s.info.SandboxMode, workDir)
}

func (s StatusBar) renderMiddleAbbreviated() string {
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)
	workDir := s.abbreviateDir(s.info.WorkingDir)

	// Just icon + dir
	return fmt.Sprintf("%s | %s", sandboxIcon, workDir)
}

func (s StatusBar) renderRight() string {
	// Status icon + token usage
	statusIcon := s.info.Status.Icon()

	// Format token counts
	turnTokens := s.formatTokens(s.info.TurnTokens)
	sessionTokens := s.formatTokens(s.info.SessionTokens)

	if s.width > 100 {
		return fmt.Sprintf("%s %s | %s / %s tokens", statusIcon, s.info.Status.String(), turnTokens, sessionTokens)
	}
	return fmt.Sprintf("%s | %s/%s", statusIcon, turnTokens, sessionTokens)
}

func (s StatusBar) renderCompact() string {
	// Minimal info for very narrow terminals
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)
	statusIcon := s.info.Status.Icon()
	turnTokens := s.formatTokens(s.info.TurnTokens)

	return s.style.Normal.Width(s.width).Render(
		fmt.Sprintf("%s | %s | %s | %s", s.info.Model, sandboxIcon, statusIcon, turnTokens),
	)
}
```

### 3.2 Helper Functions

```go
// getSandboxIcon returns the icon for the sandbox mode.
func (s StatusBar) getSandboxIcon(mode string) string {
	switch mode {
	case "read-only":
		return "🔒"
	case "workspace-write":
		return "📝"
	case "unrestricted":
		return "🔓"
	default:
		return "?"
	}
}

// abbreviateDir abbreviates the working directory path.
func (s StatusBar) abbreviateDir(dir string) string {
	if dir == "" {
		return "~"
	}

	// Replace home directory with ~
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(dir, homeDir) {
		dir = "~" + strings.TrimPrefix(dir, homeDir)
	}

	// Truncate if too long
	maxLen := 30
	if len(dir) > maxLen {
		// Show start and end
		return dir[:10] + "..." + dir[len(dir)-15:]
	}

	return dir
}

// formatTokens formats token counts with K/M suffixes.
func (s StatusBar) formatTokens(count int) string {
	if count == 0 {
		return "0"
	}
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 1000000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000.0)
	}
	return fmt.Sprintf("%.1fM", float64(count)/1000000.0)
}
```

### 3.3 Styling

```go
// DefaultStatusBarStyle creates the default status bar style.
func DefaultStatusBarStyle() StatusBarStyle {
	base := lipgloss.NewStyle().
		Background(lipgloss.Color("236")). // Dark gray background
		Foreground(lipgloss.Color("250")). // Light gray foreground
		Padding(0, 1)

	return StatusBarStyle{
		Normal: base,
		Active: base.Foreground(lipgloss.Color("10")), // Green for active
		Error:  base.Foreground(lipgloss.Color("9")),  // Red for error
	}
}

// ApplyStatus applies color based on connection status.
func (s StatusBar) ApplyStatus() lipgloss.Style {
	switch s.info.Status {
	case StatusActive:
		return s.style.Active
	case StatusError:
		return s.style.Error
	default:
		return s.style.Normal
	}
}
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

```go
func TestStatusBar_Rendering(t *testing.T) {
	tests := []struct {
		name  string
		info  StatusInfo
		width int
		want  string // Expected substring
	}{
		{
			name: "full width with all info",
			info: StatusInfo{
				Model:         "llama3.1",
				SandboxMode:   "read-only",
				WorkingDir:    "/home/user/project",
				Status:        StatusActive,
				TurnTokens:    1200,
				SessionTokens: 5400,
			},
			width: 120,
			want:  "llama3.1",
		},
		{
			name: "medium width abbreviated",
			info: StatusInfo{
				Model:       "gpt-4o",
				SandboxMode: "workspace-write",
				WorkingDir:  "/project",
				Status:      StatusIdle,
			},
			width: 80,
			want:  "gpt-4o",
		},
		{
			name: "narrow width compact",
			info: StatusInfo{
				Model:       "mixtral",
				SandboxMode: "read-only",
				Status:      StatusError,
			},
			width: 40,
			want:  "mixtral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStatusBar(tt.width)
			sb.SetInfo(tt.info)
			got := sb.View()
			assert.Contains(t, got, tt.want)
		})
	}
}

func TestStatusBar_TokenFormatting(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1.2K"},
		{5400, "5.4K"},
		{1234567, "1.2M"},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.count), func(t *testing.T) {
			got := sb.formatTokens(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusBar_SandboxIcons(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"read-only", "🔒"},
		{"workspace-write", "📝"},
		{"unrestricted", "🔓"},
		{"unknown", "?"},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := sb.getSandboxIcon(tt.mode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusBar_DirectoryAbbreviation(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"empty", "", "~"},
		{"short", "/project", "/project"},
		{"home directory", "/home/user/project", "~/project"}, // if /home/user is $HOME
		{"very long", "/very/long/path/that/needs/to/be/truncated/project", "..."},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sb.abbreviateDir(tt.dir)
			if tt.want == "..." {
				assert.Contains(t, got, "...")
			} else {
				assert.Contains(t, got, tt.want)
			}
		})
	}
}

func TestConnectionStatus_String(t *testing.T) {
	tests := []struct {
		status ConnectionStatus
		want   string
	}{
		{StatusIdle, "idle"},
		{StatusActive, "active"},
		{StatusConnecting, "connecting"},
		{StatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConnectionStatus_Icon(t *testing.T) {
	tests := []struct {
		status ConnectionStatus
		want   string
	}{
		{StatusIdle, "⏸"},
		{StatusActive, "⚡"},
		{StatusConnecting, "🔄"},
		{StatusError, "⚠"},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			got := tt.status.Icon()
			assert.Equal(t, tt.want, got)
		})
	}
}
```

### 4.2 Integration Tests

```go
func TestStatusBar_UpdatesInRealTime(t *testing.T) {
	sb := NewStatusBar(120)

	// Initial state
	info1 := StatusInfo{
		Model:       "llama3.1",
		Status:      StatusIdle,
		TurnTokens:  0,
	}
	sb.SetInfo(info1)
	view1 := sb.View()
	assert.Contains(t, view1, "llama3.1")
	assert.Contains(t, view1, "⏸") // Idle icon

	// Update to active
	info2 := info1
	info2.Status = StatusActive
	info2.TurnTokens = 1200
	sb.SetInfo(info2)
	view2 := sb.View()
	assert.Contains(t, view2, "⚡") // Active icon
	assert.Contains(t, view2, "1.2K")
}

func TestStatusBar_ResponsiveLayout(t *testing.T) {
	info := StatusInfo{
		Model:         "llama3.1",
		SandboxMode:   "read-only",
		WorkingDir:    "/home/user/project",
		Status:        StatusActive,
		TurnTokens:    1200,
		SessionTokens: 5400,
	}

	// Test different widths
	widths := []int{40, 60, 80, 100, 120}
	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			sb := NewStatusBar(width)
			sb.SetInfo(info)
			view := sb.View()

			// Should always contain model name
			assert.Contains(t, view, "llama3.1")

			// Width should match
			assert.LessOrEqual(t, lipgloss.Width(view), width)
		})
	}
}
```

### 4.3 Benchmark Tests

```go
func BenchmarkStatusBar_Render(b *testing.B) {
	sb := NewStatusBar(120)
	info := StatusInfo{
		Model:         "llama3.1",
		SandboxMode:   "read-only",
		WorkingDir:    "/home/user/project",
		Status:        StatusActive,
		TurnTokens:    1200,
		SessionTokens: 5400,
	}
	sb.SetInfo(info)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sb.View()
	}
}
```

---

## 5. Integration with TUI

### 5.1 Model Updates

```go
// internal/tui/app.go

type Model struct {
	// ... existing fields ...
	statusBar ui.StatusBar // Add status bar (Phase 3.6)
}

func NewModel() Model {
	return Model{
		// ... existing initialization ...
		statusBar: ui.NewStatusBar(0), // Will be sized on first resize
	}
}

func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Update status bar width
	m.statusBar.SetWidth(m.width)

	// ... rest of resize logic ...
	return m, nil
}
```

### 5.2 View Rendering

```go
// internal/tui/view.go

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.chat.View(),      // Chat transcript
		m.input.View(),     // Input widget
		m.statusBar.View(), // Status bar (NEW)
	)
}
```

### 5.3 Status Updates

```go
// Update status bar when state changes
func (m Model) updateStatusBar() {
	info := ui.StatusInfo{
		Model:       m.config.Model,     // From config
		SandboxMode: m.config.Sandbox,   // From config
		WorkingDir:  m.workingDir,       // Current working directory
		Status:      m.connectionStatus(), // Based on app state
		TurnTokens:  m.turnTokens,       // From core events
		SessionTokens: m.sessionTokens,  // Accumulated
	}
	m.statusBar.SetInfo(info)
}

func (m Model) connectionStatus() ui.ConnectionStatus {
	switch m.state {
	case StateWaitingResponse:
		return ui.StatusActive
	case StateExiting:
		return ui.StatusError
	default:
		return ui.StatusIdle
	}
}
```

---

## 6. Performance Requirements

### 6.1 Rendering Performance

**Target: <1ms render time**

- Minimal string allocations
- Use strings.Builder for concatenation
- Cache formatted values where possible

**Measurement:**
```bash
go test -bench=BenchmarkStatusBar_Render -benchmem
```

**Expected:**
```
BenchmarkStatusBar_Render-8   	 2000000	       500 ns/op	     256 B/op	       8 allocs/op
```

### 6.2 Memory Usage

**Target: <1KB per render**

- Reuse buffers
- Avoid unnecessary allocations
- Profile with `go test -memprofile=mem.prof`

---

## 7. Dependencies

### 7.1 Required Packages

```go
import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)
```

No additional dependencies beyond what's already used in other UI components.

---

## 8. Quality Checklist

### 8.1 Definition of Ready (DoR)

- [x] Status bar information list defined
- [x] Layout specification reviewed
- [x] Dynamic updates understood
- [x] Responsive behavior designed

### 8.2 Definition of Done (DoD)

- [ ] Tests for rendering (≥85% coverage)
- [ ] Tests for token formatting
- [ ] Tests for responsive layout
- [ ] Status bar updates in real-time
- [ ] Layout works at different widths (40, 60, 80, 100, 120 cols)
- [ ] Icons display correctly
- [ ] Token count formatting accurate
- [ ] All tests passing with race detector
- [ ] Linter clean (make lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] Integrated into TUI model
- [ ] ROADMAP updated

---

## 9. Risks and Mitigations

### 9.1 Risks

1. **Icon compatibility** - Not all terminals support Unicode emojis
   - **Mitigation:** Fallback to ASCII characters if NO_COLOR set or emoji detection fails

2. **Layout complexity** - Many different terminal widths to support
   - **Mitigation:** Test with multiple widths (40-120), progressive enhancement

3. **Performance** - Frequent status updates could slow rendering
   - **Mitigation:** Cache formatted values, only re-render when info changes

### 9.2 Fallback Plan

If Unicode icons cause issues:
- Provide ASCII-only mode (e.g., `[R]` for read-only, `[W]` for write)
- Detect terminal capabilities and choose appropriate glyphs
- Add `--no-icons` flag for compatibility

---

## 10. Success Criteria

Phase 3.6 is complete when:

1. ✅ Status bar displays all required information
2. ✅ Layout adapts to terminal width (40-120+ columns)
3. ✅ Icons display correctly (with fallback for incompatible terminals)
4. ✅ Token counts formatted correctly (K/M suffixes)
5. ✅ Working directory abbreviated appropriately
6. ✅ Status updates in real-time based on app state
7. ✅ All tests passing with ≥85% coverage
8. ✅ Performance target met (<1ms render)
9. ✅ No lint errors, complexity ≤15
10. ✅ Integrated into TUI app.go and view.go

---

## 11. Future Enhancements (Out of Scope)

These will be considered for later iterations:

- Clickable status bar elements (mouse support)
- Customizable status bar format (via config)
- Additional metrics (response time, error rate)
- Color themes based on user preference
- Status bar notifications (transient messages)

---

## 12. References

- [specs/ui-modules/spec.md](../ui-modules/spec.md) - Status bar requirements
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.6
- [FRD-UI-3.1](FRD-UI-3.1.md) - TUI application setup
- [FRD-UI-3.2](FRD-UI-3.2.md) - Chat interface components
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss) - Styling framework
- [AGENTS.md](../../AGENTS.md) - Quality standards

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** Spin Development Team
