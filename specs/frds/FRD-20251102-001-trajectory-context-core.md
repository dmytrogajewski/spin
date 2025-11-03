# FRD-20251102-001: TrajectoryContext Core Data Structure

**Feature:** Progressive Trajectory Context - Core Data Structure  
**Status:** Implementation  
**Created:** 2025-11-02  
**Phase:** 1.1  
**Roadmap:** [ace-progressive-context/ROADMAP.md](../ace-progressive-context/ROADMAP.md)

---

## Overview

Implement the core `TrajectoryContext` data structure that serves as the single source of truth for execution state during agent loop. This context tracks steps, retrieval events, and bullet cache to enable dynamic retrieval and enriched Reflector input.

---

## Requirements

### Functional Requirements

**FR1: Context Lifecycle Management**
- Create new context with user query
- Generate unique session ID
- Track start time for duration calculations

**FR2: Step Tracking**
- Append execution steps progressively
- Maintain step order (FIFO)
- Support empty step list initialization

**FR3: Retrieval Event Recording**
- Record when retrieval occurred (turn number)
- Track why retrieval triggered (trigger type)
- Store query used for retrieval
- List bullet IDs added to cache
- Timestamp each retrieval event

**FR4: Bullet Caching**
- Cache bullets by ID (deduplication)
- Track bullet metadata:
  - Retrieved at turn
  - Access count
  - Last accessed turn
- Count cache hits vs misses

**FR5: Active Bullet Retrieval**
- Get bullets within TTL (10 turns default)
- Update last accessed time
- Return bullets in consistent order

**FR6: Trajectory Conversion**
- Convert TrajectoryContext → generator.Trajectory
- Include all steps
- Include all cached bullets
- Include retrieval events in metadata

### Non-Functional Requirements

**NFR1: Performance**
- AppendSteps: O(n) where n = new steps
- RecordRetrieval: O(m) where m = new bullets
- GetActiveBullets: O(k) where k = cached bullets
- All operations < 10ms for typical sizes

**NFR2: Thread Safety**
- Not required (single-threaded agent loop)
- Document non-thread-safe in godoc

**NFR3: Memory Efficiency**
- Cache size limited by max bullets (configurable)
- No memory leaks in long conversations

---

## Data Structures

### TrajectoryContext

```go
// TrajectoryContext is the progressive execution context built during agent loop.
// It serves as the SINGLE SOURCE OF TRUTH for:
// - Retrieval: Dynamic query building and bullet caching
// - Reflector: Rich trajectory with retrieval provenance
// - Agent: Execution state tracking
//
// NOT thread-safe. Must be used within single goroutine.
type TrajectoryContext struct {
    // Immutable (set at creation)
    Query     string    // Initial user query
    SessionID string    // Unique session identifier
    StartTime time.Time // Context creation time
    
    // Progressive state (updated each turn)
    Steps       []generator.TrajectoryStep
    CurrentTurn int
    Success     bool  // Set at end of execution
    
    // Retrieval management
    RetrievalEvents   []RetrievalEvent
    BulletCache       map[string]*CachedBullet
    LastRetrievalTurn int
    
    // Metrics
    TotalRetrievals int
    CacheHits       int
    CacheMisses     int
}
```

### RetrievalEvent

```go
// RetrievalEvent records when, why, and what bullets were retrieved.
// This enriches the trajectory for Reflector analysis.
type RetrievalEvent struct {
    Turn         int         // Turn when retrieval occurred
    Trigger      TriggerType // Why retrieval was triggered
    Query        string      // Query used for retrieval
    BulletsAdded []string    // Bullet IDs added to cache
    Timestamp    time.Time   // When retrieval occurred
}
```

### CachedBullet

```go
// CachedBullet tracks bullet usage and lifecycle.
type CachedBullet struct {
    Bullet       *bullet.Bullet // The bullet instance
    RetrievedAt  int            // Turn when first retrieved
    AccessCount  int            // Times included in LLM prompt
    LastAccessed int            // Last turn accessed
}
```

### TriggerType

```go
// TriggerType defines why retrieval was triggered.
type TriggerType string

const (
    TriggerInitial    TriggerType = "initial"      // Turn 0
    TriggerError      TriggerType = "error"        // Tool/LLM error
    TriggerToolChange TriggerType = "tool_change"  // Different tool
    TriggerInterval   TriggerType = "interval"     // Cache expired
)
```

---

## API Specification

### NewTrajectoryContext

```go
// NewTrajectoryContext creates a new progressive context.
// Generates unique session ID and initializes empty collections.
func NewTrajectoryContext(query string) *TrajectoryContext
```

**Behavior:**
- Generate UUID for session ID
- Set start time to current time
- Initialize empty steps slice
- Initialize empty retrieval events slice
- Initialize empty bullet cache map
- Set current turn to 0

### AppendSteps

```go
// AppendSteps adds new execution steps to context.
// Steps are appended in order (FIFO).
func (tc *TrajectoryContext) AppendSteps(steps []generator.TrajectoryStep)
```

**Behavior:**
- Append steps to existing steps slice
- Handle nil steps (no-op)
- Handle empty steps (no-op)

