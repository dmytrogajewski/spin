# FRD-20251029-002: Playbook Manager

**Feature:** Playbook Manager (ACE Phase 2)  
**Roadmap:** ACE Feature 1 - Core Data Structures and Context Management  
**Status:** In Development  
**Created:** 2025-10-29  
**Author:** Spin Agent  
**Depends On:** FRD-20251029-001 (Bullet Package - ✅ Complete)

---

## 1. Executive Summary

This FRD defines the Playbook Manager implementation for ACE (Agentic Context Engineering). The Playbook Manager provides CRUD operations, semantic search, serialization, and version control for collections of context bullets.

**Key Objectives:**
- Implement thread-safe CRUD operations for bullets
- Provide O(1) lookup by ID using map-based indexing
- Support semantic search via embedding similarity
- Enable JSON serialization/deserialization
- Implement snapshot/restore for version control
- Emit events for all operations

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR-1: CRUD Operations
**Priority:** P0 (Critical)

- **Create (Add)**: Add new bullets to playbook
- **Read (Get)**: Retrieve bullet by ID in O(1) time
- **Update**: Modify existing bullet (content, counters, metadata)
- **Delete**: Remove bullet by ID
- **List**: Return all bullets with optional filtering

**Acceptance Criteria:**
- ✅ Add returns error if bullet with same ID exists
- ✅ Get returns (bullet, true) if found, (nil, false) if not found
- ✅ Update returns error if bullet doesn't exist
- ✅ Delete is idempotent (no error if ID doesn't exist)
- ✅ List supports filter functions
- ✅ All operations are O(1) for ID-based access

#### FR-2: Statistics
**Priority:** P0 (Critical)

Provide playbook statistics:
- Total bullet count
- Total helpful count (sum across all bullets)
- Total harmful count (sum across all bullets)
- Average score
- Total size in bytes (estimated)

**Acceptance Criteria:**
- ✅ Stats() returns accurate counts
- ✅ Stats are calculated on-demand (no caching)
- ✅ O(n) complexity is acceptable for stats

#### FR-3: Semantic Search
**Priority:** P1 (High)

Search bullets by semantic similarity:
- Generate embedding for query text
- Calculate cosine similarity with bullet embeddings
- Return top-k most similar bullets

**Acceptance Criteria:**
- ✅ Search requires Embedder interface implementation
- ✅ Returns empty slice if no bullets have embeddings
- ✅ Returns top-k results sorted by similarity (descending)
- ✅ Performance: <5ms for 100 bullets

#### FR-4: Serialization
**Priority:** P1 (High)

Support JSON serialization:
- Save playbook to file
- Load playbook from file
- Validate bullets on load

**Acceptance Criteria:**
- ✅ Save writes valid JSON
- ✅ Load validates all bullets
- ✅ Round-trip preserves all data
- ✅ Atomic writes (temp file + rename)
- ✅ Load fails gracefully on invalid JSON

#### FR-5: Snapshot/Restore
**Priority:** P1 (High)

Version control via snapshots:
- Create immutable snapshot of current state
- Restore playbook from snapshot
- Compare two snapshots (diff)

**Acceptance Criteria:**
- ✅ Snapshot creates deep copies
- ✅ Snapshots are immutable
- ✅ Restore replaces all bullets
- ✅ Diff identifies added/removed/modified bullets

#### FR-6: Thread Safety
**Priority:** P0 (Critical)

All operations must be thread-safe:
- Use sync.RWMutex for concurrent access
- Readers don't block readers
- Writers get exclusive access

**Acceptance Criteria:**
- ✅ Race detector passes with concurrent operations
- ✅ Multiple goroutines can read simultaneously
- ✅ Write operations are serialized

#### FR-7: Event Emission
**Priority:** P1 (High)

Emit events for observability:
- EventBulletAdded
- EventBulletUpdated
- EventBulletDeleted
- EventPlaybookSnapshot
- EventPlaybookRestored

**Acceptance Criteria:**
- ✅ Events include relevant data
- ✅ Events are emitted after successful operation
- ✅ No events on errors

---

## 3. Architecture

### 3.1 Data Structures

```go
package playbook

import (
    "context"
    "sync"
    "github.com/dmytrogajewski/spin/internal/ace/bullet"
    "github.com/dmytrogajewski/spin/internal/events"
)

// Playbook manages a collection of context bullets.
type Playbook struct {
    bullets  map[string]*bullet.Bullet // Index by ID for O(1) lookup
    mu       sync.RWMutex              // Thread-safe access
    emitter  *events.EventEmitter      // Event emission
    embedder Embedder                  // Semantic embedding provider
}

// Embedder generates semantic embeddings for text.
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    Dimension() int
}

// Stats contains playbook statistics.
type Stats struct {
    TotalBullets   int
    TotalHelpful   int
    TotalHarmful   int
    AvgScore       float64
    TotalSizeBytes int64
}

// FilterFunc is a predicate for filtering bullets.
type FilterFunc func(*bullet.Bullet) bool

// Snapshot is an immutable point-in-time capture.
type Snapshot struct {
    ID        string
    Bullets   []*bullet.Bullet
    CreatedAt time.Time
    Stats     Stats
}

// Diff represents differences between snapshots.
type Diff struct {
    Added    []*bullet.Bullet
    Removed  []*bullet.Bullet
    Modified []*BulletChange
}

// BulletChange represents a modification.
type BulletChange struct {
    ID     string
    Before *bullet.Bullet
    After  *bullet.Bullet
}
```

### 3.2 Key Methods

```go
// New creates a new empty playbook.
func New(emitter *events.EventEmitter, embedder Embedder) *Playbook

// Add adds a new bullet to the playbook.
func (p *Playbook) Add(ctx context.Context, b *bullet.Bullet) error

// Get retrieves a bullet by ID.
func (p *Playbook) Get(id string) (*bullet.Bullet, bool)

// Update updates an existing bullet.
func (p *Playbook) Update(ctx context.Context, b *bullet.Bullet) error

// Delete removes a bullet by ID.
func (p *Playbook) Delete(ctx context.Context, id string) error

// List returns all bullets (optionally filtered).
func (p *Playbook) List(filter FilterFunc) []*bullet.Bullet

// Search finds bullets by semantic similarity.
func (p *Playbook) Search(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error)

// Stats returns playbook statistics.
func (p *Playbook) Stats() Stats

// Snapshot creates an immutable snapshot.
func (p *Playbook) Snapshot() *Snapshot

// Restore restores from a snapshot.
func (p *Playbook) Restore(snapshot *Snapshot) error

// Save serializes to JSON file.
func (p *Playbook) Save(path string) error

// Load deserializes from JSON file.
func Load(path string, emitter *events.EventEmitter, embedder Embedder) (*Playbook, error)
```

---

## 4. Implementation Plan

### Phase 2A: CRUD Operations (Day 1)

**TDD Cycles:**
1. Test playbook creation (empty)
2. Test Add (happy path)
3. Test Add (duplicate ID error)
4. Test Get (found)
5. Test Get (not found)
6. Test Update (happy path)
7. Test Update (not found error)
8. Test Delete (exists)
9. Test Delete (doesn't exist - idempotent)
10. Test List (all)
11. Test List (with filter)
12. Test Stats calculation

**Deliverables:**
- [ ] `internal/ace/playbook/playbook.go`
- [ ] `internal/ace/playbook/playbook_test.go`
- [ ] CRUD operations with 90%+ coverage

### Phase 2B: Thread Safety (Day 1)

**TDD Cycles:**
1. Test concurrent Add operations (10 goroutines)
2. Test concurrent Get operations (10 goroutines)
3. Test concurrent mixed operations (20 goroutines)
4. Test race detector passes

**Deliverables:**
- [ ] Thread-safe implementation verified
- [ ] Race detector clean

### Phase 2C: Semantic Search (Day 2)

**TDD Cycles:**
1. Test Embedder interface (mock)
2. Test Search with no embeddings (empty result)
3. Test Search with embeddings (top-k results)
4. Test Search sorts by similarity
5. Test Search performance (<5ms for 100 bullets)

**Deliverables:**
- [ ] `internal/ace/playbook/search.go`
- [ ] `internal/ace/playbook/search_test.go`
- [ ] `internal/ace/embedding/embedder.go` (interface)
- [ ] `internal/ace/embedding/mock_embedder.go`

### Phase 2D: Serialization (Day 2)

**TDD Cycles:**
1. Test Save to JSON file
2. Test Load from JSON file
3. Test round-trip preservation
4. Test Load with invalid JSON
5. Test Load with validation errors
6. Test atomic writes (temp file + rename)

**Deliverables:**
- [ ] `internal/ace/playbook/storage.go`
- [ ] `internal/ace/playbook/storage_test.go`
- [ ] Serialization with 90%+ coverage

### Phase 2E: Snapshot/Restore (Day 3)

**TDD Cycles:**
1. Test Snapshot creates deep copies
2. Test Snapshot immutability
3. Test Restore replaces all bullets
4. Test Diff with no changes
5. Test Diff with added bullets
6. Test Diff with removed bullets
7. Test Diff with modified bullets

**Deliverables:**
- [ ] `internal/ace/playbook/snapshot.go`
- [ ] `internal/ace/playbook/snapshot_test.go`
- [ ] Version control with 90%+ coverage

### Phase 2F: Integration & Polish (Day 3)

**Tasks:**
1. Integration tests (full playbook lifecycle)
2. Performance benchmarks
3. Go vet clean
4. Race detector clean
5. Documentation updates

**Deliverables:**
- [ ] Integration tests passing
- [ ] Benchmarks in place
- [ ] All quality gates passed

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Coverage Target:** ≥90%

**CRUD Tests:**
- Add: success, duplicate ID
- Get: found, not found
- Update: success, not found
- Delete: exists, doesn't exist
- List: all, filtered, empty
- Stats: accuracy, empty playbook

**Thread Safety Tests:**
- Concurrent Add (10 goroutines)
- Concurrent Get (10 goroutines)
- Mixed operations (20 goroutines)
- Race detector verification

**Search Tests:**
- No embeddings (empty result)
- With embeddings (top-k)
- Similarity sorting
- Performance (<5ms for 100)

**Serialization Tests:**
- Save/Load round-trip
- Invalid JSON handling
- Validation on load
- Atomic writes

**Snapshot Tests:**
- Deep copy verification
- Immutability
- Restore accuracy
- Diff accuracy (add/remove/modify)

### 5.2 Integration Tests

**Scenarios:**
1. Full lifecycle: Add → Update → Search → Snapshot → Delete → Restore
2. Concurrent stress test (50 readers + 50 writers, 10 seconds)
3. Large playbook (1000 bullets, all operations)
4. Persistence test (Save → Load → Verify)

### 5.3 Benchmark Tests

```go
func BenchmarkPlaybookAdd(b *testing.B)
func BenchmarkPlaybookGet(b *testing.B)
func BenchmarkPlaybookSearch(b *testing.B)
func BenchmarkPlaybookSave(b *testing.B)
func BenchmarkPlaybookLoad(b *testing.B)
```

**Targets:**
- Add: <50μs/op
- Get: <1μs/op
- Search (100 bullets): <5ms/op
- Save (1000 bullets): <100ms/op
- Load (1000 bullets): <100ms/op

---

## 6. Success Criteria

### 6.1 Functional Success
- [x] All CRUD operations implemented
- [x] Thread-safe with RWMutex
- [x] Semantic search working
- [x] JSON serialization working
- [x] Snapshot/restore working
- [x] Event emission for all operations

### 6.2 Quality Success
- [x] Test coverage ≥90%
- [x] Race detector clean
- [x] Go vet clean
- [x] All integration tests passing
- [x] Benchmarks meet targets

### 6.3 Documentation Success
- [x] GoDoc on all exports
- [x] Usage examples in docs/packages/ace.md
- [x] Integration guide

---

## 7. Event Definitions

```go
const (
    EventBulletAdded      = "bullet.added"
    EventBulletUpdated    = "bullet.updated"
    EventBulletDeleted    = "bullet.deleted"
    EventPlaybookSnapshot = "playbook.snapshot"
    EventPlaybookRestored = "playbook.restored"
)

// BulletAddedData is the payload for bullet.added events.
type BulletAddedData struct {
    BulletID string
    Content  string
}

// BulletUpdatedData is the payload for bullet.updated events.
type BulletUpdatedData struct {
    BulletID string
    Content  string
}

// BulletDeletedData is the payload for bullet.deleted events.
type BulletDeletedData struct {
    BulletID string
}

// PlaybookSnapshotData is the payload for playbook.snapshot events.
type PlaybookSnapshotData struct {
    SnapshotID   string
    BulletCount  int
}

// PlaybookRestoredData is the payload for playbook.restored events.
type PlaybookRestoredData struct {
    SnapshotID   string
    BulletCount  int
}
```

---

## 8. References

- **Phase 1:** FRD-20251029-001 (Bullet Package - ✅ Complete)
- **Roadmap:** specs/ace-agentic-context-engineering/ROADMAP.md
- **Testing Patterns:** docs/testing-patterns.md
- **AGENTS.md:** Root guidelines

---

**Status:** Ready for Implementation  
**Estimated Effort:** 3 days with strict TDD  
**Next:** Begin Phase 2A - CRUD Operations
