# FRD-20251030-008: Grow-and-Refine Mechanism

**Feature ID**: 6 (ACE Integration Roadmap)  
**Created**: 2025-10-30  
**Status**: Implementation  
**Component**: `internal/ace/refine`

---

## Executive Summary

Implement the grow-and-refine mechanism that balances context expansion with redundancy control. This system monitors playbook growth, triggers refinement when thresholds are breached, and applies compression strategies to maintain a high-quality, manageable context size.

**Problem**: As bullets accumulate through the Generator, Reflector, and Curator, the playbook can grow unbounded. Without refinement, this leads to:
1. Excessive token usage (performance degradation)
2. High redundancy (duplicate or similar bullets)
3. Low signal-to-noise ratio (outdated or low-value bullets)
4. Difficult navigation and retrieval

**Solution**: Introduce a Grow-and-Refine system that:
1. Monitors playbook growth metrics (bullet count, token estimate, redundancy)
2. Triggers refinement based on configurable strategies (proactive, lazy, hybrid)
3. Prunes low-utility bullets using score-based ranking
4. Merges similar bullets to reduce redundancy
5. Archives historical context for audit trail

**Note**: This FRD builds on the existing Curator refinement features (Feature 4) and integrates them into a comprehensive growth management system.

---

## Goals

### Primary Goals
1. **Growth Monitoring**: Track playbook size, redundancy, and quality metrics
2. **Refinement Strategies**: Support proactive, lazy, and hybrid refinement modes
3. **Pruning**: Remove low-utility bullets based on configurable thresholds
4. **Merging**: Combine similar bullets to reduce redundancy
5. **Archival**: Preserve removed bullets for audit and recovery

### Secondary Goals
1. **Configuration**: Flexible thresholds and strategy parameters
2. **Metrics**: Track refinement effectiveness (bullets removed, tokens saved)
3. **Performance**: Sub-second refinement for typical playbooks (< 1000 bullets)
4. **Integration**: Seamless integration with existing Curator

### Non-Goals
1. **ML-Based Merging**: Not using LLM for bullet merging (simple similarity-based for now)
2. **Distributed Coordination**: Single-process only
3. **Real-Time Streaming**: Batch refinement only
4. **UI Visualization**: No TUI components (future)

---

## Technical Design

### Overview

Feature 6 extends the existing Curator refinement infrastructure (from Feature 4) with:
1. **GrowthMonitor**: Tracks playbook metrics and triggers refinement
2. **RefinementOrchestrator**: Coordinates pruning, merging, and archival
3. **MergeEngine**: Identifies and merges similar bullets
4. **Archive**: Stores removed bullets with metadata

**Key Insight**: The Curator already has refinement strategies (None, Lazy, Proactive). Feature 6 **enhances** these with merging and archival capabilities.

### Architecture

```
internal/ace/refine/
├── monitor.go       # GrowthMonitor - tracks metrics
├── orchestrator.go  # RefinementOrchestrator - coordinates refinement
├── merge.go         # MergeEngine - merges similar bullets
├── archive.go       # Archive - stores removed bullets
├── metrics.go       # Growth and refinement metrics
└── *_test.go        # Comprehensive tests
```

**Integration Points**:
- **Curator**: Already has refinement strategies, Feature 6 adds merging/archival
- **Playbook**: Provides bullet access and stats
- **Delta**: Could track refinement operations as deltas (future)

### Data Structures

#### GrowthMetrics

```go
// GrowthMetrics tracks playbook growth statistics.
type GrowthMetrics struct {
	BulletCount      int       // Total bullets
	EstimatedTokens  int       // Approximate token count
	AvgUtilityScore  float64   // Average bullet utility
	RedundancyScore  float64   // Estimated redundancy (0.0-1.0)
	LastRefinement   time.Time // When last refined
	GrowthRate       float64   // Bullets per hour
}
```

#### GrowthMonitor

