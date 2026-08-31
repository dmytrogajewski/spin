# Testing Guide

## Overview

Spin uses a three-tier testing strategy: unit tests, integration tests, and fixture-driven E2E tests. All tiers run via `make test` and must pass before merge.

```
make test       # all tests
make lint       # golangci-lint + deadcode analysis
go test ./...   # Go tests only
```

## Fixture-Driven E2E Tests

Fixture tests run the real `spin exec` binary against pre-recorded LLM responses. This gives deterministic, reproducible E2E coverage without requiring an LLM API key.

### How It Works

1. **Fixture files** live in `tests/e2e/fixtures/*.jsonl`.
2. Each JSONL line represents one LLM response (one `Stream()` call).
3. The `test-llm` provider replays fixture lines sequentially.
4. Tests assert on terminal output, file system side-effects, and exit codes.

### JSONL Format

Each line is a JSON object with a `chunks` array:

```json
{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello world."},"finish_reason":"stop"}]}]}
```

For tool calls:

```json
{"chunks":[{"id":"c1","model":"fix","object":"chat.completion.chunk","created":0,"choices":[{"index":0,"delta":{"role":"assistant","content":"Reading file.","tool_calls":[{"index":0,"id":"tc-1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]},"finish_reason":"tool_calls"}]}]}
```

Multi-turn conversations use multiple lines (one per LLM response):

```jsonl
{"chunks":[...tool call response...]}
{"chunks":[...final text response after tool result...]}
```

Optional `delay_ms` field pauses before sending chunks (for timeout testing):

```json
{"chunks":[...],"delay_ms":5000}
```

### Recording Fixtures from Real Sessions

Use `--record-fixture` to capture a real LLM session as a replayable fixture:

```bash
spin exec "read test.txt and summarize it" \
  --record-fixture tests/e2e/fixtures/read_and_summarize.jsonl
```

This wraps the real provider with a recording layer that:
- Captures every streaming chunk from the LLM.
- Writes one JSONL line per `Stream()` call.
- Produces output in the exact format `FixtureProvider` expects.
- Does not affect the session — recording is transparent.

The recorded fixture can then be used in tests:

```go
func TestFixture_ReadAndSummarize(t *testing.T) {
    t.Parallel()

    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    workDir := setupFixtureWorkDir(t, map[string]string{
        "test.txt": "content to summarize",
    })

    r := runFixtureExec(t, "read_and_summarize.jsonl",
        "read test.txt and summarize it",
        withWorkDir(workDir), withAutoApprove())
    assertNoError(t, r)
    assertOutputContains(t, r, "summary")
}
```

### Writing Fixture Tests

All fixture tests live in `tests/e2e/fixture_exec_test.go`. Helper functions are in `tests/e2e/fixture_helpers_test.go`.

Key helpers:

| Helper | Purpose |
|--------|---------|
| `runFixtureExec(t, fixture, prompt, opts...)` | Run `spin exec` with a fixture file |
| `setupFixtureWorkDir(t, files)` | Create temp dir with files |
| `assertOutputContains(t, r, substr)` | Assert output contains text |
| `assertNoError(t, r)` | Assert clean exit |
| `withAutoApprove()` | Pass `--auto-approve` |
| `withWorkDir(dir)` | Set working directory |
| `withTimeout(d)` | Set test timeout |
| `withExecTimeout(val)` | Pass `--timeout` flag |

### Running Fixture Tests

