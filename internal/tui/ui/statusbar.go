package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ConnectionStatus represents the current activity state of the TUI.
type ConnectionStatus int

const (
	// StatusIdle indicates the TUI is waiting for user input.
	StatusIdle ConnectionStatus = iota
	// StatusActive indicates the TUI is actively processing (streaming response).
	StatusActive
	// StatusConnecting indicates the TUI is connecting to the LLM provider.
	StatusConnecting
	// StatusError indicates an error has occurred.
	StatusError
)

// String returns the string representation of the connection status.
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

// Icon returns the icon for the connection status.
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

// StatusInfo contains all information displayed in the status bar.
type StatusInfo struct {
	// Model is the name of the current LLM model (e.g., "llama3.1", "gpt-4o").
	Model string
	// Provider is the name of the LLM provider (e.g., "ollama", "openai").
	Provider string
	// SandboxMode is the current sandbox mode ("read-only", "workspace-write", "unrestricted").
	SandboxMode string
	// WorkingDir is the current working directory.
	WorkingDir string
	// Status is the current connection/activity status.
	Status ConnectionStatus
	// TurnTokens is the number of tokens used in the current turn.
	TurnTokens int
	// SessionTokens is the total number of tokens used in the session.
	SessionTokens int
}

// StatusBar represents the status bar component.
type StatusBar struct {
	info         StatusInfo
	width        int
	style        StatusBarStyle
	errorMsg     *ErrorDisplay // Current error being displayed (nil if none)
	errorShownAt time.Time     // When the error was shown (for auto-dismiss)
}

// StatusBarStyle defines the visual styling for the status bar.
type StatusBarStyle struct {
	Normal lipgloss.Style
	Active lipgloss.Style
	Error  lipgloss.Style
}

// NewStatusBar creates a new status bar with the given width.
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

// View renders the status bar to a string.
func (s StatusBar) View() string {
	return s.render()
}

// render creates the status bar content.
func (s StatusBar) render() string {
	// If error is present, render error instead of normal content
	if s.HasError() {
		return s.renderError()
	}

	// Format each section
	left := s.renderLeft()
	middle := s.renderMiddle()
	right := s.renderRight()

	// Layout based on available width
	return s.layout(left, middle, right)
}

// layout arranges the status bar sections based on available width.
func (s StatusBar) layout(left, middle, right string) string {
	separator := " | "

	// Very narrow: compact mode
	if s.width < 40 {
		return s.renderCompact()
	}

	// Calculate required width
	requiredWidth := lipgloss.Width(left) + lipgloss.Width(middle) + lipgloss.Width(right) + len(separator)*2

	// Medium width: abbreviate middle section
	if requiredWidth > s.width {
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

	// Apply styling based on status and pad to full width
	style := s.ApplyStatus()
	return style.Width(s.width).Render(content)
}

// renderLeft renders the left section (model name).
func (s StatusBar) renderLeft() string {
	model := s.info.Model
	if model == "" {
		model = "no-model"
	}
	return model
}

// renderMiddle renders the middle section (sandbox + working directory).
func (s StatusBar) renderMiddle() string {
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)
	workDir := s.abbreviateDir(s.info.WorkingDir)

	return fmt.Sprintf("%s %s | %s", sandboxIcon, s.info.SandboxMode, workDir)
}

// renderMiddleAbbreviated renders an abbreviated middle section.
func (s StatusBar) renderMiddleAbbreviated() string {
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)
	workDir := s.abbreviateDir(s.info.WorkingDir)

	// Just icon + dir (omit mode text)
	return fmt.Sprintf("%s | %s", sandboxIcon, workDir)
}

// renderRight renders the right section (status + token usage).
func (s StatusBar) renderRight() string {
	statusIcon := s.info.Status.Icon()

	// Format token counts
	turnTokens := s.formatTokens(s.info.TurnTokens)
	sessionTokens := s.formatTokens(s.info.SessionTokens)

	if s.width > 100 {
		// Wide: full text with labels
		return fmt.Sprintf("%s %s | %s / %s tokens", statusIcon, s.info.Status.String(), turnTokens, sessionTokens)
	}
	// Medium: compact format
	return fmt.Sprintf("%s | %s/%s", statusIcon, turnTokens, sessionTokens)
}

