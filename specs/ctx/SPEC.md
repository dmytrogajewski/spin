# Missing Context Propagation Audit

## Summary

- **Total findings**: 50
- **Definite issues**: 14
- **Likely issues**: 19
- **Design recommendations**: 17

## Audit Rules Used

Every Go source file was inspected for these patterns:

1. **Missing parameter**: Functions performing I/O, network, or blocking work without accepting `context.Context`.
2. **Dropped propagation**: A parent function has `ctx` but downstream calls drop it or use `_`.
3. **Background/TODO misuse**: `context.Background()` or `context.TODO()` used where a caller-provided context should be threaded.
4. **Missing cancellation**: Goroutines, loops, retries, or workers that should listen to `ctx.Done()` but do not.
5. **Missing timeout**: Long-running or remote operations with no deadline boundary.
6. **Non-context API used**: Using non-context-aware variants (e.g., `exec.Command` vs `exec.CommandContext`, `http.NewRequest` vs `http.NewRequestWithContext`).
7. **Context misuse**: Storing context in structs, nil context, or deriving context without calling cancel.

Entrypoints traced: CLI commands (`cmd/spin/`), ACP protocol server, MCP registries, LSP transport, TUI event loop, background task manager. Call paths followed top-down from entrypoints and bottom-up from I/O boundaries.

---

## Findings

### CTX-001 BackgroundTaskManager.Start uses context.Background()
- **Severity**: Definite
- **File**: `internal/agent/executor/background.go`
- **Symbol**: `BackgroundTaskManager.Start`
- **Category**: Background/TODO misuse | Missing parameter
- **Evidence**:
  - Line 141: `cmd := exec.CommandContext(context.Background(), program, args...)`
  - `Start` does not accept a `context.Context` parameter at all.
  - Background commands are spawned with `context.Background()`, making them completely uncancelable through context.
  - The only termination path is calling `Kill` explicitly or `Cleanup`.
- **Why this matters**: If the parent agent/conversation is canceled, background processes continue running. No graceful shutdown path via context exists.
- **Recommended fix**: Add `ctx context.Context` as first parameter to `Start`. Use it (or derive from it with `context.WithCancel`) when creating `exec.CommandContext`. The monitor goroutine should also listen to `ctx.Done()`.
- **Confidence**: High

---

### CTX-002 BackgroundTaskManager.monitor goroutine has no context
- **Severity**: Definite
- **File**: `internal/agent/executor/background.go`
- **Symbol**: `BackgroundTaskManager.monitor`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 261-293: The goroutine calls `task.cmd.Wait()` and has no context awareness.
  - No way to be notified of graceful shutdown other than the process exiting.
- **Why this matters**: Combined with CTX-001, there is no context propagation path from application layer to background task lifecycle management.
- **Recommended fix**: Accept a context, select on `ctx.Done()` alongside process completion via the `done` channel.
- **Confidence**: High

---

### CTX-003 ProviderCache.Get passes caller's context to background refresh
- **Severity**: Definite
- **File**: `internal/llm/cache/provider_cache.go`
- **Symbol**: `ProviderCache.Get`
- **Category**: Context misuse
- **Evidence**:
  - Line 77: `go pc.refreshInBackground(ctx, provider, model)` -- the `ctx` is the caller's request context.
  - When `Get` returns immediately (stale-while-revalidate), the caller may cancel their context, which cancels the background refresh.
- **Why this matters**: The entire purpose of stale-while-revalidate is to refresh in the background. If the caller's context is short-lived, the background fetch is cancelled before completion, defeating the purpose.
- **Recommended fix**: Create a detached context: `bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)` and use it in `refreshInBackground`.
- **Confidence**: High

---

### CTX-004 OpenAI Provider.Models pagination ignores context
- **Severity**: Definite
- **File**: `internal/llm/openai/provider.go`
- **Symbol**: `Provider.Models`
- **Category**: Missing cancellation | Dropped propagation
- **Evidence**:
  - Lines 189-197: Pagination loop calls `resp.GetNextPage()` (HTTP requests) without context checks.
  - Error from `GetNextPage()` is silently discarded (`_`).
  - If the caller cancels, the pagination loop continues making HTTP calls.
- **Why this matters**: Uninterruptible network I/O loop. Resource waste and unresponsive to cancellation.
- **Recommended fix**: Check `ctx.Err()` at each iteration. Handle the error from `GetNextPage`.
- **Confidence**: High

---

### CTX-005 LSP readLoop silent exit leaves pending callers hanging
- **Severity**: Definite
- **File**: `internal/lsp/transport.go`
- **Symbol**: `StdioTransport.readLoop`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 199-227: When read errors occur (e.g., server crash), all pending `Send` calls hang until their context times out.
  - The `done` channel is only closed by `Close()`, not by `readLoop` exiting on error.
  - `headerErr` causes silent `return`; `unmarshalErr` causes silent `continue`.
- **Why this matters**: Server crash leaves callers blocked until context timeout. No immediate error propagation.
- **Recommended fix**: When `readLoop` exits due to error (not clean close), close the `done` channel or set an error state so pending `Send` calls return immediately.
- **Confidence**: High

---

