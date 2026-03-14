// Package playbook provides playbook management for agent workflows.
package playbook

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/syncmap"
)

const bytesPerFloat32 = 4

var (
	// ErrBulletCannotBeNil is a sentinel error.
	ErrBulletCannotBeNil = errors.New("bullet cannot be nil")
	// ErrBulletWithIDAlreadyExists is a sentinel error.
	ErrBulletWithIDAlreadyExists = errors.New("bullet with ID  already exists")
	// ErrBulletWithIDNotFound is a sentinel error.
	ErrBulletWithIDNotFound = errors.New("bullet with ID  not found")
)

// Playbook manages a collection of context bullets.
// It provides thread-safe CRUD operations, semantic search,
// serialization, and version control capabilities.
type Playbook struct {
	bullets  *syncmap.Map[string, *bullet.Bullet] // Index by ID for O(1) lookup.
	emitter  *events.EventEmitter                 // Event emission (optional).
	embedder embedding.Embedder                   // Semantic embedding provider (optional).
}

// Stats contains playbook statistics.
type Stats struct {
	TotalBullets   int
	TotalHelpful   int
	TotalHarmful   int
	AvgScore       float64
	TotalSizeBytes int64
}

// New creates a new empty playbook.
// Both emitter and embedder are optional (can be nil).
func New(emitter *events.EventEmitter, embedder embedding.Embedder) *Playbook {
	return &Playbook{
		bullets:  syncmap.New[string, *bullet.Bullet](),
		emitter:  emitter,
		embedder: embedder,
	}
}

// Stats returns playbook statistics.
func (p *Playbook) Stats() Stats {
	var stats Stats

	totalScore := 0.0

	p.bullets.Range(func(_ string, b *bullet.Bullet) bool {
		stats.TotalBullets++
		stats.TotalHelpful += b.HelpfulCount
		stats.TotalHarmful += b.HarmfulCount
		totalScore += b.Score()

		// Estimate size: ID + Content + counters + timestamps.
		stats.TotalSizeBytes += int64(len(b.ID))
		stats.TotalSizeBytes += int64(len(b.Content))
		stats.TotalSizeBytes += 16 // counters + timestamps.

		stats.TotalSizeBytes += int64(len(b.Embedding) * bytesPerFloat32) // float32 = 4 bytes.
		for k, v := range b.Tags {
			stats.TotalSizeBytes += int64(len(k) + len(v))
		}

		return true
	})

	if stats.TotalBullets > 0 {
		stats.AvgScore = totalScore / float64(stats.TotalBullets)
	}

	return stats
}

// Add adds a new bullet to the playbook.
// Returns an error if a bullet with the same ID already exists.
func (p *Playbook) Add(_ context.Context, b *bullet.Bullet) error {
	if b == nil {
		return ErrBulletCannotBeNil
	}

	if !p.bullets.SetIfAbsent(b.ID, b) {
		return fmt.Errorf("bullet with ID %s already exists: %w", b.ID, ErrBulletWithIDAlreadyExists)
	}

	return nil
}

// Get retrieves a bullet by ID.
// Returns (bullet, true) if found, (nil, false) if not found.
func (p *Playbook) Get(id string) (*bullet.Bullet, bool) {
	return p.bullets.Get(id)
}

// Update updates an existing bullet in the playbook.
// Returns an error if the bullet doesn't exist.
func (p *Playbook) Update(_ context.Context, b *bullet.Bullet) error {
	if b == nil {
		return ErrBulletCannotBeNil
	}

	if !p.bullets.SetIfPresent(b.ID, b) {
		return fmt.Errorf("bullet with ID %s not found: %w", b.ID, ErrBulletWithIDNotFound)
	}

	return nil
}

// Delete removes a bullet by ID.
// This operation is idempotent - no error if ID doesn't exist.
func (p *Playbook) Delete(_ context.Context, id string) error {
	p.bullets.Delete(id)

	return nil
}

// FilterFunc is a predicate for filtering bullets.
type FilterFunc func(*bullet.Bullet) bool

// List returns all bullets, optionally filtered.
// If filter is nil, returns all bullets.
func (p *Playbook) List(filter FilterFunc) []*bullet.Bullet {
	if filter == nil {
		return p.bullets.Values()
	}

	var result []*bullet.Bullet

	p.bullets.Range(func(_ string, b *bullet.Bullet) bool {
		if filter(b) {
			result = append(result, b)
		}

		return true
	})

	return result
}
