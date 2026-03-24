# Generalization Opportunities

## Dedup opportunities

### 1. internal/tools

   **Function:** `TruncateHeadTail`
   **Position:** internal/tools/truncate.go
   **Findings:** Thin wrapper around `stringsx.TruncateHeadTail()` — adds no logic
   **Could replace:** Nothing — should be removed; callers use `pkg/alg/stringsx` directly

   **Function:** `TruncateLines`
   **Position:** internal/tools/truncate.go
   **Findings:** Thin wrapper around `stringsx.TruncateLines()` — adds no logic
   **Could replace:** Nothing — should be removed; callers use `pkg/alg/stringsx` directly

   **Function:** `TruncateOutput`
   **Position:** internal/tools/truncate.go
   **Findings:** Combines two `stringsx` calls — convenience only
   **Could replace:** Nothing — should be removed; callers compose `stringsx` directly

   **Function:** `resolvePath`
   **Position:** internal/tools/path.go
   **Findings:** Thin wrapper around `pathx.ResolvePath()` — adds no logic
   **Could replace:** Nothing — should be removed; callers use `pkg/alg/pathx` directly

   **Function:** `trimLines`
   **Position:** internal/tools/fuzzy/collapse.go
   **Findings:** Trims trailing whitespace from each line — duplicates `stringsx.TrimTrailingPerLine()`
   **Could replace:**
     - pkg/alg/stringsx:TrimTrailingPerLine

   **Function:** `collapseWhitespace`
   **Position:** internal/tools/fuzzy/whitespace.go
   **Findings:** Regex-based whitespace collapse — nearly identical to `stringsx.CollapseWhitespace()`
   **Could replace:**
     - pkg/alg/stringsx:CollapseWhitespace

   **Function:** `countNonBlankLines`
   **Position:** internal/tools/fuzzy/anchor.go
   **Findings:** Could use `stringsx.CountLines()` with a non-blank predicate
   **Could replace:**
     - pkg/alg/stringsx:CountLines

   **Function:** `capOutput`
   **Position:** internal/tools/web_fetch.go
   **Findings:** Text truncation with marker — similar to `stringsx.TruncateWithEllipsis`
   **Could replace:**
     - pkg/alg/stringsx:TruncateWithEllipsis

   **Function:** `extractString`, `extractIntValue`
   **Position:** internal/tui/mapper.go:905-911
   **Findings:** Could be replaced by existing `ToolParameters.GetStringOr`, `GetIntOr`
   **Could replace:**
     - pkg/llmutil/params:ToolParameters.GetStringOr
     - pkg/llmutil/params:ToolParameters.GetIntOr

### 2. internal/tools/fuzzy (extractable to pkg/alg)

   **Function:** `findByNormalized`
   **Position:** internal/tools/fuzzy/whitespace.go
   **Findings:** (a)(c) Maps matches in normalized strings back to original offsets — generic utility for any fuzzy matching that normalizes before searching
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `mapNormalizedOffset`
   **Position:** internal/tools/fuzzy/whitespace.go
   **Findings:** (a)(c) Character-level offset mapping algorithm — generic offset translation
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `findLineSequence`
   **Position:** internal/tools/fuzzy/indent.go
   **Findings:** (a) Line-by-line pattern matching — generic enough for any sequential line searching
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `matchesAt`
   **Position:** internal/tools/fuzzy/indent.go
   **Findings:** (c) Generic slice matcher — checks if target lines match file lines at position
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `lineOffset`, `lineOffsetEnd`
   **Position:** internal/tools/fuzzy/indent.go
   **Findings:** (c) Byte offset from line indices — generic offset calculation
   **Could replace:** New candidate for `pkg/alg/stringsx`

   **Function:** `normalizeEscapes`
   **Position:** internal/tools/fuzzy/escape.go
   **Findings:** (a) Converts literal escape sequences to actual characters — reusable text processing
   **Could replace:** New candidate for `pkg/alg/stringsx`

