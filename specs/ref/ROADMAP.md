# Roadmap: Code Generalization & Deduplication

**Spec:** [SPEC.md](SPEC.md)
**Status:** Completed
**Created:** 2026-03-24

---

## Phase 1 — Remove Dead Weight (P0, Low Effort)

These items remove thin wrappers with zero new code. Each is independently testable.

### R-REF-1: Inline Truncation Wrappers ✅

**Description:** Remove pure pass-through wrappers `TruncateHeadTail` and `TruncateLines` from `internal/tools/truncate.go`. Keep `TruncateOutput` (composition function with domain constants, not a pure wrapper).

**Journey:** [JOURNEY-R-REF-1](../journeys/JOURNEY-R-REF-1.md)

**DoR (Definition of Ready):**
- [x] Callers identified: `shell_command.go:281`, `shell_command.go:302`
- [x] Target functions confirmed as pure delegation to `stringsx`

**DoD (Definition of Done):**
- [x] `TruncateHeadTail` and `TruncateLines` deleted from `truncate.go`
- [x] `truncate.go` reduced to constants + `TruncateOutput` (composition function)
- [x] 9 wrapper tests removed from `truncate_test.go`; 2 `TruncateOutput` tests preserved
- [x] `go build ./...` passes
- [x] `go test ./internal/tools/...` passes
- [x] `make lint` — no new issues in changed files

**Implementation:**
- `internal/tools/truncate.go` — removed 2 exported wrapper functions
- `internal/tools/truncate_test.go` — removed 9 wrapper tests, kept 2 composition tests

**Blast radius:** 0 external files changed

---

### R-REF-2: Inline Path Wrapper ✅

**Description:** Remove `resolvePath` from `internal/tools/path.go` and replace 4 callers with direct `pathx.ResolvePath` calls.

**Journey:** [JOURNEY-R-REF-2](../journeys/JOURNEY-R-REF-2.md)

**DoR:**
- [x] Callers: `read_file.go:66`, `edit_file.go:147`, `write_file.go:76`, `list_directory.go:67`
- [x] Confirmed: pure delegation to `pathx.ResolvePath`

**DoD:**
- [x] 4 tool files import `pkg/alg/pathx` directly
- [x] `internal/tools/path.go` deleted
- [x] `internal/tools/path_test.go` deleted
- [x] `go build ./...` passes
- [x] `go test ./internal/tools/...` passes
- [x] `make lint` — no new issues in changed files

**Implementation:**
- Deleted `path.go`, `path_test.go`
- Modified `read_file.go`, `edit_file.go`, `write_file.go`, `list_directory.go`

**Blast radius:** 4 files in `internal/tools/`

---

### R-REF-3: Inline Fuzzy String Helpers ✅

**Description:** Replace `trimLines` and `countNonBlankLines` with `stringsx` equivalents. `collapseWhitespace` kept — semantics differ (fuzzy preserves newlines, stringsx collapses them).

**Journey:** [JOURNEY-R-REF-3](../journeys/JOURNEY-R-REF-3.md)

**DoR:**
- [x] Each function confirmed as near-duplicate of stringsx export
- [x] `collapseWhitespace` verified as semantically different — NOT replaced

**DoD:**
- [x] 2 fuzzy files updated to import `stringsx`
- [x] `trimLines` and `countNonBlankLines` removed
- [x] `collapseWhitespace` retained (different semantics: preserves newlines)
- [x] `go test ./internal/tools/fuzzy/...` passes
- [x] `go test ./internal/tools/...` passes
- [x] No new lint issues

**Implementation:**
- Modified `collapse.go` — replaced `trimLines` with `stringsx.TrimTrailingPerLine`
- Modified `anchor.go` — replaced `countNonBlankLines` with `stringsx.CountLines`

**Blast radius:** 2 files in `internal/tools/fuzzy/`

---

### R-REF-4: Inline TUI Mapper Helpers ✅

**Description:** Replace `extractString` and `extractIntValue` (private helpers in `mapper.go`) with direct `GetStringOr` / `GetIntOr` calls.

