# Feature Requirements Document: Curator Refinement Features

**FRD ID:** FRD-20251030-006  
**Feature:** Parallel Delta Merging & Lazy/Proactive Refinement Modes  
**Phase:** 4 - Refinement & Optimization  
**Status:** Draft  
**Created:** 2025-10-30  
**Author:** AI Agent (Spin)

---

## Executive Summary

This FRD describes two deferred features from the Curator component (Feature 4):
1. **Parallel Delta Merging Support** - Process multiple merge requests concurrently
2. **Lazy vs. Proactive Refinement Modes** - Control when playbook refinement occurs

**Key Goals:**
- Increase curation throughput for high-volume scenarios
- Provide flexible refinement strategies to manage playbook size
- Maintain deterministic, thread-safe operations

---

## Background

### Context

The Curator component is complete with basic functionality (92.9% coverage):
- ✅ Insight to bullet conversion
- ✅ Semantic deduplication
- ✅ Playbook merging
- ✅ Statistics tracking

However, two features were deferred:
- Parallel processing of multiple merge requests
- Configurable refinement modes (lazy vs. proactive)

### Problem Statement

**Current Limitations:**

1. **Sequential Processing**: Large batches of insights are processed sequentially, limiting throughput
2. **No Refinement Strategy**: Playbook grows indefinitely without automatic pruning
3. **No Size Management**: No automatic triggers for removing low-utility bullets

**Impact:**
- Slow curation for batch offline learning scenarios
- Playbook bloat over time (memory and search performance degradation)
- No control over when/how playbook is refined

---

## Requirements

### Feature 1: Parallel Delta Merging Support

#### FR-1.1: Concurrent Curate Operations
**Priority:** SHOULD  
**Description:** Support concurrent calls to Curate() safely.

**Acceptance Criteria:**
- Multiple goroutines can call Curate() simultaneously
- No race conditions (verified by `go test -race`)
- Results are deterministic (same inputs → same outputs)
- Thread-safe access to playbook (already provided by playbook.Playbook)

#### FR-1.2: Batch Parallel Processing
**Priority:** SHOULD  
**Description:** Process multiple MergeRequest batches in parallel.