```go
// GrowthMonitor tracks playbook growth and triggers refinement.
type GrowthMonitor struct {
	playbook      *playbook.Playbook
	thresholds    GrowthThresholds
	metrics       GrowthMetrics
	lastCheck     time.Time
	mu            sync.RWMutex
}

// GrowthThresholds defines when to trigger refinement.
type GrowthThresholds struct {
	MaxBullets      int     // Trigger when bullet count exceeds
	MaxTokens       int     // Trigger when estimated tokens exceed
	MinUtility      float64 // Trigger when avg utility drops below
	MaxRedundancy   float64 // Trigger when redundancy exceeds
	CheckInterval   time.Duration // How often to check metrics
}
```

#### RefinementOrchestrator

```go
// RefinementOrchestrator coordinates refinement operations.
type RefinementOrchestrator struct {
	playbook    *playbook.Playbook
	mergeEngine *MergeEngine
	archive     *Archive
	curator     *curator.Curator // Uses existing curator for pruning
}

// RefinementRequest specifies what to refine.
type RefinementRequest struct {
	PruneEnabled  bool
	MergeEnabled  bool
	ArchiveEnabled bool
	MinUtility    float64 // Prune bullets below this utility
	MergeSimilarity float64 // Merge bullets above this similarity
}

// RefinementResult contains refinement outcomes.
type RefinementResult struct {
	Pruned      int      // Bullets pruned
	Merged      int      // Bullets merged
	Archived    int      // Bullets archived
	PrunedIDs   []string // IDs of pruned bullets
	MergedPairs []MergePair // Pairs of merged bullets
	TokensSaved int      // Estimated tokens saved
	Duration    time.Duration
}
```

#### MergeEngine

```go
// MergeEngine identifies and merges similar bullets.
type MergeEngine struct {
	embedder    embedding.Embedder
	similarity  float64 // Threshold for merging (default 0.90)
}

// MergePair represents two bullets to merge.
type MergePair struct {
	SourceID    string  // Bullet to merge from
	TargetID    string  // Bullet to merge into
	Similarity  float64 // Similarity score
}

// MergeResult contains merge operation outcome.
type MergeResult struct {
	KeptID      string // ID of kept bullet
	RemovedID   string // ID of removed bullet
	MergedContent string // Combined content (if applicable)
}
```

#### Archive

```go
// Archive stores removed bullets with metadata.
type Archive struct {
	bullets  map[string]ArchivedBullet
	mu       sync.RWMutex
}

// ArchivedBullet represents a bullet removed from playbook.
type ArchivedBullet struct {
	Bullet       *bullet.Bullet
	RemovedAt    time.Time
	Reason       ArchiveReason
	Metadata     map[string]string
}

// ArchiveReason explains why bullet was archived.
type ArchiveReason string

const (
	ReasonLowUtility    ArchiveReason = "low_utility"
	ReasonMerged        ArchiveReason = "merged"
	ReasonManual        ArchiveReason = "manual"
	ReasonSuperseded    ArchiveReason = "superseded"
)
```

### Core Algorithms

#### Growth Monitoring

```go
// CheckGrowth evaluates current playbook state and returns metrics.
func (m *GrowthMonitor) CheckGrowth(ctx context.Context) (GrowthMetrics, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats := m.playbook.Stats()
	
	// Calculate metrics
	metrics := GrowthMetrics{
		BulletCount:     stats.TotalBullets,
		EstimatedTokens: m.estimateTokens(stats),
		AvgUtilityScore: stats.AvgScore,
		RedundancyScore: m.estimateRedundancy(ctx),
		LastRefinement:  m.metrics.LastRefinement,
		GrowthRate:      m.calculateGrowthRate(stats),
	}
	
	m.metrics = metrics
	m.lastCheck = time.Now()
	
	// Check if refinement needed
	needsRefinement := m.shouldRefine(metrics)
	
	return metrics, needsRefinement
}

// shouldRefine determines if any threshold is breached.
func (m *GrowthMonitor) shouldRefine(metrics GrowthMetrics) bool {
	if metrics.BulletCount >= m.thresholds.MaxBullets {
		return true
	}
	if metrics.EstimatedTokens >= m.thresholds.MaxTokens {
		return true
	}
	if metrics.AvgUtilityScore < m.thresholds.MinUtility {
		return true
	}
	if metrics.RedundancyScore > m.thresholds.MaxRedundancy {
		return true
	}
	return false
}
```

