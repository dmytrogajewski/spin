# FRD-20251030-007: Incremental Delta Updates System

**Feature ID**: 5 (ACE Integration Roadmap)  
**Created**: 2025-10-30  
**Status**: Implementation  
**Component**: `internal/ace/delta`

---

## Executive Summary

Implement the incremental delta update mechanism that replaces costly monolithic bullet rewrites with localized, versioned edits. This system tracks changes per adaptation step, maintains a complete delta history with rollback capabilities, and provides efficient batch processing for high-throughput scenarios.

**Problem**: Currently, the ACE system can only add new bullets or update entire bullets. There is no mechanism to track incremental changes (deltas) to bullets over time, making it difficult to understand how bullets evolved, roll back problematic changes, or efficiently apply localized updates.

**Solution**: Introduce a Delta tracking system that:
1. Records fine-grained changes to individual bullets (content updates, counter adjustments, tag modifications)
2. Maintains a versioned history of all deltas for audit and rollback
3. Supports efficient batch processing of multiple deltas in parallel
4. Enables localized updates (modify only affected fields) instead of full bullet replacements
5. Provides rollback capabilities to undo recent changes

---

## Goals

### Primary Goals
1. **Delta Tracking**: Record all changes to bullets as discrete delta objects
2. **History Management**: Maintain complete delta history with timestamps and metadata
3. **Rollback Support**: Enable undoing recent changes by replaying inverse deltas
4. **Localized Updates**: Apply changes only to affected fields, preserving unchanged state
5. **Batch Processing**: Process multiple deltas efficiently with parallelization

### Secondary Goals
1. **Integration**: Seamless integration with existing Curator and Playbook
2. **Observability**: Events for all delta operations
3. **Performance**: Sub-millisecond delta application for individual changes
4. **Storage**: Efficient delta serialization and persistence

### Non-Goals
1. **Distributed Synchronization**: Not solving multi-process playbook sync
2. **Conflict Resolution**: Assumes single-writer for now (future: multi-writer CRDTs)
3. **Schema Migration**: Not handling bullet schema evolution
4. **UI Integration**: No TUI visualization (future feature)

---

## Technical Design

### Data Structures

#### Delta

```go
// Delta represents a single change to a bullet
type Delta struct {
    // ID is the unique identifier for this delta
    ID string `json:"id"`
    
    // BulletID is the ID of the bullet being changed
    BulletID string `json:"bullet_id"`
    
    // Operation is the type of change
    Operation DeltaOperation `json:"operation"`
    
    // Fields contains the changes (field name → new value)
    Fields map[string]interface{} `json:"fields"`
    
    // Metadata contains contextual information
    Metadata DeltaMetadata `json:"metadata"`
    
    // CreatedAt is when the delta was created
    CreatedAt time.Time `json:"created_at"`
}

type DeltaOperation string

const (
    OpUpdateContent     DeltaOperation = "update_content"     // Change bullet content
    OpIncrementHelpful  DeltaOperation = "increment_helpful"  // Increment helpful count
    OpIncrementHarmful  DeltaOperation = "increment_harmful"  // Increment harmful count
    OpAddTag            DeltaOperation = "add_tag"            // Add/update a tag
    OpRemoveTag         DeltaOperation = "remove_tag"         // Remove a tag
    OpUpdateEmbedding   DeltaOperation = "update_embedding"   // Update semantic embedding
)

type DeltaMetadata struct {
    Source    string            `json:"source"`     // "reflector", "curator", "adapter", "manual"
    SessionID string            `json:"session_id"` // Adaptation session ID (if applicable)
    Reason    string            `json:"reason"`     // Human-readable reason for change
    Tags      map[string]string `json:"tags"`       // Arbitrary metadata
}
```

#### DeltaHistory

```go
// DeltaHistory manages versioned delta records
type DeltaHistory struct {
    deltas   []Delta          // Ordered list of deltas (append-only)
    byBullet map[string][]int // Index: bulletID → delta indices
    mu       sync.RWMutex     // Thread-safe access
}

// DeltaHistoryStats contains history statistics
type DeltaHistoryStats struct {
    TotalDeltas      int
    UniqueBullets    int
    OldestDelta      time.Time
    NewestDelta      time.Time
    DeltasByOperation map[DeltaOperation]int
}
```

#### DeltaApplier

