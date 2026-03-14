# Reusability & Dedup Specification

Organized clusters of reusable patterns extracted from LIST.md (78 findings across all Go files).

---

## Cluster 1: Generic Concurrent Map / Registry

**Problem:** At least 8 independent implementations of `map[K]V` + `sync.RWMutex` + Register/Get/List/Close.

**Instances:**
| # | Location | Type stored |
|---|----------|-------------|
| 28 | `conversation.Manager` | `*Conversation` |
| 37 | `tools.Registry` | `Tool` |
| 39 | `mcp.DefaultRegistryManager` | `Registry` |
| 47 | `ace/playbook.Playbook` | `*bullet.Bullet` |
| 52 | `ace/refine.Archive` | `*ArchivedBullet` |
| 54 | `ace/adapter.sessions` | `*Session` |
| 7  | `commands` registry | `Command` |
| 65 | `protocol/acp.fileContentTracker` | `string` (x3 maps) |

**Proposed extraction:**
```go
// pkg: internal/syncmap (or internal/container)
type Map[K comparable, V any] struct { ... }
func (m *Map[K,V]) Set(key K, val V)
func (m *Map[K,V]) Get(key K) (V, bool)
func (m *Map[K,V]) Delete(key K)
func (m *Map[K,V]) Range(fn func(K, V) bool)
func (m *Map[K,V]) Len() int
func (m *Map[K,V]) Close(closeFn func(V)) // lifecycle
```

**Impact:** Eliminates ~8 mutex+map boilerplate sites, standardizes locking strategy.

---

## Cluster 2: Atomic File Write (temp + rename)

**Problem:** 4+ independent implementations of the same crash-safe write pattern.

**Instances:**
| # | Location |
|---|----------|
| 1  | `storage.FileStore.Save` |
| 10 | `config.MCPConfigStore.writeConfig` |
| 22 | `memory.PersistentStore` |
| 48 | `ace/playbook.Save` |

**Proposed extraction:**
```go
// Already exists conceptually in storage.FileStore — make the atomic write
// a standalone function:
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error
```

**Impact:** Removes ~4 duplicated temp-file+rename blocks.

---

## Cluster 3: Duplicate Sentinel Errors

**Problem:** Linter forces unique variable names, leading to `ErrFoo`, `ErrFoo2`, `ErrFoo3` with identical messages.

**Instances:**
| # | Location | Count |
|---|----------|-------|
| 1  | `storage` — `ErrKeyCannotBeEmpty` | 4 |
| 16 | `patchapply` — `ErrInvalidPath` | 6 |
| 27 | `conversation` — `ErrConversationNotFound` | 4 |
| 31 | `llm/factory` — `ErrAuthenticationRequiredFor` | 2 |
| 36 | `tools` — `ErrParameterNotFound` | 5 |
| 46 | `ace/playbook` — `ErrBulletCannotBeNil` | 2 |
| 49 | `ace/curator` — `ErrDeltaApplierNotInitialized` | 5 |
| 53 | `ace/adapter` — `ErrSessionNotFound` | 3 |
| 72 | `cmd/spin/config` — `ErrValidationFailed`, `ErrNoConfigFile` | 2+2 |

**Fix:** Use a single sentinel per message. If multiple call-sites need the same error, reference the same variable. If distinct context is needed, wrap with `fmt.Errorf("context: %w", ErrFoo)`.

**Impact:** Removes ~35 unnecessary error variables.

---

## Cluster 4: Cosine Similarity (Quadruplicate)

**Problem:** 4 identical `cosineSimilarity(a, b []float32) float64` implementations.

**Instances:**
| # | Location |
|---|----------|
| 51 | `ace/playbook/search.go` |
| 51 | `ace/curator/deduplicator.go` |
| 51 | `ace/retrieval/hnsw_retriever.go` |
| 51 | `ace/refine/merge.go` |

