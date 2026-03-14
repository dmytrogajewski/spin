# Dedup & Reusability Opportunities

## Progress Graph

```
[ internal/apperr [✅] ]
[ internal/appinfo [✅] ]
[ internal/storage [✅] ]
[ internal/state [✅] ]
[ internal/message [✅] ]
[ internal/events [✅] ]
[ internal/dbg [✅] ]
[ internal/detection [✅] ]
[ internal/tokenizer [✅] ]
[ internal/shell [✅] ]
[ internal/auth [✅] ]
[ internal/commands [✅] ]
[ internal/config [✅] ]
[ internal/session [✅] ]
[ internal/filesearch [✅] ]
[ internal/git [✅] ]
[ internal/patchapply [✅] ]
[ internal/planning [✅] ]
[ internal/security [✅] ]
[ internal/memory [✅] ]
[ internal/history [✅] ]
[ internal/context/summarizer [✅] ]
[ internal/conversation [✅] ]
[ internal/cycle [✅] ]
[ internal/llm/* [✅] ]
[ internal/mcp [✅] ]
[ internal/tools [✅] ]
[ internal/agent/* [✅] ]
[ internal/agentsmd [✅] ]
[ internal/ace/* [✅] ]
[ internal/task [✅] ]
[ internal/protocol/acp [✅] ]
[ internal/tui [✅] ]
[ internal/ui/* [✅] ]
[ cmd/spin [✅] ]
```

## Dedup opportunities

1. **internal/storage**

   Function: `FileStore[T]` (generic key-value file store)
   Position: `internal/storage/store.go:52`
   Findings: Already generic with `Store[T]` interface and `FileStore[T]` implementation. Has atomic writes, mutex protection. However, has **4 duplicate sentinel errors** (`ErrKeyCannotBeEmpty`, `ErrKeyCannotBeEmpty2`, `ErrKeyCannotBeEmpty3`, `ErrKeyCannotBeEmpty4`) that should be collapsed into one.
   Could replace:
     - Check `internal/session/file_storage.go` — likely has its own file-based storage that could use `FileStore[T]`
     - Check `internal/history/storage.go` — likely has its own JSON file storage
     - Check `internal/memory/store.go` — might have its own persistence layer
     - Check `internal/ace/playbook/storage.go` — might have its own file store

2. **internal/events**

   Function: `EventEmitter` (pub/sub with backpressure)
   Position: `internal/events/event.go:395`
   Findings: Generic pub/sub event emitter with configurable backpressure modes (Drop, Block, Buffer). Thread-safe. This is a **fully reusable generic pub/sub component**. However, the `Event` struct uses `Data any` with ~12 type assertion accessor methods (ToolCallStartData, ToolCallCompleteData, etc.) — these follow an identical pattern and could be replaced by a generic typed event or a single generic accessor function.
   Could replace:
     - `internal/detection/detection.go:82-92` — defines its own `EventEmitter` and `Event` interfaces that duplicate the events package

3. **internal/state**

   Function: `State` FSM (finite state machine with transitions)
   Position: `internal/state/state.go:1`
   Findings: Implements a state machine with `CanTransitionTo()`, `allowedTransitions` map, `IsTerminal()`, `IsActive()`. The FSM pattern (states + transition map + validation) is **generic and reusable** — could be parameterized with generics as `FSM[S comparable]` with configurable transitions.

4. **internal/detection**

   Function: `Message`, `Event`, `EventEmitter` interfaces
   Position: `internal/detection/detection.go:76-92`
   Findings: Defines its own `Message`, `Event`, `EventEmitter` interfaces that overlap with `internal/events` and `internal/message`. Also has its own private `message` and `event` struct implementations. These are **duplicate abstractions** that could use the existing packages.
   Could replace:
     - Should use `internal/events.EventEmitter` instead of its own `EventEmitter` interface
     - Should use `internal/message.Message` or a shared interface instead of its own

5. **internal/events**

   Function: Event type assertion accessors (12 methods)
   Position: `internal/events/event.go:44-137`
   Findings: 12 methods like `ToolCallStartData()`, `ContentDeltaData()`, etc. all follow identical pattern: `data, ok := e.Data.(ConcreteType); return data, ok`. Could be replaced with a single generic function: `func GetEventData[T any](e Event) (T, bool)`.

6. **internal/dbg**

   Function: `initServices`, `createGitService`, `createShellService`, `createMCPService`
   Position: `internal/dbg/events.go:67-155`
   Findings: Service creation/wiring code that likely duplicates `cmd/spin/services.go`. Both create git, shell, MCP services from config. This is a **service factory pattern** that should be consolidated.
   Could replace:
     - Check `cmd/spin/services.go` — likely has very similar service wiring code