### 3. internal/tools (extractable to pkg/alg)

   **Function:** `buildSimpleDiff`
   **Position:** internal/tools/edit_file.go
   **Findings:** (a) Simple unified diff generation — reusable diff formatting utility
   **Could replace:** New candidate for `pkg/alg/diff`

   **Function:** `findUniqueMatch`
   **Position:** internal/tools/edit_file.go
   **Findings:** (a) Match detection with ambiguity handling — useful for any fuzzy matching with uniqueness requirement
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `parseDiffFormat`
   **Position:** internal/tools/apply_patch.go
   **Findings:** (a) Unified diff format parser — reusable diff parsing
   **Could replace:** New candidate for `pkg/alg/diff`

   **Function:** `detectTruncation`
   **Position:** internal/tools/write_file.go
   **Findings:** (a) Detects truncated code via unmatched delimiters — generic truncation detection
   **Could replace:** New candidate for `pkg/alg/stringsx`

   **Function:** `ConvertHTML`
   **Position:** internal/tools/html_convert.go
   **Findings:** (a) Generic HTML-to-markdown converter — widely applicable
   **Could replace:** New candidate for `pkg/alg/stringsx` or separate `pkg/htmlconv`

   **Function:** `buildCommand`, `isShellCmd`
   **Position:** internal/tools/shell_command.go
   **Findings:** (a) Shell command parsing and detection — reusable for any tool that needs command parsing
   **Could replace:** New candidate for `pkg/alg/execx`

   **Function:** `clampViewport`
   **Position:** internal/tools/web_screenshot.go
   **Findings:** (a)(c) Clamps value to range — generic min/max clamping
   **Could replace:** New candidate for `pkg/alg/collections`

### 4. internal/ace

   **Function:** `Clone`
   **Position:** internal/ace/bullet/bullet.go:88
   **Findings:** (a) Deep copy with embeddings and tags — used by curator, refine, playbook
   **Could replace:** Could use a generic `Clone[T]` utility if one existed

   **Function:** `FindMergeCandidates`
   **Position:** internal/ace/refine/merge.go:49
   **Findings:** (a)(b) O(n²) similarity-based pairing — generic deduplication pattern for any embeddable items
   **Could replace:** New candidate for `pkg/alg/similarity`

   **Function:** `calculateSimilarity`
   **Position:** internal/ace/refine/merge.go:118
   **Findings:** (a) Cosine + Jaccard fallback — generic multi-strategy similarity
   **Could replace:** New candidate for `pkg/alg/similarity`

   **Function:** `GetByBullet`, `GetRecent`, `GetSince`
   **Position:** internal/ace/delta/history.go:48-74
   **Findings:** (a)(b) Generic history lookup patterns: by entity, tail-N, time-based filtering
   **Could replace:** New candidate for `pkg/alg/ds` as generic event log

   **Function:** `ValidateBatch`
   **Position:** internal/ace/reflector/validator.go:32
   **Findings:** (a)(b) Batch validation collecting all errors — generic validation composition
   **Could replace:** New candidate for `pkg/apperr`

   **Function:** `FilterByQuality`
   **Position:** internal/ace/reflector/validator.go:46
   **Findings:** (a)(b) Threshold-based filtering — generic for any scored items
   **Could replace:** New candidate for `pkg/alg/collections`

   **Function:** `ExtractConcepts`
   **Position:** internal/ace/trajectory/analysis.go:99
   **Findings:** (a) Word-frequency-based concept extraction — generic text analysis
   **Could replace:** New candidate for `pkg/alg/similarity` or `pkg/alg/stringsx`

### 5. internal/agent

   **Function:** `CommandCache` type
   **Position:** internal/agent/cache.go:42
   **Findings:** (a)(b) Currently `Result`-specific; could be generic `Cache[K, V]` with TTL/LRU eviction
   **Could replace:**
     - pkg/alg/ds:Cache (if extended with LRU eviction)

   **Function:** `DoomLoopGuard.Check`
   **Position:** internal/agent/harness/doomloop.go:67
   **Findings:** (a)(b) Fingerprint-based consecutive-call detection — could work with any sequence
   **Could replace:** New candidate for `pkg/alg/ds` or `pkg/alg/concurrency`

   **Function:** `GatherEnvironment`
   **Position:** internal/agent/environment.go:113
   **Findings:** (a)(b) Comprehensive environment introspection (OS, Git, files, languages)
   **Could replace:** New candidate for `pkg/alg/execx`

   **Function:** `detectProjectType`, `detectLanguages`
   **Position:** internal/agent/environment.go:445,495
   **Findings:** (a)(b) Project/language detection by marker files and extensions
   **Could replace:** New candidate for `pkg/alg/execx`

   **Function:** `ParseToolCallsFromXML`
   **Position:** internal/agent/tool/xml.go:32
   **Findings:** (a)(b) Forward-scanning XML parser — reusable for any XML tool-call format
   **Could replace:** New candidate for `pkg/llmutil`

   **Function:** `filterEnvironment`
   **Position:** internal/agent/environment.go:528
   **Findings:** (d) Delegates to `execx.FilterEnvironment()` — thin wrapper
   **Could replace:**
     - pkg/alg/execx:FilterEnvironment

