# JOURNEY-016-a2a-types-and-local-json-rpc-codec: A2A types and local JSON-RPC codec

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: A2A types and local JSON-RPC codec

## 1. Journey

When **Alex (or a later parent harness) talks to a peer agent over a local pipe** I want **one A2A 1.0 type stack (Agent Card, Task, Message, Part, Artifact) and a JSON-RPC 2.0 codec that round-trips `message/send` and `tasks/get`** so I **can drive a child without HTTP, without an OS process, and without a second type stack when HTTPS lands later**.

## 2. CJM

Alex already has ACP (`internal/protocol/acp`) for IDE hosts and `pkg/protocol/jsonrpc` Content-Length transport for LSP. There is no A2A package. Subagents are still in-process stubs. This journey adds `internal/protocol/a2a` types plus an in-memory client/server over a pipe. It does **not** spawn `spin a2a` (Step 17), does **not** open an HTTP listener, and does **not** mix ACP types into A2A.

Assumption (SPEC vs official A2A JSON-RPC binding): spin SPEC and Step 16 name slash methods (`message/send`, `message/stream`, `tasks/get`, `tasks/list`, `tasks/cancel`, card fetch as `agent/getAuthenticatedExtendedCard`). Official A2A 1.0 HTTP JSON-RPC uses PascalCase (`SendMessage`). This journey uses the SPEC slash names. Assumption (transport): SPEC local binding is NDJSON-RPC on a stream. `pkg/protocol/jsonrpc` is a client-only Content-Length transport used by LSP and does not dispatch incoming requests, so it is not reused as the A2A codec. Bindings map transport only; types stay in one package.

### Phase 1: Name the A2A objects

**User Intent:** Hold a Card, Task, Message, Part, and Artifact that serialize as A2A 1.0 JSON (camelCase fields, proto enum strings).

**Actions:** Construct a card with a skill and a JSONRPC/NDJSON interface. Construct a user message with a text part. Attach an artifact to a task.

**Pain / Risk:** ACP session types leak into A2A; snake_case tags break A2A 1.0; Part allows more than one of text/raw/url/data; Task state strings drift from `TASK_STATE_*`.

**Success Signal:** Types marshal with `contextId`, `messageId`, `artifactId`, `supportedInterfaces`, `TASK_STATE_COMPLETED`, `ROLE_USER`. No import of `internal/protocol/acp`.

### Phase 2: Speak JSON-RPC 2.0 on a pipe

**User Intent:** Send a framed request and get a framed response without HTTP or a child process.

**Actions:** Pair a client and server on `net.Pipe` (or `io.Pipe`). Call `message/send`, then `tasks/get` with the returned task id.

**Pain / Risk:** Content-Length vs NDJSON mismatch; server never reads requests; deadlock on an unbuffered pipe; tests exec a spin binary; a goroutine listen on a TCP port.

**Success Signal:** Client receives a Task from `message/send` and the same Task from `tasks/get`. Tests import only the a2a package plus stdlib/testify.

### Phase 3: Fail with standard and domain codes

**User Intent:** Tell a broken peer why the call failed using numbers other agents already know.

**Actions:** Send invalid JSON, a non-2.0 envelope, an unknown method, `tasks/get` for a missing id, `tasks/cancel` on a terminal task, `message/stream` when streaming is off.

**Pain / Risk:** Parse errors use a made-up code; domain errors sit outside −3200x; invalid request looks like internal error; card fetch has no method name.

**Success Signal:** Parse → −32700; invalid request → −32600; method not found → −32601; invalid params → −32602; task missing → −32001; not cancelable → −32002; unsupported stream → −32004.

### Phase 4: Fetch the card without leaving the stream

**User Intent:** Learn who is on the other end of the pipe before sending work.

**Actions:** Read the first framed message (local card announce). Call `agent/getAuthenticatedExtendedCard`. List and cancel tasks created by send.

**Pain / Risk:** First line is raw JSON not JSON-RPC and breaks the codec; card method is HTTP-only; list/cancel are stubs that panic.

**Success Signal:** Client sees the announced card. Method fetch returns the same identity. `tasks/list` and `tasks/cancel` complete over the same pipe.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| No A2A types; ACP is the only protocol package | 1 | One `internal/protocol/a2a` type stack |
| HTTP mental model for a local child | 2 | NDJSON JSON-RPC on a pipe |
| Dual Content-Length vs NDJSON stacks | 2 | Bindings map transport; types stay shared |
| Opaque failures | 3 | Standard JSON-RPC + A2A −3200x codes |
| Card only on HTTPS well-known URL | 4 | First framed message + card method |
| Process spawn leaking into codec tests | 2 | In-memory handler; no `os/exec` |

