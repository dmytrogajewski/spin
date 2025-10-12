# FRD-20251012104414: Advanced File Search and Ranking

**Package:** `internal/filesearch`
**Feature:** Advanced Search with Async Indexing and Smart Ranking
**Priority:** P2 (Nice to have)
**Status:** Planning
**Created:** 2025-10-12

---

## Overview

Enhance the existing `filesearch` package with advanced search capabilities including asynchronous indexing, context-aware cancellation, and intelligent ranking algorithm that prioritizes relevance based on filename, path, and fuzzy matching scores.

### Current State

The `filesearch` package currently provides:
- ✅ Basic Scanner with gitignore support
- ✅ Simple Matcher with fuzzy matching
- ✅ IgnoreHandler for .gitignore/.spinignore patterns

**Current Limitations:**
- Scanner is synchronous (blocks during indexing)
- No cancellation support
- Basic scoring algorithm (consecutive matches, separator bonuses, path length)
- No ranking based on filename vs path matches
- No exact match prioritization
- Limited scoring granularity

### Goals

1. **Async Indexing**: Enable non-blocking file scanning with context-based cancellation
2. **Advanced Ranking**: Implement sophisticated scoring that prioritizes:
   - Exact filename matches
   - Filename prefix matches
   - Filename contains matches
   - Path segment matches
   - Fuzzy matches
3. **Performance**: Maintain fast search (<10ms for 100k file index)
4. **API Compatibility**: Maintain backward compatibility with existing Scanner/Matcher

---

## Requirements

### Functional Requirements

#### FR-1: Async Indexing
- Scanner can run in background without blocking
- Context-aware cancellation
- Progress reporting (optional callback)
- Incremental indexing support

#### FR-2: Advanced Scoring Algorithm
Implement multi-tier scoring with the following priorities:

| Match Type | Score | Example |
|------------|-------|---------|
| Exact filename match | 100 | Query: "main.go" → File: "main.go" |
| Filename starts with query | 90 | Query: "config" → File: "config.toml" |
| Filename contains query (early) | 80-70 | Query: "test" → File: "test_utils.go" (80), "my_test.go" (75) |
| Path segment exact match | 60 | Query: "src" → File: "src/main.go" |
| Path segment prefix | 50 | Query: "int" → File: "internal/core.go" |
| Fuzzy match (consecutive chars) | 40+ | Query: "cfg" → File: "config.go" |
| Fuzzy match (scattered) | 20+ | Query: "mgo" → File: "main.go" |

**Position Weighting:**
- Earlier matches score higher
- Filename matches score higher than path matches
- Short paths get bonus points

#### FR-3: Searcher Interface
New high-level `Searcher` type that wraps Scanner and Matcher:

```go
type Searcher struct {
    root    string
    scanner *Scanner
    matcher *Matcher
    index   []string
    indexed bool
}

// NewSearcher creates a new searcher for the given root directory
func NewSearcher(root string) (*Searcher, error)

// IndexAsync indexes files asynchronously with cancellation support
func (s *Searcher) IndexAsync(ctx context.Context) error

// Search performs ranked search, returns top matches
func (s *Searcher) Search(query string, limit int) []SearchResult

// IsIndexed returns true if indexing is complete
func (s *Searcher) IsIndexed() bool
```

### Non-Functional Requirements

#### NFR-1: Performance
- Indexing: <1s for 10k files
- Search: <10ms for query on 100k file index
- Memory: <10MB for 100k file index

#### NFR-2: Backward Compatibility
- Existing Scanner and Matcher APIs unchanged
- New Searcher is additive, not breaking

#### NFR-3: Code Quality
- Test coverage ≥85%
- Cyclomatic complexity ≤15 per function
- Zero lint errors
- Race detector clean

---

## Design

### Architecture

```
internal/filesearch/
├── scanner.go         # Existing (unchanged)
├── matcher.go         # Enhanced with advanced scoring
├── ignore.go          # Existing (unchanged)
├── searcher.go        # NEW: High-level searcher with async indexing
├── doc.go             # Updated package documentation
├── scanner_test.go    # Existing tests
├── matcher_test.go    # Enhanced tests for new scoring
├── ignore_test.go     # Existing tests
└── searcher_test.go   # NEW: Searcher tests
```

### Data Structures

#### Enhanced Match Type

