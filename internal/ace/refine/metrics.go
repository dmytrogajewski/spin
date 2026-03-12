package refine

import (
	"context"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

const (
	defaultRefineMaxBullets = 1000
	defaultRefineMaxTokens  = 100000
	defaultRefineMinUtility = 0.1
	metricHistoryCapacity   = 100
	tokensPerBulletEstimate = 50
	minHistoryForGrowthRate = 2
)

// GrowthMetrics tracks playbook growth statistics.
type GrowthMetrics struct {
	BulletCount     int       // Total bullets.
	EstimatedTokens int       // Approximate token count.
	AvgUtilityScore float64   // Average bullet utility.
	LastRefinement  time.Time // When last refined.
	GrowthRate      float64   // Bullets per hour.
}

// GrowthThresholds defines when to trigger refinement.
type GrowthThresholds struct {
	MaxBullets    int           // Trigger when bullet count exceeds.
	MaxTokens     int           // Trigger when estimated tokens exceed.
	MinUtility    float64       // Trigger when avg utility drops below.
	CheckInterval time.Duration // How often to check metrics.
}

// GrowthMonitor tracks playbook growth and triggers refinement.
type GrowthMonitor struct {
	playbook      *playbook.Playbook
	thresholds    GrowthThresholds
	metrics       GrowthMetrics
	lastCheck     time.Time
	bulletHistory []int       // Historical bullet counts for growth rate.
	timeHistory   []time.Time // Timestamps for growth rate.
	mu            sync.RWMutex
}

// DefaultGrowthThresholds returns sensible default thresholds.
func DefaultGrowthThresholds() GrowthThresholds {
	return GrowthThresholds{
		MaxBullets:    defaultRefineMaxBullets,
		MaxTokens:     defaultRefineMaxTokens,
		MinUtility:    defaultRefineMinUtility,
		CheckInterval: 1 * time.Minute,
	}
}

// NewGrowthMonitor creates a new growth monitor.
func NewGrowthMonitor(pb *playbook.Playbook, thresholds GrowthThresholds) *GrowthMonitor {
	return &GrowthMonitor{
		playbook:      pb,
		thresholds:    thresholds,
		lastCheck:     time.Now(),
		bulletHistory: make([]int, 0, metricHistoryCapacity),
		timeHistory:   make([]time.Time, 0, metricHistoryCapacity),
	}
}

// CheckGrowth evaluates current playbook state and returns metrics.
func (m *GrowthMonitor) CheckGrowth(_ context.Context) (GrowthMetrics, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.playbook.Stats()
	now := time.Now()

	// Update history for growth rate calculation.
	m.bulletHistory = append(m.bulletHistory, stats.TotalBullets)
	m.timeHistory = append(m.timeHistory, now)

	// Keep only last 100 data points.
	if len(m.bulletHistory) > metricHistoryCapacity {
		m.bulletHistory = m.bulletHistory[1:]
		m.timeHistory = m.timeHistory[1:]
	}

	// Calculate metrics.
	metrics := GrowthMetrics{
		BulletCount:     stats.TotalBullets,
		EstimatedTokens: m.estimateTokens(stats),
		AvgUtilityScore: stats.AvgScore,
		LastRefinement:  m.metrics.LastRefinement,
		GrowthRate:      m.calculateGrowthRate(),
	}

	m.metrics = metrics
	m.lastCheck = now

	// Check if refinement needed.
	needsRefinement := m.checkRefineNeeded(metrics)

	return metrics, needsRefinement
}

// GetMetrics returns current metrics without checking.
func (m *GrowthMonitor) GetMetrics() GrowthMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.metrics
}

// ShouldRefine checks if refinement is needed based on current metrics.
func (m *GrowthMonitor) ShouldRefine() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.checkRefineNeeded(m.metrics)
}

// MarkRefinement records that refinement occurred.
func (m *GrowthMonitor) MarkRefinement() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.LastRefinement = time.Now()
}

// checkRefineNeeded determines if any threshold is breached (internal, no lock).
func (m *GrowthMonitor) checkRefineNeeded(metrics GrowthMetrics) bool {
	// Don't refine empty playbook.
	if metrics.BulletCount == 0 {
		return false
	}

	if m.thresholds.MaxBullets > 0 && metrics.BulletCount >= m.thresholds.MaxBullets {
		return true
	}

	if m.thresholds.MaxTokens > 0 && metrics.EstimatedTokens >= m.thresholds.MaxTokens {
		return true
	}

	if m.thresholds.MinUtility > 0 && metrics.AvgUtilityScore < m.thresholds.MinUtility {
		return true
	}

	return false
}

// estimateTokens provides rough token count estimate.
func (m *GrowthMonitor) estimateTokens(stats playbook.Stats) int {
	// Rough estimate: average bullet is ~50 tokens.
	return stats.TotalBullets * tokensPerBulletEstimate
}

// calculateGrowthRate computes bullets per hour based on history.
func (m *GrowthMonitor) calculateGrowthRate() float64 {
	if len(m.bulletHistory) < minHistoryForGrowthRate {
		return 0.0
	}

	// Calculate rate over last hour.
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// Find first data point within last hour.
	startIdx := -1

	for i := len(m.timeHistory) - 1; i >= 0; i-- {
		if m.timeHistory[i].Before(oneHourAgo) {
			startIdx = i

			break
		}
	}

	if startIdx < 0 || startIdx >= len(m.bulletHistory)-1 {
		// Not enough history, use all available.
		startIdx = 0
	}

	bulletDiff := m.bulletHistory[len(m.bulletHistory)-1] - m.bulletHistory[startIdx]
	timeDiff := m.timeHistory[len(m.timeHistory)-1].Sub(m.timeHistory[startIdx])

	if timeDiff.Hours() == 0 {
		return 0.0
	}

	return float64(bulletDiff) / timeDiff.Hours()
}
