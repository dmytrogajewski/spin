package status

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatAdaptive selects the appropriate format based on terminal width.
func (m *Manager) FormatAdaptive(width int) string {
	if width < 60 {
		return m.FormatCompact(width)
	} else if width < 100 {
		return m.FormatMedium(width)
	}

	return m.FormatFull(width)
}

// FormatMedium formats status for medium-width terminals (60-100 columns).
// Shows: activity, context%, state, provider/model, TPS.
func (m *Manager) FormatMedium(width int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return ""
	}

	parts := []string{}

	// Activity indicator with spinner.
	spinnerFrame := ""
	if m.spinner != nil && m.spinner.IsRunning() {
		spinnerFrame = m.spinner.Frame()
	}

	activity := activityIndicatorWithSpinner(m.status.Metrics.Connected, spinnerFrame)
	parts = append(parts, activity)

	// Context percentage.
	if m.status.Metrics.MaxTokens > 0 {
		pct := formatPercentage(m.status.Metrics.TokenUsage)
		parts = append(parts, pct)
	}

	// Agent state.
	state := m.status.Metrics.AgentState
	if state == "" {
		state = "Ready"
	}

	parts = append(parts, truncate(state, 15))

	// Provider/model (truncated).
	if m.status.Metrics.Provider != "" {
		provider := fmt.Sprintf("%s/%s",
			m.status.Metrics.Provider,
			truncate(m.status.Metrics.Model, 12))
		parts = append(parts, provider)
	}

	// TPS (only if actively generating and non-zero).
	if m.status.Metrics.TokensPerSec > 1.0 { // Use 1.0 threshold to filter out noise.
		tps := fmt.Sprintf("%.0ftok/s", m.status.Metrics.TokensPerSec)
		parts = append(parts, tps)
	}

	return strings.Join(parts, "  ")
}

// FormatFull formats status for wide terminals (≥100 columns).
// Shows all fields including context absolute values, conversation ID, and hotkeys.
func (m *Manager) FormatFull(width int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return ""
	}

	parts := []string{}

	// Activity indicator with spinner.
	spinnerFrame := ""
	if m.spinner != nil && m.spinner.IsRunning() {
		spinnerFrame = m.spinner.Frame()
	}

	activity := activityIndicatorWithSpinner(m.status.Metrics.Connected, spinnerFrame)
	parts = append(parts, activity)

	// Context usage (percentage and absolute).
	if m.status.Metrics.MaxTokens > 0 {
		pct := formatPercentage(m.status.Metrics.TokenUsage)
		abs := fmt.Sprintf("(%s/%s)",
			humanizeNumber(m.status.Metrics.TokenCount),
			humanizeNumber(m.status.Metrics.MaxTokens))
		parts = append(parts, pct, abs)
	}

	// Agent state.
	state := m.status.Metrics.AgentState
	if state == "" {
		state = "Ready"
	}

	parts = append(parts, truncate(state, 20))

	// Task mode (if not default).
	if m.status.Metrics.TaskMode != "" && m.status.Metrics.TaskMode != "regular" {
		parts = append(parts, capitalize(m.status.Metrics.TaskMode))
	}

	// Provider/model.
	if m.status.Metrics.Provider != "" {
		provider := fmt.Sprintf("%s/%s",
			m.status.Metrics.Provider,
			truncate(m.status.Metrics.Model, 20))
		parts = append(parts, provider)
	}

	// TPS (only if actively generating and non-zero).
	if m.status.Metrics.TokensPerSec > 1.0 { // Use 1.0 threshold to filter out noise.
		tps := fmt.Sprintf("%.0f tok/s", m.status.Metrics.TokensPerSec)
		parts = append(parts, tps)
	}

	// Conversation ID (shortened to first 6 chars).
	if m.status.Metrics.ConversationID != "" {
		shortID := m.status.Metrics.ConversationID
		if len(shortID) > 6 {
			shortID = shortID[:6]
		}

		convID := "conv:" + shortID
		parts = append(parts, convID)
	}

	// Hotkeys (only on very wide terminals, and only if explicitly enabled)
	// Disabled as it adds clutter
	// if width >= 140 {
	// 	parts = append(parts, "?:help ^C:quit")
	// }.

	return strings.Join(parts, "  ")
}

// Helper functions.

// activityIndicator returns the activity indicator based on connection status.
func activityIndicator(connected bool) string {
	if connected {
		return "[●]" // Active (colored green in render phase).
	}

	return "[○]" // Idle (colored gray in render phase).
}

// activityIndicatorWithSpinner returns the activity indicator with optional spinner.
// When spinner is active, shows the spinning animation instead of static indicator.
func activityIndicatorWithSpinner(connected bool, spinnerFrame string) string {
	if spinnerFrame != "" {
		return "[" + spinnerFrame + "]"
	}

	return activityIndicator(connected)
}

// formatPercentage formats a percentage value.
func formatPercentage(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

// humanizeNumber formats large numbers with K/M suffixes.
func humanizeNumber(n int64) string {
	if n < 1000 {
		return strconv.FormatInt(n, 10)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}

	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// truncate truncates a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen < 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