### 6. internal/contexteng

   **Function:** `HybridCompressor.selectMessages`
   **Position:** internal/contexteng/compress/hybrid.go:142
   **Findings:** (a)(b) Greedy selection with constraint checking — reusable algorithm
   **Could replace:** New candidate for `pkg/alg/collections` as generic greedy selector

   **Function:** `HybridCompressor.enforceMinRetention`
   **Position:** internal/contexteng/compress/hybrid.go:210
   **Findings:** (a)(b) Minimum retention enforcement — generic for any selection problem
   **Could replace:** New candidate for `pkg/alg/collections`

   **Function:** `Cache.evictLRU`
   **Position:** internal/contexteng/summarizer/cache.go:119
   **Findings:** (a)(b) LRU eviction — domain-agnostic, needed for many caches
   **Could replace:**
     - pkg/alg/ds:Cache (if extended with LRU)

   **Function:** `CachingSummarizer` type
   **Position:** internal/contexteng/summarizer/caching.go:9
   **Findings:** (a)(b) Cache decorator pattern — reusable for any `Summarize(ctx, input) (*Result, error)` interface
   **Could replace:** New candidate for `pkg/alg/ds` as generic `Caching[I, O]` decorator

   **Function:** `Classifier.isErrorMessage`, `hasCodeBlock`
   **Position:** internal/contexteng/compress/classifier.go:128,162
   **Findings:** (a)(c) Error detection and code block detection — reusable content analysis
   **Could replace:** New candidate for `pkg/alg/stringsx`

   **Function:** `SummarizeError`
   **Position:** internal/contexteng/observation/summarizer.go:137
   **Findings:** (a) Error output truncation — reusable across packages
   **Could replace:** New candidate for `pkg/alg/stringsx` or `pkg/apperr`

### 7. internal/cycle

   **Function:** `Detector.checkRepeatedTool`, `checkSameError`
   **Position:** internal/cycle/detector.go:201,299
   **Findings:** (c) Both follow identical pattern: `count >= limit && allSame(recent)` — could be generic `detectRepeatPattern[T]`
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `Detector.checkOscillation`
   **Position:** internal/cycle/detector.go:265
   **Findings:** (b) A→B→A→B pattern matching — could work on any slice of comparable values
   **Could replace:** New candidate for `pkg/alg/search`

   **Function:** `PatternDetector.detectOscillatingTools`
   **Position:** internal/cycle/patterns.go:282
   **Findings:** (c) A→B→A→B pattern matching duplicated from detector — extract as `detectAlternatingPattern[T comparable]`
   **Could replace:**
     - internal/cycle/detector.go:checkOscillation (both could use shared generic)

   **Function:** `Detector.getRecentSnapshots`
   **Position:** internal/cycle/detector.go:149
   **Findings:** (c) Rolling window slicing appears 3 times — extract as `sliceRecent[T](items []T, limit int) []T`
   **Could replace:** New candidate for `pkg/alg/collections`

### 8. internal/conversation

   **Function:** `Manager.GetOrCreate`
   **Position:** internal/conversation/manager.go:82
   **Findings:** (a)(b) Double-checked locking for lazy init — widely reusable concurrency pattern
   **Could replace:**
     - pkg/alg/ds/syncmap:Map.GetOrCreate (similar pattern already exists)

### 9. internal/llm

   **Function:** `Router` + fallback chains
   **Position:** internal/llm/router.go
   **Findings:** (a)(b) Generic routing pattern — could use type parameters for different role types
   **Could replace:** New candidate for `pkg/alg/ds`

### 10. internal/lsp

   **Function:** `Cache` with two-level caching
   **Position:** internal/lsp/cache.go
   **Findings:** (a)(b) Generic key-value + content hash strategy — reusable for other cacheable resources
   **Could replace:**
     - pkg/alg/ds:Cache (if extended with content-hash strategy)

   **Function:** `serverForLanguage` double-check locking
   **Position:** internal/lsp/manager.go:64-85
   **Findings:** (a)(b) Lazy init with health checking — generic resource pool pattern
   **Could replace:**
     - pkg/alg/ds/syncmap:Map.GetOrCreate

### 11. internal/safety

   **Function:** `MemoryPolicyStore` with janitor
   **Position:** internal/safety/policy.go:113-150
   **Findings:** (a)(b) In-memory store with TTL eviction — reusable time-based expiration
   **Could replace:**
     - pkg/alg/ds:Cache (if extended with janitor/TTL)

   **Function:** `NewPolicyKey` normalization
   **Position:** internal/safety/policy.go:61-85
   **Findings:** (a) Command string normalization for consistent matching — reusable for command-based keys
   **Could replace:** New candidate for `pkg/alg/execx`

### 12. internal/storage

   **Function:** `FileStore[T]`
   **Position:** internal/storage/store.go
   **Findings:** (a)(b) Already generic JSON file store using Go 1.18 generics — model of reusability
   **Could replace:** Already in good shape; candidate for promotion to `pkg/`

   **Function:** `AtomicWriteFile`
   **Position:** internal/storage/atomic.go
   **Findings:** (a)(b) Staging file + rename strategy — fundamental utility
   **Could replace:** Already in good shape; candidate for promotion to `pkg/`