#### Merging Similar Bullets

```go
// FindMergeCandidates identifies bullet pairs to merge.
func (m *MergeEngine) FindMergeCandidates(ctx context.Context, bullets []*bullet.Bullet) ([]MergePair, error) {
	pairs := make([]MergePair, 0)
	
	// O(n²) similarity comparison (acceptable for < 1000 bullets)
	for i := 0; i < len(bullets); i++ {
		for j := i + 1; j < len(bullets); j++ {
			similarity := m.calculateSimilarity(bullets[i], bullets[j])
			
			if similarity >= m.similarity {
				// Choose which to keep based on utility
				sourceID, targetID := m.chooseMergeDirection(bullets[i], bullets[j])
				
				pairs = append(pairs, MergePair{
					SourceID:   sourceID,
					TargetID:   targetID,
					Similarity: similarity,
				})
			}
		}
	}
	
	return pairs, nil
}

// MergeBullets combines source into target.
func (m *MergeEngine) MergeBullets(ctx context.Context, source, target *bullet.Bullet) (*MergeResult, error) {
	// Keep bullet with higher utility
	kept := target
	removed := source
	
	if source.Score() > target.Score() {
		kept = source
		removed = target
	}
	
	// Transfer utility counters
	kept.HelpfulCount += removed.HelpfulCount
	kept.HarmfulCount += removed.HarmfulCount
	
	// Merge tags (kept's tags take precedence)
	for k, v := range removed.Tags {
		if _, exists := kept.Tags[k]; !exists {
			kept.Tags[k] = v
		}
	}
	
	return &MergeResult{
		KeptID:      kept.ID,
		RemovedID:   removed.ID,
		MergedContent: kept.Content, // Could combine content in future
	}, nil
}
```

#### Refinement Orchestration

```go
// Refine executes full refinement workflow.
func (o *RefinementOrchestrator) Refine(ctx context.Context, req RefinementRequest) (*RefinementResult, error) {
	start := time.Now()
	result := &RefinementResult{}
	
	bullets := o.playbook.List(nil)
	
	// 1. Merge similar bullets (if enabled)
	if req.MergeEnabled {
		pairs, err := o.mergeEngine.FindMergeCandidates(ctx, bullets)
		if err != nil {
			return nil, err
		}
		
		for _, pair := range pairs {
			source, _ := o.playbook.Get(pair.SourceID)
			target, _ := o.playbook.Get(pair.TargetID)
			
			mergeResult, err := o.mergeEngine.MergeBullets(ctx, source, target)
			if err != nil {
				continue // Skip failed merges
			}
			
			// Archive removed bullet
			if req.ArchiveEnabled {
				o.archive.Archive(source, ReasonMerged, map[string]string{
					"merged_into": mergeResult.KeptID,
				})
			}
			
			// Remove from playbook
			o.playbook.Delete(ctx, mergeResult.RemovedID)
			
			result.Merged++
			result.MergedPairs = append(result.MergedPairs, pair)
		}
	}
	
	// 2. Prune low-utility bullets (if enabled)
	if req.PruneEnabled {
		// Use curator's existing refinement
		curatorResult, err := o.curator.Refine(ctx)
		if err != nil {
			return nil, err
		}
		
		result.Pruned = curatorResult.Pruned
		result.PrunedIDs = curatorResult.PrunedIDs
		
		// Archive pruned bullets
		if req.ArchiveEnabled {
			for _, id := range curatorResult.PrunedIDs {
				if b, exists := o.playbook.Get(id); exists {
					o.archive.Archive(b, ReasonLowUtility, nil)
				}
			}
		}
	}
	
	// 3. Calculate tokens saved
	result.TokensSaved = (result.Pruned + result.Merged) * 50 // Rough estimate
	result.Duration = time.Since(start)
	
	return result, nil
}
```