**Acceptance Criteria:**
- Accept `[]MergeRequest` as input
- Process each request in separate goroutine
- Return `[]MergeResult` preserving order
- Handle errors per-request (don't fail entire batch)
- Configurable concurrency limit (default: NumCPU)

**API Design:**
```go
type BatchMergeRequest struct {
    Requests   []MergeRequest
    MaxWorkers int // 0 = runtime.NumCPU()
}

type BatchMergeResult struct {
    Results []MergeResult
    Errors  []error // per-request errors (nil if successful)
}

// CurateBatch processes multiple merge requests in parallel
func (c *curator) CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error)
```

#### FR-1.3: Performance Improvement
**Priority:** SHOULD  
**Description:** Achieve measurable throughput improvement.

**Targets:**
- 10 requests in parallel: ~8x faster than sequential (on 8-core CPU)
- 100 requests in parallel: ~10x faster (with worker pool)
- Memory overhead: < 10MB additional for worker pool

---

### Feature 2: Lazy vs. Proactive Refinement Modes

#### FR-2.1: Refinement Strategy Interface
**Priority:** MUST  
**Description:** Define interface for refinement strategies.

**API Design:**
```go
type RefinementStrategy interface {
    // ShouldRefine returns true if playbook should be refined now
    ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error)
    
    // Refine performs playbook refinement (prune low-utility bullets)
    Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error)
}

type RefinementResult struct {
    Pruned   int      // Bullets removed
    PrunedIDs []string // IDs of removed bullets
    Reason   string   // Why refinement occurred
}
```

#### FR-2.2: Proactive Refinement Mode
**Priority:** MUST  
**Description:** Refine playbook after every Curate() call.

**Acceptance Criteria:**
- Check refinement condition after each Curate()
- Trigger refinement if threshold exceeded
- Return refinement stats in CurateResult
- Configurable thresholds (bullet count, size, utility)

**Configuration:**
```go
type ProactiveRefinementConfig struct {
    MaxBullets      int     // Trigger at N bullets (default: 1000)
    MaxSizeBytes    int64   // Trigger at N bytes (default: 1MB)
    MinUtilityScore float64 // Prune bullets below score (default: 0.1)
}
```

**Behavior:**
```
After each Curate():
1. Check if bullet count > MaxBullets
2. If yes: Prune bullets with Score() < MinUtilityScore
3. Stop when count <= (MaxBullets * 0.9)
4. Return pruned count in result
```

#### FR-2.3: Lazy Refinement Mode
**Priority:** MUST  
**Description:** Refine playbook only on explicit request.

**Acceptance Criteria:**
- No automatic refinement after Curate()
- Explicit Refine() method must be called
- User controls when refinement occurs
- No performance overhead when unused

**API Design:**
```go
// Refine explicitly prunes low-utility bullets
func (c *curator) Refine(ctx context.Context) (*RefinementResult, error)
```

**Behavior:**
```
On Refine() call:
1. Calculate Score() for all bullets
2. Sort by score (ascending)
3. Prune bullets with score < MinUtilityScore
4. Return pruned count and IDs
```

#### FR-2.4: Configurable Refinement Strategy
**Priority:** MUST  
**Description:** Allow choosing between lazy and proactive at creation.

**API Design:**
```go
type RefinementMode string

const (
    RefinementModeNone       RefinementMode = "none"       // No refinement
    RefinementModeLazy       RefinementMode = "lazy"       // Manual only
    RefinementModeProactive  RefinementMode = "proactive"  // After each Curate
)

// WithRefinementMode sets refinement strategy
func WithRefinementMode(mode RefinementMode, config interface{}) Option

// Usage:
curator := NewCurator(pb, embedder,
    WithRefinementMode(RefinementModeProactive, ProactiveRefinementConfig{
        MaxBullets: 500,
        MinUtilityScore: 0.2,
    }),
)
```

#### FR-2.5: Refinement Observability
**Priority:** SHOULD  
**Description:** Track and report refinement events.

**Acceptance Criteria:**
- Log when refinement triggers
- Track pruned bullet IDs
- Expose refinement stats in result
- Optional callback for refinement events

**API Extension:**
```go
type MergeResult struct {
    Added        int
    Skipped      int
    Updated      int
    Duplicates   []string
    AddedBullets []*bullet.Bullet
    
    // New fields
    Refined      bool             // Was refinement triggered?
    Refinement   *RefinementResult // Refinement stats (if refined)
}
```

---

## Non-Functional Requirements

### NFR-1: Performance
- Parallel processing: 8x speedup on 8-core CPU
- Refinement overhead: < 50ms for 1000 bullets
- Memory: < 10MB additional for parallel processing

### NFR-2: Thread Safety
- All operations safe for concurrent use
- No race conditions (verified by `go test -race`)
- Deterministic results (same inputs → same outputs)

### NFR-3: Testability
- Unit test coverage ≥ 90%
- Benchmark tests for parallel processing
- Integration tests with refinement strategies

### NFR-4: Backward Compatibility
- Existing Curate() API unchanged
- New features opt-in via WithRefinementMode()
- Default behavior: no refinement (backward compatible)

---

## Architecture

### Package Structure

```
internal/ace/curator/
├── curator.go          # Main Curator (existing)
├── parallel.go         # Parallel processing (NEW)
├── refinement.go       # Refinement interface & strategies (NEW)
├── proactive.go        # Proactive refinement strategy (NEW)
├── lazy.go             # Lazy refinement strategy (NEW)
├── curator_test.go     # Unit tests (existing)
├── parallel_test.go    # Parallel tests (NEW)
└── refinement_test.go  # Refinement tests (NEW)
```

### Data Structures

#### Parallel Processing

```go
type parallelProcessor struct {
    curator    *curator
    maxWorkers int
}

func newParallelProcessor(c *curator, maxWorkers int) *parallelProcessor {
    if maxWorkers <= 0 {
        maxWorkers = runtime.NumCPU()
    }
    return &parallelProcessor{
        curator:    c,
        maxWorkers: maxWorkers,
    }
}
```

#### Refinement Strategies

```go
// NoRefinementStrategy never refines
type noRefinementStrategy struct{}

func (n *noRefinementStrategy) ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error) {
    return false, nil
}

func (n *noRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
    return &RefinementResult{Pruned: 0}, nil
}

// LazyRefinementStrategy refines only on explicit call
type lazyRefinementStrategy struct {
    minUtilityScore float64
}

// ProactiveRefinementStrategy refines after each curate
type proactiveRefinementStrategy struct {
    maxBullets      int
    maxSizeBytes    int64
    minUtilityScore float64
}

func (p *proactiveRefinementStrategy) ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error) {
    stats := pb.Stats()
    return stats.Count >= p.maxBullets, nil
}
```

### Updated Curator Structure

```go
type curator struct {
    playbook           *playbook.Playbook
    embedder           embedding.Embedder
    threshold          float64
    refinementStrategy RefinementStrategy  // NEW
    parallel           *parallelProcessor  // NEW
}

type Option func(*curator)

func WithRefinementMode(mode RefinementMode, config interface{}) Option {
    return func(c *curator) {
        switch mode {
        case RefinementModeNone:
            c.refinementStrategy = &noRefinementStrategy{}
        case RefinementModeLazy:
            cfg := config.(LazyRefinementConfig)
            c.refinementStrategy = newLazyRefinementStrategy(cfg)
        case RefinementModeProactive:
            cfg := config.(ProactiveRefinementConfig)
            c.refinementStrategy = newProactiveRefinementStrategy(cfg)
        }
    }
}

func WithMaxWorkers(maxWorkers int) Option {
    return func(c *curator) {
        c.parallel = newParallelProcessor(c, maxWorkers)
    }
}
```

---

## Implementation Plan

### Phase 1: Parallel Delta Merging (Days 1-2)

#### Day 1: Worker Pool & Batch Processing
**Tasks:**
1. Create `parallel.go` with worker pool
2. Implement `CurateBatch()` method
3. Handle context cancellation
4. Handle per-request errors

**Tests:**
- Batch processing (2, 10, 100 requests)
- Error handling (partial failures)
- Context cancellation
- Worker pool limits

**DoD:**
- CurateBatch() working
- 90% coverage on parallel.go
- Benchmarks show 8x speedup

#### Day 2: Thread Safety & Benchmarks
**Tasks:**
1. Verify thread safety with race detector
2. Add concurrent Curate() tests
3. Performance benchmarks
4. Memory profiling

**Tests:**
- Concurrent Curate() from multiple goroutines
- Race detector clean
- Memory bounds respected

**DoD:**
- All tests pass with `-race`
- Benchmarks documented
- Memory overhead < 10MB

---

### Phase 2: Refinement Strategies (Days 3-5)

#### Day 3: Refinement Interface & NoRefine Strategy
**Tasks:**
1. Define RefinementStrategy interface
2. Implement noRefinementStrategy
3. Update curator struct
4. Add WithRefinementMode() option

**Tests:**
- NoRefine never triggers
- Backward compatibility (default = NoRefine)

**DoD:**
- Interface defined
- NoRefine working
- Tests pass

#### Day 4: Proactive Refinement
**Tasks:**
1. Implement proactiveRefinementStrategy
2. Add ShouldRefine() trigger logic
3. Add Refine() pruning logic
4. Integrate with Curate()

**Tests:**
- Trigger at max bullets threshold
- Prune low-utility bullets
- Stats correct
- No false triggers

**DoD:**
- Proactive refinement working
- Automatic pruning at threshold
- 90% coverage

#### Day 5: Lazy Refinement
**Tasks:**
1. Implement lazyRefinementStrategy
2. Add explicit Refine() method
3. Test manual refinement
4. Integration tests

**Tests:**
- Lazy never auto-refines
- Manual Refine() works
- Stats correct

**DoD:**
- Lazy refinement working
- Explicit control
- 90% coverage

---

### Phase 3: Integration & Polish (Day 6)

**Tasks:**
1. Integration tests (Reflector → Curator → Refinement)
2. Documentation updates
3. Example usage in ace.md
4. Benchmark all strategies

**Tests:**
- End-to-end with refinement
- Performance benchmarks
- Memory profiling

**DoD:**
- All integration tests pass
- Documentation complete
- Examples working

---

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥ 90%

**Test Categories:**
1. Parallel processing (worker pool, batch)
2. Refinement strategies (proactive, lazy, none)
3. Thread safety (concurrent access)
4. Error handling (partial failures)
5. Configuration (options, defaults)

**Key Test Cases:**
```go
// Parallel
func TestCurator_CurateBatch_Sequential(t *testing.T)
func TestCurator_CurateBatch_Parallel(t *testing.T)
func TestCurator_CurateBatch_ErrorHandling(t *testing.T)
func TestCurator_CurateBatch_ContextCancellation(t *testing.T)

// Refinement
func TestRefinement_NoRefine(t *testing.T)
func TestRefinement_Proactive_Trigger(t *testing.T)
func TestRefinement_Proactive_Prune(t *testing.T)
func TestRefinement_Lazy_ManualOnly(t *testing.T)
func TestRefinement_Lazy_Refine(t *testing.T)

// Thread Safety
func TestCurator_ConcurrentCurate(t *testing.T)
```

### Benchmarks

```go
func BenchmarkCurator_CurateBatch_Sequential(b *testing.B)
func BenchmarkCurator_CurateBatch_Parallel2(b *testing.B)
func BenchmarkCurator_CurateBatch_Parallel8(b *testing.B)
func BenchmarkRefinement_Proactive_ShouldRefine(b *testing.B)
func BenchmarkRefinement_Proactive_Refine(b *testing.B)
func BenchmarkRefinement_Lazy_Refine(b *testing.B)
```

### Integration Tests

```go
func TestIntegration_Reflector_Curator_Refinement(t *testing.T) {
    // Full workflow: trajectories → insights → bullets → refine
}

func TestIntegration_Parallel_Refinement(t *testing.T) {
    // Parallel batch curation + proactive refinement
}
```

---

## Success Criteria

### Functional Success
- ✅ Parallel batch processing working
- ✅ Proactive refinement triggers correctly
- ✅ Lazy refinement waits for explicit call
- ✅ All refinement strategies tested
- ✅ Thread-safe concurrent access

### Quality Success
- ✅ Test coverage ≥ 90%
- ✅ No lint errors
- ✅ Race detector clean
- ✅ Integration tests pass

### Performance Success
- ✅ Parallel: 8x speedup on 8-core CPU
- ✅ Refinement: < 50ms for 1000 bullets
- ✅ Memory: < 10MB overhead

---

## Dependencies

### Internal Dependencies
- ✅ `internal/ace/curator` (existing implementation)
- ✅ `internal/ace/bullet` (Score() method)
- ✅ `internal/ace/playbook` (Stats() method)
- ✅ `internal/ace/reflector` (Insight type)

### External Dependencies
- `runtime` (NumCPU for worker pool)
- `sync` (WaitGroup, Mutex)
- `context` (cancellation support)

---

## Risks and Mitigations

### Risk 1: Race Conditions in Parallel Processing
**Impact:** HIGH  
**Probability:** MEDIUM  
**Mitigation:**
- Use playbook's existing RWMutex
- Verify with `go test -race`
- Limit concurrent access to shared state
- Use channels for coordination

### Risk 2: Refinement Removes Useful Bullets
**Impact:** MEDIUM  
**Probability:** LOW  
**Mitigation:**
- Conservative default thresholds (0.1 score, 1000 bullets)
- Test with real data before aggressive pruning
- Log all pruned bullets for debugging
- Future: user approval for pruning

### Risk 3: Performance Overhead from Refinement
**Impact:** MEDIUM  
**Probability:** LOW  
**Mitigation:**
- Benchmark refinement operations
- Lazy mode available (no overhead)
- Configurable triggers (user control)
- Early exit if no pruning needed

---

## Future Enhancements (Post-MVP)

### Phase 5+
- Intelligent refinement (cluster similar bullets, merge)
- LLM-based quality scoring (beyond simple helpful/harmful)
- Archival system (move old bullets to archive, not delete)
- User approval UI for refinement actions
- Refinement history (undo pruning)
- Multi-level refinement (aggressive, moderate, gentle)

---

## Appendix A: Usage Examples

### Example 1: Parallel Batch Processing

```go
import "github.com/dmytrogajewski/spin/internal/ace/curator"

// Create curator with parallel support
cur := curator.NewCurator(pb, embedder,
    curator.WithMaxWorkers(8), // Use 8 workers
)

// Process multiple insight batches in parallel
batch := curator.BatchMergeRequest{
    Requests: []curator.MergeRequest{
        {Insights: insights1},
        {Insights: insights2},
        {Insights: insights3},
        // ... up to 100 batches
    },
    MaxWorkers: 8,
}

result, err := cur.CurateBatch(ctx, batch)

// Check results
for i, res := range result.Results {
    if result.Errors[i] != nil {
        log.Printf("Request %d failed: %v", i, result.Errors[i])
        continue
    }
    log.Printf("Request %d: added=%d, skipped=%d", i, res.Added, res.Skipped)
}
```

### Example 2: Proactive Refinement

```go
// Create curator with proactive refinement
cur := curator.NewCurator(pb, embedder,
    curator.WithRefinementMode(
        curator.RefinementModeProactive,
        curator.ProactiveRefinementConfig{
            MaxBullets:      500,  // Refine at 500 bullets
            MinUtilityScore: 0.2,  // Remove bullets with score < 0.2
        },
    ),
)

// Curate insights
result, err := cur.Curate(ctx, curator.MergeRequest{Insights: insights})

// Check if refinement happened
if result.Refined {
    log.Printf("Refinement triggered: pruned %d bullets", result.Refinement.Pruned)
    log.Printf("Reason: %s", result.Refinement.Reason)
}
```

### Example 3: Lazy Refinement

```go
// Create curator with lazy refinement
cur := curator.NewCurator(pb, embedder,
    curator.WithRefinementMode(
        curator.RefinementModeLazy,
        curator.LazyRefinementConfig{
            MinUtilityScore: 0.1,
        },
    ),
)

// Curate insights (no auto-refinement)
result1, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights1})
result2, _ := cur.Curate(ctx, curator.MergeRequest{Insights: insights2})

// Manually trigger refinement when desired
refinement, err := cur.Refine(ctx)
if err != nil {
    log.Fatal(err)
}

log.Printf("Manual refinement: pruned %d bullets", refinement.Pruned)
for _, id := range refinement.PrunedIDs {
    log.Printf("  Removed: %s", id)
}
```

---

**Document Status:** Ready for Implementation  
**Next Steps:** Begin Phase 1 - Parallel Delta Merging  
**Estimated Completion:** 6 days from start
