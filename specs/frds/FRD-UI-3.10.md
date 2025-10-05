# FRD-UI-3.10: TUI Styling & Themes

**Feature**: Centralized theme system with multiple color schemes
**Package**: `internal/tui/theme`
**Priority**: Medium
**Complexity**: Medium
**Status**: In Progress

---

## 1. Overview

Implement a centralized theme system for the TUI that:
- Consolidates all hardcoded colors into a theme abstraction
- Supports multiple color schemes (dark, light, auto-detect)
- Respects NO_COLOR environment variable for accessibility
- Integrates with existing configuration system
- Provides pre-computed lipgloss styles for optimal performance

---

## 2. Current State

### Problems
1. **Hardcoded colors** scattered across UI components:
   - `chat.go`: User (12), Assistant (10), System (11), Tool (14) colors
   - `statusbar.go`: Background (236), Foreground (250), Active (10), Error (9)
   - `approval.go`: Border colors, button colors
   - `help.go`: Title colors, section colors
   - `filepicker.go`: Selection colors

2. **No theme support**: Cannot switch between dark/light modes
3. **No NO_COLOR support**: Accessibility issue for users who need plain text
4. **Poor maintainability**: Changing colors requires edits in multiple files
5. **No auto-detection**: Cannot adapt to terminal color scheme

### Goals
- Centralize all color definitions in `internal/tui/theme` package
- Support 3 theme modes: `dark` (default), `light`, `auto`
- Respect NO_COLOR environment variable
- Integrate with `internal/config` package
- Pre-compute styles for <16ms render latency
- Maintain ≥85% test coverage

---

## 3. Architecture

### 3.1 Package Structure

```
internal/tui/theme/
├── doc.go            # Package documentation
├── theme.go          # Theme interface + factory
├── dark.go           # Dark theme implementation
├── light.go          # Light theme implementation
├── auto.go           # Auto-detect theme
├── scheme.go         # ColorScheme type
├── styles.go         # Pre-computed lipgloss styles
└── theme_test.go     # Tests (≥85% coverage)
```

### 3.2 Core Types

```go
// Theme provides consistent styling for all TUI components.
type Theme interface {
    // Name returns the theme name ("dark", "light", "auto").
    Name() string

    // Colors returns the color scheme.
    Colors() ColorScheme

    // ChatStyles returns pre-computed styles for chat component.
    ChatStyles() ChatStyleSet

    // StatusBarStyles returns pre-computed styles for status bar.
    StatusBarStyles() StatusBarStyleSet

    // ApprovalStyles returns pre-computed styles for approval modal.
    ApprovalStyles() ApprovalStyleSet

    // HelpStyles returns pre-computed styles for help modal.
    HelpStyles() HelpStyleSet

    // FilePickerStyles returns pre-computed styles for file picker.
    FilePickerStyles() FilePickerStyleSet

    // InputStyles returns pre-computed styles for input widget.
    InputStyles() InputStyleSet

    // SupportsColors returns true if colors are enabled.
    SupportsColors() bool
}

// ColorScheme defines the color palette.
type ColorScheme struct {
    // Role colors
    User      lipgloss.Color
    Assistant lipgloss.Color
    System    lipgloss.Color
    Tool      lipgloss.Color

    // State colors
    Error     lipgloss.Color
    Success   lipgloss.Color
    Warning   lipgloss.Color
    Info      lipgloss.Color

    // UI element colors
    Background       lipgloss.Color
    Foreground       lipgloss.Color
    Border           lipgloss.Color
    BorderActive     lipgloss.Color
    Selection        lipgloss.Color
    Highlight        lipgloss.Color

    // Status bar colors
    StatusBarBg      lipgloss.Color
    StatusBarFg      lipgloss.Color
    StatusBarActive  lipgloss.Color
    StatusBarError   lipgloss.Color
}

// ChatStyleSet contains all styles for chat component.
type ChatStyleSet struct {
    User        lipgloss.Style
    Assistant   lipgloss.Style
    System      lipgloss.Style
    Tool        lipgloss.Style
    ToolCall    lipgloss.Style  // Border style for tool calls
    ToolResult  lipgloss.Style  // Border style for results
    Reasoning   lipgloss.Style  // Border style for reasoning blocks
    Error       lipgloss.Style  // Border style for errors
    Highlight   lipgloss.Style  // Border style for highlighted messages
}

// StatusBarStyleSet contains all styles for status bar.
type StatusBarStyleSet struct {
    Normal lipgloss.Style
    Active lipgloss.Style
    Error  lipgloss.Style
}

// ApprovalStyleSet contains all styles for approval modal.
type ApprovalStyleSet struct {
    Modal       lipgloss.Style
    Title       lipgloss.Style
    Command     lipgloss.Style
    Reason      lipgloss.Style
    ButtonBase  lipgloss.Style
    ButtonFocus lipgloss.Style
}

// HelpStyleSet contains all styles for help modal.
type HelpStyleSet struct {
    Modal    lipgloss.Style
    Title    lipgloss.Style
    Section  lipgloss.Style
    Shortcut lipgloss.Style
    Desc     lipgloss.Style
}

// FilePickerStyleSet contains all styles for file picker.
type FilePickerStyleSet struct {
    Modal     lipgloss.Style
    Title     lipgloss.Style
    Selected  lipgloss.Style
    Normal    lipgloss.Style
    Matched   lipgloss.Style
}

// InputStyleSet contains all styles for input widget.
type InputStyleSet struct {
    Normal lipgloss.Style
    Focused lipgloss.Style
    Placeholder lipgloss.Style
}
```

