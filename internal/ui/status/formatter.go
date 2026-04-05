package status

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

// DetailLevel controls how much information the status bar displays.
type DetailLevel int

const (
	// DetailCompact shows minimal info: activity, context%, state.
	DetailCompact DetailLevel = iota
	// DetailMedium adds provider/model and TPS.
	DetailMedium
	// DetailFull adds absolute token counts, task mode, conversation ID.
	DetailFull
)

// PercentMultiplier converts a ratio to a percentage value.
const PercentMultiplier = 100

// Terminal width breakpoints for adaptive formatting.
const (
	compactWidthThreshold = 60
	mediumWidthThreshold  = 100

	// Truncation limits for status bar fields.
	mediumStateTruncate = 15
	mediumModelTruncate = 12
	fullStateTruncate   = 20
	fullModelTruncate   = 20
	convIDShortLength   = 6

	// Number formatting thresholds.
	kiloThreshold = 1000
	megaThreshold = 1000000

	// StateReady is the default agent state.
	StateReady = "Ready"

	// tpsNoiseThreshold filters out negligible TPS values.
	tpsNoiseThreshold = 1.0
)

// FormatAdaptive selects the appropriate detail level based on terminal width.
func (m *Manager) FormatAdaptive(width int) string {
	if width < compactWidthThreshold {
		return m.Format(DetailCompact, width)
	} else if width < mediumWidthThreshold {
		return m.Format(DetailMedium, width)
	}

	return m.Format(DetailFull, width)
}

// FormatCompact formats status for narrow terminals (<60 columns).
// Delegates to Format with DetailCompact.
func (m *Manager) FormatCompact(width int) string {
	return m.Format(DetailCompact, width)
}

// FormatMedium formats status for medium-width terminals (60-100 columns).
// Delegates to Format with DetailMedium.
func (m *Manager) FormatMedium(width int) string {
	return m.Format(DetailMedium, width)
}

// FormatFull formats status for wide terminals (>=100 columns).
// Delegates to Format with DetailFull.
func (m *Manager) FormatFull(width int) string {
	return m.Format(DetailFull, width)
}

// Format produces a status string at the requested detail level.
//
//   - DetailCompact: activity, context%, state
//   - DetailMedium: + provider/model, TPS
//   - DetailFull: + absolute token counts, task mode, conversation ID
func (m *Manager) Format(level DetailLevel, _ int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return ""
	}

	// Compact-only: honor explicit status text (legacy behavior).
	if level == DetailCompact && m.status.Text != "" {
		return m.status.Text
	}

	spinnerFrame := ""
	if m.spinner != nil && m.spinner.IsRunning() {
		spinnerFrame = m.spinner.Frame()
	}

	return FormatMetrics(&m.status.Metrics, level, spinnerFrame)
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
	if n < kiloThreshold {
		return strconv.FormatInt(n, 10)
	} else if n < megaThreshold {
		return fmt.Sprintf("%.1fK", float64(n)/kiloThreshold)
	}

	return fmt.Sprintf("%.1fM", float64(n)/megaThreshold)
}

// FormatMetrics formats raw metrics at the given detail level without requiring
// a Manager. This is the shared formatting core used by both Manager.Format
// and Renderer.buildMetricsLine.
func FormatMetrics(metrics *Metrics, level DetailLevel, spinnerFrame string) string {
	parts := []string{}

	// Activity indicator.
	parts = append(parts, activityIndicatorWithSpinner(metrics.Connected, spinnerFrame))

	// Context usage.
	parts = appendContextParts(parts, metrics, level)

	// Agent state.
	parts = appendStateParts(parts, metrics, level)

	// Provider/model and TPS (medium and full).
	parts = appendProviderAndTPS(parts, metrics, level)

	// Conversation ID (full only).
	parts = appendConversationID(parts, metrics, level)

	sep := "  "
	if level == DetailCompact {
		sep = " "
	}

	return strings.Join(parts, sep)
}

// appendContextParts appends context usage percentage (and absolute counts at full level).
func appendContextParts(parts []string, metrics *Metrics, level DetailLevel) []string {
	if metrics.MaxTokens <= 0 {
		return parts
	}

	parts = append(parts, formatPercentage(metrics.TokenUsage))

	if level >= DetailFull {
		abs := fmt.Sprintf("(%s/%s)",
			humanizeNumber(metrics.TokenCount),
			humanizeNumber(metrics.MaxTokens))
		parts = append(parts, abs)
	}

	return parts
}

// appendStateParts appends agent state and task mode.
func appendStateParts(parts []string, metrics *Metrics, level DetailLevel) []string {
	stateTrunc := mediumStateTruncate
	if level >= DetailFull {
		stateTrunc = fullStateTruncate
	}

	state := metrics.AgentState
	if state == "" {
		state = StateReady
	}

	parts = append(parts, textwidth.TruncateRight(state, stateTrunc))

	// Task mode (full only, non-default).
	if level >= DetailFull && metrics.TaskMode != "" && metrics.TaskMode != "regular" {
		parts = append(parts, capitalize(metrics.TaskMode))
	}

	return parts
}

// appendProviderAndTPS appends provider/model and TPS for medium and full levels.
func appendProviderAndTPS(parts []string, metrics *Metrics, level DetailLevel) []string {
	if level < DetailMedium {
		return parts
	}

	modelTrunc := mediumModelTruncate
	if level >= DetailFull {
		modelTrunc = fullModelTruncate
	}

	if metrics.Provider != "" {
		provider := fmt.Sprintf("%s/%s",
			metrics.Provider,
			textwidth.TruncateRight(metrics.Model, modelTrunc))
		parts = append(parts, provider)
	}

	if metrics.TokensPerSec > tpsNoiseThreshold {
		tpsFmt := "%.0ftok/s"
		if level >= DetailFull {
			tpsFmt = "%.0f tok/s"
		}

		parts = append(parts, fmt.Sprintf(tpsFmt, metrics.TokensPerSec))
	}

	return parts
}

// appendConversationID appends shortened conversation ID for full level.
func appendConversationID(parts []string, metrics *Metrics, level DetailLevel) []string {
	if level < DetailFull || metrics.ConversationID == "" {
		return parts
	}

	shortID := metrics.ConversationID
	if len(shortID) > convIDShortLength {
		shortID = shortID[:convIDShortLength]
	}

	return append(parts, "conv:"+shortID)
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}
