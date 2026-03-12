package status

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Metrics represents the current status metrics.
type Metrics struct {
	// Conversation metrics.
	TurnCount  int
	TokenCount int64
	MaxTokens  int64
	TokenUsage float64 // Percentage (0-100).

	// Performance metrics.
	ResponseTime time.Duration
	TokensPerSec float64

	// Connection metrics.
	Provider  string
	Model     string
	Connected bool

	// Agent state.
	AgentState     string // Current agent activity (e.g., "Calling tools", "Thinking").
	TaskMode       string // Task mode: regular, review, compact, planning.
	ConversationID string // Session/conversation identifier.

	// Timestamps.
	LastUpdate   time.Time
	SessionStart time.Time
}

// Status represents the current status information.
type Status struct {
	Text    string // Human-readable status text.
	Metrics Metrics
}

// Manager handles status data and updates.
// This is a pure data container with no rendering responsibilities.
type Manager struct {
	mu      sync.RWMutex
	status  Status
	enabled bool
	spinner *ActivitySpinner
}

// NewManager creates a new status manager.
func NewManager() *Manager {
	return &Manager{
		status: Status{
			Metrics: Metrics{
				SessionStart: time.Now(),
				LastUpdate:   time.Now(),
			},
		},
		enabled: true,
		spinner: NewActivitySpinner(SpinnerDots),
	}
}

// GetStatus returns the current status.
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status
}

// GetMetrics returns the current metrics.
func (m *Manager) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status.Metrics
}

// SetStatus sets the status text.
func (m *Manager) SetStatus(text string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.Text = text
	m.status.Metrics.LastUpdate = time.Now()
}

// UpdateMetrics updates the metrics.
func (m *Manager) UpdateMetrics(updater func(*Metrics)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	updater(&m.status.Metrics)
	m.status.Metrics.LastUpdate = time.Now()
}

// SetProvider sets the LLM provider information.
func (m *Manager) SetProvider(provider, model string) {
	m.UpdateMetrics(func(m *Metrics) {
		m.Provider = provider
		m.Model = model
		m.Connected = true
	})
}

// IncrementTurn increments the turn count.
func (m *Manager) IncrementTurn() {
	m.UpdateMetrics(func(m *Metrics) {
		m.TurnCount++
	})
}

// AddTokens adds tokens to the count and updates usage percentage.
func (m *Manager) AddTokens(prompt, completion int64) {
	m.UpdateMetrics(func(m *Metrics) {
		m.TokenCount += prompt + completion
		if m.MaxTokens > 0 {
			m.TokenUsage = float64(m.TokenCount) / float64(m.MaxTokens) * 100
		}
	})
}

// SetResponseTime sets the response time and calculates tokens per second.
func (m *Manager) SetResponseTime(duration time.Duration, tokens int64) {
	m.UpdateMetrics(func(m *Metrics) {
		m.ResponseTime = duration
		if duration > 0 {
			m.TokensPerSec = float64(tokens) / duration.Seconds()
		}
	})
}

// SetMaxTokens sets the maximum token limit.
func (m *Manager) SetMaxTokens(maxTokens int64) {
	m.UpdateMetrics(func(m *Metrics) {
		m.MaxTokens = maxTokens
		if m.MaxTokens > 0 {
			m.TokenUsage = float64(m.TokenCount) / float64(m.MaxTokens) * 100
		}
	})
}

// SetConnected sets the connection status.
func (m *Manager) SetConnected(connected bool) {
	m.UpdateMetrics(func(m *Metrics) {
		m.Connected = connected
	})
}

// SetAgentState sets the current agent activity state.
// Also updates the spinner animation based on the state.
func (m *Manager) SetAgentState(state string) {
	m.mu.Lock()
	m.status.Metrics.AgentState = state
	m.status.Metrics.LastUpdate = time.Now()
	spinner := m.spinner
	m.mu.Unlock()

	// Update spinner state outside the lock to avoid deadlock.
	if spinner != nil {
		spinner.UpdateState(context.Background(), state)
	}
}