7. **internal/commands**

   Function: `Command` registry (global command registration + dispatch)
   Position: `internal/commands/commands.go:47-83`
   Findings: Generic command registry pattern using global `map[string]Command` with `RegisterCommand`, `GetCommand`, `ListCommands`, `ParseCommand`. This is a **generic registry pattern** that could be parameterized. The registry stores `Command` interface implementations with Name/Description/Execute. Similar pattern may appear in tools registry.
   Could replace:
     - Check `internal/tools/registry.go` — likely has similar tool registration pattern

8. **internal/config**

   Function: `ValidationErrors` (multi-error collector)
   Position: `internal/config/config_v2.go:100-143`
   Findings: Generic multi-error collector with `Add(err)`, `HasErrors()`, `ToError()`. This pattern is **generic and reusable** across any validation scenario. Could be extracted as a shared utility or replaced by `errors.Join()` from stdlib.

9. **internal/config**

   Function: `MCPConfigStore` validation methods (duplicated validation logic)
   Position: `internal/config/mcp_manager.go:157-233`
   Findings: `MCPConfigStore.validateStdio/validateRemote/validateSmithery` duplicate nearly identical logic from `MCPServerConfigV2.validateStdio/validateRemote/validateSmithery` in config_v2.go. **Same validation written twice** — once for the config struct, once for the config store. Should be consolidated.

10. **internal/config**

    Function: `MCPConfigStore.writeConfig` (atomic file write)
    Position: `internal/config/mcp_manager.go:236-283`
    Findings: Implements atomic write (temp file + rename) — **same pattern** as `storage.FileStore.Save()`. Could reuse storage utilities.
    Could replace:
      - Uses same atomic write pattern as `internal/storage/store.go:104-136`

11. **internal/config**

    Function: `MCPServer` vs `MCPServerConfigV2` (duplicate MCP server types)
    Position: `internal/config/mcp_manager.go:23-47` vs `internal/config/config_v2.go:277-301`
    Findings: Two nearly identical MCP server config structs: `MCPServer` (with json/toml/yaml tags) and `MCPServerConfigV2` (with mapstructure/yaml tags). **Same data, different serialization tags**. Should be unified into one type.

12. **internal/session**

    Function: `session.Session.validateStateTransition` (duplicate state transition logic)
    Position: `internal/session/session.go:141-153`
    Findings: Implements its own state transition validation that **partially duplicates** `state.State.CanTransitionTo()`. The session package aliases `state.State` but then implements its own simpler transition rules instead of using the state package's transition map.
    Could replace:
      - Should use `internal/state.State.CanTransitionTo()` instead of custom logic

13. **internal/session**

    Function: `session.NewFileStorage` (FileStore instantiation)
    Position: `internal/session/file_storage.go:1-18`
    Findings: **Already using `storage.FileStore[Session]`** — good dedup. Confirms storage.FileStore is reusable. This is a **positive example** of reuse.

14. **internal/auth**

    Function: `Keystore` interface (key-value string store)
    Position: `internal/auth/keystore.go:13-33`
    Findings: Simple key-value store interface (`Get`, `Set`, `Delete`, `List`) for strings. This is essentially a **non-generic version** of `storage.Store[T]` specialized for `string` values. Could potentially be replaced by `storage.Store[string]` or share a common base interface.
    Could replace:
      - Overlaps conceptually with `internal/storage.Store[T]` (string specialization)

15. **internal/git + internal/shell**

    Function: Service wrapper pattern (Service wraps lower-level struct)
    Position: `internal/git/service.go:1-50` and `internal/shell/service.go:1-51`
    Findings: Both `git.Service` and `shell.Service` follow identical patterns: wrap a lower-level type (Integration/Context), provide `NewService(ctx, enabled, workDir, logger)`, forward methods like `GetContextInfo()`, `IsEnabled()`/`IsRepository()`, `Close()`. This is a **repeated service adapter pattern** that could be formalized or generated.

16. **internal/patchapply**

    Function: Duplicate sentinel errors
    Position: `internal/patchapply/parser.go:10-37`
    Findings: Has **6 duplicate `ErrInvalidPath`** errors (`ErrInvalidPath` through `ErrInvalidPath6`) and **2 duplicate `ErrInvalidNewPath`** errors. Same issue as `storage.ErrKeyCannotBeEmpty` duplicates. Should be collapsed.

