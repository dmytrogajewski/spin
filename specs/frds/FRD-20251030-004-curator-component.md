# Feature Requirements Document: Curator Component

**FRD ID:** FRD-20251030-004  
**Feature:** ACE Curator Component  
**Phase:** 3 - Learning & Reflection  
**Status:** Draft  
**Created:** 2025-10-30  
**Author:** AI Agent (Spin)

---

## Executive Summary

The Curator component transforms insights from the Reflector into bullets and merges them into the playbook. It implements the "Curate" step of the ACE learning loop: Generate → Reflect → **Curate**.

**Key Goal:** Add new knowledge to the playbook without duplicates, using deterministic (non-LLM) merge logic.

---

## Background

### Context

The ACE learning loop:
1. **Generate** (✅ Complete) - Execute tasks with context bullets
2. **Reflect** (✅ Complete) - Extract insights from trajectories  
3. **Curate** (🔄 This Feature) - Add insights to playbook as bullets

The Curator bridges reflection and playbook growth by converting insights to bullets and preventing duplicates.

### Problem Statement

Insights from Reflector need to be:
- Converted to bullet format
- Checked for duplicates (semantic similarity)
- Added to playbook
- Tracked with metadata

**Challenge:** Avoid playbook bloat from duplicate/redundant bullets while maintaining comprehensive coverage.

---

## Requirements

### Functional Requirements

#### FR-1: Insight to Bullet Conversion
**Priority:** MUST  
**Description:** Convert Insight structs to Bullet structs.

**Acceptance Criteria:**
- Accept `[]*Insight` as input
- Create Bullet for each unique insight
- Preserve content, confidence as helpful count
- Add source metadata (trajectory ID)
- Return `[]*Bullet`

**Conversion Rules:**
```
Insight → Bullet:
- Content → Content (1:1)
- Confidence → HelpfulCount (scale 0.0-1.0 to 0-10)
- Category → Tags["category"]
- Source → Tags["source"]
- Evidence → Tags["evidence"] (joined)
```

#### FR-2: Semantic De-duplication
**Priority:** MUST  
**Description:** Detect duplicate bullets using semantic similarity.

**Acceptance Criteria:**
- Compare new bullet against existing playbook bullets
- Use cosine similarity on embeddings
- Configurable similarity threshold (default 0.85)
- Return list of duplicates found
- Skip adding if duplicate exists

**Algorithm:**
```
For each new bullet:
  1. Get embedding
  2. Compare to all playbook bullets
  3. If similarity > threshold: mark as duplicate
  4. If not duplicate: add to playbook
```

#### FR-3: Playbook Merging
**Priority:** MUST  
**Description:** Add new bullets to playbook deterministically.

**Acceptance Criteria:**
- Accept `[]*Bullet` to add
- Check each for duplicates
- Add non-duplicates to playbook
- Update existing bullet counters if duplicate
- Return merge statistics (added, skipped)

**Merge Strategy:**
- **New bullet**: Add to playbook
- **Duplicate found**: Increment helpful count of existing bullet
- **Thread-safe**: Use playbook's existing mutex

#### FR-4: Merge Statistics
**Priority:** SHOULD  
**Description:** Track merge results for observability.

**Output:**
```go
type MergeResult struct {
    Added      int      // New bullets added
    Skipped    int      // Duplicates skipped
    Updated    int      // Existing bullets updated
    Duplicates []string // IDs of duplicate bullets
}
```

#### FR-5: Batch Processing
**Priority:** SHOULD  
**Description:** Process multiple insights efficiently.

**Acceptance Criteria:**
- Accept batch of insights
- Convert all to bullets
- De-duplicate as batch
- Merge all at once
- Return aggregate statistics

---

### Non-Functional Requirements

#### NFR-1: Performance
- Similarity check per bullet: < 10ms
- Batch merge (100 insights): < 1 second
- Memory efficient (no full playbook duplication)

#### NFR-2: Determinism
- Same inputs always produce same outputs
- No LLM calls (fully deterministic)
- Reproducible merge results

#### NFR-3: Testability
- Unit test coverage ≥ 90%
- Integration tests with Reflector
- Similarity threshold configurable for testing

#### NFR-4: Thread Safety
- Safe concurrent merges (use playbook mutex)
- No race conditions
- Atomic playbook updates

---

## Architecture

### Package Structure