### 3.3 Factory Pattern

```go
// New creates a new theme based on name and NO_COLOR setting.
func New(name string, noColor bool) (Theme, error) {
    if noColor || os.Getenv("NO_COLOR") != "" {
        return newPlainTheme(), nil
    }

    switch name {
    case "dark", "":
        return newDarkTheme(), nil
    case "light":
        return newLightTheme(), nil
    case "auto":
        return newAutoTheme(), nil
    default:
        return nil, fmt.Errorf("unknown theme: %s", name)
    }
}

// FromConfig creates a theme from configuration.
func FromConfig(cfg *config.Config) (Theme, error) {
    themeName := cfg.Appearance.Theme
    noColor := cfg.Appearance.NoColor
    return New(themeName, noColor)
}
```

---

## 4. Implementation Details

### 4.1 Dark Theme (Default)

```go
// DarkTheme implements Theme for dark terminal backgrounds.
type DarkTheme struct {
    colors ColorScheme
    chat   ChatStyleSet
    status StatusBarStyleSet
    // ... other style sets
}

func newDarkTheme() *DarkTheme {
    colors := ColorScheme{
        // Role colors (256-color palette)
        User:      lipgloss.Color("12"),  // Bright blue
        Assistant: lipgloss.Color("10"),  // Bright green
        System:    lipgloss.Color("11"),  // Bright yellow
        Tool:      lipgloss.Color("14"),  // Bright cyan

        // State colors
        Error:     lipgloss.Color("9"),   // Bright red
        Success:   lipgloss.Color("10"),  // Bright green
        Warning:   lipgloss.Color("11"),  // Bright yellow
        Info:      lipgloss.Color("14"),  // Bright cyan

        // UI element colors
        Background:      lipgloss.Color("0"),    // Black
        Foreground:      lipgloss.Color("7"),    // White
        Border:          lipgloss.Color("240"),  // Dark gray
        BorderActive:    lipgloss.Color("12"),   // Blue
        Selection:       lipgloss.Color("4"),    // Blue bg
        Highlight:       lipgloss.Color("226"),  // Yellow

        // Status bar
        StatusBarBg:     lipgloss.Color("236"),  // Very dark gray
        StatusBarFg:     lipgloss.Color("250"),  // Light gray
        StatusBarActive: lipgloss.Color("10"),   // Green
        StatusBarError:  lipgloss.Color("9"),    // Red
    }

    // Pre-compute all styles for performance
    chat := ChatStyleSet{
        User: lipgloss.NewStyle().
            Foreground(colors.User).
            Bold(true),
        Assistant: lipgloss.NewStyle().
            Foreground(colors.Assistant).
            Bold(true),
        // ... etc
    }

    // ... pre-compute other style sets

    return &DarkTheme{
        colors: colors,
        chat:   chat,
        // ...
    }
}

func (t *DarkTheme) Name() string { return "dark" }
func (t *DarkTheme) Colors() ColorScheme { return t.colors }
func (t *DarkTheme) ChatStyles() ChatStyleSet { return t.chat }
func (t *DarkTheme) SupportsColors() bool { return true }
// ... other methods
```

