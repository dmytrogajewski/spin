# FRD-20251012015732: Fuzzy Matcher for internal/patchapply

**Feature:** Fuzzy Matcher (Feature 2.2 from tools-modules ROADMAP)
**Priority:** P0 (Critical for robust patch application)
**Status:** Planning
**Created:** 2025-10-12 01:57:32
**Updated:** 2025-10-12 01:57:32
**Related:** [tools-modules.md](../tools-modules/tools-modules.md), [ROADMAP.md](../tools-modules/ROADMAP.md), [FRD-20251012030000-patchapply-parser.md](./FRD-20251012030000-patchapply-parser.md)

---

## Overview

The Fuzzy Matcher is a critical component that enables robust patch application even when the target file has minor variations from the patch context. It finds the correct location to apply a hunk by comparing context lines with file content using similarity algorithms, whitespace normalization, and configurable thresholds.

### Goals

1. **Find Context Reliably:** Locate hunk context in target files with high accuracy (≥95%)
2. **Handle Variations:** Tolerate whitespace changes, minor edits, indentation differences
3. **Disambiguate:** Use context headers to distinguish multiple occurrences
4. **Performance:** Match context in <1ms for 10k line files
5. **Clarity:** Provide detailed error messages when context cannot be found

### Non-Goals

- Semantic understanding of code (we match text, not meaning)
- Three-way merging (that's for git merge, not patch application)
- Conflict resolution (if fuzzy match fails, we error clearly)

---

## Background

### Problem Statement

When AI models generate patches, they extract context from the current file state. By the time the patch is applied, the file may have changed due to:
- Other patches applied first
- Manual edits by the user
- Whitespace normalization (e.g., formatter ran)
- Small refactorings (variable renames)

A pure exact-match strategy would fail frequently. The fuzzy matcher provides resilience while maintaining safety.

### Real-World Examples

#### Example 1: Whitespace Variations

**Patch context:**
```go
func main() {
    fmt.Println("hello")
}
```

**Actual file:**
```go
func main() {
  fmt.Println("hello")  // Comment added
}
```

**Solution:** Normalize whitespace, treat extra spaces/tabs as equivalent

---

#### Example 2: Multiple Occurrences

**Patch context:**
```go
@@ func Process
func Process(data string) error {
    return nil
}
```

**File has two `Process` functions:**
```go
func Process(data string) error {
    return nil
}

func ProcessBatch(items []string) error {
    for _, item := range items {
        if err := Process(item); err != nil {
            return err
        }
    }
    return nil
}
```

**Solution:** Use header `@@ func Process` to match the first occurrence (exact function name match)

---

#### Example 3: Similarity Threshold

**Patch context:**
```go
func Calculate(x, y int) int {
    return x + y
}
```

**File after minor refactoring:**
```go
func Calculate(a, b int) int {
    return a + b
}
```

**Solution:** Compute similarity (85% match due to parameter renames), accept if ≥ threshold

---

### Current State

- ✅ Parser (Feature 2.1) complete - provides `Hunk` with context
- ✅ `pkg/strutil` has `LevenshteinDistance`, `Similarity`, `NormalizeWhitespace`
- ❌ No fuzzy matching implementation exists yet

---

## Requirements

### Functional Requirements

#### FR1: Exact Match (Fast Path)

**Description:** When context lines match exactly, return immediately without fuzzy logic

**Acceptance Criteria:**
- ✅ Compares hunk context lines directly with file lines
- ✅ Returns line index immediately on exact match
- ✅ Performance: <100μs for typical hunks (5-10 context lines)

**Test Cases:**
```go
// TC1: Exact match at start
fileLines := []string{"func main() {", "    fmt.Println(\"hello\")", "}"}
contextLines := []string{"func main() {", "    fmt.Println(\"hello\")"}
expected := 0  // Match starts at line 0

// TC2: Exact match in middle
fileLines := []string{"package main", "", "func main() {", "    return nil", "}"}
contextLines := []string{"func main() {", "    return nil"}
expected := 2  // Match starts at line 2

// TC3: No exact match
fileLines := []string{"func foo() {", "}"}
contextLines := []string{"func bar() {"}
expected := -1  // No exact match, fallback to fuzzy
```

---

#### FR2: Fuzzy Match with Whitespace Normalization

**Description:** When exact match fails, try fuzzy matching with normalized whitespace

**Acceptance Criteria:**
- ✅ Normalizes leading/trailing whitespace
- ✅ Collapses multiple spaces to single space
- ✅ Compares normalized lines
- ✅ Returns line index if similarity ≥ threshold (default 85%)

**Algorithm:**
```
1. For each possible position in file:
   a. Extract window of same size as context
   b. Normalize all lines (context and window)
   c. Compute similarity score (0.0-1.0)
   d. If score ≥ threshold, return position
2. If no match found, return -1
```

**Test Cases:**
```go
// TC1: Whitespace variation
fileLines := []string{
    "func main() {",
    "  fmt.Println(\"hello\")",  // Different indentation
    "}",
}
contextLines := []string{
    "func main() {",
    "    fmt.Println(\"hello\")",  // Original indentation
}
threshold := 0.85
expected := 0  // Match despite whitespace difference

// TC2: Extra trailing spaces
fileLines := []string{"func foo() {   ", "}"}  // Trailing spaces
contextLines := []string{"func foo() {", "}"}
threshold := 0.85
expected := 0  // Match after normalization

// TC3: Below threshold
fileLines := []string{"func totally_different() {", "}"}
contextLines := []string{"func original() {", "}"}
threshold := 0.85
expected := -1  // Similarity too low
```

---

#### FR3: Similarity Computation

**Description:** Calculate similarity between two sets of lines

**Algorithm:**
```
Similarity(contextLines, windowLines) -> float64:
  1. Normalize all lines
  2. For each pair of lines:
     a. Compute Levenshtein distance
     b. Convert to similarity: 1.0 - (distance / max(len1, len2))
  3. Return average similarity across all lines
```

**Acceptance Criteria:**
- ✅ Returns 1.0 for identical lines
- ✅ Returns 0.0 for completely different lines
- ✅ Uses `strutil.Similarity()` for line-by-line comparison
- ✅ Averages scores across all lines

**Test Cases:**
```go
// TC1: Identical lines
contextLines := []string{"hello", "world"}
windowLines := []string{"hello", "world"}
expected := 1.0

// TC2: One line different
contextLines := []string{"hello", "world"}
windowLines := []string{"hello", "universe"}
expected := ~0.72  // (1.0 + 0.43) / 2 ≈ 0.72

// TC3: All different
contextLines := []string{"foo", "bar"}
windowLines := []string{"xyz", "abc"}
expected := <0.3
```

---

#### FR4: Context Header Matching

**Description:** Use hunk headers to disambiguate multiple occurrences

**Acceptance Criteria:**
- ✅ If hunk has non-empty header, prefer matches near header occurrence
- ✅ Search for header text in file first
- ✅ Search for context starting near header location
- ✅ If header not found, fall back to full-file search

**Algorithm:**
```
FindContext(hunk, fileLines) -> int:
  if hunk.Header != "":
    headerPos := findHeader(hunk.Header, fileLines)
    if headerPos >= 0:
      // Search in window around header (±50 lines)
      start := max(0, headerPos - 50)
      end := min(len(fileLines), headerPos + 50)
      if pos := findInRange(start, end); pos >= 0:
        return pos
  // Fallback: search entire file
  return findInRange(0, len(fileLines))
```

**Test Cases:**
```go
// TC1: Header helps disambiguate
fileLines := []string{
    "func ProcessA(x int) {",
    "    return x + 1",
    "}",
    "",
    "func ProcessB(x int) {",  // Line 4
    "    return x + 1",         // Same context as ProcessA!
    "}",
}
header := "func ProcessB"
contextLines := []string{"    return x + 1"}
expected := 5  // Match in ProcessB, not ProcessA

// TC2: Header not found, fallback to full search
fileLines := []string{"func main() {", "    return 0", "}"}
header := "func nonexistent"
contextLines := []string{"    return 0"}
expected := 1  // Found via fallback

// TC3: Empty header, full search
fileLines := []string{"line1", "line2", "line3"}
header := ""
contextLines := []string{"line2"}
expected := 1
```

---

#### FR5: Configurable Threshold

**Description:** Allow threshold adjustment for different use cases

**Acceptance Criteria:**
- ✅ Default threshold: 0.85 (85%)
- ✅ Threshold can be set via `SetThreshold(threshold float64)`
- ✅ Validates threshold is in range [0.0, 1.0]

**Test Cases:**
```go
// TC1: Default threshold
m := NewMatcher(fileLines)
// m.threshold == 0.85

// TC2: Custom threshold
m := NewMatcher(fileLines)
m.SetThreshold(0.90)
// m.threshold == 0.90

// TC3: Invalid threshold
m := NewMatcher(fileLines)
err := m.SetThreshold(1.5)
// err != nil, "threshold must be between 0.0 and 1.0"
```

---

#### FR6: Error Reporting

**Description:** Provide detailed errors when context cannot be found

**Acceptance Criteria:**
- ✅ Error includes hunk header if present
- ✅ Error includes first few context lines for debugging
- ✅ Error suggests possible reasons (file changed, wrong file, etc.)

**Error Format:**
```
failed to find context for hunk "@@ func Process":
  Context lines:
    func Process(data string) error {
        return nil

  Searched 1000 lines, best match: 67% at line 45

  Possible reasons:
    - File has been modified since patch was generated
    - Patch is for a different version of the file
    - Context threshold (85%) is too strict
```

---

### Non-Functional Requirements

#### NFR1: Performance

**Target:** <1ms for 10k line file

**Rationale:** Fuzzy matching is O(n×m) where n=file size, m=context size. Must optimize.

**Optimizations:**
- Early exit on exact match
- Skip positions that can't possibly match (use heuristics)
- Limit search window when header is present (±50 lines)
- Pre-normalize file lines once (cache normalized versions)

**Measurement:**
```go
func BenchmarkMatcher_FindContext_10kLines(b *testing.B) {
    fileLines := generateLines(10000)
    contextLines := []string{"func Process(data string) {", "    return nil"}
    m := NewMatcher(fileLines)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.FindContext(contextLines, "")
    }
}
// Target: <1ms per operation
```

---

#### NFR2: Memory Efficiency

**Target:** O(n) memory where n=file size

**Rationale:** Should not duplicate file content unnecessarily

**Strategy:**
- Store single copy of file lines
- Reuse similarity computation buffers
- Avoid creating intermediate strings during normalization

---

#### NFR3: Complexity

**Target:** Cyclomatic complexity ≤10 per function

**Rationale:** Matching logic is inherently complex, must be well-factored

**Measurement:** `gocyclo -over 10 ./internal/patchapply/`

---

### Security Requirements

#### SR1: No Path Access

**Requirement:** Matcher operates only on in-memory lines, never touches filesystem

**Rationale:** Separation of concerns. Parser validates paths, applier does I/O, matcher just matches.

---

#### SR2: Resource Limits

**Requirement:** Prevent O(n²) worst-case from becoming DoS

**Limits:**
- Max file size: 1,000,000 lines (handled by caller)
- Max context size: 1,000 lines (handled by caller)
- Timeout: None (caller can use context.Context if needed)

---

## Design

### Architecture

```
Matcher
  ├── NewMatcher(fileLines []string) *Matcher
  ├── FindContext(contextLines []string, header string) int
  ├── SetThreshold(threshold float64) error
  └── Internal methods:
      ├── findExact(contextLines []string) int
      ├── findFuzzy(contextLines []string) int
      ├── findInRange(start, end int, contextLines []string) int
      ├── findHeader(header string) int
      ├── computeSimilarity(contextLines, windowLines []string) float64
      └── normalizeLines(lines []string) []string
```

### Data Structures

```go
// Matcher finds hunk context in file content using fuzzy matching.
type Matcher struct {
    fileLines         []string
    normalizedLines   []string  // Cached normalized versions
    threshold         float64   // Default 0.85
}

// MatchResult contains the result of a context search.
type MatchResult struct {
    Found    bool
    Position int     // Line index where context starts
    Score    float64 // Similarity score (0.0-1.0)
}
```

### Implementation

```go
package patchapply

import (
    "fmt"
    "strings"

    "github.com/dmytrogajewski/spin/pkg/strutil"
)

// Matcher finds hunk context in file content using fuzzy matching algorithms.
type Matcher struct {
    fileLines       []string
    normalizedLines []string
    threshold       float64
}

// NewMatcher creates a new matcher for the given file content.
// The file content is provided as a slice of lines (without newline characters).
//
// The matcher normalizes whitespace once during initialization for performance.
func NewMatcher(fileLines []string) *Matcher {
    m := &Matcher{
        fileLines: fileLines,
        threshold: 0.85, // Default 85% similarity
    }
    m.normalizedLines = m.normalizeLines(fileLines)
    return m
}

// FindContext finds the line index where the context lines match in the file.
// Returns the starting line index (0-based) or -1 if not found.
//
// Algorithm:
//   1. If header is provided, search near header location first
//   2. Try exact match (fast path)
//   3. Try fuzzy match with whitespace normalization
//   4. Return best match if similarity ≥ threshold
//
// Parameters:
//   - contextLines: The context lines from the hunk (without +/- prefixes)
//   - header: Optional hunk header (e.g., "func Process") for disambiguation
func (m *Matcher) FindContext(contextLines []string, header string) int {
    if len(contextLines) == 0 {
        return 0 // Empty context matches at start
    }

    // If header provided, search near header first
    if header != "" {
        headerPos := m.findHeader(header)
        if headerPos >= 0 {
            // Search in window around header (±50 lines)
            start := max(0, headerPos-50)
            end := min(len(m.fileLines), headerPos+50)
            if pos := m.findInRange(start, end, contextLines); pos >= 0 {
                return pos
            }
        }
    }

    // Fallback: search entire file
    return m.findInRange(0, len(m.fileLines), contextLines)
}

// SetThreshold sets the similarity threshold for fuzzy matching.
// Threshold must be between 0.0 (match anything) and 1.0 (exact match).
// Default is 0.85 (85% similarity).
func (m *Matcher) SetThreshold(threshold float64) error {
    if threshold < 0.0 || threshold > 1.0 {
        return fmt.Errorf("threshold must be between 0.0 and 1.0, got %.2f", threshold)
    }
    m.threshold = threshold
    return nil
}

// findInRange searches for context within a specific range of file lines.
func (m *Matcher) findInRange(start, end int, contextLines []string) int {
    // Try exact match first (fast path)
    if pos := m.findExact(start, end, contextLines); pos >= 0 {
        return pos
    }

    // Try fuzzy match
    return m.findFuzzy(start, end, contextLines)
}

// findExact performs exact string matching for context lines.
func (m *Matcher) findExact(start, end int, contextLines []string) int {
    contextLen := len(contextLines)
    for i := start; i <= end-contextLen; i++ {
        match := true
        for j := 0; j < contextLen; j++ {
            if m.fileLines[i+j] != contextLines[j] {
                match = false
                break
            }
        }
        if match {
            return i
        }
    }
    return -1
}

// findFuzzy performs fuzzy matching with whitespace normalization.
func (m *Matcher) findFuzzy(start, end int, contextLines []string) int {
    normalizedContext := m.normalizeLines(contextLines)
    contextLen := len(contextLines)

    bestScore := 0.0
    bestPos := -1

    for i := start; i <= end-contextLen; i++ {
        // Extract window
        window := m.normalizedLines[i : i+contextLen]

        // Compute similarity
        score := m.computeSimilarity(normalizedContext, window)

        if score > bestScore {
            bestScore = score
            bestPos = i
        }

        // Early exit if perfect match
        if score >= 1.0 {
            return bestPos
        }
    }

    // Return best match if above threshold
    if bestScore >= m.threshold {
        return bestPos
    }

    return -1
}

// findHeader finds the first line containing the header text.
func (m *Matcher) findHeader(header string) int {
    headerLower := strings.ToLower(strings.TrimSpace(header))
    for i, line := range m.fileLines {
        if strings.Contains(strings.ToLower(line), headerLower) {
            return i
        }
    }
    return -1
}

// computeSimilarity computes average similarity between context and window lines.
func (m *Matcher) computeSimilarity(contextLines, windowLines []string) float64 {
    if len(contextLines) != len(windowLines) {
        return 0.0
    }

    totalSimilarity := 0.0
    for i := range contextLines {
        similarity := strutil.Similarity(contextLines[i], windowLines[i])
        totalSimilarity += similarity
    }

    return totalSimilarity / float64(len(contextLines))
}

// normalizeLines normalizes whitespace in all lines for fuzzy comparison.
func (m *Matcher) normalizeLines(lines []string) []string {
    normalized := make([]string, len(lines))
    for i, line := range lines {
        normalized[i] = strutil.NormalizeWhitespace(line)
    }
    return normalized
}

// Helper functions for min/max (Go 1.24 has these in stdlib, but for compatibility)
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

---

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥90%

#### Test Categories

1. **Exact Matching**
   - Match at start, middle, end of file
   - No match scenarios
   - Empty context
   - Single-line context

2. **Fuzzy Matching**
   - Whitespace variations (tabs vs spaces)
   - Trailing whitespace
   - Minor text differences within threshold
   - Below-threshold differences

3. **Header Matching**
   - Header helps find correct occurrence
   - Header not found (fallback to full search)
   - Empty header
   - Multiple header occurrences

4. **Threshold Configuration**
   - Default threshold (0.85)
   - Custom threshold
   - Invalid threshold (out of range)

5. **Edge Cases**
   - Empty file
   - Empty context
   - Context larger than file
   - Very large files (10k+ lines)
   - Unicode content
   - Very long lines

6. **Performance**
   - Benchmark exact match
   - Benchmark fuzzy match
   - Benchmark large files (10k lines)

### Table-Driven Tests

```go
func TestMatcher_FindContext(t *testing.T) {
    tests := []struct {
        name         string
        fileLines    []string
        contextLines []string
        header       string
        threshold    float64
        want         int // Expected line index, -1 if not found
    }{
        {
            name: "exact match at start",
            fileLines: []string{
                "func main() {",
                "    fmt.Println(\"hello\")",
                "}",
            },
            contextLines: []string{
                "func main() {",
                "    fmt.Println(\"hello\")",
            },
            header:    "",
            threshold: 0.85,
            want:      0,
        },
        {
            name: "fuzzy match with whitespace difference",
            fileLines: []string{
                "func main() {",
                "  fmt.Println(\"hello\")",  // 2 spaces instead of 4
                "}",
            },
            contextLines: []string{
                "func main() {",
                "    fmt.Println(\"hello\")",  // 4 spaces
            },
            header:    "",
            threshold: 0.85,
            want:      0,
        },
        {
            name: "header helps disambiguate",
            fileLines: []string{
                "func ProcessA(x int) {",
                "    return x + 1",
                "}",
                "",
                "func ProcessB(x int) {",
                "    return x + 1",  // Same context!
                "}",
            },
            contextLines: []string{"    return x + 1"},
            header:       "func ProcessB",
            threshold:    0.85,
            want:         5,
        },
        {
            name: "no match - below threshold",
            fileLines: []string{
                "func totally_different() {",
                "}",
            },
            contextLines: []string{"func original() {"},
            header:       "",
            threshold:    0.85,
            want:         -1,
        },
        {
            name:         "empty context matches at start",
            fileLines:    []string{"line1", "line2"},
            contextLines: []string{},
            header:       "",
            threshold:    0.85,
            want:         0,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            m := NewMatcher(tt.fileLines)
            if tt.threshold != 0 {
                m.SetThreshold(tt.threshold)
            }

            got := m.FindContext(tt.contextLines, tt.header)
            if got != tt.want {
                t.Errorf("FindContext() = %d, want %d", got, tt.want)
            }
        })
    }
}
```

### Benchmark Tests

```go
func BenchmarkMatcher_FindContext_ExactMatch(b *testing.B) {
    fileLines := generateLines(1000)
    contextLines := fileLines[500:505] // Exact match
    m := NewMatcher(fileLines)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.FindContext(contextLines, "")
    }
}