17. **internal/patchapply**

    Function: `patchapply.Error` struct (structured error with Op/Path/Err)
    Position: `internal/patchapply/applier.go:92-121`
    Findings: Structured error type with `Op`, `Path`, `Err`, `Context`, `Unwrap()` — very similar to `apperr.Error` which has `Code`, `Op`, `Err`, `Message`, `Unwrap()`. **Two structured error types** with overlapping patterns. Could be unified into one shared structured error.
    Could replace:
      - Overlaps with `internal/apperr/errors.go:52-86` — both are structured errors with Op+Err+context

18. **internal/planning**

    Function: `PlanStatus` + `StepStatus` enums (status state machines)
    Position: `internal/planning/service.go:177-220`
    Findings: `PlanStatus` (Pending/InProgress/Completed/Failed/Cancelled) and `StepStatus` (Pending/Ready/Running/Completed/Failed/Skipped) are **very similar to `state.State`**. Same concepts (pending, running, completed, failed, cancelled) defined in three different places.
    Could replace:
      - Overlaps with `internal/state/state.go` — same lifecycle states

19. **internal/security**

    Function: `MemoryPolicyStore` (in-memory key-value store with TTL)
    Position: `internal/security/policy.go:119-312`
    Findings: In-memory key-value store with TTL eviction (background janitor goroutine). Thread-safe with `sync.RWMutex`. This is a **generic TTL cache pattern** that could be extracted as a reusable `TTLCache[K, V]` component. The eviction janitor pattern is independently useful.
    Could replace:
      - Could serve as basis for any in-memory cache needing TTL eviction across the codebase

20. **internal/filesearch**

    Function: `Matcher` (fuzzy string matching with scoring)
    Position: `internal/filesearch/matcher.go:1-301`
    Findings: Generic fuzzy matching algorithm with configurable scoring. Works on string slices. The core algorithm (matchCharacters, consecutive bonus, separator bonus) is **reusable for any fuzzy search** scenario (not just file paths). Could be extracted as a generic fuzzy matcher.

21. **internal/patchapply**

    Function: Path validation (`validatePath`, `resolvePath`)
    Position: `internal/patchapply/parser.go:250-260` and `internal/patchapply/applier.go:344-359`
    Findings: Path validation (no absolute paths, no `..` traversal, workspace confinement) appears in **both parser and applier**. The parser validates during parsing, the applier validates during application. Could be unified into one shared path validator.

22. **internal/memory**

    Function: `PersistentStore` (file-based key-value store with atomic writes)
    Position: `internal/memory/persistent.go:37-421`
    Findings: Implements its own file-based JSON store with **atomic write (temp + rename)**, home directory expansion (`~/`), directory creation, and JSON serialization. This is essentially a **reimplementation of `storage.FileStore[T]`** with a different interface (memory.Store vs storage.Store). The core persistence logic is identical.
    Could replace:
      - `internal/storage/store.go:104-136` — identical atomic write pattern
      - `internal/config/mcp_manager.go:236-283` — identical atomic write pattern

23. **internal/memory**

    Function: `containsIgnoreCase`, `findIgnoreCase`, `toLower`, `matchPattern` (string utilities)
    Position: `internal/memory/scratchpad.go:320-381`
    Findings: Generic case-insensitive string search and glob matching utilities. `containsIgnoreCase` does byte-level case-insensitive search with custom `toLower`. These are **generic string utilities** that could be shared.

24. **internal/cycle**

    Function: `calculateSimilarity` (Jaccard similarity), `extractWords`, `createWordSet`
    Position: `internal/cycle/similarity.go:1-103`
    Findings: Generic text similarity computation using Jaccard coefficient. `extractWords` normalizes text into word sets. These are **reusable NLP/text analysis utilities** not specific to cycle detection.

25. **internal/cycle** vs **internal/detection**

    Function: Duplicate type hierarchies (Snapshot, Result, CycleType)
    Position: `internal/cycle/` vs `internal/detection/detection.go`
    Findings: `cycle.Snapshot` (Turn, Response, ToolCalls, Error, Timestamp) overlaps with `detection.Snapshot` (same fields). `cycle.Result` (Type, Confidence, Details, Timestamp) overlaps with `detection.CycleResult` (same fields). `cycle.CycleType` overlaps with `detection.CycleType`. **Two packages define essentially the same domain types**.
    Could replace:
      - Should unify `cycle` and `detection` packages — they model the same domain

26. **internal/history**

    Function: `history.NewFileStorage` (FileStore instantiation)
    Position: `internal/history/storage.go:31-36`
    Findings: **Already using `storage.FileStore[Data]`** — same good pattern as `session.NewFileStorage`. Confirms `storage.FileStore` is widely reusable.