```
internal/ace/curator/
├── curator.go          # Main Curator service
├── converter.go        # Insight → Bullet conversion
├── deduplicator.go     # Semantic de-duplication
├── curator_test.go     # Unit tests
└── integration_test.go # Integration tests
```

### Data Structures

#### MergeRequest

```go
type MergeRequest struct {
    // Insights to curate into playbook
    Insights []*reflector.Insight
    
    // SimilarityThreshold for de-duplication (default 0.85)
    SimilarityThreshold float64
}
```

#### MergeResult

```go
type MergeResult struct {
    // Added is count of new bullets added
    Added int
    
    // Skipped is count of duplicates not added
    Skipped int
    
    // Updated is count of existing bullets updated
    Updated int
    
    // Duplicates are IDs of bullets that were duplicates
    Duplicates []string
    
    // AddedBullets are the new bullets that were added
    AddedBullets []*bullet.Bullet
}
```

### Core Interface

```go
type Curator interface {
    // Curate converts insights to bullets and merges into playbook
    Curate(ctx context.Context, req MergeRequest) (*MergeResult, error)
    
    // ConvertInsights transforms insights to bullets
    ConvertInsights(insights []*reflector.Insight) ([]*bullet.Bullet, error)
    
    // FindDuplicates detects semantic duplicates
    FindDuplicates(ctx context.Context, newBullets []*bullet.Bullet) (map[string]string, error)
}
```

### Implementation

```go
type curator struct {
    playbook  *playbook.Playbook
    embedder  embedding.Embedder
    threshold float64
}

func NewCurator(pb *playbook.Playbook, emb embedding.Embedder, opts ...Option) Curator {
    c := &curator{
        playbook:  pb,
        embedder:  emb,
        threshold: 0.85,
    }
    
    for _, opt := range opts {
        opt(c)
    }
    
    return c
}
```

---

## Implementation Plan

### Phase 1: Insight to Bullet Conversion (Day 1)
**Tasks:**
1. Create `curator` package
2. Define `MergeRequest` and `MergeResult` types
3. Implement `ConvertInsights()` method
4. Conversion logic: Insight → Bullet

**Tests:**
- Convert single insight
- Convert multiple insights
- Confidence scaling (0.0-1.0 → 0-10)
- Metadata preservation

**DoD:**
- All conversion tests pass
- 100% coverage on converter

### Phase 2: Semantic De-duplication (Day 2)
**Tasks:**
1. Implement `FindDuplicates()` method
2. Cosine similarity calculation
3. Threshold-based duplicate detection
4. Return duplicate mapping

**Tests:**
- No duplicates (empty playbook)
- Exact duplicate detection
- Near-duplicate with threshold
- Multiple duplicates
- Non-duplicates

**DoD:**
- Duplicate detection works
- Threshold configurable
- 90% coverage

### Phase 3: Playbook Merging (Day 3)
**Tasks:**
1. Implement `Curate()` method
2. Batch conversion
3. Duplicate checking
4. Playbook update (add/increment)
5. Statistics tracking

**Tests:**
- Merge new bullets
- Skip duplicates
- Update existing bullets
- Batch merge
- Statistics accuracy

**DoD:**
- End-to-end curate works
- Statistics correct
- 90% coverage

### Phase 4: Integration & Performance (Day 4)
**Tasks:**
1. Integration test with Reflector
2. Performance benchmarks
3. Thread safety verification
4. Documentation

**Tests:**
- Reflector → Curator integration
- Concurrent merges
- Large batch performance

**DoD:**
- All integration tests pass
- Performance targets met
- Race detector clean

---

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥ 90%

**Test Categories:**
1. Insight to bullet conversion
2. Duplicate detection
3. Merge logic
4. Statistics calculation
5. Error handling

**Key Test Cases:**
```go
func TestCurator_ConvertInsights(t *testing.T)
func TestCurator_ConvertInsights_ConfidenceScaling(t *testing.T)
func TestCurator_FindDuplicates_NoDuplicates(t *testing.T)
func TestCurator_FindDuplicates_ExactMatch(t *testing.T)
func TestCurator_FindDuplicates_ThresholdBoundary(t *testing.T)
func TestCurator_Curate_NewBullets(t *testing.T)
func TestCurator_Curate_DuplicatesSkipped(t *testing.T)
func TestCurator_Curate_BatchProcessing(t *testing.T)
```