func BenchmarkMatcher_FindContext_FuzzyMatch(b *testing.B) {
    fileLines := generateLines(1000)
    contextLines := generateSimilarLines(fileLines[500:505], 0.90) // 90% similar
    m := NewMatcher(fileLines)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.FindContext(contextLines, "")
    }
}

func BenchmarkMatcher_FindContext_10kLines(b *testing.B) {
    fileLines := generateLines(10000)
    contextLines := fileLines[5000:5005]
    m := NewMatcher(fileLines)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        m.FindContext(contextLines, "")
    }
}
// Target: <1ms per operation for 10k lines
```

---

## Implementation Plan

### Phase 1: Core Matcher Structure (1 day)

**File:** `internal/patchapply/matcher.go`

- [ ] Define `Matcher` struct
- [ ] Implement `NewMatcher()`
- [ ] Implement `SetThreshold()`
- [ ] Add helper functions (min, max)

### Phase 2: Exact Matching (1 day)

**File:** `internal/patchapply/matcher.go`

- [ ] Implement `findExact()`
- [ ] Implement basic `FindContext()` using exact match only
- [ ] Write tests for exact matching
- [ ] Verify fast-path performance

### Phase 3: Fuzzy Matching (1 day)

**File:** `internal/patchapply/matcher.go`

- [ ] Implement `normalizeLines()`
- [ ] Implement `computeSimilarity()`
- [ ] Implement `findFuzzy()`
- [ ] Update `FindContext()` to try fuzzy after exact fails
- [ ] Write tests for fuzzy matching

### Phase 4: Header Matching (1 day)

**File:** `internal/patchapply/matcher.go`

- [ ] Implement `findHeader()`
- [ ] Implement `findInRange()`
- [ ] Update `FindContext()` to use header hints
- [ ] Write tests for header-based disambiguation

### Phase 5: Testing & Optimization (1 day)

**File:** `internal/patchapply/matcher_test.go`

- [ ] Write comprehensive table-driven tests
- [ ] Write edge case tests (empty, large files, unicode)
- [ ] Write benchmark tests
- [ ] Optimize hot paths (profiling with pprof)
- [ ] Achieve ≥90% coverage

### Phase 6: Analysis & Refinement (1 day)

- [ ] Run `uast parse matcher.go | herr analyze`
- [ ] Run `make lint`
- [ ] Run `gocyclo -over 10 ./internal/patchapply/`
- [ ] Run benchmarks and verify <1ms target
- [ ] Refactor if complexity >10
- [ ] Final test pass with `-race`

**Total Estimated Time:** 6 days

---

## Acceptance Criteria

### Functionality

- [ ] ✅ Finds exact matches in <100μs
- [ ] ✅ Finds fuzzy matches with whitespace tolerance
- [ ] ✅ Uses header to disambiguate multiple occurrences
- [ ] ✅ Configurable similarity threshold (default 0.85)
- [ ] ✅ Returns -1 when no match found

### Quality

- [ ] ✅ Test coverage ≥90%
- [ ] ✅ `make lint` passes (zero errors)
- [ ] ✅ Cyclomatic complexity ≤10 per function
- [ ] ✅ `go test -race` passes
- [ ] ✅ All godoc comments present

### Performance

- [ ] ✅ Exact match: <100μs for typical context (5-10 lines)
- [ ] ✅ Fuzzy match: <1ms for 10k line file
- [ ] ✅ Memory: O(n) where n=file size
- [ ] ✅ No performance regressions

### Integration

- [ ] ✅ Can be used by applier (Feature 2.3)
- [ ] ✅ Works with parser output (Feature 2.1)
- [ ] ✅ Uses `pkg/strutil` functions correctly

---

## Dependencies

### Internal Packages

- `pkg/strutil` - Similarity, NormalizeWhitespace (REQUIRED)
- `internal/patchapply` - Hunk, LineChange types (parser output)

### Standard Library

- `strings` - String manipulation
- `fmt` - Error formatting

---

## Risks & Mitigations

### Risk 1: False Positives

**Impact:** Matcher finds wrong location, applies hunk incorrectly

**Probability:** Low (similarity threshold prevents this)

**Mitigation:**
- Use conservative threshold (0.85 by default)
- Require majority of lines to match
- Use header for disambiguation
- Extensive testing with real-world patches

### Risk 2: Performance

**Impact:** Fuzzy matching too slow for large files

**Probability:** Medium

**Mitigation:**
- Exact match fast path
- Header-based range limiting (±50 lines)
- Early exit on perfect match
- Pre-normalize file lines (cache)
- Benchmark-driven optimization

### Risk 3: Complexity

**Impact:** Complex logic → bugs

**Probability:** Medium

**Mitigation:**
- Keep functions small and focused
- Table-driven tests for all scenarios
- Code review
- Cyclomatic complexity enforcement (≤10)

---

## Success Metrics

### Coverage

- **Target:** ≥90% test coverage
- **Measurement:** `go test -cover ./internal/patchapply/`

### Quality

- **Target:** Zero lint errors
- **Measurement:** `make lint`

### Performance

- **Target:** <1ms for 10k line file
- **Measurement:** `go test -bench=. ./internal/patchapply/`

### Complexity

- **Target:** Cyclomatic complexity ≤10
- **Measurement:** `gocyclo -over 10 ./internal/patchapply/`

---

## Documentation

### Package Documentation

**File:** `internal/patchapply/doc.go` (update)

Add section:
```go
// # Fuzzy Matching
//
// The matcher uses a multi-strategy approach to find hunk context:
//
// 1. Exact match (fast path): Direct string comparison
// 2. Fuzzy match: Whitespace-normalized similarity ≥85%
// 3. Header hints: Use @@ context to disambiguate
//
// Example:
//
//     m := patchapply.NewMatcher(fileLines)
//     pos := m.FindContext(hunk.GetContextLines(), hunk.Header)
//     if pos < 0 {
//         return fmt.Errorf("context not found")
//     }
```

### Function Documentation

```go
// NewMatcher creates a new fuzzy matcher for the given file content.
//
// The matcher pre-normalizes file lines for performance. Subsequent
// calls to FindContext reuse this normalization.
//
// Example:
//
//     fileLines := strings.Split(fileContent, "\n")
//     m := NewMatcher(fileLines)
//     pos := m.FindContext(contextLines, "func Process")
func NewMatcher(fileLines []string) *Matcher { ... }