27. **internal/conversation**

    Function: Duplicate sentinel errors in `Manager`
    Position: `internal/conversation/manager.go:14-35`
    Findings: Has **4 duplicate `ErrConversationNotFound`** errors (`ErrConversationNotFound` through `ErrConversationNotFound4`) and **2 duplicate `ErrHistoryStorageNotConfigured`** errors. Same pattern as storage and patchapply duplicate errors.

28. **internal/conversation**

    Function: `Manager` (concurrent map with GetOrCreate/Remove/List/Close)
    Position: `internal/conversation/manager.go:45-282`
    Findings: `Manager` is a **generic concurrent map** pattern: `map[string]*Conversation` with `sync.RWMutex`, double-checked locking in `GetOrCreate`, `Remove`, `List`, `Count`, `Close`. The same pattern appears in `mcp.DefaultRegistryManager` (map[string]Registry with same methods). Could be extracted as a generic `ConcurrentMap[K, V]` or `Registry[T]`.
    Could replace:
      - Overlaps structurally with `mcp.DefaultRegistryManager` (map + mutex + Register/Get/All/Close)

29. **internal/conversation**

    Function: `boolString` utility
    Position: `internal/conversation/builder.go:331-337`
    Findings: Simple `bool → "true"/"false"` string converter. Generic utility that may be duplicated elsewhere. Could use `strconv.FormatBool()` from stdlib instead.

30. **internal/conversation**

    Function: `resolveSessionDir` (home directory expansion with `~`)
    Position: `internal/conversation/events.go:70-89`
    Findings: Resolves `~` prefix to home directory, same pattern as `memory.PersistentStore` home directory expansion. **Duplicated `~` expansion logic**.
    Could replace:
      - Same pattern as `internal/memory/persistent.go` home dir expansion

31. **internal/llm/factory**

    Function: Duplicate sentinel errors
    Position: `internal/llm/factory/factory.go:19-35`
    Findings: Has **2 duplicate `ErrAuthenticationRequiredFor`** errors (`ErrAuthenticationRequiredFor` and `ErrAuthenticationRequiredFor2`). Same duplicate sentinel error pattern.

32. **internal/llm/factory**

    Function: Provider creation boilerplate (timeout defaulting, config mapping)
    Position: `internal/llm/factory/factory.go:263-333`
    Findings: `newOpenAIProvider`, `newOllamaProvider`, `newLMStudioProvider` all follow identical pattern: resolve credential → set default timeout → create provider-specific config → call NewProvider. The timeout defaulting (`if timeout <= 0 { timeout = llm.DefaultTimeout }`) is **repeated 3 times**. Could extract a `resolveTimeout` helper.

33. **internal/llm/openai**

    Function: `mapError` (HTTP status → domain error mapping)
    Position: `internal/llm/openai/errors.go:16-55`
    Findings: Maps HTTP status codes to domain errors. This is a **reusable error mapping pattern**. If other providers (Ollama, LMStudio) need to map HTTP errors, this pattern would be duplicated. Currently only used by OpenAI provider since Ollama uses native SDK.

34. **internal/llm/ollama**

    Function: `mapOllamaDoneReasonToOpenAI` vs `mapOllamaDoneReasonToOpenAICompletion`
    Position: `internal/llm/ollama/convert.go:506-525` and `internal/llm/ollama/convert.go:589-611`
    Findings: **Two nearly identical functions** mapping Ollama's `done_reason` to OpenAI's `finish_reason`. One returns `ChatCompletionChoicesFinishReason` (non-streaming), the other returns `ChatCompletionChunkChoicesFinishReason` (streaming). The logic is identical — only the return type differs. Could be unified if OpenAI SDK types had a common base.

35. **internal/llm/mock**

    Function: `sendChunk` and `waitDelay` (context-aware channel send and delay)
    Position: `internal/llm/mock.go:264-281`
    Findings: `sendChunk` (context-aware channel send) and `waitDelay` (context-aware sleep) are **generic concurrency utilities** that could be reused by any streaming component. The pattern `select { case <-ctx.Done(): return false; case ch <- val: return true }` appears repeatedly.

36. **internal/tools**

    Function: Duplicate sentinel errors in `parameters.go`
    Position: `internal/tools/parameters.go:10-21`
    Findings: Has **5 duplicate `ErrParameterNotFound`** errors (`ErrParameterNotFound` through `ErrParameterNotFound5`). Each used by a different accessor (GetString, GetInt, GetBool, GetFloat64, GetObject). Same duplicate error pattern.

