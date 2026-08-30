# SPEC: Recursive Language Models (RLM) for spin

## 1. Summary

Implement the Recursive Language Models technique (Zhang, Kraska, Khattab — arXiv:2512.24601) in spin: a "root" LLM never sees the long context directly; instead, the context is held as a Go variable inside an embedded Go interpreter (Yaegi) that the root LLM drives via Go code execution, including recursive sub-LM calls over chunks. Target users are spin operators dealing with prompts that exceed the model's effective context window (large repos, long logs, multi-document Q&A) where vanilla long-context degrades sharply. Success looks like spin handling inputs at least 10× the model's native window with quality on long-context tasks meeting or beating the baseline of dumping the full context into a single call — with zero non-Go runtime dependencies.

## 2. Background & Research

### Market Context

| Product | Approach | Takeaway for spin |
|---|---|---|
| **Anthropic Claude `code_execution` + Files API** | Tool-call sandbox where the model writes Python that reads/searches uploaded files; the file content is *not* in the prompt. | Validates the "context-as-variable + REPL" pattern at production scale. Confirms users accept latency for quality. |
| **OpenAI Assistants Code Interpreter / Responses API `python` tool** | Persistent Python session with file inputs; model decides what to load into context. | Same pattern; persistent kernel matters — destroying state per turn is too expensive. |
| **LangChain / LlamaIndex agentic retrieval** | RAG with re-rank, hierarchical summarisation, multi-hop question decomposition. | RLM subsumes RAG: the root LM can grep/embed/chunk *programmatically* per query rather than using a fixed pipeline. spin should not bolt RLM next to RAG — it replaces it. |
| **Cursor / Cline "agent" repo search** | Recursive grep/find tools with summarising sub-agents. | Closest existing pattern in coding agents. They lack a *programmable* environment; everything is a fixed tool. RLM is the generalisation. |
| **DSPy / RAFT / Chain-of-Agents (Google, NeurIPS '24)** | Pipeline-style decomposition of long context across worker LMs. | Static topology. RLM lets the root LM choose topology per query. |

User complaints about competitors: opaque trajectories (hard to debug what the sub-agent did), runaway costs from poor recursion bounds, sandbox cold-start latency, and brittle JSON/tool-call schemas when the model wants to do something the tool didn't anticipate.

### Technical Context

The paper's reference implementation (`github.com/alexzhang13/rlm`) is structured as:

- **`RLMChatCompletion`** wraps a base LLM client. The user's original long prompt is *not* sent to the model.
- **Python REPL sandbox** (`local` / `docker` / `modal` / `prime` / `daytona` / `e2b`) is pre-loaded with the context as a Python variable (e.g. `ctx`, or as files on disk). Backed by an `exec`-style notebook session that retains state across turns.
- **Root LM loop**: receives the *query only*, plus a system prompt describing available tools (peek, grep, slice, `call_lm(prompt, snippet)`), then emits Python code blocks. The harness executes them, returns truncated stdout/stderr, repeats.
- **Termination**: root LM emits `FINAL(answer)` (literal string) or `FINAL_VAR(name)` (read from a REPL variable so large structured outputs don't have to round-trip through the LM's output channel).
- **Sub-LM calls**: a `call_lm(prompt: str, model: str | None) -> str` builtin in the REPL invokes a flat (non-recursive in v1) LLM call over a snippet. Depth is capped at 1 — sub-LMs cannot themselves recurse, which keeps cost bounded and trajectories debuggable.

Emergent strategies the paper highlights: **peek** (read first/last K tokens), **grep** (regex over context variable), **partition+map** (chunk the variable, parallel `call_lm` over each, aggregate), **summarise** (sub-LM on each chunk, then root reasons over summaries).

Cost: paper reports comparable wall-clock cost vs. vanilla long-context, because savings from not paying for a giant input prompt offset the multiple smaller calls. Quality: +28.3% avg on RLM-Qwen3-8B over the base model on four long-context benchmarks.

### Deep Dives

- **REPL-as-tool over typed-tool zoo**: the paper's strongest claim is *generality*. A typed-tool agent must anticipate every operation (grep, head, tail, json.loads, …); a REPL agent gets all of the host language's stdlib for free. Same insight as Voyager (Wang et al., 2023) and CodeAct (Wang et al., ICML 2024): code is a more expressive action space than JSON tool calls.
- **Persistent kernel matters**: the paper assumes intermediate variables survive across REPL turns. Without persistence, the root LM has to recompute or re-marshal every turn — token cost explodes.
- **Why Go (Yaegi) instead of Python**:
  - **Same-process, same-binary**: no `python3` runtime dep, no subprocess pipe, no JSON marshalling. spin stays a single static Go binary.
  - **Stdlib parity for the operations that matter**: `strings`, `regexp`, `bufio`, `encoding/json`, `sort`, `unicode/utf8` cover the entire emergent toolkit (peek, grep, chunk, map, summarise). The paper's Python sessions don't actually use anything richer.
  - **LM fluency**: frontier LMs write idiomatic Go for string/regex/slice work. Less fluent than Python on numpy/pandas-style data, but RLM workloads are text manipulation, not numerics.
  - **Sandbox by allowlist**: Yaegi imports are explicit — we register only the packages we want exposed. No `os/exec`, no `net`, no `syscall`. This is a stronger isolation primitive than Python's "trust the OS sandbox" model and works identically on Linux and macOS.
  - **Cold start ~5 ms** (interpreter init + symbol registration) vs. ~50 ms for a Python subprocess.
- **`FINAL_VAR` is non-obvious but important**: when the answer is itself large (a constructed list, a generated patch, a JSON blob), forcing it through the LM's output channel re-introduces the context-window problem in reverse. Returning by interpreter variable name is the escape hatch.

## 3. Proposal

### Approach

Add an optional **RLM execution mode** that lives alongside spin's existing harness loop, not inside it. When the user invokes a query against an oversized context (file, directory, log, URL set) the input is *not* fed into the harness conversation — it is stored as a `Ctx` variable in an embedded **Yaegi** Go interpreter, and a dedicated **RLM root loop** runs. The root loop uses a single REPL tool (`rlm_exec`) and a single termination tool (`rlm_final`); inside the interpreter, an injected `CallLM(prompt, snippet)` Go function spawns flat (depth=1) sub-LM calls via spin's existing `llm.Provider`.

Wiring:

```
User query + large input
        │
        ▼
┌────────────────────────────────┐
│  RLM Orchestrator (Go)         │
│  - Boots Yaegi interpreter     │
│  - Binds Ctx variable          │
│  - Drives root loop            │
└──────────┬─────────────────────┘
           │
           ▼
┌────────────────────────────────┐
│  Root LM (via llm.Provider)    │  ◀── system prompt: Go REPL guide
│  - sees query only             │
│  - emits Go via rlm_exec       │
│  - terminates via rlm_final    │
└──────────┬─────────────────────┘
           │ rlm_exec(code)
           ▼
┌────────────────────────────────┐
│  Yaegi interpreter (in-proc)   │
│  - persistent globals          │
│  - Ctx, helpers, CallLM()      │
│  - import allowlist            │
│  - stdout truncated to N kB    │
└──────────┬─────────────────────┘
           │ CallLM(prompt, snippet)
           ▼
┌────────────────────────────────┐
│  Sub-LM (llm.Provider, flat)   │
│  - one-shot call, no tools     │
│  - returns string              │
└────────────────────────────────┘
```

This is purposefully *parallel to* — not entangled with — the existing harness loop. The harness is for general agentic coding tasks; RLM is for long-context Q&A and transformation. They share infrastructure (`llm.Provider`, event emitter) but not control flow.

### Key Decisions

| Decision | Choice | Reasoning | Alternatives rejected |
|---|---|---|---|
| **REPL language** | Go, executed via embedded **Yaegi** (`github.com/traefik/yaegi`) interpreter, in-process. | Single-binary deployment, zero runtime deps, stdlib parity for text-processing workloads (`strings`/`regexp`/`bufio`/`encoding/json`), import allowlist gives stronger sandboxing than OS-level. | Python subprocess (adds runtime dep, IPC overhead, weaker default sandbox), Starlark (no regex/json, not Go syntax — loses the "familiar stdlib" win), embedded JS (V8 CGo nightmare), shelling to `go run` (compile latency per turn, no persistence). |
| **Sandbox model** | Yaegi `Use()` allowlist of stdlib symbols only: `strings`, `regexp`, `bufio`, `bytes`, `encoding/json`, `encoding/csv`, `sort`, `unicode/utf8`, `fmt`, `errors`, `math`, `slices`, `maps`, `time` (read-only). **Not exposed**: `os/exec`, `os` (except `os.Stdout`/`Stderr` proxies), `net*`, `syscall`, `runtime`, `unsafe`, `plugin`, `reflect`. | Allowlist is the only correct posture for an interpreter — denylist leaks. spin's existing OS sandbox (Landlock/Seatbelt) wraps the whole spin process as a defence-in-depth backstop. | Open imports (catastrophic — `os/exec.Command` would escape trivially), Yaegi-in-subprocess + OS sandbox (loses the in-process win without strengthening isolation meaningfully). |
| **Recursion depth** | Cap at 1 (root + sub-LMs). Sub-LMs receive no tools and no interpreter. | Matches paper. Bounds cost. Keeps trajectory readable. Multi-level can come later if a use case demands it. | Unbounded recursion (cost explosion, the paper itself avoids it). |
| **Sub-LM concurrency** | Goroutine-pooled, default max 4 in flight, configurable. | The paper relies on `partition+map` for speedup. Serial sub-calls would erase the latency win. Native fit for Go. | Serial only (too slow on real long-context), unbounded (rate-limit blowups). |
| **Termination** | Two interpreter builtins: `Final(value string)` and `FinalVar(name string)`. Also accept tool-call form `rlm_final({answer})` from the LM directly. | `FinalVar` is the paper's escape hatch for large structured outputs (reads the named global out of the interpreter). Dual surface (builtin + tool call) lets the model pick whichever it remembers. | Single termination path (loses the large-output escape hatch). |
| **Activation** | Explicit: a new `rlm` subcommand (`spin rlm --input file.txt "query"`) and a new `rlm_query` tool exposed inside the harness for the main agent to delegate to. **Not** auto-triggered on context pressure. | Auto-trigger conflates two distinct workflows (coding-agent vs. long-context Q&A). Explicit is cheaper to debug and ship. Once stable, a compactor middleware can opt to invoke it. | Always-on wrapper around the harness (entangles the loops, hard to disable for debugging). |
| **Trajectory storage** | Reuse the existing event emitter (`internal/agent/harness/bridge`); emit `RLMStepEvent` per root turn and `RLMSubCallEvent` per `CallLM`. | Already wired to ACP and TUI. Makes the trajectory inspectable in the same UI as everything else. | Custom JSONL log (forks observability story). |

### ML (Minimum Loveable)

**IN:**

- `spin rlm` subcommand: `--input <path|->`, `--query <string>`, `--model <id>`, `--max-turns N`, `--max-subcalls N`, prints final answer + trajectory summary.
- Embedded Yaegi interpreter (one per RLM session, discarded on exit) with stdlib allowlist.
- Two LM-facing tools: `rlm_exec(code: string)` and `rlm_final(answer: string)`.
- Interpreter-injected symbols (package `rlm`): `Ctx string` (or `CtxMap map[string]string` for multi-input), `CallLM(prompt, snippet string, opts ...Option) (string, error)`, `CallLMMany(reqs []Request) ([]Result, error)`, `Final(value string)`, `FinalVar(name string)`, `Print(v ...any)` / `Printf(format string, v ...any)` (capture to truncated buffer).
- Depth=1 recursion (root + flat sub-LMs).
- Bounded `CallLM` concurrency via worker pool.
- Cost & turn budget enforcement with explicit termination on overrun.
- Per-`exec` wall-clock timeout (default 30 s) using a goroutine + interpreter cancellation channel.
- Trajectory events on the existing emitter; visible in TUI.
- Reasonable defaults: max 20 root turns, max 16 sub-calls total, 8 kB stdout truncation, 4 concurrent sub-calls.
- One worked example in `examples/rlm/` (long log triage).

**OUT:**

- Multi-level recursion (sub-LMs cannot recurse).
- Compiled-Go execution (we run Yaegi only — no `go build` per turn).
- Auto-activation from compactor (explicit invocation only).
- Native RLM-trained model integration (`RLM-Qwen3-8B` from the paper) — we use whatever `llm.Provider` is configured.
- Cross-session interpreter persistence (each `spin rlm` run starts fresh).
- Streaming partial answers from the root LM (final-only).
- CGo / unsafe / network packages exposed to the interpreter.

### Anti-Goals

- **We will not replace the harness loop.** The harness handles coding agency; RLM handles long-context Q&A. Conflating them produces an unmaintainable god-loop.
- **We will not auto-trigger RLM on context pressure in v1.** Compactor + summariser already handle that path. Auto-routing to RLM is a decision-theoretic problem we are not ready to solve.
- **We will not ship a typed-tool surface inside the interpreter** (no `peek`, `grep`, `chunk` tools). The whole point of the paper is that the host language *is* the tool surface.
- **We will not expose Python or any other non-Go runtime.** Single-binary deployment is a load-bearing property of spin.
- **We will not let sub-LMs recurse.** The paper deliberately bounds at depth 1; we follow.
- **We will not ship a "Yaegi escape hatch" that loosens the import allowlist on user request.** A loose allowlist is not a sandbox.

## 4. Technical Design

### Architecture

New package: `internal/rlm/`

```
internal/rlm/
  orchestrator.go     // root loop driver
  interp.go           // Yaegi interpreter lifecycle, allowlist, exec timeout
  symbols.go          // injected rlm.* symbols (Ctx, CallLM, Final, ...)
  prompt.go           // root LM system prompt assembly
  pool.go             // sub-LM worker pool
  events.go           // RLMStepEvent, RLMSubCallEvent
  budget.go           // turn / token / subcall accounting
  config.go           // RLMConfig validation & defaults
```

New tool: `internal/tools/rlm/query.go` — `rlm_query` tool exposed inside the harness so the main agent can delegate.

New command: `cmd/spin/rlm.go` — top-level `spin rlm` subcommand.

Wiring:

- `internal/conversation/builder.go`: optional `WithRLM(cfg RLMConfig)`; registers the `rlm_query` tool when configured.
- `internal/config/config_v2.go`: add `RLM RLMV2` section (max turns, max subcalls, sub-call concurrency, default sub-LM model override, exec timeout, stdout truncation).
- `internal/llm/`: no changes — orchestrator and pool both consume `Provider`.
- `internal/agent/harness/bridge`: register two new event types on the emitter.

**New Go dependency**: `github.com/traefik/yaegi` (BSD-3, ~30 MB binary impact). Vendored.

**Interpreter setup** (pseudocode):

```go
i := interp.New(interp.Options{
    GoPath:               "",            // no module resolution
    Unrestricted:         false,
    BuildTags:            nil,
})
// stdlib allowlist — explicit symbol map, NOT stdlib.Symbols (which exposes everything).
i.Use(rlmSymbols())   // strings, regexp, bufio, bytes, encoding/json, encoding/csv,
                      // sort, unicode/utf8, fmt, errors, math, slices, maps, time
i.Use(injectedRLMPackage(orchestrator))  // package "rlm": Ctx, CallLM, Final, FinalVar, Print
_, _ = i.Eval(`import (. "rlm"; "strings"; "regexp"; "fmt")`)  // pre-import for ergonomics
```

The injected `rlm` package is a `map[string]map[string]reflect.Value` built at runtime that closes over the orchestrator: `rlm.Ctx` is the user's input; `rlm.CallLM` is a Go closure that pushes a request onto the sub-LM pool's channel and blocks on the result; `rlm.Final` / `rlm.FinalVar` write to the orchestrator's termination channel.

**Per-`exec` cancellation**: Yaegi supports `interp.Interpreter.Stop()` to abort an in-flight `Eval`. The orchestrator wraps each `Eval` in a goroutine, runs a `time.AfterFunc` for the timeout, and calls `Stop()` on overrun. The interpreter is *not* discarded after a timeout — only the current Eval is aborted, so the persistent globals (including `Ctx`) survive.

**Print capture**: we don't redirect `os.Stdout` (would affect the host process). Instead, `rlm.Print` / `rlm.Printf` write to a `bytes.Buffer` owned by the orchestrator, truncated to the configured limit. `fmt.Println` from inside the interpreter is also redirected by registering a custom `fmt` package that wraps the stdlib `fmt` and overrides `Print*` writers — simpler approach: tell the LM via system prompt to use `rlm.Print` / `rlm.Printf` exclusively, and reject `fmt.Print*` in the prompt examples.

**Parallel `CallLM`** is the injected `CallLMMany([]Request) []Result` — internally fans out N goroutines, each acquiring a slot from the pool semaphore, each calling the same single-shot path as `CallLM`.

### Non-Functional Requirements

- **Performance**: P50 root-loop overhead per turn ≤ 20 ms (Yaegi parse+eval for typical 10–50-line snippets, no IPC). Sub-LM calls dominate; pool keeps ≥ 4 in flight by default. Cold-start of the interpreter ≤ 50 ms including symbol registration.
- **Reliability**: Interpreter panic → orchestrator catches via `recover()`, returns a structured error frame to the LM (so it can adjust its next code), no partial answer claimed as final. Hard timeouts on every `Eval` (default 30 s, via `Stop()`) and every `CallLM` (default 60 s). Budget exceeded → orchestrator force-terminates with a partial-answer-or-failure message.
- **Security**: Yaegi import allowlist is the primary boundary — only safe stdlib packages registered; `os/exec`, `net*`, `syscall`, `unsafe`, `plugin`, `reflect`, `runtime` are unreachable. No filesystem access from inside the interpreter (no `os.Open` etc.). spin's existing process-level sandbox (Landlock/Seatbelt) provides defence-in-depth. `CallLM` cannot be re-entered: the pool refuses calls whose originating goroutine is already serving a `CallLM`.
- **Observability**: One event per root turn (model in/out token counts, exec duration, stdout bytes). One event per sub-call (model, prompt size, response size, latency, cost-est). Final event with total budget consumed. All events flow through the existing emitter — visible in TUI, ACP, and JSONL logs.

### Testing Strategy

- **Unit**:
  - `interp_test.go`: import allowlist enforcement (assert `os/exec` import fails), `Stop()` aborts an infinite loop within timeout+slack, Print capture truncation, panic recovery.
  - `pool_test.go`: concurrency cap, cancellation propagation, error fan-in.
  - `budget_test.go`: turn/subcall/cost overflow detection.
  - `prompt_test.go`: golden-file system prompt assembly.
  - `symbols_test.go`: `Ctx` is read-only-by-convention but assignment is allowed (LM may want to overwrite it with a slice); `Final` short-circuits subsequent ops.
- **Integration** (with a fake `llm.Provider` that returns scripted Go):
  - Happy path: peek (`Ctx[:200]`) → grep (`regexp.MustCompile(...)`) → `Final`.
  - `FinalVar` round-trip with a 1 MB string global.
  - `CallLMMany` with 8 prompts under concurrency=2.
  - Interpreter timeout on `for {}` loop in `exec`.
  - Sub-LM error surfaced as a returned `error` from `CallLM` inside the interpreter.
  - LM tries `import "os/exec"` → eval returns import-denied error → LM adjusts.
- **E2E** (real Yaegi, scripted local LLM via Ollama in CI):
  - `examples/rlm/long_log/` triage scenario reproduces the expected answer deterministically with a fixed-seed local model.
  - `spin rlm --input testdata/big.txt --query …` exits 0 with non-empty answer and a recoverable trajectory.
- **Sandbox**: explicit "evil prompt" tests that the interpreter cannot import `os/exec`, cannot construct a `net.Dial`, cannot reach `syscall.Exec`. Run inside a temp dir with a canary file at `../canary`; assert it's untouched.

### Migration & Compatibility

- Pure addition. No existing behaviour changes when `RLM` config block is absent.
- New CLI subcommand and new tool are opt-in.
- Adds `github.com/traefik/yaegi` to `go.mod` (one new direct dep, vendored). Binary size grows ~30 MB.
- **Zero new runtime dependencies.** No Python, no Docker, no external sandbox service. spin remains a single static binary.

### Dependencies

- **`github.com/traefik/yaegi`** (BSD-3) — Go interpreter. Pinned to a tagged release; we do not depend on `yaegi/stdlib` (which exposes everything) — we build our own symbol map.
- No other new Go dependencies.
- No runtime dependencies.

## 5. User Journey

### Persona

**"Riya"** — staff engineer, drops large artifacts (a 200k-line build log, a 50-file diff, a directory of YAML configs) into spin and asks targeted questions ("which deployment broke first?", "summarise IAM diffs by service"). She has tried piping the file into the chat and watched quality collapse past ~50k tokens. She wants the *right* answer and is willing to wait 30–60 s for it.

### CJM Phases

1. **Trigger** — Riya has a 4 MB log and a question. She knows from past pain that pasting it into the harness will degrade.
   - *Action*: `spin rlm --input build.log "which step failed first and why?"`
   - *Pain*: needs to remember the subcommand exists; needs to know whether to use `rlm` vs. normal `spin chat`.
   - *Success signal*: spin prints "RLM session started, interpreter up, 4 MB context bound to `rlm.Ctx`".

2. **Watching the trajectory** — root LM peeks (`rlm.Print(rlm.Ctx[:500])`), greps (`regexp.MustCompile("(?i)error|fatal|traceback").FindAllStringIndex(rlm.Ctx, -1)`), partitions hits into chunks, fires off `rlm.CallLMMany` with 6 sub-prompts.
   - *Action*: she watches the TUI render each step; sub-call events show prompt/response sizes.
   - *Pain*: too much trajectory noise drowns the signal; she can't tell what code the LM is running.
   - *Success signal*: collapsible per-turn cards in the TUI, with the executed Go shown verbatim and syntax-highlighted.

3. **Termination** — root LM emits `rlm.Final("the failure originated in step 47, …")`.
   - *Action*: she reads the answer, checks the cited step.
   - *Pain*: she wants to ask a follow-up but the interpreter was discarded.
   - *Success signal*: `--keep-interp` flag (post-ML) lets her continue interactively. For ML, she just re-runs.

4. **Cost reconciliation** — she wants to know how much that cost.
   - *Action*: trajectory summary at the end shows total tokens in/out across root + all sub-calls and an estimated dollar figure.
   - *Pain*: per-call breakdown not shown by default (too noisy).
   - *Success signal*: `--verbose` shows the per-call ledger.

5. **Delegation from the main agent** — later, Riya is in a normal `spin chat` session and asks "look through this log and tell me what failed". The harness model decides to call `rlm_query` itself and returns the synthesised answer.
   - *Pain*: she can't tell the harness used RLM under the hood.
   - *Success signal*: a single "delegated to RLM (12 sub-calls, 18 s)" line in the trajectory.

### Friction Map

| Friction | Phase | Opportunity |
|---|---|---|
| Discoverability of `spin rlm` | 1 | `spin chat` detects oversized `--input` flag and prints a one-line hint suggesting `spin rlm`. |
| Trajectory noise vs. signal | 2 | Collapsible per-turn cards; default-collapsed sub-call details. |
| Lost interpreter after termination | 3 | Post-ML `--keep-interp` interactive mode that re-enters the Yaegi REPL with all globals intact. |
| Cost surprise | 4 | Per-session budget cap (`--max-cost-usd`) with hard stop and graceful partial answer. |
| Implicit delegation invisibility | 5 | Standard "delegated to RLM" trajectory marker; click-through to full RLM trajectory. |
| LM-writes-Python-by-mistake | 2 | System prompt opens with a 5-line Go primer + 3 worked snippets; first-turn parse error returns a "this is a Go interpreter, not Python" hint. |

### North Star

A senior engineer drops a 10 MB artefact, asks one targeted question in English, walks to get coffee, comes back to a correct answer with an inspectable, reproducible trajectory under \$0.50 — from a single `spin` static binary with no Python or Docker installed, and trusts it enough to act on it without re-checking.

## 6. Risks & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|
| LM tries to import `os/exec` / `net` to escape | High | High (LMs reach for it) | Yaegi import allowlist refuses unknown packages with a clear error; system prompt enumerates the available packages so the LM doesn't waste turns guessing. |
| Yaegi has an interpreter-escape bug | High | Low | Defence-in-depth: spin's existing process-level Landlock/Seatbelt sandbox still wraps the spin process. Pin Yaegi to vetted releases; track upstream CVEs. |
| Cost runaway from poorly-bounded loops | High | Medium | Hard turn cap, sub-call cap, optional dollar cap; force-terminate with partial answer. |
| LM writes infinite loop in `Eval` | Medium | Medium | Per-`Eval` `time.AfterFunc` calls `interp.Stop()`; the interpreter survives, only the current eval aborts; LM gets a "execution timed out at 30 s" frame. |
| LM writes Python syntax | Low | High at first | System prompt is explicit ("This is Go, not Python"); first parse error returns a hint with a worked Go example; we don't auto-translate. |
| LM fluency in Go is worse than Python | Medium | Medium | Frontier models (Sonnet 4.6, GPT-5) are strong at idiomatic Go for text/regex/slice work, which is what RLM needs. Benchmarked in CI against the long-log scenario. If quality drops vs. paper baseline, fall back to a Python kernel as a v2 option (additive, not replacement). |
| Truncated stdout hides the answer the LM needed | Medium | High | Truncation marker in output; LM is instructed to slice/repeat with smaller windows; `FinalVar` escape for large outputs. |
| Sub-LM rate limits cascade into a stalled session | Medium | Medium | Pool respects 429s with exponential backoff; surfaces stall in the trajectory; budget enforced. |
| LM emits both `Final` and continues executing | Low | Low | Orchestrator treats first `Final`/`FinalVar` as authoritative; subsequent ops ignored with a warning event. |
| Prompt-injection inside the long context steers the root LM | High | Medium | Root LM never sees raw context tokens; injection has to survive a sub-LM hop and a Go-side summarisation, which materially raises the bar. Document non-zero residual risk. |
| Yaegi binary size impact (~30 MB) | Low | High | Accept the cost; spin is a developer tool, not a size-constrained binary. Build tag (`//go:build !no_rlm`) lets size-sensitive deployments compile without it. |

## 7. Open Questions

1. Should the `rlm_query` tool (the in-harness delegator) inherit the parent session's `llm.Provider`, or be configurable separately so users can route sub-calls to a cheaper model? Default proposal: inherit, allow override via config.
2. Do we expose `CallLM`'s tool surface (i.e. can sub-LMs themselves do tool calls)? Paper says no. Proposal: no for v1, revisit if a clear use case emerges.
3. How do we represent multi-input contexts (e.g. a directory of files)? As `rlm.CtxMap map[string]string` keyed by relative path? Or as paths exposed via `rlm.CtxPaths []string` and a `rlm.ReadCtx(path string) string` reader (no real filesystem access — backed by an in-memory map)? Proposal: both — `CtxMap` for small inputs, `CtxPaths` + `ReadCtx` indirection for >10 MB.
4. Should we expose a curated `rlm.Grep`, `rlm.Chunk`, `rlm.Peek` helper layer on top of the raw stdlib, or stay strict to the paper's "stdlib only" line? Proposal: stay strict in v1; add helpers only if benchmarks show LMs waste turns reinventing them.
5. Should `FinalVar` be size-limited? Proposal: yes, configurable, default 10 MB; over-limit returns an error to the LM so it can chunk.
6. How do RLM events interact with ACP? Proposal: emit them as generic "subagent" notifications with a `kind=rlm` discriminator until ACP gains first-class support.
7. Do we want a build tag (`no_rlm`) to exclude Yaegi for slim builds? Proposal: yes — costs nothing and keeps the door open for embedded use cases.

## 8. Implementation Roadmap

Order is dependency-ordered; each phase is a shippable, testable increment.

1. **Phase 1 — Interpreter foundation (no LLM)**
   `internal/rlm/interp.go`, `symbols.go`. Embed Yaegi, build the stdlib allowlist symbol map, register an injected `rlm` package with a stub `CallLM`, implement per-`Eval` timeout via `Stop()`, implement Print capture with truncation. Tests: import allowlist deny (`os/exec`, `net`, `syscall`), `Stop()` aborts infinite loop, panic recovery, Print truncation, Ctx round-trip.

2. **Phase 2 — Root loop & sub-LM pool (real LLM)**
   `orchestrator.go`, `pool.go`, `prompt.go`, `budget.go`. Wire `CallLM` and `CallLMMany` to the pool; enforce concurrency, turn budget, sub-call budget. Tests with a scripted fake `llm.Provider`.

3. **Phase 3 — Termination & events**
   `Final` / `FinalVar` paths (`FinalVar` reads the named global out of the Yaegi interpreter via `Eval(name)`); `RLMStepEvent` / `RLMSubCallEvent` on the existing emitter. TUI rendering of collapsible per-turn cards with Go syntax highlighting.

4. **Phase 4 — `spin rlm` CLI**
   `cmd/spin/rlm.go`. Flags: `--input`, `--query`, `--model`, `--max-turns`, `--max-subcalls`, `--max-cost-usd`, `--verbose`. End-to-end smoke test against Ollama in CI.

5. **Phase 5 — In-harness delegation**
   `internal/tools/rlm/query.go`: `rlm_query` tool registered when RLM config is present. Builder integration via `WithRLM`. Trajectory marker in the harness when delegated.

6. **Phase 6 — Worked example & docs**
   `examples/rlm/long_log/`, README section, AGENTS.md note. One-page "when to use RLM vs. chat" guide. Include the 5-line Go primer the system prompt uses, so users can predict what the LM will write.

7. **Phase 7 (post-ML) — Quality of life**
   `--keep-interp` interactive REPL passthrough (Yaegi has a built-in REPL we can hand control to); per-session budget caps with graceful degradation; auto-suggest from `spin chat` on oversized inputs; optional curated helper layer (`rlm.Grep`, `rlm.Chunk`, `rlm.Peek`) if benchmarks justify.

## 9. Prompt Templates

We adopt the paper's reference prompts (from `alexzhang13/rlm/rlm/utils/prompts.py`) verbatim in structure — tool enumeration, worked examples, `FINAL` / `FINAL_VAR` rules, iteration-0 safeguard, per-turn continuation prompt. We translate Python idioms to Go because our interpreter is Yaegi. Preserving the paper's prompt design is load-bearing: the reported quality numbers (+28.3% avg) are coupled to these exact instructions.

Templates live in `internal/rlm/prompt.go` as Go string constants; the system prompt is assembled from a base template plus a metadata suffix, matching `build_rlm_system_prompt` upstream.

### 9.1 Root system prompt (`RLMSystemPrompt`)

Fenced code blocks the LM emits are tagged ` ```go ` (not ` ```repl `) — the orchestrator parses them out of the root LM's message, strips the fence, and feeds the body to `rlm_exec`. Prose is kept close to the original; only tool signatures and worked examples are Go-ified.

