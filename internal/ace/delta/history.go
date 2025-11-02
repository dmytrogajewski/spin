package delta

import (
	"sync"
	"time"
)

// DeltaHistory manages versioned delta records.
type DeltaHistory struct {
	deltas   []Delta          // Ordered list of deltas (append-only)
	byBullet map[string][]int // Index: bulletID → delta indices
	mu       sync.RWMutex     // Thread-safe access
}

// DeltaHistoryStats contains history statistics.
type DeltaHistoryStats struct {
	TotalDeltas       int
	UniqueBullets     int
	OldestDelta       time.Time
	NewestDelta       time.Time
	DeltasByOperation map[DeltaOperation]int
}

// NewDeltaHistory creates a new delta history.
func NewDeltaHistory() *DeltaHistory {
	return &DeltaHistory{
		deltas:   make([]Delta, 0),
		byBullet: make(map[string][]int),
	}
}

// Record adds a delta to the history.
func (h *DeltaHistory) Record(delta Delta) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Append to main list
	index := len(h.deltas)
	h.deltas = append(h.deltas, delta)

	// Update bullet index
	h.byBullet[delta.BulletID] = append(h.byBullet[delta.BulletID], index)
}

// GetByBullet returns all deltas for a bullet.
func (h *DeltaHistory) GetByBullet(bulletID string) []Delta {
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
func (h *DeltaHistory) GetRecent(count int) []Delta {
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
func (h *DeltaHistory) GetSince(since time.Time) []Delta {
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
func (h *DeltaHistory) Stats() DeltaHistoryStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := DeltaHistoryStats{
		TotalDeltas:       len(h.deltas),
		UniqueBullets:     len(h.byBullet),
		DeltasByOperation: make(map[DeltaOperation]int),
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
func (h *DeltaHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.deltas = make([]Delta, 0)
	h.byBullet = make(map[string][]int)
}

// Len returns the total number of deltas in history.
func (h *DeltaHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.deltas)
}