### North Star Summary

A parent (or a test) can open two ends of a pipe, read an Agent Card, send a message, get a Task, list and cancel tasks, and receive protocol errors with the numbers A2A 1.0 already published. The same types will later ride stdio (Step 17) and HTTPS without a second Card/Task/Message package. No HTTP server and no OS child exist in this journey.

### Stressors

1. ACP types or `coder/acp-go-sdk` imported from `internal/protocol/a2a` (protocol mix).
2. JSON tags use snake_case (`context_id`) and fail A2A 1.0 camelCase interop.
3. `pkg/protocol/jsonrpc.StdioTransport` is reused as-is: it only reads responses, so a server never dispatches `message/send`.
4. Tests start `build/bin/spin` or `os/exec` and violate “no process spawn / no spin binary”.
5. An HTTP `Listen` or `http.Server` appears and violates “no HTTP listener”.
6. Invalid JSON is mapped to −32603 or a custom code instead of −32700.
7. `tasks/get` for an unknown id returns −32602 or a Go error instead of −32001.
8. `tasks/cancel` on `TASK_STATE_COMPLETED` succeeds instead of −32002.
9. `message/stream` is accepted when `capabilities.streaming` is false (must be −32004).
10. Unbuffered `net.Pipe` deadlock if both sides write before either reads the card announce.
11. Dual type stacks: a “local Task” plus a “HTTPS Task” with different JSON.
12. Method names use PascalCase `SendMessage` and miss SPEC `message/send`.
13. Card fetch has no method (`agent/getAuthenticatedExtendedCard`) and only a side-channel file.
14. Part JSON emits more than one of `text` / `raw` / `url` / `data`.
15. A2A domain errors use −32000 or codes outside −32001..−32009.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] A test constructs a Card and sends `message/send` without a binary or port
- [x] First framed message is an Agent Card (no extra discovery step)

### Onboarding Clarity
- [x] Method names match SPEC (`message/send`, `tasks/get`, …)
- [x] Error codes are numeric JSON-RPC codes, not opaque strings only

### Production-Ready Defaults
- [x] Local binding is NDJSON JSON-RPC 2.0 (no HTTP stack)
- [x] In-memory handler works with a fixture card and no config file

### Golden Path Quality
- [x] `message/send` then `tasks/get` round-trip the same task id
- [x] Returned Task carries status and optional artifacts

### Decision Load
- [x] One types package; caller does not choose local vs HTTPS structs
- [x] Client Call is sequential (one in-flight request) so tests stay simple

### Progressive Complexity
- [x] Text parts are enough for send/get
- [x] Stream, list, cancel, card fetch exist but do not require a child process

### Error Quality
- [x] Invalid JSON-RPC uses −32700 / −32600 / −32601 / −32602
- [x] A2A domain errors use −32001 / −32002 / −32004 as specified

### Failure Safety
- [x] Missing task does not panic
- [x] Cancel of a terminal task is a domain error, not a hang

### Runtime Transparency
- [x] Task `id` and `status.state` are present on send/get results
- [x] Card `name` / `version` / `supportedInterfaces` are readable

### Debuggability
- [x] Envelope is one JSON object per line (inspectable in a buffer)
- [x] RPCError includes code and message

### Cross-Surface Consistency
- [x] Same types will be usable later on stdio and HTTPS
- [x] Terminology is Card, Task, Message, Part, Artifact (not ACP session)

### Workflow Consistency
- [x] Package layout matches `internal/protocol/<name>`
- [x] Journey comment on tests points at this file

### Change Safety
- [x] LSP `pkg/protocol/jsonrpc` is unchanged (no Content-Length fork for A2A)
- [x] ACP package is unchanged

### Experimentation Safety
- [x] Tests use `net.Pipe` / `io.Pipe`; no listening socket left behind
- [x] In-memory tasks vanish when the test process ends

### Interaction Latency
- [x] In-process pipe; no network listen
- [x] Sequential Call returns when the server writes one line

### Developer Feedback Speed
- [x] Pipe tests fail in this package without building spin
- [x] Error-code tests name the expected integer

### Team Scale
- [x] Types live in one package for every binding
- [x] Method name constants are the shared vocabulary

### System Scale
- [x] New A2A methods add a constant + handler case, not a new type stack
- [x] Handler interface is one method (dispatch stays in the server)

### Right Behavior by Default
- [x] `jsonrpc` field must be `"2.0"` or the call is −32600
- [x] Streaming off ⇒ `message/stream` is −32004

### Anti-Bypass Design
- [x] Tests assert JSON-RPC error codes, not only `err != nil`
- [x] Round-trip test uses a pipe, not a mocked codec shortcut that skips framing

## 4. Tests