### CTX-006 SmitheryRegistry.Close doesn't close dynamically loaded servers
- **Severity**: Definite
- **File**: `internal/mcp/registry_smithery.go`
- **Symbol**: `SmitheryRegistry.Close`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 298-314: `Close()` only closes `r.client`. Dynamically loaded `RemoteRegistry` instances in `r.loadedServers` are never closed.
  - Their connections, goroutines, and resources leak.
- **Why this matters**: Resource leak -- connections and goroutines from dynamically loaded MCP servers are never cleaned up.
- **Recommended fix**: Iterate over `r.loadedServers` and call `Close()` on each `RemoteRegistry`.
- **Confidence**: High

---

### CTX-007 FilePolicyStore.persistGlobalLocked blocks on flock without context
- **Severity**: Definite
- **File**: `internal/safety/policy_file_store.go`
- **Symbol**: `FilePolicyStore.persistGlobalLocked`
- **Category**: Missing cancellation | Missing timeout
- **Evidence**:
  - Acquires `syscall.Flock(LOCK_EX)` which can block indefinitely if another process holds the lock.
  - No timeout or cancellation mechanism.
  - Callers (`Save`, `Delete`, `Clear`) accept `_ context.Context` but never use it.
- **Why this matters**: `syscall.Flock` can block indefinitely with no cancellation. The context parameter is already in the method signatures but is discarded.
- **Recommended fix**: Use the `ctx` parameter. Check `ctx.Err()` before locking, or use a non-blocking flock with retry and ctx checks.
- **Confidence**: High

---

### CTX-008 storage.Store interface lacks context on all methods
- **Severity**: Definite
- **File**: `internal/storage/store.go`
- **Symbol**: `Store[T]` interface
- **Category**: Missing parameter
- **Evidence**:
  - Interface: `Save(key string, data T) error`, `Load(key string) (T, error)`, `Delete(key string) error`, `Exists(key string) (bool, error)`, `List() ([]string, error)`
  - `FileStore` implementation performs file I/O (reads, writes, directory listing) on all methods.
  - Used by `session.Storage`, `history.Storage`, persistent memory, and others.
- **Why this matters**: Systemic interface-level omission. All consumers inherit the inability to cancel storage operations. This is the root cause for multiple downstream findings.
- **Recommended fix**: Add `ctx context.Context` as first parameter to all interface methods. Update `FileStore` and all consumers.
- **Confidence**: High

---

### CTX-009 AtomicWriteFile performs multi-step I/O without context
- **Severity**: Definite
- **File**: `internal/storage/atomic.go`
- **Symbol**: `AtomicWriteFile`
- **Category**: Missing parameter
- **Evidence**:
  - Creates temp file, writes data, sets permissions, renames -- multiple I/O operations.
  - Used throughout: playbook save, config write, session index, persistent memory.
  - No context parameter, no cancellation between steps.
- **Why this matters**: On slow/hung filesystems, blocks indefinitely. Widely used across the codebase.
- **Recommended fix**: Add `ctx context.Context` and check `ctx.Err()` between I/O operations.
- **Confidence**: High

---

### CTX-010 TranscriptWriter.Append and ReadAll block on flock without context
- **Severity**: Definite
- **File**: `internal/session/transcript.go`
- **Symbol**: `TranscriptWriter.Append`, `TranscriptWriter.ReadAll`
- **Category**: Missing cancellation | Missing timeout
- **Evidence**:
  - `Append` acquires `syscall.Flock(LOCK_EX)` then writes data.
  - `ReadAll` acquires `syscall.Flock(LOCK_SH)` then reads entire file.
  - Neither accepts context. Called from the request/message path.
- **Why this matters**: Exclusive flock can block indefinitely on lock contention. This is in the hot path for every conversation turn.
- **Recommended fix**: Add `ctx context.Context` parameter. Use non-blocking flock with retry loop checking `ctx.Done()`.
- **Confidence**: High

---

### CTX-011 PersistentStore.Search accepts context but ignores it
- **Severity**: Definite
- **File**: `internal/memory/persistent.go`
- **Symbol**: `PersistentStore.Search`
- **Category**: Dropped propagation
- **Evidence**:
  - Line 281: `func (s *PersistentStore) Search(_ context.Context, query string, topK int)`
  - Reads potentially many files from disk (one per index entry) in a loop.
  - Context parameter is named `_` -- explicitly ignored.
- **Why this matters**: A large memory store means hundreds of file reads with no cancellation. The API surface already has context but the implementation drops it.
- **Recommended fix**: Use the `ctx` parameter. Check `ctx.Err()` in the loop and return early if cancelled.
- **Confidence**: High

---

### CTX-012 PersistentStore Put/Get/Delete/List all ignore context
- **Severity**: Definite
- **File**: `internal/memory/persistent.go`
- **Symbol**: `PersistentStore.Put`, `.Get`, `.Delete`, `.List`
- **Category**: Dropped propagation
- **Evidence**:
  - All methods use `_ context.Context` as the first parameter but never use it.
  - `Put` creates directories, marshals JSON, does atomic file writes.
  - `Get` reads files. `Delete` removes files. `List` reads directory.
  - None check for cancellation.
- **Why this matters**: File I/O without cancellation. The API promises context support but the implementation ignores it entirely.
- **Recommended fix**: Use the `ctx` parameter. Check `ctx.Err()` before I/O operations.
- **Confidence**: High