37. **internal/tools**

    Function: `Registry` (tool registration + lookup + validation + execution)
    Position: `internal/tools/registry.go:19-288`
    Findings: Tool registry with `map[string]Tool`, `Register`, `RegisterOrReplace`, `Get`, `List`, `Execute`, parameter validation. This is **structurally identical** to `commands.Command` registry (item 7) and `mcp.DefaultRegistryManager` (item 28). All three are `map[string]Interface` with Register/Get/List/Close.
    Could replace:
      - Overlaps with `internal/commands/commands.go` — same generic registry pattern
      - Overlaps with `mcp.DefaultRegistryManager` — same concurrent map pattern

38. **internal/tools**

    Function: `ToolParameters` typed accessors (GetString, GetInt, GetBool, GetFloat64, GetObject)
    Position: `internal/tools/parameters.go:88-173`
    Findings: 5 typed accessor methods all follow identical pattern: check key exists → json.Unmarshal into typed var → return. Could be replaced with a single generic function: `func GetParam[T any](p ToolParameters, key string) (T, error)`.

39. **internal/mcp**

    Function: `DefaultRegistryManager` (concurrent registry map)
    Position: `internal/mcp/registry_manager.go:22-225`
    Findings: Thread-safe map of `Registry` instances with `Register`, `Unregister`, `Get`, `All`, `Close`, `Count`. **Structurally identical** to `conversation.Manager` and `tools.Registry` — all are `map[string]T` with mutex + CRUD.
    Could replace:
      - Overlaps with `conversation.Manager` — same concurrent map with lifecycle management
      - Overlaps with `tools.Registry` — same thread-safe map + Register/Get/List

40. **internal/mcp + internal/tools**

    Function: `ErrToolNotFound` defined in both packages
    Position: `internal/mcp/errors.go:10` and `internal/tools/tool.go:12`
    Findings: Same sentinel error `ErrToolNotFound` defined independently in both `mcp` and `tools` packages. Should be unified — `tools.ErrToolNotFound` should be the single source.

41. **internal/llm**

    Function: `ProviderConfig` interface + `BaseConfig` struct (provider configuration)
    Position: `internal/llm/config.go:9-39`
    Findings: `ProviderConfig` interface (GetBaseURL, GetModel, GetTimeout, GetAPIKey, Validate) and `BaseConfig` struct define provider config. But each provider (openai, ollama, lmstudio) defines its **own Config struct** with identical fields (BaseURL, Model, Timeout). `BaseConfig` is never embedded by any provider. The `ProviderConfig` interface is not implemented by any of the provider Config structs.
    Could replace:
      - openai.Config, ollama.Config, lmstudio.Config could all embed `llm.BaseConfig`

42. **internal/agent**

    Function: `ErrToolNotFound` (third definition)
    Position: `internal/agent/tool_runtime.go:17`
    Findings: Third definition of `ErrToolNotFound` alongside `tools.ErrToolNotFound` and `mcp.ErrToolNotFound`. All three are `errors.New("tool not found")`. Should be unified to a single definition in `internal/tools`.
    Could replace:
      - Duplicate of `internal/tools/tool.go:12` and `internal/mcp/errors.go:10`

43. **internal/agent**

    Function: `CommandCache` (TTL + size-based cache with sync.Map)
    Position: `internal/agent/cache.go:1-120`
    Findings: In-memory cache with TTL expiration and maximum size, using `sync.Map`. Contains `IsCacheable` with read-only command detection. Overlaps with `security.MemoryPolicyStore` (finding 19) — both are TTL-based in-memory caches with different backing stores (`sync.Map` vs `map` + `sync.RWMutex`). Could be unified into a generic `TTLCache[K, V]`.
    Could replace:
      - Overlaps with `internal/security/policy.go:119-312` — same TTL cache pattern

44. **internal/agent**

    Function: `ToolExecutorAdapter` (adapts agent.Executor to tools.CommandExecutor)
    Position: `internal/agent/tool_adapter.go:1-50`
    Findings: Adapter bridging `agent.Executor` to `tools.CommandExecutor` interface. **Structurally identical** to `conversation/adapters.go:executorAdapter` which does the same thing. Same adapter proliferation pattern (finding 4 in detection).
    Could replace:
      - Duplicate of `internal/conversation/adapters.go` executorAdapter

45. **internal/agent**

    Function: `eventEmitterAdapter` (adapts events.EventEmitter to detection.EventEmitter)
    Position: `internal/agent/loop.go:656-679`
    Findings: Adapter bridging `events.EventEmitter` to `detection.EventEmitter`. This exists because `detection` defines its own `EventEmitter` interface instead of using `internal/events.EventEmitter` (finding 4). Eliminating the duplicate interface would eliminate this adapter.
    Could replace:
      - Would be unnecessary if `detection` used `events.EventEmitter` directly