### Integration with Curator

The Curator (Feature 4) already has refinement strategies. Feature 6 **extends** these:

**Existing Curator Refinement**:
- `RefinementModeNone`: No automatic refinement
- `RefinementModeLazy`: Manual `Refine()` calls
- `RefinementModeProactive`: Auto-refine after `Curate()` when threshold reached

**Feature 6 Enhancements**:
- Adds **merging** capability (combine similar bullets)
- Adds **archival** system (preserve removed bullets)
- Adds **growth monitoring** (track metrics over time)
- Adds **orchestration** (coordinate prune + merge + archive)

**Usage**:
```go
// Create curator with lazy refinement (existing)
curator := curator.NewCurator(pb, embedder,
	curator.WithRefinementMode(
		curator.RefinementModeLazy,
		curator.LazyRefinementConfig{MinUtilityScore: 0.1},
	),
)

// Create refine orchestrator (new)
archive := refine.NewArchive()
mergeEngine := refine.NewMergeEngine(embedder, 0.90)
orchestrator := refine.NewRefinementOrchestrator(pb, mergeEngine, archive, curator)

// Refined workflow with merging
result, err := orchestrator.Refine(ctx, refine.RefinementRequest{
	PruneEnabled:    true,  // Use curator's pruning
	MergeEnabled:    true,  // Add merging
	ArchiveEnabled:  true,  // Preserve removed bullets
	MinUtility:      0.1,
	MergeSimilarity: 0.90,
})
```

---

## API Reference

### GrowthMonitor

```go
// NewGrowthMonitor creates a new growth monitor.
func NewGrowthMonitor(pb *playbook.Playbook, thresholds GrowthThresholds) *GrowthMonitor

// CheckGrowth evaluates current playbook state.
func (m *GrowthMonitor) CheckGrowth(ctx context.Context) (GrowthMetrics, bool)

// GetMetrics returns current metrics.
func (m *GrowthMonitor) GetMetrics() GrowthMetrics

// ShouldRefine checks if refinement is needed.
func (m *GrowthMonitor) ShouldRefine() bool
```

### RefinementOrchestrator

```go
// NewRefinementOrchestrator creates a new orchestrator.
func NewRefinementOrchestrator(
	pb *playbook.Playbook,
	mergeEngine *MergeEngine,
	archive *Archive,
	curator *curator.Curator,
) *RefinementOrchestrator

// Refine executes full refinement workflow.
func (o *RefinementOrchestrator) Refine(ctx context.Context, req RefinementRequest) (*RefinementResult, error)
```

### MergeEngine

```go
// NewMergeEngine creates a new merge engine.
func NewMergeEngine(embedder embedding.Embedder, similarityThreshold float64) *MergeEngine

// FindMergeCandidates identifies bullets to merge.
func (m *MergeEngine) FindMergeCandidates(ctx context.Context, bullets []*bullet.Bullet) ([]MergePair, error)

// MergeBullets combines two bullets.
func (m *MergeEngine) MergeBullets(ctx context.Context, source, target *bullet.Bullet) (*MergeResult, error)
```

### Archive

```go
// NewArchive creates a new archive.
func NewArchive() *Archive

// Archive stores a removed bullet.
func (a *Archive) Archive(b *bullet.Bullet, reason ArchiveReason, metadata map[string]string)

// Get retrieves an archived bullet.
func (a *Archive) Get(id string) (*ArchivedBullet, bool)

// List returns all archived bullets.
func (a *Archive) List(filter func(*ArchivedBullet) bool) []*ArchivedBullet

// Stats returns archive statistics.
func (a *Archive) Stats() ArchiveStats
```

---

## Performance Targets

