package ui

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar(120)

	assert.Equal(t, 120, sb.width)
	assert.Equal(t, StatusIdle, sb.info.Status)
	assert.NotNil(t, sb.style)
}

func TestStatusBar_SetInfo(t *testing.T) {
	sb := NewStatusBar(120)

	info := StatusInfo{
		Model:         "llama3.1",
		Provider:      "ollama",
		SandboxMode:   "read-only",
		WorkingDir:    "/home/user/project",
		Status:        StatusActive,
		TurnTokens:    1200,
		SessionTokens: 5400,
	}

	sb.SetInfo(info)

	assert.Equal(t, "llama3.1", sb.info.Model)
	assert.Equal(t, "ollama", sb.info.Provider)
	assert.Equal(t, "read-only", sb.info.SandboxMode)
	assert.Equal(t, "/home/user/project", sb.info.WorkingDir)
	assert.Equal(t, StatusActive, sb.info.Status)
	assert.Equal(t, 1200, sb.info.TurnTokens)
	assert.Equal(t, 5400, sb.info.SessionTokens)
}

func TestStatusBar_SetWidth(t *testing.T) {
	sb := NewStatusBar(80)
	assert.Equal(t, 80, sb.width)

	sb.SetWidth(120)
	assert.Equal(t, 120, sb.width)
}

func TestStatusBar_View_FullWidth(t *testing.T) {
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

	view := sb.View()

	// Should contain model name
	assert.Contains(t, view, "llama3.1")
	// Should contain sandbox mode
	assert.Contains(t, view, "read-only")
	// Width should not exceed terminal width
	assert.LessOrEqual(t, lipgloss.Width(view), 120)
}

func TestStatusBar_View_MediumWidth(t *testing.T) {
	sb := NewStatusBar(80)
	info := StatusInfo{
		Model:       "gpt-4o",
		SandboxMode: "workspace-write",
		WorkingDir:  "/project",
		Status:      StatusIdle,
		TurnTokens:  500,
	}
	sb.SetInfo(info)

	view := sb.View()

	assert.Contains(t, view, "gpt-4o")
	assert.LessOrEqual(t, lipgloss.Width(view), 80)
}

func TestStatusBar_View_NarrowWidth(t *testing.T) {
	sb := NewStatusBar(40)
	info := StatusInfo{
		Model:       "mixtral",
		SandboxMode: "read-only",
		Status:      StatusError,
		TurnTokens:  100,
	}
	sb.SetInfo(info)

	view := sb.View()

	// Should still contain essential info (model)
	assert.Contains(t, view, "mixtral")
	// Should fit within width
	assert.LessOrEqual(t, lipgloss.Width(view), 40)
}

func TestStatusBar_View_VeryNarrowWidth(t *testing.T) {
	sb := NewStatusBar(30)
	info := StatusInfo{
		Model:       "llama3.1",
		SandboxMode: "read-only",
		Status:      StatusActive,
		TurnTokens:  1200,
	}
	sb.SetInfo(info)

	view := sb.View()

	// Compact mode: should contain model
	assert.Contains(t, view, "llama3.1")
	assert.LessOrEqual(t, lipgloss.Width(view), 30)
}

func TestStatusBar_TokenFormatting(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "0"},
		{5, "5"},
		{123, "123"},
		{999, "999"},
		{1000, "1.0K"},
		{1234, "1.2K"},
		{5400, "5.4K"},
		{10000, "10.0K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1234567, "1.2M"},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(fmt.Sprintf("count_%d", tt.count), func(t *testing.T) {
			got := sb.formatTokens(tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusBar_SandboxIcon(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"read-only", "🔒"},
		{"workspace-write", "📝"},
		{"unrestricted", "🔓"},
		{"unknown", "?"},
		{"", "?"},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := sb.getSandboxIcon(tt.mode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusBar_AbbreviateDir(t *testing.T) {
	// Save current HOME for restoration
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Set a known HOME for testing
	testHome := "/home/testuser"
	os.Setenv("HOME", testHome)

	tests := []struct {
		name     string
		dir      string
		contains string // What the result should contain
	}{
		{
			name:     "empty directory",
			dir:      "",
			contains: "~",
		},
		{
			name:     "short path",
			dir:      "/project",
			contains: "/project",
		},
		{
			name:     "home directory replacement",
			dir:      testHome + "/project",
			contains: "~/project",
		},
		{
			name:     "very long path",
			dir:      "/very/long/path/that/definitely/needs/to/be/truncated/because/it/is/too/long/project",
			contains: "...",
		},
	}

	sb := NewStatusBar(100)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sb.abbreviateDir(tt.dir)
			assert.Contains(t, got, tt.contains)
		})
	}
}