```go
// SearchResult represents a ranked search result with detailed scoring breakdown
type SearchResult struct {
    Path        string
    Score       int
    MatchType   MatchType
    Indices     []int
    ScoreDetail ScoreDetail
}

// MatchType indicates how the query matched the file
type MatchType int

const (
    MatchExactFilename MatchType = iota
    MatchFilenamePrefix
    MatchFilenameContains
    MatchPathSegment
    MatchFuzzyConsecutive
    MatchFuzzyScattered
)

// ScoreDetail provides scoring breakdown for debugging
type ScoreDetail struct {
    BaseScore         int
    PositionBonus     int
    ConsecutiveBonus  int
    SeparatorBonus    int
    PathLengthBonus   int
    FilenameBonus     int
}
```

#### Searcher Type

```go
// Searcher provides high-level file search with async indexing
type Searcher struct {
    root        string
    scanner     *Scanner
    matcher     *Matcher
    index       []string
    indexed     bool
    indexMu     sync.RWMutex
    indexErr    error
}
```

### Advanced Scoring Algorithm

#### Algorithm Flow

```
1. Parse query and file path:
   - Extract filename from path
   - Extract path segments
   - Normalize to lowercase

2. Check for exact/prefix matches:
   - Exact filename match → 100 points
   - Filename starts with query → 90 points
   - Skip fuzzy matching if found

3. Check for contains matches:
   - Filename contains query → 80 - (position / length * 10)
   - Earlier position = higher score
   - Skip fuzzy matching if score > 70

4. Check path segments:
   - Exact segment match → 60 points
   - Segment prefix match → 50 points

5. Fuzzy matching:
   - Consecutive character matches → 15 points per consecutive run
   - Separator bonuses → 10 points
   - Base score for each match → 1 point

6. Apply bonuses:
   - Path length bonus: <20 chars (+50), <40 chars (+25), else (+10)
   - Filename bonus: Match in filename (+30 vs path)

7. Return sorted results by score (descending)
```

#### Scoring Implementation

```go
func (m *Matcher) scoreAdvanced(query, path string) (int, MatchType, []int) {
    queryLower := strings.ToLower(query)
    pathLower := strings.ToLower(path)

    // Extract filename
    filename := filepath.Base(pathLower)

    // 1. Exact filename match
    if filename == queryLower {
        return 100, MatchExactFilename, allIndices(path, filename)
    }

    // 2. Filename prefix match
    if strings.HasPrefix(filename, queryLower) {
        return 90, MatchFilenamePrefix, prefixIndices(len(query))
    }

    // 3. Filename contains match
    if idx := strings.Index(filename, queryLower); idx >= 0 {
        score := 80 - (idx * 10 / len(filename))
        if score < 70 {
            score = 70
        }
        return score, MatchFilenameContains, containsIndices(path, idx, len(query))
    }

    // 4. Path segment matching
    segments := strings.Split(filepath.Dir(pathLower), string(filepath.Separator))
    for _, seg := range segments {
        if seg == queryLower {
            return 60, MatchPathSegment, segmentIndices(path, seg)
        }
        if strings.HasPrefix(seg, queryLower) {
            return 50, MatchPathSegment, segmentIndices(path, seg)
        }
    }

    // 5. Fuzzy matching (existing algorithm enhanced)
    score, indices := m.matchCharacters(queryLower, pathLower)
    if score == -1 {
        return -1, MatchType(-1), nil
    }

    matchType := MatchFuzzyConsecutive
    if !hasConsecutiveMatches(indices) {
        matchType = MatchFuzzyScattered
    }

    // Add filename bonus if match is in filename
    if isMatchInFilename(path, indices) {
        score += 30
    }

    score += m.pathLengthBonus(path)

    return score, matchType, indices
}
```

### Async Indexing

```go
func (s *Searcher) IndexAsync(ctx context.Context) error {
    s.indexMu.Lock()
    if s.indexed {
        s.indexMu.Unlock()
        return nil
    }
    s.indexMu.Unlock()

    // Run scanning in background
    files, err := s.scanner.ScanWithContext(ctx)
    if err != nil {
        s.indexMu.Lock()
        s.indexErr = err
        s.indexMu.Unlock()
        return err
    }

    s.indexMu.Lock()
    s.index = files
    s.indexed = true
    s.indexMu.Unlock()

    return nil
}
```

**Note:** Scanner needs enhancement to support context cancellation. This can be done by checking `ctx.Done()` periodically during `WalkDir`.

---

## Implementation Plan

### Phase 1: Enhanced Matcher (1 day)
1. Add `scoreAdvanced()` method with multi-tier scoring
2. Update `Score()` to use new algorithm
3. Add `SearchResult` type with score breakdown
4. Maintain backward compatibility with `Match` type