---

### CTX-013 runACPServer uses bare context.Background() for infrastructure creation
- **Severity**: Definite
- **File**: `cmd/spin/acp.go`
- **Symbol**: `runACPServer`
- **Category**: Background/TODO misuse
- **Evidence**:
  - Line 248: `ctx := context.Background()` used for `createACPInfra` (line 250).
  - `ctx, cancel := context.WithCancel(ctx)` only created at line 288, after all services are initialized.
  - If a signal arrives during infrastructure creation, operations cannot be cancelled.
- **Why this matters**: Services like `git.NewService`, `shell.NewService`, MCP registry initialization, and LLM provider building all receive uncancelable context.
- **Recommended fix**: Move `ctx, cancel := context.WithCancel(context.Background())` to the top, before `createACPInfra`.
- **Confidence**: High

---

### CTX-014 ACP SetMode uses context.Background() when caller context available
- **Severity**: Definite
- **File**: `internal/protocol/acp/commands.go`
- **Symbol**: `acpCommandContext.SetMode`
- **Category**: Background/TODO misuse
- **Evidence**:
  - Line 37: `_, err := c.agent.SetSessionMode(context.Background(), req)`
  - Called from `executeCommand` which has `ctx context.Context` available.
  - The `CommandContext` interface's `SetMode` does not accept context.
- **Why this matters**: Caller context (with cancellation from protocol layer) is discarded. Mode changes cannot be cancelled.
- **Recommended fix**: Add `context.Context` to `CommandContext.SetMode` interface, or capture context in `acpCommandContext` struct.
- **Confidence**: High

---

### CTX-015 BackgroundTaskManager.waitStartup goroutine not context-aware
- **Severity**: Likely
- **File**: `internal/agent/executor/background.go`
- **Symbol**: `BackgroundTaskManager.waitStartup`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 300-315: Goroutine reads from startup pipe with no context awareness.
  - Only terminates when pipe is closed or data drained.
- **Why this matters**: If the parent context is canceled, this goroutine blocks on pipe read until the background process exits.
- **Recommended fix**: Accept context, select on `ctx.Done()` alongside scanner loop, or set a read deadline.
- **Confidence**: Medium

---

### CTX-016 streamOutput channel send not guarded by ctx select
- **Severity**: Likely
- **File**: `internal/agent/executor.go`
- **Symbol**: `Executor.streamOutput`
- **Category**: Missing cancellation
- **Evidence**:
  - Line 878: `chunks <- OutputChunk{...}` is not guarded by a select on `ctx.Done()`.
  - If the consumer abandons the channel (buffer fills up), the goroutine blocks on send despite context cancellation.
  - Context check happens before `r.Read()` but not around the channel send.
- **Why this matters**: Goroutine leak when consumer abandons channel on cancellation.
- **Recommended fix**: `select { case chunks <- chunk: case <-ctx.Done(): return }`.
- **Confidence**: Medium

---

### CTX-017 Ollama setContextOptions has race condition on detectedCtxLen
- **Severity**: Likely
- **File**: `internal/llm/ollama/provider.go`
- **Symbol**: `Provider.setContextOptions`
- **Category**: Context misuse
- **Evidence**:
  - Lines 161-174: `if p.detectedCtxLen == 0 { p.detectedCtxLen = p.detectContextLength(ctx) }`
  - No locking. Multiple concurrent callers trigger `detectContextLength` simultaneously (network calls).
- **Why this matters**: Race condition causing redundant network calls. Results are identical but wastes resources.
- **Recommended fix**: Use `sync.Once` or a mutex.
- **Confidence**: Medium

---

### CTX-018 handleGitStatus drops context and returns stale data
- **Severity**: Likely
- **File**: `internal/tools/git_operation_tool.go`
- **Symbol**: `handleGitStatus`
- **Category**: Dropped propagation
- **Evidence**:
  - Signature: `func handleGitStatus(_ context.Context, t *GitOperationTool, _ ToolParameters)`
  - Calls `t.gitIntegration.GetStatus()` which returns cached status without refreshing.
  - Should call `RefreshStatus(ctx)` first, which does require context.
- **Why this matters**: Returns stale data without indicating staleness. Context dropped prevents refresh.
- **Recommended fix**: Call `t.gitIntegration.RefreshStatus(ctx)` before `GetStatus()`.
- **Confidence**: Medium

---

### CTX-019 TaskManager interface lacks context on all methods
- **Severity**: Likely
- **File**: `internal/tools/task_manager.go`
- **Symbol**: `TaskManager` interface
- **Category**: Missing parameter
- **Evidence**:
  - `List() []TaskSnapshot`, `GetOutput(taskID string, maxLines int) (string, error)`, `Kill(taskID string) error`
  - None accept context. This structurally blocks context propagation for ListProcesses, GetProcessOutput, and KillProcess tools.
- **Why this matters**: Interface design prevents proper context propagation through all tools that use it.
- **Recommended fix**: Add `context.Context` as first parameter to all `TaskManager` methods.
- **Confidence**: Medium

---