```go
// DeltaApplier applies deltas to bullets in a playbook
type DeltaApplier struct {
    playbook *playbook.Playbook
    history  *DeltaHistory
}

// ApplyResult contains the result of applying a delta
type ApplyResult struct {
    Success     bool
    DeltaID     string
    BulletID    string
    OldValue    interface{} // Previous value before change
    NewValue    interface{} // New value after change
    Error       error
    AppliedAt   time.Time
}

// BatchApplyRequest contains multiple deltas to apply
type BatchApplyRequest struct {
    Deltas     []Delta
    MaxWorkers int // 0 = runtime.NumCPU()
    Atomic     bool // If true, rollback all if any fails
}

// BatchApplyResult contains results for batch application
type BatchApplyResult struct {
    Results     []ApplyResult
    Applied     int // Number successfully applied
    Failed      int // Number failed
    RolledBack  bool // True if atomic=true and rollback occurred
}
```

#### RollbackManager

```go
// RollbackManager handles delta rollbacks
type RollbackManager struct {
    applier *DeltaApplier
    history *DeltaHistory
}

// RollbackRequest specifies what to rollback
type RollbackRequest struct {
    // One of the following must be set:
    DeltaID   string    // Rollback specific delta
    BulletID  string    // Rollback all deltas for bullet
    Since     time.Time // Rollback all deltas since time
    Count     int       // Rollback last N deltas
}

// RollbackResult contains the result of a rollback
type RollbackResult struct {
    RolledBack int       // Number of deltas rolled back
    DeltaIDs   []string  // IDs of rolled back deltas
    Errors     []error   // Errors encountered (partial success possible)
}
```

### Component Architecture

```
internal/ace/delta/
├── delta.go              # Delta data structure and operations
├── delta_test.go         # Delta unit tests
├── history.go            # DeltaHistory implementation
├── history_test.go       # History unit tests
├── applier.go            # DeltaApplier implementation
├── applier_test.go       # Applier unit tests
├── rollback.go           # RollbackManager implementation
├── rollback_test.go      # Rollback unit tests
├── batch.go              # Batch processing with worker pool
├── batch_test.go         # Batch processing tests
├── integration_test.go   # Integration tests with Playbook
└── delta_bench_test.go   # Performance benchmarks
```

### Core Algorithms

#### Delta Application

```go
// Apply a single delta to the playbook
func (a *DeltaApplier) Apply(ctx context.Context, delta Delta) (*ApplyResult, error) {
    // 1. Get bullet from playbook
    bullet, exists := a.playbook.Get(delta.BulletID)
    if !exists {
        return nil, fmt.Errorf("bullet %s not found", delta.BulletID)
    }
    
    // 2. Create clone for modification (copy-on-write)
    modified := bullet.Clone()
    
    // 3. Apply delta based on operation
    oldValue, newValue, err := applyDeltaOperation(modified, delta)
    if err != nil {
        return &ApplyResult{Success: false, Error: err}, err
    }
    
    // 4. Update bullet in playbook
    if err := a.playbook.Update(ctx, modified); err != nil {
        return &ApplyResult{Success: false, Error: err}, err
    }
    
    // 5. Record delta in history
    a.history.Record(delta)
    
    return &ApplyResult{
        Success:   true,
        DeltaID:   delta.ID,
        BulletID:  delta.BulletID,
        OldValue:  oldValue,
        NewValue:  newValue,
        AppliedAt: time.Now(),
    }, nil
}

// Apply delta operation to a bullet (copy-on-write)
func applyDeltaOperation(b *bullet.Bullet, delta Delta) (oldValue, newValue interface{}, err error) {
    switch delta.Operation {
    case OpUpdateContent:
        oldValue = b.Content
        b.Content = delta.Fields["content"].(string)
        newValue = b.Content
        
    case OpIncrementHelpful:
        oldValue = b.HelpfulCount
        b.IncrementHelpful()
        newValue = b.HelpfulCount
        
    case OpIncrementHarmful:
        oldValue = b.HarmfulCount
        b.IncrementHarmful()
        newValue = b.HarmfulCount
        
    case OpAddTag:
        key := delta.Fields["key"].(string)
        value := delta.Fields["value"].(string)
        if b.Tags == nil {
            b.Tags = make(map[string]string)
        }
        oldValue = b.Tags[key]
        b.Tags[key] = value
        newValue = value
        
    case OpRemoveTag:
        key := delta.Fields["key"].(string)
        oldValue = b.Tags[key]
        delete(b.Tags, key)
        newValue = nil
        
    case OpUpdateEmbedding:
        oldValue = b.Embedding
        b.Embedding = delta.Fields["embedding"].([]float32)
        newValue = b.Embedding
        
    default:
        return nil, nil, fmt.Errorf("unknown operation: %s", delta.Operation)
    }
    
    // Update timestamp
    b.UpdatedAt = time.Now()
    
    return oldValue, newValue, nil
}
```

