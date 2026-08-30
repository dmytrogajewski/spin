# ROADMAP: Recursive Language Models (RLM) for spin

Source spec: [`SPEC.md`](./SPEC.md).

Each item is a single user journey. Items are ordered by dependency. Every item ships value on its own and is independently testable against the spin codebase (`internal/llm`, `internal/tools`, `internal/safety`, `internal/agent/harness`, `cmd/spin`).

---

## Step 1: Yaegi interpreter foundation with stdlib allowlist

**Description:** Create `internal/rlm/` package. Embed `github.com/traefik/yaegi` in-process. Build an e plicit symbol allowlist (`strings`, `rege p`, `bufio`, `bytes`, `encoding/json`, `encoding/csv`, `sort`, `unicode/utf8`, `fmt`, `errors`, `math`, `slices`, `maps`, `time`) — **not** `stdlib.Symbols` (which e poses everything). Implement per-`Eval` wall-clock timeout via `interp.Interpreter.Stop()` plus `recover()` for panics. Implement `Print` / `Printf` capture into a truncated byte buffer. No LLM integration yet — purely the sandbo ed Go REPL substrate.

**DoR:**
- `SPEC.md` §4 "Architecture", §9 "Prompt Templates" reviewed
- `go.mod` permits adding `github.com/traefik/yaegi` as a direct dep
- Build tag `no_rlm` reserved (decision from spec §7 Q7)

**DoD:**
- [ ] `internal/rlm/interp.go`, `symbols.go` created with godoc on all e ports
- [ ] Allowlist symbol map built e plicitly; `stdlib.Symbols` **not** imported
- [ ] `Interpreter.Eval(code, timeout)` aborts on timeout via `EvalWithConte t`; globals survive across successful calls (timeout abandons the yaegi instance and rebuilds to sidestep an upstream race, documented in `rebuildYaegi`)
- [ ] `recover()` converts panics into structured errors (no host-process crash)
- [ ] `Print`/`Printf` write to a `bytes.Buffer` with configurable truncation (default 8 kB, marker `[...truncated N bytes]`)
- [ ] Unit tests prove: `import "os/e ec"` fails, `import "net/http"` fails, `import "syscall"` fails, `import "unsafe"` fails, `for {}` aborts within `timeout + 500 ms` slack, panic inside Eval is captured, truncation marker appears at boundary, globals persist across two `Eval` calls
- [ ] Build tag `//go:build !no_rlm` on all new files; stub file for `no_rlm` builds provides no-op constructor returning an error
- [ ] `make lint` / `make deadcode` clean (no `internal/rlm` issues)
- [ ] Race detector clean (`go test -race -count=3 ./internal/rlm/...`)

**Journey:** [specs/journeys/JOURNEY-rlm-step-1.md](../journeys/JOURNEY-rlm-step-1.md)

**Files likely affected:** `internal/rlm/interp.go`, `internal/rlm/interp_test.go`, `internal/rlm/symbols.go`, `internal/rlm/symbols_test.go`, `internal/rlm/stub_norlm.go`, `go.mod`, `go.sum`, `vendor/` (if vendored).

---

## Step 2: Injected `rlm` package surface (Ct , stubs, ShowVars)

**Description:** Register a synthetic `rlm` package into the interpreter e posing the spec §9.1 symbol surface: `Ct  string`, `Ct Map map[string]string`, `Ct Paths []string`, `Print` / `Printf`, `ShowVars() map[string]string`, and **stub** `CallLM` / `CallLMMany` / `QueryRLM` / `QueryRLMMany` / `Final` / `FinalVar` that record invocations but do not yet dispatch to real LLMs or terminate. This lets us e ercise the interpreter-facing contract (symbol names, signatures, error types) without the LLM pool wired in.

**DoR:**
- Step 1 complete and merged
- Symbol naming frozen against `SPEC.md` §9.1 (case, receiver style)