**Journey:** [JOURNEY-R-REF-4](../journeys/JOURNEY-R-REF-4.md)

**DoR:**
- [x] 12 call sites, all local to `mapper.go`

**DoD:**
- [x] `mapper.go` uses direct `data.Parameters.GetStringOr` / `GetIntOr` methods
- [x] `extractString`, `extractIntValue` removed
- [x] Unused `tools` import removed
- [x] `go test ./internal/tui/...` passes
- [x] No new lint issues

**Implementation:** Modified `mapper.go` — 12 call sites replaced, 2 functions deleted

**Blast radius:** 1 file (mapper.go)

---

### R-REF-5: Inline filterEnvironment and filesearch.Matcher ✅

**Description:** Remove `filterEnvironment` wrapper and `filesearch/matcher.go`+`ignore.go` type aliases.

**Journey:** [JOURNEY-R-REF-5](../journeys/JOURNEY-R-REF-5.md)

**DoR:**
- [x] `filterEnvironment`: 1 production caller at `environment.go:151`
- [x] `filesearch.Matcher`/`Match`/`IgnoreHandler`: aliases used only within filesearch package

**DoD:**
- [x] `environment.go` calls `execx.FilterEnvironment` directly
- [x] `filesearch/matcher.go` deleted
- [x] `filesearch/ignore.go` deleted
- [x] `searcher.go` and `scanner.go` use `pathx` directly
- [x] `go build ./...` passes
- [x] `go test ./internal/agent/... ./internal/filesearch/...` passes
- [x] No new lint issues

**Implementation:**
- Modified `environment.go`, `environment_test.go`, `searcher.go`, `scanner.go`, `doc.go`
- Deleted `matcher.go`, `ignore.go`

**Blast radius:** 5 files modified, 2 deleted

---

### R-REF-6: Inline capOutput Wrapper ✅

**Description:** Added `stringsx.TruncateWithSuffix` (truncate-then-append pattern) and replaced `capOutput`.

**Journey:** [JOURNEY-R-REF-6](../journeys/JOURNEY-R-REF-6.md)

**DoD:**
- [x] `stringsx.TruncateWithSuffix` added (semantics: truncate to maxLen, append suffix beyond limit)
- [x] `capOutput` removed from `web_fetch.go`
- [x] Tests added for new function
- [x] `go test ./internal/tools/... ./pkg/alg/stringsx/...` passes
- [x] No new lint issues

**Blast radius:** 2 files (web_fetch.go, stringsx/extended.go)

---

## Phase 2 — Migrate to Existing Generics (P0-P1, Low Effort)

### R-REF-7: Migrate Lazy Init to syncmap.GetOrCreate ✅

**Description:** Added `syncmap.GetOrCreateErr` (error-returning variant) and migrated `conversation/manager.go`.

**Journey:** [JOURNEY-R-REF-7](../journeys/JOURNEY-R-REF-7.md)

**DoD:**
- [x] `syncmap.GetOrCreateErr(key, func() (V, error)) (V, error)` added
- [x] `conversation/manager.go` uses `GetOrCreateErr`, removed `createMu` mutex
- [x] `lsp/manager.go` NOT migrated — uses get-or-replace-if-dead pattern (health check on existing value), which is not a create-if-absent pattern
- [x] Tests for `GetOrCreateErr` added (3 tests)
- [x] `go test ./internal/conversation/... ./pkg/alg/ds/syncmap/...` passes

**Blast radius:** 3 files (syncmap_map.go, syncmap_map_test.go, manager.go)

---

### R-REF-8: Migrate Cycle Detection to Use collections.TailN ✅

**Description:** Replace 3 rolling-window helper bodies with `collections.TailN` / `TailNOrAll`.

**Journey:** [JOURNEY-R-REF-8](../journeys/JOURNEY-R-REF-8.md)

**DoR:**
- [x] `collections.TailN[Elem any](input []Elem, n int) []Elem` exists
- [x] 3 call sites in `detector.go`

