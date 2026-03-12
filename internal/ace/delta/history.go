package delta

import (
	"sync"
	"time"
)

// History manages versioned delta records.
type History struct {
	deltas   []Delta          // Ordered list of deltas (append-only).
	byBullet map[string][]int // Index: bulletID → delta indices.
	mu       sync.RWMutex     // Thread-safe access.
}

// HistoryStats contains history statistics.
type HistoryStats struct {
	TotalDeltas       int
	UniqueBullets     int
	OldestDelta       time.Time
	NewestDelta       time.Time
	DeltasByOperation map[Operation]int
}

// NewHistory creates a new delta history.
func NewHistory() *History {
	return &History{
		deltas:   make([]Delta, 0),
		byBullet: make(map[string][]int),
	}
}

// Record adds a delta to the history.
func (h *History) Record(delta Delta) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Append to main list.
	index := len(h.deltas)
	h.deltas = append(h.deltas, delta)

	// Update bullet index.
	h.byBullet[delta.BulletID] = append(h.byBullet[delta.BulletID], index)
}

// GetByBullet returns all deltas for a bullet.
func (h *History) GetByBullet(bulletID string) []Delta {
	h.mu.RLock()
	defer h.mu.RUnlock()

	indices, exists := h.byBullet[bulletID]
	if !exists {
		return nil
	}

	result := make([]Delta, 0, len(indices))
	for _, idx := range indices {
		result = append(result, h.deltas[idx])
	}

	return result
}

// GetRecent returns the N most recent deltas.
func (h *History) GetRecent(count int) []Delta {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if count <= 0 {
		return nil
	}

	total := len(h.deltas)
	if count > total {
		count = total
	}

	start := total - count
	result := make([]Delta, count)
	copy(result, h.deltas[start:])

	return result
}

// GetSince returns all deltas since a timestamp.
func (h *History) GetSince(since time.Time) []Delta {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]Delta, 0)

	for i := range h.deltas {
		if h.deltas[i].CreatedAt.After(since) || h.deltas[i].CreatedAt.Equal(since) {
			result = append(result, h.deltas[i])
		}
	}

	return result
}

// Stats returns history statistics.
func (h *History) Stats() HistoryStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := HistoryStats{
		TotalDeltas:       len(h.deltas),
		UniqueBullets:     len(h.byBullet),
		DeltasByOperation: make(map[Operation]int),
	}

	if stats.TotalDeltas == 0 {
		return stats
	}

	stats.OldestDelta = h.deltas[0].CreatedAt
	stats.NewestDelta = h.deltas[len(h.deltas)-1].CreatedAt

	for i := range h.deltas {
		stats.DeltasByOperation[h.deltas[i].Operation]++
	}

	return stats
}

// Clear removes all history (use with caution).
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.deltas = make([]Delta, 0)
	h.byBullet = make(map[string][]int)
}

// Len returns the total number of deltas in history.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.deltas)
}