### CTX-020 FilePolicyStore.loadFromDisk acquires flock without context
- **Severity**: Likely
- **File**: `internal/safety/policy_file_store.go`
- **Symbol**: `FilePolicyStore.loadFromDisk`
- **Category**: Missing cancellation
- **Evidence**:
  - Opens file, acquires `syscall.Flock(LOCK_SH)`, reads and decodes JSON -- all without context.
  - Called once from constructor.
- **Why this matters**: Shared lock can block if exclusive lock is held by another process.
- **Recommended fix**: Accept context parameter and check for cancellation.
- **Confidence**: Medium

---

### CTX-021 savePlaybookAfterUpdate fire-and-forget goroutine without context
- **Severity**: Likely
- **File**: `internal/ace/service.go`
- **Symbol**: `Service.savePlaybookAfterUpdate`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 464-477: `go func() { _ = s.SavePlaybook() }()` when `UpdateAsync` is true.
  - `SavePlaybook()` does file I/O but doesn't accept context.
  - The caller (`UpdateBullets`) has a `ctx` parameter but it's not passed to the goroutine.
- **Why this matters**: Fire-and-forget I/O with no cancellation or timeout. Partial writes possible on process exit.
- **Recommended fix**: Pass `context.WithoutCancel(ctx)` with a timeout. Propagate context into `SavePlaybook`.
- **Confidence**: Medium

---

### CTX-022 checkGrowthAndRefine background goroutine has no timeout
- **Severity**: Likely
- **File**: `internal/ace/service.go`
- **Symbol**: `Service.checkGrowthAndRefine`
- **Category**: Missing timeout
- **Evidence**:
  - Lines 810-835: `bgCtx := context.WithoutCancel(ctx)` used in background goroutine.
  - `Refine` call can trigger LLM calls that hang indefinitely. No deadline set on `bgCtx`.
- **Why this matters**: Background LLM calls can hang forever without a timeout boundary.
- **Recommended fix**: `bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)`.
- **Confidence**: Medium

---

### CTX-023 OllamaEmbedder HTTP client has no timeout
- **Severity**: Likely
- **File**: `internal/ace/embedding/ollama_embedder.go`
- **Symbol**: `NewOllamaEmbedder`
- **Category**: Missing timeout
- **Evidence**:
  - Line 56: `client := api.NewClient(baseURLParsed, &http.Client{})` -- no timeout set.
  - While `Embed()` passes `ctx`, if ctx has no deadline, HTTP calls hang indefinitely.
- **Why this matters**: Embedding calls to Ollama can hang forever without a client timeout safety net.
- **Recommended fix**: Set `&http.Client{Timeout: 30 * time.Second}`.
- **Confidence**: Medium

---

### CTX-024 session.Index save/load do file I/O without context
- **Severity**: Likely
- **File**: `internal/session/index.go`
- **Symbol**: `Index.save`, `Index.load`
- **Category**: Missing parameter
- **Evidence**:
  - `save()` calls `storage.AtomicWriteFile` (blocked by CTX-009).
  - `load()` calls `os.ReadFile`.
  - Public methods `Update`, `Remove`, `Rebuild` don't accept context.
- **Why this matters**: File I/O on request path without cancellation.
- **Recommended fix**: Add ctx to `Update`, `Remove`, `Rebuild`. Propagate to `save()` and `load()`.
- **Confidence**: Medium

---

### CTX-025 Index.Rebuild and MetadataScanner.ScanSessions lack context
- **Severity**: Likely
- **File**: `internal/session/index.go`
- **Symbol**: `Index.Rebuild`, `MetadataScanner`
- **Category**: Missing parameter
- **Evidence**:
  - `Rebuild(scanner MetadataScanner)` -- neither `Rebuild` nor `MetadataScanner.ScanSessions()` accept context.
  - `ScanSessions()` likely scans the filesystem.
- **Why this matters**: Filesystem scan not cancelable.
- **Recommended fix**: Add `ctx context.Context` to both `MetadataScanner.ScanSessions()` and `Index.Rebuild()`.
- **Confidence**: Medium

---

### CTX-026 PersistentStore.rebuildIndex walks directory without context
- **Severity**: Likely
- **File**: `internal/memory/persistent.go`
- **Symbol**: `PersistentStore.rebuildIndex`
- **Category**: Missing parameter
- **Evidence**:
  - Lines 328-373: `filepath.Walk` reads every `.json` file in the directory tree.
  - Called from `NewPersistentStore()` constructor. No way to cancel.
- **Why this matters**: Large memory store means significant uncancelable I/O on startup.
- **Recommended fix**: Accept `ctx context.Context` in `NewPersistentStore` and check `ctx.Err()` in the walk callback.
- **Confidence**: Medium

---

### CTX-027 Keystore interface methods lack context
- **Severity**: Likely
- **File**: `internal/auth/auth.go`
- **Symbol**: `Keystore` interface
- **Category**: Missing parameter
- **Evidence**:
  - `Manager.GetCredential` checks `ctx.Err()` once at entry, then calls `m.keystore.Get(...)` which has no context.
  - On Linux, `linuxKeystore` calls `keyring.Get/Set/Delete` which make D-Bus calls that can block.
- **Why this matters**: D-Bus calls to system keyring can hang. No cancellation possible.
- **Recommended fix**: Add `ctx context.Context` to `Keystore` interface methods.
- **Confidence**: Medium

---