```bash
# Build the test binary first (requires e2e_llm_test tag).
make build-test

# Run all fixture tests.
go test ./tests/e2e/... -v

# Run a specific fixture test.
go test ./tests/e2e/... -v -run TestFixture_SimpleResponse

# Run ACP protocol tests.
go test ./tests/e2e/acp/... -v
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `SPIN_TEST_FIXTURE` | Path to JSONL fixture file (used by test-llm provider) |

## Integration Tests

Integration tests verify wiring between components. They live alongside the code they test:

| Journey | Test File | What It Covers |
|---------|-----------|----------------|
| 001 Parse Agent Skills | `internal/skills/parse_test.go` | Frontmatter + body parse, name/description rules, shipped `.agents/skills`, catalog p99 |
| 002 Discover skill catalog | `internal/skills/discover_test.go`, `internal/agent/prompt/catalog_test.go`, `internal/commands/commands_test.go` | Root order, collision `source`, missing roots, empty catalog, metadata-only Composer, `/skills` + `/help` |
| 003 Activate skill body | `internal/skills/activate_test.go`, `internal/skills/resolve_test.go`, `internal/tools/skill_test.go`, `internal/tools/skill_integration_test.go`, `internal/tui/mapper_skill_test.go`, `internal/conversation/builder_test.go` | `skill`/`load_skill` body+root, unknown-name typed error, `../` reject, one-hop references, `allowed-tools` recorded, SKILL block name+source |
| 004 Parse Agent Plugins | `internal/plugins/load_test.go`, `cmd/spin/plugin_test.go` | Closed `plugin.json` schema, unknown fields reported+ignored, `./` containment, nested skills ignored, `spin plugin validate` |
| 005 Load plugins merge skills | `internal/plugins/discover_test.go`, `internal/plugins/mcpjson_test.go`, `internal/skills/discover_test.go` | Plugin catalog `source=plugin:<name>`, empty `mcpServers`, explicit transports, failing MCP keeps skills, fatal `plugin.json` skips one root |
| 006 Load com.spin.agent hooks | `internal/plugins/extensions_test.go`, `internal/safety/hooks/runner_test.go` | `Event.ScriptName()` discovery, foreign `com.example.client/` ignored, `Contain` on hook paths, plugin cwd, PRE_TOOL_USE fake runner |
| 007 Finish hook runner contract | `internal/safety/hooks/dir_test.go`, `internal/agent/tool/runtime_updated_input_test.go`, `internal/conversation/hooks_dir_test.go`, `cmd/spin/acp_hooks_dir_test.go` | Home expansion of `GlobalDir`, ACP/TUI expanded dir, JSON `updated_input` object replace, empty/non-object leave original args |
| 008 Wire every defined lifecycle hook event | `internal/conversation/parent_lifecycle_integration_test.go`, `internal/agent/tool/runtime_lifecycle_hooks_test.go`, `internal/agent/harness/lifecycle_hooks_test.go`, `internal/conversation/lifecycle_hooks_test.go`, `cmd/spin/tui_quit_test.go` | Parent-side eight events via recording scripts (all ten names registered); `POST_TOOL_USE_FAILURE`; `PRE_COMPACT` before rewrite; `STOP` then `SESSION_END` including Ctrl-C teardown |
| 009 Compact pipeline core | `internal/contexteng/compact/pipeline_test.go` | `Apply` identity/passthrough (R14), fail-safe panic/error (R12), exit-code preserve (R13), `ceil(bytes/4)` ledger (R15), 64 KiB unknown-command p99 |
| 010 Compact command registry | `internal/contexteng/compact/registry_test.go` | Spec table goldens (raw→compact, same exit), R2–R10 filters, R4 `×N` dedup, R8 read levels, R9 go test NDJSON, no network/git |
| 011 Apply compact to shell exec | `internal/tools/shell_command_compact_test.go`, `internal/agent/tool/runtime_compact_test.go`, `internal/contexteng/compact/rewrite_test.go` | Shell post-filter on exec results, R13 exit preserve, R11 argv rewrite, rtk PATH/fallback, `SPIN_COMPACT=0` / enabled false, git status observation |
| 012 Apply compact to built-in read/grep/glob/ls | `internal/tools/read_file_compact_test.go`, `internal/tools/list_directory_compact_test.go`, `internal/tools/file_search_compact_test.go`, `internal/tools/grep_compact_test.go`, `internal/filesearch/grep_test.go` | R8 default minimal + level/config none, R10 tree on list/glob, R2+R3 grep grouping/truncation, same escape hatch, temp fixtures only |
| 013 Compact status chip and operator escape | `internal/ui/status/formatter_test.go`, `internal/ui/status/aggregator_test.go`, `internal/ui/status/manager_test.go`, `internal/commands/commands_test.go`, `cmd/spin/tui_welcome_test.go` | Formatter 0/mid/100% −N% from ledger bytes; off does not claim on; help `SPIN_COMPACT=0` + `compact.enabled`; welcome off line; aggregator last-turn metadata |
| 014 TaskFrame on every parent turn | `internal/agent/frame/frame_test.go`, `internal/agent/prompt/taskframe_test.go`, `internal/conversation/conversation_test.go`, `internal/conversation/builder_test.go`, `cmd/spin/mode_test.go` | Mode→phase 1:1, sources drop bodies, MarshalStable, ComposeTwoPart dynamic-only, compose includes frame without AGENTS.md body, `/mode review` → CurrentFrame phase review |
| 015 Assemble retrieval on the turn path | `internal/agent/harness/retrieval_test.go`, `internal/conversation/retrieval_turn_integration_test.go` | `GetRetrievalPipeline` on `RunTurn`; Assemble fragments in composed-turn messages; nil pipeline no-op; sentinel not duplicated |
| 016 A2A types and local JSON-RPC codec | `internal/protocol/a2a/card_test.go`, `internal/protocol/a2a/codec_test.go`, `internal/protocol/a2a/coverage_test.go` | Card/Task/Message/Part/Artifact camelCase JSON; `message/send`+`tasks/get` over `net.Pipe`; JSON-RPC −32700/−32600/−32601/−32602; A2A −32001/−32002/−32004; no HTTP, no spin binary |
| 017 Local A2A server process | `internal/agent/child/card_test.go`, `internal/agent/child/harness_test.go`, `internal/agent/child/server_test.go`, `internal/agent/child/listen_test.go`, `internal/agent/child/p99_test.go`, `cmd/spin/a2a_test.go`, `internal/agent/subagent/lookup_test.go` | `spin a2a --spec explorer --stdio` card then `message/send`; card skills from Spec allowlist; isolated TaskFrame/history; `unix://` listen; card-received p99 < 200 ms |
| 018 Spawn process children from the parent | `internal/agent/child/spawn_test.go`, `internal/agent/child/spawn_exec_test.go`, `internal/conversation/subagents_test.go` | Parent starts `build/bin/spin a2a` (`pid > 0`); blocking send returns artifact; child crash → Task failed + stderr; spawn/complete events; config `Subagents` extras |
| 019 Subagent hooks and no silent drop | `internal/agent/child/spawn_hooks_test.go`, `internal/agent/child/hooks_inherit_test.go`, `internal/safety/hooks/inherit_test.go`, `internal/agent/subagent/spec_test.go`, `internal/conversation/subagents_test.go` | `SUBAGENT_START` exit 2 → no pid; `SUBAGENT_STOP` on success/failure/crash; plugin extra copied onto child Harness (marker file); missing child extras fail `require`; spawn tool deny-by-default |
| 020 Agent task registry | `internal/agent/tasks/registry_test.go`, `internal/agent/tasks/persist_test.go`, `internal/agent/tasks/deadlock_test.go`, `internal/commands/tasks_test.go`, `internal/tools/agent_tasks_test.go`, `internal/agent/child/handle_test.go`, `internal/agent/subagent/background_test.go` | Non-blocking spawn returns id; `/tasks` id/spec/state; wait until terminal or ctx cancel; cancel then SIGTERM; persist across parent turn; Wait does not acquire spawn semaphore |
| 021 Unified task view | `internal/agent/tasks/view_test.go`, `internal/agent/tasks/format_test.go`, `internal/commands/tasks_test.go`, `internal/tools/process_tools_test.go`, `internal/agent/executor/background_test.go` | `/tasks` `kind=agent\|shell`; mixed 7-hex namespace; shell cancel → Kill (SIGTERM/SIGKILL); `list_processes` / `kill_process` unchanged |
| 023 TUI and ACP surfaces | `internal/tui/mapper_harness_test.go`, `internal/ui/status/formatter_test.go`, `internal/ui/overlay/harness_test.go`, `internal/ui/overlay/hook_veto_test.go`, `internal/protocol/acp/a2a_notifications_test.go`, `internal/commands/agents_test.go`, `cmd/spin/tui_welcome_test.go` | Mapper TASK/SUBAGENT/HOOK/COMPACT; status `tasks N/M`; palette Skills/Tasks/Agents; ACP `kind=a2a` thoughts; `/agents`; hook veto reason visible |
| 0.1 DoomLoop | `internal/agent/harness/doomloop_integration_test.go` | Guard detection, reminder injection, reset |
| 1.1 Pipeline | `internal/agent/executor/adapter_pipeline_test.go` | Pipeline stages, halt behavior |
| 1.2 Blocklist | `internal/agent/executor/stage_blocklist_test.go` | Dangerous pattern blocking |
| 1.3 Hooks | `internal/safety/hooks/lifecycle_test.go` | SESSION_START, USER_PROMPT_SUBMIT, blocking |
| 2.1 Undo | `internal/undo/service_integration_test.go` | Full rollback flow, multi-snapshot |
| 2.3 Session | `internal/session/persistence_integration_test.go` | Index reopen, transcript write/read |
| 3.1 SubAgent | `internal/agent/subagent/manager_integration_test.go` | Builtins, spec lookup, override |
| 3.2 Context | `internal/contexteng/retrieval/pipeline_integration_test.go` | Multi-source assembly, bullet source |
| 3.3 Cache | `internal/llm/cache/persistence_integration_test.go` | Cross-instance persistence, staleness |
| 4.2 Web | `internal/tools/web_fetch_integration_test.go` | HTTP fetch + HTML conversion |
| R1.1 Config Options | `internal/protocol/acp/session_mode_test.go` | SetSessionConfigOption for mode category |
| R1.2 Config Notify | `internal/protocol/acp/session_mode_test.go` | config_option_update notification from both endpoints |
| R1.3 Session Config | `internal/protocol/acp/session_mode_test.go` | ConfigOptions in NewSessionResponse |
| R2.1 Session List | `internal/protocol/acp/session_list_test.go` | UnstableListSessions with filter and pagination |
| R2.2 List Capability | `internal/protocol/acp/session_list_test.go` | SessionCapabilities.List advertisement |
| R3.1 Session Info | `internal/protocol/acp/session_info_test.go`, `title_test.go` | Session title generation and notification |
| R4.1 Tool Kinds | `internal/protocol/acp/notifications_test.go` | ACP tool kind mapping for all 23 tools |
| R1 stdlib Quick Wins | Existing tests in `session/`, `memory/`, `patchapply/` | stdlib replacements: slices.DeleteFunc, time.Compare, slices.SortFunc, strings.Fields |
| R2 genutil | `internal/genutil/genutil_test.go` | TailN, ToSet, AllSame, Mean, Ratio, EnsureMap, Ptr — 28 tests, 100% coverage |
| R3 stringutil | `internal/stringutil/stringutil_test.go` | CollapseWhitespace, CollapseBlankLines, TrimTrailingPerLine, TruncateWithEllipsis, ContainsAnyKeyword, CountLines, StripCodeFence, StripListPrefix — 56 tests, 97.9% coverage |
| R4 chanutil | `internal/chanutil/chanutil_test.go` | TrySend, SendOrCancel, SendWithTimeout, DrainChannel, SleepCtx, CallWithContext — 14 tests, 100% coverage, race-clean |
| R5 digest | `internal/digest/digest_test.go` | SHA256Hex, ShortHash, RandomHexID, NewAtomicIDGenerator — 14 tests, 93.3% coverage, NIST test vectors |
| R6 pathutil | `internal/pathutil/*_test.go` | WalkUpFind, ResolvePath, IsUnsafeWorkDir, ReadFileWithLimit, ReadLastLines — 28 tests, 89.3% coverage |
| R7 wire genutil | Existing tests in `ace/`, `cycle/`, `dbg/`, `llm/ollama/`, `tui/`, `agent/harness/` | Wire genutil.TailN/ToSet/EnsureMap/Ptr, delete 4 dead functions |
| R8 wire stringutil | Existing tests in `safety/`, `patchapply/`, `tools/`, `memory/`, `ace/trajectory/` | Wire stringutil.CollapseWhitespace/CollapseBlankLines/ContainsAnyKeyword, delete 5 dead functions |
| R9 wire chanutil | Existing tests in `events/`, `llm/`, `safety/` (with `-race`) | Wire chanutil.TrySend/SendWithTimeout/SendOrCancel/SleepCtx/CallWithContext, delete 5 dead functions |
| R10 wire digest | Existing tests in `undo/`, `contexteng/summarizer/`, `lsp/`, `agent/executor/` | Wire digest.SHA256Hex/ShortHash/RandomHexID, remove MD5+nolint, delete dead const |
| R11 wire pathutil | Existing tests in `tools/`, `undo/`, `agentsmd/`, `agent/executor/` | Wire pathutil.ResolvePath/IsUnsafeWorkDir/ReadFileWithLimit/ReadLastLines, delete 2 dead functions |
| R12 executil | `internal/executil/executil_test.go` + existing tests in `tools/`, `shell/` | MergeOutputs, EffectiveTimeout — 8 tests, 100% coverage, delete 2 dead functions |
| R13 mathutil generics | `internal/mathutil/vector_test.go` | CosineSimilarity/DotProduct/Magnitude parameterized with Float constraint, 4 float64 tests added, 100% coverage |
| R14 similarity | `internal/similarity/similarity_test.go` + existing tests in `cycle/`, `mcp/` | Levenshtein, JaccardSimilarity, NGrams, MaxByFrequency — 22 tests, 98.6% coverage. Wire + 11 lint fixes |
| R15 search | `internal/search/ranked_test.go` + existing tests in `mcp/`, `filesearch/` | RankedSearch[Item] — 9 tests, 100% coverage. Wire into SearchTools + Match |
| R16 worker pool | `internal/chanutil/pool_test.go` + existing tests in `ace/curator/`, `ace/delta/` | WorkerPool[Job,Result], EffectiveWorkers — 11 tests, 98.4% coverage, race-clean. Delete 11 dead functions/types |
| R17 event emitter | — | DEFERRED: blast radius too high (60+ test files, 20+ prod files) for theoretical benefit |
| R18 JSONL writer | `internal/ds/jsonl_test.go` + existing tests in `session/` | JSONLWriter[T] — 7 tests, race-clean. TranscriptWriter refactored to compose ds.JSONLWriter |
| R19 TTL cache | `internal/ds/cache_test.go` | Cache[K,V] with TTL + count eviction + injectable clock — 9 tests, race-clean |
| R20 chain | `internal/ds/chain_test.go` + existing tests in `tools/fuzzy/` | Chain[Input,Output] with Find/FindAll — 6 tests. fuzzy.Chain refactored to compose ds.Chain |
| R21 llmutil | `internal/llmutil/*_test.go` + existing tests in `llm/ollama/`, `llm/openai/` | ExtractContent + ModelContextWindow — 13 tests, 100% coverage |
| R22 errlist | `internal/apperr/errlist_test.go` + existing tests in `session/` | ErrorList with Add/HasErrors/Err — 5 tests, 100% coverage |
| Refactor | All `pkg/alg/*` tests + `pkg/apperr/`, `pkg/llmutil/`, `pkg/tokenizer/` | Package structure refactoring: internal → pkg/alg/{collections,stringsx,concurrency,hashx,similarity,search,vector,execx,pathx,ds,ds/syncmap} + pkg/{apperr,llmutil,tokenizer} |
| R-REF-1 Inline truncation wrappers | `internal/tools/truncate_test.go` | Remove pure pass-through TruncateHeadTail/TruncateLines, keep TruncateOutput composition |
| R-REF-2 Inline path wrapper | — (tests deleted, covered by `pkg/alg/pathx/`) | Remove resolvePath wrapper, 4 callers use pathx.ResolvePath directly |
| R-REF-3 Inline fuzzy helpers | `internal/tools/fuzzy/*_test.go` | Replace trimLines/countNonBlankLines with stringsx; keep collapseWhitespace (different semantics) |
| R-REF-4 Inline TUI mapper helpers | `internal/tui/*_test.go` | Replace extractString/extractIntValue with direct GetStringOr/GetIntOr calls |
| R-REF-5 Inline filterEnvironment + filesearch aliases | `internal/agent/environment_test.go` | Remove filterEnvironment wrapper, remove filesearch type aliases, use pathx directly |
| R-REF-8 Migrate cycle detection to TailN | `internal/cycle/*_test.go` | Replace rolling-window helpers with collections.TailN/TailNOrAll |
| R-REF-9 Add Filter/Clamp/ValidateAll | `pkg/alg/collections/genutil_test.go` | Filter[T], Clamp[T], ValidateAll[T] — 15 tests, 100% coverage |
| R-REF-11 Generic pattern detection | `pkg/alg/search/pattern_test.go` | DetectRepeat[T], DetectAlternating[T] — 20 tests, 95.6% coverage |
| R-REF-12 Wire cycle detection | `internal/cycle/*_test.go` | Migrate allToolsAreSame/allErrorsAreSame/detectOscillatingTools to search generics |
| R-REF-14 Migrate summarizer cache | `internal/contexteng/summarizer/cache_test.go` | Compose ds.Cache[string, *Result] with LRU — 7 tests pass unchanged |
| R-REF-16 Inline cycle similarity | `internal/cycle/*_test.go` | Delete similarity.go wrapper, inline JaccardSimilarity/ExtractWords calls |
| R-REF-6 Inline capOutput | `pkg/alg/stringsx/extended_test.go` | TruncateWithSuffix — 4 tests |
| R-REF-7 syncmap.GetOrCreateErr | `pkg/alg/ds/syncmap/syncmap_map_test.go` | GetOrCreateErr — 3 tests; conversation/manager migrated |
| R-REF-15 MultiStrategySimilarity | `pkg/alg/similarity/similarity_test.go` | MultiStrategySimilarity + FindSimilarPairs — 8 tests, 99% coverage |
| R-REF-17 Text analysis | `pkg/alg/stringsx/extended_test.go` | NormalizeEscapes + DetectTruncation — 14 tests, 96% coverage |
| R-REF-18 Wire text analysis | `internal/tools/*_test.go` | escape.go → stringsx.NormalizeEscapes, write_file.go → stringsx.DetectTruncation |
| R-REF-19 diff package | `pkg/alg/diff/diff_test.go` | Generate + Parse — 10 tests, 92.9% coverage |
| R-REF-21 fuzzy search | `pkg/alg/search/fuzzy_test.go` | MapNormalizedOffset, FindAllNormalized, MatchesAt, LineOffset — 11 tests |
| R-REF-23 textwidth | `pkg/ui/textwidth/textwidth_test.go` | ExtractGraphemes, TotalWidth, MidEllipsize, GutterWidth — 100% coverage |
| R-REF-25 Registry+StateMachine | `pkg/alg/ds/registry_test.go`, `statemachine_test.go` | Generic Registry[K,V] + StateMachine[S] |
| R-REF-26 storage promotion | `pkg/storage/*_test.go` | FileStore[T], AtomicWriteFile, FileLock promoted to pkg/storage |