// SetAgentStateWithContext sets the agent state with a context for spinner control.
func (m *Manager) SetAgentStateWithContext(ctx context.Context, state string) {
	m.mu.Lock()
	m.status.Metrics.AgentState = state
	m.status.Metrics.LastUpdate = time.Now()
	spinner := m.spinner
	m.mu.Unlock()

	// Update spinner state outside the lock to avoid deadlock.
	if spinner != nil {
		spinner.UpdateState(ctx, state)
	}
}

// SetTaskMode sets the current task mode.
func (m *Manager) SetTaskMode(mode string) {
	m.UpdateMetrics(func(m *Metrics) {
		m.TaskMode = mode
	})
}

// SetConversationID sets the conversation/session identifier.
func (m *Manager) SetConversationID(id string) {
	m.UpdateMetrics(func(m *Metrics) {
		m.ConversationID = id
	})
}

// CalculateTPS calculates tokens per second from token count and duration.
func (m *Manager) CalculateTPS(tokens int64, duration time.Duration) {
	m.UpdateMetrics(func(m *Metrics) {
		if duration > 0 {
			m.TokensPerSec = float64(tokens) / duration.Seconds()
		}
	})
}

// Enable activates the status manager.
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = true
}

// Disable implements the Disable operation.
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enabled = false
}

// IsEnabled implements the IsEnabled operation.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.enabled
}

// GetSpinner returns the activity spinner.
func (m *Manager) GetSpinner() *ActivitySpinner {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.spinner
}

// SpinnerFrame returns the current spinner animation frame.
// Returns empty string if spinner is not running.
func (m *Manager) SpinnerFrame() string {
	m.mu.RLock()
	spinner := m.spinner
	m.mu.RUnlock()

	if spinner == nil {
		return ""
	}

	return spinner.Frame()
}

// SetSpinnerCallback sets the callback function for spinner updates.
// This is called on each animation frame to trigger UI refresh.
func (m *Manager) SetSpinnerCallback(callback func()) {
	m.mu.Lock()
	spinner := m.spinner
	m.mu.Unlock()

	if spinner != nil {
		spinner.SetUpdateCallback(callback)
	}
}

// StopSpinner stops the spinner animation.
func (m *Manager) StopSpinner() {
	m.mu.Lock()
	spinner := m.spinner
	m.mu.Unlock()

	if spinner != nil {
		spinner.Stop()
	}
}

// Reset resets all metrics to initial state.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status = Status{
		Metrics: Metrics{
			SessionStart: time.Now(),
			LastUpdate:   time.Now(),
		},
	}
}

// FormatCompact formats status for narrow terminals (<60 columns).
// Shows: activity indicator, context%, and agent state only.
// NOTE: This replaces the old FormatCompact() implementation.
func (m *Manager) FormatCompact(_ int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return ""
	}

	// If there's explicit status text, return it (legacy behavior).
	if m.status.Text != "" {
		return m.status.Text
	}

	parts := []string{}

	// Activity indicator with spinner.
	spinnerFrame := ""
	if m.spinner != nil && m.spinner.IsRunning() {
		spinnerFrame = m.spinner.Frame()
	}

	if spinnerFrame != "" {
		parts = append(parts, "["+spinnerFrame+"]")
	} else if m.status.Metrics.Connected {
		parts = append(parts, "[●]")
	} else {
		parts = append(parts, "[○]")
	}

	// Context percentage (if available).
	if m.status.Metrics.MaxTokens > 0 {
		pct := fmt.Sprintf("%.0f%%", m.status.Metrics.TokenUsage)
		parts = append(parts, pct)
	}

	// Agent state.
	state := m.status.Metrics.AgentState
	if state == "" {
		state = "Ready"
	}
	// Truncate state to fit narrow terminal.
	if len(state) > 15 {
		state = state[:12] + "..."
	}

	parts = append(parts, state)

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}