### RecordRetrieval

```go
// RecordRetrieval records a retrieval event and merges bullets into cache.
// Updates cache hits/misses metrics.
func (tc *TrajectoryContext) RecordRetrieval(event RetrievalEvent, bullets []*bullet.Bullet)
```

**Behavior:**
- Append event to retrieval events
- Update last retrieval turn
- Increment total retrievals
- For each bullet:
  - If already cached: increment access count, increment cache hits
  - If new: create CachedBullet entry, increment cache misses

### GetActiveBullets

```go
// GetActiveBullets returns bullets for LLM prompt (cache + TTL filtering).
// Updates last accessed time for returned bullets.
// TTL is hardcoded to 10 turns for now (will be configurable later).
func (tc *TrajectoryContext) GetActiveBullets() []*bullet.Bullet
```

**Behavior:**
- Filter cached bullets by TTL (current turn - retrieved at <= 10)
- Update last accessed time to current turn
- Return bullet instances (not CachedBullet wrappers)
- Return in deterministic order (sorted by bullet ID)

### ToTrajectory

```go
// ToTrajectory converts context to Trajectory for Reflector.
// Includes all steps, bullets, and retrieval events.
func (tc *TrajectoryContext) ToTrajectory() *generator.Trajectory
```

**Behavior:**
- Create Trajectory with session ID
- Copy query, steps, success
- Extract all bullets from cache (regardless of TTL)
- Include retrieval events in metadata
- Calculate duration from start time

---

## Test Strategy

### Unit Tests

**Test Coverage Target: 95%+**

1. **TestNewTrajectoryContext**
   - Verify session ID generated (not empty)
   - Verify start time set
   - Verify query stored
   - Verify empty collections initialized

2. **TestAppendSteps**
   - Single step append
   - Multiple steps append
   - Nil steps (no-op)
   - Empty steps (no-op)
   - Order preservation

3. **TestRecordRetrieval**
   - First retrieval (all cache misses)
   - Duplicate bullets (cache hits)
   - Mixed new and cached bullets
   - Metrics updated correctly

4. **TestGetActiveBullets**
   - Fresh bullets (within TTL)
   - Expired bullets (beyond TTL)
   - Mixed fresh and expired
   - Last accessed updated
   - Deterministic ordering

5. **TestToTrajectory**
   - Empty context
   - Context with steps only
   - Context with bullets only
   - Context with retrieval events
   - Full context (steps + bullets + events)

### Benchmarks

```go
func BenchmarkAppendSteps(b *testing.B)
func BenchmarkRecordRetrieval(b *testing.B)
func BenchmarkGetActiveBullets(b *testing.B)
func BenchmarkToTrajectory(b *testing.B)
```

**Performance Targets:**
- AppendSteps: < 1ms for 100 steps
- RecordRetrieval: < 5ms for 10 bullets
- GetActiveBullets: < 2ms for 100 cached bullets
- ToTrajectory: < 10ms for 200-turn context

---

## Implementation Notes

### Package Location
- `internal/ace/trajectory/context.go`
- `internal/ace/trajectory/context_test.go`
- `internal/ace/trajectory/types.go`

### Dependencies
- `github.com/dmytrogajewski/spin/internal/ace/bullet`
- `github.com/dmytrogajewski/spin/internal/ace/generator`
- `github.com/google/uuid` (for session ID)

### Session ID Generation
Use `uuid.New().String()` for unique session IDs.

### TTL Hardcoding
For this feature, TTL is hardcoded to 10 turns. Configuration will be added in Feature 4.1.

### Deterministic Ordering
`GetActiveBullets()` must return bullets in deterministic order to ensure test reproducibility. Sort by bullet ID.

---

## Acceptance Criteria

- [x] All unit tests pass ✅
- [x] Test coverage ≥ 95% (achieved 100%) ✅
- [x] Race detector clean (`go test -race`) ✅
- [x] Linter clean (`go vet` + `go fmt`) ✅
- [x] Benchmarks meet performance targets ✅
- [x] Godoc comments complete ✅
- [x] No TODOs in production code ✅

---

## Definition of Done

- [x] All types defined with godoc comments ✅
- [x] All core methods implemented ✅
- [x] Unit tests written (100% coverage) ✅
- [x] Benchmarks written and passing ✅
- [x] Race detector passes ✅
- [x] Linter clean ✅
- [x] Code reviewed (self-reviewed during TDD) ✅
- [x] Documentation updated (FRD + Roadmap) ✅
- [x] Roadmap item closed ✅

**Completion Date:** 2025-11-02
**Implementation Notes:**
- Types consolidated into context.go per Go best practices (no separate types.go)
- Used interface{} for RetrievalEvents in TrajectoryMetadata to avoid import cycle
- Achieved 100% test coverage (exceeded 95% target)
- All tests pass with race detector
- go vet and go fmt clean

---

## Follow-Up Features

- Feature 1.2: Context Helper Methods (error detection, tool extraction)
- Feature 1.3: Trajectory Metadata Extension
- Feature 4.1: Configuration System (configurable TTL)