```text
You are tasked with answering a query with associated context. You can access,
transform, and analyze this context interactively in a Go REPL environment
(Yaegi interpreter) that can recursively query sub-LLMs, which you are strongly
encouraged to use as much as possible. You will be queried iteratively until
you provide a final answer.

The REPL environment is initialized with:
1. A package `rlm` that is dot-imported. It exposes:
   - `rlm.Ctx` (string) — the context variable. Contains extremely important
     information about your query. Check its content to understand what you
     are working with. Look through it sufficiently as you answer.
   - `rlm.CtxMap` (map[string]string) — present only if the input is multiple
     documents, keyed by path.
2. `rlm.CallLM(prompt string, snippet string, opts ...rlm.Option) (string, error)`
   — a single LLM completion call (no REPL, no iteration). Fast and
   lightweight. Use for simple extraction, summarization, or Q&A over a chunk.
   The sub-LLM can handle around 500K characters.
3. `rlm.CallLMMany(reqs []rlm.Request) ([]rlm.Result, error)` — runs multiple
   `CallLM` calls concurrently; returns results in the same order as input.
   Much faster than sequential calls for independent queries.
4. `rlm.QueryRLM(prompt string, opts ...rlm.Option) (string, error)` — spawns
   a **recursive RLM sub-call** for deeper thinking subtasks. The child gets
   its own REPL and can reason iteratively. Use when a subtask requires
   multi-step reasoning, code execution, or its own iterative problem-solving.
   Falls back to `CallLM` if recursion is disabled.
5. `rlm.QueryRLMMany(reqs []rlm.Request) ([]rlm.Result, error)` — spawns
   multiple recursive RLM sub-calls concurrently. Falls back to `CallLMMany`
   if recursion is disabled.
6. `rlm.ShowVars() map[string]string` — returns all globals you have defined
   in the REPL (name → short type/preview). Use this to check what variables
   exist before using FINAL_VAR.
7. `rlm.Print(v ...any)` / `rlm.Printf(format string, v ...any)` — capture
   stdout for your own reasoning. Do not use `fmt.Print*`; it is not wired.
{CustomToolsSection}

Available imports (allowlist): strings, regexp, bufio, bytes, encoding/json,
encoding/csv, sort, unicode/utf8, fmt (format-only; use rlm.Print for output),
errors, math, slices, maps, time. Anything else will fail to import — do not
try `os`, `os/exec`, `net`, `syscall`.

**When to use CallLM vs QueryRLM:**
- Use `CallLM` for simple, one-shot tasks: extracting info from a chunk,
  summarizing text, answering a factual question, classifying content.
- Use `QueryRLM` when the subtask itself requires deeper thinking:
  multi-step reasoning, solving a sub-problem that needs its own REPL and
  iteration, or tasks where a single LLM call might not be enough.

**Breaking down problems:** You must break problems into digestible
components — whether chunking or summarizing a large context, or decomposing
a hard task into easier sub-problems and delegating them via `CallLM` /
`QueryRLM`. Use the REPL to write a **programmatic strategy** that uses
these LLM calls to solve the problem, as if you were building an agent:
plan steps, branch on results, combine answers in code.

**REPL for computation:** You can also compute programmatic steps
(e.g. `math.Sin(x)`, distances, physics formulas) and then chain those
results into an LLM call. For complex math, compute intermediate quantities
in code and pass the numbers to the LM for interpretation. Example —
electron in a magnetic field undergoing helical motion, find entry angle:
```go
import "math"
// Suppose context or an earlier CallLM gave us: B, m, q, pitch, R.
vParallel := pitch * (q * B) / (2 * math.Pi * m)
vPerp    := R * (q * B) / m
thetaRad := math.Atan2(vPerp, vParallel)
thetaDeg := thetaRad * 180 / math.Pi
ans, _ := rlm.CallLM(
    fmt.Sprintf("Electron in B field, helical motion. Entry angle: %.2f deg. State the answer clearly.", thetaDeg),
    "",
)
finalAnswer = ans
```

You will only see truncated output from the REPL, so use `CallLM` on
variables you want to analyze semantically. Use variables as buffers to
build up your final answer.
Make sure to explicitly look through the entire context in REPL before
answering. Break the context and problem into digestible pieces: decide a
chunking strategy, chunk smartly, query an LLM per chunk and save answers
to a buffer, then query an LLM over the buffers to produce your final
answer.

Your sub-LLMs are powerful — they can fit around 500K characters in their
context window, so don't be afraid to put a lot of context into them.
Feeding 10 documents per sub-LLM query is a viable strategy.

When you want to execute Go code in the REPL, wrap it in triple backticks
with the `go` language identifier. Example — search for the magic number
in the context:
```go
chunk := rlm.Ctx[:10000]
answer, _ := rlm.CallLM("What is the magic number in this chunk?", chunk)
rlm.Print(answer)
```

Suppose you're answering a question about a book. Iterate section by
section, query an LLM on each chunk, track relevant information in a
buffer:
```go
query := "In Harry Potter and the Sorcerer's Stone, did Gryffindor win the House Cup because they led?"
sections := strings.Split(rlm.Ctx, "\n## ") // assume markdown section headers
var buffers string
for i, section := range sections {
    var prompt string
    if i == len(sections)-1 {
        prompt = fmt.Sprintf("Last section. So far you know: %s. Gather from this last section to answer %q.", buffers, query)
    } else {
        prompt = fmt.Sprintf("Section %d of %d. Gather information to help answer %q.", i, len(sections), query)
    }
    ans, _ := rlm.CallLM(prompt, section)
    buffers += ans + "\n"
    rlm.Printf("After section %d/%d, tracked: %s\n", i, len(sections), ans)
}
finalAnswer = buffers
```

When the context isn't huge, combine chunks and recursively query an LLM.
Use `CallLMMany` for concurrent processing — much faster than sequential:
```go
query := "A man became famous for his book 'The Great Gatsby'. How many jobs did he have?"
// Split ~1 MB of context into 10 ~100 KB chunks.
n := 10
chunkSize := len(rlm.Ctx) / n
reqs := make([]rlm.Request, n)
for i := 0; i < n; i++ {
    start := i * chunkSize
    end := start + chunkSize
    if i == n-1 {
        end = len(rlm.Ctx)
    }
    reqs[i] = rlm.Request{
        Prompt:  fmt.Sprintf("Try to answer: %q. Only answer if confident based on evidence.", query),
        Snippet: rlm.Ctx[start:end],
    }
}
results, _ := rlm.CallLMMany(reqs)
var joined string
for i, r := range results {
    rlm.Printf("Answer from chunk %d: %s\n", i, r.Text)
    joined += r.Text + "\n"
}
final, _ := rlm.CallLM(
    fmt.Sprintf("Aggregate these per-chunk answers and answer the original query: %q", query),
    joined,
)
finalAnswer = final
```

For subtasks needing deeper reasoning, use `QueryRLM`. The child gets its
own REPL to iterate; use the result in parent logic:
```go
trend, _ := rlm.QueryRLM("Analyze this dataset and conclude with one word: up, down, or stable: " + data)
var rec string
switch {
case strings.Contains(strings.ToLower(trend), "up"):
    rec = "Consider increasing exposure."
case strings.Contains(strings.ToLower(trend), "down"):
    rec = "Consider hedging."
default:
    rec = "Hold position."
}
summary, _ := rlm.CallLM(
    fmt.Sprintf("Given trend=%s and recommendation=%s, one-sentence summary for the user.", trend, rec),
    "",
)
finalAnswer = summary
```

Implement the solution as a **program**: try one approach via `QueryRLM`;
inspect; branch. More branches, one path runs — don't load the model.
Example — prove sqrt 2 irrational:
```go
r, _ := rlm.QueryRLM("Prove sqrt 2 is irrational. Give a 1-2 sentence proof, or reply only: USE_LEMMA or USE_CONTRADICTION.")
if strings.Contains(strings.ToUpper(r), "USE_LEMMA") {
    finalAnswer, _ = rlm.QueryRLM("Prove 'n^2 even => n even', then use it to show sqrt 2 irrational. Two sentences.")
} else {
    finalAnswer = r
}
```

IMPORTANT: When you are done with the iterative process, you MUST provide a
final answer using a FINAL marker in your message (NOT in code). Do not use
these markers unless you have completed your task. Two options:
1. `FINAL(your final answer here)` — provide the answer directly.
2. `FINAL_VAR(variable_name)` — return a variable you created in the REPL
   environment as the final output (for large structured outputs).

WARNING — COMMON MISTAKE: FINAL_VAR retrieves an EXISTING REPL variable.
You MUST create and assign the variable in a ```go``` block FIRST, then
call FINAL_VAR in a SEPARATE step.
- WRONG: calling `FINAL_VAR(myAnswer)` without first defining `myAnswer`.
- CORRECT: first run
  ```go
  myAnswer := "the result"
  rlm.Print(myAnswer)
  ```
  then in the NEXT response call `FINAL_VAR(myAnswer)`.

If unsure what variables exist, call `rlm.ShowVars()` in a ```go``` block
to see all available globals.

Note: this is a Go REPL, not Python. Use `:=` for new variables, `=` for
assignment, `fmt.Sprintf` for string interpolation (not f-strings), and
`rlm.Print` / `rlm.Printf` for output.

Think step by step carefully, plan, and execute this plan immediately in
your response — do not just say "I will do this". Output to the REPL and
sub-LLMs as much as possible. Remember to explicitly answer the original
query in your final answer.
```