#### Rollback

```go
// Rollback deltas based on request
func (r *RollbackManager) Rollback(ctx context.Context, req RollbackRequest) (*RollbackResult, error) {
    // 1. Identify deltas to rollback
    deltas := r.identifyDeltasToRollback(req)
    
    // 2. Sort in reverse chronological order (undo newest first)
    sort.Slice(deltas, func(i, j int) bool {
        return deltas[i].CreatedAt.After(deltas[j].CreatedAt)
    })
    
    // 3. Create inverse deltas
    inverseDeltas := make([]Delta, len(deltas))
    for i, delta := range deltas {
        inverseDeltas[i] = r.createInverseDelta(delta)
    }
    
    // 4. Apply inverse deltas
    result := &RollbackResult{
        DeltaIDs: make([]string, 0, len(deltas)),
    }
    
    for _, inverseDelta := range inverseDeltas {
        applyResult, err := r.applier.Apply(ctx, inverseDelta)
        if err != nil {
            result.Errors = append(result.Errors, err)
            continue
        }
        if applyResult.Success {
            result.RolledBack++
            result.DeltaIDs = append(result.DeltaIDs, inverseDelta.BulletID)
        }
    }
    
    return result, nil
}

// Create inverse delta (undo operation)
func (r *RollbackManager) createInverseDelta(delta Delta) Delta {
    inverse := Delta{
        ID:        uuid.New().String(),
        BulletID:  delta.BulletID,
        CreatedAt: time.Now(),
        Metadata: DeltaMetadata{
            Source: "rollback",
            Reason: fmt.Sprintf("Rollback of delta %s", delta.ID),
        },
    }
    
    switch delta.Operation {
    case OpUpdateContent:
        // Get old value from apply result (would need to store this)
        inverse.Operation = OpUpdateContent
        inverse.Fields = map[string]interface{}{
            "content": delta.Metadata.Tags["old_content"], // Stored during apply
        }
        
    case OpIncrementHelpful:
        // Decrement by creating opposite operation
        inverse.Operation = "decrement_helpful" // Would need to add this op
        
    case OpIncrementHarmful:
        inverse.Operation = "decrement_harmful"
        
    case OpAddTag:
        inverse.Operation = OpRemoveTag
        inverse.Fields = map[string]interface{}{
            "key": delta.Fields["key"],
        }
        
    case OpRemoveTag:
        inverse.Operation = OpAddTag
        inverse.Fields = map[string]interface{}{
            "key":   delta.Fields["key"],
            "value": delta.Metadata.Tags["old_value"], // Stored during apply
        }
        
    case OpUpdateEmbedding:
        inverse.Operation = OpUpdateEmbedding
        inverse.Fields = map[string]interface{}{
            "embedding": delta.Metadata.Tags["old_embedding"], // Stored during apply
        }
    }
    
    return inverse
}
```

#### Batch Processing

```go
// Apply multiple deltas in parallel
func (a *DeltaApplier) ApplyBatch(ctx context.Context, req BatchApplyRequest) (*BatchApplyResult, error) {
    workers := req.MaxWorkers
    if workers == 0 {
        workers = runtime.NumCPU()
    }
    
    result := &BatchApplyResult{
        Results: make([]ApplyResult, len(req.Deltas)),
    }
    
    // Worker pool with error collection
    type job struct {
        index int
        delta Delta
    }
    
    jobs := make(chan job, len(req.Deltas))
    results := make(chan struct {
        index  int
        result *ApplyResult
        err    error
    }, len(req.Deltas))
    
    // Start workers
    var wg sync.WaitGroup
    for w := 0; w < workers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := range jobs {
                res, err := a.Apply(ctx, j.delta)
                results <- struct {
                    index  int
                    result *ApplyResult
                    err    error
                }{j.index, res, err}
            }
        }()
    }
    
    // Submit jobs
    for i, delta := range req.Deltas {
        jobs <- job{i, delta}
    }
    close(jobs)
    
    // Collect results
    go func() {
        wg.Wait()
        close(results)
    }()
    
    for r := range results {
        if r.err != nil {
            result.Failed++
            result.Results[r.index] = ApplyResult{Success: false, Error: r.err}
            
            // Atomic mode: rollback all if any fails
            if req.Atomic {
                result.RolledBack = true
                // Rollback logic here...
                return result, fmt.Errorf("atomic batch failed, rolled back")
            }
        } else {
            result.Applied++
            result.Results[r.index] = *r.result
        }
    }
    
    return result, nil
}
```