### CTX-028 contexteng/history Save/Load delegate to context-less Store
- **Severity**: Likely
- **File**: `internal/contexteng/history/storage.go`
- **Symbol**: `History.Save`, `History.Load`
- **Category**: Missing parameter
- **Evidence**:
  - Neither `Save` nor `Load` accept context.
  - Delegate to `storage.Store` which also lacks context (CTX-008).
- **Why this matters**: File I/O without cancellation on the conversation history path.
- **Recommended fix**: Add `ctx context.Context`. Blocked by CTX-008 (Store interface needs context first).
- **Confidence**: Medium

---

### CTX-029 ACE delta batch workers don't check ctx.Done()
- **Severity**: Likely
- **File**: `internal/ace/delta/batch.go`
- **Symbol**: `Applier.runBatchWorkers`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 58-89: Workers iterate over `jobs` channel without selecting on `ctx.Done()`.
  - If context is cancelled, workers continue processing remaining jobs until channel closed.
- **Why this matters**: No early exit on cancellation for batch LLM operations.
- **Recommended fix**: Add `select { case j := <-jobs: ... case <-ctx.Done(): return }` in the worker loop.
- **Confidence**: Medium

---

### CTX-030 MCP Smithery search HTTP client has no timeout
- **Severity**: Likely
- **File**: `cmd/spin/mcp.go`
- **Symbol**: `searchSmitheryAPI`
- **Category**: Missing timeout
- **Evidence**:
  - Line 1003: `http.Client{}` has no timeout configured.
  - Context passed to `NewRequestWithContext` provides cancellation but no deadline.
- **Why this matters**: If Smithery API is unresponsive, request hangs indefinitely.
- **Recommended fix**: `&http.Client{Timeout: 30 * time.Second}` or `context.WithTimeout`.
- **Confidence**: Medium

---

### CTX-031 EventEmitter.Emit BackpressureBlock can deadlock without cancellation
- **Severity**: Likely
- **File**: `internal/events/event.go`
- **Symbol**: `EventEmitter.Emit`
- **Category**: Missing parameter | Missing cancellation
- **Evidence**:
  - Line 632: `Emit(event Event)` has no context parameter.
  - In `BackpressureBlock` mode (line 673), `emitBlock` does a blocking channel send with no cancellation.
  - If a subscriber is permanently stuck, the emitter goroutine deadlocks.
- **Why this matters**: Potential deadlock in event emission. No way to cancel or timeout the emit.
- **Recommended fix**: Add `context.Context` to `Emit`, or use `BackpressureBlock` with an internal timeout.
- **Confidence**: Medium

---

### CTX-032 git Repository.Status: go-git worktree.Status() is uncancelable
- **Severity**: Likely
- **File**: `internal/git/status.go`
- **Symbol**: `Repository.Status`
- **Category**: Missing cancellation
- **Evidence**:
  - Line 41: `gitStatus, err := worktree.Status()` -- go-git does not accept context.
  - Scans all files in working tree, can be slow for large repos.
  - The `select` on `ctx.Done()` at the top only catches already-canceled contexts.
- **Why this matters**: For large repos, `worktree.Status()` can take seconds with no way to interrupt.
- **Recommended fix**: Run `worktree.Status()` in a goroutine and select on both the result and `ctx.Done()`. This is a go-git library limitation.
- **Confidence**: Medium

---

### CTX-033 ApplyPatchTool discards context for multi-file operations
- **Severity**: Likely
- **File**: `internal/tools/apply_patch.go`
- **Symbol**: `ApplyPatchTool.Execute`
- **Category**: Dropped propagation
- **Evidence**:
  - `Execute(_ context.Context, params ToolParameters)` -- context discarded.
  - `applyPatch` method calls `patchapply.NewApplier` and `applier.Apply(patch)` performing multi-file I/O.
- **Why this matters**: Multi-file patch operations can take meaningful time with no way to cancel.
- **Recommended fix**: Thread context through to `patchapply` package, check `ctx.Err()` between file operations.
- **Confidence**: Medium

---

### CTX-034 Multiple file tools (Read, Write, Edit, List, Search) discard context
- **Severity**: Recommendation
- **File**: `internal/tools/read_file.go`, `write_file.go`, `edit_file.go`, `list_directory.go`, `file_search.go`
- **Symbol**: `ReadFileTool.Execute`, `WriteFileTool.Execute`, `EditFileTool.Execute`, `ListDirectoryTool.Execute`
- **Category**: Dropped propagation
- **Evidence**:
  - All use `Execute(_ context.Context, params ToolParameters)` -- context explicitly discarded.
  - All perform file I/O (`os.ReadFile`, `os.WriteFile`, `os.ReadDir`) without cancellation.
- **Why this matters**: On network filesystems or with large files, these block without cancellation. For local files, typically fast.
- **Recommended fix**: Check `ctx.Err()` before I/O operations at minimum.
- **Confidence**: Low

---

### CTX-035 LSP NewStdioTransport readLoop has no context-based shutdown
- **Severity**: Recommendation
- **File**: `internal/lsp/transport.go`
- **Symbol**: `NewStdioTransport`, `StdioTransport.readLoop`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 79-88: `go transport.readLoop()` -- goroutine spawned with no context.
  - Only exits when `st.closed.Load()` is true or read error occurs.
  - No way for a caller to cancel via context without calling `Close()`.