| Operation | Target | Rationale |
|-----------|--------|-----------|
| CheckGrowth | < 10ms | Lightweight metric calculation |
| FindMergeCandidates (100 bullets) | < 100ms | O(n²) acceptable for small playbooks |
| FindMergeCandidates (1000 bullets) | < 10s | Larger playbooks, infrequent operation |
| Refine (full workflow, 100 bullets) | < 200ms | Prune + merge + archive |
| Archive storage | < 1MB per 1000 bullets | Efficient serialization |

---

## Testing Strategy

### Unit Tests (90%+ Coverage)

1. **GrowthMonitor**
   - Metric calculation
   - Threshold evaluation
   - Growth rate tracking

2. **MergeEngine**
   - Similarity calculation
   - Merge candidate identification
   - Bullet merging logic
   - Utility counter transfer

3. **Archive**
   - Store and retrieve
   - Filtering and stats
   - Thread safety

4. **RefinementOrchestrator**
   - Full workflow
   - Partial refinement (prune-only, merge-only)
   - Error handling

### Integration Tests

1. **End-to-End Refinement**
   - Playbook with 100 bullets
   - Trigger refinement
   - Verify prune + merge + archive

2. **Growth Monitoring**
   - Add bullets over time
   - Monitor metrics
   - Trigger refinement automatically

3. **Curator Integration**
   - Existing curator refinement
   - Enhanced with merging
   - Archive preservation

### Benchmarks

```go
func BenchmarkCheckGrowth(b *testing.B)
func BenchmarkFindMergeCandidates_100(b *testing.B)
func BenchmarkFindMergeCandidates_1000(b *testing.B)
func BenchmarkRefine_Full_100(b *testing.B)
```

---

## Definition of Done

- [x] FRD created and reviewed
- [ ] `monitor.go` - GrowthMonitor with metrics
- [ ] `orchestrator.go` - RefinementOrchestrator
- [ ] `merge.go` - MergeEngine with similarity
- [ ] `archive.go` - Archive with storage
- [ ] `metrics.go` - Growth and refinement metrics
- [ ] Unit tests for all packages (≥90% coverage)
- [ ] Integration tests with Curator
- [ ] Benchmarks for performance validation
- [ ] Race detector clean (`go test -race`)
- [ ] Linter clean (`make lint`)
- [ ] Documentation in `docs/packages/ace.md`
- [ ] ROADMAP.md Feature 6 marked complete

---

## Open Questions

1. **Merge Content Strategy**: Should we combine content strings, or just keep one?
   - **Decision**: Keep higher-utility bullet's content for simplicity. Future: LLM-based merging.

2. **Archive Persistence**: Should archive be persisted to disk?
   - **Decision**: In-memory for MVP. Future: add `Archive.Save/Load()`.

3. **Merge Similarity Threshold**: What's the right default?
   - **Decision**: 0.90 (very similar). Make configurable.

4. **Growth Rate Calculation**: How to track over time?
   - **Decision**: Use time-windowed average (last 1 hour).

5. **Integration Point**: Should refinement use Delta system?
   - **Decision**: No for MVP. Refinement is higher-level. Future: track as deltas.

---

## References

- [ACE Roadmap Feature 6](../../ace-agentic-context-engineering/ROADMAP.md#feature-6-grow-and-refine-mechanism)
- [FRD-20251030-004: Curator Component](./FRD-20251030-004-curator-component.md)
- [FRD-20251030-005: Online Context Adaptation](./FRD-20251030-005-online-context-adaptation.md)
- [FRD-20251030-007: Incremental Delta Updates](./FRD-20251030-007-incremental-delta-updates.md)
- [ACE Paper](../../ace-agentic-context-engineering/2510.04618v1.pdf)
- [Curator Package](../../docs/packages/ace.md#curator-component)

---

**Status**: Ready for Implementation  
**Estimated Effort**: 10-14 hours (5-7 hours implementation + 5-7 hours testing)  
**Dependencies**: Curator (Feature 4), Playbook (Feature 1)