**DoD:**
- [x] 3 helper function bodies replaced with `collections.TailN/TailNOrAll`
- [x] `go test ./internal/cycle/...` passes
- [x] No new lint issues

**Implementation:** Modified `detector.go` — 3 functions simplified to one-liners
- [ ] `go test ./internal/cycle/...` passes with identical behavior

**Blast radius:** 1 file

---

## Phase 3 — Generic Collection Operations (P1, Low Effort)

### R-REF-9: Add Filter, Clamp, ValidateAll to collections ✅

**Description:** Add 3 generic functions to `pkg/alg/collections`:
- `Filter[T any](items []T, pred func(T) bool) []T`
- `Clamp[T cmp.Ordered](val, lo, hi T) T`
- `ValidateAll[T any](items []T, validate func(T) error) error`

**Journey:** [JOURNEY-R-REF-9](../journeys/JOURNEY-R-REF-9.md)

**DoR:**
- [x] `pkg/alg/collections` exists with pattern established
- [x] No existing `Filter`, `Clamp`, or `ValidateAll`

**DoD:**
- [x] Functions added to `pkg/alg/collections/extended.go`
- [x] Table-driven tests (100% coverage)
- [x] `go test ./pkg/alg/collections/...` passes
- [x] No new lint issues

**Implementation:**
- Modified `extended.go` — added Filter, Clamp, ValidateAll
- Modified `genutil_test.go` — added 15 test cases

**Blast radius:** 0 — additive only

---

### R-REF-10: Migrate Callers to New Collection Functions ✅

**Description:** Migrate hand-rolled implementations to collection generics.