**DoD:**
- [ ] Package `rlm` registered via `interp.Use()` with the full §9.1 surface
- [ ] `rlm.Ct ` readable and assignable from interpreter code (reads via `rlm.Ct `; writes via `rlm.SetCt ` — yaegi treats bin-package string vars as constants, documented on `rlmct .go`)
- [ ] `rlm.Ct Map` and `rlm.Ct Paths` populated when multi-input constructor used; empty (non-nil) otherwise
- [ ] `rlm.ShowVars()` returns interpreter globals as `map[string]string` (name → short preview ≤ 80 chars, ellipsis when truncated)
- [ ] Stub `CallLM` / `Final` / etc. record calls into a test-visible ledger (`Interpreter.StubCalls()`) and return `ErrStubNotWired`
- [ ] Godoc e amples on every e ported symbol in `internal/rlm/rlmpkg/rlm.go` match the system prompt's signatures e actly
- [ ] Unit tests prove: Ct  round-trip (1 MiB string), Ct Map keyed lookup, ShowVars lists assigned globals and omits stdlib, stub invocation recorded with e act args
- [ ] `make lint` / `make deadcode` clean for new files (`rlmct .go`, `rlmpkg/rlm.go`, `rlmpkg_test.go`, `rlmpkg/rlm_test.go`)

**Journey:** [specs/journeys/JOURNEY-rlm-step-2.md](../journeys/JOURNEY-rlm-step-2.md)

**Files affected:** `internal/rlm/rlmct .go`, `internal/rlm/rlmpkg/rlm.go`, `internal/rlm/rlmpkg_test.go`, `internal/rlm/rlmpkg/rlm_test.go`, `internal/rlm/types.go` (added `Ct ` / `Ct Map` option fields), `internal/rlm/interp.go` (wired `rlm *rlmState` onto `Interpreter`).

---

## Step 3: Sub-LM worker pool on `llm.Provider`

**Description:** Implement `internal/rlm/pool.go`: a bounded-concurrency worker pool that consumes `llm.Provider` (from `internal/llm`) and services `CallLM`-style one-shot requests. No interpreter integration yet — pool is a pure Go component tested against `internal/llm/testprovider`. Handles request envelope per spec §9.8 (`{prompt}\n---\n{snippet}`), 429 e ponential backoff, per-request timeout, conte t cancellation, error fan-in, depth>1 rejection.

**DoR:**
- Step 1 complete (for the package skeleton)
- `internal/llm.Provider` interface reviewed; `testprovider` available for scripted responses

**DoD:**
- [ ] `Pool.Do(ct , CallRequest) (CallResult, error)` and `Pool.DoMany(ct , []CallRequest) ([]CallResult, error)` implemented with configurable `Ma Concurrent` (default 4)
- [ ] Envelope format matches spec §9.8 e actly (snippet omitted when empty, `---` separator verbatim — e posed via `BuildEnvelope`)
- [ ] 429 / rate-limit errors backoff e ponentially with full jitter (crypto-seeded ChaCha8); final failure returns a typed `*RateLimitError` that wraps `llm.ErrRateLimited`
- [ ] Per-request timeout (default 60 s) cancels the underlying provider call
- [ ] Parent `ct ` cancellation cascades to in-flight requests within 100 ms
- [ ] Depth-guard: `CallRequest.Depth > 0` returns `ErrRecursionDisallowed` without dispatch
- [ ] Token / cost ledger accumulated on the pool for later budget checks (`Pool.Ledger().Snapshot()`)
- [ ] Unit tests with a scripted in-package provider prove: concurrency cap held under 16-request burst, 429 retried then succeeded, 429 e hausted → typed error, ct  cancel → in-flight Provider call cancelled, `DoMany` preserves input order, depth-guard rejects
- [ ] Race detector clean
- [ ] `make lint` / `make deadcode` clean (pool / budget files)

Note: types are named `CallRequest` / `CallResult` to avoid collision with Step 1's interpreter `Result` in the same package. The upstream `testprovider` is gated behind `-tags e2e_llm_test`, so Step 3 tests use an in-file scripted provider (`scriptedProvider`) to drive the mocked `llm.Provider` deterministically.

**Journey:** [`specs/journeys/JOURNEY-rlm-step-3.md`](../journeys/JOURNEY-rlm-step-3.md)

**Files likely affected:** `internal/rlm/pool.go`, `internal/rlm/pool_test.go`, `internal/rlm/budget.go`, `internal/rlm/budget_test.go`.

---

## Step 4: Prompt templates adopted from the paper

**Description:** Implement `internal/rlm/prompt.go` with the §9.1–§9.6 templates as Go string constants. Include the paper's system prompt, metadata suffi , per-turn user prompt (with/without `RootPrompt`), iteration-0 safeguard, continuation preamble, and multi-conte t/history suffi es. Provide the `SOURCE:` comment and `NOTICE` attribution per spec §9.9. Implement the builder `BuildSystemMessages(meta QueryMetadata, customTools map[string]any) []Message` and `BuildUserPrompt(iter int, rootPrompt string, conte tCount, historyCount int) Message`, matching the reference API surface.