### 9.2 Metadata suffix (appended as a `user` message)

Matches the paper's `metadata_prompt`:

```text
Your context is a {ContextType} with {ContextTotalChars} total characters,
and is broken up into chunks of char lengths: {ContextLengths}.
```

If `len(ContextLengths) > 100`, truncate to the first 100 entries and append `"... [N others]"`, matching the reference.

### 9.3 Per-turn user prompt (`UserPrompt` / `UserPromptWithRoot`)

Direct translation; fence tag updated from ` ```repl ``` ` to ` ```go ``` `:

```text
Think step-by-step on what to do using the REPL environment (which contains
the context) to answer the prompt.

Continue using the REPL environment, which has the `rlm.Ctx` variable, and
querying sub-LLMs by writing to ```go``` tags, and determine your answer.
Your next action:
```

With a root prompt (used by recursive sub-RLMs to keep the original query in view):

```text
Think step-by-step on what to do using the REPL environment (which contains
the context) to answer the original prompt: "{RootPrompt}".

Continue using the REPL environment, which has the `rlm.Ctx` variable, and
querying sub-LLMs by writing to ```go``` tags, and determine your answer.
Your next action:
```

### 9.4 Iteration-0 safeguard

Prepended on the first user turn only, verbatim from the reference:

```text
You have not interacted with the REPL environment or seen your prompt /
context yet. Your next action should be to look through and figure out how
to answer the prompt, so don't just provide a final answer yet.
```