### Integration Points

#### Curator Integration

```go
// Curator generates deltas instead of directly modifying bullets
type Curator struct {
    // ... existing fields
    deltaApplier *delta.DeltaApplier
}

// Curate with delta tracking
func (c *Curator) Curate(ctx context.Context, req MergeRequest) (*MergeResult, error) {
    // ... existing logic to convert insights to bullets
    
    // Instead of playbook.Add(), create deltas
    deltas := make([]delta.Delta, 0)
    
    for _, bullet := range newBullets {
        // Check for duplicates
        duplicateID, isDuplicate := c.findDuplicate(ctx, bullet)
        
        if isDuplicate {
            // Create delta to increment helpful count
            deltas = append(deltas, delta.Delta{
                ID:        uuid.New().String(),
                BulletID:  duplicateID,
                Operation: delta.OpIncrementHelpful,
                Fields:    map[string]interface{}{},
                Metadata: delta.DeltaMetadata{
                    Source: "curator",
                    Reason: "Duplicate insight detected",
                },
                CreatedAt: time.Now(),
            })
        } else {
            // Add new bullet (not a delta, direct add)
            c.playbook.Add(ctx, bullet)
        }
    }
    
    // Apply deltas in batch
    batchReq := delta.BatchApplyRequest{
        Deltas:     deltas,
        MaxWorkers: 4,
        Atomic:     false,
    }
    batchResult, err := c.deltaApplier.ApplyBatch(ctx, batchReq)
    
    // ... process results
}
```

#### Adapter Integration

```go
// Adapter uses deltas for online updates
type Adapter struct {
    // ... existing fields
    deltaApplier *delta.DeltaApplier
}

// AdaptOnline with delta tracking
func (a *Adapter) AdaptOnline(ctx context.Context, signal ExecutionSignal) (*AdaptationResult, error) {
    // ... existing decision logic
    
    if action == ActionQuickAdd {
        // Generate bullet from signal
        bullet := a.generateBulletFromSignal(signal)
        
        // Add as delta for tracking
        delta := delta.Delta{
            ID:        uuid.New().String(),
            BulletID:  bullet.ID,
            Operation: delta.OpUpdateContent, // New bullet, so all fields
            Fields: map[string]interface{}{
                "content": bullet.Content,
            },
            Metadata: delta.DeltaMetadata{
                Source:    "adapter",
                SessionID: signal.SessionID,
                Reason:    fmt.Sprintf("Quick add from %s signal", signal.SignalType),
            },
            CreatedAt: time.Now(),
        }
        
        result, err := a.deltaApplier.Apply(ctx, delta)
        // ... handle result
    }
}
```

---

## API Reference

### DeltaApplier

```go
// NewDeltaApplier creates a new delta applier
func NewDeltaApplier(pb *playbook.Playbook) *DeltaApplier

// Apply applies a single delta
func (a *DeltaApplier) Apply(ctx context.Context, delta Delta) (*ApplyResult, error)

// ApplyBatch applies multiple deltas in parallel
func (a *DeltaApplier) ApplyBatch(ctx context.Context, req BatchApplyRequest) (*BatchApplyResult, error)

// GetHistory returns the delta history
func (a *DeltaApplier) GetHistory() *DeltaHistory
```

### DeltaHistory

```go
// NewDeltaHistory creates a new delta history
func NewDeltaHistory() *DeltaHistory

// Record adds a delta to the history
func (h *DeltaHistory) Record(delta Delta)

// GetByBullet returns all deltas for a bullet
func (h *DeltaHistory) GetByBullet(bulletID string) []Delta

// GetRecent returns the N most recent deltas
func (h *DeltaHistory) GetRecent(count int) []Delta

// GetSince returns all deltas since a timestamp
func (h *DeltaHistory) GetSince(since time.Time) []Delta

// Stats returns history statistics
func (h *DeltaHistory) Stats() DeltaHistoryStats

// Clear removes all history (use with caution)
func (h *DeltaHistory) Clear()
```

### RollbackManager

```go
// NewRollbackManager creates a new rollback manager
func NewRollbackManager(applier *DeltaApplier) *RollbackManager

// Rollback rolls back deltas based on request
func (r *RollbackManager) Rollback(ctx context.Context, req RollbackRequest) (*RollbackResult, error)

// CanRollback checks if a rollback is possible
func (r *RollbackManager) CanRollback(req RollbackRequest) (bool, error)
```

---

## Performance Targets