// renderCompact renders a compact status bar for very narrow terminals.
func (s StatusBar) renderCompact() string {
	sandboxIcon := s.getSandboxIcon(s.info.SandboxMode)
	statusIcon := s.info.Status.Icon()
	turnTokens := s.formatTokens(s.info.TurnTokens)

	content := fmt.Sprintf("%s | %s | %s | %s", s.info.Model, sandboxIcon, statusIcon, turnTokens)

	style := s.ApplyStatus()
	return style.Width(s.width).Render(content)
}

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
		// Show start and end with ellipsis in middle
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

// ApplyStatus applies color styling based on the current connection status.
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

// SetTokens updates the token counts.
func (s *StatusBar) SetTokens(turnTokens, sessionTokens int) {
	s.info.TurnTokens = turnTokens
	s.info.SessionTokens = sessionTokens
}

// TurnTokens returns the current turn token count (for testing).
func (s StatusBar) TurnTokens() int {
	return s.info.TurnTokens
}

// SessionTokens returns the total session token count (for testing).
func (s StatusBar) SessionTokens() int {
	return s.info.SessionTokens
}

// ShowError displays a transient error in the status bar.
// The error will override the normal status bar content until dismissed or expired.
func (s *StatusBar) ShowError(err ErrorDisplay) {
	s.errorMsg = &err
	s.errorShownAt = time.Now()
}

// ClearError clears the current error message.
func (s *StatusBar) ClearError() {
	s.errorMsg = nil
}

// HasError returns true if an error is currently displayed.
func (s *StatusBar) HasError() bool {
	return s.errorMsg != nil
}

// ErrorExpired returns true if the current error has passed its auto-dismiss duration.
// Returns false if no error is set or if the error has no auto-dismiss.
func (s *StatusBar) ErrorExpired() bool {
	if s.errorMsg == nil {
		return false
	}
	if s.errorMsg.AutoDismiss == 0 {
		return false
	}
	elapsed := time.Since(s.errorShownAt)
	return elapsed >= time.Duration(s.errorMsg.AutoDismiss)*time.Second
}

// TimeUntilErrorDismiss returns the remaining time until auto-dismiss.
// Returns 0 if no error, no auto-dismiss, or already expired.
func (s *StatusBar) TimeUntilErrorDismiss() time.Duration {
	if s.errorMsg == nil || s.errorMsg.AutoDismiss == 0 {
		return 0
	}
	elapsed := time.Since(s.errorShownAt)
	remaining := time.Duration(s.errorMsg.AutoDismiss)*time.Second - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// renderError renders the error message in the status bar.
func (s StatusBar) renderError() string {
	if s.errorMsg == nil {
		return ""
	}

	// Get error icon based on severity
	icon := s.getErrorIcon(s.errorMsg.Severity)

	// Build error message
	var content string
	if s.errorMsg.AutoDismiss > 0 && !s.ErrorExpired() {
		// Show countdown for auto-dismiss
		remaining := s.TimeUntilErrorDismiss()
		seconds := int(remaining.Seconds()) + 1 // Round up
		content = fmt.Sprintf("%s %s (dismiss in %ds)", icon, s.errorMsg.Message, seconds)
	} else {
		// No auto-dismiss or expired
		content = fmt.Sprintf("%s %s", icon, s.errorMsg.Message)
	}

	// Apply error styling based on severity
	var style lipgloss.Style
	switch s.errorMsg.Severity {
	case 0: // Info
		style = s.style.Normal
	case 1: // Warning
		style = s.style.Normal.Foreground(lipgloss.Color("11")) // Yellow
	case 2: // Error
		style = s.style.Error
	case 3: // Critical
		style = s.style.Error.Bold(true)
	default:
		style = s.style.Error
	}

	return style.Width(s.width).Render(content)
}

// getErrorIcon returns the emoji icon for error severity.
func (s StatusBar) getErrorIcon(severity int) string {
	switch severity {
	case 0:
		return "ℹ️" // Info
	case 1:
		return "⚠️" // Warning
	case 2:
		return "❌" // Error
	case 3:
		return "🔥" // Critical
	default:
		return "❓"
	}
}