// FindContext finds the line index where context lines match.
//
// The search uses a multi-strategy approach:
//   1. If header is provided, search near header location (±50 lines)
//   2. Try exact match (fast path)
//   3. Try fuzzy match with whitespace normalization
//   4. Return best match if similarity ≥ threshold
//
// Returns -1 if no match found above the similarity threshold.
//
// Example:
//
//     pos := m.FindContext(contextLines, "@@ func Process")
//     if pos < 0 {
//         log.Fatalf("context not found")
//     }
//     log.Printf("Found context at line %d", pos)
func (m *Matcher) FindContext(contextLines []string, header string) int { ... }
```

---

## References

1. [Spin Tools & Utility Modules](../tools-modules/tools-modules.md)
2. [Tools-Modules ROADMAP](../tools-modules/ROADMAP.md)
3. [FRD-20251012030000-patchapply-parser.md](./FRD-20251012030000-patchapply-parser.md)
4. [pkg/strutil Documentation](../../docs/packages/strutil.md)
5. [AGENTS.md](../../AGENTS.md) - Development workflow

---

## Appendix A: Similarity Algorithm Details

### Levenshtein Distance

The matcher uses the Levenshtein distance algorithm from `pkg/strutil` to compute edit distance between strings.

**Definition:** Minimum number of single-character edits (insertions, deletions, substitutions) to transform string A into string B.

**Example:**
```
Levenshtein("hello", "hallo") = 1  // One substitution: e→a
Levenshtein("foo", "bar") = 3      // Three substitutions
```

### Similarity Score

Converts Levenshtein distance to a 0.0-1.0 similarity score:

```
Similarity(a, b) = 1.0 - (LevenshteinDistance(a, b) / max(len(a), len(b)))
```

**Example:**
```
Similarity("hello", "hallo") = 1.0 - (1 / 5) = 0.80 (80%)
Similarity("hello", "hello") = 1.0 - (0 / 5) = 1.00 (100%)
Similarity("foo", "bar") = 1.0 - (3 / 3) = 0.00 (0%)
```

### Multi-Line Similarity

For multiple context lines, compute average similarity:

```
MultiLineSimilarity(contextLines, windowLines) =
    (Σ Similarity(context[i], window[i])) / len(contextLines)
