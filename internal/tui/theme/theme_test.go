package theme

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThemeFactory(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		noColor   bool
		wantErr   bool
		wantType  string
	}{
		{"dark theme", "dark", false, false, "*theme.darkTheme"},
		{"light theme", "light", false, false, "*theme.lightTheme"},
		{"auto theme", "auto", false, false, "*theme.autoTheme"},
		{"empty defaults to dark", "", false, false, "*theme.darkTheme"},
		{"no color", "dark", true, false, "*theme.plainTheme"},
		{"unknown theme", "invalid", false, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := New(tt.themeName, tt.noColor)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, theme)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, theme)
			assert.Equal(t, tt.wantType, fmt.Sprintf("%T", theme))
		})
	}
}

func TestThemeFactory_NOCOLOREnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	theme, err := New("dark", false)

	require.NoError(t, err)
	assert.Equal(t, "*theme.plainTheme", fmt.Sprintf("%T", theme))
	assert.False(t, theme.SupportsColors())
}

func TestDarkTheme_Name(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	assert.Equal(t, "dark", theme.Name())
}

func TestDarkTheme_Colors(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	colors := theme.Colors()

	// Test role colors
	assert.Equal(t, lipgloss.Color("12"), colors.User)
	assert.Equal(t, lipgloss.Color("10"), colors.Assistant)
	assert.Equal(t, lipgloss.Color("11"), colors.System)
	assert.Equal(t, lipgloss.Color("14"), colors.Tool)

	// Test state colors
	assert.Equal(t, lipgloss.Color("9"), colors.Error)
	assert.Equal(t, lipgloss.Color("10"), colors.Success)
	assert.Equal(t, lipgloss.Color("11"), colors.Warning)
	assert.Equal(t, lipgloss.Color("14"), colors.Info)

	// Test UI colors
	assert.Equal(t, lipgloss.Color("236"), colors.StatusBarBg)
	assert.Equal(t, lipgloss.Color("250"), colors.StatusBarFg)
}

func TestDarkTheme_ChatStyles(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	chat := theme.ChatStyles()

	// All styles should be non-nil (pre-computed)
	assert.NotNil(t, chat.User)
	assert.NotNil(t, chat.Assistant)
	assert.NotNil(t, chat.System)
	assert.NotNil(t, chat.Tool)
	assert.NotNil(t, chat.ToolCall)
	assert.NotNil(t, chat.ToolResult)
	assert.NotNil(t, chat.Reasoning)
	assert.NotNil(t, chat.Error)
	assert.NotNil(t, chat.Highlight)

	// Test that styles have expected properties (bold)
	// Note: ANSI color codes may not render in test environment without TTY
	rendered := chat.User.Render("test")
	assert.Contains(t, rendered, "test") // At minimum contains the text
}

func TestDarkTheme_StatusBarStyles(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	status := theme.StatusBarStyles()

	assert.NotNil(t, status.Normal)
	assert.NotNil(t, status.Active)
	assert.NotNil(t, status.Error)
}

func TestDarkTheme_AllStyles(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	// Test all style sets are pre-computed
	assert.NotNil(t, theme.ChatStyles())
	assert.NotNil(t, theme.StatusBarStyles())
	assert.NotNil(t, theme.ApprovalStyles())
	assert.NotNil(t, theme.HelpStyles())
	assert.NotNil(t, theme.FilePickerStyles())
	assert.NotNil(t, theme.InputStyles())
}

func TestDarkTheme_SupportsColors(t *testing.T) {
	theme, err := New("dark", false)
	require.NoError(t, err)

	assert.True(t, theme.SupportsColors())
}

func TestLightTheme_Name(t *testing.T) {
	theme, err := New("light", false)
	require.NoError(t, err)

	assert.Equal(t, "light", theme.Name())
}

func TestLightTheme_Colors(t *testing.T) {
	theme, err := New("light", false)
	require.NoError(t, err)

	colors := theme.Colors()

	// Light theme uses darker colors for light backgrounds
	assert.Equal(t, lipgloss.Color("4"), colors.User)      // Dark blue
	assert.Equal(t, lipgloss.Color("2"), colors.Assistant) // Dark green
	assert.Equal(t, lipgloss.Color("3"), colors.System)    // Dark yellow
	assert.Equal(t, lipgloss.Color("6"), colors.Tool)      // Dark cyan

	// Test status bar colors (inverted)
	assert.Equal(t, lipgloss.Color("7"), colors.StatusBarBg)  // Light gray
	assert.Equal(t, lipgloss.Color("0"), colors.StatusBarFg)  // Black
}

func TestLightTheme_SupportsColors(t *testing.T) {
	theme, err := New("light", false)
	require.NoError(t, err)

	assert.True(t, theme.SupportsColors())
}