### 13. internal/session

   **Function:** `Index` with lazy rebuild
   **Position:** internal/session/index.go:56-100
   **Findings:** (a)(b) Auto-rebuild from metadata when corrupted — generic resilient index pattern
   **Could replace:** New candidate for `pkg/alg/ds`

   **Function:** `Transcript.Append` with file lock
   **Position:** internal/session/transcript.go:40-59
   **Findings:** (a)(c) JSONL append under exclusive file lock — decomposable
   **Could replace:**
     - pkg/alg/ds:JSONLWriter (similar pattern already exists)

### 14. internal/memory

   **Function:** `Scratchpad` LRU eviction
   **Position:** internal/memory/scratchpad.go:67-124
   **Findings:** (a)(c) Eviction strategy decomposable into `EvictionPolicy` interface
   **Could replace:**
     - pkg/alg/ds:Cache (if extended with LRU policy)

### 15. internal/filesearch

   **Function:** `Matcher` alias
   **Position:** internal/filesearch/matcher.go
   **Findings:** (d) Thin wrapper; could be removed if callers use `pathx.Matcher` directly
   **Could replace:**
     - pkg/alg/pathx:Matcher

### 16. internal/patchapply

   **Function:** `Matcher` with fuzzy matching
   **Position:** internal/patchapply/matcher.go
   **Findings:** (a)(b) Multi-strategy matching (exact → fuzzy → header hints) — reusable for any context-based matching
   **Could replace:** New candidate for `pkg/alg/search`

### 17. internal/ui/blocks

   **Function:** `MetadataCodec` pattern (ParseExecuteMeta, SetExecuteMeta, etc.)
   **Position:** internal/ui/blocks/metadata.go:262-458
   **Findings:** (b) Repetitive JSON marshal/unmarshal with validation — unify as generic `MetadataCodec[T Validator]`
   **Could replace:** New candidate for `pkg/alg/ds`

   **Function:** `midEllipsize`
   **Position:** internal/ui/blocks/renderer.go:504
   **Findings:** (a)(b) Generic string utility for middle-ellipsis truncation
   **Could replace:** New candidate for `pkg/alg/stringsx`

   **Function:** `extractGraphemes`, `calculateTotalWidth`
   **Position:** internal/ui/blocks/renderer.go:520-587
   **Findings:** (a)(b) Grapheme/width utilities using `uniseg` — reusable across UI components
   **Could replace:** New candidate for `pkg/ui/textwidth`

   **Function:** `Timeline.matchesFilter` chain
   **Position:** internal/ui/blocks/timeline.go:500-594
   **Findings:** (a)(b)(c) Predicate-based filtering — could extract as `FilterChain[T any]`
   **Could replace:** New candidate for `pkg/alg/collections`

   **Function:** `ParamsFormatterRegistry`
   **Position:** internal/ui/blocks/tool_params.go:40
   **Findings:** (a)(b) Strategy registry pattern — could extract as generic `Registry[K, V Strategy]`
   **Could replace:** New candidate for `pkg/alg/ds`

### 18. internal/ui/output

   **Function:** `Printer.printChunksWithCoalescing`
   **Position:** internal/ui/output/printer.go:79-133
   **Findings:** (a)(b)(c) Chunk coalescing with timer-based flushing — generic pattern
   **Could replace:** New candidate for `pkg/alg/concurrency`

### 19. internal/ui/overlay

   **Function:** `Palette` fuzzy search + selection
   **Position:** internal/ui/overlay/palette.go:53-100
   **Findings:** (a)(b)(c) Fuzzy search + selection state machine — extract as `FuzzySelector[T any]`
   **Could replace:** New candidate for `pkg/ui`

### 20. internal/ui/status

   **Function:** `Formatter.FormatAdaptive`
   **Position:** internal/ui/status/formatter.go:32-93
   **Findings:** (a)(b)(c) Width-based adaptive rendering — extract as `AdaptiveFormatter[T Model]`
   **Could replace:** New candidate for `pkg/ui`

### 21. internal/state

   **Function:** State machine with transitions
   **Position:** internal/state/state.go
   **Findings:** (a)(b) Validated state transitions — reusable FSM pattern for any state machine with terminal states
   **Could replace:** New candidate for `pkg/alg/ds`

### 22. examples/

   **Function:** `startSignalHandler` (duplicated in tui-blocks, tui-demo)
   **Position:** examples/tui-blocks/main.go:46-54 and examples/tui-demo/main.go
   **Findings:** (a)(b)(d) Signal handling is duplicated across examples — extract as shared utility
   **Could replace:** New candidate for `pkg/alg/concurrency`
