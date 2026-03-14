// Package refine provides response refinement and archival.
package refine

import (
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/syncmap"
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
	bullets *syncmap.Map[string, *ArchivedBullet]
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
		bullets: syncmap.New[string, *ArchivedBullet](),
	}
}

// Archive stores a removed bullet.
func (a *Archive) Archive(b *bullet.Bullet, reason ArchiveReason, metadata map[string]string) {
	archived := &ArchivedBullet{
		Bullet:    b.Clone(), // Clone to preserve state.
		RemovedAt: time.Now(),
		Reason:    reason,
		Metadata:  metadata,
	}

	if archived.Metadata == nil {
		archived.Metadata = make(map[string]string)
	}

	a.bullets.Set(b.ID, archived)
}

// Get retrieves an archived bullet by ID.
func (a *Archive) Get(id string) (*ArchivedBullet, bool) {
	return a.bullets.Get(id)
}

// List returns all archived bullets, optionally filtered.
func (a *Archive) List(filter func(*ArchivedBullet) bool) []*ArchivedBullet {
	if filter == nil {
		return a.bullets.Values()
	}

	var result []*ArchivedBullet

	a.bullets.Range(func(_ string, archived *ArchivedBullet) bool {
		if filter(archived) {
			result = append(result, archived)
		}

		return true
	})

	return result
}

// Stats returns archive statistics.
func (a *Archive) Stats() ArchiveStats {
	stats := ArchiveStats{
		ByReason: make(map[ArchiveReason]int),
	}

	first := true

	a.bullets.Range(func(_ string, archived *ArchivedBullet) bool {
		stats.TotalBullets++
		stats.ByReason[archived.Reason]++

		if first || archived.RemovedAt.Before(stats.OldestArchive) {
			stats.OldestArchive = archived.RemovedAt
		}

		if first || archived.RemovedAt.After(stats.NewestArchive) {
			stats.NewestArchive = archived.RemovedAt
		}

		first = false

		return true
	})

	return stats
}

// Clear removes all archived bullets.
func (a *Archive) Clear() {
	a.bullets.Clear()
}

// Len returns the total number of archived bullets.
func (a *Archive) Len() int {
	return a.bullets.Len()
}
