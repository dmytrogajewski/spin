# Generalization Spec — Clustered by Problem Domain

## Cluster 1: String Wrappers & Duplicates (Remove)

**Problem:** Thin wrappers in `internal/` that delegate directly to `pkg/alg/stringsx` or `pkg/alg/pathx` with no added logic. These add indirection without value.

| Location | Function | Delegates to |
|----------|----------|-------------|
| `internal/tools/truncate.go` | `TruncateHeadTail` | `stringsx.TruncateHeadTail` |
| `internal/tools/truncate.go` | `TruncateLines` | `stringsx.TruncateLines` |
| `internal/tools/truncate.go` | `TruncateOutput` | Two `stringsx` calls |
| `internal/tools/path.go` | `resolvePath` | `pathx.ResolvePath` |
| `internal/tools/fuzzy/collapse.go` | `trimLines` | `stringsx.TrimTrailingPerLine` |
| `internal/tools/fuzzy/whitespace.go` | `collapseWhitespace` | `stringsx.CollapseWhitespace` |
| `internal/tools/fuzzy/anchor.go` | `countNonBlankLines` | `stringsx.CountLines` |
| `internal/tools/web_fetch.go` | `capOutput` | `stringsx.TruncateWithEllipsis` |
| `internal/filesearch/matcher.go` | `Matcher` | `pathx.Matcher` |
| `internal/agent/environment.go` | `filterEnvironment` | `execx.FilterEnvironment` |
| `internal/tui/mapper.go` | `extractString`, `extractIntValue` | `params.ToolParameters.GetStringOr/GetIntOr` |

**Action:** Inline usages → delete wrappers.

---

## Cluster 2: Fuzzy Matching Primitives (Extract to `pkg/alg/search`)

**Problem:** `internal/tools/fuzzy/` and `internal/patchapply/` contain generic search/matching algorithms that are locked inside domain-specific packages.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `findByNormalized` | `fuzzy/whitespace.go` | Map matches in normalized strings back to original offsets |
| `mapNormalizedOffset` | `fuzzy/whitespace.go` | Character-level offset translation |
| `findLineSequence` | `fuzzy/indent.go` | Sequential line-by-line pattern matching |
| `matchesAt` | `fuzzy/indent.go` | Slice position matching |
| `lineOffset`, `lineOffsetEnd` | `fuzzy/indent.go` | Byte offset from line indices |
| `findUniqueMatch` | `tools/edit_file.go` | Match detection with ambiguity handling |
| `Matcher` (fuzzy) | `patchapply/matcher.go` | Multi-strategy matching (exact → fuzzy → header) |

**Action:** Extract to `pkg/alg/search` with generic signatures. Update callers.

---

## Cluster 3: Diff Generation & Parsing (Extract to `pkg/alg/diff`)

**Problem:** Diff utilities are scattered across tool implementations.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `buildSimpleDiff` | `tools/edit_file.go` | Unified diff generation |
| `parseDiffFormat` | `tools/apply_patch.go` | Unified diff parsing |
| `parseDiffLine` | `tools/apply_patch.go` | Single diff line parser |

**Action:** Create `pkg/alg/diff` with `Generate()` and `Parse()` functions.

---

## Cluster 4: LRU/TTL Cache Consolidation (Extend `pkg/alg/ds.Cache`)

**Problem:** Multiple packages implement their own LRU/TTL eviction logic instead of using `pkg/alg/ds.Cache`.

| Location | Pattern |
|----------|---------|
| `internal/agent/cache.go` | `CommandCache` — LRU eviction, Result-specific |
| `internal/contexteng/summarizer/cache.go` | `Cache.evictLRU` — LRU eviction for summaries |
| `internal/memory/scratchpad.go` | `Scratchpad` — LRU eviction for key-value store |
| `internal/safety/policy.go` | `MemoryPolicyStore` — TTL eviction with janitor |
| `internal/lsp/cache.go` | Two-level cache (key + content hash) |