### 4.2 Light Theme

```go
// LightTheme implements Theme for light terminal backgrounds.
type LightTheme struct {
    colors ColorScheme
    // ... similar to DarkTheme
}

func newLightTheme() *LightTheme {
    colors := ColorScheme{
        // Role colors (darker for light backgrounds)
        User:      lipgloss.Color("4"),   // Dark blue
        Assistant: lipgloss.Color("2"),   // Dark green
        System:    lipgloss.Color("3"),   // Dark yellow/brown
        Tool:      lipgloss.Color("6"),   // Dark cyan

        // State colors
        Error:     lipgloss.Color("1"),   // Dark red
        Success:   lipgloss.Color("2"),   // Dark green
        Warning:   lipgloss.Color("3"),   // Dark yellow
        Info:      lipgloss.Color("4"),   // Dark blue

        // UI elements (inverted from dark theme)
        Background:      lipgloss.Color("15"),   // White
        Foreground:      lipgloss.Color("0"),    // Black
        Border:          lipgloss.Color("8"),    // Gray
        BorderActive:    lipgloss.Color("4"),    // Blue
        Selection:       lipgloss.Color("12"),   // Light blue bg
        Highlight:       lipgloss.Color("3"),    // Dark yellow

        // Status bar
        StatusBarBg:     lipgloss.Color("7"),    // Light gray
        StatusBarFg:     lipgloss.Color("0"),    // Black
        StatusBarActive: lipgloss.Color("2"),    // Green
        StatusBarError:  lipgloss.Color("1"),    // Red
    }

    // ... pre-compute styles

    return &LightTheme{colors: colors /* ... */}
}
```

### 4.3 Auto Theme

```go
// AutoTheme detects terminal background and chooses dark/light.
type AutoTheme struct {
    delegate Theme
}

func newAutoTheme() *AutoTheme {
    // Heuristic: check terminal background color
    // Use golang.org/x/term or fallback to dark

    isDark := detectDarkTerminal()

    var delegate Theme
    if isDark {
        delegate = newDarkTheme()
    } else {
        delegate = newLightTheme()
    }

    return &AutoTheme{delegate: delegate}
}

func (t *AutoTheme) Name() string { return "auto" }
// Delegate all methods to underlying theme
func (t *AutoTheme) Colors() ColorScheme { return t.delegate.Colors() }
// ...

// detectDarkTerminal attempts to detect if terminal has dark background.
func detectDarkTerminal() bool {
    // Method 1: Check $COLORFGBG environment variable (some terminals)
    if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
        // Format: "foreground;background"
        // Dark bg typically has low number, light bg has high number
        parts := strings.Split(colorfgbg, ";")
        if len(parts) == 2 {
            if bg, err := strconv.Atoi(parts[1]); err == nil {
                return bg < 8 // Colors 0-7 are dark
            }
        }
    }

    // Method 2: Check terminal type
    term := os.Getenv("TERM")
    if strings.Contains(term, "dark") {
        return true
    }
    if strings.Contains(term, "light") {
        return false
    }

    // Method 3: Fallback to dark (most common for developers)
    return true
}
```

### 4.4 Plain Theme (NO_COLOR)

```go
// PlainTheme provides no colors for accessibility (NO_COLOR support).
type PlainTheme struct {
    colors ColorScheme
    chat   ChatStyleSet
    // ... other style sets with no colors
}

func newPlainTheme() *PlainTheme {
    // All colors are empty/default
    colors := ColorScheme{
        User:      lipgloss.NoColor{},
        Assistant: lipgloss.NoColor{},
        // ... all NoColor
    }

    // Styles with no colors, only structural formatting
    chat := ChatStyleSet{
        User: lipgloss.NewStyle().Bold(true),  // Just bold, no color
        Assistant: lipgloss.NewStyle().Bold(true),
        // ...
    }

    return &PlainTheme{colors: colors, chat: chat}
}

func (t *PlainTheme) SupportsColors() bool { return false }
```

