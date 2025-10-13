package status

import (
	"fmt"
	"sync"
	"time"
)

// Metrics represents the current status metrics.
type Metrics struct {
	// Conversation metrics
	TurnCount  int
	TokenCount int64
	MaxTokens  int64
	TokenUsage float64 // Percentage (0-100)

	// Performance metrics
	ResponseTime time.Duration
	TokensPerSec float64

	// Connection metrics
	Provider  string
	Model     string
	Connected bool

	// Timestamps
	LastUpdate   time.Time
	SessionStart time.Time
}

// Status represents the current status information.
type Status struct {
	Text    string // Human-readable status text
	Metrics Metrics
}

// Manager handles status data and updates.
// This is a pure data container with no rendering responsibilities.
type Manager struct {
	mu      sync.RWMutex
	status  Status
	enabled bool
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

// Enable/Disable controls whether the status manager is active.
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
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

// FormatCompact formats the status as a compact string suitable for display in the prompt line.
// Returns an empty string if the manager is disabled or there's no meaningful data.
func (m *Manager) FormatCompact() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return ""
	}

	// If there's explicit status text, return it
	if m.status.Text != "" {
		return m.status.Text
	}

	// Otherwise, format metrics compactly
	// Priority: Show most useful info in limited space
	var parts []string

	// Show provider/model if set
	if m.status.Metrics.Provider != "" {
		parts = append(parts, m.status.Metrics.Provider)
	}

	// Show turn count
	if m.status.Metrics.TurnCount > 0 {
		parts = append(parts, "T:"+intToStr(m.status.Metrics.TurnCount))
	}

	// Show tokens if non-zero
	if m.status.Metrics.TokenCount > 0 {
		parts = append(parts, "Tok:"+int64ToStr(m.status.Metrics.TokenCount))
	}

	// Show TPS if non-zero
	if m.status.Metrics.TokensPerSec > 0 {
		parts = append(parts, "TPS:"+floatToStr(m.status.Metrics.TokensPerSec))
	}

	if len(parts) == 0 {
		return ""
	}

	// Join with " | " separator
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " | "
		}
		result += part
	}
	return result
}

// Helper functions for compact formatting
func intToStr(i int) string {
	return fmt.Sprintf("%d", i)
}

func int64ToStr(i int64) string {
	return fmt.Sprintf("%d", i)
}

func floatToStr(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