**Action:** Extend `pkg/alg/ds.Cache` with configurable eviction policies (LRU, TTL, content-hash). Migrate internal caches.

---

## Cluster 5: Pattern/Cycle Detection (Extract to `pkg/alg/search`)

**Problem:** `internal/cycle/` contains generic pattern detection algorithms hardcoded to cycle-specific types.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `checkRepeatedTool` | `cycle/detector.go:201` | `detectRepeatPattern[T](items, limit, equals)` |
| `checkSameError` | `cycle/detector.go:299` | Same pattern as above — duplicated |
| `checkOscillation` | `cycle/detector.go:265` | `detectAlternating[T comparable](items)` |
| `detectOscillatingTools` | `cycle/patterns.go:282` | Duplicate of `checkOscillation` |
| `getRecentSnapshots` | `cycle/detector.go:149` | `sliceRecent[T](items, limit)` — appears 3 times |

**Action:** Extract as generic functions in `pkg/alg/search`:
- `DetectRepeat[T](items []T, window int, eq func(T, T) bool) bool`
- `DetectAlternating[T comparable](items []T) bool`
- Move `sliceRecent` to `pkg/alg/collections.TailN` (already exists).

---

## Cluster 6: Similarity Algorithms (Extend `pkg/alg/similarity`)

**Problem:** Multi-strategy similarity computation is duplicated in ACE and cycle packages.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `calculateSimilarity` | `ace/refine/merge.go:118` | Cosine + Jaccard fallback |
| `calculateSimilarity` | `cycle/similarity.go:10` | Jaccard-based |
| `FindMergeCandidates` | `ace/refine/merge.go:49` | O(n²) similarity pairing — generic dedup |
| `ExtractConcepts` | `ace/trajectory/analysis.go:99` | Word-frequency concept extraction |

**Action:** Add `MultiStrategySimilarity()` and `FindSimilarPairs[T]()` to `pkg/alg/similarity`.

---

## Cluster 7: Generic Collection Operations (Add to `pkg/alg/collections`)

**Problem:** Several packages implement collection operations that should be generic utilities.

| Function | Source | Generic Signature |
|----------|--------|-------------------|
| `FilterByQuality` | `ace/reflector/validator.go:46` | `Filter[T](items []T, pred func(T) bool) []T` |
| `ValidateBatch` | `ace/reflector/validator.go:32` | `ValidateAll[T](items []T, validate func(T) error) []error` |
| `getRecentSnapshots` | `cycle/detector.go:149` | Already exists as `TailN` |
| `clampViewport` | `tools/web_screenshot.go` | `Clamp[T constraints.Ordered](val, min, max T) T` |
| `Timeline.matchesFilter` | `ui/blocks/timeline.go:500` | `FilterChain[T](items []T, preds ...func(T) bool) []T` |

**Action:** Add `Filter`, `ValidateAll`, `Clamp` to `pkg/alg/collections`.

---

## Cluster 8: Lazy Init / Double-Check Locking (Already in `pkg/alg/ds/syncmap`)

**Problem:** Multiple packages implement double-checked locking for lazy initialization.

| Location | Pattern |
|----------|---------|
| `conversation/manager.go:82` | `GetOrCreate` with lock |
| `lsp/manager.go:64-85` | `serverForLanguage` with health check |

**Action:** Migrate to `pkg/alg/ds/syncmap.Map.GetOrCreate` which already implements this.

---

## Cluster 9: Text Analysis Utilities (Add to `pkg/alg/stringsx`)

**Problem:** Content analysis functions are scattered across packages.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `normalizeEscapes` | `tools/fuzzy/escape.go` | Escape sequence normalization |
| `detectTruncation` | `tools/write_file.go` | Unmatched delimiter detection |
| `isErrorMessage` | `contexteng/compress/classifier.go:128` | Error pattern detection |
| `hasCodeBlock` | `contexteng/compress/classifier.go:162` | Code block detection |
| `midEllipsize` | `ui/blocks/renderer.go:504` | Middle-ellipsis truncation |
| `SummarizeError` | `contexteng/observation/summarizer.go:137` | Error output truncation |

