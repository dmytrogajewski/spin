package playbook

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/events"
)

// Playbook manages a collection of context bullets.
// It provides thread-safe CRUD operations, semantic search,
// serialization, and version control capabilities.
type Playbook struct {
	bullets  map[string]*bullet.Bullet // Index by ID for O(1) lookup.
	mu       sync.RWMutex              // Thread-safe access.
	emitter  *events.EventEmitter      // Event emission (optional).
	embedder embedding.Embedder        // Semantic embedding provider (optional).
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
		bullets:  make(map[string]*bullet.Bullet),
		emitter:  emitter,
		embedder: embedder,
	}
}

// Stats returns playbook statistics.
func (p *Playbook) Stats() Stats {
	p.mu.RLock()
	defer p.mu.RUnlock()

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

		// Estimate size: ID + Content + counters + timestamps.
		stats.TotalSizeBytes += int64(len(b.ID))
		stats.TotalSizeBytes += int64(len(b.Content))
		stats.TotalSizeBytes += 16 // counters + timestamps.

		stats.TotalSizeBytes += int64(len(b.Embedding) * 4) // float32 = 4 bytes.
		for k, v := range b.Tags {
			stats.TotalSizeBytes += int64(len(k) + len(v))
		}
	}

	stats.AvgScore = totalScore / float64(stats.TotalBullets)

	return stats
}

// Add adds a new bullet to the playbook.
// Returns an error if a bullet with the same ID already exists.
func (p *Playbook) Add(ctx context.Context, b *bullet.Bullet) error {
	if b == nil {
		return errors.New("bullet cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for duplicate ID.
	if _, exists := p.bullets[b.ID]; exists {
		return fmt.Errorf("bullet with ID %s already exists", b.ID)
	}

	p.bullets[b.ID] = b

	return nil
}

// Get retrieves a bullet by ID.
// Returns (bullet, true) if found, (nil, false) if not found.
func (p *Playbook) Get(id string) (*bullet.Bullet, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	b, found := p.bullets[id]

	return b, found
}

// Update updates an existing bullet in the playbook.
// Returns an error if the bullet doesn't exist.
func (p *Playbook) Update(ctx context.Context, b *bullet.Bullet) error {
	if b == nil {
		return errors.New("bullet cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.bullets[b.ID]; !exists {
		return fmt.Errorf("bullet with ID %s not found", b.ID)
	}

	p.bullets[b.ID] = b

	return nil
}

// Delete removes a bullet by ID.
// This operation is idempotent - no error if ID doesn't exist.
func (p *Playbook) Delete(ctx context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.bullets, id)

	return nil
}

// FilterFunc is a predicate for filtering bullets.
type FilterFunc func(*bullet.Bullet) bool

// List returns all bullets, optionally filtered.
// If filter is nil, returns all bullets.
func (p *Playbook) List(filter FilterFunc) []*bullet.Bullet {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*bullet.Bullet, 0, len(p.bullets))

	for _, b := range p.bullets {
		if filter == nil || filter(b) {
			result = append(result, b)
		}
	}

	return result
}
