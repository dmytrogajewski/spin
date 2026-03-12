package trajectory

import (
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

// TriggerType defines why retrieval was triggered.
type TriggerType string

const (
	// TriggerInitial is triggered on turn 0 (initial retrieval).
	TriggerInitial TriggerType = "initial"
	// TriggerError is triggered when tool or LLM error occurs.
	TriggerError TriggerType = "error"
	// TriggerToolChange is triggered when a different tool is used.
	TriggerToolChange TriggerType = "tool_change"
	// TriggerInterval is triggered when cache TTL expires.
	TriggerInterval TriggerType = "interval"
)

// RetrievalEvent records when, why, and what bullets were retrieved.
// This enriches the trajectory for Reflector analysis.
type RetrievalEvent struct {
	Turn         int         // Turn when retrieval occurred.
	Trigger      TriggerType // Why retrieval was triggered.
	Query        string      // Query used for retrieval.
	BulletsAdded []string    // Bullet IDs added to cache.
	Timestamp    time.Time   // When retrieval occurred.
}

// CachedBullet tracks bullet usage and lifecycle.
type CachedBullet struct {
	Bullet       *bullet.Bullet // The bullet instance.
	RetrievedAt  int            // Turn when first retrieved.
	AccessCount  int            // Times included in LLM prompt.
	LastAccessed int            // Last turn accessed.
}

// Context is the progressive execution context built during agent loop.
// It serves as the SINGLE SOURCE OF TRUTH for:
// - Retrieval: Dynamic query building and bullet caching
// - Reflector: Rich trajectory with retrieval provenance
// - Agent: Execution state tracking
//
// NOT thread-safe. Must be used within single goroutine.
type Context struct {
	// Immutable (set at creation).
	Query     string    // Initial user query.
	SessionID string    // Unique session identifier.
	StartTime time.Time // Context creation time.

	// Progressive state (updated each turn).
	Steps       []generator.TrajectoryStep
	CurrentTurn int
	Success     bool // Set at end of execution.

	// Retrieval management.
	RetrievalEvents   []RetrievalEvent
	BulletCache       map[string]*CachedBullet
	LastRetrievalTurn int

	// Configuration.
	BulletTTL int // TTL for bullets in cache (default: 10 turns).

	// Metrics.
	TotalRetrievals int
	CacheHits       int
	CacheMisses     int
}

// NewContext creates a new progressive context.
// Generates unique session ID and initializes empty collections.
func NewContext(query string) *Context {
	return &Context{
		Query:           query,
		SessionID:       uuid.New().String(),
		StartTime:       time.Now(),
		Steps:           make([]generator.TrajectoryStep, 0),
		RetrievalEvents: make([]RetrievalEvent, 0),
		BulletCache:     make(map[string]*CachedBullet),
		BulletTTL:       10, // Default TTL of 10 turns.
	}
}

// SetBulletTTL sets the bullet cache TTL (time-to-live in turns).
func (tc *Context) SetBulletTTL(ttl int) {
	if ttl > 0 {
		tc.BulletTTL = ttl
	}
}

// AppendSteps adds new execution steps to context.
// Steps are appended in order (FIFO).
func (tc *Context) AppendSteps(steps []generator.TrajectoryStep) {
	if len(steps) == 0 {
		return
	}

	tc.Steps = append(tc.Steps, steps...)
}

// RecordRetrieval records a retrieval event and merges bullets into cache.
// Updates cache hits/misses metrics.
func (tc *Context) RecordRetrieval(event RetrievalEvent, bullets []*bullet.Bullet) {
	// Record event.
	tc.RetrievalEvents = append(tc.RetrievalEvents, event)
	tc.LastRetrievalTurn = event.Turn
	tc.TotalRetrievals++

	// Merge bullets into cache.
	for _, b := range bullets {
		if cached, exists := tc.BulletCache[b.ID]; exists {
			// Already cached - increment access count.
			cached.AccessCount++
			tc.CacheHits++
		} else {
			// New bullet - add to cache.
			tc.BulletCache[b.ID] = &CachedBullet{
				Bullet:       b,
				RetrievedAt:  event.Turn,
				AccessCount:  1,
				LastAccessed: event.Turn,
			}
			tc.CacheMisses++
		}
	}
}

// GetActiveBullets returns bullets for LLM prompt (cache + TTL filtering).
// Updates last accessed time for returned bullets.
// Uses the configured BulletTTL to determine which bullets are still active.
func (tc *Context) GetActiveBullets() []*bullet.Bullet {
	bullets := make([]*bullet.Bullet, 0, len(tc.BulletCache))

	// Collect active bullets within TTL.
	for _, cached := range tc.BulletCache {
		if tc.CurrentTurn-cached.RetrievedAt <= tc.BulletTTL {
			bullets = append(bullets, cached.Bullet)
			cached.LastAccessed = tc.CurrentTurn
		}
	}

	// Sort by ID for deterministic ordering.
	sort.Slice(bullets, func(i, j int) bool {
		return bullets[i].ID < bullets[j].ID
	})

	return bullets
}

// ToTrajectory converts context to Trajectory for Reflector.
// Includes all steps, bullets, and retrieval events.
func (tc *Context) ToTrajectory() *generator.Trajectory {
	// Extract all bullets from cache (regardless of TTL).
	allBullets := make([]*bullet.Bullet, 0, len(tc.BulletCache))
	for _, cached := range tc.BulletCache {
		allBullets = append(allBullets, cached.Bullet)
	}

	return &generator.Trajectory{
		ID:               tc.SessionID,
		Query:            tc.Query,
		RetrievedBullets: allBullets,
		Steps:            tc.Steps,
		Success:          tc.Success,
		Metadata: generator.TrajectoryMetadata{
			Turns:           tc.CurrentTurn + 1,
			Duration:        time.Since(tc.StartTime),
			RetrievalEvents: tc.RetrievalEvents,
		},
		CreatedAt: tc.StartTime,
	}
}