---

## 5. Configuration Integration

### 5.1 Config Schema (Already Exists)

```yaml
# ~/.spin/spin.yaml
appearance:
  theme: auto          # Options: auto, dark, light
  no_color: false      # Disable colors (accessibility)
```

### 5.2 CLI Flags (TUI Command)

```go
// cmd/spin/tui.go
tuiCmd.Flags().String("theme", "auto", "Color theme (auto, dark, light)")
tuiCmd.Flags().Bool("no-color", false, "Disable colors")
```

### 5.3 Environment Variables

```bash
SPIN_APPEARANCE_THEME=dark     # Override config
SPIN_APPEARANCE_NO_COLOR=1     # Disable colors
NO_COLOR=1                     # Standard NO_COLOR support
```

---

## 6. Component Integration

### 6.1 TUI Model (app.go)

```go
// Model initialization
func InitialModel(cfg *config.Config) Model {
    theme, err := theme.FromConfig(cfg)
    if err != nil {
        // Fallback to dark theme
        theme, _ = theme.New("dark", false)
    }

    return Model{
        theme:     theme,
        chat:      ui.NewChatWithTheme(width, height, theme),
        statusBar: ui.NewStatusBarWithTheme(width, theme),
        approval:  ui.NewApprovalWithTheme(theme),
        help:      ui.NewHelpWithTheme(theme),
        // ...
    }
}
```

### 6.2 UI Components Update

**Before (chat.go):**
```go
// Hardcoded colors
style := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
```

**After (chat.go):**
```go
// Use theme styles
func NewChatWithTheme(width, height int, theme theme.Theme) Chat {
    styles := theme.ChatStyles()

    return Chat{
        // ...
        styles: styles,
    }
}

// In render method
func (c Chat) renderUserMessage(msg Message) string {
    return c.styles.User.Render(msg.Content)
}
```

---

## 7. Testing Strategy

### 7.1 Test Coverage (≥85%)

```go
// theme_test.go

func TestThemeFactory(t *testing.T) {
    tests := []struct {
        name     string
        themeName string
        noColor  bool
        wantErr  bool
        wantType string
    }{
        {"dark theme", "dark", false, false, "*DarkTheme"},
        {"light theme", "light", false, false, "*LightTheme"},
        {"auto theme", "auto", false, false, "*AutoTheme"},
        {"no color", "dark", true, false, "*PlainTheme"},
        {"NO_COLOR env", "dark", false, false, "*PlainTheme"}, // with env set
        {"unknown theme", "invalid", false, true, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.name == "NO_COLOR env" {
                t.Setenv("NO_COLOR", "1")
            }

            theme, err := New(tt.themeName, tt.noColor)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.wantType, fmt.Sprintf("%T", theme))
        })
    }
}

func TestDarkTheme_Colors(t *testing.T) {
    theme := newDarkTheme()

    colors := theme.Colors()

    // Test role colors
    assert.Equal(t, lipgloss.Color("12"), colors.User)
    assert.Equal(t, lipgloss.Color("10"), colors.Assistant)
    // ... test all colors
}

func TestDarkTheme_Styles(t *testing.T) {
    theme := newDarkTheme()

    // Test chat styles
    chat := theme.ChatStyles()
    assert.NotNil(t, chat.User)
    assert.NotNil(t, chat.Assistant)

    // Test styles are pre-computed (not nil)
    assert.NotNil(t, theme.StatusBarStyles().Normal)
    assert.NotNil(t, theme.ApprovalStyles().Modal)
}

func TestPlainTheme_NoColors(t *testing.T) {
    theme := newPlainTheme()

    assert.False(t, theme.SupportsColors())

    colors := theme.Colors()
    // All colors should be NoColor type
    assert.IsType(t, lipgloss.NoColor{}, colors.User)
}

func TestAutoTheme_Detection(t *testing.T) {
    tests := []struct {
        name        string
        colorfgbg   string
        term        string
        expectDark  bool
    }{
        {"dark bg", "7;0", "", true},
        {"light bg", "0;15", "", false},
        {"term dark", "", "xterm-256color-dark", true},
        {"term light", "", "xterm-light", false},
        {"fallback", "", "", true}, // Default to dark
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.colorfgbg != "" {
                t.Setenv("COLORFGBG", tt.colorfgbg)
            }
            if tt.term != "" {
                t.Setenv("TERM", tt.term)
            }

            isDark := detectDarkTerminal()
            assert.Equal(t, tt.expectDark, isDark)
        })
    }
}

func TestFromConfig(t *testing.T) {
    cfg := &config.Config{
        Appearance: config.Appearance{
            Theme:   "light",
            NoColor: false,
        },
    }

    theme, err := FromConfig(cfg)

    require.NoError(t, err)
    assert.Equal(t, "light", theme.Name())
}

// Benchmark style creation
func BenchmarkThemeCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = newDarkTheme()
    }
}

// Benchmark style access (should be fast - cached)
func BenchmarkStyleAccess(b *testing.B) {
    theme := newDarkTheme()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = theme.ChatStyles()
    }
}
```