func TestStatusBar_AbbreviateDir_LongPath(t *testing.T) {
	sb := NewStatusBar(100)

	// Create a path that's definitely too long (>30 chars)
	longPath := "/very/long/path/that/needs/truncation/project/subdir"
	got := sb.abbreviateDir(longPath)

	// Should contain ellipsis
	assert.Contains(t, got, "...")
	// Should be shorter than original
	assert.Less(t, len(got), len(longPath))
	// Should not be empty
	assert.NotEmpty(t, got)
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
		{ConnectionStatus(999), "unknown"},
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
		{ConnectionStatus(999), "?"},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			got := tt.status.Icon()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatusBar_UpdatesInRealTime(t *testing.T) {
	sb := NewStatusBar(120)

	// Initial state: idle
	info1 := StatusInfo{
		Model:      "llama3.1",
		Status:     StatusIdle,
		TurnTokens: 0,
	}
	sb.SetInfo(info1)
	view1 := sb.View()
	assert.Contains(t, view1, "llama3.1")
	assert.Contains(t, view1, "⏸") // Idle icon

	// Update to active with tokens
	info2 := info1
	info2.Status = StatusActive
	info2.TurnTokens = 1200
	sb.SetInfo(info2)
	view2 := sb.View()
	assert.Contains(t, view2, "⚡")    // Active icon
	assert.Contains(t, view2, "1.2K") // Formatted tokens

	// Update to error
	info3 := info2
	info3.Status = StatusError
	sb.SetInfo(info3)
	view3 := sb.View()
	assert.Contains(t, view3, "⚠") // Error icon
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

	widths := []int{30, 40, 60, 80, 100, 120}
	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			sb := NewStatusBar(width)
			sb.SetInfo(info)
			view := sb.View()

			// Should always contain model name
			assert.Contains(t, view, "llama3.1")

			// Width should not exceed terminal width
			assert.LessOrEqual(t, lipgloss.Width(view), width)
		})
	}
}

func TestStatusBar_EmptyModel(t *testing.T) {
	sb := NewStatusBar(120)
	info := StatusInfo{
		Model:  "", // Empty model
		Status: StatusIdle,
	}
	sb.SetInfo(info)

	view := sb.View()

	// Should show default "no-model"
	assert.Contains(t, view, "no-model")
}

func TestStatusBar_ZeroTokens(t *testing.T) {
	sb := NewStatusBar(120)
	info := StatusInfo{
		Model:         "llama3.1",
		TurnTokens:    0,
		SessionTokens: 0,
	}
	sb.SetInfo(info)

	view := sb.View()

	// Should handle zero tokens gracefully
	assert.Contains(t, view, "llama3.1")
	assert.Contains(t, view, "0")
}

func TestStatusBar_DefaultStyle(t *testing.T) {
	style := DefaultStatusBarStyle()

	assert.NotNil(t, style.Normal)
	assert.NotNil(t, style.Active)
	assert.NotNil(t, style.Error)
}

func TestStatusBar_ApplyStatus(t *testing.T) {
	sb := NewStatusBar(120)

	tests := []struct {
		status ConnectionStatus
		name   string
	}{
		{StatusIdle, "idle"},
		{StatusActive, "active"},
		{StatusError, "error"},
		{StatusConnecting, "connecting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb.info.Status = tt.status
			style := sb.ApplyStatus()
			assert.NotNil(t, style)
		})
	}
}

// Benchmark tests
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

func BenchmarkStatusBar_FormatTokens(b *testing.B) {
	sb := NewStatusBar(120)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sb.formatTokens(1234567)
	}
}

func BenchmarkStatusBar_AbbreviateDir(b *testing.B) {
	sb := NewStatusBar(120)
	dir := "/very/long/path/that/needs/to/be/truncated/project"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sb.abbreviateDir(dir)
	}
}
