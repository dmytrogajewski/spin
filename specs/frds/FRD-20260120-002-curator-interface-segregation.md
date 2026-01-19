# FRD-20260120-002: Curator Interface Segregation

**Created:** 2026-01-20  
**Author:** Architecture Refactoring  
**Status:** In Progress  
**Priority:** P1 (High)

## Problem Statement

The `Curator` interface in `internal/ace/curator/curator.go` violates the Interface Segregation Principle (ISP) by combining 9 methods that serve 3 distinct concerns:

```go
type Curator interface {
    // Curation/Merging (concern 1)
    Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)
    CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error)
    FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)

    // Refinement (concern 2)
    Refine(ctx context.Context) (*RefinementResult, error)

    // Bullet Updates (concern 3)
    ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error
    UpdateBulletContent(ctx context.Context, bulletID, newContent string) error
    AddBulletTag(ctx context.Context, bulletID, key, value string) error
    RemoveBulletTag(ctx context.Context, bulletID, key string) error
    UpdateBulletEmbedding(ctx context.Context, bulletID string, embedding []float32) error
}
```

### Impact

1. **Clients are forced to depend on methods they don't use:**
   - `adapter/adapter.go` only uses `Curate()` but depends on entire interface
   - `agent/ace_service.go` uses `Curate()`, `ApplyBulletFeedback()`, and `Refine()` but not other update methods

2. **Testing overhead:** Mock implementations must implement all 9 methods even when testing single concern

3. **Violation of Single Responsibility:** The interface mixes curation, refinement, and CRUD operations

4. **Harder to extend:** Adding new bullet update methods affects all implementers and consumers

## Solution

Split the `Curator` interface into three focused interfaces following ISP:

### 1. BulletMerger (Curation + Deduplication)

```go
// BulletMerger handles insight-to-bullet conversion and playbook merging.
type BulletMerger interface {
    // Curate converts insights to bullets and merges into playbook
    Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)

    // CurateBatch processes multiple merge requests in parallel
    CurateBatch(ctx context.Context, req BatchMergeRequest) (*BatchMergeResult, error)

    // FindDuplicates detects semantic duplicates using cosine similarity
    FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)
}
```

### 2. BulletRefiner (Refinement/Pruning)

```go
// BulletRefiner handles playbook quality maintenance through pruning.
type BulletRefiner interface {
    // Refine explicitly prunes low-utility bullets
    Refine(ctx context.Context) (*RefinementResult, error)
}
```

### 3. BulletUpdater (CRUD Operations)

```go
// BulletUpdater handles individual bullet modifications via delta operations.
type BulletUpdater interface {
    // ApplyBulletFeedback applies helpful/harmful feedback using batch delta operations
    ApplyBulletFeedback(ctx context.Context, feedback map[string]string) error

    // UpdateBulletContent updates bullet content using delta operation
    UpdateBulletContent(ctx context.Context, bulletID, newContent string) error

    // AddBulletTag adds or updates a tag on a bullet using delta operation
    AddBulletTag(ctx context.Context, bulletID, key, value string) error

    // RemoveBulletTag removes a tag from a bullet using delta operation
    RemoveBulletTag(ctx context.Context, bulletID, key string) error

    // UpdateBulletEmbedding updates bullet embedding using delta operation
    UpdateBulletEmbedding(ctx context.Context, bulletID string, embedding []float32) error
}
```

### 4. Curator (Composite Interface for Backward Compatibility)

```go
// Curator is the composite interface combining all bullet management capabilities.
// Use the specific interfaces (BulletMerger, BulletRefiner, BulletUpdater) when
// only a subset of functionality is needed.
type Curator interface {
    BulletMerger
    BulletRefiner
    BulletUpdater
}
```

## Affected Files

### Primary Changes

| File | Change |
|------|--------|
| `internal/ace/curator/curator.go` | Add new interfaces, keep Curator as composite |
| `internal/ace/curator/curator_test.go` | Update to test individual interfaces |

### Consumer Updates (Interface Usage)

| File | Current Usage | New Interface |
|------|---------------|---------------|
| `internal/ace/adapter/adapter.go:26` | `curator.Curator` | `curator.BulletMerger` |
| `internal/agent/ace_service.go:34` | `curator.Curator` | `curator.Curator` (needs all) |

## Implementation Plan

### Phase 1: Add New Interfaces (Non-Breaking)

1. Add `BulletMerger`, `BulletRefiner`, `BulletUpdater` interfaces to `curator.go`
2. Redefine `Curator` as composite of the three interfaces
3. Ensure `curator` struct still implements all interfaces (compile-time check)

### Phase 2: Update Tests

1. Add interface-specific test helpers
2. Update existing tests to use appropriate narrower interfaces where applicable
3. Add tests verifying interface satisfaction

### Phase 3: Update Consumers (Optional - Future)

1. Update `adapter/adapter.go` to use `BulletMerger` instead of `Curator`
2. Document when to use which interface

## Acceptance Criteria

1. **Backward Compatible:** Existing code using `Curator` continues to work unchanged
2. **Interface Satisfaction:** `curator` struct implements all four interfaces
3. **Tests Pass:** All existing tests pass without modification
4. **Coverage:** Maintain >= 90% test coverage
5. **Lint Clean:** `make lint` passes
6. **Documentation:** All new interfaces have godoc comments

## Test Plan

### Unit Tests

```go
// Test interface satisfaction at compile time
func TestInterfaceSatisfaction(t *testing.T) {
    var _ BulletMerger = (*curator)(nil)
    var _ BulletRefiner = (*curator)(nil)
    var _ BulletUpdater = (*curator)(nil)
    var _ Curator = (*curator)(nil)
}

// Test BulletMerger interface
func TestBulletMerger_Curate(t *testing.T) { /* existing test */ }
func TestBulletMerger_CurateBatch(t *testing.T) { /* existing test */ }
func TestBulletMerger_FindDuplicates(t *testing.T) { /* existing test */ }

// Test BulletRefiner interface
func TestBulletRefiner_Refine(t *testing.T) { /* existing test */ }

// Test BulletUpdater interface
func TestBulletUpdater_ApplyBulletFeedback(t *testing.T) { /* existing test */ }
func TestBulletUpdater_UpdateBulletContent(t *testing.T) { /* existing test */ }
func TestBulletUpdater_AddBulletTag(t *testing.T) { /* existing test */ }
func TestBulletUpdater_RemoveBulletTag(t *testing.T) { /* existing test */ }
func TestBulletUpdater_UpdateBulletEmbedding(t *testing.T) { /* existing test */ }
```

### Integration Tests

- Verify `adapter.NewAdapter()` accepts `BulletMerger` parameter
- Verify `ACEService` continues to work with `Curator` composite

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Breaking existing code | Low | High | Curator remains composite interface |
| Test failures | Low | Medium | Run tests after each change |
| Consumer confusion | Low | Low | Clear documentation on when to use which |

## References

- ROADMAP.md Section 2.3: Interface Pollution - Curator
- Interface Segregation Principle: https://en.wikipedia.org/wiki/Interface_segregation_principle
- Go Proverbs - "The bigger the interface, the weaker the abstraction"