### Integration Tests

**Tests:**
1. Reflector insights → Curator → Playbook
2. Multiple curate calls (idempotent)
3. Concurrent curate operations

### Table-Driven Tests

All conversion and duplicate detection should use table-driven tests:

```go
func TestConvertInsights(t *testing.T) {
    tests := []struct {
        name     string
        insights []*reflector.Insight
        wantLen  int
        wantTags map[string]string
    }{
        {
            name: "single insight",
            insights: []*reflector.Insight{
                {Content: "test", Confidence: 0.8, Category: reflector.CategorySuccessPattern},
            },
            wantLen: 1,
            wantTags: map[string]string{"category": "success_pattern"},
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            bullets, err := ConvertInsights(tt.insights)
            require.NoError(t, err)
            assert.Equal(t, tt.wantLen, len(bullets))
        })
    }
}
```

---

## Success Criteria

### Functional Success
- ✅ Convert insights to bullets
- ✅ Detect semantic duplicates
- ✅ Merge into playbook
- ✅ Track merge statistics
- ✅ Batch processing works

### Quality Success
- ✅ Test coverage ≥ 90%
- ✅ No lint errors
- ✅ Race detector clean
- ✅ Integration tests pass

### Performance Success
- ✅ Similarity check: < 10ms per bullet
- ✅ Batch merge (100): < 1s
- ✅ No memory leaks

---

## Dependencies

### Internal Dependencies
- ✅ `internal/ace/reflector` (Insight type)
- ✅ `internal/ace/bullet` (Bullet type)
- ✅ `internal/ace/playbook` (Playbook management)
- ✅ `internal/ace/embedding` (Embedder interface)

### External Dependencies
- None (self-contained)

---

## Risks and Mitigations

### Risk 1: False Positive Duplicates
**Impact:** HIGH  
**Probability:** MEDIUM  
**Mitigation:**
- Configurable similarity threshold
- Log duplicate detections
- Allow manual override (future)
- Test with real-world data

### Risk 2: Performance Degradation
**Impact:** MEDIUM  
**Probability:** LOW  
**Mitigation:**
- Benchmark similarity checks
- Optimize embedding comparison
- Batch processing
- Early termination on exact match

### Risk 3: Playbook Bloat
**Impact:** MEDIUM  
**Probability:** MEDIUM  
**Mitigation:**
- Aggressive de-duplication
- Periodic playbook pruning (future)
- Low-utility bullet removal (future)

---

## Future Enhancements (Post-MVP)

### Phase 4+
- LLM-based similarity (semantic understanding)
- Bullet merging (combine similar bullets)
- Automatic bullet refinement
- Playbook compression
- User feedback integration
- Bullet lifecycle management (archive old bullets)

---

## Appendix A: Conversion Examples

### Example 1: Basic Conversion

```go
// Input: Insight
insight := &reflector.Insight{
    Content:    "Always validate input parameters before processing",
    Source:     "traj-123",
    Confidence: 0.85,
    Category:   reflector.CategorySuccessPattern,
    Evidence:   []string{"validation prevented error"},
}

// Output: Bullet
bullet := &bullet.Bullet{
    Content:      "Always validate input parameters before processing",
    HelpfulCount: 8,  // 0.85 * 10 = 8.5 → 8
    HarmfulCount: 0,
    Tags: map[string]string{
        "category": "success_pattern",
        "source":   "traj-123",
        "evidence": "validation prevented error",
    },
}
```

### Example 2: Confidence Scaling

```
Confidence 0.0 → HelpfulCount 0
Confidence 0.5 → HelpfulCount 5
Confidence 0.85 → HelpfulCount 8
Confidence 1.0 → HelpfulCount 10
```

---

## Appendix B: Similarity Threshold

### Recommended Thresholds

| Threshold | Behavior | Use Case |
|-----------|----------|----------|
| 0.95 | Very strict | Exact duplicates only |
| 0.85 | Strict (default) | Near-exact duplicates |
| 0.75 | Moderate | Similar concepts |
| 0.65 | Loose | Related ideas |

### Testing Thresholds

For testing, use lower thresholds (0.7) to ensure duplicate detection works.

---

**Document Status:** Ready for Implementation  
**Next Steps:** Begin Phase 1 - Insight to Bullet Conversion  
**Estimated Completion:** 4 days from start