### TC-01: types cover Card, Task, Message, Part, Artifact

**Given** fixture values for each A2A object.
**When** they are JSON-marshaled and unmarshaled.
**Then** camelCase keys and enum strings round-trip (`contextId`, `TASK_STATE_COMPLETED`, `ROLE_USER`).

### TC-02: message/send over a pipe

**Given** a MemoryHandler and a `net.Pipe` pair.
**When** the client calls `message/send` with a text part.
**Then** the result is a Task with a non-empty id and a non-unspecified state.

### TC-03: tasks/get round-trip

**Given** a task created by `message/send`.
**When** the client calls `tasks/get` with that id.
**Then** the returned Task has the same id.

### TC-04: invalid JSON yields −32700

**Given** a server reading NDJSON.
**When** the client writes a non-JSON line.
**Then** the response error code is −32700.

### TC-05: invalid request yields −32600

**Given** a server on a pipe.
**When** the client writes JSON without `jsonrpc: "2.0"`.
**Then** the response error code is −32600.

### TC-06: unknown method yields −32601

**Given** a server on a pipe.
**When** the client calls `no/such`.
**Then** the response error code is −32601.

### TC-07: tasks/get missing id yields −32001

**Given** an empty in-memory store.
**When** the client calls `tasks/get` with `missing`.
**Then** the response error code is −32001.

### TC-08: cancel terminal task yields −32002

**Given** a completed task.
**When** the client calls `tasks/cancel`.
**Then** the response error code is −32002.

### TC-09: stream without capability yields −32004

**Given** a card with `streaming` false.
**When** the client calls `message/stream`.
**Then** the response error code is −32004.

### TC-10: card announce and card fetch

**Given** Serve writes the card as the first framed notification.
**When** NewClient reads the announce and later calls `agent/getAuthenticatedExtendedCard`.
**Then** both cards share name and version.

### TC-11: tasks/list returns sent tasks

**Given** one successful `message/send`.
**When** the client calls `tasks/list`.
**Then** the result contains that task id.

### TC-12: no HTTP and no spin binary

**Given** the a2a test package.
**When** tests run under `go test ./internal/protocol/a2a`.
**Then** they do not import `net/http` for a listener and do not exec a spin binary.

## Acceptance Criteria

- [x] Types cover Card, Task, Message, Part, Artifact
- [x] Client and server round-trip `message/send` and `tasks/get` over a pipe
- [x] Invalid JSON-RPC yields standard error codes; A2A domain errors use the −3200x range where specified
- [x] No HTTP listener in this step
- [x] Unit tests do not require a spin binary
- [x] `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 16
- Implementation files: `internal/protocol/a2a/card.go`, `internal/protocol/a2a/task.go`, `internal/protocol/a2a/message.go`, `internal/protocol/a2a/methods.go`, `internal/protocol/a2a/errors.go`, `internal/protocol/a2a/codec.go`, `internal/protocol/a2a/server.go`, `internal/protocol/a2a/client.go`, `internal/protocol/a2a/handler.go`
- Test files: `internal/protocol/a2a/card_test.go`, `internal/protocol/a2a/codec_test.go`, `internal/protocol/a2a/coverage_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-016-a2a-types-and-local-json-rpc-codec.md` — this journey
- `internal/protocol/a2a/card.go` — Agent Card and interface/skill/capability
- `internal/protocol/a2a/task.go` — Task, TaskStatus, TaskState
- `internal/protocol/a2a/message.go` — Message, Part, Artifact, Role
- `internal/protocol/a2a/methods.go` — slash method names and params/result types
- `internal/protocol/a2a/errors.go` — JSON-RPC −326xx and A2A −3200x codes
- `internal/protocol/a2a/codec.go` — NDJSON JSON-RPC 2.0 envelope
- `internal/protocol/a2a/server.go` — Serve + card announce
- `internal/protocol/a2a/client.go` — sequential Client over a pipe
- `internal/protocol/a2a/handler.go` — in-memory method implementation
- `internal/protocol/a2a/card_test.go` — camelCase JSON and enum constants
- `internal/protocol/a2a/codec_test.go` — pipe round-trip and error codes
- `internal/protocol/a2a/coverage_test.go` — client/server edge paths

Files modified:
- `.golangci.yml` — tagliatelle exclusion for A2A camelCase (same pattern as Smithery)
- `docs/testing.md` — journey 016 row
- `specs/agent-harness/ROADMAP.md` — Step 16 DoD and traceability

Deviation: `pkg/protocol/jsonrpc` was not reused. It is a client-only Content-Length transport (LSP). SPEC local binding is NDJSON-RPC; the A2A codec is NDJSON in this package so types stay in one stack and the server can dispatch requests.
