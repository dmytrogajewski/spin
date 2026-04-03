package playbook

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// ErrSnapshotCannotBeNil is a sentinel error.
var ErrSnapshotCannotBeNil = errors.New("snapshot cannot be nil")

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
// Bullets and stats are collected in a single pass for consistency.
func (p *Playbook) Snapshot() *Snapshot {
	var (
		bullets []*bullet.Bullet
		acc     statsAccumulator
	)

	p.bullets.Range(func(_ string, b *bullet.Bullet) bool {
		bullets = append(bullets, b.Clone())
		acc.add(b)

		return true
	})

	return &Snapshot{
		ID:        uuid.New().String(),
		Bullets:   bullets,
		CreatedAt: time.Now(),
		Stats:     acc.finalize(),
	}
}

// Restore restores the playbook from a snapshot.
// This replaces all current bullets with those from the snapshot.
func (p *Playbook) Restore(snapshot *Snapshot) error {
	if snapshot == nil {
		return ErrSnapshotCannotBeNil
	}

	p.bullets.Clear()

	// Add bullets from snapshot (deep copy).
	for _, b := range snapshot.Bullets {
		p.bullets.Set(b.ID, b.Clone())
	}

	return nil
}

// Diff compares this snapshot with another and returns the differences.
func (s *Snapshot) Diff(other *Snapshot) *Diff {
	if other == nil {
		return &Diff{}
	}

	// Build maps for fast lookup.
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

	// Find added and modified bullets.
	for id, otherBullet := range otherMap {
		thisBullet, exists := thisMap[id]
		if !exists {
			// Bullet exists in other but not in this = added.
			diff.Added = append(diff.Added, otherBullet)
		} else if !bulletsEqual(thisBullet, otherBullet) {
			// Bullet exists in both but different = modified.
			diff.Modified = append(diff.Modified, &BulletChange{
				ID:     id,
				Before: thisBullet,
				After:  otherBullet,
			})
		}
	}

	// Find removed bullets.
	for id, thisBullet := range thisMap {
		if _, exists := otherMap[id]; !exists {
			// Bullet exists in this but not in other = removed.
			diff.Removed = append(diff.Removed, thisBullet)
		}
	}

	return diff
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