- **Why this matters**: If parent only signals via context cancellation (without calling Close), this goroutine leaks.
- **Recommended fix**: Accept `context.Context` in `NewStdioTransport` and check `ctx.Done()` in `readLoop`.
- **Confidence**: Low

---

### CTX-036 Ollama global http.Client.Timeout may conflict with context
- **Severity**: Recommendation
- **File**: `internal/llm/ollama/provider.go`
- **Symbol**: `NewProvider`
- **Category**: Missing timeout
- **Evidence**:
  - Lines 81-82: `&http.Client{Timeout: timeout}` sets a global 5-minute timeout.
  - For streaming, this global timeout may be too short. The ctx-based timeout is already correctly propagated to `p.client.Chat(ctx, ...)`.
- **Why this matters**: Global client timeout could prematurely kill streaming requests.
- **Recommended fix**: Remove global `http.Client.Timeout` (or set to 0) and rely on context-based timeouts.
- **Confidence**: Low

---

### CTX-037 scanProjectFiles performs file walk without context
- **Severity**: Recommendation
- **File**: `internal/agent/environment.go`
- **Symbol**: `scanProjectFiles`
- **Category**: Missing parameter
- **Evidence**:
  - `GatherEnvironment` accepts `ctx` but `scanProjectFiles` is called without it.
  - `filepath.WalkDir` scans potentially thousands of files.
  - Bounded by `maxFiles` and `maxDepth`, limiting damage.
- **Why this matters**: Context is available in the parent but not passed down. Walk should be interruptible.
- **Recommended fix**: Pass context into `scanProjectFiles`, check `ctx.Err()` in the walk callback.
- **Confidence**: Low

---

### CTX-038 Conversation stores CancelFunc in struct
- **Severity**: Recommendation
- **File**: `internal/conversation/conversation.go`
- **Symbol**: `Conversation` struct
- **Category**: Context misuse
- **Evidence**:
  - Line 68: `cancel context.CancelFunc` stored in struct.
  - `SetCancel` (line 217) and `GetCancel` (line 209) manage it.
  - Protected by mutex, used for ACP protocol-level turn cancellation.
- **Why this matters**: Storing `CancelFunc` in a struct is a known Go anti-pattern. However, the current usage appears intentional and safe due to mutex protection.
- **Recommended fix**: Consider returning cancel from `RunTurn` instead of storing, or document the rationale clearly.
- **Confidence**: Low

---

### CTX-039 Conversation builder initialization methods lack context
- **Severity**: Recommendation
- **File**: `internal/conversation/builder.go`, `internal/conversation/memory.go`
- **Symbol**: `Builder.initializeCoreDependencies`, `Builder.initializeMemory`
- **Category**: Missing parameter
- **Evidence**:
  - `initializeCoreDependencies` creates `session.NewFileStorage(sessionDir)` -- filesystem operations without context.
  - `initializeMemory` creates `memory.NewPersistentStore(basePath)` -- filesystem operations without context.
  - `Build` receives a context but doesn't forward it.
- **Why this matters**: Inconsistent -- parent has context but children don't receive it.
- **Recommended fix**: Pass `ctx` to initialization methods.
- **Confidence**: Low

---

### CTX-040 ACE middleware AfterExecution lacks separate timeout boundary
- **Severity**: Recommendation
- **File**: `internal/agent/middleware/ace/ace.go`
- **Symbol**: `Middleware.AfterExecution`
- **Category**: Missing timeout
- **Evidence**:
  - Line 91: `learnedBullets, err := am.generateBulletsFromTrajectory(ctx, traj)` -- LLM calls happen synchronously.
  - Competes for remaining budget of the parent context's timeout.
- **Why this matters**: LLM calls in middleware may timeout under tight parent deadlines.
- **Recommended fix**: Run `AfterExecution` bullet generation with a separate timeout boundary.
- **Confidence**: Low

---

### CTX-041 Approval commands use context.Background() instead of cmd.Context()
- **Severity**: Recommendation
- **File**: `cmd/spin/approval.go`
- **Symbol**: `newApprovalListCmd`, `newApprovalRevokeCmd`, `newApprovalClearCmd`
- **Category**: Background/TODO misuse
- **Evidence**:
  - All three: `ctx, cancel := context.WithTimeout(context.Background(), approvalTimeout)`
  - Cobra provides `cmd.Context()` that is cancelled on signal.
- **Why this matters**: Ctrl-C during policy store operations won't cancel them -- only the 5s timeout fires.
- **Recommended fix**: Replace `context.Background()` with `cmd.Context()`.
- **Confidence**: Low

---

### CTX-042 Auth commands use context.Background()
- **Severity**: Recommendation
- **File**: `cmd/spin/auth.go`
- **Symbol**: `runAuthLogin`, `runAuthLogout`, `runAuthList`
- **Category**: Background/TODO misuse
- **Evidence**:
  - `ctx := context.Background()` used for keystore operations that can block (D-Bus on Linux).
- **Why this matters**: Hung keystore call cannot be interrupted.
- **Recommended fix**: Use `cmd.Context()` or `context.WithTimeout(cmd.Context(), ...)`.
- **Confidence**: Low

---