## Test Patterns

### Table-Driven Tests

```go
tests := []struct {
    name    string
    input   string
    want    string
}{
    {name: "basic", input: "hello", want: "HELLO"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        got := Transform(tt.input)
        require.Equal(t, tt.want, got)
    })
}
```

### Test Helpers

- Use `t.Helper()` for all helper functions.
- Use `t.Parallel()` for all tests.
- Use `t.TempDir()` for file system tests.
- Use `testify/require` for assertions (fail-fast).
- Prefer interfaces + test doubles over mocking frameworks.

### Journey Comments

Every test file links to its journey spec:

```go
// Journey: specs/journeys/JOURNEY-1.1.md.
```

Structured navigation index tests (`kind` records, R10 path listings, body-escape) live in `internal/nav/index_test.go`, `internal/tools/navigate_test.go`, and `internal/conversation/navigate_test.go` (journey `JOURNEY-022-structured-navigation-index.md`).

## Architecture

```
tests/
  e2e/
    fixtures/           # JSONL fixture files (recorded or hand-crafted)
    fixture_exec_test.go      # Fixture-driven exec tests
    fixture_helpers_test.go   # Test helpers (runFixtureExec, etc.)
    acp/                # ACP protocol E2E tests
  compliance/           # Protocol compliance tests

internal/
  llm/
    recorder/           # Recording provider wrapper
    testprovider/       # Fixture replay provider (build tag: e2e_llm_test)
    mock.go             # Mock provider for unit tests
```