Plus a custom `sqrt` (Newton's method) in HNSW retriever instead of `math.Sqrt` (#58).

**Proposed extraction:**
```go
// pkg: internal/mathutil (or internal/ace/vecmath)
func CosineSimilarity(a, b []float32) float64
func DotProduct(a, b []float32) float64
func Magnitude(v []float32) float64
```

**Impact:** Removes 3 duplicate functions + 1 unnecessary custom sqrt.

---

## Cluster 5: TTL Cache

**Problem:** 2 independent TTL-based in-memory caches with different backing stores.

**Instances:**
| # | Location | Backing |
|---|----------|---------|
| 19 | `security.MemoryPolicyStore` | `map` + `sync.RWMutex` + background janitor |
| 43 | `agent.CommandCache` | `sync.Map` + size cap |

**Proposed extraction:**
```go
// pkg: internal/cache
type TTLCache[K comparable, V any] struct { ... }
func New[K comparable, V any](ttl time.Duration, opts ...Option) *TTLCache[K,V]
// Options: WithMaxSize(n), WithJanitorInterval(d)
```

**Impact:** Unifies 2 cache implementations with consistent eviction semantics.

---

## Cluster 6: ErrToolNotFound (Triplicate)

**Problem:** Same sentinel `errors.New("tool not found")` in 3 packages.

**Instances:**
| # | Location |
|---|----------|
| 40 | `tools.ErrToolNotFound` |
| 40 | `mcp.ErrToolNotFound` |
| 42 | `agent.ErrToolNotFound` |

**Fix:** Keep `tools.ErrToolNotFound` as canonical. Others import it or wrap it.

**Impact:** Single source of truth for tool-not-found semantics.

---

## Cluster 7: Worker Pool (Fan-out/Fan-in)

**Problem:** 2 identical worker pool implementations.

**Instances:**
| # | Location |
|---|----------|
| 57 | `ace/delta/batch.go` — `ApplyBatch` |
| 57 | `ace/curator/parallel.go` — `curateBatchParallel` |

Both use: jobs channel → N workers (WaitGroup) → results channel → collect.

**Proposed extraction:**
```go
// pkg: internal/workerpool
func Run[In, Out any](ctx context.Context, workers int, inputs []In, fn func(In) Out) []Out
```

**Impact:** Removes 1 duplicate fan-out/fan-in implementation.

---

## Cluster 8: Home Directory Expansion (`~`)

**Problem:** 2+ implementations of `~` → home dir resolution.

**Instances:**
| # | Location |
|---|----------|
| 30 | `conversation/events.go:resolveSessionDir` |
| 22 | `memory/persistent.go` — inline expansion |

**Proposed extraction:**
```go
func ExpandHome(path string) (string, error)
```

**Impact:** Small but eliminates a recurring pattern.

---

## Cluster 9: Generic Typed Accessors

**Problem:** N accessor methods following identical type-assertion or json.Unmarshal pattern.

**Instances:**
| # | Location | Count |
|---|----------|-------|
| 5  | `events.Event` — 12 `XxxData()` methods | 12 |
| 38 | `tools.ToolParameters` — GetString/Int/Bool/Float64/Object | 5 |
| 66 | `cmd/spin/root.go` — flagModel/Provider/WorkDir/ConfigFile/AgentsMD | 5 |

**Proposed extraction:**
```go
// Event data:
func GetEventData[T any](e Event) (T, bool) { d, ok := e.Data.(T); return d, ok }

// Tool params:
func GetParam[T any](p ToolParameters, key string) (T, error)

// Cobra flags:
func flagString(cmd *cobra.Command, name string) string
```

**Impact:** Removes ~22 boilerplate accessor methods.

---

## Cluster 10: Duplicate Detection / Events Abstraction

**Problem:** `internal/detection` redefines interfaces from `internal/events` and `internal/message`.

**Instances:**
| # | Findings |
|---|----------|
| 4  | `detection` defines own `Message`, `Event`, `EventEmitter` interfaces |
| 45 | `agent/loop.go` has `eventEmitterAdapter` bridging events→detection |
| 25 | `cycle` and `detection` define overlapping domain types (Snapshot, Result, CycleType) |

**Fix:** Have `detection` use `events.EventEmitter` and `message.Message` directly. Merge `cycle` types into `detection` or vice versa. Eliminates the adapter (#45).

**Impact:** Removes 1 adapter, 3 duplicate interfaces, and overlapping type hierarchies.

---

## Cluster 11: Duplicate cmd/spin Wiring

**Problem:** TUI and exec modes duplicate conversation setup, event loops, signal handling, and provider creation.

**Instances:**
| # | Description |
|---|-------------|
| 67 | `processEvent` vs `processExecEvent` — duplicate event processors |
| 68 | `startEventLoop` vs `startExecEventLoop` — duplicate event loops |
| 69 | `createConversationForTUI` inlines `buildConversation` logic |
| 70 | Inline session resolution vs `resolveSessionID` |
| 71 | `setupACPServerSignalHandling` vs `setupSignalHandling` |
| 73 | `printJSON` vs `outputJSON` — duplicate JSON encoders |
| 74 | Inline filter vs `filterServersByRegistry` |
| 75 | 8x repeated "load config + MCP store" boilerplate |
| 77 | `buildProviderForACP` vs `buildProvider` — duplicate provider factories |

**Fix:** Extract shared helpers and have TUI/exec/ACP call them:
- Unified `processEvent` + `startEventLoop`
- TUI calls `buildConversation` + `resolveSessionID`
- Single `setupSignalHandling(cancel, logMsg)` with optional message
- Single `printJSON(out, data)` used by both config and mcp commands
- `loadMCPConfigStore(cmd)` helper for the 8 repeated sites
- Single `buildProvider` factory with extra-provider hook

**Impact:** Significant reduction in cmd/spin code duplication.

---

## Cluster 12: State / Lifecycle Enums

**Problem:** Multiple packages define overlapping lifecycle state enums.

**Instances:**
| # | Location | States |
|---|----------|--------|
| 3  | `state.State` | Created→Running→Completed/Failed/Cancelled |
| 12 | `session.validateStateTransition` | Custom transition rules |
| 18 | `planning.PlanStatus/StepStatus` | Pending→InProgress→Completed/Failed/Cancelled |

**Fix:** Reuse `state.State` FSM or define a shared lifecycle enum. Planning could use `state.State` transitions.

**Impact:** Consolidates 3 lifecycle definitions.

---

## Cluster 13: Structured Error Types

**Problem:** Two structured error types with overlapping shapes.

**Instances:**
| # | Location | Fields |
|---|----------|--------|
| 17 | `patchapply.Error` | Op, Path, Err, Context |
| — | `apperr.Error` | Code, Op, Err, Message |

**Fix:** Consider unifying into a single structured error or having `patchapply.Error` embed `apperr.Error`.

**Impact:** Minor — reduces conceptual duplication.

---

## Cluster 14: Task Validate() Quadruplicate

**Problem:** 4 task types with identical `Validate()` method.

**Instances:**
| # | Location |
|---|----------|
| 61 | `task/compact.go`, `planning.go`, `regular.go`, `review.go` |
| 62 | Duplicate `MaxAllowedTokens` constant (public + private) |

**Fix:** Extract `validateMaxTokens(maxTokens int) error` as shared function or embed in base struct. Remove duplicate constant.

**Impact:** Removes 3 duplicate methods + 1 duplicate constant.

---

## Cluster 15: Text Similarity Utilities

**Problem:** Multiple text similarity/comparison functions across packages.

**Instances:**
| # | Location | Algorithm |
|---|----------|-----------|
| 24 | `cycle/similarity.go` | Jaccard coefficient (word sets) |
| 55 | `ace/generator/generator.go:checkSuccess` | Multi-strategy (exact, normalized, word overlap) |

**Fix:** Extract shared text similarity utilities:
```go
func JaccardSimilarity(a, b string) float64
func WordOverlap(a, b string) float64
```

**Impact:** Enables reuse of text comparison across cycle detection and ACE.

---

## Cluster 16: LLM Response Cleaning

**Problem:** Identical `cleanJSONResponse` in 2 packages.

**Instances:**
| # | Location |
|---|----------|
| 50 | `ace/curator/curator.go` |
| 50 | `ace/reflector/reflector.go` |

**Fix:** Extract to shared utility:
```go
// pkg: internal/llmutil
func CleanJSONResponse(raw string) string  // strips ```json markers
```

**Impact:** Removes 1 duplicate function.

---

## Cluster 17: Service Adapter / Wrapper Pattern

**Problem:** Repeated pattern of wrapping a lower-level type with a Service struct.

**Instances:**
| # | Location |
|---|----------|
| 15 | `git.Service` wraps `git.Integration` |
| 15 | `shell.Service` wraps `shell.Context` |
| 44 | `agent.ToolExecutorAdapter` — adapts Executor→CommandExecutor |

**Fix:** Consider a generic adapter or code generation for the wrapper pattern. Or simply document the pattern as intentional.

**Impact:** Low — primarily a documentation/convention concern.

---

## Cluster 18: MCP Server Config Duplication

**Problem:** Two nearly identical MCP server config structs with different serialization tags.

**Instances:**
| # | Location |
|---|----------|
| 11 | `config.MCPServer` (json/toml/yaml) vs `config.MCPServerConfigV2` (mapstructure/yaml) |
| 9  | Duplicate validation methods between MCPConfigStore and MCPServerConfigV2 |

**Fix:** Unify into one struct with all necessary tags. Consolidate validation into one set of methods.

**Impact:** Removes ~150 lines of duplicate validation + 1 duplicate struct.

---

## Cluster 19: Miscellaneous Single-site Opportunities

| # | Finding | Action |
|---|---------|--------|
| 8  | `config.ValidationErrors` — generic multi-error | Consider `errors.Join()` from stdlib |
| 20 | `filesearch.Matcher` — generic fuzzy matching | Already standalone, document as reusable |
| 29 | `conversation.boolString` | Replace with `strconv.FormatBool()` |
| 56 | `ace/delta.History` — append-only indexed log | Document as reusable pattern |
| 60 | `agentsmd.readWithLimit` — size-bounded file read | Document as reusable utility |
| 63 | `tui.atomicCounter` — custom atomic counter | Replace with `sync/atomic.Int64` |
| 76 | `approval_handlers` — auto-approve/deny factories | Move to `security` package |
| 78 | `tui.configureMaxTokens` — dead iteration loop | Fix or remove dead code |

---

## Priority Ranking

| Priority | Cluster | Effort | Impact |
|----------|---------|--------|--------|
| P0 | 3 — Duplicate Sentinel Errors | Low | High (35 vars) |
| P0 | 6 — ErrToolNotFound Triplicate | Low | Medium |
| P1 | 1 — Generic Concurrent Map | Medium | High (8 sites) |
| P1 | 4 — Cosine Similarity | Low | Medium (4 sites) |
| P1 | 11 — cmd/spin Wiring Duplication | Medium | High |
| P1 | 14 — Task Validate() | Low | Medium |
| P2 | 2 — Atomic File Write | Low | Medium (4 sites) |
| P2 | 5 — TTL Cache | Medium | Medium (2 sites) |
| P2 | 7 — Worker Pool | Medium | Low (2 sites) |
| P2 | 9 — Generic Typed Accessors | Medium | Medium (22 methods) |
| P2 | 10 — Detection/Events Abstraction | Medium | Medium |
| P2 | 16 — cleanJSONResponse | Low | Low |
| P3 | 8 — Home Dir Expansion | Low | Low |
| P3 | 12 — State/Lifecycle Enums | Medium | Low |
| P3 | 13 — Structured Error Types | Medium | Low |
| P3 | 15 — Text Similarity | Low | Low |
| P3 | 17 — Service Adapter Pattern | Low | Low |
| P3 | 18 — MCP Config Duplication | Medium | Medium |
| P3 | 19 — Miscellaneous | Low | Low |