### CTX-043 filesearch.Scanner.Scan() uses context.Background()
- **Severity**: Recommendation
- **File**: `internal/filesearch/scanner.go`
- **Symbol**: `Scanner.Scan`
- **Category**: Background/TODO misuse
- **Evidence**:
  - Line 36: `return s.ScanWithContext(context.Background())`
  - `ScanWithContext` exists and properly checks `ctx.Done()`.
- **Why this matters**: Callers of `Scan()` cannot cancel directory walks.
- **Recommended fix**: Deprecate `Scan()` in favor of always requiring `ScanWithContext`.
- **Confidence**: Low

---

### CTX-044 patchapply.Apply has no context support
- **Severity**: Recommendation
- **File**: `internal/patchapply/applier.go`
- **Symbol**: `Applier.Apply`
- **Category**: Missing parameter
- **Evidence**:
  - No function in this package accepts `context.Context`.
  - Patch application involves file reads, writes, and directory creation for multiple files.
- **Why this matters**: Very large patches cannot be cancelled mid-application.
- **Recommended fix**: Add `context.Context` to `Apply()` and check `ctx.Err()` between operations.
- **Confidence**: Low

---

### CTX-045 Policy store janitor goroutines use stopCh instead of context
- **Severity**: Recommendation
- **File**: `internal/safety/policy.go`, `internal/safety/policy_file_store.go`
- **Symbol**: `MemoryPolicyStore.janitor`, `FilePolicyStore.janitor`
- **Category**: Missing cancellation
- **Evidence**:
  - Both: `go s.janitor()` with only `s.stopCh` for shutdown.
  - No context awareness. If `Close()` is not called, goroutine leaks.
- **Why this matters**: Requires explicit `Close()` call; forgetting it leaks the goroutine.
- **Recommended fix**: Accept context in constructor and listen to `ctx.Done()` in janitor.
- **Confidence**: Low

---

### CTX-046 hooks.Runner.executeAsync fire-and-forget goroutines
- **Severity**: Recommendation
- **File**: `internal/safety/hooks/runner.go`
- **Symbol**: `Runner.executeAsync`
- **Category**: Missing cancellation
- **Evidence**:
  - Lines 160-171: Goroutines spawned but never joined. Method returns immediately.
  - Goroutines use `exec.CommandContext` with parent ctx, so commands themselves get cancelled.
- **Why this matters**: Goroutines cannot be waited on during shutdown. Hook outcomes silently lost.
- **Recommended fix**: Use `sync.WaitGroup` or `errgroup` for graceful shutdown tracking.
- **Confidence**: Low

---

### CTX-047 SavePlaybook and playbook storage lack context
- **Severity**: Recommendation
- **File**: `internal/ace/service.go`, `internal/ace/playbook/storage.go`
- **Symbol**: `Service.SavePlaybook`, `Playbook.Save`, `Playbook.Load`
- **Category**: Missing parameter
- **Evidence**:
  - `SavePlaybook()`, `Playbook.Save(path)`, `Load(path)` all do file I/O without context.
  - Called from request paths and background goroutines.
- **Why this matters**: File I/O without cancellation.
- **Recommended fix**: Add `ctx context.Context` parameter.
- **Confidence**: Low

---

### CTX-048 MCPConfigStore writeConfig/Add/Remove lack context
- **Severity**: Recommendation
- **File**: `internal/config/mcp_manager.go`
- **Symbol**: `MCPConfigStore.writeConfig`, `.Add`, `.Remove`
- **Category**: Missing parameter
- **Evidence**:
  - `writeConfig()` calls `storage.AtomicWriteFile` without context (blocked by CTX-009).
  - `Add()` and `Remove()` don't accept context.
- **Why this matters**: Config file writes without cancellation.
- **Recommended fix**: Add `ctx context.Context` to `Add()`, `Remove()`, and `writeConfig()`.
- **Confidence**: Low

---

### CTX-049 Scratchpad methods accept but ignore context
- **Severity**: Recommendation
- **File**: `internal/memory/scratchpad.go`
- **Symbol**: `Scratchpad.Put`, `.Get`, `.Delete`, `.List`, `.Search`
- **Category**: Dropped propagation
- **Evidence**:
  - All use `_ context.Context`. Scratchpad is in-memory so I/O isn't an issue.
  - `Search` does string matching over all entries which could be slow with many entries.
- **Why this matters**: Breaks the context contract. Callers may depend on cancellation.
- **Recommended fix**: Check `ctx.Err()` at entry, especially in `Search` and `List` loops.
- **Confidence**: Low

---

### CTX-050 OpenAI Stream errors silently swallowed
- **Severity**: Recommendation
- **File**: `internal/llm/openai/provider.go`
- **Symbol**: `Provider.Stream`
- **Category**: Context misuse
- **Evidence**:
  - Lines 162-170: `err := stream.Err(); if err != nil { _ = mapError(err); return }`
  - Stream errors (including context cancellation errors) are mapped but never sent to the consumer.
  - Consumer sees only a closed channel with no way to distinguish success from error.
- **Why this matters**: Context cancellation errors during streaming are silently dropped.
- **Recommended fix**: Log the error at minimum. Consider adding an error channel or final-error mechanism.
- **Confidence**: Low

---

## Call-Chain Hotspots