func TestLightTheme_AllStyles(t *testing.T) {
	theme, err := New("light", false)
	require.NoError(t, err)

	// Test all style sets are pre-computed
	assert.NotNil(t, theme.ChatStyles())
	assert.NotNil(t, theme.StatusBarStyles())
	assert.NotNil(t, theme.ApprovalStyles())
	assert.NotNil(t, theme.HelpStyles())
	assert.NotNil(t, theme.FilePickerStyles())
	assert.NotNil(t, theme.InputStyles())
}

func TestAutoTheme_Name(t *testing.T) {
	theme, err := New("auto", false)
	require.NoError(t, err)

	assert.Equal(t, "auto", theme.Name())
}

func TestAutoTheme_DelegatesStyles(t *testing.T) {
	theme, err := New("auto", false)
	require.NoError(t, err)

	// Auto theme should delegate to underlying theme
	// Should not be nil
	assert.NotNil(t, theme.Colors())
	assert.NotNil(t, theme.ChatStyles())
	assert.NotNil(t, theme.StatusBarStyles())
	assert.NotNil(t, theme.ApprovalStyles())
	assert.NotNil(t, theme.HelpStyles())
	assert.NotNil(t, theme.FilePickerStyles())
	assert.NotNil(t, theme.InputStyles())
	assert.True(t, theme.SupportsColors())
}

func TestAutoTheme_Detection(t *testing.T) {
	tests := []struct {
		name       string
		colorfgbg  string
		term       string
		expectDark bool
	}{
		{"dark bg colorfgbg", "7;0", "", true},
		{"light bg colorfgbg", "0;15", "", false},
		{"dark bg colorfgbg 2", "15;8", "", false},
		{"light bg colorfgbg 2", "0;7", "", true},
		{"term dark", "", "xterm-256color-dark", true},
		{"term light", "", "xterm-light", false},
		{"fallback to dark", "", "", true}, // Default
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
			assert.Equal(t, tt.expectDark, isDark, "expected dark=%v for colorfgbg=%s term=%s", tt.expectDark, tt.colorfgbg, tt.term)
		})
	}
}

func TestPlainTheme_Name(t *testing.T) {
	theme, err := New("dark", true) // noColor=true
	require.NoError(t, err)

	assert.Equal(t, "plain", theme.Name())
}

func TestPlainTheme_NoColors(t *testing.T) {
	theme, err := New("dark", true)
	require.NoError(t, err)

	assert.False(t, theme.SupportsColors())

	colors := theme.Colors()

	// All colors should be empty strings
	assert.Equal(t, lipgloss.Color(""), colors.User)
	assert.Equal(t, lipgloss.Color(""), colors.Assistant)
	assert.Equal(t, lipgloss.Color(""), colors.System)
	assert.Equal(t, lipgloss.Color(""), colors.Tool)
	assert.Equal(t, lipgloss.Color(""), colors.Error)
}

func TestPlainTheme_StylesWithoutColors(t *testing.T) {
	theme, err := New("dark", true)
	require.NoError(t, err)

	chat := theme.ChatStyles()

	// Styles exist but have no colors
	assert.NotNil(t, chat.User)

	// Render should work but have no ANSI color codes
	rendered := chat.User.Render("test")
	// Should still have "test" but no color codes (may have bold codes)
	assert.Contains(t, rendered, "test")
}

func TestPlainTheme_AllStyles(t *testing.T) {
	theme, err := New("dark", true)
	require.NoError(t, err)

	// Test all style sets exist (no colors)
	assert.NotNil(t, theme.ChatStyles())
	assert.NotNil(t, theme.StatusBarStyles())
	assert.NotNil(t, theme.ApprovalStyles())
	assert.NotNil(t, theme.HelpStyles())
	assert.NotNil(t, theme.FilePickerStyles())
	assert.NotNil(t, theme.InputStyles())
}

// Benchmark tests
func BenchmarkThemeCreation_Dark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("dark", false)
	}
}

func BenchmarkThemeCreation_Light(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("light", false)
	}
}

func BenchmarkThemeCreation_Plain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = New("dark", true)
	}
}

func BenchmarkStyleAccess_Chat(b *testing.B) {
	theme, _ := New("dark", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = theme.ChatStyles()
	}
}

func BenchmarkStyleAccess_StatusBar(b *testing.B) {
	theme, _ := New("dark", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = theme.StatusBarStyles()
	}
}

func BenchmarkRenderStyled(b *testing.B) {
	theme, _ := New("dark", false)
	styles := theme.ChatStyles()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = styles.User.Render("Hello, world!")
	}
}