46. **internal/ace/playbook**

    Function: Duplicate sentinel errors (`ErrBulletCannotBeNil`, `ErrBulletCannotBeNil2`)
    Position: `internal/ace/playbook/playbook.go:18-25`
    Findings: Two identical `ErrBulletCannotBeNil` errors with the same message. Same duplicate sentinel error anti-pattern seen throughout the codebase.

47. **internal/ace/playbook**

    Function: `Playbook` (concurrent map of bullets with mutex)
    Position: `internal/ace/playbook/playbook.go:31-172`
    Findings: `map[string]*bullet.Bullet` with `sync.RWMutex` and Add/Get/Update/Delete/List methods. **Another instance** of the generic concurrent map pattern (finding 28 conversation.Manager, finding 37 tools.Registry, finding 39 mcp.DefaultRegistryManager). Could be replaced by a generic `ConcurrentMap[K, V]`.

48. **internal/ace/playbook**

    Function: `Save` (atomic file write with temp + rename)
    Position: `internal/ace/playbook/storage.go:22-76`
    Findings: Atomic write using temp file + `os.Rename`. **Same pattern** as `storage.FileStore.Save`, `memory.PersistentStore`, `config.MCPConfigStore.writeConfig` (findings 1, 10, 22).

49. **internal/ace/curator**

    Function: Duplicate sentinel errors (`ErrDeltaApplierNotInitialized` x5)
    Position: `internal/ace/curator/curator.go:31-41`
    Findings: **5 duplicate** `ErrDeltaApplierNotInitialized` errors, each used in a different method. Same anti-pattern.

50. **internal/ace/curator + ace/reflector**

    Function: `cleanJSONResponse` (strip markdown code blocks from JSON)
    Position: `internal/ace/curator/curator.go:513-532` and `internal/ace/reflector/reflector.go:349-369`
    Findings: **Two identical implementations** of `cleanJSONResponse` that strip ```json and ``` markers from LLM responses. Should be a shared utility function.

51. **internal/ace (cosine similarity triplicate)**

    Function: `cosineSimilarity(a, b []float32) float64`
    Position: `internal/ace/playbook/search.go:141-158`, `internal/ace/curator/deduplicator.go:65-83`, `internal/ace/retrieval/hnsw_retriever.go:164-181`, `internal/ace/refine/merge.go:143-160`
    Findings: **Four independent implementations** of the same cosine similarity function for `[]float32` vectors. All are identical in logic. The HNSW retriever also has its own `sqrt` instead of using `math.Sqrt`. Should be a single shared function in `ace/embedding` or a math utility package.

52. **internal/ace/refine**

    Function: `Archive` (concurrent map of archived bullets)
    Position: `internal/ace/refine/archive.go:34-146`
    Findings: `map[string]*ArchivedBullet` with `sync.RWMutex` and Archive/Get/List/Stats/Clear/Len methods. **Yet another concurrent map** instance (finding 47, 28, 37, 39).

53. **internal/ace/adapter**

    Function: Duplicate sentinel errors (`ErrSessionNotFound` x3)
    Position: `internal/ace/adapter/adapter.go:28-34`
    Findings: Three identical `ErrSessionNotFound` errors. Same anti-pattern.

54. **internal/ace/adapter**

    Function: `adapter.sessions` (session map with mutex)
    Position: `internal/ace/adapter/adapter.go:61-62`
    Findings: `map[string]*Session` with `sync.RWMutex` and Start/Get/End methods. **Another concurrent map** instance.

55. **internal/ace/generator**

    Function: `checkSuccess` (text similarity comparison)
    Position: `internal/ace/generator/generator.go:318-363`
    Findings: Multi-strategy text comparison (exact, normalized, contains, word overlap with 70% threshold). The word-overlap strategy is similar to `cycle.calculateSimilarity` (finding 24) — both compute Jaccard-like word overlap. Could share a common text similarity utility.
    Could replace:
      - Overlaps conceptually with `internal/cycle/similarity.go` word-based similarity

56. **internal/ace/delta**

    Function: `History` (append-only delta log with index)
    Position: `internal/ace/delta/history.go:9-141`
    Findings: Append-only log with `sync.RWMutex`, indexed by bulletID. Has `GetByBullet`, `GetRecent`, `GetSince`, `Stats`, `Clear`, `Len`. While domain-specific, the underlying pattern (append-only indexed log) is generic and reusable.