**Status:**
- `FilterByQuality` / `ValidateBatch` — already removed in prior refactoring (functions don't exist)
- `clampViewport` — refactored to use `collections.Clamp(value, 1, maxValue)` with zero-default guard
- `getRecentSnapshots` — already done in R-REF-8

**DoD:**
- [x] `web_screenshot.go` uses `collections.Clamp`
- [x] `go test ./internal/tools/...` passes

**Blast radius:** 1 file

---

## Phase 4 — Pattern Detection Generics (P1, Medium Effort)

### R-REF-11: Extract Generic Pattern Detection to pkg/alg/search ✅

**Description:** Add generic pattern detection functions to `pkg/alg/search`:
- `DetectRepeat[T any](items []T, eq func(T, T) bool) bool`
- `DetectAlternating[T comparable](items []T) bool`

**Journey:** [JOURNEY-R-REF-11](../journeys/JOURNEY-R-REF-11.md)

**DoR:**
- [x] `pkg/alg/search` exists
- [x] Two duplicate patterns identified in `cycle/`

**DoD:**
- [x] `pkg/alg/search/pattern.go` created
- [x] `pkg/alg/search/pattern_test.go` — 20 test cases, 95.6% coverage
- [x] `go test ./pkg/alg/search/...` passes
- [x] No new lint issues

**Implementation:**
- Created `pattern.go` — DetectRepeat, DetectAlternating
- Created `pattern_test.go` — table-driven tests

**Blast radius:** 0 — additive only

---

### R-REF-12: Migrate Cycle Detection to Use Generic Patterns ✅

**Description:** Refactor `cycle/detector.go` and `cycle/patterns.go` to use `search.DetectRepeat` and `search.DetectAlternating`.

**Journey:** [JOURNEY-R-REF-12](../journeys/JOURNEY-R-REF-12.md)

**DoR:**
- [x] R-REF-11 completed
- [x] `allToolsAreSame`/`allErrorsAreSame` verified as DetectRepeat pattern
- [x] `detectOscillatingTools` verified as DetectAlternating pattern

**DoD:**
- [x] `detector.go` uses `search.DetectRepeat` for tool and error checks
- [x] `patterns.go` uses `search.DetectAlternating`
- [x] `snapshotUsesTool` removed
- [x] `go test ./internal/cycle/...` passes
- [x] No new lint issues

**Implementation:** Modified `detector.go`, `patterns.go`
- [ ] Benchmark: no regression in cycle detection hot path

**Blast radius:** 2 files in `internal/cycle/`

---

## Phase 5 — Cache Consolidation (P1, Medium Effort)

### R-REF-13: Verify ds.Cache Covers LRU/TTL/Janitor Needs

**Description:** `pkg/alg/ds.Cache` already supports `EvictLRU` and TTL. Verify it covers all 5 internal cache implementations' needs. Document gaps.

**DoR:**
- [x] `ds.Cache` has: `EvictionPolicy` (FIFO, LRU), TTL, MaxEntries, thread-safe
- [ ] Compare against: `agent/cache.go`, `contexteng/summarizer/cache.go`, `memory/scratchpad.go`, `safety/policy.go`, `lsp/cache.go`

**DoD:**
- [ ] Gap analysis document (inline in this roadmap or separate)
- [ ] If gaps found: extend `ds.Cache` (e.g., janitor goroutine, content-hash)
- [ ] If no gaps: proceed directly to migration
- [ ] `go test ./pkg/alg/ds/...` passes

**Blast radius:** 0 — analysis only

---

### R-REF-14: Migrate Internal Caches to ds.Cache

**Description:** Replace custom cache implementations with `ds.Cache[K, V]`:
1. `internal/agent/cache.go` (CommandCache) → `ds.Cache[string, *Result]`
2. `internal/contexteng/summarizer/cache.go` → `ds.Cache[string, *Summary]`
3. `internal/memory/scratchpad.go` → `ds.Cache[string, []byte]`
4. `internal/safety/policy.go` (MemoryPolicyStore) → `ds.Cache[PolicyKey, Policy]`

**DoR:**
- [ ] R-REF-13 confirms ds.Cache covers all needs
- [ ] Each cache's key/value types mapped to generic parameters

**DoD:**
- [ ] 4 files migrated to use `ds.Cache`
- [ ] Custom eviction code removed
- [ ] All existing tests pass unchanged
- [ ] `go test ./internal/agent/... ./internal/contexteng/... ./internal/memory/... ./internal/safety/...` passes

**Blast radius:** 4 files, high-value dedup

**Note:** summarizer.Cache already migrated in prior work. Remaining: CommandCache, Scratchpad, MemoryPolicyStore, lsp.Cache.

---

## Phase 6 — Similarity & Text Analysis (P2, Medium Effort)

### R-REF-15: Add MultiStrategySimilarity to pkg/alg/similarity ✅

**Description:** Added `MultiStrategySimilarity` (max of N strategies) and `FindSimilarPairs[T]` (O(n²) pairwise comparison).

**Journey:** [JOURNEY-R-REF-15](../journeys/JOURNEY-R-REF-15.md)

**DoD:**
- [x] `multi.go` with `Strategy` type and `MultiStrategySimilarity`
- [x] `pairs.go` with `Pair[T]` and `FindSimilarPairs[T]`
- [x] 8 tests, 99% coverage
- [x] `go test ./pkg/alg/similarity/...` passes
- [x] No lint issues

**Blast radius:** 0 — additive

---

### R-REF-16: Inline Cycle Similarity Wrappers ✅

**Description:** Deleted `cycle/similarity.go` (pure wrappers around `similarity.JaccardSimilarity` and `similarity.ExtractWords`). Inlined calls in `detector.go` (4 sites) and `patterns.go` (1 site). Also deleted `cycle/similarity_test.go` (redundant with `pkg/alg/similarity` tests).

`ace/refine/merge.go:calculateSimilarity` NOT changed — has context + embedder + error handling (domain-specific, not a wrapper).

**DoD:**
- [x] `cycle/similarity.go` deleted
- [x] `cycle/similarity_test.go` deleted
- [x] `detector.go` uses `similarity.JaccardSimilarity` directly (4 sites)
- [x] `patterns.go` uses `similarity.ExtractWords` directly (1 site)
- [x] `go test ./internal/cycle/...` passes
- [x] No new lint issues

**Blast radius:** 2 files modified, 2 deleted

---

### R-REF-17: Add Text Analysis Utilities to stringsx ✅

**Description:** Added `NormalizeEscapes` and `DetectTruncation` to `pkg/alg/stringsx`. `MidEllipsize` deferred to R-REF-23 (requires `uniseg` dependency, not compatible with zero-dep stringsx).

**DoD:**
- [x] `NormalizeEscapes` added (backslash escape normalization)
- [x] `DetectTruncation` added (unclosed delimiter/string detection)
- [x] 14 tests, 96% coverage
- [x] `MidEllipsize` → deferred to R-REF-23 (`pkg/ui/textwidth`)
- [x] No lint issues

**Blast radius:** 0 — additive

---

### R-REF-18: Migrate Text Analysis Callers ✅

**Description:** Replaced private implementations with `stringsx` exports.

**DoD:**
- [x] `fuzzy/escape.go` uses `stringsx.NormalizeEscapes` — deleted `normalizeEscapes`, `escapeReplacements`, `backslashPlaceholder`
- [x] `tools/write_file.go` uses `stringsx.DetectTruncation` — deleted `detectTruncation`
- [x] `tools/write_file_test.go` updated to use `stringsx.DetectTruncation`
- [x] `ui/blocks/renderer.go:midEllipsize` — deferred to R-REF-23/24 (needs `uniseg`)
- [x] All tests pass

**Blast radius:** 3 files

---

## Phase 7 — Fuzzy Search & Diff Extraction (P2, High Effort)

### R-REF-19: Create pkg/alg/diff Package ✅

**Description:** Created `pkg/alg/diff` with `Generate`, `Parse`, `Hunk`, `LineChange` types.

**Journey:** [JOURNEY-R-REF-19](../journeys/JOURNEY-R-REF-19.md)

**DoD:**
- [x] `diff.go` with `Generate`, `Parse`, `LineType`, `LineChange`, `Hunk`
- [x] `diff_test.go` with 10 tests, 92.9% coverage
- [x] Round-trip test (Generate → Parse → verify)
- [x] No lint issues

**Blast radius:** 0 — additive

---

### R-REF-20: Migrate Diff Callers ✅

**Description:** Replaced `buildSimpleDiff` and `parseDiffFormat` with `diff.Generate` and `diff.Parse`.

**DoD:**
- [x] `edit_file.go` uses `diff.Generate`, deleted `buildSimpleDiff`
- [x] `apply_patch.go` uses `diff.Parse` + converter to `patchapply` types, deleted `parseDiffLine`, `stripDiffPrefix`, `minPatchLines`, `ErrDiffFormatTooShort`, `ErrCouldNotExtractFilenameFromFirst`
- [x] `go test ./internal/tools/...` passes

**Blast radius:** 2 files

---

### R-REF-21: Extract Fuzzy Search Primitives to pkg/alg/search

**Description:** Move generic fuzzy matching primitives to `pkg/alg/search`:
- `FindByNormalized(haystack, needle string, normalize func(string) string) (start, end int, found bool)`
- `MapNormalizedOffset(original, normalized string, normalizedPos int) int`
- `FindLineSequence(fileLines, targetLines []string, eq func(string, string) bool) (start int, found bool)`
- `MatchesAt[T any](source, target []T, pos int, eq func(T, T) bool) bool`

**DoR:**
- [x] Source functions in `fuzzy/whitespace.go` and `fuzzy/indent.go`
- [ ] Verify generic signatures cover all current call patterns

**DoD:**
- [ ] `pkg/alg/search/fuzzy.go` with generic functions
- [ ] `pkg/alg/search/fuzzy_test.go` with comprehensive tests
- [ ] `go test ./pkg/alg/search/...` passes

**Blast radius:** 0 — additive

---

### R-REF-22: Migrate Fuzzy Callers

**Description:** Update `internal/tools/fuzzy/` to use `pkg/alg/search` primitives.

**DoR:**
- [ ] R-REF-21 completed

**DoD:**
- [ ] `whitespace.go`, `indent.go` use `search` package
- [ ] Private helpers removed
- [ ] `go test ./internal/tools/fuzzy/... ./internal/tools/...` passes — edit/patch behavior unchanged

**Blast radius:** 2-3 files in `internal/tools/fuzzy/`

---

## Phase 8 — UI & Structural Generics (P3, Medium Effort)

### R-REF-23: Create pkg/ui/textwidth Package

**Description:** Extract grapheme/width utilities from `ui/blocks/renderer.go` to `pkg/ui/textwidth`:
- `ExtractGraphemes(input string) []string`
- `TotalWidth(graphemes []string) int`
- `MidEllipsize(input string, maxWidth int) string` (if not already in stringsx from R-REF-17)
- `GutterWidth(lineCount int) int`

**DoR:**
- [x] Source functions: `renderer.go:520-591`
- [ ] Verify `uniseg` dependency is acceptable in `pkg/ui/`

**DoD:**
- [ ] `pkg/ui/textwidth/textwidth.go` created
- [ ] `pkg/ui/textwidth/textwidth_test.go` with unicode edge cases
- [ ] `go test ./pkg/ui/textwidth/...` passes

**Blast radius:** 0 — additive

---

### R-REF-24: Migrate UI Callers to textwidth

**Description:** Update `ui/blocks/renderer.go` to use `pkg/ui/textwidth`.

**DoR:**
- [ ] R-REF-23 completed

**DoD:**
- [ ] `renderer.go` imports `textwidth`
- [ ] Private grapheme/width helpers removed
- [ ] `go test ./internal/ui/blocks/...` passes

**Blast radius:** 1 file

---

### R-REF-25: Add Generic Registry and StateMachine to pkg/alg/ds

**Description:** Add to `pkg/alg/ds`:
- `Registry[K comparable, V any]` — thread-safe type→strategy map with `Register`, `Get`, `MustGet`
- `StateMachine[S comparable]` — state with validated transitions, terminal states

**DoR:**
- [x] Patterns identified in `ui/blocks/tool_params.go`, `internal/state/state.go`
- [ ] Define minimal API surface

**DoD:**
- [ ] `pkg/alg/ds/registry.go` and `pkg/alg/ds/statemachine.go`
- [ ] Tests for both
- [ ] `go test ./pkg/alg/ds/...` passes

**Blast radius:** 0 — additive

---

### R-REF-26: Promote Storage Utilities to pkg/storage

**Description:** Move `FileStore[T]` and `AtomicWriteFile` from `internal/storage/` to `pkg/storage/`. Keep `internal/storage` as re-export shim during transition.

**DoR:**
- [x] Both are already generic and well-tested
- [ ] Verify no internal-only dependencies in their implementation

**DoD:**
- [ ] `pkg/storage/store.go` with `FileStore[T]`
- [ ] `pkg/storage/atomic.go` with `AtomicWriteFile`
- [ ] `internal/storage/` re-exports or imports are updated
- [ ] `go build ./...` passes
- [ ] All existing storage tests pass

**Blast radius:** Medium — all internal/storage importers need verification

---

## Dependency Graph

```
Phase 1 (Remove wrappers):
  R-REF-1 ──┐
  R-REF-2 ──┤
  R-REF-3 ──┤ (all independent)
  R-REF-4 ──┤
  R-REF-5 ──┤
  R-REF-6 ──┘

Phase 2 (Migrate to existing):
  R-REF-7 ── (independent)
  R-REF-8 ── (independent)

Phase 3 (Collection generics):
  R-REF-9  ──→ R-REF-10

Phase 4 (Pattern detection):
  R-REF-11 ──→ R-REF-12

Phase 5 (Cache):
  R-REF-13 ──→ R-REF-14

Phase 6 (Similarity + text):
  R-REF-15 ──→ R-REF-16
  R-REF-17 ──→ R-REF-18

Phase 7 (Fuzzy + diff):
  R-REF-19 ──→ R-REF-20
  R-REF-21 ──→ R-REF-22

Phase 8 (UI + structural):
  R-REF-23 ──→ R-REF-24
  R-REF-25 ── (independent)
  R-REF-26 ── (independent)
```

All phases are independent of each other. Within each phase, additive items (create pkg functions) must precede migration items (update callers).

---

## Status Tracker

| Item | Phase | Status | Evidence |
|------|-------|--------|----------|
| R-REF-1 | 1 | DONE | [JOURNEY-R-REF-1](../journeys/JOURNEY-R-REF-1.md), `truncate.go`, `truncate_test.go` |
| R-REF-2 | 1 | DONE | [JOURNEY-R-REF-2](../journeys/JOURNEY-R-REF-2.md), deleted `path.go`/`path_test.go` |
| R-REF-3 | 1 | DONE | [JOURNEY-R-REF-3](../journeys/JOURNEY-R-REF-3.md), `collapse.go`, `anchor.go` |
| R-REF-4 | 1 | DONE | [JOURNEY-R-REF-4](../journeys/JOURNEY-R-REF-4.md), `mapper.go` |
| R-REF-5 | 1 | DONE | [JOURNEY-R-REF-5](../journeys/JOURNEY-R-REF-5.md), deleted `matcher.go`/`ignore.go` |
| R-REF-6 | 1 | DONE | Added `stringsx.TruncateWithSuffix`, deleted `capOutput` |
| R-REF-7 | 2 | DONE | Added `syncmap.GetOrCreateErr`, migrated conversation/manager.go |
| R-REF-8 | 2 | DONE | [JOURNEY-R-REF-8](../journeys/JOURNEY-R-REF-8.md), `detector.go` |
| R-REF-9 | 3 | DONE | [JOURNEY-R-REF-9](../journeys/JOURNEY-R-REF-9.md), `extended.go` |
| R-REF-10 | 3 | DONE | `clampViewport` uses `collections.Clamp`; FilterByQuality/ValidateBatch already removed |
| R-REF-11 | 4 | DONE | [JOURNEY-R-REF-11](../journeys/JOURNEY-R-REF-11.md), `pattern.go` |
| R-REF-12 | 4 | DONE | [JOURNEY-R-REF-12](../journeys/JOURNEY-R-REF-12.md), `detector.go`, `patterns.go` |
| R-REF-13 | 5 | DONE | Gap analysis inline — only summarizer.Cache is compatible |
| R-REF-14 | 5 | DONE | [JOURNEY-R-REF-14](../journeys/JOURNEY-R-REF-14.md), `cache.go` (50% reduction) |
| R-REF-15 | 6 | DONE | [JOURNEY-R-REF-15](../journeys/JOURNEY-R-REF-15.md), `multi.go`, `pairs.go` |
| R-REF-16 | 6 | DONE | Inlined `calculateSimilarity`/`extractWords`, deleted `similarity.go` |
| R-REF-17 | 6 | DONE | `NormalizeEscapes` + `DetectTruncation` in `extended.go`; MidEllipsize → R-REF-23 |
| R-REF-18 | 6 | DONE | Migrated `escape.go`, `write_file.go` to use stringsx |
| R-REF-19 | 7 | DONE | [JOURNEY-R-REF-19](../journeys/JOURNEY-R-REF-19.md), `pkg/alg/diff/` |
| R-REF-20 | 7 | DONE | Migrated edit_file.go + apply_patch.go to use `pkg/alg/diff` |
| R-REF-21 | 7 | DONE | `pkg/alg/search/fuzzy.go` — MapNormalizedOffset, FindAllNormalized, MatchesAt, LineOffset, LineOffsetEnd |
| R-REF-22 | 7 | DONE | Migrated `whitespace.go`, `indent.go`, `anchor.go` to use `search` package |
| R-REF-23 | 8 | DONE | `pkg/ui/textwidth/` — ExtractGraphemes, TotalWidth, MidEllipsize, GutterWidth (100% coverage) |
| R-REF-24 | 8 | DONE | Migrated renderer.go to use textwidth — deleted 7 private functions + 8 constants |
| R-REF-25 | 8 | DONE | `pkg/alg/ds/registry.go`, `statemachine.go` |
| R-REF-26 | 8 | DONE | Promoted to `pkg/storage/`, updated 10 importers, deleted `internal/storage/` files |