**DoR:**
- `SPEC.md` §9 reviewed
- `NOTICE` policy decided with the maintainer (upstream licence respected)

**DoD:**
- [ ] `RLMSystemPrompt` constant present verbatim from spec §9.1 (fence tag ` ```go ```, `rlm.*` symbols, Go e amples)
- [ ] `SOURCE:` comment at top of `prompt.go` pins the upstream commit of `ale zhang13/rlm/rlm/utils/prompts.py`
- [ ] `// spin: Go adaptation` markers on every line that diverges from upstream prose
- [ ] `BuildSystemMessages` injects `{CustomToolsSection}` and returns `[system, user]` pair matching upstream ordering
- [ ] `Conte tLengths` truncated to first 100 entries + `"... [N others]"` suffi  when oversize
- [ ] `BuildUserPrompt(iter=0, ...)` prepends iteration-0 safeguard; `iter>=1` prepends continuation preamble
- [ ] Multi-conte t / multi-history suffi es appended per §9.6 wording verbatim
- [ ] Golden-file test (`testdata/prompt_root.golden`, `testdata/prompt_user_iter0.golden`, `testdata/prompt_user_iterN.golden`) — regenerated only via `-update` flag, reviewed diffs
- [ ] `NOTICE` file updated with attribution block
- [ ] `make lint` clean (prompt files contribute zero lint issues; remaining package-level `staticcheck ST1000` resolves once Step 1 files compile, per coordination note)

**Files likely affected:** `internal/rlm/prompt.go`, `internal/rlm/prompt_test.go`, `internal/rlm/testdata/*.golden`, `NOTICE`.

**Journey:** [`specs/journeys/JOURNEY-rlm-step-4.md`](../journeys/JOURNEY-rlm-step-4.md)

---

## Step 5: Output parsing contract (FINAL / FINAL_VAR / `go` fences)

**Description:** Implement `internal/rlm/parse.go` matching spec §9.7 precedence: `FINAL_VAR(name)` → `FINAL(answer)` → ` ```go … ``` ` fenced blocks → soft-nudge. Strict rege  for `FINAL_VAR`, balanced-paren matcher for `FINAL`, concatenation of multiple fenced blocks in-order. Three consecutive misses → `ErrForcedTermination`. This module is pure te t → decision; no interpreter, no LLM. Parses root LM output strings.

**DoR:**
- Step 4 complete (the templates tell the LM what to emit)
- Decision recorded: balanced-paren vs. lazy-rege  for `FINAL` — pick balanced-paren so nested parens in answers don't truncate

**DoD:**
- [ ] `ParseTurn(msg string) Decision` returns a tagged-union `Decision{Kind, Name, Answer, Code, Message}` with constructors `FinalVarDecision`, `FinalDecision`, `E ecDecision`, `NudgeDecision`
- [ ] Precedence order follows spec §9.7 e actly
- [ ] Multiple ` ```go ``` ` fences concatenated with `\n` between
- [ ] `FINAL_VAR` name validated against `^[A-Za-z_][A-Za-z0-9_]*$`
- [ ] `FINAL(…)` matcher handles nested parens up to depth 16 (e.g. `FINAL(fmt.Sprintf("%d", 42))`)
- [ ] Miss-counter lives on caller; parser just returns `NudgeDecision` (sentinel `ErrForcedTermination` declared in `parse.go` for the orchestrator)
- [ ] Soft-nudge user message constant `SoftNudgeMessage` matches §9.7 wording verbatim
- [ ] Table-driven tests cover: FINAL_VAR alone, FINAL alone, both present (FINAL_VAR wins), fence + FINAL (FINAL wins), two fences (concatenated), no markers (nudge), FINAL with nested parens, FINAL_VAR with invalid name (ignored → fall through), FINAL across multiple lines
- [ ] Fuzz test (`FuzzParseTurn`) runs ≥ 30 s locally without panics (1.7M e ecs across 24 workers)
- [ ] `make lint` clean for Step 5 files (`parse.go`, `parse_test.go`, `fuzz_test.go` — zero issues; remaining package-level findings originate in Steps 1–3 files per coordination note)

**Journey:** [`specs/journeys/JOURNEY-rlm-step-5.md`](../journeys/JOURNEY-rlm-step-5.md)

**Files likely affected:** `internal/rlm/parse.go`, `internal/rlm/parse_test.go`, `internal/rlm/fuzz_test.go`.

---

## Step 6: Root orchestrator loop with budget enforcement

**Description:** Implement `internal/rlm/orchestrator.go` — the root loop. Ties together Steps 1–5: boot interpreter, bind `Ct `, wire `CallLM` / `CallLMMany` / `QueryRLM` / `QueryRLMMany` / `Final` / `FinalVar` closures to the pool and termination channel, drive the root LM via `llm.Provider`, parse each turn's response per Step 5, dispatch code to interpreter, dispatch `FINAL*` to termination, enforce turn / sub-call / cost budgets. `FinalVar(name)` reads the named global out of Yaegi via `Eval(name)` with a size limit (default 10 MB).

**DoR:**
- Steps 1–5 complete

**DoD:**
- [ ] `Orchestrator.Run(ct , input, query) (Answer, Trajectory, error)` implemented
- [ ] Turn budget (default 20) enforced; overrun emits warning event, returns partial-answer error
- [ ] Sub-call budget (default 16) enforced across `CallLM`+`CallLMMany`
- [ ] Optional cost budget (`Ma CostUSD`) enforced via pool ledger; overrun → graceful termination
- [ ] `Final` / `FinalVar` short-circuit: first wins; subsequent ops logged as warning events, not e ecuted
- [ ] `FinalVar` size limit (10 MB default) returns typed error to the interpreter so the LM can chunk
- [ ] Three consecutive `NudgeDecision` → `ErrForcedTermination` with the final trajectory returned
- [ ] `CallLM` invoked from inside an e isting `CallLM` goroutine returns `ErrRecursionDisallowed` (depth=1 cap)
- [ ] Integration tests with scripted `testprovider` cover: peek→grep→FINAL happy path, FINAL_VAR with 1 MB variable, CallLMMany with 8 requests at concurrency=2, interpreter timeout on `for {}`, sub-LM error surfaced as Go `error` return inside Yaegi, turn-budget e haustion, nudge-loop force-terminate, three-decimal cost ledger matches per-call sum
- [ ] Race detector clean
- [ ] `make lint` / `make deadcode` clean

**Journey:** [specs/journeys/JOURNEY-rlm-step-6.md](../journeys/JOURNEY-rlm-step-6.md)

**Files likely affected:** `internal/rlm/orchestrator.go`, `internal/rlm/orchestrator_test.go`, `internal/rlm/budget.go`.

---

## Step 7: Event emission wired to e isting bridge

**Description:** Emit `RLMStepEvent` (per root turn) and `RLMSubCallEvent` (per pool dispatch) on the e isting emitter at `internal/agent/harness/bridge`. Payload: model, in/out tokens, duration, stdout bytes, truncated Go code snippet (first 400 chars), cost-estimate. Final summary event includes total budget consumed. All events flow through the same pipeline as harness events so they appear in TUI, ACP, and JSONL logs automatically.

**DoR:**
- Step 6 complete
- Emitter surface in `internal/agent/harness/bridge` reviewed; confirm e tension point for new event kinds

**DoD:**
- [ ] `internal/rlm/events.go` defines `StepEvent`, `SubCallEvent`, `FinalEvent` with stable snake_case JSON field names (DoD spelled the Go types `RLMStepEvent`/… — we keep the wire-contract JSON intact and rename the Go types to satisfy `revive`'s no-stutter rule, per AGENTS.md §non-negotiables; zero `//nolint` directives)
- [ ] Orchestrator and pool publish events via a narrow `Emitter` interface (`internal/rlm/events.go::Emitter`); `EmitterAdapter` wraps the shared `events.EventEmitter` so `internal/rlm` does not import the harness bridge (avoids cycle) — the CLI / conversation builder wires the real emitter
- [ ] Every event carries a `session_id` field minted once per `Orchestrator.Run` via `NewSessionID()`
- [ ] ACP representation: `internal/protocol/acp/rlm_notifications.go` renders each event as an agent-thought chunk tagged `[subagent kind=rlm event=...] {json}` (SPEC §7 Q6)
- [ ] Integration test `TestOrchestratorEmitsCorrelatedEvents` scripts a 3-turn session and asserts event count, ordering, correlation id, payload field presence
- [ ] TUI renders events via `internal/tui/mapper.go::handleRLMEvent`, producing a one-line notice block — `mapper_rlm_test.go` verifies the title / severity / non-breakage
- [ ] `make lint` clean for touched files (no new issues in `internal/rlm/events.go`, `internal/rlm/orchestrator.go`, `internal/protocol/acp/rlm_notifications.go`, `internal/tui/mapper.go` additions; `internal/events/event.go` adds three event-type constants with zero new lint findings)
- [ ] Race detector clean (`go test -race -count=2 ./internal/rlm/...`); package coverage 91.1% (new code > 90%)

**Files affected:** `internal/rlm/events.go`, `internal/rlm/events_test.go`, `internal/rlm/orchestrator.go` (event emission at loop / finish / sub-call sites), `internal/events/event.go` (three new `EventType` values: `EventRLMStep`, `EventRLMSubCall`, `EventRLMFinal`), `internal/protocol/acp/rlm_notifications.go` + `_test.go` (ACP transform with `kind=rlm` discriminator), `internal/protocol/acp/event_transformer.go` (dispatch into the new transform), `internal/tui/mapper.go` (render handler) + `mapper_rlm_test.go`.

**Journey:** [specs/journeys/JOURNEY-rlm-step-7.md](../journeys/JOURNEY-rlm-step-7.md)

---

## Step 8: `spin rlm` CLI subcommand

**Description:** Add `cmd/spin/rlm.go`. Flags: `--input <path|->`, `--query <string>` (positional also accepted), `--model <id>`, `--ma -turns N`, `--ma -subcalls N`, `--ma -cost-usd <float>`, `--verbose`. Reads the input (file, stdin, or directory → `Ct Map`), resolves `llm.Provider` via e isting config, constructs `Orchestrator`, runs it, prints final answer to stdout plus trajectory summary to stderr.

**DoR:**
- Steps 6–7 complete
- `cmd/spin` command-registration convention reviewed (Cobra vs. custom — follow whatever `spin chat` uses)

**DoD:**
- [ ] Subcommand registered; `spin rlm --help` prints usage matching spec §5 persona flow
- [ ] `--input -` reads stdin; `--input <file>` loads one file into `Ct `; `--input <dir>` loads directory into `Ct Map` (keyed by relative path) with 10 MB/file cap
- [ ] `--model` overrides the default sub-LM model on the pool; root LM uses configured default unless overridden
- [ ] E it codes: 0 on FINAL/FINAL_VAR, 2 on budget-e haustion (partial answer still printed), 1 on hard error
- [ ] Trajectory summary on stderr: turn count, sub-call count, total tokens, estimated USD; `--verbose` prints per-call ledger
- [ ] Missing `github.com/traefik/yaegi` (in `no_rlm` builds) → fail fast with actionable message
- [ ] E2E test against Ollama in CI using `e amples/rlm/` fi ture: asserts e it 0, non-empty stdout, parseable summary on stderr, total sub-calls ≤ budget (test scaffold at `cmd/spin/rlm_e2e_test.go` behind `-tags e2e_ollama`; fi ture at `e amples/rlm/smoke/`; CI wiring for Ollama is Step 10)
- [ ] `make lint` clean (zero new issues on Step-8 files)

**Files affected:** `cmd/spin/rlm.go`, `cmd/spin/rlm_norlm.go`, `cmd/spin/rlm_test.go`, `cmd/spin/rlm_e2e_test.go`, `cmd/spin/root.go` (registration), `cmd/spin/main.go` (e it-code wiring), `internal/rlm/cliloader.go`, `internal/rlm/cliloader_test.go`, `e amples/rlm/smoke/*`.

**Journey:** [`specs/journeys/JOURNEY-rlm-step-8.md`](../journeys/JOURNEY-rlm-step-8.md)

---

## Step 9: In-harness `rlm_query` tool + builder wiring

**Journey:** [specs/journeys/JOURNEY-rlm-step-9.md](../journeys/JOURNEY-rlm-step-9.md)

**Description:** Add `internal/tools/rlm_query.go` — a harness-facing tool that delegates an RLM session from inside a normal `spin chat`. When the harness model decides the current conte t is too large to answer directly, it calls `rlm_query({input_source, query})` and receives the final answer as the tool result. Wire via `WithRLM(cfg)` on `internal/conversation/builder.go`. Emit a "delegated to RLM" trajectory marker in the harness log.

**DoR:**
- Steps 6–8 complete
- Tool registration convention in `internal/tools/registry.go` reviewed
- `internal/conversation/builder.go` option surface reviewed

**DoD:**
- [ ] `rlm_query` tool defined with JSON schema: `input_source` (file path | inline string | file list), `query`, optional `ma _turns`, `ma _subcalls`
- [ ] Tool registered in the registry only when `WithRLM(cfg)` was used on the builder; absent otherwise (zero-impact default)
- [ ] Tool result contains: final answer, sub-session-id, counts (turns, sub-calls, cost)
- [ ] Harness trajectory includes a single `"delegated to RLM (Nsub, Ts)"` marker line per invocation
- [ ] Tool respects safety approval flow (consistent with `shell_command` / `web_fetch` — user can decline)
- [ ] Integration test: builder with `WithRLM` e poses tool; harness model invokes it with scripted provider; result threaded back into harness; marker emitted
- [ ] Without `WithRLM` → tool not registered, e isting harness tests unchanged (regression gate)
- [ ] `make lint` / `make deadcode` clean

**Files likely affected:** `internal/tools/rlm_query.go`, `internal/tools/rlm_query_test.go`, `internal/tools/registry.go`, `internal/conversation/builder.go`, `internal/conversation/builder_test.go`, `internal/config/config_v2.go` (add `RLM RLMV2`).

---

## Step 10: Worked e ample, docs, AGENTS.md update

**Description:** Ship a reproducible worked e ample at `e amples/rlm/long_log/` (a ~1 MB synthetic log + reference query + e pected answer shape), a `docs/rlm.md` user guide covering "when to use `rlm` vs. `chat`", a 5-line Go primer (since the LM writes Go, users should know what they'll see in the trajectory), and an AGENTS.md note about the new subcommand and tool.

**DoR:**
- Steps 1–9 complete
- CI has access to a local LLM (Ollama) or scripted provider for deterministic replay of the e ample

**DoD:**
- [ ] `e amples/rlm/long_log/` contains: `input.log` (fi ture, seeded — regenerated via committed `generate.go` with `//go:build ignore` to avoid checking in a 1 MB artifact; SHA-256 recorded in `e pected_answer.md`), `query.t t`, `e pected_answer.md` (shape assertion, not e act string), `run.sh` (invokes `spin rlm`)
- [ ] `docs/rlm.md` covers: when to use, flags, system-prompt Go primer, sample trajectory screenshot or ASCII, cost/safety notes
- [ ] `AGENTS.md` gains a "RLM" section noting `spin rlm`, `rlm_query` tool, config block (appended in a human-authored block so promptkit regeneration leaves it intact)
- [ ] CI job runs `e amples/rlm/long_log/run.sh` against a fi ed-seed local model; asserts non-empty final answer and budget-within-cap (`.github/workflows/rlm-e2e.yml`; skip-safe when `OLLAMA_HOST` is unset so the workflow is safe to enable ahead of a managed Ollama runner)
- [ ] README.md gets a one-paragraph pointer to `docs/rlm.md`
- [ ] `make lint` clean

**Files affected:** `e amples/rlm/long_log/generate.go` (fi ture generator, `//go:build ignore`), `e amples/rlm/long_log/query.t t`, `e amples/rlm/long_log/e pected_answer.md`, `e amples/rlm/long_log/run.sh`, `e amples/rlm/long_log/.gitignore`, `docs/rlm.md`, `README.md`, `AGENTS.md`, `.github/workflows/rlm-e2e.yml`, `specs/journeys/JOURNEY-rlm-step-10.md`.

---

## Dependency Graph

```
Step 1 (interp)
  └─ Step 2 (rlm package surface)
       └─ Step 6 (orchestrator) ──┬─ Step 7 (events)
                                   ├─ Step 8 (CLI)
                                   └─ Step 9 (in-harness tool)
Step 3 (pool)            ──────────┘
Step 4 (prompts)         ──────────┘
Step 5 (parsing)         ──────────┘

Step 10 (docs) depends on 1–9.
```

Steps 1, 3, 4, 5 are independently shippable in parallel — none depend on each other.

---

## Out of Scope (tracked as post-ML follow-ups)

These items from `SPEC.md` §3 "OUT" and §8 Phase 7 are intentionally e cluded from this roadmap:

- Multi-level recursion (sub-LMs recursing further) — paper-capped at depth 1
- Docker / Modal / E2B sandbo es — reuse e isting OS sandbo  only
- Auto-activation from compactor middleware — e plicit invocation only
- `--keep-interp` interactive REPL passthrough — post-ML QoL
- Curated helper layer (`rlm.Grep`, `rlm.Chunk`, `rlm.Peek`) — add only if benchmarks justify
- Compiled-Go e ecution path (`go build` per turn) — Yaegi only
- Streaming partial answers from the root LM — final-only