57. **internal/ace/delta**

    Function: Worker pool in `ApplyBatch`
    Position: `internal/ace/delta/batch.go:40-89`
    Findings: Worker pool pattern (jobs channel → N workers → results channel → collect). **Same pattern** as `curator.curateBatchParallel` (curator/parallel.go:18-118). Both use identical structure: effectiveWorkers, WaitGroup, channel-based fan-out/fan-in. Could be extracted as a generic `WorkerPool[In, Out]`.
    Could replace:
      - Same worker pool pattern as `internal/ace/curator/parallel.go`

58. **internal/ace/retrieval**

    Function: Custom `sqrt` function (Newton's method)
    Position: `internal/ace/retrieval/hnsw_retriever.go:184-195`
    Findings: Reimplements square root using Newton's method instead of using `math.Sqrt` from stdlib. Unnecessary — `math.Sqrt` is faster and more accurate. Should be replaced.

59. **internal/ace/playbook**

    Function: `Search` and `SearchWithScores` (duplicated search logic)
    Position: `internal/ace/playbook/search.go:20-138`
    Findings: `Search` and `SearchWithScores` contain **nearly identical implementations** — both generate query embedding, iterate bullets, compute cosine similarity, clamp, sort, and return top-k. The only difference is the return type. `Search` should call `SearchWithScores` and strip the scores.

60. **internal/agentsmd**

    Function: `readWithLimit` (size-bounded file reading)
    Position: `internal/agentsmd/service.go:139-181`
    Findings: Reads file with size cap and truncation notice. Generic utility for safe file reading with limits. Not duplicated elsewhere but could be useful in other contexts (e.g., reading large config files, log files).

61. **internal/task**

    Function: 4 identical `Validate()` implementations across task types
    Position: `internal/task/compact.go:80-90`, `planning.go:99-110`, `regular.go:101-112`, `review.go:95-105`
    Findings: `Compact.Validate()`, `Planning.Validate()`, `Regular.Validate()`, `Review.Validate()` are **4 identical implementations** — all check `maxTokens <= 0` and `maxTokens > MaxAllowedTokens`. Could be extracted to a shared `validateMaxTokens(maxTokens int) error` function or embedded in a base struct.

62. **internal/task**

    Function: Duplicate `MaxAllowedTokens` constant
    Position: `internal/task/constants.go:5` and `internal/task/regular.go:7`
    Findings: `MaxAllowedTokens = 100000` (exported, in constants.go) and `maxAllowedTokens = 100000` (unexported, in regular.go). **Same constant defined twice** — `regular.go` uses the private version while other files use the public one.

63. **internal/tui**

    Function: `atomicCounter` (custom atomic counter with mutex)
    Position: `internal/tui/mapper.go:941-954`
    Findings: Custom atomic counter using `sync.Mutex` for thread-safe increment. Should use `sync/atomic.Int64` from stdlib — simpler, lock-free, and faster.

64. **internal/protocol/acp**

    Function: `extractToolName` defined twice
    Position: `internal/protocol/acp/approval_handler.go:107-113` and `internal/ace/trajectory/analysis.go:72-93`
    Findings: Two different `extractToolName` functions with different logic — one extracts from `security.ApprovalRequest`, the other from step content text. Not directly duplicated but name collision suggests domain concept overlap.

65. **internal/protocol/acp**

    Function: `fileContentTracker` (map with mutex for tracking file state)
    Position: `internal/protocol/acp/notifications.go:15-90`
    Findings: `map[string]string` x3 with `sync.Mutex` tracking old/new file content by toolID. Yet another concurrent map pattern, though specialized for diff generation.

66. **cmd/spin/root.go**

    Function: 5 `flagX` helper functions (flagModel, flagProvider, flagWorkDir, flagConfigFile, flagAgentsMD)
    Position: `cmd/spin/root.go:10-42`
    Findings: All 5 functions follow identical pattern: `v, _ := cmd.Root().PersistentFlags().GetString(name); return v`. Could be replaced by a single generic helper: `func flagString(cmd *cobra.Command, name string) string`.

67. **cmd/spin/tui.go + exec.go**

    Function: `processEvent` and `processExecEvent` (duplicate event processing)
    Position: `cmd/spin/tui.go:188-195` and `cmd/spin/exec.go:321-335`
    Findings: Both functions call `mapper.MapEvent()` and then update token counts from events. Nearly identical logic — TUI version calls `updateTokensFromEvent`, exec version inlines the same switch. Should share a common event processor.

68. **cmd/spin/tui.go + exec.go**

    Function: `startEventLoop` and `startExecEventLoop` (duplicate event loops)
    Position: `cmd/spin/tui.go:210-235` and `cmd/spin/exec.go:338-357`
    Findings: Both goroutine-based event loops with identical `select` on `ctx.Done()` and event channel. The only difference is TUI version returns `chan struct{}` while exec version does not. Should be unified.

69. **cmd/spin/tui.go**

    Function: `createConversationForTUI` duplicates `buildConversation`
    Position: `cmd/spin/tui.go:376-467` and `cmd/spin/exec.go:229-260`
    Findings: `createConversationForTUI` manually wires `conversation.NewBuilder` with Git/Shell/MCP/ToolSelector — the **same logic** as `buildConversation` in exec.go. The TUI version inlines what exec.go already extracted into a reusable function. TUI should call `buildConversation` instead.

70. **cmd/spin/tui.go**

    Function: Inline session resolution (duplicates `resolveSessionID`)
    Position: `cmd/spin/tui.go:397-404`
    Findings: Session ID resolution logic (`if storage != nil { sess := session.NewSession(workDir); sessionID = sess.ID } else { "tui-" + timestamp }`) is **identical** to `resolveSessionID` in exec.go but written inline. Should call `resolveSessionID(storage, workDir, "tui")` instead.

71. **cmd/spin/acp.go**

    Function: `setupACPServerSignalHandling` (duplicate signal handling)
    Position: `cmd/spin/acp.go:423-432`
    Findings: Signal handling with `signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)` + goroutine that calls `cancel()`. **Identical** to `setupSignalHandling` in tui.go:470-478. ACP version adds a stderr message but core logic is the same. Should be unified.

72. **cmd/spin/config.go**

    Function: Duplicate sentinel errors (`ErrValidationFailed` x2, `ErrNoConfigFile` x2)
    Position: `cmd/spin/config.go:18-29`
    Findings: `ErrValidationFailed` and `ErrValidationFailed2` (both `"validation failed"`), `ErrNoConfigFile` and `ErrNoConfigFile2` (both `"no config file"`). Same duplicate sentinel error anti-pattern.

73. **cmd/spin/config.go**

    Function: `printJSON` and `outputJSON` (two JSON output functions)
    Position: `cmd/spin/config.go:369-378` and `cmd/spin/mcp.go:1245-1254`
    Findings: `printJSON[T any](out io.Writer, data T)` in config.go and `outputJSON[T any](data T)` in mcp.go both create `json.NewEncoder`, set indent, and encode. **Two implementations** differing only in the output destination (`io.Writer` param vs hardcoded `os.Stdout`). `outputJSON` should call `printJSON(os.Stdout, data)`.

74. **cmd/spin/mcp.go**

    Function: Inline registry filtering in `runMCPListTools` (duplicates `filterServersByRegistry`)
    Position: `cmd/spin/mcp.go:1077-1087` vs `cmd/spin/mcp.go:860-878`
    Findings: `runMCPListTools` has inline filter loop that **duplicates** `filterServersByRegistry` from the same file. Should call the existing function.

75. **cmd/spin/mcp.go**

    Function: Repeated "load config + create MCP store" boilerplate
    Position: `cmd/spin/mcp.go:220-228, 330-337, 468-474, 531-537, 604-610, 733-738, 937-943, 1062-1069`
    Findings: 8 command handlers all repeat the same 4-line pattern: `loader := config.NewLoaderV2(); _, err := loader.LoadFromFile(flagConfigFile(cmd)); mgr := config.NewMCPConfigStore(loader)`. Should extract a helper: `func loadMCPConfigStore(cmd *cobra.Command) (*config.MCPConfigStore, error)`.

76. **cmd/spin/approval_handlers.go**

    Function: `createAutoApproveHandler`, `createDenyHandler` (approval handler factories)
    Position: `cmd/spin/approval_handlers.go:11-30`
    Findings: Simple factory functions creating `security.ApprovalHandler` closures. These are TUI/exec specific but the auto-approve/deny patterns could live in the `security` package as `security.AutoApproveHandler()` and `security.DenyHandler(reason)` for wider reuse.

77. **cmd/spin/acp.go**

    Function: `buildProviderForACP` duplicates `buildProvider` pattern
    Position: `cmd/spin/acp.go:345-359` and `cmd/spin/exec.go:156-167`
    Findings: Both follow same pattern: try extra provider → fallback to `builder.NewBuilder(cfg, authMgr).Build(ctx)`. Could be unified into a single factory function with an optional extra-provider hook.

78. **cmd/spin/tui.go**

    Function: `configureMaxTokens` (dead iteration)
    Position: `cmd/spin/tui.go:158-171`
    Findings: Iterates `provider.Models()` looking for current model but never uses the found model — just breaks. The loop body is empty. Either incomplete or dead code.


