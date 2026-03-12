package refine

import (
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// ArchiveReason explains why a bullet was archived.
type ArchiveReason string

const (
	// ReasonLowUtility indicates bullet was removed due to low utility score.
	ReasonLowUtility ArchiveReason = "low_utility"
	// ReasonMerged indicates bullet was merged into another bullet.
	ReasonMerged ArchiveReason = "merged"
	// ReasonManual indicates bullet was manually archived.
	ReasonManual ArchiveReason = "manual"
	// ReasonSuperseded indicates bullet was superseded by a better version.
	ReasonSuperseded ArchiveReason = "superseded"
)

// ArchivedBullet represents a bullet removed from playbook.
type ArchivedBullet struct {
	Bullet    *bullet.Bullet
	RemovedAt time.Time
	Reason    ArchiveReason
	Metadata  map[string]string
}

// Archive stores removed bullets with metadata.
type Archive struct {
	bullets map[string]*ArchivedBullet
	mu      sync.RWMutex
}

// ArchiveStats contains archive statistics.
type ArchiveStats struct {
	TotalBullets  int
	ByReason      map[ArchiveReason]int
	OldestArchive time.Time
	NewestArchive time.Time
}

// NewArchive creates a new archive.
func NewArchive() *Archive {
	return &Archive{
		bullets: make(map[string]*ArchivedBullet),
	}
}

// Archive stores a removed bullet.
func (a *Archive) Archive(b *bullet.Bullet, reason ArchiveReason, metadata map[string]string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	archived := &ArchivedBullet{
		Bullet:    b.Clone(), // Clone to preserve state.
		RemovedAt: time.Now(),
		Reason:    reason,
		Metadata:  metadata,
	}

	if archived.Metadata == nil {
		archived.Metadata = make(map[string]string)
	}

	a.bullets[b.ID] = archived
}

// Get retrieves an archived bullet by ID.
func (a *Archive) Get(id string) (*ArchivedBullet, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	archived, exists := a.bullets[id]

	return archived, exists
}

// List returns all archived bullets, optionally filtered.
func (a *Archive) List(filter func(*ArchivedBullet) bool) []*ArchivedBullet {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*ArchivedBullet, 0, len(a.bullets))

	for _, archived := range a.bullets {
		if filter == nil || filter(archived) {
			result = append(result, archived)
		}
	}

	return result
}

// Stats returns archive statistics.
func (a *Archive) Stats() ArchiveStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := ArchiveStats{
		TotalBullets: len(a.bullets),
		ByReason:     make(map[ArchiveReason]int),
	}

	if stats.TotalBullets == 0 {
		return stats
	}

	first := true

	for _, archived := range a.bullets {
		stats.ByReason[archived.Reason]++

		if first || archived.RemovedAt.Before(stats.OldestArchive) {
			stats.OldestArchive = archived.RemovedAt
		}

		if first || archived.RemovedAt.After(stats.NewestArchive) {
			stats.NewestArchive = archived.RemovedAt
		}

		first = false
	}

	return stats
}

// Clear removes all archived bullets.
func (a *Archive) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bullets = make(map[string]*ArchivedBullet)
}

// Len returns the total number of archived bullets.
func (a *Archive) Len() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.bullets)
}