**Action:** Extract to `pkg/alg/stringsx` as generic text analysis utilities.

---

## Cluster 10: UI Text Measurement (Extract to `pkg/ui/textwidth`)

**Problem:** Grapheme/width calculation utilities in `internal/ui/blocks/renderer.go` are reusable across UI components.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `extractGraphemes` | `ui/blocks/renderer.go:520` | Unicode grapheme extraction |
| `calculateTotalWidth` | `ui/blocks/renderer.go:540` | Display width calculation |
| `buildLeftPart`, `buildRightPart` | `ui/blocks/renderer.go:560-587` | Width-aware text building |
| `calcGutterWidth` | `ui/blocks/renderer.go:591` | Line-number-to-width mapping |

**Action:** Extract to `pkg/ui/textwidth` for shared use across UI components.

---

## Cluster 11: Generic Registry/Strategy Patterns (Extract to `pkg/alg/ds`)

**Problem:** Registry + strategy dispatch patterns are reimplemented across multiple packages.

| Location | Pattern |
|----------|---------|
| `ui/blocks/tool_params.go` | `ParamsFormatterRegistry` — type→formatter map |
| `internal/state/state.go` | State machine with validated transitions |
| `ui/blocks/metadata.go` | `MetadataCodec` — repeated JSON marshal/unmarshal |

**Action:** Create generic `Registry[K comparable, V any]` and `StateMachine[S comparable]` in `pkg/alg/ds`.

---

## Cluster 12: Storage Promotion (Promote to `pkg/`)

**Problem:** High-quality generic storage utilities in `internal/` could serve external consumers.

| Type | Source | Why Promote |
|------|--------|-------------|
| `FileStore[T]` | `internal/storage/store.go` | Already uses Go generics, fully generic |
| `AtomicWriteFile` | `internal/storage/atomic.go` | Fundamental file safety utility |
| `Index` (resilient) | `internal/session/index.go` | Auto-rebuild from metadata pattern |

**Action:** Promote to `pkg/storage` or `pkg/alg/pathx`.

---

## Cluster 13: Concurrency Utilities (Add to `pkg/alg/concurrency`)

**Problem:** Reusable concurrency patterns in internal packages.

| Function | Source | Generic Use |
|----------|--------|-------------|
| `printChunksWithCoalescing` | `ui/output/printer.go` | Timer-based chunk coalescing |
| `startSignalHandler` | `examples/*/main.go` | Duplicated signal handling |
| `CachingSummarizer` | `contexteng/summarizer/caching.go` | Generic cache decorator |
| `DoomLoopGuard` | `agent/harness/doomloop.go` | Fingerprint-based loop detection |

**Action:** Extract generic patterns to `pkg/alg/concurrency`.

---

## Priority Order

| Priority | Cluster | Effort | Impact |
|----------|---------|--------|--------|
| **P0** | 1. String Wrappers (Remove) | Low | Reduces indirection |
| **P0** | 8. Lazy Init (Migrate) | Low | Uses existing `syncmap.GetOrCreate` |
| **P1** | 4. LRU/TTL Cache | Medium | Consolidates 5 implementations |
| **P1** | 5. Pattern Detection | Medium | Deduplicates cycle detection |
| **P1** | 7. Collection Operations | Low | Small generic additions |
| **P2** | 2. Fuzzy Matching | High | Major extraction to `pkg/alg/search` |
| **P2** | 3. Diff Utilities | Medium | New `pkg/alg/diff` package |
| **P2** | 6. Similarity | Medium | Extends existing package |
| **P2** | 9. Text Analysis | Medium | Extends `stringsx` |
| **P3** | 10. UI Text Measurement | Low | New `pkg/ui/textwidth` |
| **P3** | 11. Registry/Strategy | Medium | New generic types |
| **P3** | 12. Storage Promotion | Low | Move + re-export |
| **P3** | 13. Concurrency | Medium | Generic patterns |