### 7.2 Integration Tests

```go
// Test theme integration with UI components
func TestChatWithTheme(t *testing.T) {
    theme := newDarkTheme()
    chat := ui.NewChatWithTheme(80, 24, theme)

    chat.AddMessage(ui.Message{
        Role:    ui.RoleUser,
        Content: "Hello",
    })

    view := chat.View()

    // View should contain styled content (has ANSI codes)
    assert.Contains(t, view, "\x1b[") // ANSI escape code
}
```

---

## 8. Performance Considerations

### 8.1 Style Caching
- **Pre-compute all styles** in theme constructor
- **Cache style objects** (don't recreate in render loop)
- **Target**: <16ms render latency (60 FPS)

### 8.2 Benchmarks

```go
func BenchmarkRenderWithTheme(b *testing.B) {
    theme := newDarkTheme()
    chat := ui.NewChatWithTheme(80, 24, theme)

    // Add test messages
    for i := 0; i < 100; i++ {
        chat.AddMessage(ui.Message{
            Role: ui.RoleAssistant,
            Content: "Test message",
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = chat.View()
    }
}
```

**Expected**: <1ms for 100 messages (well under 16ms target)

---

## 9. Documentation

### 9.1 Package Doc (doc.go)

```go
// Package theme provides centralized color schemes and styling for the Spin TUI.
//
// The theme package supports multiple color schemes (dark, light, auto-detect)
// and integrates with the configuration system. It respects the NO_COLOR
// environment variable for accessibility.
//
// # Themes
//
// Three built-in themes are available:
//
//   - dark: Optimized for dark terminal backgrounds (default)
//   - light: Optimized for light terminal backgrounds
//   - auto: Automatically detects terminal background color
//
// # Usage
//
// Create a theme from configuration:
//
//     theme, err := theme.FromConfig(cfg)
//     if err != nil {
//         // Handle error
//     }
//
// Or create directly:
//
//     theme, err := theme.New("dark", false)
//
// Apply theme to UI components:
//
//     chat := ui.NewChatWithTheme(width, height, theme)
//     statusBar := ui.NewStatusBarWithTheme(width, theme)
//
// # NO_COLOR Support
//
// The theme system respects the NO_COLOR environment variable for accessibility.
// When NO_COLOR is set, all colors are disabled and a plain theme is used:
//
//     NO_COLOR=1 spin tui
//
// This can also be configured via config file:
//
//     appearance:
//       no_color: true
//
// # Configuration
//
// Themes can be configured via:
//
//   - Config file: ~/.spin/spin.yaml (appearance.theme)
//   - CLI flag: --theme dark|light|auto
//   - Environment: SPIN_APPEARANCE_THEME=dark
//   - NO_COLOR env var (disables all colors)
//
// # Performance
//
// All styles are pre-computed in the theme constructor for optimal performance.
// Style access (e.g., theme.ChatStyles()) returns cached style objects with
// no runtime overhead.
//
// # Related Packages
//
//   - internal/config: Configuration loading and precedence
//   - internal/tui/ui: UI components that consume themes
package theme
```

### 9.2 User Documentation

Update `docs/packages/` with theme documentation (done after implementation).

---

## 10. Migration Plan

### 10.1 Phase 1: Create Theme Package
1. Create `internal/tui/theme/` package
2. Implement Theme interface and factory
3. Implement DarkTheme, LightTheme, AutoTheme, PlainTheme
4. Write comprehensive tests (≥85% coverage)

### 10.2 Phase 2: Update UI Components
1. Add `WithTheme()` constructors to all UI components
2. Replace hardcoded colors with theme.Styles()
3. Update existing tests
4. Add integration tests

### 10.3 Phase 3: Integration
1. Update `cmd/spin/tui.go` to load theme from config
2. Add CLI flags (--theme, --no-color)
3. Test all theme modes
4. Update documentation

### 10.4 Backward Compatibility
- Default theme is "dark" (current behavior)
- If theme system fails, fallback to hardcoded dark theme
- No breaking changes to public APIs

---

## 11. Quality Gates

### Before Marking Complete

- [ ] All tests passing (≥85% coverage)
- [ ] Race detector clean (`go test -race ./internal/tui/theme`)
- [ ] Linter clean (`make lint`)
- [ ] Complexity ≤15 (all functions)
- [ ] Godoc complete on all exports
- [ ] Benchmarks show <16ms render time
- [ ] NO_COLOR support verified
- [ ] All 3 themes render correctly
- [ ] Auto-detection works on test terminals
- [ ] Integration with config system complete
- [ ] CLI flags functional

---

## 12. Success Criteria

1. **Functionality**:
   - ✅ Dark theme works (current behavior preserved)
   - ✅ Light theme works (new)
   - ✅ Auto-detection works (new)
   - ✅ NO_COLOR support works (new)
   - ✅ Config integration works
   - ✅ CLI flags work

2. **Code Quality**:
   - ✅ Tests ≥85% coverage
   - ✅ All lint checks pass
   - ✅ Complexity ≤15
   - ✅ No hardcoded colors in UI components
   - ✅ Godoc complete

3. **Performance**:
   - ✅ Render latency <16ms (60 FPS)
   - ✅ Theme creation <10ms
   - ✅ Style access <1μs (cached)

4. **Documentation**:
   - ✅ Package doc.go complete
   - ✅ All exports documented
   - ✅ Usage examples provided
   - ✅ Migration guide written

---

## 13. Related Files

**Create:**
- `internal/tui/theme/doc.go`
- `internal/tui/theme/theme.go`
- `internal/tui/theme/dark.go`
- `internal/tui/theme/light.go`
- `internal/tui/theme/auto.go`
- `internal/tui/theme/plain.go`
- `internal/tui/theme/scheme.go`
- `internal/tui/theme/styles.go`
- `internal/tui/theme/theme_test.go`

**Modify:**
- `internal/tui/app.go` (theme initialization)
- `internal/tui/ui/chat.go` (use theme styles)
- `internal/tui/ui/statusbar.go` (use theme styles)
- `internal/tui/ui/approval.go` (use theme styles)
- `internal/tui/ui/help.go` (use theme styles)
- `internal/tui/ui/filepicker.go` (use theme styles)
- `internal/tui/ui/input.go` (use theme styles)
- `cmd/spin/tui.go` (add flags, load theme)
- `internal/config/config.go` (ensure Appearance struct exists)

**Documentation:**
- `docs/packages/theme.md` (new, created after implementation)
- Update `specs/ui-modules/ROADMAP.md` (mark 3.10 complete)

---

## 14. References

- [AGENTS.md](../../AGENTS.md) - Implementation workflow
- [architecture-overview.md](../architecture-overview.md) - Overall architecture
- [config.md](../../docs/packages/config.md) - Configuration patterns
- [lipgloss documentation](https://github.com/charmbracelet/lipgloss) - Styling library
- [NO_COLOR specification](https://no-color.org/) - Accessibility standard
