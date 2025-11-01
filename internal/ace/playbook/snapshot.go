package playbook

import (
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/google/uuid"
)

// Snapshot is an immutable point-in-time capture of a playbook.
type Snapshot struct {
	ID        string
	Bullets   []*bullet.Bullet
	CreatedAt time.Time
	Stats     Stats
}

// Diff represents differences between two snapshots.
type Diff struct {
	Added    []*bullet.Bullet
	Removed  []*bullet.Bullet
	Modified []*BulletChange
}

// BulletChange represents a modification to a bullet.
type BulletChange struct {
	ID     string
	Before *bullet.Bullet
	After  *bullet.Bullet
}

// Snapshot creates an immutable snapshot of the current playbook state.
// All bullets are deep copied to ensure immutability.
func (p *Playbook) Snapshot() *Snapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	bullets := make([]*bullet.Bullet, 0, len(p.bullets))
	for _, b := range p.bullets {
		bullets = append(bullets, b.Clone())
	}

	return &Snapshot{
		ID:        uuid.New().String(),
		Bullets:   bullets,
		CreatedAt: time.Now(),
		Stats:     p.statsLocked(),
	}
}

// Restore restores the playbook from a snapshot.
// This replaces all current bullets with those from the snapshot.
func (p *Playbook) Restore(snapshot *Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear current bullets
	p.bullets = make(map[string]*bullet.Bullet)

	// Add bullets from snapshot (deep copy)
	for _, b := range snapshot.Bullets {
		p.bullets[b.ID] = b.Clone()
	}

	return nil
}

// Diff compares this snapshot with another and returns the differences.
func (s *Snapshot) Diff(other *Snapshot) *Diff {
	if other == nil {
		return &Diff{}
	}

	// Build maps for fast lookup
	thisMap := make(map[string]*bullet.Bullet)
	for _, b := range s.Bullets {
		thisMap[b.ID] = b
	}

	otherMap := make(map[string]*bullet.Bullet)
	for _, b := range other.Bullets {
		otherMap[b.ID] = b
	}

	diff := &Diff{
		Added:    make([]*bullet.Bullet, 0),
		Removed:  make([]*bullet.Bullet, 0),
		Modified: make([]*BulletChange, 0),
	}

	// Find added and modified bullets
	for id, otherBullet := range otherMap {
		thisBullet, exists := thisMap[id]
		if !exists {
			// Bullet exists in other but not in this = added
			diff.Added = append(diff.Added, otherBullet)
		} else if !bulletsEqual(thisBullet, otherBullet) {
			// Bullet exists in both but different = modified
			diff.Modified = append(diff.Modified, &BulletChange{
				ID:     id,
				Before: thisBullet,
				After:  otherBullet,
			})
		}
	}

	// Find removed bullets
	for id, thisBullet := range thisMap {
		if _, exists := otherMap[id]; !exists {
			// Bullet exists in this but not in other = removed
			diff.Removed = append(diff.Removed, thisBullet)
		}
	}

	return diff
}

// statsLocked returns stats without acquiring lock (caller must hold lock).
func (p *Playbook) statsLocked() Stats {
	stats := Stats{
		TotalBullets: len(p.bullets),
	}

	if stats.TotalBullets == 0 {
		return stats
	}

	totalScore := 0.0
	for _, b := range p.bullets {
		stats.TotalHelpful += b.HelpfulCount
		stats.TotalHarmful += b.HarmfulCount
		totalScore += b.Score()
		stats.TotalSizeBytes += int64(len(b.ID) + len(b.Content) + 16 + len(b.Embedding)*4)
		for k, v := range b.Tags {
			stats.TotalSizeBytes += int64(len(k) + len(v))
		}
	}

	stats.AvgScore = totalScore / float64(stats.TotalBullets)

	return stats
}

// bulletsEqual checks if two bullets are equal (for diff purposes).
func bulletsEqual(a, b *bullet.Bullet) bool {
	if a.Content != b.Content {
		return false
	}
	if a.HelpfulCount != b.HelpfulCount || a.HarmfulCount != b.HarmfulCount {
		return false
	}
	if len(a.Embedding) != len(b.Embedding) {
		return false
	}
	for i := range a.Embedding {
		if a.Embedding[i] != b.Embedding[i] {
			return false
		}
	}
	if len(a.Tags) != len(b.Tags) {
		return false
	}
	for k, v := range a.Tags {
		if b.Tags[k] != v {
			return false
		}
	}
	return true
}