```

**Example:**
```
Context: ["func main() {", "    return 0"]
Window:  ["func main() {", "    return 1"]

Line 1: Similarity = 1.00 (exact match)
Line 2: Similarity = 0.91 (one character different)
Average: (1.00 + 0.91) / 2 = 0.955 (95.5%)
```

---

## Appendix B: Whitespace Normalization

### NormalizeWhitespace Algorithm

From `pkg/strutil.NormalizeWhitespace()`:

1. Trim leading whitespace
2. Trim trailing whitespace
3. Replace all runs of whitespace (spaces, tabs, newlines) with a single space

**Examples:**
```
"  hello   world  " → "hello world"
"func  main() {  " → "func main() {"
"\t\thello\t\tworld\t" → "hello world"
```

### Why Normalize?

Code formatters, IDEs, and manual edits often change whitespace:
- Tabs ↔ Spaces
- Trailing whitespace added/removed
- Indentation changed (2-space vs 4-space)

Normalization makes matching resilient to these changes while still catching meaningful differences.

---

## Appendix C: Performance Optimization Strategies

### 1. Exact Match Fast Path

Most patches will have exact context matches. Optimize this path:
- Simple string equality checks (no normalization, no similarity computation)
- Expected time: O(n×m) but with very small constant factor
- Typical performance: <100μs for 1000-line file, 5-line context

### 2. Header-Based Range Limiting

If hunk has header, search only ±50 lines around header:
- Reduces search space from O(n) to O(100) in typical cases
- Especially effective for files with multiple similar functions

### 3. Pre-Normalized Cache

Normalize file lines once in `NewMatcher()`:
- Subsequent `FindContext()` calls reuse normalized lines
- Saves repeated normalization for same file
- Trade-off: O(n) memory for O(1) normalization per search

### 4. Early Exit on Perfect Match

If similarity = 1.0, return immediately:
- No need to check remaining positions
- Especially effective when exact match is near start of file

### 5. Sliding Window without Copies

Use slices to create windows, avoid copying:
```go
window := m.normalizedLines[i : i+contextLen]  // O(1) slice operation
```

---

**END OF FRD**