### Phase 2: Searcher (1 day)
1. Implement `Searcher` struct
2. Implement `NewSearcher()` constructor
3. Implement `IndexAsync()` with context support
4. Implement `Search()` with ranking
5. Add `IsIndexed()` helper

### Phase 3: Scanner Enhancement (0.5 day)
1. Add `ScanWithContext()` method for cancellation
2. Maintain backward compatibility with `Scan()`

### Phase 4: Testing (0.5 day)
1. Write tests for advanced scoring algorithm
2. Write tests for Searcher
3. Write tests for context cancellation
4. Benchmark performance

---

## Testing Strategy

### Unit Tests

#### Matcher Tests
```go
TestScore_ExactFilenameMatch
TestScore_FilenamePrefix
TestScore_FilenameContains
TestScore_PathSegment
TestScore_FuzzyConsecutive
TestScore_FuzzyScattered
TestScore_PositionWeighting
TestScore_FilenameBonus
TestScore_PathLengthBonus
```

#### Searcher Tests
```go
TestSearcher_NewSearcher
TestSearcher_IndexAsync
TestSearcher_IndexAsync_Cancellation
TestSearcher_Search
TestSearcher_Search_Ranking
TestSearcher_Search_Limit
TestSearcher_IsIndexed
```

### Integration Tests
```go
TestSearcher_RealProject      // Test on real project structure
TestSearcher_LargeRepository  // Test with 10k+ files
TestSearcher_Concurrent       // Test concurrent searches
```

### Benchmarks
```go
BenchmarkMatcher_Score        // Benchmark scoring algorithm
BenchmarkSearcher_Index       // Benchmark indexing speed
BenchmarkSearcher_Search      // Benchmark search performance
```

### Performance Targets
- `Matcher.Score()`: <1μs per path
- `Searcher.IndexAsync()`: <1s for 10k files
- `Searcher.Search()`: <10ms for 100k file index

---

## Acceptance Criteria

### Functional
- ✅ Exact filename matches score 100
- ✅ Filename prefix matches score 90
- ✅ Filename contains matches score 80-70
- ✅ Path segment matches score 60-50
- ✅ Fuzzy matches score appropriately
- ✅ Results sorted by score (descending)
- ✅ Async indexing works with context cancellation
- ✅ Search returns top N results correctly

### Non-Functional
- ✅ Test coverage ≥85%
- ✅ All benchmarks meet performance targets
- ✅ Zero lint errors
- ✅ Race detector clean
- ✅ Backward compatible with existing API

---

## Migration Path

No migration needed - this is additive:

**Existing code continues to work:**
```go
scanner := filesearch.NewScanner(".", false)
files, _ := scanner.Scan()
matcher := filesearch.NewMatcher(false)
matches := matcher.Match("test", files)
```

**New code can use Searcher:**
```go
searcher, _ := filesearch.NewSearcher(".")
searcher.IndexAsync(context.Background())
results := searcher.Search("test", 10)
```

---

## Future Enhancements

1. **Real-time Indexing**: Watch for file changes and update index
2. **Trigram Index**: Use trigram index for faster searching (100k+ files)
3. **Query Syntax**: Support advanced query syntax (e.g., `path:src ext:.go`)
4. **Learning**: Learn from user selections to improve ranking
5. **Incremental Indexing**: Support adding/removing files without full re-index

---

## References

1. [Fuzzy String Matching Algorithms](https://en.wikipedia.org/wiki/Approximate_string_matching)
2. [Sublime Text Fuzzy Matching](https://github.com/forrestthewoods/lib_fts)
3. [VSCode Quick Open Algorithm](https://code.visualstudio.com/blogs/2021/02/03/custom-syntax-extensions#_fuzzy-matching)
4. [FZF Scoring Algorithm](https://github.com/junegunn/fzf/blob/master/src/algo/algo.go)

---

## Dependencies

**Internal:**
- `internal/filesearch/scanner.go` - Base scanner (enhance for context)
- `internal/filesearch/matcher.go` - Base matcher (enhance scoring)
- `pkg/strutil` - String utilities

**Standard Library:**
- `context` - Cancellation support
- `path/filepath` - Path operations
- `sort` - Result sorting
- `sync` - Thread-safe indexing

**External:**
- None (keep it dependency-free)

---

**Author:** Spin AI Agent
**Reviewer:** TBD
**Approved:** TBD