| Operation | Target | Rationale |
|-----------|--------|-----------|
| Single delta apply | < 100μs | Fast enough for online adaptation |
| Batch apply (100 deltas) | < 10ms | Efficient bulk processing |
| History lookup by bullet | < 1ms | Quick audit trail access |
| Rollback (single delta) | < 200μs | 2x apply time (inverse + apply) |
| Rollback (10 deltas) | < 2ms | Linear scaling with count |
| Memory overhead per delta | < 500 bytes | Reasonable for 10K+ deltas |

---

## Testing Strategy

### Unit Tests (90%+ Coverage)

1. **Delta Creation and Validation**
   - Valid delta operations
   - Invalid operations (unknown type)
   - Field validation per operation type

2. **DeltaHistory**
   - Record and retrieve deltas
   - GetByBullet correctness
   - GetRecent ordering
   - GetSince filtering
   - Thread-safety with concurrent writes

3. **DeltaApplier**
   - Apply each operation type
   - Error handling (bullet not found, invalid fields)
   - Copy-on-write semantics
   - Old/new value tracking
   - Integration with playbook.Update

4. **RollbackManager**
   - Single delta rollback
   - Bullet rollback (all deltas)
   - Time-based rollback
   - Count-based rollback
   - Inverse delta generation
   - Partial rollback on errors

5. **Batch Processing**
   - Parallel application
   - Worker pool scaling
   - Atomic mode (rollback on failure)
   - Per-delta error handling

### Integration Tests

1. **Curator Integration**
   - Curation generates deltas
   - Duplicate detection creates increment deltas
   - History tracks all curator operations

2. **Adapter Integration**
   - Online adaptation creates deltas
   - Session metadata preserved
   - Rollback of session changes

3. **Playbook Integration**
   - Deltas correctly update bullets
   - Search works after delta application
   - Snapshots include delta state

### Benchmarks

```go
func BenchmarkDeltaApply_Single(b *testing.B)
func BenchmarkDeltaApply_Batch100(b *testing.B)
func BenchmarkHistoryLookup_ByBullet(b *testing.B)
func BenchmarkRollback_Single(b *testing.B)
func BenchmarkRollback_Count10(b *testing.B)
```

---

## Definition of Done

- [x] FRD created and reviewed
- [ ] `delta.go` - Delta data structures with validation
- [ ] `history.go` - DeltaHistory with indexing and stats
- [ ] `applier.go` - DeltaApplier with copy-on-write
- [ ] `rollback.go` - RollbackManager with inverse deltas
- [ ] `batch.go` - Batch processing with worker pool
- [ ] Unit tests for all packages (≥90% coverage)
- [ ] Integration tests with Playbook, Curator, Adapter
- [ ] Benchmarks for performance validation
- [ ] Race detector clean (`go test -race`)
- [ ] Linter clean (`make lint`)
- [ ] Documentation in `docs/packages/ace.md`
- [ ] ROADMAP.md Feature 5 marked complete

---

## Open Questions

1. **Delta Persistence**: Should delta history be persisted to disk, or kept in-memory only?
   - **Decision**: In-memory for now. Future: add `DeltaHistory.Save/Load` similar to Playbook serialization.

2. **Decrement Operations**: Should we add OpDecrementHelpful/OpDecrementHarmful for rollback?
   - **Decision**: Yes, add these operations to support proper rollback.

3. **Old Value Storage**: How to store old values for rollback without bloating deltas?
   - **Decision**: Store in `Delta.Metadata.Tags` with special keys like `old_content`, `old_value`.

4. **Conflict Detection**: What if a bullet is modified between delta creation and application?
   - **Decision**: Not solving for MVP. Assume single-writer. Future: add version numbers to bullets.

5. **History Size Limit**: Should we limit delta history size?
   - **Decision**: Add `DeltaHistory.Prune(olderThan time.Time)` method. Manual for now, auto-prune later.

---

## References

- [ACE Roadmap Feature 5](../../ace-agentic-context-engineering/ROADMAP.md#feature-5-incremental-delta-updates-system)
- [FRD-20251030-004: Curator Component](./FRD-20251030-004-curator-component.md)
- [FRD-20251030-005: Online Context Adaptation](./FRD-20251030-005-online-context-adaptation.md)
- [ACE Paper](../../ace-agentic-context-engineering/2510.04618v1.pdf)
- [Bullet Package](../../docs/packages/ace.md#bullet-package)
- [Playbook Package](../../docs/packages/ace.md#playbook-package)

---

**Status**: Ready for Implementation  
**Estimated Effort**: 8-12 hours (4-6 hours implementation + 4-6 hours testing)  
**Dependencies**: Playbook, Curator, Adapter (all complete)