### Hotspot 1: storage.Store -> session/history/memory
Fixing **CTX-008** (Store interface) unlocks fixes for:
- CTX-009 (AtomicWriteFile)
- CTX-024 (session.Index)
- CTX-028 (history Save/Load)
- CTX-011, CTX-012 (PersistentStore)
- CTX-048 (MCPConfigStore)

**Chain**: `conversation.Build()` -> `session.NewFileStorage()` -> `storage.FileStore.Save/Load` -> `os.ReadFile/WriteFile`

### Hotspot 2: BackgroundTaskManager lifecycle
Fixing **CTX-001** unlocks fixes for:
- CTX-002 (monitor goroutine)
- CTX-015 (waitStartup goroutine)

**Chain**: `tool.Execute("shell_command")` -> `BackgroundTaskManager.Start()` -> `exec.CommandContext(context.Background())` -> `monitor()` -> `waitStartup()`

### Hotspot 3: File locking paths
Fixing **CTX-007** and **CTX-010** together:
- Both use `syscall.Flock` without context
- Both are in hot paths (policy store and transcript)

**Chain**: `conversation.RunTurn()` -> `TranscriptWriter.Append()` -> `syscall.Flock(LOCK_EX)` (blocks forever)

### Hotspot 4: ACE service background operations
Fixing **CTX-021** and **CTX-022** together:
- Both involve fire-and-forget goroutines with LLM/I/O work

**Chain**: `ace.AfterExecution()` -> `savePlaybookAfterUpdate()` -> `go SavePlaybook()` (no ctx)
**Chain**: `ace.GenerateBullets()` -> `checkGrowthAndRefine()` -> `go Refine(bgCtx)` (no timeout)

### Hotspot 5: ACP server startup
Fixing **CTX-013** unlocks:
- CTX-014 (SetMode)
- Proper cancellation for all ACP infrastructure

**Chain**: `runACPServer()` -> `context.Background()` -> `createACPInfra()` -> all service creation

---

## Recommended Refactor Order

### Phase 1: High-leverage infrastructure fixes (unblocks many downstream fixes)
1. **CTX-008**: Add `ctx` to `storage.Store` interface -- cascades to 6+ dependent packages
2. **CTX-009**: Add `ctx` to `AtomicWriteFile` -- used by all atomic writes
3. **CTX-001 + CTX-002**: Add `ctx` to `BackgroundTaskManager.Start` and `monitor`

### Phase 2: Safety-critical fixes (blocking/deadlock risk)
4. **CTX-007**: FilePolicyStore flock with context
5. **CTX-010**: TranscriptWriter flock with context
6. **CTX-005**: LSP readLoop error propagation to pending callers
7. **CTX-006**: SmitheryRegistry.Close -- close loaded servers

### Phase 3: Correctness fixes (context misuse)
8. **CTX-003**: ProviderCache background refresh -- detach context
9. **CTX-004**: OpenAI pagination -- add context checks
10. **CTX-013**: ACP server -- move WithCancel before infrastructure creation
11. **CTX-014**: ACP SetMode -- propagate caller context
12. **CTX-011 + CTX-012**: PersistentStore -- use the ctx parameter already in signatures

### Phase 4: Propagation completeness
13. **CTX-019**: TaskManager interface -- add context
14. **CTX-027**: Keystore interface -- add context
15. **CTX-024 + CTX-025 + CTX-028**: Session/history -- add context
16. **CTX-029**: ACE delta batch workers -- add ctx.Done() check
17. **CTX-021 + CTX-022**: ACE background goroutines -- add timeout
18. **CTX-023 + CTX-030**: HTTP clients without timeouts

### Phase 5: Polish and consistency
19. **CTX-034**: File tools -- check ctx.Err() before I/O
20. **CTX-041 + CTX-042**: CLI commands -- use cmd.Context()
21. Remaining recommendation-level findings

---

## Notes

### Uncertain cases needing human review

1. **CTX-038 (CancelFunc in struct)**: The `Conversation.cancel` field appears intentionally designed for ACP protocol-level turn cancellation. Verify whether the current pattern is the desired architecture or if cancel should be returned to callers instead.

2. **CTX-032 (go-git worktree.Status)**: This is a third-party library limitation. The workaround (goroutine + select) adds complexity. Evaluate whether the current behavior is acceptable for the expected repo sizes.

3. **CTX-036 (Ollama client timeout)**: The global `http.Client.Timeout` acts as a safety net for streaming. Removing it requires confidence that all callers always provide proper context deadlines.

4. **CTX-017 (Ollama race condition)**: The `detectedCtxLen` race is benign in practice (same result computed twice). Fix with `sync.Once` only if test tooling flags it.

5. **CTX-031 (EventEmitter BackpressureBlock)**: Verify whether `BackpressureBlock` mode is actually used in production. If only `BackpressureDrop` is used, the deadlock risk is theoretical.

6. **CTX-050 (OpenAI stream errors)**: The comment in code acknowledges this limitation. Evaluate whether the streaming protocol can support error reporting without breaking the channel-based consumer pattern.

7. **File tools (CTX-034)**: For tools operating exclusively on local filesystems, the practical risk of blocking is very low. Adding context checks adds complexity. Consider whether the consistency benefit justifies the change.