### 9.5 Continuation preamble (iteration ≥ 1)

Prepended on subsequent turns, verbatim from the reference:

```text
The history before is your previous interactions with the REPL environment.
```

### 9.6 Multi-context / history suffix

Appended to the per-turn user prompt when applicable, verbatim in structure:

- Multiple contexts: `"Note: You have {N} contexts available (ctx_0 through ctx_{N-1})."`
- One history: `"Note: You have 1 prior conversation history available in the `history` variable."`
- Many histories: `"Note: You have {N} prior conversation histories available (history_0 through history_{N-1})."`

### 9.7 Output parsing contract

The orchestrator parses the root LM's response in this order (first match wins per turn):

1. `FINAL_VAR(name)` — strict regex `FINAL_VAR\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)` — resolve via Yaegi `Eval(name)` and terminate.
2. `FINAL(answer)` — multiline, greedy within a single balanced-paren pair — terminate with the literal string.
3. ` ```go … ``` ` fenced block(s) — concatenate all blocks in the message (matching the reference behaviour of running each code cell in order), feed to `rlm_exec`.
4. No match → soft nudge turn: emit a user message `"I didn't see a ```go``` block or a FINAL/FINAL_VAR marker in your last response. Please either run code or provide a final answer."` and increment a "missed turn" counter; three misses in a row → orchestrator force-terminates with a failure message.

### 9.8 Sub-LM (flat `CallLM`) prompt

Sub-LMs receive no system prompt scaffolding — they are one-shot completions. The envelope is the paper's minimal form:

```text
{prompt}

---
{snippet}
```

Snippet is omitted if empty. No tools, no REPL, no recursion. The sub-LM's raw response text is returned to the interpreter as the `CallLM` result.

### 9.9 Attribution & drift policy

- Prompts are derived from `alexzhang13/rlm` under its upstream licence; we include a NOTICE entry and a `SOURCE:` comment at the top of `prompt.go` pointing at the specific commit we forked from.
- When upstream revises the prompts, we diff against our baseline and port non-cosmetic changes (new tool descriptions, new worked examples, safety clauses) in a dedicated PR — not silently, so quality regressions are attributable.
- Our Go-translation deltas (fence tag, `rlm.*` symbol names, Go syntax in examples, allowlist note) are marked inline with `// spin: Go adaptation` comments in `prompt.go` so a reviewer can audit exactly where we diverge from the paper.
